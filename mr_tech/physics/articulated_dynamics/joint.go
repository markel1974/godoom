package articulated_dynamics

import "math"

// Joint defines a 3D skeletal joint specification with ragdoll limits, damping, stress, and detachment state.
type Joint struct {
	localCenterX    float64
	localCenterY    float64
	localCenterZ    float64
	partRadius      float64
	maxAngleRagdoll float64
	damping         float64
	canDetach       bool
	detachThreshold float64

	damage   float64
	detached bool
}

// NewJoint2 creates a Joint specification with default geometric radius.
func NewJoint2(partRadius float64) *Joint {
	return &Joint{
		partRadius: partRadius,
	}
}

// GetMaxAngleRagdoll returns the cone-limit half-angle (radians) used by ragdoll PGS joints.
func (j *Joint) GetMaxAngleRagdoll() float64 {
	return j.maxAngleRagdoll
}

// GetDamping returns the angular damping coefficient for this joint.
func (j *Joint) GetDamping() float64 {
	return j.damping
}

// CanDetach reports whether this joint may sever under high impact stress.
func (j *Joint) CanDetach() bool {
	return j.canDetach
}

// GetDetachThreshold returns the accumulated stress threshold needed to sever this joint.
func (j *Joint) GetDetachThreshold() float64 {
	return j.detachThreshold
}

// LocalCenter returns the part's local geometric center relative to this joint's origin.
func (j *Joint) LocalCenter() (float64, float64, float64) {
	return j.localCenterX, j.localCenterY, j.localCenterZ
}

// PartRadius returns the bounding radius estimate for this joint's attached part.
func (j *Joint) PartRadius() float64 {
	return j.partRadius
}

// AccumulateStress adds impact stress to this joint and reports whether it reaches its detach threshold.
func (j *Joint) AccumulateStress(stress float64) bool {
	if !j.canDetach || j.detached {
		return false
	}
	j.damage += stress
	return j.damage >= j.detachThreshold
}

// IsCriticallyDamaged reports whether this detachable joint accumulated enough pre-mortem stress to sever on ragdoll transition.
func (j *Joint) IsCriticallyDamaged() bool {
	return j.canDetach && !j.detached && j.damage >= j.detachThreshold*0.45
}

// Detach marks the joint as detached and computes initial linear and angular launch velocities for rigid-body handoff.
func (j *Joint) Detach(dirX, dirY, dirZ, fMag, sideSign float64) (vx, vy, vz, wx, wy, wz float64) {
	j.detached = true
	launchSpeed := 125.0 + math.Min(fMag/500000.0, 2.5)*70.0
	vx = dirX*launchSpeed - dirY*(25.0*sideSign)
	vy = dirY*launchSpeed + dirX*(25.0*sideSign)
	vz = math.Max(dirZ*launchSpeed+105.0, 75.0)
	wx = -dirY*8.5 + sideSign*3.5
	wy = dirX*8.5 - 3.0
	wz = sideSign * 7.0
	return vx, vy, vz, wx, wy, wz
}

// SetLocalCenter sets the local center coordinates and radius estimate from local bounds.
func (j *Joint) SetLocalCenter(minX, minY, minZ, maxX, maxY, maxZ float64) {
	j.localCenterX = (minX + maxX) * 0.5
	j.localCenterY = (minY + maxY) * 0.5
	j.localCenterZ = (minZ + maxZ) * 0.5
	dx, dy, dz := maxX-minX, maxY-minY, maxZ-minZ
	r := math.Max(dx, math.Max(dy, dz)) * 0.5
	j.partRadius = math.Max(4.0, math.Min(48.0, r))
}

// SetMaxAngleRagdoll sets the maximum angular limit (in radians) for the joint in its ragdoll state.
func (j *Joint) SetMaxAngleRagdoll(f float64) {
	j.maxAngleRagdoll = f
}

// SetDamping sets the damping coefficient of the joint, controlling the rate at which motion is resisted.
func (j *Joint) SetDamping(f float64) {
	j.damping = f
}

// SetDetachThreshold sets the force threshold at which the joint can detach.
func (j *Joint) SetDetachThreshold(f float64) {
	j.detachThreshold = f
}

// SetCanDetach sets whether the joint can be detached or not based on the provided boolean value.
func (j *Joint) SetCanDetach(f bool) {
	j.canDetach = f
}
