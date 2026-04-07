package model

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/model/config"
	"github.com/markel1974/godoom/mr_tech/model/geometry"
	"github.com/markel1974/godoom/mr_tech/physics"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// ThingBase represents the fundamental attributes and behaviors of an object in the system.
type ThingBase struct {
	id         string
	position   geometry.XYZ
	mass       float64
	radius     float64
	height     float64
	angle      float64
	maxStep    float64
	kind       config.ThingType
	speed      float64
	volume     *Volume
	animation  *textures.Animation
	volumes    *Volumes
	entities   *Entities
	entity     *physics.Entity
	isActive   bool
	identifier int
	lastTx     float64
	lastTy     float64
	wall       *ThingWall
}

// NewThingBase creates a new ThingBase instance with specified configuration, animation, sector, volumes, and entities.
func NewThingBase(cfg *config.ConfigThing, pos geometry.XYZ, anim *textures.Animation, volume *Volume, volumes *Volumes, entities *Entities) *ThingBase {
	radius := cfg.Radius
	entX := pos.X - radius
	entY := pos.Y - radius
	entZ := pos.Z
	entW := radius * 2
	entH := radius * 2
	entD := cfg.Height // In 3D, la profondità è l'altezza reale dell'entità
	thing := &ThingBase{
		id:         cfg.Id,
		position:   pos,
		angle:      cfg.Angle,
		kind:       cfg.Kind,
		mass:       cfg.Mass,
		radius:     cfg.Radius,
		height:     cfg.Height,
		speed:      cfg.Speed,
		volume:     volume,
		animation:  anim,
		volumes:    volumes,
		entities:   entities,
		maxStep:    cfg.Height * 0.5,
		entity:     physics.NewEntity(entX, entY, entZ, entW, entH, entD, cfg.Mass),
		isActive:   true,
		identifier: -1,
		wall:       NewThingWall(volumes),
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

// GetPosition returns the X, Y, and Z coordinates of the ThingBase instance as a tuple of three float64 values.
func (t *ThingBase) GetPosition() (float64, float64, float64) {
	return t.position.X, t.position.Y, t.position.Z
}

// GetLight retrieves the Light object associated with the ThingBase's current sector.
func (t *ThingBase) GetLight() *Light {
	return t.volume.Light
}

// GetMinZ retrieves the minimum Z-coordinate (floor height) of the volume associated with the ThingBase instance.
func (t *ThingBase) GetMinZ() float64 {
	return t.volume.GetMinZ()
}

// GetMaxZ retrieves the maximum Z-coordinate (height) of the volume associated with the ThingBase instance.
func (t *ThingBase) GetMaxZ() float64 {
	return t.volume.GetMaxZ()
}

// Compute performs computations or updates related to the ThingBase object based on the player's coordinates.
func (t *ThingBase) Compute(playerX float64, playerY float64, playerZ float64) {
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
	// 1. Recupero dati dal motore impulsivo
	if !t.entity.IsMoving() {
		return
	}
	eX, eY, eZ := t.entity.GetCenter()
	currentBaseZ := eZ - (t.entity.GetDepth() / 2.0)
	// 2. Calcolo dei delta
	deltaX := (eX - t.position.X) + t.entity.GetVx()
	deltaY := (eY - t.position.Y) + t.entity.GetVy()
	deltaZ := (currentBaseZ - t.position.Z) + t.entity.GetVz()
	if deltaX == 0 && deltaY == 0 && deltaZ == 0 {
		return
	}
	viewX, viewY, viewZ := t.position.X, t.position.Y, t.position.Z
	zBottom := viewZ
	zTop := viewZ + t.height
	zMinLimit := t.volume.GetMinZ()
	zMaxLimit := t.volume.GetMaxZ() - t.height
	velX, velY, velZ, _ := t.wall.Compute(viewX, viewY, viewZ, deltaX, deltaY, deltaZ, zTop, zBottom, zMinLimit, zMaxLimit, t.radius, false)
	// 4. Applichiamo il movimento se significativo
	if math.Abs(velX) > minMovement || math.Abs(velY) > minMovement || math.Abs(velZ) > minMovement {
		//t.entity.SetVx(velX)
		//t.entity.SetVy(velY)
		//t.entity.SetVz(velZ)
		t.position.X += velX
		t.position.Y += velY
		t.position.Z += velZ
		baseZ := t.position.Z
		topZ := t.position.Z + t.height
		if newVolume := t.volumes.SearchVolume3d(t.volume, t.position.X, t.position.Y, baseZ, topZ, t.maxStep); newVolume != nil && newVolume != t.volume {
			t.volume = newVolume
		}
		t.entities.UpdateThing(t, t.position.X, t.position.Y, t.position.Z)
	} else {
		t.entity.Stop()
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
