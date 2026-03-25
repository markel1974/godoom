package model

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/physics"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// ThingBase represents the fundamental attributes and behaviors of an object in the system.
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

// NewThingBase creates a new ThingBase instance with specified configuration, animation, sector, sectors, and entities.
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

// GetId returns the identifier string of the ThingBase instance.
func (t *ThingBase) GetId() string {
	return t.id
}

// GetKind returns the type of the ThingBase as a value of the ThingType enumeration.
func (t *ThingBase) GetKind() ThingType {
	return t.kind
}

// GetAABB retrieves the axis-aligned bounding box (AABB) of the associated physics entity.
func (t *ThingBase) GetAABB() *physics.AABB {
	return t.entity.GetAABB()
}

// GetEntity returns the physics.Entity associated with the current ThingBase instance.
func (t *ThingBase) GetEntity() *physics.Entity {
	return t.entity
}

// GetAnimation returns the animation associated with the ThingBase instance.
func (t *ThingBase) GetAnimation() *textures.Animation {
	return t.animation
}

// GetSector retrieves the current sector associated with the ThingBase instance.
func (t *ThingBase) GetSector() *Sector {
	return t.sector
}

func (t *ThingBase) GetPosition() (float64, float64) {
	return t.position.X, t.position.Y
}

// GetLight retrieves the Light object associated with the ThingBase's current sector.
func (t *ThingBase) GetLight() *Light {
	return t.sector.Light
}

// GetFloorY returns the floor Y-coordinate of the sector associated with the ThingBase instance.
func (t *ThingBase) GetFloorY() float64 {
	return t.sector.FloorY
}

// GetCeilY returns the ceiling height of the sector associated with the ThingBase instance.
func (t *ThingBase) GetCeilY() float64 {
	return t.sector.CeilY
}

// Compute performs computations or updates related to the ThingBase object based on the player's coordinates.
func (t *ThingBase) Compute(playerX float64, playerY float64) {
	//nothing to do
}

// SetIdentifier sets the unique identifier for the ThingBase instance.
func (t *ThingBase) SetIdentifier(identifier int) {
	t.identifier = identifier
}

// GetIdentifier returns the unique identifier of the ThingBase instance.
func (t *ThingBase) GetIdentifier() int {
	return t.identifier
}

// PhysicsApply updates the position of the object based on passive and active physics-driven deltas.
func (t *ThingBase) PhysicsApply() {
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

// adjustPassage adjusts X and Y velocities to account for wall collisions and elevation differences such as floor height.
func (t *ThingBase) adjustPassage(velX float64, velY float64) (float64, float64) {
	// Things rest on the floor. We simulate head/knee height for elevation differences
	top := t.sector.FloorY + t.height
	bottom := t.sector.FloorY + 2.0
	viewX, viewY := t.position.X, t.position.Y
	pX := viewX + velX
	pY := viewY + velY
	radius := t.entity.GetWidth() / 2
	velX, velY = WallSlidingEffect(t.sector, viewX, viewY, pX, pY, velX, velY, top, bottom, radius)
	return velX, velY
}
