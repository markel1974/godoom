package model

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/physics"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// ThingBase represents a physical or logical entity in the environment with attributes like position, mass, and associated data.
type ThingBase struct {
	id        string
	position  XY
	mass      float64
	radius    float64
	height    float64
	angle     float64
	kind      int
	speed     float64
	sector    *Sector
	animation *textures.Animation
	sectors   *Sectors
	entities  *Entities
	entity    *physics.Entity
}

// NewThingBase creates and initializes a new ThingEnemy instance with the specified configuration, animation, sector, and entities.
func NewThingBase(cfg *ConfigThing, anim *textures.Animation, sector *Sector, sectors *Sectors, entities *Entities) *ThingEnemy {
	w := cfg.Radius * 2
	h := cfg.Radius * 2
	x := cfg.Position.X - cfg.Radius
	y := cfg.Position.Y - cfg.Radius
	thing := &ThingEnemy{
		id:        cfg.Id,
		position:  cfg.Position,
		angle:     cfg.Angle,
		kind:      cfg.Kind,
		mass:      cfg.Mass,
		radius:    cfg.Radius,
		height:    cfg.Height,
		speed:     cfg.Speed,
		sector:    sector,
		animation: anim,
		sectors:   sectors,
		entities:  entities,
		entity:    physics.NewEntity(x, y, w, h, cfg.Mass),
	}
	thing.entities.AddEntity(thing.id, thing.entity)
	return thing
}

// GetId returns the unique identifier of the Thing as a string.
func (t *ThingBase) GetId() string {
	return t.id
}

func (t *ThingBase) GetKind() int {
	return t.kind
}

// GetAnimation retrieves the animation associated with the Thing and returns it as a pointer to textures.Animation.
func (t *ThingBase) GetAnimation() *textures.Animation {
	return t.animation
}

// GetPosition retrieves the current position of the Thing as an XY value.
func (t *ThingBase) GetPosition() XY {
	return t.position
}

// GetLight retrieves the light source associated with the Thing's current sector and returns it as a pointer to Light.
func (t *ThingBase) GetLight() *Light {
	return t.sector.Light
}

// GetFloorY returns the Y-coordinate of the floor in the Thing's current sector as a float64.
func (t *ThingBase) GetFloorY() float64 {
	return t.sector.FloorY
}

// GetCeilY retrieves the ceiling height of the sector associated with the Thing and returns it as a float64.
func (t *ThingBase) GetCeilY() float64 {
	return t.sector.CeilY
}

// Compute updates the Thing's direction and position based on the player's coordinates and its current speed.
func (t *ThingBase) Compute(playerX float64, playerY float64) {
	//nothing to do
}

// MoveApply adjusts the Thing's position and updates its sector affiliation and physical bounds accordingly.
func (t *ThingBase) MoveApply(tx float64, ty float64) {
	x, y := t.clipMovement(tx, ty)
	t.position.X += x
	t.position.Y += y
	if newSector := t.sectors.SectorSearch(t.sector, t.position.X, t.position.Y); newSector != nil {
		t.sector = newSector
	}
	// 4. Retro-Correzione (Sync-Back) AABB fisico
	r := t.entity.GetWidth() / 2.0
	t.entity.MoveTo(t.position.X-r, t.position.Y-r)
	t.entities.UpdateObject(t.entity)
}

// MoveEntityApply processes the logical and physical movement of the entity, adjusting positions and sector affiliations.
func (t *ThingBase) MoveEntityApply() {
	ex, ey := t.entity.GetCenterXY()
	// Passive Delta (bounces computed by SetupCollision)
	tx := ex - t.position.X
	ty := ey - t.position.Y
	// Active Delta (Kinematic Drive) added only if there is intentionality
	if t.entity.G > 0 {
		tx += t.entity.Vx
		ty += t.entity.Vy
	}
	if math.Abs(tx) > minMovement || math.Abs(ty) > minMovement {
		t.MoveApply(tx, ty)
	}
}

// clipMovement adjusts movement vectors to handle collisions with environment walls or obstacles in a 2D space.
// It takes initial deltas in X and Y directions (dx, dy) and returns the adjusted movement vector after collision checks.
func (t *ThingBase) clipMovement(velX float64, velY float64) (float64, float64) {
	// Things rest on the floor. We simulate head/knee height for elevation differences
	headPos := t.sector.FloorY + t.height
	kneePos := t.sector.FloorY + 2.0
	viewX, viewY := t.position.X, t.position.Y
	pX := viewX + velX
	pY := viewY + velY
	velX, velY = t.sector.ClipVelocity(viewX, viewY, pX, pY, velX, velY, headPos, kneePos)
	return velX, velY
}

// modifyDirection adjusts the velocity of the entity towards the specified direction with a constant acceleration factor.
func (t *ThingBase) modifyDirection(dirX, dirY float64) {
	const acceleration = 0.15
	t.entity.Vx = t.entity.Vx*(1-acceleration) + (dirX * acceleration)
	t.entity.Vy = t.entity.Vy*(1-acceleration) + (dirY * acceleration)
	if t.entity.GForce == 0 {
		t.entity.GForce = 1.0
	}
	if t.entity.Friction < 0.2 {
		t.entity.Friction = 0.99
	}
}
