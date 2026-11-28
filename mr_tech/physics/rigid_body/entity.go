package rigid_body

import (
	"math"
	"sync/atomic"

	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/physics/aabb"
)

var _dt float64
var _fps uint64

// SetFps sets the desired frames per second (fps) and adjusts the internal frame time interval (_dt) accordingly.
func SetFps(fps uint64) {
	_fps = fps
	_dt = 1.0 / float64(fps) //_dt = 1.0 / (float64(fps) * 0.5)
}

// GetFps returns the current frames-per-second (FPS) value as an unsigned 64-bit integer.
func GetFps() uint64 {
	return _fps
}

// _globalId is an internal counter used to generate unique identifiers in a thread-safe manner.
var _globalId int64

// GetGlobalId generates and returns a unique globally incremental identifier using atomic operations.
func GetGlobalId() int64 {
	return atomic.AddInt64(&_globalId, 1)
}

func init() {
	_globalId = -1
	SetFps(120)
}

// Entity represents a game object with a unique identifier, bounding box, and cinematic behaviors.
type Entity struct {
	id               uint64
	renderMarker     int
	shadowMarker     int
	ecx              float64
	ecy              float64
	ecz              float64
	hasCentroid      bool
	eixx, eixy, eixz float64
	eiyy, eiyz, eizz float64
	hasInertia       bool
	hasActiveContact bool
	contactNX        float64
	contactNY        float64
	contactNZ        float64
	contactPX        float64
	contactPY        float64
	contactPZ        float64
	*aabb.BoundingBox
	*Cinematic
}

// NewEntity creates and returns a pointer to a new Entity with specified mass, restitution, ground friction, and gravity force.
func NewEntity(mass, restitution, groundFriction float64, restCombine, frictionCombine int, gForce float64) *Entity {
	e := &Entity{
		id:           uint64(GetGlobalId()),
		renderMarker: -1,
		shadowMarker: -1,
		BoundingBox:  aabb.NewBoundingBox(),
		Cinematic:    NewCinematic(_dt, mass, restitution, groundFriction, restCombine, frictionCombine, gForce),
		hasCentroid:  false,
	}
	return e
}

// GetId returns the unique identifier of the Entity as a uint64.
func (e *Entity) GetId() uint64 {
	return e.id
}

// HasDynamicRestFloorGap reports whether the entity is within a dynamic rest floor gap threshold.
func (e *Entity) HasDynamicRestFloorGap(selfDZ, gap, partnerDZ float64) bool {
	const sixDofFloorGapSlop = 0.025
	dt := e.GetDt()
	gravVel := e.GetGForce() * dt
	maxDZ := gravVel*dt*1.5 + 0.01
	maxVZ := gravVel*1.35 + 0.5
	return selfDZ >= -maxDZ && e.GetVz() >= -maxVZ && gap <= sixDofFloorGapSlop+math.Max(0.0, -partnerDZ)
}

// GetRenderMarker returns the renderer cache slot index assigned to this Entity, or -1 if unassigned.
func (e *Entity) GetRenderMarker() int {
	return e.renderMarker
}

// SetRenderMarker assigns the renderer cache slot index for this Entity.
func (e *Entity) SetRenderMarker(marker int) {
	e.renderMarker = marker
}

// GetShadowMarker returns the shadow cache slot index assigned to this Entity, or -1 if unassigned.
func (e *Entity) GetShadowMarker() int {
	return e.shadowMarker
}

// GetHash returns a nonzero stable hash derived from the entity ID, center position, rotation, and yaw.
func (e *Entity) GetHash(vHash uint64) uint64 {
	tPosX, tPosY, zBot := e.GetCenter()
	qx, qy, qz, qw := e.GetQRotation().Get()
	ang := e.GetYaw()
	h := geometry.MixU64(e.GetId(), vHash)
	h = geometry.MixU64(h, geometry.F32BitsStable(float32(tPosX)))
	h = geometry.MixU64(h, geometry.F32BitsStable(float32(tPosY)))
	h = geometry.MixU64(h, geometry.F32BitsStable(float32(zBot)))
	h = geometry.MixU64(h, geometry.F32BitsStable(float32(qx)))
	h = geometry.MixU64(h, geometry.F32BitsStable(float32(qy)))
	h = geometry.MixU64(h, geometry.F32BitsStable(float32(qz)))
	h = geometry.MixU64(h, geometry.F32BitsStable(float32(qw)))
	h = geometry.MixU64(h, geometry.F32BitsStable(float32(ang)))
	return h
}

// SetShadowMarker assigns the shadow cache slot index for this Entity.
func (e *Entity) SetShadowMarker(marker int) {
	e.shadowMarker = marker
}

// syncInertia updates the cuboid fallback inertia for (w, h, d), reapplies any custom inertia tensor, and syncs world tensors.
func (e *Entity) syncInertia(w, h, d float64) {
	e.Cinematic.UpdateInertia(w, h, d)
	if e.hasInertia {
		e.Cinematic.SetInertiaTensor(e.eixx, e.eixy, e.eixz, e.eiyy, e.eiyz, e.eizz)
	}
	e.Cinematic.SyncWorldInertia(e.BoundingBox.GetAxes())
}

// Rebuild updates the bounding box position and dimensions and refreshes the inertia tensor.
func (e *Entity) Rebuild(x, y, z, w, h, d float64) {
	e.BoundingBox.Rebuild(x, y, z, w, h, d)
	e.syncInertia(w, h, d)
}

// SetCentroid sets the entity centroid to the given world-space coordinates.
func (e *Entity) SetCentroid(cx, cy, cz float64) {
	gx, gy, gz := e.BoundingBox.GetCenter()
	dx := cx - gx
	dy := cy - gy
	dz := cz - gz
	ax0, ay0, az0, ax1, ay1, az1, ax2, ay2, az2 := e.BoundingBox.GetAxes()
	// World space -> local space.
	e.ecx = dx*ax0 + dy*ay0 + dz*az0
	e.ecy = dx*ax1 + dy*ay1 + dz*az1
	e.ecz = dx*ax2 + dy*ay2 + dz*az2
	e.hasCentroid = true
}

// SetCentroidLocal sets the center of mass offset relative to the bounding box center in local body space.
func (e *Entity) SetCentroidLocal(ecx, ecy, ecz float64) {
	e.ecx = ecx
	e.ecy = ecy
	e.ecz = ecz
	e.hasCentroid = true
}

// SetInertia sets a custom body-space inertia tensor per unit mass and applies it immediately if valid.
func (e *Entity) SetInertia(ixx, ixy, ixz, iyy, iyz, izz float64) {
	e.eixx, e.eixy, e.eixz = ixx, ixy, ixz
	e.eiyy, e.eiyz, e.eizz = iyy, iyz, izz
	e.hasInertia = true
	if e.Cinematic.SetInertiaTensor(ixx, ixy, ixz, iyy, iyz, izz) {
		e.Cinematic.SyncWorldInertia(e.BoundingBox.GetAxes())
	}
}

// SetKeepUpright enables or disables upright stabilization and the bounding box tilt limit.
func (e *Entity) SetKeepUpright(state bool) {
	if !state && e.Cinematic.IsUprightLocked() {
		return
	}
	if e.Cinematic.KeepUpright() != state {
		e.Cinematic.WakeUp()
	}
	e.BoundingBox.SetTiltLimitEnabled(state)
	e.Cinematic.SetKeepUpright(state)
}

// LockUpright keeps the entity permanently upright with tilt limits enabled.
func (e *Entity) LockUpright() {
	e.Cinematic.LockUpright()
	e.BoundingBox.SetTiltLimitEnabled(true)
}

// GetSize returns the width, height, and depth of the entity's bounding box as three float64 values.
func (e *Entity) GetSize() (float64, float64, float64) {
	return e.BoundingBox.GetSize()
}

// SetSize updates the bounding box dimensions and refreshes the inertia tensor.
func (e *Entity) SetSize(w, h, d float64) {
	e.BoundingBox.SetSize(w, h, d)
	e.syncInertia(w, h, d)
}

// AddSize increments the bounding box dimensions and refreshes the inertia tensor.
func (e *Entity) AddSize(w, h, d float64) {
	e.BoundingBox.AddSize(w, h, d)
	e.syncInertia(e.GetWidth(), e.GetHeight(), e.GetDepth())
}

// SetRotationAngleZ sets the OBB orientation to a yaw rotation around Z and synchronizes the world inertia tensor.
func (e *Entity) SetRotationAngleZ(angleRad float64) {
	e.Cinematic.WakeUp()
	e.BoundingBox.SetRotationAngleZ(angleRad)
	e.Cinematic.SyncWorldInertia(e.BoundingBox.GetAxes())
}

// SetRotationEuler sets the OBB orientation from Euler angles and synchronizes the world inertia tensor.
func (e *Entity) SetRotationEuler(rollX, pitchY, yawZ float64) {
	e.Cinematic.WakeUp()
	e.BoundingBox.SetRotationEuler(rollX, pitchY, yawZ)
	e.Cinematic.SyncWorldInertia(e.BoundingBox.GetAxes())
}

// SetRotationQuaternion sets the OBB orientation from a unit quaternion (x, y, z, w) and synchronizes the world inertia tensor.
func (e *Entity) SetRotationQuaternion(x, y, z, w float64) {
	e.Cinematic.WakeUp()
	e.BoundingBox.SetRotationQuaternion(x, y, z, w)
	e.Cinematic.SyncWorldInertia(e.BoundingBox.GetAxes())
}

// SyncRotation updates the OBB world axes and synchronizes the world-space inverse inertia tensor.
func (e *Entity) SyncRotation() {
	e.BoundingBox.SyncRotation()
	e.Cinematic.SyncWorldInertia(e.BoundingBox.GetAxes())
}

// AddForceAtPoint applies a force at the given point relative to the entity's center of mass.
func (e *Entity) AddForceAtPoint(fx, fy, fz, px, py, pz float64) {
	cx, cy, cz := e.getCenterOfMass()
	e.Cinematic.AddForceAtPoint(fx, fy, fz, px, py, pz, cx, cy, cz)
}

// SetActiveContact scopes the contact normal and point used by ComputeKineticImpact during collision callbacks.
func (e *Entity) SetActiveContact(nx, ny, nz, px, py, pz float64) {
	e.contactNX, e.contactNY, e.contactNZ = nx, ny, nz
	e.contactPX, e.contactPY, e.contactPZ = px, py, pz
	e.hasActiveContact = true
}

// ClearActiveContact clears the active contact scope after a collision callback completes.
func (e *Entity) ClearActiveContact() {
	e.hasActiveContact = false
}

// ComputeNormalKineticImpactAt computes physical impact energy (Joules, 50 units/m) between this entity
// and another along contact normal (nx, ny, nz) at world contact point (px, py, pz).
func (e *Entity) ComputeNormalKineticImpactAt(otherEntity *Entity, nx, ny, nz, px, py, pz float64) float64 {
	if otherEntity == nil {
		return 0
	}
	cx1, cy1, cz1 := e.getCenterOfMass()
	cx2, cy2, cz2 := otherEntity.getCenterOfMass()
	return e.Cinematic.ComputeNormalKineticImpact(otherEntity.Cinematic, nx, ny, nz, cx1, cy1, cz1, cx2, cy2, cz2, px, py, pz)
}

// ComputeStaticNormalKineticImpactAt computes physical impact energy (Joules, 50 units/m) between this entity
// and a fixed static surface along contact normal (nx, ny, nz) at world contact point (px, py, pz).
func (e *Entity) ComputeStaticNormalKineticImpactAt(nx, ny, nz, px, py, pz float64) float64 {
	cx1, cy1, cz1 := e.getCenterOfMass()
	return e.Cinematic.ComputeStaticNormalKineticImpact(nx, ny, nz, cx1, cy1, cz1, px, py, pz)
}

// ComputeKineticImpact calculates the physical kinetic impact energy (Joules, 50 units/m) between this entity and another.
// When called inside a collision callback with an active contact, it evaluates pre-solve closing normal speed at the contact point.
// It returns 0 if either entity has non-positive mass.
func (e *Entity) ComputeKineticImpact(otherEntity *Entity) float64 {
	if otherEntity == nil {
		return 0
	}
	if e.hasActiveContact {
		return e.ComputeNormalKineticImpactAt(otherEntity, e.contactNX, e.contactNY, e.contactNZ, e.contactPX, e.contactPY, e.contactPZ)
	}
	ovx, ovy, ovz := otherEntity.GetVelocity()
	svx, svy, svz := e.GetVelocity()
	otherMass := otherEntity.GetMass()
	selfMass := e.GetMass()
	dvx := ovx - svx
	dvy := ovy - svy
	dvz := ovz - svz
	relativeSpeedSq := dvx*dvx + dvy*dvy + dvz*dvz
	totalMass := otherMass + selfMass
	if otherMass <= 0 || selfMass <= 0 || totalMass <= 0 {
		return 0
	}
	reducedMass := (otherMass * selfMass) / totalMass
	return 0.5 * reducedMass * relativeSpeedSq * invWorldUnitsSq
}

// Update integrates linear and angular velocities, applies righting torque on Roll/Pitch, and updates the OBB quaternion.
func (e *Entity) Update() {
	e.BoundingBox.SnapshotRotation()
	if !e.Cinematic.KeepUpright() {
		e.Cinematic.SetIsTilted(!e.BoundingBox.IsFaceAligned())
	}
	e.Cinematic.Update(e.BoundingBox.GetAxes2())
	if e.Cinematic.GetInvMass() <= 0.0 || (!e.Cinematic.IsRotating() && !e.Cinematic.IsTilted()) {
		return
	}
	// Preserve the world-space center of mass during rotation.
	cx, cy, cz := e.getCenterOfMass()
	reset := e.BoundingBox.Rotate(e.wx, e.wy, e.wz, e.GetDt())
	// Rotation changes the world-space COM when the local centroid is offset from the geometric center.
	ncx, ncy, ncz := e.getCenterOfMass()
	e.BoundingBox.AddTo(cx-ncx, cy-ncy, cz-ncz)

	if reset {
		e.wx = 0.0
		e.wy = 0.0
	}

	// Synchronize world inertia using the final orientation.
	e.Cinematic.SyncWorldInertia(e.BoundingBox.GetAxes())
}

// predictedContactCOMs returns the next-frame centers of mass for this entity and the given entity.
func (e *Entity) predictedContactCOMs(e2 *Entity) (float64, float64, float64, float64, float64, float64) {
	cx1, cy1, cz1 := e.getCenterOfMass()
	cx2, cy2, cz2 := e2.getCenterOfMass()
	dx1, dy1, dz1 := e.GetFrameDisplacement()
	dx2, dy2, dz2 := e2.GetFrameDisplacement()
	return cx1 + dx1, cy1 + dy1, cz1 + dz1, cx2 + dx2, cy2 + dy2, cz2 + dz2
}

// WarmStartImpact pre-applies persisted normal and tangential impulses for this entity pair before PGS iterations.
func (e *Entity) WarmStartImpact(e2 *Entity, nx, ny, nz float64, px, py, pz float64, acc *Accumulator) {
	cx1, cy1, cz1, cx2, cy2, cz2 := e.predictedContactCOMs(e2)
	e.Cinematic.WarmStartImpact(e2.Cinematic, nx, ny, nz, cx1, cy1, cz1, cx2, cy2, cz2, px, py, pz, acc)
}

// ResolveImpact resolves a 6-DoF collision impact with another entity.
func (e *Entity) ResolveImpact(e2 *Entity, nx, ny, nz float64, px, py, pz float64) {
	e.ResolveImpactAccumulated(e2, nx, ny, nz, px, py, pz, nil)
}

// ResolveImpactAccumulated resolves a 6-DoF collision impact with another entity using accumulated PGS impulse clamping.
func (e *Entity) ResolveImpactAccumulated(e2 *Entity, nx, ny, nz float64, px, py, pz float64, acc *Accumulator) {
	cx1, cy1, cz1, cx2, cy2, cz2 := e.predictedContactCOMs(e2)
	e.Cinematic.ResolveImpactAccumulated(e2.Cinematic, nx, ny, nz, cx1, cy1, cz1, cx2, cy2, cz2, px, py, pz, acc)
}

// ResolveElasticPenetration separates overlapping entities using an elastic, inverse-mass-weighted correction.
// It honors penetration slop and avoids pushing grounded entities downward.
func (e *Entity) ResolveElasticPenetration(other *Entity, otherFloor bool, nx, ny, nz float64, penetration float64) {
	e.ResolveElasticPenetrationConstrained(other, otherFloor, nx, ny, nz, penetration, true, true)
}

// ResolveElasticPenetrationConstrained separates overlapping entities while preventing either body from being pushed
// along a direction blocked by static obstacles or pinned neighbors.
func (e *Entity) ResolveElasticPenetrationConstrained(other *Entity, otherFloor bool, nx, ny, nz float64, penetration float64, selfMovable, otherMovable bool) {
	p1, p2, ok := e.Cinematic.ResolveElasticPenetrationConstrained(other.Cinematic, otherFloor, nx, ny, nz, penetration, selfMovable, otherMovable)
	if !ok {
		return
	}
	if p1 > 0.0 {
		e.AddTo(nx*p1, ny*p1, nz*p1)
	}
	if p2 > 0.0 {
		other.AddTo(-nx*p2, -ny*p2, -nz*p2)
	}
}

// ResolveInelasticPenetration moves the entity along the collision normal to clear an inelastic overlap.
// It ignores nonpositive penetration and applies vertical slop for upward corrections.
func (e *Entity) ResolveInelasticPenetration(nx, ny, nz float64, pen float64) {
	p, ok := e.Cinematic.ResolveInelasticPenetration(nz, pen)
	if !ok {
		return
	}
	e.AddTo(nx*p, ny*p, nz*p)
}

// HalfExtents returns the collision half-extents for the entity's current upright state and orientation.
func (e *Entity) HalfExtents() (float64, float64, float64) {
	if e.KeepUpright() {
		w, h, d := e.GetSize()
		r := 0.25 * (w + h)
		return r, r, d * 0.5
	}
	return e.GetOrientedHalfExtents()
}

// getCenterOfMass returns the entity's world-space center of mass.
// It returns the bounding box center when no custom centroid is set.
func (e *Entity) getCenterOfMass() (float64, float64, float64) {
	gx, gy, gz := e.BoundingBox.GetCenter()
	if !e.hasCentroid {
		return gx, gy, gz
	}
	ax0, ay0, az0, ax1, ay1, az1, ax2, ay2, az2 := e.BoundingBox.GetAxes()
	return gx + e.ecx*ax0 + e.ecy*ax1 + e.ecz*ax2, gy + e.ecx*ay0 + e.ecy*ay1 + e.ecz*ay2, gz + e.ecx*az0 + e.ecy*az1 + e.ecz*az2
}
