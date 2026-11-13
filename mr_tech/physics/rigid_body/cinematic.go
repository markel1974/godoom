package rigid_body

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/physics/material"
)

const (
	airFriction = 0.99

	// minVelocity represents the minimum velocity threshold below which motion is considered negligible.
	minVelocity = 0.1

	// minThickness represents the minimum allowable thickness value to prevent calculations with near-zero dimensions.
	minThickness = 0.01

	// safetyMargin defines a proportional buffer value, typically used to ensure numerical stability in calculations.
	safetyMargin float64 = 0.90

	defaultPenetrationDepth = 0.7

	defaultPenetrationSlop = 0.02

	defaultPositionalPercent = 0.3
)

// Cinematic represents the physical state and 6-DoF rigid-body dynamics of an object in a simulation.
type Cinematic struct {
	mass    float64
	invMass float64
	invIx   float64
	invIy   float64
	invIz   float64
	invIxx  float64
	invIxy  float64
	invIxz  float64
	invIyy  float64
	invIyz  float64
	invIzz  float64
	ixx     float64
	ixy     float64
	ixz     float64
	iyy     float64
	iyz     float64
	izz     float64
	vx      float64
	vy      float64
	vz      float64
	ax      float64
	ay      float64
	az      float64
	wx      float64
	wy      float64
	wz      float64
	torqueX float64
	torqueY float64
	torqueZ float64
	vMin    float64
	//groundFriction float64
	airFriction float64
	//frictionActive float64
	airDamping float64
	//groundDamping float64
	//dampingActive  float64
	gForce float64
	//restitution       float64
	material          *material.PhysicsMaterial
	maxVelocitySq     float64
	sleepThresholdSq  float64
	dt                float64
	terminalZVelocity float64
	onGround          bool
	isTilted          bool
}

// NewCinematic initializes a new Cinematic instance with specified mass, restitution, ground friction, and gravitational force.
// Mass and restitution values are clamped to ensure physical validity. Returns a pointer to the initialized Cinematic.
func NewCinematic(dt float64, mass, restitution, groundFriction, gForce float64) *Cinematic {
	if restitution < 0.0 {
		restitution = 0.0
	} else if restitution > 1.0 {
		restitution = 1.0
	}
	if groundFriction <= 0.0 {
		groundFriction = 0.2
	}
	invMass := 0.0
	if mass <= 0.0 {
		mass = 0.0
	} else {
		invMass = 1.0 / mass
	}
	mat := material.NewPhysicsMaterial(restitution, groundFriction, material.CombineAverage, material.CombineAverage, dt)
	a := &Cinematic{
		gForce:           gForce,
		vMin:             minVelocity,
		sleepThresholdSq: minVelocity * minVelocity,
		mass:             mass,
		invMass:          invMass,
		material:         mat,
		onGround:         false,
	}
	a.UpdateInertia(32.0, 32.0, 32.0)
	a.SetOptions(dt, airFriction)
	a.SetOnGround(false)
	return a
}

// UpdateInertia computes the inverse principal moments of inertia for a solid cuboid of dimensions (w, h, d)
// and initializes the world-space inverse inertia tensor.
func (e *Cinematic) UpdateInertia(w, h, d float64) {
	if e.invMass == 0.0 || e.mass <= 0.0 {
		e.invIx, e.invIy, e.invIz = 0.0, 0.0, 0.0
		e.ixx, e.ixy, e.ixz = 0.0, 0.0, 0.0
		e.iyy, e.iyz, e.izz = 0.0, 0.0, 0.0
		e.invIxx, e.invIxy, e.invIxz = 0.0, 0.0, 0.0
		e.invIyy, e.invIyz, e.invIzz = 0.0, 0.0, 0.0
		return
	}
	if w < 32.0 {
		w = 32.0
	}
	if h < 32.0 {
		h = 32.0
	}
	if d < 32.0 {
		d = 32.0
	}
	effMass := math.Max(e.mass, 25.0)
	const rotScale = 8.0
	factor := (effMass * rotScale) / 12.0
	e.invIx = 1.0 / (factor * (h*h + d*d))
	e.invIy = 1.0 / (factor * (w*w + d*d))
	e.invIz = 1.0 / (factor * (w*w + h*h))
	// Initialize body-space inertia tensor.
	e.ixx = 1.0 / e.invIx
	e.ixy = 0.0
	e.ixz = 0.0
	e.iyy = 1.0 / e.invIy
	e.iyz = 0.0
	e.izz = 1.0 / e.invIz
	// Initialize world-space inverse inertia tensor.
	// Initially the body has identity orientation.
	e.invIxx = e.invIx
	e.invIxy = 0.0
	e.invIxz = 0.0
	e.invIyy = e.invIy
	e.invIyz = 0.0
	e.invIzz = e.invIz
}

// SyncWorldInertia updates the world-space inertia tensors:
// I_world = R * I_body * R^T
// I_world^-1 = R * I_body^-1 * R^T
func (e *Cinematic) SyncWorldInertia(a00, a01, a02, a10, a11, a12, a20, a21, a22 float64) {
	if e.invMass == 0.0 {
		e.ixx, e.ixy, e.ixz = 0.0, 0.0, 0.0
		e.iyy, e.iyz, e.izz = 0.0, 0.0, 0.0
		e.invIxx, e.invIxy, e.invIxz = 0.0, 0.0, 0.0
		e.invIyy, e.invIyz, e.invIzz = 0.0, 0.0, 0.0
		return
	}

	ix, iy, iz := e.invIx, e.invIy, e.invIz
	// Direct principal moments.
	bix := 1.0 / ix
	biy := 1.0 / iy
	biz := 1.0 / iz
	// World-space inertia tensor.
	e.ixx = bix*a00*a00 + biy*a10*a10 + biz*a20*a20
	e.ixy = bix*a00*a01 + biy*a10*a11 + biz*a20*a21
	e.ixz = bix*a00*a02 + biy*a10*a12 + biz*a20*a22
	e.iyy = bix*a01*a01 + biy*a11*a11 + biz*a21*a21
	e.iyz = bix*a01*a02 + biy*a11*a12 + biz*a21*a22
	e.izz = bix*a02*a02 + biy*a12*a12 + biz*a22*a22

	// World-space inverse inertia tensor.
	e.invIxx = ix*a00*a00 + iy*a10*a10 + iz*a20*a20
	e.invIxy = ix*a00*a01 + iy*a10*a11 + iz*a20*a21
	e.invIxz = ix*a00*a02 + iy*a10*a12 + iz*a20*a22
	e.invIyy = ix*a01*a01 + iy*a11*a11 + iz*a21*a21
	e.invIyz = ix*a01*a02 + iy*a11*a12 + iz*a21*a22
	e.invIzz = ix*a02*a02 + iy*a12*a12 + iz*a22*a22
}

// ApplyWorldInvInertia multiplies a world-space vector by the 3x3 world-space inverse inertia tensor I_world^-1.
func (e *Cinematic) ApplyWorldInvInertia(tx, ty, tz float64) (float64, float64, float64) {
	return e.invIxx*tx + e.invIxy*ty + e.invIxz*tz,
		e.invIxy*tx + e.invIyy*ty + e.invIyz*tz,
		e.invIxz*tx + e.invIyz*ty + e.invIzz*tz
}

// SetOptions configures the time step, ground friction, and air friction values for the cinematic simulation.
func (e *Cinematic) SetOptions(dt, aFriction float64) {
	e.dt = dt
	e.material.SetDt(dt)
	//e.groundDamping = math.Pow(e.groundFriction, e.dt)
	e.airFriction = aFriction
	e.airDamping = math.Pow(e.airFriction, e.dt)
	if e.airDamping >= 1.0 {
		// No atmospheric friction, terminal velocity would tend to infinity
		e.terminalZVelocity = -math.MaxFloat64
	} else {
		// Calculation of the integrator's asymptote
		e.terminalZVelocity = (-e.gForce * e.dt * e.airDamping) / (1.0 - e.airDamping)
	}
	maxVelocity := (minThickness * safetyMargin) / e.dt
	e.maxVelocitySq = maxVelocity * maxVelocity
	e.SetOnGround(e.onGround)
}

// Stop halts the entity's motion by setting its linear and angular velocity components to zero.
func (e *Cinematic) Stop() {
	e.ax = 0.0
	e.ay = 0.0
	e.az = 0.0
	e.vx = 0.0
	e.vy = 0.0
	e.vz = 0.0
	e.wx = 0.0
	e.wy = 0.0
	e.wz = 0.0
	e.torqueX = 0.0
	e.torqueY = 0.0
	e.torqueZ = 0.0
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
	e.wx = wx
	e.wy = wy
	e.wz = wz
}

// AddTorque applies a 3D torque vector (tx, ty, tz) in world space using the world-space inverse inertia tensor.
func (e *Cinematic) AddTorque(tx, ty, tz float64) {
	ax, ay, az := e.ApplyWorldInvInertia(tx, ty, tz)
	e.torqueX += ax
	e.torqueY += ay
	e.torqueZ += az
}

// AngularMomentum returns the world-space angular momentum L = I_world * omega.
func (e *Cinematic) AngularMomentum() (float64, float64, float64) {
	return e.ixx*e.wx + e.ixy*e.wy + e.ixz*e.wz, e.ixy*e.wx + e.iyy*e.wy + e.iyz*e.wz, e.ixz*e.wx + e.iyz*e.wy + e.izz*e.wz
}

// AddImpulseAtOffset applies an angular impulse Δω = I_world^-1 (r × J) generated by linear impulse (jx, jy, jz) at lever arm (rx, ry, rz).
func (e *Cinematic) AddImpulseAtOffset(jx, jy, jz, rx, ry, rz float64) {
	if e.invMass == 0.0 {
		return
	}

	// Angular impulse: ΔL = r × J
	tx := ry*jz - rz*jy
	ty := rz*jx - rx*jz
	tz := rx*jy - ry*jx

	// Angular velocity variation: Δω = I_world^-1 * ΔL
	dwx, dwy, dwz := e.ApplyWorldInvInertia(tx, ty, tz)

	e.wx += dwx
	e.wy += dwy
	e.wz += dwz

	const maxTiltVel = 2.5
	const maxYawVel = 4.5

	e.wx = math.Max(-maxTiltVel, math.Min(maxTiltVel, e.wx))
	e.wy = math.Max(-maxTiltVel, math.Min(maxTiltVel, e.wy))
	e.wz = math.Max(-maxYawVel, math.Min(maxYawVel, e.wz))
}

// IsRotating reports whether the object has any non-zero angular velocity.
func (e *Cinematic) IsRotating() bool {
	return e.wx != 0.0 || e.wy != 0.0 || e.wz != 0.0
}

// IsTilted reports whether the cinematic is tilted.
func (e *Cinematic) IsTilted() bool {
	return e.isTilted
}

// SetDt updates the time step (dt) and recalculates damping and velocity limits based on current friction values.
func (e *Cinematic) SetDt(dt float64) {
	e.SetOptions(dt, e.airFriction)
}

// SetOnGround sets the entity's ground state and adjusts active friction and damping values based on the state.
func (e *Cinematic) SetOnGround(onGround bool) {
	e.onGround = onGround
	if e.onGround {
		//e.frictionActive = e.groundFriction
		//e.dampingActive = e.groundDamping
		if e.vz < 0.0 {
			e.vz = 0.0
		}
	} else {
		//e.frictionActive = e.airFriction
		//e.dampingActive = e.airDamping
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
func (e *Cinematic) GetVz() float64 { return e.vz }

// SetVx sets the velocity along the X-axis for the Cinematic object.
func (e *Cinematic) SetVx(vx float64) {
	e.vx = vx
}

// SetVy sets the vertical velocity (vy) of the Cinematic object to the specified value.
func (e *Cinematic) SetVy(vy float64) {
	e.vy = vy
}

// SetVz sets the z-axis velocity (vz) for the Cinematic object.
func (e *Cinematic) SetVz(vz float64) { e.vz = vz }

// SetV sets the velocity components vx, vy, and vz for the cinematic entity.
func (e *Cinematic) SetV(vx, vy, vz float64) {
	e.vx = vx
	e.vy = vy
	e.vz = vz
}

// AddV increments the velocity components (vx, vy, vz) of the Cinematic object by the specified values.
func (e *Cinematic) AddV(vx, vy, vz float64) {
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

// GetRestitution returns the restitution coefficient of the cinematic object, representing its bounciness.
//func (e *Cinematic) GetRestitution() float64 {
//	return e.restitution
//}

// GetPhysicsMaterial returns the surface physical characteristics of the cinematic object.
func (e *Cinematic) GetPhysicsMaterial() *material.PhysicsMaterial {
	return e.material
}

// SetPhysicsMaterial sets the surface physical characteristics and updates internal restitution and ground friction.
func (e *Cinematic) SetPhysicsMaterial(m *material.PhysicsMaterial) {
	e.material = m
	//e.restitution = m.Restitution
	//e.groundFriction = m.Friction
	//if e.onGround {
	//	e.frictionActive = m.Friction
	//}
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
	moving := e.vx != 0 || e.vy != 0 || e.vz != 0 || e.wx != 0 || e.wy != 0 || e.wz != 0
	return moving
}

// AddForce applies a force to the object by modifying its acceleration components (ax, ay, az) using the force and inverse mass.
func (e *Cinematic) AddForce(fx, fy, fz float64) {
	e.ax += fx * e.invMass
	e.ay += fy * e.invMass
	e.az += fz * e.invMass
}

// AddForceAtPoint applies a linear force (fx, fy, fz) at world point (px, py, pz) and generates 3D torque τ = r × F on the OBB
// using exact Ray-OBB slab intersection and the world-space inverse inertia tensor I_world^-1 = R * I_body^-1 * R^T.
func (e *Cinematic) AddForceAtPoint(fx, fy, fz, px, py, pz float64, cx, cy, cz float64, hx, hy, hz float64, a00, a01, a02, a10, a11, a12, a20, a21, a22 float64) {
	e.AddForce(fx, fy, fz)
	if e.GetInvMass() == 0.0 {
		return
	}
	fMag := math.Sqrt(fx*fx + fy*fy + fz*fz)
	if fMag <= 1e-9 {
		return
	}
	invF := 1.0 / fMag

	//cx, cy, cz := bb.GetCenter()
	r0x, r0y, r0z := px-cx, py-cy, pz-cz

	//hx, hy, hz := bb.GetSizeCenter()
	//a00, a01, a02, a10, a11, a12, a20, a21, a22 := bb.GetAxes()

	// Project impact point and ray direction into OBB local frame
	plx := r0x*a00 + r0y*a01 + r0z*a02
	ply := r0x*a10 + r0y*a11 + r0z*a12
	plz := r0x*a20 + r0y*a21 + r0z*a22

	dlx := (fx*a00 + fy*a01 + fz*a02) * invF
	dly := (fx*a10 + fy*a11 + fz*a12) * invF
	dlz := (fx*a20 + fy*a21 + fz*a22) * invF

	// Shift to closest point on the ray to OBB center, clamped within the OBB interior
	tProj := plx*dlx + ply*dly + plz*dlz
	plx = math.Max(-hx*0.98, math.Min(hx*0.98, plx-tProj*dlx))
	ply = math.Max(-hy*0.98, math.Min(hy*0.98, ply-tProj*dly))
	plz = math.Max(-hz*0.98, math.Min(hz*0.98, plz-tProj*dlz))

	// March backward along -dl to find the exact entry point on the struck OBB face
	tBack := math.MaxFloat64
	hitAxis := 0
	hitSign := 1.0
	if math.Abs(dlx) > 1e-6 {
		s := -1.0
		if dlx < 0 {
			s = 1.0
		}
		if tb := (plx - s*hx) / dlx; tb >= 0.0 && tb < tBack {
			tBack = tb
			hitAxis = 0
			hitSign = s
		}
	}
	if math.Abs(dly) > 1e-6 {
		s := -1.0
		if dly < 0 {
			s = 1.0
		}
		if tb := (ply - s*hy) / dly; tb >= 0.0 && tb < tBack {
			tBack = tb
			hitAxis = 1
			hitSign = s
		}
	}
	if math.Abs(dlz) > 1e-6 {
		s := -1.0
		if dlz < 0 {
			s = 1.0
		}
		if tb := (plz - s*hz) / dlz; tb >= 0.0 && tb < tBack {
			tBack = tb
			hitAxis = 2
			hitSign = s
		}
	}
	if tBack < math.MaxFloat64 {
		plx -= tBack * dlx
		ply -= tBack * dly
		plz -= tBack * dlz
	}

	// Reconstruct world-space lever arm r from OBB center to surface entry point
	rx := a00*plx + a10*ply + a20*plz
	ry := a01*plx + a11*ply + a21*plz
	rz := (a02*plx + a12*ply + a22*plz) * 0.25

	// Direct moment of force: τ = r × F
	tx := ry*fz - rz*fy
	ty := rz*fx - rx*fz
	tz := rx*fy - ry*fx

	// Normal pressure moment on the struck OBB face: τ_n = r × F_n
	var fnX, fnY, fnZ float64
	switch hitAxis {
	case 0:
		fnX, fnY, fnZ = hitSign*a00, hitSign*a01, hitSign*a02
	case 1:
		fnX, fnY, fnZ = hitSign*a10, hitSign*a11, hitSign*a12
	default:
		fnX, fnY, fnZ = hitSign*a20, hitSign*a21, hitSign*a22
	}
	if fDotN := fx*fnX + fy*fnY + fz*fnZ; fDotN < 0.0 {
		fnFx, fnFy, fnFz := fDotN*fnX, fDotN*fnY, fDotN*fnZ
		tx += (ry*fnFz - rz*fnFy) * 0.25
		ty += (rz*fnFx - rx*fnFz) * 0.25
		tz += rx*fnFy - ry*fnFx
	}

	e.AddTorque(tx, ty, tz)
}

// Update advances the cinematic physics state using the supplied orientation axis components.
func (e *Cinematic) Update(a20, a21, a22 float64) {
	if e.invMass == 0.0 {
		return
	}

	tiltSq := a20*a20 + a21*a21
	e.isTilted = tiltSq > 1e-7 || a22 < 0.999999

	if e.isTilted {
		stiffness := 140.0
		damping := 16.0
		if !e.IsOnGround() {
			stiffness = 60.0
			damping = 10.0
		}
		errX := a21
		errY := -a20
		if a22 < 0.0 {
			lenXY := math.Hypot(errX, errY)
			if lenXY > 1e-6 {
				errX /= lenXY
				errY /= lenXY
			} else {
				errX = 1.0
			}
		}
		dt := e.GetDt()
		e.wx += (errX*stiffness - e.wx*damping) * dt
		e.wy += (errY*stiffness - e.wy*damping) * dt
	}

	const sleepEpsilon = 0.005
	const maxTiltVel = 2.5
	const maxYawVel = 4.5
	const angSleepEpsilon = 0.05

	// Angular momentum: L = I_world * omega
	lx, ly, lz := e.AngularMomentum()

	// Gyroscopic term: omega × L
	gx := e.wy*lz - e.wz*ly
	gy := e.wz*lx - e.wx*lz
	gz := e.wx*ly - e.wy*lx

	// Gyroscopic angular acceleration:
	// alpha = I_world^-1 * (tau - omega × L)
	ax, ay, az := e.ApplyWorldInvInertia(-gx, -gy, -gz)

	// Integrate angular acceleration
	e.wx += (e.torqueX + ax) * e.dt
	e.wy += (e.torqueY + ay) * e.dt
	e.wz += (e.torqueZ + az) * e.dt

	// Reset angular acceleration accumulators
	e.torqueX = 0.0
	e.torqueY = 0.0
	e.torqueZ = 0.0

	// Angular damping
	tiltDamping := e.airDamping
	yawDamping := e.airDamping
	groundDamping := e.material.GetGroundDamping()

	if e.onGround {
		tiltDamping = groundDamping
		yawDamping = math.Pow(0.02, e.dt)
	}

	e.wx *= tiltDamping
	e.wy *= tiltDamping
	e.wz *= yawDamping

	// Angular velocity limits
	e.wx = math.Max(-maxTiltVel, math.Min(maxTiltVel, e.wx))
	e.wy = math.Max(-maxTiltVel, math.Min(maxTiltVel, e.wy))
	e.wz = math.Max(-maxYawVel, math.Min(maxYawVel, e.wz))

	// Angular sleep
	if math.Abs(e.wx) < angSleepEpsilon {
		e.wx = 0.0
	}
	if math.Abs(e.wy) < angSleepEpsilon {
		e.wy = 0.0
	}
	if math.Abs(e.wz) < angSleepEpsilon {
		e.wz = 0.0
	}

	e.vx += e.ax * e.dt
	e.vy += e.ay * e.dt

	if e.onGround {
		if e.vz < 0.0 {
			e.vz = 0.0
		}
		if e.az > 0.0 {
			e.vz += e.az * e.dt
		}
	} else {
		e.vz += (e.az - e.gForce) * e.dt
	}

	// Reset linear acceleration accumulators
	e.ax = 0.0
	e.ay = 0.0
	e.az = 0.0

	// Frame displacement
	dx := e.vx * e.dt
	dy := e.vy * e.dt
	dz := e.vz * e.dt

	// Sleep check
	if math.Abs(dx) < sleepEpsilon &&
		math.Abs(dy) < sleepEpsilon &&
		math.Abs(dz) < sleepEpsilon {

		e.vx = 0.0
		e.vy = 0.0

		if e.onGround {
			e.vz = 0.0
		}
		return
	}

	// Linear damping
	e.vx *= groundDamping
	e.vy *= groundDamping
	e.vz *= e.airDamping

	// Planar sleep
	if math.Abs(e.vx) < e.vMin {
		e.vx = 0.0
	}
	if math.Abs(e.vy) < e.vMin {
		e.vy = 0.0
	}

	if e.onGround && e.vz <= 0.0 {
		e.vz = 0.0
	}

	// Terminal velocity clamping
	if e.vz < e.terminalZVelocity {
		e.vz = e.terminalZVelocity
	}
}

// ResolveImpact resolves a 6-DoF rigid-body collision between two Cinematic objects at contact lever arms (r1x, r1y, r1z)
// and (r2x, r2y, r2z), applying normal and Coulomb tangential impulses to both linear velocity v and angular velocity ω
// using the explicit contact surface material properties (matOther).
func (e *Cinematic) ResolveImpact(e2 *Cinematic, nx, ny, nz float64, cx1, cy1, cz1, ex1, ey1, ez1, cx2, cy2, cz2, ex2, ey2, ez2 float64) (float64, float64, float64, bool) {
	var r1x, r1y, r1z float64
	var r2x, r2y, r2z float64

	if e2.GetInvMass() > 0.0 {
		minX := math.Max(cx1-ex1, cx2-ex2)
		maxX := math.Min(cx1+ex1, cx2+ex2)
		minY := math.Max(cy1-ey1, cy2-ey2)
		maxY := math.Min(cy1+ey1, cy2+ey2)
		minZ := math.Max(cz1-ez1, cz2-ez2)
		maxZ := math.Min(cz1+ez1, cz2+ez2)

		px := math.Max(cx1-ex1, math.Min(cx1+ex1, (minX+maxX)*0.5))
		py := math.Max(cy1-ey1, math.Min(cy1+ey1, (minY+maxY)*0.5))
		pz := math.Max(cz1-ez1, math.Min(cz1+ez1, (minZ+maxZ)*0.5))

		r1x, r1y, r1z = px-cx1, py-cy1, (pz-cz1)*0.25
		r2x, r2y, r2z = px-cx2, py-cy2, (pz-cz2)*0.25
	} else if math.Abs(nz) < 0.7 {
		// Impact against static wall/obstacle: contact point on OBB perimeter along -n
		r1x = -nx * ex1
		r1y = -ny * ey1
		r1z = -nz * ez1 * 0.25
	}

	// safe normalization (anti-explosion)
	nLen := math.Sqrt(nx*nx + ny*ny + nz*nz)
	if nLen > 0.0 {
		nx /= nLen
		ny /= nLen
		nz /= nLen
	} else {
		return 0, 0, 0, false // null vector, cannot resolve
	}

	// relative velocity at contact point: v_p = v + ω × r
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

	// if bodies are separating at the contact point, no collision to resolve
	if vRelDotN > 0.0 {
		return 0, 0, 0, false
	}

	invMass1 := e.invMass
	invMass2 := e2.invMass
	if e2.onGround && nz > defaultPenetrationDepth {
		invMass2 = 0.0
	} else if e.onGround && nz < -defaultPenetrationDepth {
		invMass1 = 0.0
	}

	var u1nx, u1ny, u1nz, rotMass1N float64
	if invMass1 > 0.0 {
		c1nx := r1y*nz - r1z*ny
		c1ny := r1z*nx - r1x*nz
		c1nz := r1x*ny - r1y*nx
		u1nx, u1ny, u1nz = e.ApplyWorldInvInertia(c1nx, c1ny, c1nz)
		rotMass1N = c1nx*u1nx + c1ny*u1ny + c1nz*u1nz
	}

	var u2nx, u2ny, u2nz, rotMass2N float64
	if invMass2 > 0.0 {
		c2nx := r2y*nz - r2z*ny
		c2ny := r2z*nx - r2x*nz
		c2nz := r2x*ny - r2y*nx
		u2nx, u2ny, u2nz = e2.ApplyWorldInvInertia(c2nx, c2ny, c2nz)
		rotMass2N = c2nx*u2nx + c2ny*u2ny + c2nz*u2nz
	}

	effMassN := invMass1 + invMass2 + rotMass1N + rotMass2N
	if effMassN <= 0.0 {
		return 0, 0, 0, false
	}

	// restitution combining with threshold (micro-bounce slop)
	actualRestitution := e.material.CombineRestitution(e2.material)
	if math.Abs(actualRestitution) > 1e-6 {
		restitutionSlop := 2.0 * math.Max(e.gForce, e2.gForce) * e.dt
		if restitutionSlop < 2.0 {
			restitutionSlop = 2.0
		}
		if math.Abs(vRelDotN) < restitutionSlop || (e.onGround && nz > defaultPenetrationDepth) || (e2.onGround && nz < -defaultPenetrationDepth) {
			actualRestitution = 0.0
		}
	} else {
		actualRestitution = 0.0
	}

	// normal impulse including rotational inertia denominator K_n
	impulse := -(1.0 + actualRestitution) * vRelDotN
	j := impulse / effMassN

	const maxTiltVel = 2.5
	const maxYawVel = 4.5

	// apply normal impulse to linear and angular velocities
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
		e.wx = math.Max(-maxTiltVel, math.Min(maxTiltVel, e.wx+j*u1nx))
		e.wy = math.Max(-maxTiltVel, math.Min(maxTiltVel, e.wy+j*u1ny))
		e.wz = math.Max(-maxYawVel, math.Min(maxYawVel, e.wz+j*u1nz))
	}
	if invMass2 > 0.0 {
		e2.wx = math.Max(-maxTiltVel, math.Min(maxTiltVel, e2.wx-j*u2nx))
		e2.wy = math.Max(-maxTiltVel, math.Min(maxTiltVel, e2.wy-j*u2ny))
		e2.wz = math.Max(-maxYawVel, math.Min(maxYawVel, e2.wz-j*u2nz))
	}

	// tangential impulse (Coulomb friction)
	// recalculate relative velocity at the contact point after normal impulse
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

	// find tangent vector
	tx := vrx - (vRelDotNPost * nx)
	ty := vry - (vRelDotNPost * ny)
	tz := vrz - (vRelDotNPost * nz)

	if tLen := math.Sqrt(tx*tx + ty*ty + tz*tz); tLen > 1e-8 {
		tx /= tLen
		ty /= tLen
		tz /= tLen

		var u1tx, u1ty, u1tz, rotMass1T float64
		if invMass1 > 0.0 {
			c1tx := r1y*tz - r1z*ty
			c1ty := r1z*tx - r1x*tz
			c1tz := r1x*ty - r1y*tx
			u1tx, u1ty, u1tz = e.ApplyWorldInvInertia(c1tx, c1ty, c1tz)
			rotMass1T = c1tx*u1tx + c1ty*u1ty + c1tz*u1tz
		}

		var u2tx, u2ty, u2tz, rotMass2T float64
		if invMass2 > 0.0 {
			c2tx := r2y*tz - r2z*ty
			c2ty := r2z*tx - r2x*tz
			c2tz := r2x*ty - r2y*tx
			u2tx, u2ty, u2tz = e2.ApplyWorldInvInertia(c2tx, c2ty, c2tz)
			rotMass2T = c2tx*u2tx + c2ty*u2ty + c2tz*u2tz
		}

		effMassT := invMass1 + invMass2 + rotMass1T + rotMass2T
		if effMassT > 0.0 {
			vRelDotT := vrx*tx + vry*ty + vrz*tz
			jt := -vRelDotT / effMassT

			// friction mixing: explicit contact material properties
			mu := e.material.CombineFriction(e2.material)
			maxFriction := j * mu

			// clamp tangential impulse within coulomb cone
			if math.Abs(jt) > maxFriction {
				jt = math.Copysign(maxFriction, jt)
			}

			// apply friction impulse to linear and angular velocities
			jtx := jt * tx
			jty := jt * ty
			jtz := jt * tz
			e.vx += jtx * invMass1
			e.vy += jty * invMass1
			e.vz += jtz * invMass1
			e2.vx -= jtx * invMass2
			e2.vy -= jty * invMass2
			e2.vz -= jtz * invMass2
			if invMass1 > 0.0 {
				e.wx = math.Max(-maxTiltVel, math.Min(maxTiltVel, e.wx+jt*u1tx))
				e.wy = math.Max(-maxTiltVel, math.Min(maxTiltVel, e.wy+jt*u1ty))
				e.wz = math.Max(-maxYawVel, math.Min(maxYawVel, e.wz+jt*u1tz))
			}
			if invMass2 > 0.0 {
				e2.wx = math.Max(-maxTiltVel, math.Min(maxTiltVel, e2.wx-jt*u2tx))
				e2.wy = math.Max(-maxTiltVel, math.Min(maxTiltVel, e2.wy-jt*u2ty))
				e2.wz = math.Max(-maxYawVel, math.Min(maxYawVel, e2.wz-jt*u2tz))
			}
			jx += jtx
			jy += jty
			jz += jtz
		}
	}

	return jx, jy, jz, true
}

// ResolveElasticPenetration computes positional correction for overlapping cinematic bodies along a collision normal.
// It returns each body's correction amount and whether a correction should be applied.
func (e *Cinematic) ResolveElasticPenetration(other *Cinematic, otherFloor bool, nx, ny, nz float64, penetration float64) (float64, float64, bool) {
	depth := penetration - defaultPenetrationSlop
	if depth <= 0.0 {
		return 0, 0, false
	}
	invMass1 := e.GetInvMass()
	invMass2 := other.GetInvMass()
	// Se il corpo di appoggio poggia a terra lungo la verticale, non può scendere verso il basso (massa infinita -> invMass = 0)
	if (otherFloor || other.IsOnGround()) && nz > defaultPenetrationDepth {
		invMass2 = 0.0
	} else if e.IsOnGround() && nz < -defaultPenetrationDepth {
		invMass1 = 0.0
	}
	invMassSum := invMass1 + invMass2
	if invMassSum <= 0.0 {
		return 0, 0, false
	}
	correction := depth
	p1 := correction * (invMass1 / invMassSum)
	p2 := correction * (invMass2 / invMassSum)

	if nz > defaultPenetrationDepth && (otherFloor || other.IsOnGround()) {
		if e.vz < 0.0 {
			e.vz = 0.0
		}
	} else if nz < -defaultPenetrationDepth && e.IsOnGround() {
		if other.vz < 0.0 {
			other.vz = 0.0
		}
	}

	return p1, p2, true
}

// ResolveInelasticPenetration computes positional correction for an inelastic collision with static geometry.
func (e *Cinematic) ResolveInelasticPenetration(nz float64, pen float64) (float64, bool) {
	pen -= defaultPenetrationSlop
	if pen <= 0.0 {
		return 0, false
	}
	if nz > defaultPenetrationDepth {
		if e.vz < 0.0 {
			e.vz = 0.0
		}
	}
	return pen, true
}
