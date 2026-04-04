package model

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/model/config"
	"github.com/markel1974/godoom/mr_tech/model/mathematic"
	"github.com/markel1974/godoom/mr_tech/physics"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// ThingBullet represents a specialized type of Thing designed to simulate projectile-like behavior in the environment.
type ThingBullet struct {
	*ThingBase
	wall        *physics.Entity
	floorStartY float64
}

// NewThingBullet creates and initializes a new ThingBullet instance with specific properties and links it to the game world.
// cfg specifies the configuration of the bullet, anim defines its animation, and sector represents its initial sector.
// sectors and entities provide references to all sectors and entities in the game world.
func NewThingBullet(cfg *config.ConfigThing, anim *textures.Animation, sector *Sector, sectors *Sectors, entities *Entities) *ThingBullet {
	p := &ThingBullet{
		ThingBase:   NewThingBase(cfg, anim, sector, sectors, entities),
		wall:        physics.NewEntity(0, 0, 0, 0, 0),
		floorStartY: sector.GetFloorY(),
	}
	// Annulla il decadimento inerziale per mantenere una velocità lineare costante
	p.entity.SetFriction(0.99)
	p.entity.SetGForce(1.0)
	p.entities.AddThing(p)

	// Calculate the directional vector based on the original firing angle
	dirX := math.Cos(p.angle) * p.speed
	dirY := math.Sin(p.angle) * p.speed

	const acceleration = 0.15
	p.entity.SetVx(p.entity.GetVx()*(1-acceleration) + (dirX * acceleration))
	p.entity.SetVy(p.entity.GetVy()*(1-acceleration) + (dirY * acceleration))
	return p
}

func (t *ThingBullet) GetFloorY() float64 {
	// 1. Magnitudo vettoriale corrente
	velSq := (t.entity.GetVx() * t.entity.GetVx()) + (t.entity.GetVy() * t.entity.GetVy())
	// Se l'energia cinetica è esaurita o malformata, il proiettile è a terra
	if velSq <= 0.01 || t.speed <= 0 {
		return t.sector.GetFloorY()
	}
	// 2. Fattore T di decadimento: velocità corrente normalizzata sulla velocità originale
	ratio := math.Sqrt(velSq) / t.speed
	// Clamping di sicurezza vettoriale in caso di impulsi esterni imprevisti
	if ratio <= 0 {
		return t.sector.GetFloorY()
	}
	if ratio > 1.0 {
		ratio = 1.0
	}
	// 3. LERP tra la quota del suolo e la quota di sparo
	currentY := t.floorStartY * ratio
	return currentY
}

// Compute updates the bullet's direction and handles its collision, potentially triggering its deallocation.
func (t *ThingBullet) Compute(playerX float64, playerY float64) {
	//if t.speed == 0 {
	//	return
	//}
	//if math.Abs(t.entity.GetVx()) < 0.1 && math.Abs(t.entity.GetVy()) < 0.1 {
	//	t.entity.SetVx(0)
	//	t.entity.Vy = 0
	//	t.speed = 0
	//	t.entity.Invalidate()
	//	//TODO REMOVE!!!!
	//	return
	//}

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
	tx += t.entity.GetVx()
	ty += t.entity.GetVy()
	//}
	if math.Abs(tx) > minMovement || math.Abs(ty) > minMovement {
		x, y := t.adjustPassage(tx, ty)
		t.position.X += x
		t.position.Y += y
		if newSector := t.sectors.SectorSearch(t.sector, t.position.X, t.position.Y); newSector != nil {
			t.sector = newSector
		}
		t.entities.UpdateThing(t, t.position.X, t.position.Y)
	}
}

// slidingMovement adjusts the movement velocity based on collisions and elevation differences in the current sector.
func (t *ThingBullet) adjustPassage(velX float64, velY float64) (float64, float64) {
	bottom := t.floorStartY
	top := bottom + t.height
	viewX, viewY := t.position.X, t.position.Y
	pX := viewX + velX
	pY := viewY + velY
	velX, velY = t.EffectBounce(viewX, viewY, pX, pY, velX, velY, top, bottom)
	return velX, velY
}

// EffectBounce calculates the resulting direction of a projectile after collision and applies bounce physics adjustments.
func (t *ThingBullet) EffectBounce(viewX, viewY, pX, pY, velX, velY, top, bottom float64) (float64, float64) {
	moveX := pX - viewX
	moveY := pY - viewY
	minT := 1.0
	var hit *Face = nil

	for _, seg := range t.sector.Segments {
		neighbor := seg.GetNeighbor()
		if neighbor != nil {
			if top > t.sector.GetCeilY() || bottom < t.sector.GetFloorY() {
				continue
			}
		}
		start := seg.GetStart()
		end := seg.GetEnd()
		dx := end.X - start.X
		dy := end.Y - start.Y
		den := moveX*dy - moveY*dx
		if den == 0 {
			continue
		}
		// Calcolo parametrico
		t1 := ((start.X-viewX)*dy - (start.Y-viewY)*dx) / den
		u1 := ((start.X-viewX)*moveY - (start.Y-viewY)*moveX) / den
		// CULLING: Memorizza l'impatto solo se è geometricamente il più vicino all'origine
		if t1 >= 0 && t1 <= minT && u1 >= 0 && u1 <= 1 {
			holeLow, holeHigh := 9e9, -9e9
			if neighbor != nil {
				holeLow = mathematic.MaxF(t.sector.GetFloorY(), neighbor.GetFloorY())
				holeHigh = mathematic.MinF(t.sector.GetCeilY(), neighbor.GetCeilY())
			}
			if holeHigh < top || holeLow > bottom {
				minT = t1
				hit = seg
			}
		}
	}
	// Risolvi l'impulso esclusivamente sulla faccia corretta
	if hit != nil {
		start := hit.GetStart()
		end := hit.GetEnd()
		dx := end.X - start.X
		dy := end.Y - start.Y
		lenSq := dx*dx + dy*dy
		// 1. Proiezione Ortogonale (Closest Point on Line Face)
		var cx, cy float64
		if lenSq > 0 {
			tProj := ((viewX-start.X)*dx + (viewY-start.Y)*dy) / lenSq
			tProj = math.Max(0, math.Min(1, tProj))
			cx = start.X + tProj*dx
			cy = start.Y + tProj*dy
		} else {
			cx, cy = start.X, start.Y
		}
		// 2. Istanziazione Static Body
		// cx, cy: Centro spoofato sul punto d'impatto per generare la normale perfetta
		// 0, 0: Width/Height nulli affinché Baumgarte usi solo il raggio del proiettile
		// 1e12: Massa infinita per assorbire l'impulso al 100% (InverseMass ~ 0)
		t.wall.Reset(cx, cy, 0, 0, 0, 1e12)
		// 3. Risoluzione Newton + Baumgarte
		t.entity.SetupCollision(t.wall)
		return t.entity.GetVx(), t.entity.GetVy()
	}
	return velX, velY
}

// OnCollide handles the interaction when the bullet collides with another object, applying damage and deactivating itself.
func (t *ThingBullet) OnCollide(other IThing) {
	//other.TakeDamage(t.damage)
	if enemy, ok := other.(*ThingEnemy); ok {
		// enemy.TakeDamage(...)
		_ = enemy
		// Marca il proiettile per la rimozione al prossimo frame
		//t.SetActive(false)
		// Spawn particellare / Suono impatto qui
	}
}

// IsActive checks if the ThingBullet is currently active and operational. Returns true if active, false otherwise.
func (t *ThingBullet) IsActive() bool {
	return t.isActive
}
