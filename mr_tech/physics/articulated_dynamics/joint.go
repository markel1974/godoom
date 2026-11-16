package articulated_dynamics

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/physics/aabb"
)

type IElement interface {
	FreezeCurrentFrame()
}

// Joint defines a 3D vertex joint with angular motion, damping, damage, and detachment state.
type Joint struct {
	elem            IElement
	localCenterX    float64
	localCenterY    float64
	localCenterZ    float64
	partRadius      float64
	dofScaleX       float64
	dofScaleY       float64
	dofScaleZ       float64
	maxAngleAlive   float64
	maxAngleRagdoll float64
	stiffness       float64
	damping         float64
	canDetach       bool
	detachThreshold float64

	angX          float64
	angY          float64
	angZ          float64
	velX          float64
	velY          float64
	velZ          float64
	damage        float64
	hasDeflection bool
	detached      bool

	quat      *aabb.Quaternion
	axis      *TagAxis
	combinedA *TagAxis
	combinedB *TagAxis
}

// NewJoint2 creates a Joint with identity orientation quaternions and axes.
func NewJoint2(elem IElement, partRadius float64) *Joint {
	j := &Joint{
		elem:      elem,
		quat:      aabb.NewQuaternionIdentity(),
		axis:      NewTagAxisIdentity(),
		combinedA: NewTagAxisIdentity(),
		combinedB: NewTagAxisIdentity(),
	}
	j.quat.SetIdentity()
	j.axis.Reset()
	j.combinedA.Reset()
	j.combinedB.Reset()
	j.partRadius = partRadius
	return j
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

// Connect advances the live hit-flinch spring-damper state and cone limits, returning true while motion remains active.
func (j *Joint) Connect(dt float64) bool {
	if j.damage > 0 {
		j.damage = math.Max(0.0, j.damage-0.10*dt)
	}
	ks := j.stiffness
	kd := j.damping
	maxAng := j.maxAngleAlive
	active := false

	alphaX := -ks*j.angX - kd*j.velX
	alphaY := -ks*j.angY - kd*j.velY
	alphaZ := -ks*j.angZ - kd*j.velZ

	j.velX += alphaX * dt
	j.velY += alphaY * dt
	j.velZ += alphaZ * dt

	j.angX += j.velX * dt
	j.angY += j.velY * dt
	j.angZ += j.velZ * dt

	angMagSq := j.angX*j.angX + j.angY*j.angY + j.angZ*j.angZ
	if maxAng > 0.0 && angMagSq > maxAng*maxAng {
		angMag := math.Sqrt(angMagSq)
		nx, ny, nz := j.angX/angMag, j.angY/angMag, j.angZ/angMag
		j.angX, j.angY, j.angZ = nx*maxAng, ny*maxAng, nz*maxAng
		angMagSq = maxAng * maxAng
		vDotN := j.velX*nx + j.velY*ny + j.velZ*nz
		if vDotN > 0.0 {
			j.velX -= 1.30 * vDotN * nx
			j.velY -= 1.30 * vDotN * ny
			j.velZ -= 1.30 * vDotN * nz
		}
	}

	velMagSq := j.velX*j.velX + j.velY*j.velY + j.velZ*j.velZ
	if angMagSq > 1e-6 || velMagSq > 1e-5 {
		active = true
	}

	if angMagSq > 1e-6 {
		angMag := math.Sqrt(angMagSq)
		j.quat.SetAxisAngle(j.angX/angMag, j.angY/angMag, j.angZ/angMag, angMag)
		j.axis.SetFromQuaternion(j.quat)
		j.hasDeflection = true
	} else {
		j.axis.Reset()
		j.hasDeflection = false
	}

	if !active {
		j.angX, j.angY, j.angZ = 0, 0, 0
		j.velX, j.velY, j.velZ = 0, 0, 0
	}

	return active
}

// SetLocalCenter sets the local center coordinates of the joint based on the provided bounding box dimensions.
func (j *Joint) SetLocalCenter(minX, minY, minZ, maxX, maxY, maxZ float64) {
	j.localCenterX = (minX + maxX) * 0.5
	j.localCenterY = (minY + maxY) * 0.5
	j.localCenterZ = (minZ + maxZ) * 0.5
	dx, dy, dz := maxX-minX, maxY-minY, maxZ-minZ
	r := math.Max(dx, math.Max(dy, dz)) * 0.5
	j.partRadius = math.Max(4.0, math.Min(48.0, r))
}

// SetDofScale sets the scaling factors for the degrees of freedom (DOF) along the x, y, and z axes of the joint.
func (j *Joint) SetDofScale(x, y, z float64) {
	j.dofScaleX = x
	j.dofScaleY = y
	j.dofScaleZ = z
}

// SetMaxAngleAlive sets the maximum angular limit (in radians) for the joint in its active, non-detached state.
func (j *Joint) SetMaxAngleAlive(f float64) {
	j.maxAngleAlive = f
}

// SetMaxAngleRagdoll sets the maximum angular limit (in radians) for the joint in its ragdoll state.
func (j *Joint) SetMaxAngleRagdoll(f float64) {
	j.maxAngleRagdoll = f
}

// SetStiffness sets the stiffness value for the joint's spring dynamics.
func (j *Joint) SetStiffness(f float64) {
	j.stiffness = f
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
