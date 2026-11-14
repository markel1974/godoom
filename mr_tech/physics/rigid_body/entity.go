package rigid_body

import (
	"sync/atomic"

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
	ecx              float64
	ecy              float64
	ecz              float64
	hasCentroid      bool
	eixx, eixy, eixz float64
	eiyy, eiyz, eizz float64
	hasInertia       bool
	*aabb.BoundingBox
	*Cinematic
}

// NewEntity creates and returns a pointer to a new Entity with specified mass, restitution, ground friction, and gravity force.
func NewEntity(mass, restitution, groundFriction float64, restCombine, frictionCombine int, gForce float64) *Entity {
	e := &Entity{
		id:          uint64(GetGlobalId()),
		BoundingBox: aabb.NewBoundingBox(),
		Cinematic:   NewCinematic(_dt, mass, restitution, groundFriction, restCombine, frictionCombine, gForce),
		hasCentroid: false,
	}
	return e
}

// GetId returns the unique identifier of the Entity as a uint64.
func (e *Entity) GetId() uint64 {
	return e.id
}

// Rebuild updates the bounding box position and dimensions and refreshes the cuboid inertia tensor.
func (e *Entity) Rebuild(x, y, z, w, h, d float64) {
	e.BoundingBox.Rebuild(x, y, z, w, h, d)
	e.Cinematic.UpdateInertia(w, h, d)
	if e.hasInertia {
		e.Cinematic.SetInertiaTensor(e.eixx, e.eixy, e.eixz, e.eiyy, e.eiyz, e.eizz)
	}
	e.Cinematic.SyncWorldInertia(e.BoundingBox.GetAxes())
}

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

func (e *Entity) SetInertia(ixx, ixy, ixz, iyy, iyz, izz float64) {
	e.eixx, e.eixy, e.eixz = ixx, ixy, ixz
	e.eiyy, e.eiyz, e.eizz = iyy, iyz, izz
	e.hasInertia = true
}

// SetKeepUpright enables or disables upright stabilization and the bounding box tilt limit.
func (e *Entity) SetKeepUpright(state bool) {
	e.BoundingBox.SetTiltLimitEnabled(state)
	e.Cinematic.SetKeepUpright(state)
}

// GetSize returns the width, height, and depth of the entity's bounding box as three float64 values.
func (e *Entity) GetSize() (float64, float64, float64) {
	return e.BoundingBox.GetSize()
}

// SetSize updates the bounding box dimensions and refreshes the cuboid inertia tensor.
func (e *Entity) SetSize(w, h, d float64) {
	e.BoundingBox.SetSize(w, h, d)
	e.Cinematic.UpdateInertia(w, h, d)
	e.Cinematic.SyncWorldInertia(e.BoundingBox.GetAxes())
}

// AddSize increments the bounding box dimensions and refreshes the cuboid inertia tensor.
func (e *Entity) AddSize(w, h, d float64) {
	e.BoundingBox.AddSize(w, h, d)
	e.Cinematic.UpdateInertia(e.GetWidth(), e.GetHeight(), e.GetDepth())
	e.Cinematic.SyncWorldInertia(e.BoundingBox.GetAxes())
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

// SyncRotation updates the OBB world axes and synchronizes the world-space inverse inertia tensor.
func (e *Entity) SyncRotation() {
	e.BoundingBox.SyncRotation()
	e.Cinematic.SyncWorldInertia(e.BoundingBox.GetAxes())
}

// AddForceAtPoint applies a linear force (fx, fy, fz) at world point (px, py, pz) and generates 3D torque τ = r × F on the OBB
// using exact Ray-OBB slab intersection and the world-space inverse inertia tensor I_world^-1 = R * I_body^-1 * R^T.
func (e *Entity) AddForceAtPoint(fx, fy, fz, px, py, pz float64) {
	cx, cy, cz := e.getCenterOfMass()
	e.Cinematic.AddForceAtPoint(fx, fy, fz, px, py, pz, cx, cy, cz)
}

// Update integrates linear and angular velocities, applies righting torque on Roll/Pitch, and updates the OBB quaternion.

func (e *Entity) Update() {
	e.BoundingBox.SnapshotRotation()
	if !e.Cinematic.KeepUpright() {
		e.Cinematic.SetIsTilted(!e.BoundingBox.IsFaceAligned())
	}
	e.Cinematic.Update(e.BoundingBox.GetAxes2())
	if e.Cinematic.GetInvMass() <= 0.0 ||
		(!e.Cinematic.IsRotating() && !e.Cinematic.IsTilted()) {
		return
	}
	// Preserve the world-space center of mass during rotation.
	cx, cy, cz := e.getCenterOfMass()
	reset := e.BoundingBox.Rotate(e.wx, e.wy, e.wz, e.GetDt())
	// Rotation changes the world-space COM when the local
	// centroid is offset from the geometric center.
	ncx, ncy, ncz := e.getCenterOfMass()
	e.BoundingBox.AddTo(cx-ncx, cy-ncy, cz-ncz)

	if reset {
		e.wx = 0.0
		e.wy = 0.0
	}

	// Synchronize world inertia using the final orientation.
	e.Cinematic.SyncWorldInertia(e.BoundingBox.GetAxes())
}

func (e *Entity) predictedContactCOMs(e2 *Entity) (float64, float64, float64, float64, float64, float64) {
	cx1, cy1, cz1 := e.getCenterOfMass()
	cx2, cy2, cz2 := e2.getCenterOfMass()
	dx1, dy1, dz1 := e.GetFrameDisplacement()
	dx2, dy2, dz2 := e2.GetFrameDisplacement()
	return cx1 + dx1, cy1 + dy1, cz1 + dz1, cx2 + dx2, cy2 + dy2, cz2 + dz2
}

// WarmStartImpact pre-applies persisted normal and tangential impulses for this entity pair before PGS iterations.
func (e *Entity) WarmStartImpact(e2 *Entity, nx, ny, nz float64, px, py, pz float64, accNormal, accTx, accTy, accTz *float64) {
	cx1, cy1, cz1, cx2, cy2, cz2 := e.predictedContactCOMs(e2)
	e.Cinematic.WarmStartImpact(e2.Cinematic, nx, ny, nz, cx1, cy1, cz1, cx2, cy2, cz2, px, py, pz, accNormal, accTx, accTy, accTz)
}

// ResolveImpact resolves a 6-DoF collision impact with another entity .
func (e *Entity) ResolveImpact(e2 *Entity, nx, ny, nz float64, px, py, pz float64) {
	e.ResolveImpactAccumulated(e2, nx, ny, nz, px, py, pz, nil, nil, nil, nil, nil, nil)
}

// ResolveImpactAccumulated resolves a 6-DoF collision impact with another entity using accumulated PGS impulse clamping.
func (e *Entity) ResolveImpactAccumulated(e2 *Entity, nx, ny, nz float64, px, py, pz float64, accNormal, accTx, accTy, accTz, restTarget *float64, hasRestTarget *bool) {
	cx1, cy1, cz1, cx2, cy2, cz2 := e.predictedContactCOMs(e2)
	e.Cinematic.ResolveImpactAccumulated(e2.Cinematic, nx, ny, nz, cx1, cy1, cz1, cx2, cy2, cz2, px, py, pz, accNormal, accTx, accTy, accTz, restTarget, hasRestTarget)
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
