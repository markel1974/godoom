package physics

import "math"

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

	angX           float64
	angY           float64
	angZ           float64
	velX           float64
	velY           float64
	velZ           float64
	ragdollTargetX float64
	ragdollTargetY float64
	ragdollTargetZ float64
	damage         float64
	hasDeflection  bool

	detached   bool
	detSettled bool
	detWorldX  float64
	detWorldY  float64
	detWorldZ  float64
	detVx      float64
	detVy      float64
	detVz      float64
	detWx      float64
	detWy      float64
	detWz      float64
	detFloorZ  float64

	quat      *Quaternion
	detQuat   *Quaternion
	axis      *TagAxis
	combinedA *TagAxis
	combinedB *TagAxis
}

// NewJoint2 creates a Joint with identity orientation quaternions and axes.
func NewJoint2(elem IElement, partRadius float64) *Joint {
	j := &Joint{
		elem:      elem,
		quat:      NewQuaternionIdentity(),
		detQuat:   NewQuaternionIdentity(),
		axis:      NewTagAxisIdentity(),
		combinedA: NewTagAxisIdentity(),
		combinedB: NewTagAxisIdentity(),
	}
	j.quat.SetIdentity()
	j.detQuat.SetIdentity()
	j.axis.Reset()
	j.combinedA.Reset()
	j.combinedB.Reset()
	j.partRadius = partRadius
	return j
}

// ActivateRagdoll sets clamped ragdoll targets from local motion inputs and applies angular velocity impulses.
// sign biases the Y-axis target to vary the collapse direction.
func (j *Joint) ActivateRagdoll(lx, ly, lz float64, sign float64) {
	j.ragdollTargetX = math.Max(-j.maxAngleRagdoll*0.85, math.Min(j.maxAngleRagdoll*0.85, lx*j.dofScaleX*0.95))
	j.ragdollTargetY = math.Max(-j.maxAngleRagdoll*0.85, math.Min(j.maxAngleRagdoll*0.85, (ly+0.28*sign)*j.dofScaleY*0.95))
	j.ragdollTargetZ = math.Max(-j.maxAngleRagdoll*0.70, math.Min(j.maxAngleRagdoll*0.70, lz*j.dofScaleZ*0.80))

	j.velX += j.ragdollTargetX * 5.0
	j.velY += j.ragdollTargetY * 5.0
	j.velZ += j.ragdollTargetZ * 4.0
}

// Detach marks the joint as detached and initializes its world position, floor height, launch velocity, and tumble.
func (j *Joint) Detach(halfD float64, cx, cy, cz float64, dirX, dirY, dirZ, fMag float64, orX, orY, orZ float64, sideSign float64) {
	if j.detached || !j.canDetach {
		return
	}
	j.detached = true
	j.detSettled = false
	j.detWorldX = cx + orX
	j.detWorldY = cy + orY
	j.detWorldZ = cz + orZ
	j.detFloorZ = (cz - halfD) + math.Min(j.partRadius*0.30, 5.0)
	if j.detWorldZ < j.detFloorZ {
		j.detWorldZ = j.detFloorZ
	}
	// TODO Ballistic launch velocity along impact ray + upward pop + slight lateral scatter
	launchSpeed := 125.0 + math.Min(fMag/500000.0, 2.5)*70.0
	j.detVx = dirX*launchSpeed - dirY*(25.0*sideSign)
	j.detVy = dirY*launchSpeed + dirX*(25.0*sideSign)
	j.detVz = math.Max(dirZ*launchSpeed+105.0, 75.0)
	// 3D tumbling angular velocity
	j.detWx = -dirY*8.5 + sideSign*3.5
	j.detWy = dirX*8.5 - 3.0
	j.detWz = sideSign * 7.0
}

// Settle advances detached joint motion by dt, applying gravity, damping, floor bounce, friction, and rest detection.
func (j *Joint) Settle(dt float64) {
	const limbGravity = 560.0
	j.detVz -= limbGravity * dt

	airDamp := math.Pow(0.65, dt)
	j.detVx *= airDamp
	j.detVy *= airDamp

	j.detWorldX += j.detVx * dt
	j.detWorldY += j.detVy * dt
	j.detWorldZ += j.detVz * dt

	if j.detWorldZ <= j.detFloorZ {
		j.detWorldZ = j.detFloorZ
		if j.detVz < -28.0 {
			j.detVz = -j.detVz * 0.34
			j.detWx += -j.detVy * 0.04
			j.detWy += j.detVx * 0.04
		} else {
			j.detVz = 0.0
		}
		groundFriction := math.Pow(0.015, dt)
		j.detVx *= groundFriction
		j.detVy *= groundFriction

		angFriction := math.Pow(0.025, dt)
		j.detWx *= angFriction
		j.detWy *= angFriction
		j.detWz *= angFriction

		if j.detVz == 0.0 && (j.detVx*j.detVx+j.detVy*j.detVy) < 1.0 &&
			(j.detWx*j.detWx+j.detWy*j.detWy+j.detWz*j.detWz) < 0.02 {
			j.detVx, j.detVy, j.detVz = 0, 0, 0
			j.detWx, j.detWy, j.detWz = 0, 0, 0
			j.detSettled = true
		}
	}
	j.detQuat.IntegrateAngular(j.detWx, j.detWy, j.detWz, dt)
}

// Connect advances the joint spring-damper state and cone limits, returning true while motion remains active.
func (j *Joint) Connect(isRagdoll bool, dt float64) bool {
	// Connected articulated joint (PD spring-damper + cone joint limits)
	ks := j.stiffness
	kd := j.damping
	maxAng := j.maxAngleAlive
	var targetX, targetY, targetZ float64
	active := false

	if isRagdoll {
		ks *= 0.12
		kd *= 0.55
		maxAng = j.maxAngleRagdoll
		targetX, targetY, targetZ = j.ragdollTargetX, j.ragdollTargetY, j.ragdollTargetZ
	} else if j.damage > 0 {
		j.damage = math.Max(0.0, j.damage-0.10*dt)
	}

	alphaX := -ks*(j.angX-targetX) - kd*j.velX
	alphaY := -ks*(j.angY-targetY) - kd*j.velY
	alphaZ := -ks*(j.angZ-targetZ) - kd*j.velZ

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

	diffX := j.angX - targetX
	diffY := j.angY - targetY
	diffZ := j.angZ - targetZ
	diffSq := diffX*diffX + diffY*diffY + diffZ*diffZ
	velMagSq := j.velX*j.velX + j.velY*j.velY + j.velZ*j.velZ

	if diffSq > 1e-6 || velMagSq > 1e-5 {
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

	if diffSq <= 1e-6 && velMagSq <= 1e-5 {
		j.angX, j.angY, j.angZ = targetX, targetY, targetZ
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
