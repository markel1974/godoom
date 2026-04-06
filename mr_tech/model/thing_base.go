package model

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/model/config"
	"github.com/markel1974/godoom/mr_tech/model/geometry"
	"github.com/markel1974/godoom/mr_tech/physics"
	"github.com/markel1974/godoom/mr_tech/textures"
)

//TODO
//Dobbiamo far migrare thing_base.go affinché usi il nuovo Volumes.SearchVolume(v, x, y, z)
//per l'aggiornamento spaziale e riscrivere il wall-sliding per operare contro i piani 3D.

// ThingBase represents the fundamental attributes and behaviors of an object in the system.
type ThingBase struct {
	id         string
	position   geometry.XY
	mass       float64
	radius     float64
	height     float64
	angle      float64
	kind       config.ThingType
	speed      float64
	volume     *Volume
	animation  *textures.Animation
	sectors    *Volumes
	entities   *Entities
	entity     *physics.Entity
	isActive   bool
	identifier int
}

// NewThingBase creates a new ThingBase instance with specified configuration, animation, sector, sectors, and entities.
func NewThingBase(cfg *config.ConfigThing, anim *textures.Animation, volume *Volume, sectors *Volumes, entities *Entities) *ThingBase {
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
		volume:     volume,
		animation:  anim,
		sectors:    sectors,
		entities:   entities,
		entity:     physics.NewEntity(x, y, 1.0, w, h, cfg.Mass),
		isActive:   true,
		identifier: -1,
	}
	return thing
}

// GetId returns the identifier string of the ThingBase instance.
func (t *ThingBase) GetId() string {
	return t.id
}

// GetKind returns the type of the ThingBase as a value of the ThingType enumeration.
func (t *ThingBase) GetKind() config.ThingType {
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

// GetVolume retrieves the current volume associated with the ThingBase instance.
func (t *ThingBase) GetVolume() *Volume {
	return t.volume
}

func (t *ThingBase) GetPosition() (float64, float64) {
	return t.position.X, t.position.Y
}

// GetLight retrieves the Light object associated with the ThingBase's current sector.
func (t *ThingBase) GetLight() *Light {
	return t.volume.Light
}

// GetFloorY returns the floor Y-coordinate of the sector associated with the ThingBase instance.
func (t *ThingBase) GetFloorY() float64 {
	return t.volume.GetFloorY()
}

// GetCeilY returns the ceiling height of the sector associated with the ThingBase instance.
func (t *ThingBase) GetCeilY() float64 {
	return t.volume.GetCeilY()
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
		if newVolume := t.sectors.SearchVolume2d(t.volume, t.position.X, t.position.Y); newVolume != nil {
			t.volume = newVolume
		}
		t.entities.UpdateThing(t, t.position.X, t.position.Y)
	}
}

// OnCollide handles interactions when the current object collides with another object implementing the IThing interface.
func (t *ThingBase) OnCollide(other IThing) {
	//fmt.Println("COLLISION -> ", other.GetId())
}

// IsActive checks if the ThingBase instance is currently active.
func (t *ThingBase) IsActive() bool {
	return t.isActive
}

// SetActive updates the activation state of the ThingBase instance and returns the updated state as a boolean.
func (t *ThingBase) SetActive(active bool) {
	t.isActive = active
}

// adjustPassage adjusts X and Y velocities to account for wall collisions and elevation differences such as floor height.
func (t *ThingBase) adjustPassage(velX float64, velY float64) (float64, float64) {
	// Things rest on the floor. We simulate head/knee height for elevation differences
	top := t.volume.GetFloorY() + t.height
	bottom := t.volume.GetFloorY() + 2.0
	viewX, viewY := t.position.X, t.position.Y
	pX := viewX + velX
	pY := viewY + velY
	radius := t.entity.GetWidth() / 2
	velX, velY = WallSlidingEffect(t.volume, viewX, viewY, pX, pY, velX, velY, top, bottom, radius)
	return velX, velY
}
