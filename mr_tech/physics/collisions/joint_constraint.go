package collisions

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/physics/aabb"
	"github.com/markel1974/godoom/mr_tech/physics/rigid_body"
)

// JointType identifies the kinematic classification of a mechanical joint.
type JointType int

const (
	JointBallSocket  JointType = iota // 3 linear locked, 3 angular free (cone limit + damping)
	JointHinge                        // 3 linear locked, 2 angular locked, 1 angular free/limited/motor
	JointSlider                       // 2 linear locked, 1 linear free/limited/motor, 3 angular locked
	JointFixed                        // 3 linear locked, 3 angular locked (rigid unit)
	JointUniversal                    // 3 linear locked, 1 angular locked, 2 angular free
	JointCylindrical                  // 2 linear locked, 1 linear slider, 2 angular locked, 1 angular hinge
	JointGeneric6DoF                  // Arbitrary 6-DoF axis configuration
)

// AxisState defines the constraint behavior of an individual mechanical degree of freedom.
type AxisState int

const (
	AxisFree AxisState = iota
	AxisLocked
	AxisLimited
	AxisMotor
)

// JointConstraint represents a bilateral or limited mechanical constraint between two dynamic Solvers.
type JointConstraint struct {
	jointType JointType
	solverA   *Solver
	solverB   *Solver

	anchorAX, anchorAY, anchorAZ float64
	anchorBX, anchorBY, anchorBZ float64

	// Primary local constraint axes on body A and body B
	axisAX, axisAY, axisAZ    float64
	axisBX, axisBY, axisBZ    float64
	axis2AX, axis2AY, axis2AZ float64
	axis2BX, axis2BY, axis2BZ float64

	restQ    *aabb.Quaternion
	maxAngle float64 // Cone-limit half-angle for BallSocket
	damping  float64
	active   bool

	breakThreshold float64

	// Linear limits & motor (Slider / Cylindrical / Generic6DoF)
	enableLinearLimit     bool
	minLinearLimit        float64
	maxLinearLimit        float64
	enableLinearMotor     bool
	targetLinearVelocity  float64
	maxLinearMotorImpulse float64

	// Angular limits & motor (Hinge / Cylindrical / Generic6DoF)
	enableAngularLimit     bool
	minAngularLimit        float64
	maxAngularLimit        float64
	enableAngularMotor     bool
	targetAngularVelocity  float64
	maxAngularMotorImpulse float64

	// Generic 6DoF per-axis configuration
	linearAxisStates  [3]AxisState
	angularAxisStates [3]AxisState
	minLinear6D       [3]float64
	maxLinear6D       [3]float64
	minAngular6D      [3]float64
	maxAngular6D      [3]float64

	// Persisted impulses for Warm Starting across PGS sweeps
	accJx, accJy, accJz       float64
	accAngX, accAngY, accAngZ float64
	accMotorLin               float64
	accLimitLinMin            float64
	accLimitLinMax            float64
	accMotorAng               float64
	accLimitAngMin            float64
	accLimitAngMax            float64

	// Scratch structures to guarantee zero heap allocations in solve loops
	qErrScratch    *aabb.Quaternion
	qTargetScratch *aabb.Quaternion
}

func normalizeVec3(x, y, z float64) (float64, float64, float64) {
	lenSq := x*x + y*y + z*z
	if lenSq > 1e-12 {
		inv := 1.0 / math.Sqrt(lenSq)
		return x * inv, y * inv, z * inv
	}
	return 0.0, 0.0, 1.0
}

// NewJointConstraint creates an active bilateral BallSocket constraint joining solverA and solverB at their
// respective local anchors, preserving restQ up to maxAngle radians with viscous angular damping.
func NewJointConstraint(
	solverA, solverB *Solver,
	ax, ay, az, bx, by, bz float64,
	restQ *aabb.Quaternion,
	maxAngle, damping float64,
) *JointConstraint {
	return NewBallSocketJoint(solverA, solverB, ax, ay, az, bx, by, bz, restQ, maxAngle, damping)
}

// NewBallSocketJoint creates an active spherical joint constraint with optional cone limit and damping.
func NewBallSocketJoint(
	solverA, solverB *Solver,
	ax, ay, az, bx, by, bz float64,
	restQ *aabb.Quaternion,
	maxAngle, damping float64,
) *JointConstraint {
	rq := aabb.NewQuaternionIdentity()
	if restQ != nil {
		rq.CopyFrom(restQ)
		rq.Normalize()
	}
	return &JointConstraint{
		jointType:      JointBallSocket,
		solverA:        solverA,
		solverB:        solverB,
		anchorAX:       ax,
		anchorAY:       ay,
		anchorAZ:       az,
		anchorBX:       bx,
		anchorBY:       by,
		anchorBZ:       bz,
		axisAX:         0,
		axisAY:         0,
		axisAZ:         1,
		axisBX:         0,
		axisBY:         0,
		axisBZ:         1,
		restQ:          rq,
		maxAngle:       maxAngle,
		damping:        damping,
		active:         true,
		qErrScratch:    aabb.NewQuaternionIdentity(),
		qTargetScratch: aabb.NewQuaternionIdentity(),
	}
}

// NewHingeJoint creates a revolute joint constraint allowing rotation only around axis (specified in body A's local frame).
func NewHingeJoint(
	solverA, solverB *Solver,
	ax, ay, az, bx, by, bz float64,
	axisX, axisY, axisZ float64,
	minAngle, maxAngle, damping float64,
) *JointConstraint {
	nx, ny, nz := normalizeVec3(axisX, axisY, axisZ)
	jc := &JointConstraint{
		jointType:          JointHinge,
		solverA:            solverA,
		solverB:            solverB,
		anchorAX:           ax,
		anchorAY:           ay,
		anchorAZ:           az,
		anchorBX:           bx,
		anchorBY:           by,
		anchorBZ:           bz,
		axisAX:             nx,
		axisAY:             ny,
		axisAZ:             nz,
		axisBX:             nx,
		axisBY:             ny,
		axisBZ:             nz,
		restQ:              aabb.NewQuaternionIdentity(),
		damping:            damping,
		active:             true,
		minAngularLimit:    minAngle,
		maxAngularLimit:    maxAngle,
		enableAngularLimit: minAngle < maxAngle,
		qErrScratch:        aabb.NewQuaternionIdentity(),
		qTargetScratch:     aabb.NewQuaternionIdentity(),
	}
	return jc
}

// NewSliderJoint creates a prismatic joint constraint allowing translation only along axis (in body A's local frame).
func NewSliderJoint(
	solverA, solverB *Solver,
	ax, ay, az, bx, by, bz float64,
	axisX, axisY, axisZ float64,
	minLimit, maxLimit, damping float64,
) *JointConstraint {
	nx, ny, nz := normalizeVec3(axisX, axisY, axisZ)
	return &JointConstraint{
		jointType:         JointSlider,
		solverA:           solverA,
		solverB:           solverB,
		anchorAX:          ax,
		anchorAY:          ay,
		anchorAZ:          az,
		anchorBX:          bx,
		anchorBY:          by,
		anchorBZ:          bz,
		axisAX:            nx,
		axisAY:            ny,
		axisAZ:            nz,
		axisBX:            nx,
		axisBY:            ny,
		axisBZ:            nz,
		restQ:             aabb.NewQuaternionIdentity(),
		damping:           damping,
		active:            true,
		minLinearLimit:    minLimit,
		maxLinearLimit:    maxLimit,
		enableLinearLimit: minLimit < maxLimit,
		qErrScratch:       aabb.NewQuaternionIdentity(),
		qTargetScratch:    aabb.NewQuaternionIdentity(),
	}
}

// NewFixedJoint creates a weld joint completely locking all 6 degrees of freedom between body A and body B.
func NewFixedJoint(
	solverA, solverB *Solver,
	ax, ay, az, bx, by, bz float64,
	restQ *aabb.Quaternion,
) *JointConstraint {
	rq := aabb.NewQuaternionIdentity()
	if restQ != nil {
		rq.CopyFrom(restQ)
		rq.Normalize()
	}
	return &JointConstraint{
		jointType:      JointFixed,
		solverA:        solverA,
		solverB:        solverB,
		anchorAX:       ax,
		anchorAY:       ay,
		anchorAZ:       az,
		anchorBX:       bx,
		anchorBY:       by,
		anchorBZ:       bz,
		restQ:          rq,
		active:         true,
		qErrScratch:    aabb.NewQuaternionIdentity(),
		qTargetScratch: aabb.NewQuaternionIdentity(),
	}
}

// NewUniversalJoint creates a Cardan/Universal joint allowing rotation around two orthogonal axes.
func NewUniversalJoint(
	solverA, solverB *Solver,
	ax, ay, az, bx, by, bz float64,
	axisA_X, axisA_Y, axisA_Z float64,
	axisB_X, axisB_Y, axisB_Z float64,
	damping float64,
) *JointConstraint {
	aX, aY, aZ := normalizeVec3(axisA_X, axisA_Y, axisA_Z)
	bX, bY, bZ := normalizeVec3(axisB_X, axisB_Y, axisB_Z)
	return &JointConstraint{
		jointType:      JointUniversal,
		solverA:        solverA,
		solverB:        solverB,
		anchorAX:       ax,
		anchorAY:       ay,
		anchorAZ:       az,
		anchorBX:       bx,
		anchorBY:       by,
		anchorBZ:       bz,
		axisAX:         aX,
		axisAY:         aY,
		axisAZ:         aZ,
		axisBX:         bX,
		axisBY:         bY,
		axisBZ:         bZ,
		restQ:          aabb.NewQuaternionIdentity(),
		damping:        damping,
		active:         true,
		qErrScratch:    aabb.NewQuaternionIdentity(),
		qTargetScratch: aabb.NewQuaternionIdentity(),
	}
}

// NewCylindricalJoint creates a cylindrical joint constraint combining collinear prismatic slide and revolute hinge.
func NewCylindricalJoint(
	solverA, solverB *Solver,
	ax, ay, az, bx, by, bz float64,
	axisX, axisY, axisZ float64,
	minLin, maxLin, minAng, maxAng, damping float64,
) *JointConstraint {
	nx, ny, nz := normalizeVec3(axisX, axisY, axisZ)
	return &JointConstraint{
		jointType:          JointCylindrical,
		solverA:            solverA,
		solverB:            solverB,
		anchorAX:           ax,
		anchorAY:           ay,
		anchorAZ:           az,
		anchorBX:           bx,
		anchorBY:           by,
		anchorBZ:           bz,
		axisAX:             nx,
		axisAY:             ny,
		axisAZ:             nz,
		axisBX:             nx,
		axisBY:             ny,
		axisBZ:             nz,
		restQ:              aabb.NewQuaternionIdentity(),
		damping:            damping,
		active:             true,
		minLinearLimit:     minLin,
		maxLinearLimit:     maxLin,
		enableLinearLimit:  minLin < maxLin,
		minAngularLimit:    minAng,
		maxAngularLimit:    maxAng,
		enableAngularLimit: minAng < maxAng,
		qErrScratch:        aabb.NewQuaternionIdentity(),
		qTargetScratch:     aabb.NewQuaternionIdentity(),
	}
}

// NewGeneric6DoFJoint creates a fully configurable 6-DoF constraint initialized with default locked linear axes.
func NewGeneric6DoFJoint(
	solverA, solverB *Solver,
	ax, ay, az, bx, by, bz float64,
	restQ *aabb.Quaternion,
) *JointConstraint {
	rq := aabb.NewQuaternionIdentity()
	if restQ != nil {
		rq.CopyFrom(restQ)
		rq.Normalize()
	}
	jc := &JointConstraint{
		jointType:      JointGeneric6DoF,
		solverA:        solverA,
		solverB:        solverB,
		anchorAX:       ax,
		anchorAY:       ay,
		anchorAZ:       az,
		anchorBX:       bx,
		anchorBY:       by,
		anchorBZ:       bz,
		axisAX:         1,
		axisAY:         0,
		axisAZ:         0,
		axisBX:         1,
		axisBY:         0,
		axisBZ:         0,
		restQ:          rq,
		active:         true,
		qErrScratch:    aabb.NewQuaternionIdentity(),
		qTargetScratch: aabb.NewQuaternionIdentity(),
	}
	jc.linearAxisStates[0] = AxisLocked
	jc.linearAxisStates[1] = AxisLocked
	jc.linearAxisStates[2] = AxisLocked
	return jc
}

// SetLinearLimits configures the min and max displacement limits for linear sliding motion.
func (j *JointConstraint) SetLinearLimits(min, max float64) *JointConstraint {
	j.minLinearLimit = min
	j.maxLinearLimit = max
	j.enableLinearLimit = min <= max
	return j
}

// SetAngularLimits configures the min and max angular rotation limits around the joint axis.
func (j *JointConstraint) SetAngularLimits(min, max float64) *JointConstraint {
	j.minAngularLimit = min
	j.maxAngularLimit = max
	j.enableAngularLimit = min <= max
	return j
}

// SetLinearMotor configures the target linear velocity and maximum motor impulse capacity.
func (j *JointConstraint) SetLinearMotor(targetVel, maxImpulse float64) *JointConstraint {
	j.targetLinearVelocity = targetVel
	j.maxLinearMotorImpulse = math.Abs(maxImpulse)
	j.enableLinearMotor = j.maxLinearMotorImpulse > 0.0
	return j
}

// SetAngularMotor configures the target angular velocity and maximum motor torque impulse capacity.
func (j *JointConstraint) SetAngularMotor(targetVel, maxImpulse float64) *JointConstraint {
	j.targetAngularVelocity = targetVel
	j.maxAngularMotorImpulse = math.Abs(maxImpulse)
	j.enableAngularMotor = j.maxAngularMotorImpulse > 0.0
	return j
}

// EnableLinearMotor turns the linear drive motor on or off.
func (j *JointConstraint) EnableLinearMotor(enable bool) *JointConstraint {
	j.enableLinearMotor = enable
	return j
}

// EnableAngularMotor turns the rotational drive motor on or off.
func (j *JointConstraint) EnableAngularMotor(enable bool) *JointConstraint {
	j.enableAngularMotor = enable
	return j
}

// SetDamping configures joint friction/damping coefficient.
func (j *JointConstraint) SetDamping(damping float64) *JointConstraint {
	j.damping = damping
	return j
}

// SetBreakThreshold assigns the accumulated stress threshold to snap this joint.
func (j *JointConstraint) SetBreakThreshold(threshold float64) *JointConstraint {
	j.breakThreshold = threshold
	return j
}

// GetBreakThreshold returns the stress threshold of this joint.
func (j *JointConstraint) GetBreakThreshold() float64 {
	return j.breakThreshold
}

// GetType returns the kinematic classification of this constraint.
func (j *JointConstraint) GetType() JointType {
	return j.jointType
}

// GetSolvers returns the pair of solvers linked by this joint.
func (j *JointConstraint) GetSolvers() (*Solver, *Solver) {
	return j.solverA, j.solverB
}

// IsActive reports whether the constraint is unbroken and both linked solvers are active.
func (j *JointConstraint) IsActive() bool {
	return j.active && j.solverA.IsActive() && j.solverB.IsActive()
}

// Break permanently disables this joint constraint.
func (j *JointConstraint) Break() {
	j.active = false
	j.accJx, j.accJy, j.accJz = 0.0, 0.0, 0.0
	j.accAngX, j.accAngY, j.accAngZ = 0.0, 0.0, 0.0
	j.accMotorLin, j.accMotorAng = 0.0, 0.0
	j.accLimitLinMin, j.accLimitLinMax = 0.0, 0.0
	j.accLimitAngMin, j.accLimitAngMax = 0.0, 0.0
}

// WarmStart re-applies persisted linear and angular impulses prior to iterative PGS sweeps.
func (j *JointConstraint) WarmStart() {
	if !j.IsActive() || j.solverA.GetSolverIndex() < 0 || j.solverB.GetSolverIndex() < 0 {
		return
	}

	entA := j.solverA.GetEntity()
	entB := j.solverB.GetEntity()

	switch j.jointType {
	case JointBallSocket:
		entA.WarmStartJoint(
			entB,
			j.anchorAX, j.anchorAY, j.anchorAZ,
			j.anchorBX, j.anchorBY, j.anchorBZ,
			&j.accJx, &j.accJy, &j.accJz,
		)
	case JointFixed:
		_, _, _, rAx, rAy, rAz := entA.AnchorWorld(j.anchorAX, j.anchorAY, j.anchorAZ)
		_, _, _, rBx, rBy, rBz := entB.AnchorWorld(j.anchorBX, j.anchorBY, j.anchorBZ)
		entA.ApplyLinearImpulse(entB, rAx, rAy, rAz, rBx, rBy, rBz, 1, 0, 0, j.accJx)
		entA.ApplyLinearImpulse(entB, rAx, rAy, rAz, rBx, rBy, rBz, 0, 1, 0, j.accJy)
		entA.ApplyLinearImpulse(entB, rAx, rAy, rAz, rBx, rBy, rBz, 0, 0, 1, j.accJz)
		entA.ApplyAngularImpulse(entB, 1, 0, 0, j.accAngX)
		entA.ApplyAngularImpulse(entB, 0, 1, 0, j.accAngY)
		entA.ApplyAngularImpulse(entB, 0, 0, 1, j.accAngZ)

	case JointHinge:
		_, _, _, rAx, rAy, rAz := entA.AnchorWorld(j.anchorAX, j.anchorAY, j.anchorAZ)
		_, _, _, rBx, rBy, rBz := entB.AnchorWorld(j.anchorBX, j.anchorBY, j.anchorBZ)
		entA.ApplyLinearImpulse(entB, rAx, rAy, rAz, rBx, rBy, rBz, 1, 0, 0, j.accJx)
		entA.ApplyLinearImpulse(entB, rAx, rAy, rAz, rBx, rBy, rBz, 0, 1, 0, j.accJy)
		entA.ApplyLinearImpulse(entB, rAx, rAy, rAz, rBx, rBy, rBz, 0, 0, 1, j.accJz)

		qA := entA.GetQRotation()
		hx, hy, hz := qA.Rotate(j.axisAX, j.axisAY, j.axisAZ)
		ux, uy, uz, vx, vy, vz := rigid_body.ComputeOrthonormalBasis(hx, hy, hz)
		entA.ApplyAngularImpulse(entB, ux, uy, uz, j.accAngX)
		entA.ApplyAngularImpulse(entB, vx, vy, vz, j.accAngY)
		if j.enableAngularMotor {
			entA.ApplyAngularImpulse(entB, hx, hy, hz, j.accMotorAng)
		}
		if j.enableAngularLimit {
			entA.ApplyAngularImpulse(entB, hx, hy, hz, j.accLimitAngMin+j.accLimitAngMax)
		}

	case JointSlider:
		_, _, _, rAx, rAy, rAz := entA.AnchorWorld(j.anchorAX, j.anchorAY, j.anchorAZ)
		_, _, _, rBx, rBy, rBz := entB.AnchorWorld(j.anchorBX, j.anchorBY, j.anchorBZ)
		qA := entA.GetQRotation()
		sx, sy, sz := qA.Rotate(j.axisAX, j.axisAY, j.axisAZ)
		ux, uy, uz, vx, vy, vz := rigid_body.ComputeOrthonormalBasis(sx, sy, sz)
		entA.ApplyLinearImpulse(entB, rAx, rAy, rAz, rBx, rBy, rBz, ux, uy, uz, j.accJx)
		entA.ApplyLinearImpulse(entB, rAx, rAy, rAz, rBx, rBy, rBz, vx, vy, vz, j.accJy)
		if j.enableLinearMotor {
			entA.ApplyLinearImpulse(entB, rAx, rAy, rAz, rBx, rBy, rBz, sx, sy, sz, j.accMotorLin)
		}
		if j.enableLinearLimit {
			entA.ApplyLinearImpulse(entB, rAx, rAy, rAz, rBx, rBy, rBz, sx, sy, sz, j.accLimitLinMin+j.accLimitLinMax)
		}
		entA.ApplyAngularImpulse(entB, 1, 0, 0, j.accAngX)
		entA.ApplyAngularImpulse(entB, 0, 1, 0, j.accAngY)
		entA.ApplyAngularImpulse(entB, 0, 0, 1, j.accAngZ)

	case JointUniversal, JointCylindrical, JointGeneric6DoF:
		_, _, _, rAx, rAy, rAz := entA.AnchorWorld(j.anchorAX, j.anchorAY, j.anchorAZ)
		_, _, _, rBx, rBy, rBz := entB.AnchorWorld(j.anchorBX, j.anchorBY, j.anchorBZ)
		entA.ApplyLinearImpulse(entB, rAx, rAy, rAz, rBx, rBy, rBz, 1, 0, 0, j.accJx)
		entA.ApplyLinearImpulse(entB, rAx, rAy, rAz, rBx, rBy, rBz, 0, 1, 0, j.accJy)
		entA.ApplyLinearImpulse(entB, rAx, rAy, rAz, rBx, rBy, rBz, 0, 0, 1, j.accJz)
		entA.ApplyAngularImpulse(entB, 1, 0, 0, j.accAngX)
		entA.ApplyAngularImpulse(entB, 0, 1, 0, j.accAngY)
		entA.ApplyAngularImpulse(entB, 0, 0, 1, j.accAngZ)
	}
}

// ResolveVelocity executes one bilateral/unilateral PGS velocity sweep for this joint.
func (j *JointConstraint) ResolveVelocity() {
	if !j.IsActive() || j.solverA.GetSolverIndex() < 0 || j.solverB.GetSolverIndex() < 0 {
		return
	}

	entA := j.solverA.GetEntity()
	entB := j.solverB.GetEntity()
	const inf = math.MaxFloat64

	switch j.jointType {
	case JointBallSocket:
		entA.ResolveJointVelocity(
			entB,
			j.anchorAX, j.anchorAY, j.anchorAZ,
			j.anchorBX, j.anchorBY, j.anchorBZ,
			j.restQ, j.maxAngle, j.damping,
			&j.accJx, &j.accJy, &j.accJz,
			j.qErrScratch, j.qTargetScratch,
		)

	case JointFixed:
		_, _, _, rAx, rAy, rAz := entA.AnchorWorld(j.anchorAX, j.anchorAY, j.anchorAZ)
		_, _, _, rBx, rBy, rBz := entB.AnchorWorld(j.anchorBX, j.anchorBY, j.anchorBZ)
		entA.SolveLinearAxis(entB, rAx, rAy, rAz, rBx, rBy, rBz, 1, 0, 0, &j.accJx, 0, -inf, inf)
		entA.SolveLinearAxis(entB, rAx, rAy, rAz, rBx, rBy, rBz, 0, 1, 0, &j.accJy, 0, -inf, inf)
		entA.SolveLinearAxis(entB, rAx, rAy, rAz, rBx, rBy, rBz, 0, 0, 1, &j.accJz, 0, -inf, inf)
		entA.SolveAngularAxis(entB, 1, 0, 0, &j.accAngX, 0, -inf, inf)
		entA.SolveAngularAxis(entB, 0, 1, 0, &j.accAngY, 0, -inf, inf)
		entA.SolveAngularAxis(entB, 0, 0, 1, &j.accAngZ, 0, -inf, inf)

	case JointHinge:
		_, _, _, rAx, rAy, rAz := entA.AnchorWorld(j.anchorAX, j.anchorAY, j.anchorAZ)
		_, _, _, rBx, rBy, rBz := entB.AnchorWorld(j.anchorBX, j.anchorBY, j.anchorBZ)
		// Lock 3 linear anchor DoFs
		entA.SolveLinearAxis(entB, rAx, rAy, rAz, rBx, rBy, rBz, 1, 0, 0, &j.accJx, 0, -inf, inf)
		entA.SolveLinearAxis(entB, rAx, rAy, rAz, rBx, rBy, rBz, 0, 1, 0, &j.accJy, 0, -inf, inf)
		entA.SolveLinearAxis(entB, rAx, rAy, rAz, rBx, rBy, rBz, 0, 0, 1, &j.accJz, 0, -inf, inf)

		// Lock 2 orthogonal angular DoFs (rotation only permitted around hinge axis)
		qA := entA.GetQRotation()
		hx, hy, hz := qA.Rotate(j.axisAX, j.axisAY, j.axisAZ)
		ux, uy, uz, vx, vy, vz := rigid_body.ComputeOrthonormalBasis(hx, hy, hz)
		entA.SolveAngularAxis(entB, ux, uy, uz, &j.accAngX, 0, -inf, inf)
		entA.SolveAngularAxis(entB, vx, vy, vz, &j.accAngY, 0, -inf, inf)

		// Hinge axis motor
		if j.enableAngularMotor {
			entA.SolveAngularAxis(entB, hx, hy, hz, &j.accMotorAng, j.targetAngularVelocity, -j.maxAngularMotorImpulse, j.maxAngularMotorImpulse)
		}

		// Hinge axis angular limits
		if j.enableAngularLimit {
			qB := entB.GetQRotation()
			uBx, uBy, uBz := qB.Rotate(ux, uy, uz) // rotated reference vector on B
			cosAng := ux*uBx + uy*uBy + uz*uBz
			sinAng := vx*uBx + vy*uBy + vz*uBz
			theta := math.Atan2(sinAng, cosAng)
			if theta < j.minAngularLimit {
				entA.SolveAngularAxis(entB, hx, hy, hz, &j.accLimitAngMin, 0, 0, inf)
			} else {
				j.accLimitAngMin = 0.0
			}
			if theta > j.maxAngularLimit {
				entA.SolveAngularAxis(entB, hx, hy, hz, &j.accLimitAngMax, 0, -inf, 0)
			} else {
				j.accLimitAngMax = 0.0
			}
		}

		// Damping along hinge axis
		if j.damping > 0.0 {
			avx, avy, avz := entA.GetAngularVelocity()
			bvx, bvy, bvz := entB.GetAngularVelocity()
			relW := (bvx-avx)*hx + (bvy-avy)*hy + (bvz-avz)*hz
			dampImp := -relW * math.Min(j.damping*0.05, 0.5)
			entA.ApplyAngularImpulse(entB, hx, hy, hz, dampImp)
		}

	case JointSlider:
		pAx, pAy, pAz, rAx, rAy, rAz := entA.AnchorWorld(j.anchorAX, j.anchorAY, j.anchorAZ)
		pBx, pBy, pBz, rBx, rBy, rBz := entB.AnchorWorld(j.anchorBX, j.anchorBY, j.anchorBZ)

		// Lock all 3 angular DoFs
		entA.SolveAngularAxis(entB, 1, 0, 0, &j.accAngX, 0, -inf, inf)
		entA.SolveAngularAxis(entB, 0, 1, 0, &j.accAngY, 0, -inf, inf)
		entA.SolveAngularAxis(entB, 0, 0, 1, &j.accAngZ, 0, -inf, inf)

		// Lock 2 lateral linear DoFs perpendicular to slide axis
		qA := entA.GetQRotation()
		sx, sy, sz := qA.Rotate(j.axisAX, j.axisAY, j.axisAZ)
		ux, uy, uz, vx, vy, vz := rigid_body.ComputeOrthonormalBasis(sx, sy, sz)
		entA.SolveLinearAxis(entB, rAx, rAy, rAz, rBx, rBy, rBz, ux, uy, uz, &j.accJx, 0, -inf, inf)
		entA.SolveLinearAxis(entB, rAx, rAy, rAz, rBx, rBy, rBz, vx, vy, vz, &j.accJy, 0, -inf, inf)

		// Slide axis motor
		if j.enableLinearMotor {
			entA.SolveLinearAxis(entB, rAx, rAy, rAz, rBx, rBy, rBz, sx, sy, sz, &j.accMotorLin, j.targetLinearVelocity, -j.maxLinearMotorImpulse, j.maxLinearMotorImpulse)
		}

		// Slide axis linear limits
		if j.enableLinearLimit {
			d := (pBx-pAx)*sx + (pBy-pAy)*sy + (pBz-pAz)*sz
			if d < j.minLinearLimit {
				entA.SolveLinearAxis(entB, rAx, rAy, rAz, rBx, rBy, rBz, sx, sy, sz, &j.accLimitLinMin, 0, 0, inf)
			} else {
				j.accLimitLinMin = 0.0
			}
			if d > j.maxLinearLimit {
				entA.SolveLinearAxis(entB, rAx, rAy, rAz, rBx, rBy, rBz, sx, sy, sz, &j.accLimitLinMax, 0, -inf, 0)
			} else {
				j.accLimitLinMax = 0.0
			}
		}

		// Damping along slide axis
		if j.damping > 0.0 {
			avx, avy, avz := entA.GetVelocity()
			bvx, bvy, bvz := entB.GetVelocity()
			relV := (bvx-avx)*sx + (bvy-avy)*sy + (bvz-avz)*sz
			dampImp := -relV * math.Min(j.damping*0.05, 0.5)
			entA.ApplyLinearImpulse(entB, rAx, rAy, rAz, rBx, rBy, rBz, sx, sy, sz, dampImp)
		}

	case JointUniversal:
		_, _, _, rAx, rAy, rAz := entA.AnchorWorld(j.anchorAX, j.anchorAY, j.anchorAZ)
		_, _, _, rBx, rBy, rBz := entB.AnchorWorld(j.anchorBX, j.anchorBY, j.anchorBZ)
		entA.SolveLinearAxis(entB, rAx, rAy, rAz, rBx, rBy, rBz, 1, 0, 0, &j.accJx, 0, -inf, inf)
		entA.SolveLinearAxis(entB, rAx, rAy, rAz, rBx, rBy, rBz, 0, 1, 0, &j.accJy, 0, -inf, inf)
		entA.SolveLinearAxis(entB, rAx, rAy, rAz, rBx, rBy, rBz, 0, 0, 1, &j.accJz, 0, -inf, inf)

		// Lock the mutual perpendicular axis n = axisA x axisB
		qA := entA.GetQRotation()
		qB := entB.GetQRotation()
		a1x, a1y, a1z := qA.Rotate(j.axisAX, j.axisAY, j.axisAZ)
		a2x, a2y, a2z := qB.Rotate(j.axisBX, j.axisBY, j.axisBZ)
		nx := a1y*a2z - a1z*a2y
		ny := a1z*a2x - a1x*a2z
		nz := a1x*a2y - a1y*a2x
		nLen := math.Sqrt(nx*nx + ny*ny + nz*nz)
		if nLen > 1e-12 {
			nx /= nLen
			ny /= nLen
			nz /= nLen
			entA.SolveAngularAxis(entB, nx, ny, nz, &j.accAngZ, 0, -inf, inf)
		}

	case JointCylindrical:
		pAx, pAy, pAz, rAx, rAy, rAz := entA.AnchorWorld(j.anchorAX, j.anchorAY, j.anchorAZ)
		pBx, pBy, pBz, rBx, rBy, rBz := entB.AnchorWorld(j.anchorBX, j.anchorBY, j.anchorBZ)
		qA := entA.GetQRotation()
		sx, sy, sz := qA.Rotate(j.axisAX, j.axisAY, j.axisAZ)
		ux, uy, uz, vx, vy, vz := rigid_body.ComputeOrthonormalBasis(sx, sy, sz)

		// Lock 2 lateral linear DoFs and 2 orthogonal angular DoFs
		entA.SolveLinearAxis(entB, rAx, rAy, rAz, rBx, rBy, rBz, ux, uy, uz, &j.accJx, 0, -inf, inf)
		entA.SolveLinearAxis(entB, rAx, rAy, rAz, rBx, rBy, rBz, vx, vy, vz, &j.accJy, 0, -inf, inf)
		entA.SolveAngularAxis(entB, ux, uy, uz, &j.accAngX, 0, -inf, inf)
		entA.SolveAngularAxis(entB, vx, vy, vz, &j.accAngY, 0, -inf, inf)

		if j.enableLinearMotor {
			entA.SolveLinearAxis(entB, rAx, rAy, rAz, rBx, rBy, rBz, sx, sy, sz, &j.accMotorLin, j.targetLinearVelocity, -j.maxLinearMotorImpulse, j.maxLinearMotorImpulse)
		}
		if j.enableAngularMotor {
			entA.SolveAngularAxis(entB, sx, sy, sz, &j.accMotorAng, j.targetAngularVelocity, -j.maxAngularMotorImpulse, j.maxAngularMotorImpulse)
		}
		if j.enableLinearLimit {
			d := (pBx-pAx)*sx + (pBy-pAy)*sy + (pBz-pAz)*sz
			if d < j.minLinearLimit {
				entA.SolveLinearAxis(entB, rAx, rAy, rAz, rBx, rBy, rBz, sx, sy, sz, &j.accLimitLinMin, 0, 0, inf)
			}
			if d > j.maxLinearLimit {
				entA.SolveLinearAxis(entB, rAx, rAy, rAz, rBx, rBy, rBz, sx, sy, sz, &j.accLimitLinMax, 0, -inf, 0)
			}
		}

	case JointGeneric6DoF:
		pAx, pAy, pAz, rAx, rAy, rAz := entA.AnchorWorld(j.anchorAX, j.anchorAY, j.anchorAZ)
		pBx, pBy, pBz, rBx, rBy, rBz := entB.AnchorWorld(j.anchorBX, j.anchorBY, j.anchorBZ)
		qA := entA.GetQRotation()
		a0x, a0y, a0z, a1x, a1y, a1z, a2x, a2y, a2z := qA.ToAxes()
		axes := [3][3]float64{{a0x, a0y, a0z}, {a1x, a1y, a1z}, {a2x, a2y, a2z}}
		accLin := []*float64{&j.accJx, &j.accJy, &j.accJz}
		accAng := []*float64{&j.accAngX, &j.accAngY, &j.accAngZ}

		for idx := 0; idx < 3; idx++ {
			ax, ay, az := axes[idx][0], axes[idx][1], axes[idx][2]
			switch j.linearAxisStates[idx] {
			case AxisLocked:
				entA.SolveLinearAxis(entB, rAx, rAy, rAz, rBx, rBy, rBz, ax, ay, az, accLin[idx], 0, -inf, inf)
			case AxisLimited:
				d := (pBx-pAx)*ax + (pBy-pAy)*ay + (pBz-pAz)*az
				if d < j.minLinear6D[idx] {
					entA.SolveLinearAxis(entB, rAx, rAy, rAz, rBx, rBy, rBz, ax, ay, az, accLin[idx], 0, 0, inf)
				} else if d > j.maxLinear6D[idx] {
					entA.SolveLinearAxis(entB, rAx, rAy, rAz, rBx, rBy, rBz, ax, ay, az, accLin[idx], 0, -inf, 0)
				}
			}

			switch j.angularAxisStates[idx] {
			case AxisLocked:
				entA.SolveAngularAxis(entB, ax, ay, az, accAng[idx], 0, -inf, inf)
			}
		}
	}
}

// ResolvePosition executes one positional projection relaxation pass for this joint.
func (j *JointConstraint) ResolvePosition() {
	if !j.IsActive() || j.solverA.GetSolverIndex() < 0 || j.solverB.GetSolverIndex() < 0 {
		return
	}

	entA := j.solverA.GetEntity()
	entB := j.solverB.GetEntity()

	switch j.jointType {
	case JointBallSocket:
		entA.ResolveJointPosition(
			entB,
			j.anchorAX, j.anchorAY, j.anchorAZ,
			j.anchorBX, j.anchorBY, j.anchorBZ,
			j.restQ, j.maxAngle,
			j.qErrScratch, j.qTargetScratch,
		)

	case JointFixed:
		pAx, pAy, pAz, rAx, rAy, rAz := entA.AnchorWorld(j.anchorAX, j.anchorAY, j.anchorAZ)
		pBx, pBy, pBz, rBx, rBy, rBz := entB.AnchorWorld(j.anchorBX, j.anchorBY, j.anchorBZ)
		cx, cy, cz := pBx-pAx, pBy-pAy, pBz-pAz
		dist := math.Sqrt(cx*cx + cy*cy + cz*cz)
		if dist > 1e-4 {
			entA.ProjectLinearDistance(entB, rAx, rAy, rAz, rBx, rBy, rBz, cx/dist, cy/dist, cz/dist, dist, 0.01, 0.70)
		}

		// Relative orientation error relaxation
		qA := entA.GetQRotation()
		qB := entB.GetQRotation()
		qA.Multiplies(j.restQ, j.qTargetScratch)
		j.qErrScratch.Conjugate(j.qTargetScratch)
		j.qErrScratch.MultipliesSelf(qB)
		ex, ey, ez, ew := j.qErrScratch.Get()
		if ew < 0.0 {
			ex, ey, ez, ew = -ex, -ey, -ez, -ew
		}
		vSq := ex*ex + ey*ey + ez*ez
		if vSq > 1e-12 {
			vLen := math.Sqrt(vSq)
			angle := 2.0 * math.Atan2(vLen, ew)
			axW, ayW, azW := j.qTargetScratch.Rotate(ex/vLen, ey/vLen, ez/vLen)
			entA.ProjectAngularAngle(entB, axW, ayW, azW, angle, 0.002, 0.30)
		}

	case JointHinge:
		pAx, pAy, pAz, rAx, rAy, rAz := entA.AnchorWorld(j.anchorAX, j.anchorAY, j.anchorAZ)
		pBx, pBy, pBz, rBx, rBy, rBz := entB.AnchorWorld(j.anchorBX, j.anchorBY, j.anchorBZ)
		cx, cy, cz := pBx-pAx, pBy-pAy, pBz-pAz
		dist := math.Sqrt(cx*cx + cy*cy + cz*cz)
		if dist > 1e-4 {
			entA.ProjectLinearDistance(entB, rAx, rAy, rAz, rBx, rBy, rBz, cx/dist, cy/dist, cz/dist, dist, 0.01, 0.70)
		}

		// Project off-axis angular tilt so hinge stays perfectly aligned
		qA := entA.GetQRotation()
		qB := entB.GetQRotation()
		hAx, hAy, hAz := qA.Rotate(j.axisAX, j.axisAY, j.axisAZ)
		hBx, hBy, hBz := qB.Rotate(j.axisBX, j.axisBY, j.axisBZ)
		tx := hAy*hBz - hAz*hBy
		ty := hAz*hBx - hAx*hBz
		tz := hAx*hBy - hAy*hBx
		tLen := math.Sqrt(tx*tx + ty*ty + tz*tz)
		if tLen > 1e-6 {
			tiltAngle := math.Asin(math.Min(tLen, 1.0))
			entA.ProjectAngularAngle(entB, tx/tLen, ty/tLen, tz/tLen, tiltAngle, 0.002, 0.30)
		}

		// Positional angular limit projection
		if j.enableAngularLimit {
			ux, uy, uz, vx, vy, vz := rigid_body.ComputeOrthonormalBasis(hAx, hAy, hAz)
			uBx, uBy, uBz := qB.Rotate(ux, uy, uz)
			cosAng := ux*uBx + uy*uBy + uz*uBz
			sinAng := vx*uBx + vy*uBy + vz*uBz
			theta := math.Atan2(sinAng, cosAng)
			if theta < j.minAngularLimit {
				entA.ProjectAngularAngle(entB, hAx, hAy, hAz, theta-j.minAngularLimit, 0.002, 0.30)
			} else if theta > j.maxAngularLimit {
				entA.ProjectAngularAngle(entB, hAx, hAy, hAz, theta-j.maxAngularLimit, 0.002, 0.30)
			}
		}

	case JointSlider:
		pAx, pAy, pAz, rAx, rAy, rAz := entA.AnchorWorld(j.anchorAX, j.anchorAY, j.anchorAZ)
		pBx, pBy, pBz, rBx, rBy, rBz := entB.AnchorWorld(j.anchorBX, j.anchorBY, j.anchorBZ)
		qA := entA.GetQRotation()
		sx, sy, sz := qA.Rotate(j.axisAX, j.axisAY, j.axisAZ)
		ux, uy, uz, vx, vy, vz := rigid_body.ComputeOrthonormalBasis(sx, sy, sz)

		// Lateral linear position relaxation
		dx, dy, dz := pBx-pAx, pBy-pAy, pBz-pAz
		devU := dx*ux + dy*uy + dz*uz
		devV := dx*vx + dy*vy + dz*vz
		entA.ProjectLinearDistance(entB, rAx, rAy, rAz, rBx, rBy, rBz, ux, uy, uz, devU, 0.01, 0.70)
		entA.ProjectLinearDistance(entB, rAx, rAy, rAz, rBx, rBy, rBz, vx, vy, vz, devV, 0.01, 0.70)

		// Linear limit projection along slide axis
		if j.enableLinearLimit {
			d := dx*sx + dy*sy + dz*sz
			if d < j.minLinearLimit {
				entA.ProjectLinearDistance(entB, rAx, rAy, rAz, rBx, rBy, rBz, sx, sy, sz, d-j.minLinearLimit, 0.01, 0.70)
			} else if d > j.maxLinearLimit {
				entA.ProjectLinearDistance(entB, rAx, rAy, rAz, rBx, rBy, rBz, sx, sy, sz, d-j.maxLinearLimit, 0.01, 0.70)
			}
		}

		// Angular orientation relaxation (slider must not twist)
		qB := entB.GetQRotation()
		qA.Multiplies(j.restQ, j.qTargetScratch)
		j.qErrScratch.Conjugate(j.qTargetScratch)
		j.qErrScratch.MultipliesSelf(qB)
		ex, ey, ez, ew := j.qErrScratch.Get()
		if ew < 0.0 {
			ex, ey, ez, ew = -ex, -ey, -ez, -ew
		}
		vSq := ex*ex + ey*ey + ez*ez
		if vSq > 1e-12 {
			vLen := math.Sqrt(vSq)
			angle := 2.0 * math.Atan2(vLen, ew)
			axW, ayW, azW := j.qTargetScratch.Rotate(ex/vLen, ey/vLen, ez/vLen)
			entA.ProjectAngularAngle(entB, axW, ayW, azW, angle, 0.002, 0.30)
		}

	case JointUniversal, JointCylindrical, JointGeneric6DoF:
		pAx, pAy, pAz, rAx, rAy, rAz := entA.AnchorWorld(j.anchorAX, j.anchorAY, j.anchorAZ)
		pBx, pBy, pBz, rBx, rBy, rBz := entB.AnchorWorld(j.anchorBX, j.anchorBY, j.anchorBZ)
		cx, cy, cz := pBx-pAx, pBy-pAy, pBz-pAz
		dist := math.Sqrt(cx*cx + cy*cy + cz*cz)
		if dist > 1e-4 {
			entA.ProjectLinearDistance(entB, rAx, rAy, rAz, rBx, rBy, rBz, cx/dist, cy/dist, cz/dist, dist, 0.01, 0.70)
		}
	}
}
