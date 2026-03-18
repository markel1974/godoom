package model

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/physics"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// ThingBase represents the base structure for all entities in the system, defining their physical and graphical properties.
type ThingBase struct {
	id         string
	position   XY
	mass       float64
	radius     float64
	height     float64
	angle      float64
	kind       ThingType
	speed      float64
	sector     *Sector
	animation  *textures.Animation
	sectors    *Sectors
	entities   *Entities
	entity     *physics.Entity
	identifier int
}

// NewThingBase creates and initializes a new ThingBase with the specified configuration, animation, sector, and entities.
func NewThingBase(cfg *ConfigThing, anim *textures.Animation, sector *Sector, sectors *Sectors, entities *Entities) *ThingBase {
	w := cfg.Radius * 2
	h := cfg.Radius * 2
	x := cfg.Position.X - cfg.Radius
	y := cfg.Position.Y - cfg.Radius
	thing := &ThingBase{
		id:         cfg.Id,
		position:   cfg.Position,
		angle:      cfg.Angle,
		kind:       cfg.Kind,
		mass:       cfg.Mass,
		radius:     cfg.Radius,
		height:     cfg.Height,
		speed:      cfg.Speed,
		sector:     sector,
		animation:  anim,
		sectors:    sectors,
		entities:   entities,
		entity:     physics.NewEntity(x, y, w, h, cfg.Mass),
		identifier: -1,
	}
	return thing
}

// GetId returns the unique identifier (id) of the ThingBase instance as a string.
func (t *ThingBase) GetId() string {
	return t.id
}

// GetKind returns the integer value representing the kind of the ThingBase.
func (t *ThingBase) GetKind() ThingType {
	return t.kind
}

// GetAABB retrieves the axis-aligned bounding box (AABB) associated with the ThingBase's physics entity.
func (t *ThingBase) GetAABB() *physics.AABB {
	return t.entity.GetAABB()
}

// GetEntity returns the physics.Entity associated with the ThingBase instance.
func (t *ThingBase) GetEntity() *physics.Entity {
	return t.entity
}

// GetAnimation returns the current animation associated with the instance of ThingBase.
func (t *ThingBase) GetAnimation() *textures.Animation {
	return t.animation
}

// GetSector returns the current sector associated with the ThingBase instance.
func (t *ThingBase) GetSector() *Sector {
	return t.sector
}

// GetPosition returns the X and Y coordinates of the ThingBase's position.
func (t *ThingBase) GetPosition() (float64, float64) {
	return t.position.X, t.position.Y
}

// GetLight retrieves the Light object associated with the current ThingBase instance's sector.
func (t *ThingBase) GetLight() *Light {
	return t.sector.Light
}

// GetFloorY returns the Y-coordinate of the floor level in the current sector associated with the ThingBase instance.
func (t *ThingBase) GetFloorY() float64 {
	return t.sector.FloorY
}

// GetCeilY returns the ceiling Y-coordinate of the ThingBase's associated sector.
func (t *ThingBase) GetCeilY() float64 {
	return t.sector.CeilY
}

// Compute updates the ThingBase object based on the player's position provided as playerX and playerY.
func (t *ThingBase) Compute(playerX float64, playerY float64) {
	//nothing to do
}

// SetIdentifier sets the identifier field of the ThingBase instance to the specified integer value.
func (t *ThingBase) SetIdentifier(identifier int) {
	t.identifier = identifier
}

// GetIdentifier retrieves the integer identifier associated with the ThingBase instance.
func (t *ThingBase) GetIdentifier() int {
	return t.identifier
}

// PhysicsApply updates the entity's position based on passive and active deltas, ensuring movement exceeds a minimum threshold.
func (t *ThingBase) PhysicsApply() {
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
		t.moveApply(tx, ty)
	}
}

// MoveApply updates the position of the object by applying the given translation vector (tx, ty) with movement constraints.
func (t *ThingBase) moveApply(tx float64, ty float64) {
	x, y := t.slidingMovement(tx, ty)
	t.position.X += x
	t.position.Y += y
	if newSector := t.sectors.SectorSearch(t.sector, t.position.X, t.position.Y); newSector != nil {
		t.sector = newSector
	}
	t.entities.UpdateThing(t, t.position.X, t.position.Y)
}

// slidingMovement adjusts the movement velocity based on collisions and elevation differences in the current sector.
func (t *ThingBase) slidingMovement(velX float64, velY float64) (float64, float64) {
	// Things rest on the floor. We simulate head/knee height for elevation differences
	headPos := t.sector.FloorY + t.height
	kneePos := t.sector.FloorY + 2.0
	viewX, viewY := t.position.X, t.position.Y
	pX := viewX + velX
	pY := viewY + velY
	velX, velY = t.sector.EffectSliding(viewX, viewY, pX, pY, velX, velY, headPos, kneePos)
	return velX, velY
}
