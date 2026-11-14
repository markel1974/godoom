package rigid_body

import (
	"math"
)

const (
	airFriction = 0.99

	// minVelocity represents the minimum velocity threshold below which motion is considered negligible.
	minVelocity = 0.1

	// minThickness represents the minimum allowable thickness value to prevent calculations with near-zero dimensions.
	minThickness = 0.01

	// safetyMargin defines a proportional buffer value, typically used to ensure numerical stability in calculations.
	safetyMargin float64 = 0.90

	floorNormalThreshold = 0.7

	defaultPenetrationSlop = 0.02

	// Tight skin slop for immutable static world walls, floors, and ceilings.
	staticPenetrationSlop = 0.001

	// Positional Gauss-Seidel relaxation factor applied per iteration during StageResolvePosition.
	posCorrectionFactor = 0.75

	maxTiltVel = 2.5

	maxYawVel = 4.5

	sleepEpsilon = 0.005

	angSleepEpsilon = 0.08

	// Continuous calm duration and motion thresholds before putting a grounded island to sleep.
	sleepTimeThreshold = 0.45
	sleepAngVelMax     = angSleepEpsilon * 1.5
	sleepAngVelMaxSq   = sleepAngVelMax * sleepAngVelMax
	sleepPushSlopSq    = (defaultPenetrationSlop * 2.0) * (defaultPenetrationSlop * 2.0)

	rotScale = 1.0

	defaultInertia = 32.0

	// Inertia validation and computation constants.
	minInertiaMass      = 25.0
	minInertiaDiagRatio = 0.1
	cuboidInertiaFactor = 1.0 / 12.0
	gyroEnergyEpsilon   = 1e-12
	restitutionEpsilon  = 1e-6
	tangentLenEpsilon   = 1e-8

	// Exponential damping bases (per second, scaled via math.Pow(base, dt)).
	groundYawDampingBase     = 0.02
	groundTiltDampingBase    = 0.05
	lowEnergyTiltDampingBase = 0.004
	airAngDampingBase        = 0.65

	// Upright PD stabilizer gains and thresholds (critically damped).
	uprightTiltSqThreshold      = 1e-6
	uprightSleepTiltSqThreshold = 1e-5
	uprightAxisZThreshold       = 0.999999
	uprightGroundStiffness      = 140.0
	uprightGroundDamping        = 24.0
	uprightAirStiffness         = 60.0
	uprightAirDamping           = 15.5
	uprightFlipLenEpsilon       = 1e-6

	// Low-energy ground settling and yaw stabilization thresholds.
	lowEnergyGroundTiltSq  = 2.5
	lowEnergyGroundHorizSq = 6400.0
	nearFlatFaceAlignCos   = 0.99
	lowEnergyYawSpinLimit  = 1.5

	// Impact restitution slop thresholds.
	restitutionSlopGravityFactor  = 2.0
	minRestitutionSlop            = 2.0
	groundedImpactNormalThreshold = 0.25

	// Near-flat floor face-slap absorption and unstable ridge kick constants.
	faceSlapAlignCosMin    = 0.985
	faceSlapAlignRange     = 1.0 - faceSlapAlignCosMin
	faceSlapBaseKeep       = 0.60
	faceSlapKeepDrop       = 0.45
	restingNormalAlignCos  = 0.992
	restingNormalTiltSqMax = 1.0
	faceSlapVzAlignCos     = 0.994
	ridgeMaxAlignCos       = 0.90
	ridgeLeverSqMax        = 0.04
	ridgeTiltSqMax         = 1e-4
	ridgeBreakKick         = 0.1

	// Low-energy floor friction torque attenuation thresholds.
	frictionLowEnergyHorizSq = 10000.0
	frictionLowEnergyTiltSq  = 2.25
	frictionEdgeRotScale     = 0.35
)

// Cinematic represents the physical state and 6-DoF rigid-body dynamics of an object in a simulation.
type Cinematic struct {
	mass    float64
	invMass float64
	// World-space inverse inertia tensor.
	invIxx float64
	invIxy float64
	invIxz float64
	invIyy float64
	invIyz float64
	invIzz float64
	// World-space inertia tensor.
	ixx float64
	ixy float64
	ixz float64
	iyy float64
	iyz float64
	izz float64

	vx                   float64
	vy                   float64
	vz                   float64
	ax                   float64
	ay                   float64
	az                   float64
	wx                   float64
	wy                   float64
	wz                   float64
	torqueX              float64
	torqueY              float64
	torqueZ              float64
	vMin                 float64
	airFriction          float64
	airDamping           float64
	airAngDamping        float64
	gForce               float64
	material             *PhysicsMaterial
	maxVelocitySq        float64
	sleepThresholdSq     float64
	dt                   float64
	terminalZVelocity    float64
	onGround             bool
	isTilted             bool
	keepUpright          bool
	uprightLocked        bool
	groundYawDamping     float64
	groundTiltDamping    float64
	lowEnergyTiltDamping float64
	faceAlignCos         float64
	ridgeBreakApplied    bool
	sleeping             bool
	sleepTimer           float64
	r                    *Matrix3x3
	body                 *Matrix3x3 // Body-space inertia tensor.
	bodyInv              *Matrix3x3 // Body-space inverse inertia tensor.
	world                *Matrix3x3
	worldInv             *Matrix3x3
}

// NewCinematic initializes a new Cinematic instance with specified mass, restitution, ground friction, and gravitational force.
// Mass and restitution values are clamped to ensure physical validity. Returns a pointer to the initialized Cinematic.
func NewCinematic(dt float64, mass, restitution, groundFriction float64, restCombine, frictionCombine int, gForce float64) *Cinematic {
	invMass := 0.0
	if mass <= 0.0 {
		mass = 0.0
	} else {
		invMass = 1.0 / mass
	}
	mat := NewPhysicsMaterial(restitution, groundFriction, CombineMode(restCombine), CombineMode(frictionCombine), dt)
	a := &Cinematic{
		gForce:           gForce,
		vMin:             minVelocity,
		sleepThresholdSq: minVelocity * minVelocity,
		mass:             mass,
		invMass:          invMass,
		material:         mat,
		onGround:         false,
		airFriction:      airFriction,
		keepUpright:      false,
		r:                NewMatrix3x3(),
		body:             NewMatrix3x3(),
		bodyInv:          NewMatrix3x3(),
		world:            NewMatrix3x3(),
		worldInv:         NewMatrix3x3(),
	}

	a.UpdateInertia(defaultInertia, defaultInertia, defaultInertia)
	a.SetDt(dt)
	a.SetOnGround(false)
	return a
}

// SetKeepUpright enables or disables the automatic upright PD stabilizer during Update.
func (e *Cinematic) SetKeepUpright(keepUpright bool) {
	if !keepUpright && e.uprightLocked {
		return
	}
	e.keepUpright = keepUpright
	e.SetOnGround(false)
}

// LockUpright locks upright stabilization on so billboard sprites cannot tumble on death.
func (e *Cinematic) LockUpright() {
	e.uprightLocked = true
	e.keepUpright = true
}

// IsUprightLocked reports whether upright stabilization is locked.
func (e *Cinematic) IsUprightLocked() bool {
	return e.uprightLocked
}

// KeepUpright reports whether the automatic upright PD stabilizer is enabled.
func (e *Cinematic) KeepUpright() bool {
	return e.keepUpright
}

// SetInertiaTensor sets the body-space inertia tensor from the mesh inertia tensor expressed per unit mass.
func (e *Cinematic) SetInertiaTensor(ixx, ixy, ixz, iyy, iyz, izz float64) bool {
	if e.invMass == 0.0 || e.mass <= 0.0 {
		return false
	}

	factor := math.Max(e.mass, minInertiaMass) * rotScale

	a := ixx * factor
	b := ixy * factor
	c := ixz * factor
	d := iyy * factor
	f := iyz * factor
	g := izz * factor

	// Inverse of the symmetric matrix:
	//
	// | a b c |
	// | b d f |
	// | c f g |

	c00 := d*g - f*f
	c01 := c*f - b*g
	c02 := b*f - c*d
	c11 := a*g - c*c
	c12 := b*c - a*f
	c22 := a*d - b*b

	det := a*c00 + b*c01 + c*c02

	minDiag := factor * minInertiaDiagRatio
	// Reject non-positive-definite, degenerate, or invalid inertia tensors.
	if a <= minDiag || d <= minDiag || g <= minDiag ||
		c00 <= 0.0 || c11 <= 0.0 || c22 <= 0.0 ||
		det <= 0.0 || math.IsNaN(det) || math.IsInf(det, 0) {
		return false
	}

	invDet := 1.0 / det

	invA := c00 * invDet
	invB := c01 * invDet
	invC := c02 * invDet
	invD := c11 * invDet
	invF := c12 * invDet
	invG := c22 * invDet

	if math.IsNaN(invA) || math.IsInf(invA, 0) ||
		math.IsNaN(invB) || math.IsInf(invB, 0) ||
		math.IsNaN(invC) || math.IsInf(invC, 0) ||
		math.IsNaN(invD) || math.IsInf(invD, 0) ||
		math.IsNaN(invF) || math.IsInf(invF, 0) ||
		math.IsNaN(invG) || math.IsInf(invG, 0) {
		return false
	}

	//e.bodyIxx, e.bodyIxy, e.bodyIxz = a, b, c
	//e.bodyIyy, e.bodyIyz, e.bodyIzz = d, f, g
	e.body.Set(a, b, c, b, d, f, c, f, g)

	//e.bodyInvIxx, e.bodyInvIxy, e.bodyInvIxz = invA, invB, invC
	//e.bodyInvIyy, e.bodyInvIyz, e.bodyInvIzz = invD, invF, invG
	e.bodyInv.Set(invA, invB, invC, invB, invD, invF, invC, invF, invG)

	return true
}

// UpdateInertia computes the inverse principal moments of inertia for a solid cuboid of dimensions (w, h, d)
// and initializes the world-space inverse inertia tensor.
func (e *Cinematic) UpdateInertia(w, h, d float64) {
	if e.invMass == 0.0 || e.mass <= 0.0 {
		e.body.Set(0, 0, 0, 0, 0, 0, 0, 0, 0)
		e.bodyInv.Set(0, 0, 0, 0, 0, 0, 0, 0, 0)

		e.ixx, e.ixy, e.ixz = 0, 0, 0
		e.iyy, e.iyz, e.izz = 0, 0, 0

		e.invIxx, e.invIxy, e.invIxz = 0, 0, 0
		e.invIyy, e.invIyz, e.invIzz = 0, 0, 0
		return
	}

	if w < defaultInertia {
		w = defaultInertia
	}
	if h < defaultInertia {
		h = defaultInertia
	}
	if d < defaultInertia {
		d = defaultInertia
	}

	// Solid cuboid inertia per unit mass.
	e.SetInertiaTensor((h*h+d*d)*cuboidInertiaFactor, 0, 0, (w*w+d*d)*cuboidInertiaFactor, 0, (w*w+h*h)*cuboidInertiaFactor)

	// Initially identity orientation.
	e.SyncWorldInertia(1, 0, 0, 0, 1, 0, 0, 0, 1)
}

// SyncWorldInertia updates the world-space inertia and inverse inertia tensors from the local orientation axes
// expressed in world space: row 0 = local X (a00,a01,a02), row 1 = local Y (a10,a11,a12), row 2 = local Z (a20,a21,a22).
// It clears both tensors when the body has zero inverse mass.
func (e *Cinematic) SyncWorldInertia(a00, a01, a02, a10, a11, a12, a20, a21, a22 float64) {
	e.faceAlignCos = math.Max(math.Abs(a02), math.Max(math.Abs(a12), math.Abs(a22)))
	if e.invMass == 0.0 {
		e.ixx, e.ixy, e.ixz = 0, 0, 0
		e.iyy, e.iyz, e.izz = 0, 0, 0

		e.invIxx, e.invIxy, e.invIxz = 0, 0, 0
		e.invIyy, e.invIyz, e.invIzz = 0, 0, 0
		return
	}

	e.r.Set(a00, a01, a02, a10, a11, a12, a20, a21, a22)
	e.r.Multiply(e.world, e.worldInv, e.body, e.bodyInv)

	e.ixx, e.ixy, e.ixz, e.iyy, e.iyz, e.izz = e.world.GetSymmetric()
	e.invIxx, e.invIxy, e.invIxz, e.invIyy, e.invIyz, e.invIzz = e.worldInv.GetSymmetric()
}

// ApplyWorldInvInertia multiplies a world-space vector by the 3x3 world-space inverse inertia tensor I_world^-1.
func (e *Cinematic) ApplyWorldInvInertia(tx, ty, tz float64) (float64, float64, float64) {
	return e.invIxx*tx + e.invIxy*ty + e.invIxz*tz, e.invIxy*tx + e.invIyy*ty + e.invIyz*tz, e.invIxz*tx + e.invIyz*ty + e.invIzz*tz
}

// SetDt configures the time step, ground friction, and air friction values for the cinematic simulation.
func (e *Cinematic) SetDt(dt float64) {
	e.dt = dt
	e.groundYawDamping = math.Pow(groundYawDampingBase, e.dt)
	e.groundTiltDamping = math.Pow(groundTiltDampingBase, e.dt)
	e.lowEnergyTiltDamping = math.Pow(lowEnergyTiltDampingBase, e.dt)
	e.material.SetDt(e.dt)
	e.airDamping = math.Pow(e.airFriction, e.dt)
	e.airAngDamping = math.Pow(airAngDampingBase, e.dt)
	if e.airDamping >= 1.0 {
		// No atmospheric friction, terminal velocity would tend to infinity
		e.terminalZVelocity = -math.MaxFloat64
	} else {
		// Calculation of the integrator's asymptote
		e.terminalZVelocity = (-e.gForce * e.dt * e.airDamping) / (1.0 - e.airDamping)
	}
	maxVelocity := (minThickness * safetyMargin) / e.dt
	e.maxVelocitySq = maxVelocity * maxVelocity
}

// Stop halts the entity's motion by setting its linear and angular velocity components to zero.
func (e *Cinematic) Stop() {
	e.ax = 0.0
	e.ay = 0.0
	e.az = 0.0
	e.vx = 0.0
	e.vy = 0.0
	e.vz = 0.0
	e.StopAngular()
}

// StopAngular halts the entity's rotational motion by zeroing angular velocity and angular acceleration.
func (e *Cinematic) StopAngular() {
	e.wx = 0.0
	e.wy = 0.0
	e.wz = 0.0
	e.torqueX = 0.0
	e.torqueY = 0.0
	e.torqueZ = 0.0
}

// GetAngularVelocity returns the current angular velocity components (wx, wy, wz) in rad/s.
func (e *Cinematic) GetAngularVelocity() (float64, float64, float64) {
	return e.wx, e.wy, e.wz
}

// SetAngularVelocity sets the angular velocity components (wx, wy, wz) in rad/s.
func (e *Cinematic) SetAngularVelocity(wx, wy, wz float64) {
	if wx != 0.0 || wy != 0.0 || wz != 0.0 {
		e.WakeUp()
	}
	e.wx = wx
	e.wy = wy
	e.wz = wz
}

// IsSleeping reports whether the body is currently asleep in the physics solver.
func (e *Cinematic) IsSleeping() bool {
	return e.sleeping
}

// WakeUp wakes the body from sleep and resets its calm accumulator timer.
func (e *Cinematic) WakeUp() {
	e.sleeping = false
	e.sleepTimer = 0.0
}

// SetSleeping sets the sleeping state; entering sleep zeroes residual velocities, accelerations, and torques.
func (e *Cinematic) SetSleeping(sleeping bool) {
	if !sleeping {
		e.WakeUp()
		return
	}
	e.sleeping = true
	e.sleepTimer = sleepTimeThreshold
	e.Stop()
}

// UpdateSleepState advances or resets the calm accumulator based on post-solver linear/angular speed and push-out delta.
// It returns true when the body satisfies all rest conditions for at least sleepTimeThreshold seconds.
func (e *Cinematic) UpdateSleepState(canSleep bool, pushSq float64) bool {
	if !canSleep || e.invMass <= 0.0 || !e.onGround || (e.keepUpright && e.isTilted) || pushSq > sleepPushSlopSq {
		e.sleepTimer = 0.0
		return false
	}
	if e.ax != 0.0 || e.ay != 0.0 || e.az != 0.0 || e.torqueX != 0.0 || e.torqueY != 0.0 || e.torqueZ != 0.0 {
		e.sleepTimer = 0.0
		return false
	}
	linThresh := math.Max(e.vMin*2.0, (sleepEpsilon/e.dt)*1.5)
	vSq := e.vx*e.vx + e.vy*e.vy + e.vz*e.vz
	wSq := e.wx*e.wx + e.wy*e.wy + e.wz*e.wz
	if vSq > linThresh*linThresh || wSq > sleepAngVelMaxSq {
		e.sleepTimer = 0.0
		return false
	}
	e.sleepTimer += e.dt
	return e.sleepTimer >= sleepTimeThreshold
}

// AddImpulseAtOffset applies an angular impulse Δω = I_world^-1 (r × J) generated by linear impulse (jx, jy, jz) at lever arm (rx, ry, rz).
func (e *Cinematic) AddImpulseAtOffset(jx, jy, jz, rx, ry, rz float64) {
	if e.invMass == 0.0 {
		return
	}
	if jx != 0.0 || jy != 0.0 || jz != 0.0 {
		e.WakeUp()
	}
	// Angular impulse: ΔL = r × J
	tx := ry*jz - rz*jy
	ty := rz*jx - rx*jz
	tz := rx*jy - ry*jx
	// Angular velocity variation: Δω = I_world^-1 * ΔL
	dwx, dwy, dwz := e.ApplyWorldInvInertia(tx, ty, tz)
	e.wx = math.Max(-maxTiltVel, math.Min(maxTiltVel, e.wx+dwx))
	e.wy = math.Max(-maxTiltVel, math.Min(maxTiltVel, e.wy+dwy))
	e.wz = math.Max(-maxYawVel, math.Min(maxYawVel, e.wz+dwz))
}

// IsRotating reports whether the object has any non-zero angular velocity.
func (e *Cinematic) IsRotating() bool {
	return e.wx != 0.0 || e.wy != 0.0 || e.wz != 0.0
}

// IsTilted reports whether the cinematic is tilted.
func (e *Cinematic) IsTilted() bool {
	return e.isTilted
}

// SetOnGround sets the entity's ground state and adjusts vertical velocity when grounded flat.
func (e *Cinematic) SetOnGround(onGround bool) {
	e.onGround = onGround
	if e.onGround && (!e.isTilted || e.keepUpright) && e.vz < 0.0 {
		e.vz = 0.0
	}
}

// IsOnGround returns true if the entity is currently in contact with the ground, otherwise false.
func (e *Cinematic) IsOnGround() bool {
	return e.onGround
}

// GetVx returns the current velocity of the entity along the x-axis.
func (e *Cinematic) GetVx() float64 {
	return e.vx
}

// GetVy returns the current velocity value along the Y-axis for the Cinematic instance.
func (e *Cinematic) GetVy() float64 {
	return e.vy
}

// GetVz retrieves the current velocity along the Z-axis of the Cinematic entity.
func (e *Cinematic) GetVz() float64 {
	return e.vz
}

// SetVx sets the velocity along the X-axis for the Cinematic object.
func (e *Cinematic) SetVx(vx float64) {
	if vx != 0.0 {
		e.WakeUp()
	}
	e.vx = vx
}

// SetVy sets the vertical velocity (vy) of the Cinematic object to the specified value.
func (e *Cinematic) SetVy(vy float64) {
	if vy != 0.0 {
		e.WakeUp()
	}
	e.vy = vy
}

// SetVz sets the z-axis velocity (vz) for the Cinematic object.
func (e *Cinematic) SetVz(vz float64) {
	if vz != 0.0 {
		e.WakeUp()
	}
	e.vz = vz
}

// SetV sets the velocity components vx, vy, and vz for the cinematic entity.
func (e *Cinematic) SetV(vx, vy, vz float64) {
	if vx != 0.0 || vy != 0.0 || vz != 0.0 {
		e.WakeUp()
	}
	e.vx = vx
	e.vy = vy
	e.vz = vz
}

// AddV increments the velocity components (vx, vy, vz) of the Cinematic object by the specified values.
func (e *Cinematic) AddV(vx, vy, vz float64) {
	if vx != 0.0 || vy != 0.0 || vz != 0.0 {
		e.WakeUp()
	}
	e.vx += vx
	e.vy += vy
	e.vz += vz
}

// SubV subtracts the given velocity components (vx, vy, vz) from the Cinematic object's current velocity.
func (e *Cinematic) SubV(vx, vy, vz float64) {
	e.vx -= vx
	e.vy -= vy
	e.vz -= vz
}

// ClampNormalVelocity removes any velocity component directed opposite to outward surface normal (nx, ny, nz).
func (e *Cinematic) ClampNormalVelocity(nx, ny, nz float64) {
	vn := e.vx*nx + e.vy*ny + e.vz*nz
	if vn < 0.0 {
		e.vx -= vn * nx
		e.vy -= vn * ny
		e.vz -= vn * nz
	}
}

// GetDt retrieves the current delta time (dt) value for the Cinematic instance.
func (e *Cinematic) GetDt() float64 {
	return e.dt
}

// GetInvMass returns the inverse mass of the Cinematic object, which is used for physics calculations.
func (e *Cinematic) GetInvMass() float64 {
	return e.invMass
}

// GetMass returns the mass of the Cinematic object.
func (e *Cinematic) GetMass() float64 {
	return e.mass
}

// GetGForce returns the value of the gravitational force acting on the Cinematic entity.
func (e *Cinematic) GetGForce() float64 {
	return e.gForce
}

// GetVelocity returns the current velocity components (vx, vy, vz) of the cinematic object.
func (e *Cinematic) GetVelocity() (float64, float64, float64) {
	return e.vx, e.vy, e.vz
}

// GetFrameDisplacement calculates and returns the displacement in the x, y, and z directions based on velocity and time step.
func (e *Cinematic) GetFrameDisplacement() (float64, float64, float64) {
	return e.vx * e.dt, e.vy * e.dt, e.vz * e.dt
}

// IsMoving returns true if the object has any non-zero linear or angular velocity components.
func (e *Cinematic) IsMoving() bool {
	return e.vx != 0.0 || e.vy != 0.0 || e.vz != 0.0 || e.wx != 0.0 || e.wy != 0.0 || e.wz != 0.0
}

// AddForce applies a force to the object by modifying its acceleration components (ax, ay, az) using the force and inverse mass.
func (e *Cinematic) AddForce(fx, fy, fz float64) {
	if fx != 0.0 || fy != 0.0 || fz != 0.0 {
		e.WakeUp()
	}
	e.ax += fx * e.invMass
	e.ay += fy * e.invMass
	e.az += fz * e.invMass
}

// AddForceAtPoint applies a world-space force at a point and accumulates the resulting force and torque.
func (e *Cinematic) AddForceAtPoint(fx, fy, fz float64, px, py, pz float64, cx, cy, cz float64) {
	if e.invMass == 0.0 {
		return
	}
	if fx != 0.0 || fy != 0.0 || fz != 0.0 {
		e.WakeUp()
	}
	// Accumulate linear acceleration.
	e.AddForce(fx, fy, fz)
	// Lever arm from center of mass to application point.
	rx := px - cx
	ry := py - cy
	rz := pz - cz
	// Torque: τ = r × F
	tx := ry*fz - rz*fy
	ty := rz*fx - rx*fz
	tz := rx*fy - ry*fx
	// Accumulate world-space torque.
	e.torqueX += tx
	e.torqueY += ty
	e.torqueZ += tz
}

// SetIsTilted sets the tilted state of the cinematic body.
func (e *Cinematic) SetIsTilted(isTilted bool) {
	e.isTilted = isTilted
}

// Update advances the cinematic physics state using the supplied local Z orientation axis components (ax0, ax1, ax2).
func (e *Cinematic) Update(ax0, ax1, ax2 float64) {
	if e.invMass == 0.0 {
		return
	}
	e.ridgeBreakApplied = false

	if e.keepUpright {
		tiltSq := ax0*ax0 + ax1*ax1
		e.isTilted = tiltSq > uprightTiltSqThreshold || ax2 < uprightAxisZThreshold
	}

	if e.isTilted && e.keepUpright {
		stiffness := uprightGroundStiffness
		damping := uprightGroundDamping
		if !e.onGround {
			stiffness = uprightAirStiffness
			damping = uprightAirDamping
		}
		errX := ax1
		errY := -ax0
		if ax2 < 0.0 {
			lenXY := math.Hypot(errX, errY)
			if lenXY > uprightFlipLenEpsilon {
				errX /= lenXY
				errY /= lenXY
			} else {
				errX = 1.0
			}
		}
		e.wx += (errX*stiffness - e.wx*damping) * e.dt
		e.wy += (errY*stiffness - e.wy*damping) * e.dt
	}

	// Integrate external torque first, then apply energy-preserving gyroscopic precession.
	if e.torqueX != 0.0 || e.torqueY != 0.0 || e.torqueZ != 0.0 {
		ax, ay, az := e.ApplyWorldInvInertia(e.torqueX, e.torqueY, e.torqueZ)
		e.wx += ax * e.dt
		e.wy += ay * e.dt
		e.wz += az * e.dt
	}
	e.integrateGyroscopic()
	// Reset torque accumulators.
	e.torqueX = 0.0
	e.torqueY = 0.0
	e.torqueZ = 0.0

	// Angular damping.
	tiltDamping := e.airAngDamping
	yawDamping := e.airAngDamping
	groundDamping := e.material.GetGroundDamping()
	if e.onGround {
		faceDamping := e.groundYawDamping * e.groundYawDamping
		yawDamping = e.groundYawDamping
		if !e.isTilted || e.keepUpright {
			if e.keepUpright {
				tiltDamping = groundDamping
				yawDamping = faceDamping
			} else {
				// Flat face resting contact strongly damps residual tilt and yaw micro-vibrations.
				tiltDamping = faceDamping
				yawDamping = faceDamping
			}
		} else {
			// Tilted free body touching the ground: apply rolling resistance and stronger low-energy damping.
			tiltDamping = e.groundTiltDamping
			if (e.wx*e.wx+e.wy*e.wy) < lowEnergyGroundTiltSq && (e.vx*e.vx+e.vy*e.vy) < lowEnergyGroundHorizSq {
				tiltDamping = e.lowEnergyTiltDamping
				if e.faceAlignCos > nearFlatFaceAlignCos && math.Abs(e.wz) < lowEnergyYawSpinLimit {
					yawDamping = e.lowEnergyTiltDamping
				}
			}
		}
	}
	e.wx *= tiltDamping
	e.wy *= tiltDamping
	e.wz *= yawDamping

	// Angular velocity limits.
	e.wx = math.Max(-maxTiltVel, math.Min(maxTiltVel, e.wx))
	e.wy = math.Max(-maxTiltVel, math.Min(maxTiltVel, e.wy))
	e.wz = math.Max(-maxYawVel, math.Min(maxYawVel, e.wz))

	// Angular sleep for face-aligned bodies, plus grounded yaw/upright jitter deadzone.
	if !e.isTilted || (e.keepUpright && e.onGround && (ax0*ax0+ax1*ax1) < uprightSleepTiltSqThreshold) {
		if math.Abs(e.wx) < angSleepEpsilon {
			e.wx = 0.0
		}
		if math.Abs(e.wy) < angSleepEpsilon {
			e.wy = 0.0
		}
		if math.Abs(e.wz) < angSleepEpsilon {
			e.wz = 0.0
		}
	} else if e.onGround && (e.keepUpright || e.faceAlignCos > nearFlatFaceAlignCos) {
		if math.Abs(e.wz) < angSleepEpsilon {
			e.wz = 0.0
		}
	}

	// Integrate linear velocity.
	e.vx += e.ax * e.dt
	e.vy += e.ay * e.dt

	groundedFlat := e.onGround && (!e.isTilted || e.keepUpright)

	if groundedFlat {
		if e.vz < 0.0 {
			e.vz = 0.0
		}
		if e.az > 0.0 {
			e.vz += e.az * e.dt
		}
	} else {
		e.vz += (e.az - e.gForce) * e.dt
	}

	// Reset linear acceleration accumulators.
	e.ax = 0.0
	e.ay = 0.0
	e.az = 0.0

	// Grounded sleep check (avoids freezing airborne projectiles at trajectory apex).
	if groundedFlat {
		dx := e.vx * e.dt
		dy := e.vy * e.dt
		dz := e.vz * e.dt
		if math.Abs(dx) < sleepEpsilon && math.Abs(dy) < sleepEpsilon && math.Abs(dz) < sleepEpsilon {
			e.vx = 0.0
			e.vy = 0.0
			e.vz = 0.0
			return
		}
	}

	// Linear damping and planar sleep threshold.
	if e.onGround {
		e.vx *= groundDamping
		e.vy *= groundDamping
		e.vz *= e.airDamping
		if groundedFlat {
			if math.Abs(e.vx) < e.vMin {
				e.vx = 0.0
			}
			if math.Abs(e.vy) < e.vMin {
				e.vy = 0.0
			}
			if e.vz <= 0.0 {
				e.vz = 0.0
			}
		}
	} else {
		e.vx *= e.airDamping
		e.vy *= e.airDamping
		e.vz *= e.airDamping
	}

	// Terminal velocity clamping.
	if e.vz < e.terminalZVelocity {
		e.vz = e.terminalZVelocity
	}
}

// ResolveImpact resolves a 6-DoF rigid-body collision between two Cinematic objects,
// applying normal and Coulomb tangential impulses to both linear velocity v and angular velocity ω.
func (e *Cinematic) ResolveImpact(e2 *Cinematic, nx, ny, nz float64, cx1, cy1, cz1 float64, cx2, cy2, cz2 float64, px, py, pz float64) {
	e.ResolveImpactAccumulated(e2, nx, ny, nz, cx1, cy1, cz1, cx2, cy2, cz2, px, py, pz, nil, nil, nil, nil, nil, nil)
}

// WarmStartImpact pre-applies persisted normal and tangential impulses before PGS iterations begin.
func (e *Cinematic) WarmStartImpact(e2 *Cinematic, nx, ny, nz float64, cx1, cy1, cz1 float64, cx2, cy2, cz2 float64, px, py, pz float64, accNormal, accTx, accTy, accTz *float64) {
	if accNormal == nil || *accNormal <= 0.0 {
		return
	}
	nLen := math.Sqrt(nx*nx + ny*ny + nz*nz)
	if nLen <= 0.0 {
		*accNormal = 0.0
		if accTx != nil && accTy != nil && accTz != nil {
			*accTx, *accTy, *accTz = 0.0, 0.0, 0.0
		}
		return
	}
	nx /= nLen
	ny /= nLen
	nz /= nLen

	invMass1, invMass2 := effectiveContactInvMasses(e, e2, nz)
	if invMass1 == 0.0 && invMass2 == 0.0 {
		return
	}

	r1x, r1y, r1z := px-cx1, py-cy1, pz-cz1
	r2x, r2y, r2z := px-cx2, py-cy2, pz-cz2

	jn := *accNormal
	jx, jy, jz := jn*nx, jn*ny, jn*nz
	e.vx += jx * invMass1
	e.vy += jy * invMass1
	e.vz += jz * invMass1
	e2.vx -= jx * invMass2
	e2.vy -= jy * invMass2
	e2.vz -= jz * invMass2

	rotScale1N, _, _ := e.normalRotScale(nz, 0.0)
	rotScale2N, _, _ := e2.normalRotScale(-nz, 0.0)
	if invMass1 > 0.0 {
		u1nx, u1ny, u1nz, _ := e.rotationalMass(r1x, r1y, r1z, nx, ny, nz)
		e.addClampedAngular(jn*rotScale1N, u1nx, u1ny, u1nz)
	}
	if invMass2 > 0.0 {
		u2nx, u2ny, u2nz, _ := e2.rotationalMass(r2x, r2y, r2z, nx, ny, nz)
		e2.addClampedAngular(-jn*rotScale2N, u2nx, u2ny, u2nz)
	}

	if accTx != nil && accTy != nil && accTz != nil {
		wtx, wty, wtz := *accTx, *accTy, *accTz
		tDotN := wtx*nx + wty*ny + wtz*nz
		wtx -= tDotN * nx
		wty -= tDotN * ny
		wtz -= tDotN * nz
		maxTotalFriction := jn * e.material.CombineFriction(e2.material)
		if tMag := math.Sqrt(wtx*wtx + wty*wty + wtz*wtz); tMag > maxTotalFriction && tMag > 0.0 {
			scale := maxTotalFriction / tMag
			wtx *= scale
			wty *= scale
			wtz *= scale
		}
		*accTx, *accTy, *accTz = wtx, wty, wtz
		fRotBase1 := e.frictionRotBaseScale(nz)
		fRotBase2 := e2.frictionRotBaseScale(-nz)
		applyTangentImpulse(e, e2, invMass1, invMass2, r1x, r1y, r1z, r2x, r2y, r2z, wtx, wty, wtz, fRotBase1, fRotBase2)
	}
}

func effectiveContactInvMasses(e, e2 *Cinematic, nz float64) (float64, float64) {
	invMass1 := e.invMass
	invMass2 := e2.invMass
	if e2.onGround && nz > floorNormalThreshold {
		invMass2 = 0.0
	} else if e.onGround && nz < -floorNormalThreshold {
		invMass1 = 0.0
	}
	return invMass1, invMass2
}

func applyTangentImpulse(e, e2 *Cinematic, invMass1, invMass2, r1x, r1y, r1z, r2x, r2y, r2z, jtx, jty, jtz, fRotBase1, fRotBase2 float64) {
	dtMag := math.Sqrt(jtx*jtx + jty*jty + jtz*jtz)
	if dtMag <= gyroEnergyEpsilon {
		return
	}
	fx := jtx / dtMag
	fy := jty / dtMag
	fz := jtz / dtMag

	e.vx += jtx * invMass1
	e.vy += jty * invMass1
	e.vz += jtz * invMass1
	e2.vx -= jtx * invMass2
	e2.vy -= jty * invMass2
	e2.vz -= jtz * invMass2
	if invMass1 > 0.0 {
		u1fx, u1fy, u1fz, _ := e.rotationalMass(r1x, r1y, r1z, fx, fy, fz)
		e.applyFrictionAngular(dtMag, fRotBase1, u1fx, u1fy, u1fz)
	}
	if invMass2 > 0.0 {
		u2fx, u2fy, u2fz, _ := e2.rotationalMass(r2x, r2y, r2z, fx, fy, fz)
		e2.applyFrictionAngular(-dtMag, fRotBase2, u2fx, u2fy, u2fz)
	}
}

// ResolveImpactAccumulated resolves a 6-DoF rigid-body collision between two Cinematic objects
// using optional accumulated normal/tangential impulses and target restitution velocity across PGS iterations.
func (e *Cinematic) ResolveImpactAccumulated(e2 *Cinematic, nx, ny, nz float64, cx1, cy1, cz1 float64, cx2, cy2, cz2 float64, px, py, pz float64, accNormal, accTx, accTy, accTz, restTarget *float64, hasRestTarget *bool) {
	// Safe normalization before computing lever arms or normal thresholds.
	nLen := math.Sqrt(nx*nx + ny*ny + nz*nz)
	if nLen <= 0 {
		return
	}

	nx /= nLen
	ny /= nLen
	nz /= nLen

	// Lever arms from the actual collision point to the centers of mass.
	r1x := px - cx1
	r1y := py - cy1
	r1z := pz - cz1

	r2x := px - cx2
	r2y := py - cy2
	r2z := pz - cz2

	// Relative velocity at contact point: v_p = v + ω × r
	vp1x := e.vx + (e.wy*r1z - e.wz*r1y)
	vp1y := e.vy + (e.wz*r1x - e.wx*r1z)
	vp1z := e.vz + (e.wx*r1y - e.wy*r1x)

	vp2x := e2.vx + (e2.wy*r2z - e2.wz*r2y)
	vp2y := e2.vy + (e2.wz*r2x - e2.wx*r2z)
	vp2z := e2.vz + (e2.wx*r2y - e2.wy*r2x)

	vrx := vp1x - vp2x
	vry := vp1y - vp2y
	vrz := vp1z - vp2z
	vRelDotN := vrx*nx + vry*ny + vrz*nz

	firstIter := hasRestTarget == nil || !*hasRestTarget
	warmStarted := accNormal != nil && *accNormal > 0.0
	// If bodies are separating on a fresh non-warm-started contact, no collision to resolve.
	if firstIter && !warmStarted && vRelDotN > 0.0 {
		return
	}

	invMass1, invMass2 := effectiveContactInvMasses(e, e2, nz)

	var u1nx, u1ny, u1nz, rotMass1N float64
	if invMass1 > 0.0 {
		u1nx, u1ny, u1nz, rotMass1N = e.rotationalMass(r1x, r1y, r1z, nx, ny, nz)
	}

	var u2nx, u2ny, u2nz, rotMass2N float64
	if invMass2 > 0.0 {
		u2nx, u2ny, u2nz, rotMass2N = e2.rotationalMass(r2x, r2y, r2z, nx, ny, nz)
	}

	var targetVelN float64
	var actualRestitution float64
	if firstIter {
		if warmStarted {
			// Persistent contact already carrying load: keep zero restitution bounce.
			actualRestitution = 0.0
			targetVelN = 0.0
		} else {
			// Restitution combining with threshold (micro-bounce slop), evaluated once per contact.
			actualRestitution = e.material.CombineRestitution(e2.material)
			if math.Abs(actualRestitution) > restitutionEpsilon {
				restitutionSlop := restitutionSlopGravityFactor * math.Max(e.gForce, e2.gForce) * e.dt
				if restitutionSlop < minRestitutionSlop {
					restitutionSlop = minRestitutionSlop
				}
				if math.Abs(vRelDotN) < restitutionSlop || (e.onGround && nz > groundedImpactNormalThreshold) || (e2.onGround && nz < -groundedImpactNormalThreshold) {
					actualRestitution = 0.0
				}
			} else {
				actualRestitution = 0.0
			}
			targetVelN = -actualRestitution * vRelDotN
		}
		if hasRestTarget != nil && restTarget != nil {
			*restTarget = targetVelN
			*hasRestTarget = true
		}
	} else if restTarget != nil {
		targetVelN = *restTarget
		actualRestitution = targetVelN
	}

	rotScale1N, nearFlat1, slapKeep1 := e.normalRotScale(nz, actualRestitution)
	rotScale2N, nearFlat2, slapKeep2 := e2.normalRotScale(-nz, actualRestitution)

	effMassN := invMass1 + invMass2 + rotScale1N*rotMass1N + rotScale2N*rotMass2N
	if effMassN <= 0.0 {
		return
	}

	// Unilateral accumulated normal impulse clamping (accNormal >= 0):
	// allows trimming back excess warm-start/previous-iteration load without pulling.
	rawJ := (targetVelN - vRelDotN) / effMassN
	var j float64
	if accNormal != nil {
		oldAcc := *accNormal
		candAcc := oldAcc + rawJ
		if candAcc < 0.0 {
			candAcc = 0.0
		}
		*accNormal = candAcc
		j = candAcc - oldAcc
	} else {
		j = math.Max(0.0, rawJ)
	}
	if j != 0.0 {
		// Apply normal impulse delta to linear and angular velocities.
		jx := j * nx
		jy := j * ny
		jz := j * nz
		e.vx += jx * invMass1
		e.vy += jy * invMass1
		e.vz += jz * invMass1
		e2.vx -= jx * invMass2
		e2.vy -= jy * invMass2
		e2.vz -= jz * invMass2

		if invMass1 > 0.0 {
			e.applyNormalAngular(j, rotScale1N, nearFlat1, slapKeep1, u1nx, u1ny, u1nz, r1x, r1y, nz)
		}
		if invMass2 > 0.0 {
			e2.applyNormalAngular(-j, rotScale2N, nearFlat2, slapKeep2, u2nx, u2ny, u2nz, r2x, r2y, -nz)
		}
	}
	hasAccT := accTx != nil && (*accTx != 0.0 || *accTy != 0.0 || *accTz != 0.0)
	if j <= 0.0 && (accNormal == nil || (*accNormal <= 0.0 && !hasAccT)) {
		return
	}

	// Tangential impulse (Coulomb friction): recalculate relative velocity after normal impulse.
	vp1x = e.vx + (e.wy*r1z - e.wz*r1y)
	vp1y = e.vy + (e.wz*r1x - e.wx*r1z)
	vp1z = e.vz + (e.wx*r1y - e.wy*r1x)

	vp2x = e2.vx + (e2.wy*r2z - e2.wz*r2y)
	vp2y = e2.vy + (e2.wz*r2x - e2.wx*r2z)
	vp2z = e2.vz + (e2.wx*r2y - e2.wy*r2x)

	vrx = vp1x - vp2x
	vry = vp1y - vp2y
	vrz = vp1z - vp2z
	vRelDotNPost := vrx*nx + vry*ny + vrz*nz

	// Find tangent vector.
	tx := vrx - (vRelDotNPost * nx)
	ty := vry - (vRelDotNPost * ny)
	tz := vrz - (vRelDotNPost * nz)

	tLen := math.Sqrt(tx*tx + ty*ty + tz*tz)
	if tLen <= tangentLenEpsilon {
		// If normal impulse clamped down to 0, still unwind any lingering warm-started tangential impulse.
		if accNormal != nil && *accNormal <= 0.0 && hasAccT {
			jtx, jty, jtz := -*accTx, -*accTy, -*accTz
			*accTx, *accTy, *accTz = 0.0, 0.0, 0.0
			fRotBase1 := e.frictionRotBaseScale(nz)
			fRotBase2 := e2.frictionRotBaseScale(-nz)
			applyTangentImpulse(e, e2, invMass1, invMass2, r1x, r1y, r1z, r2x, r2y, r2z, jtx, jty, jtz, fRotBase1, fRotBase2)
		}
		return
	}

	tx /= tLen
	ty /= tLen
	tz /= tLen

	var u1tx, u1ty, u1tz, rotMass1T float64
	if invMass1 > 0.0 {
		u1tx, u1ty, u1tz, rotMass1T = e.rotationalMass(r1x, r1y, r1z, tx, ty, tz)
	}

	var u2tx, u2ty, u2tz, rotMass2T float64
	if invMass2 > 0.0 {
		u2tx, u2ty, u2tz, rotMass2T = e2.rotationalMass(r2x, r2y, r2z, tx, ty, tz)
	}

	fRotBase1 := e.frictionRotBaseScale(nz)
	fRotBase2 := e2.frictionRotBaseScale(-nz)
	fRotEff1 := frictionEffectiveScale(fRotBase1, e.wx, e.wy, e.wz, -u1tx, -u1ty, -u1tz)
	fRotEff2 := frictionEffectiveScale(fRotBase2, e2.wx, e2.wy, e2.wz, u2tx, u2ty, u2tz)

	effMassT := invMass1 + invMass2 + fRotEff1*rotMass1T + fRotEff2*rotMass2T
	if effMassT <= 0.0 {
		return
	}

	vRelDotT := vrx*tx + vry*ty + vrz*tz
	candJt := -vRelDotT / effMassT
	// Friction mixing: explicit contact material properties.
	mu := e.material.CombineFriction(e2.material)
	var jtx, jty, jtz float64
	if accNormal != nil && accTx != nil && accTy != nil && accTz != nil {
		// Clamp the total 3D tangent impulse across solver iterations within the Coulomb cone.
		oldTx, oldTy, oldTz := *accTx, *accTy, *accTz
		oldDotN := oldTx*nx + oldTy*ny + oldTz*nz
		oldTx -= oldDotN * nx
		oldTy -= oldDotN * ny
		oldTz -= oldDotN * nz

		newTx := oldTx + candJt*tx
		newTy := oldTy + candJt*ty
		newTz := oldTz + candJt*tz
		maxTotalFriction := *accNormal * mu
		if totalMag := math.Sqrt(newTx*newTx + newTy*newTy + newTz*newTz); totalMag > maxTotalFriction && totalMag > 0.0 {
			scale := maxTotalFriction / totalMag
			newTx *= scale
			newTy *= scale
			newTz *= scale
		}
		*accTx, *accTy, *accTz = newTx, newTy, newTz
		jtx = newTx - oldTx
		jty = newTy - oldTy
		jtz = newTz - oldTz
	} else {
		maxFriction := j * mu
		jt := candJt
		// Clamp tangential impulse within Coulomb cone.
		if math.Abs(jt) > maxFriction {
			jt = math.Copysign(maxFriction, jt)
		}
		jtx = jt * tx
		jty = jt * ty
		jtz = jt * tz
	}

	applyTangentImpulse(e, e2, invMass1, invMass2, r1x, r1y, r1z, r2x, r2y, r2z, jtx, jty, jtz, fRotBase1, fRotBase2)
}

// ResolveElasticPenetration computes positional correction for overlapping cinematic bodies along a collision normal.
// It returns each body's correction amount and whether a correction should be applied.
func (e *Cinematic) ResolveElasticPenetration(other *Cinematic, otherFloor bool, nx, ny, nz float64, penetration float64) (float64, float64, bool) {
	return e.ResolveElasticPenetrationConstrained(other, otherFloor, nx, ny, nz, penetration, true, true)
}

// ResolveElasticPenetrationConstrained splits positional correction between e (along +n) and other (along -n),
// treating pinned/blocked directions (selfMovable == false or otherMovable == false) as having infinite mass.
func (e *Cinematic) ResolveElasticPenetrationConstrained(other *Cinematic, otherFloor bool, nx, ny, nz float64, penetration float64, selfMovable, otherMovable bool) (float64, float64, bool) {
	depth := penetration - defaultPenetrationSlop
	if depth <= 0.0 {
		return 0, 0, false
	}
	invMass1 := e.GetInvMass()
	invMass2 := other.GetInvMass()
	// A body blocked along its push direction or resting on a ground support cannot be pushed into the obstacle.
	if !selfMovable || (e.IsOnGround() && nz < -floorNormalThreshold) {
		invMass1 = 0.0
	}
	if !otherMovable || ((otherFloor || other.IsOnGround()) && nz > floorNormalThreshold) {
		invMass2 = 0.0
	}
	invMassSum := invMass1 + invMass2
	if invMassSum <= 0.0 {
		return 0, 0, false
	}
	correction := depth * posCorrectionFactor
	p1 := correction * (invMass1 / invMassSum)
	p2 := correction * (invMass2 / invMassSum)

	if nz > floorNormalThreshold && (otherFloor || other.IsOnGround()) {
		if (!e.isTilted || e.keepUpright) && e.vz < 0.0 {
			e.vz = 0.0
		}
	} else if nz < -floorNormalThreshold && e.IsOnGround() {
		if (!other.isTilted || other.keepUpright) && other.vz < 0.0 {
			other.vz = 0.0
		}
	}

	return p1, p2, true
}

// ResolveInelasticPenetration computes full rigid positional projection out of static walls, floors, and ceilings.
func (e *Cinematic) ResolveInelasticPenetration(nz float64, pen float64) (float64, bool) {
	pen -= staticPenetrationSlop
	if pen <= 0.0 {
		return 0, false
	}
	if nz > floorNormalThreshold {
		if (!e.isTilted || e.keepUpright) && e.vz < 0.0 {
			e.vz = 0.0
		}
	}
	return pen, true
}

// rotationalMass returns the inverse-inertia response for r x d and its rotational effective mass contribution.
func (e *Cinematic) rotationalMass(rx, ry, rz float64, dx, dy, dz float64) (float64, float64, float64, float64) {
	cx := ry*dz - rz*dy
	cy := rz*dx - rx*dz
	cz := rx*dy - ry*dx
	ux, uy, uz := e.ApplyWorldInvInertia(cx, cy, cz)
	mass := cx*ux + cy*uy + cz*uz
	return ux, uy, uz, mass
}

// angularMomentumFor returns the angular momentum vector for the given angular velocity.
func (e *Cinematic) angularMomentumFor(wx, wy, wz float64) (float64, float64, float64) {
	return e.ixx*wx + e.ixy*wy + e.ixz*wz, e.ixy*wx + e.iyy*wy + e.iyz*wz, e.ixz*wx + e.iyz*wy + e.izz*wz
}

// normalRotScale returns the consistent angular response scale for normal contact impulses and face-slap parameters.
func (e *Cinematic) normalRotScale(floorNz, restitution float64) (float64, bool, float64) {
	nearFlatFloor := !e.keepUpright && floorNz > floorNormalThreshold && restitution == 0.0 && e.faceAlignCos > faceSlapAlignCosMin
	if !nearFlatFloor {
		return 1.0, false, 1.0
	}
	t := (e.faceAlignCos - faceSlapAlignCosMin) / faceSlapAlignRange
	if t > 1.0 {
		t = 1.0
	}
	slapKeep := faceSlapBaseKeep - faceSlapKeepDrop*t
	// Resting near-flat on the floor: attenuate small gravity-induced normal kicks to avoid seesaw/yaw jitter.
	if e.onGround && e.faceAlignCos > restingNormalAlignCos && (e.wx*e.wx+e.wy*e.wy) < restingNormalTiltSqMax {
		return slapKeep, true, slapKeep
	}
	return 1.0, true, slapKeep
}

func (e *Cinematic) addClampedAngular(scale, ux, uy, uz float64) {
	e.wx = math.Max(-maxTiltVel, math.Min(maxTiltVel, e.wx+scale*ux))
	e.wy = math.Max(-maxTiltVel, math.Min(maxTiltVel, e.wy+scale*uy))
	e.wz = math.Max(-maxYawVel, math.Min(maxYawVel, e.wz+scale*uz))
}

// applyNormalAngular applies a normal collision impulse to angular velocity, clamping tilt and yaw spin.
// It also dampens near-flat floor slap rebound and breaks unstable tilted ridge equilibrium once per frame.
func (e *Cinematic) applyNormalAngular(j, rotScaleN float64, nearFlatFloor bool, slapKeep, unx, uny, unz, rx, ry, floorNz float64) {
	preWx, preWy, preWz := e.wx, e.wy, e.wz
	dWx := j * rotScaleN * unx
	dWy := j * rotScaleN * uny
	dWz := j * rotScaleN * unz

	e.wx = math.Max(-maxTiltVel, math.Min(maxTiltVel, preWx+dWx))
	e.wy = math.Max(-maxTiltVel, math.Min(maxTiltVel, preWy+dWy))
	e.wz = math.Max(-maxYawVel, math.Min(maxYawVel, preWz+dWz))

	if j > 0.0 && nearFlatFloor {
		// Face-slap absorption: when a rotating free body lands near-flat against a floor without bounce,
		// opposing corners absorb seesaw rebound spin and vertical kick from single-point corner contact.
		if (preWx*dWx + preWy*dWy) < 0.0 {
			e.wx *= slapKeep
			e.wy *= slapKeep
			if e.faceAlignCos > faceSlapVzAlignCos && e.vz > 0.0 {
				e.vz *= slapKeep
			}
		}
		if e.faceAlignCos > nearFlatFaceAlignCos && (preWz*dWz) < 0.0 {
			e.wz *= slapKeep
		}
	}
	if j > 0.0 && !e.ridgeBreakApplied && !e.keepUpright && floorNz > floorNormalThreshold {
		// Unstable ridge equilibrium breaking when a tilted free body balances directly above its contact.
		if e.onGround && e.isTilted && e.faceAlignCos < ridgeMaxAlignCos && (rx*rx+ry*ry) < ridgeLeverSqMax && (e.wx*e.wx+e.wy*e.wy) < ridgeTiltSqMax {
			e.wy += ridgeBreakKick
			e.ridgeBreakApplied = true
		}
	}
}

// frictionRotBaseScale returns the low-energy rotational attenuation factor for floor friction impulses.
func (e *Cinematic) frictionRotBaseScale(floorNz float64) float64 {
	if !e.keepUpright && floorNz > floorNormalThreshold {
		vHorizSq := e.vx*e.vx + e.vy*e.vy
		wTiltSq := e.wx*e.wx + e.wy*e.wy
		if vHorizSq < frictionLowEnergyHorizSq && wTiltSq < frictionLowEnergyTiltSq {
			if !e.isTilted || e.faceAlignCos > nearFlatFaceAlignCos {
				// Flat/near-flat sliding on floor at low energy: support polygon resists tipping torque.
				return 0.0
			} else if vHorizSq < lowEnergyGroundHorizSq {
				// Low-energy rolling/sliding on edge: attenuate friction-to-tipping conversion.
				return frictionEdgeRotScale
			}
		}
	}
	return 1.0
}

// frictionEffectiveScale returns the rotational denominator scale matching applyFrictionAngular for spin-up vs braking.
func frictionEffectiveScale(baseScale, wx, wy, wz, ux, uy, uz float64) float64 {
	if baseScale >= 1.0 {
		return 1.0
	}
	// If candidate impulse spins up tilt (or tilt is zero), keep denominator consistent with attenuated torque.
	if (wx*ux+wy*uy) >= 0.0 && wz*uz >= 0.0 {
		return baseScale
	}
	return 1.0
}

// applyFrictionAngular applies a tangential friction impulse to angular velocity, limiting tilt and yaw spin.
func (e *Cinematic) applyFrictionAngular(jt, frictionRotScale, utx, uty, utz float64) {
	dWx := jt * utx
	dWy := jt * uty
	dWz := jt * utz
	preTiltSq := e.wx*e.wx + e.wy*e.wy
	nwx := e.wx + dWx
	nwy := e.wy + dWy
	nwz := e.wz + dWz
	// Allow full braking of existing spin; attenuate only friction that would spin up tilt or yaw.
	if frictionRotScale < 1.0 {
		if (nwx*nwx + nwy*nwy) > preTiltSq {
			nwx = e.wx + dWx*frictionRotScale
			nwy = e.wy + dWy*frictionRotScale
		}
		if (nwz * nwz) > (e.wz * e.wz) {
			nwz = e.wz + dWz*frictionRotScale
		}
	}
	e.wx = math.Max(-maxTiltVel, math.Min(maxTiltVel, nwx))
	e.wy = math.Max(-maxTiltVel, math.Min(maxTiltVel, nwy))
	e.wz = math.Max(-maxYawVel, math.Min(maxYawVel, nwz))
}

// integrateGyroscopic advances angular velocity due to gyroscopic precession using a midpoint step.
// It projects explicit integration results back to the initial rotational energy when needed.
func (e *Cinematic) integrateGyroscopic() {
	lx, ly, lz := e.angularMomentumFor(e.wx, e.wy, e.wz)
	t0 := e.wx*lx + e.wy*ly + e.wz*lz
	if t0 <= gyroEnergyEpsilon {
		return
	}
	// Midpoint step for Euler's equation: dω/dt = -I_world^-1 * (ω × L).
	ax, ay, az := e.gyroscopicAccel(e.wx, e.wy, e.wz, lx, ly, lz)
	halfDt := 0.5 * e.dt
	mwx := e.wx + ax*halfDt
	mwy := e.wy + ay*halfDt
	mwz := e.wz + az*halfDt

	mlx, mly, mlz := e.angularMomentumFor(mwx, mwy, mwz)
	ax, ay, az = e.gyroscopicAccel(mwx, mwy, mwz, mlx, mly, mlz)

	nwx := e.wx + ax*e.dt
	nwy := e.wy + ay*e.dt
	nwz := e.wz + az*e.dt
	nlx, nly, nlz := e.angularMomentumFor(nwx, nwy, nwz)
	// Explicit steps can add numerical energy; project back onto the initial energy ellipsoid.
	t1 := nwx*nlx + nwy*nly + nwz*nlz
	if t1 > t0 && t1 > gyroEnergyEpsilon {
		scale := math.Sqrt(t0 / t1)
		nwx *= scale
		nwy *= scale
		nwz *= scale
	}
	e.wx, e.wy, e.wz = nwx, nwy, nwz
}

// gyroscopicAccel returns angular acceleration from the gyroscopic term using world-space angular velocity and momentum.
func (e *Cinematic) gyroscopicAccel(wx, wy, wz, lx, ly, lz float64) (float64, float64, float64) {
	gx := wy*lz - wz*ly
	gy := wz*lx - wx*lz
	gz := wx*ly - wy*lx
	return e.ApplyWorldInvInertia(-gx, -gy, -gz)
}
