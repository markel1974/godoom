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
	pos        geometry.XYZ
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
	things     *Things
	entity     *physics.Entity
	isActive   bool
	identifier int
	wall       *ThingWall
}

// NewThingBase creates a new ThingBase instance with specified configuration, animation, sector, volumes, and things.
func NewThingBase(cfg *config.ConfigThing, pos geometry.XYZ, anim *textures.Animation, volume *Volume, volumes *Volumes, things *Things) *ThingBase {
	entX := pos.X - cfg.Radius
	entY := pos.Y - cfg.Radius
	entZ := pos.Z
	entW := cfg.Radius * 2
	entH := cfg.Radius * 2
	entD := cfg.Height
	thing := &ThingBase{
		id:         cfg.Id,
		angle:      cfg.Angle,
		kind:       cfg.Kind,
		mass:       cfg.Mass,
		radius:     cfg.Radius,
		height:     cfg.Height,
		speed:      cfg.Speed,
		pos:        pos,
		volume:     volume,
		animation:  anim,
		volumes:    volumes,
		things:     things,
		maxStep:    cfg.Height * 0.5,
		entity:     physics.NewEntity(entX, entY, entZ, entW, entH, entD, cfg.Mass, cfg.Restitution, 0.9),
		isActive:   true,
		identifier: -1,
		wall:       NewThingWall(volumes, 0, 0),
	}
	thing.entity.SetOnGround(false)
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
	return t.pos.X, t.pos.Y, t.pos.Z
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

// PhysicsApply applies physics calculations to the ThingBase instance using its height attribute.
func (t *ThingBase) PhysicsApply() {
	t.doPhysics(t.height)
}

// doPhysics handles the physics computations for movement, collision detection, and volume transitions for the entity.
func (t *ThingBase) doPhysics(tHeight float64) {
	// extract displacement delta
	dx, dy, dz := t.entity.GetDisplacement()
	if dx == 0.0 && dy == 0.0 && dz == 0.0 {
		return
	}
	isGrounded := false
	pX, pY, pZ := t.pos.X+dx, t.pos.Y+dy, t.pos.Z+dz
	hPos := t.pos.Z + tHeight
	elevBaseZ := t.pos.Z + t.maxStep
	// continuous collision detection (ccd) & sliding
	face, nX, nY, nZ := t.wall.ClosestFace(t.pos.X, t.pos.Y, t.pos.Z, pX, pY, pZ, dx, dy, dz, hPos, elevBaseZ, t.radius)
	if face != nil {
		// apply physical response to the entity
		t.entity.ResolveImpact(t.wall.GetEntity(), nX, nY, nZ)
		vx, vy, vz := t.entity.GetVelocity()
		// handle landing on walkable planes (slope)
		if nZ >= 0.7 && vz < 0 {
			vz = 0
			isGrounded = true
		}
		// project residual velocity onto tangent plane (sliding kcc)
		newVx, newVy, newVz := t.entity.ClipVelocity(vx, vy, vz, nX, nY, nZ)
		t.entity.SetVx(newVx)
		t.entity.SetVy(newVy)
		t.entity.SetVz(newVz)
		// recalculate effective displacement for current frame
		dx, dy, dz = t.entity.GetDisplacement()
		pX, pY, pZ = t.pos.X+dx, t.pos.Y+dy, t.pos.Z+dz
	}
	// volume transition (3d portals)
	topZ := pZ + tHeight
	newVolume := t.volumes.SearchVolume3d(t.volume, pX, pY, pZ, topZ, t.maxStep)
	if newVolume != nil && newVolume != t.volume {
		if t.entity.GetVz() <= 0 {
			actualStep := newVolume.GetMinZ() - t.volume.GetMinZ()
			// automatic handling of height difference (step-up for stairs)
			if actualStep > 0 || (actualStep < 0 && math.Abs(actualStep) < t.maxStep) {
				pZ = newVolume.GetMinZ()
				t.entity.SetVz(0.0)
				isGrounded = true
			}
		}
		t.volume = newVolume
	}
	// vertical topological limits
	minZ, maxZ := t.volume.GetMinZ(), t.volume.GetMaxZ()
	if pZ <= minZ {
		pZ = minZ
		isGrounded = true
		t.entity.SetVz(0.0)
	} else if (pZ + tHeight) > maxZ {
		t.entity.ResolveImpact(t.wall.GetEntity(), 0, 0, -1)
		pZ = maxZ - tHeight
	}
	// physical state synchronization
	t.entity.SetOnGround(isGrounded)
	// final application
	t.pos.X, t.pos.Y, t.pos.Z = pX, pY, pZ
	t.things.UpdateThing(t, t.pos.X, t.pos.Y, t.pos.Z)
}
