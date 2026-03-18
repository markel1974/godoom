package model

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/textures"
)

// ThingBullet represents a specialized type of Thing designed to simulate projectile-like behavior in the environment.
type ThingBullet struct {
	*ThingBase
}

// NewThingBullet creates and initializes a new ThingBullet instance with specific properties and links it to the game world.
// cfg specifies the configuration of the bullet, anim defines its animation, and sector represents its initial sector.
// sectors and entities provide references to all sectors and entities in the game world.
func NewThingBullet(cfg *ConfigThing, anim *textures.Animation, sector *Sector, sectors *Sectors, entities *Entities) *ThingBullet {
	p := &ThingBullet{
		ThingBase: NewThingBase(cfg, anim, sector, sectors, entities),
	}
	p.entities.AddThing(p)
	// Annulla il decadimento inerziale per mantenere una velocità lineare costante
	p.entity.Friction = 1.0
	p.entity.GForce = 1.0

	// Calculate the directional vector based on the original firing angle
	dirX := math.Cos(p.angle) * p.speed
	dirY := math.Sin(p.angle) * p.speed

	const acceleration = 0.15
	p.entity.Vx = p.entity.Vx*(1-acceleration) + (dirX * acceleration)
	p.entity.Vy = p.entity.Vy*(1-acceleration) + (dirY * acceleration)
	if p.entity.GForce == 0 {
		p.entity.GForce = 1.0
	}
	if p.entity.Friction < 0.2 {
		p.entity.Friction = 0.99
	}
	return p
}

// Compute updates the bullet's direction and handles its collision, potentially triggering its deallocation.
func (t *ThingBullet) Compute(playerX float64, playerY float64) {
	if t.speed == 0 {
		return
	}
	if math.Abs(t.entity.Vx) < 0.1 && math.Abs(t.entity.Vy) < 0.1 {
		t.entity.Vx = 0
		t.entity.Vy = 0
		t.speed = 0
		t.entity.Invalidate()
		//TODO REMOVE!!!!
		return
	}

	// Calculate the directional vector based on the original firing angle
	//dirX := math.Cos(t.angle) * t.speed
	//dirY := math.Sin(t.angle) * t.speed

	//t.modifyDirection(dirX, dirY)

	// Trigger for impact handling and subsequent deallocation
	//if t.entity.Collider != nil {
	//	t.entity.Vx = 0
	//	t.entity.Vy = 0
	// Hit/explosion logic, damage application and entity removal
	//	t.speed = 0
	//	t.entity.Invalidate()
	//	//TODO REMOVE!!!!
	//}
}

// PhysicsApply updates the entity's position based on passive and active deltas, ensuring movement exceeds a minimum threshold.
func (t *ThingBullet) PhysicsApply() {
	ex, ey := t.entity.GetCenterXY()
	// Passive Delta (bounces computed by SetupCollision)
	tx := ex - t.position.X
	ty := ey - t.position.Y
	// Active Delta (Kinematic Drive) added only if there is intentionality
	//if t.entity.G > 0 {
	tx += t.entity.Vx
	ty += t.entity.Vy
	//}
	if math.Abs(tx) > minMovement || math.Abs(ty) > minMovement {
		t.moveApply(tx, ty)
	}
}

// MoveApply updates the position of the object by applying the given translation vector (tx, ty) with movement constraints.
func (t *ThingBullet) moveApply(t1x float64, t1y float64) {
	x, y := t.bounceMovement(t1x, t1y)
	//TODO WALL BOUNCE!!!
	t.position.X += x
	t.position.Y += y
	if newSector := t.sectors.SectorSearch(t.sector, t.position.X, t.position.Y); newSector != nil {
		t.sector = newSector
	}
	t.entities.UpdateThing(t, t.position.X, t.position.Y)
}

// slidingMovement adjusts the movement velocity based on collisions and elevation differences in the current sector.
func (t *ThingBullet) bounceMovement(velX float64, velY float64) (float64, float64) {
	headPos := t.sector.FloorY + t.height
	kneePos := t.sector.FloorY + 2.0
	viewX, viewY := t.position.X, t.position.Y
	pX := viewX + velX
	pY := viewY + velY
	velX, velY = t.sector.EffectBounce(viewX, viewY, pX, pY, velX, velY, headPos, kneePos)
	return velX, velY
}
