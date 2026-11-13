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

	maxTiltVel = 2.5

	maxYawVel = 4.5

	sleepEpsilon = 0.005

	angSleepEpsilon = 0.08

	rotScale = 1.0

	defaultInertia = 32.0
)

// Cinematic represents the physical state and 6-DoF rigid-body dynamics of an object in a simulation.
type Cinematic struct {
	mass    float64
	invMass float64
	// World-space inverse inertia tensor.
	invIx  float64
	invIy  float64
	invIz  float64
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
	// Body-space inertia tensor.
	bodyIxx, bodyIxy, bodyIxz float64
	bodyIyy, bodyIyz, bodyIzz float64

	// Body-space inverse inertia tensor.
	bodyInvIxx, bodyInvIxy, bodyInvIxz float64
	bodyInvIyy, bodyInvIyz, bodyInvIzz float64
	vx                                 float64
	vy                                 float64
	vz                                 float64
	ax                                 float64
	ay                                 float64
	az                                 float64
	wx                                 float64
	wy                                 float64
	wz                                 float64
	torqueX                            float64
	torqueY                            float64
	torqueZ                            float64
	vMin                               float64
	airFriction                        float64
	airDamping                         float64
	gForce                             float64
	material                           *PhysicsMaterial
	maxVelocitySq                      float64
	sleepThresholdSq                   float64
	dt                                 float64
	terminalZVelocity                  float64
	onGround                           bool
	isTilted                           bool
	keepUpright                        bool
	groundYawDamping                   float64
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
		keepUpright:      false, //true //MODIFICATO,true,
	}

	a.UpdateInertia(defaultInertia, defaultInertia, defaultInertia)
	a.SetDt(dt)
	a.SetOnGround(false)
	return a
}

// SetKeepUpright enables or disables the automatic upright PD stabilizer during Update.
func (e *Cinematic) SetKeepUpright(keepUpright bool) {
	e.keepUpright = keepUpright
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

	factor := math.Max(e.mass, 25.0) * rotScale

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

	// Reject degenerate or invalid inertia tensors.
	if det <= 0.0 || math.IsNaN(det) || math.IsInf(det, 0) {
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

	e.bodyIxx, e.bodyIxy, e.bodyIxz = a, b, c
	e.bodyIyy, e.bodyIyz, e.bodyIzz = d, f, g

	e.bodyInvIxx, e.bodyInvIxy, e.bodyInvIxz = invA, invB, invC
	e.bodyInvIyy, e.bodyInvIyz, e.bodyInvIzz = invD, invF, invG

	return true
}

// UpdateInertia computes the inverse principal moments of inertia for a solid cuboid of dimensions (w, h, d)
// and initializes the world-space inverse inertia tensor.
func (e *Cinematic) UpdateInertia(w, h, d float64) {
	if e.invMass == 0.0 || e.mass <= 0.0 {
		e.bodyIxx, e.bodyIxy, e.bodyIxz = 0, 0, 0
		e.bodyIyy, e.bodyIyz, e.bodyIzz = 0, 0, 0

		e.bodyInvIxx, e.bodyInvIxy, e.bodyInvIxz = 0, 0, 0
		e.bodyInvIyy, e.bodyInvIyz, e.bodyInvIzz = 0, 0, 0

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
	factor := 1.0 / 12.0

	e.SetInertiaTensor((h*h+d*d)*factor, 0, 0, (w*w+d*d)*factor, 0, (w*w+h*h)*factor)

	// Initially identity orientation.
	e.SyncWorldInertia(1, 0, 0, 0, 1, 0, 0, 0, 1)
}

// SyncWorldInertia updates the world-space inertia and inverse inertia tensors from the local orientation axes
// expressed in world space: row 0 = local X (a00,a01,a02), row 1 = local Y (a10,a11,a12), row 2 = local Z (a20,a21,a22).
// It clears both tensors when the body has zero inverse mass.
func (e *Cinematic) SyncWorldInertia(
	a00, a01, a02,
	a10, a11, a12,
	a20, a21, a22 float64,
) {
	if e.invMass == 0.0 {
		e.ixx, e.ixy, e.ixz = 0, 0, 0
		e.iyy, e.iyz, e.izz = 0, 0, 0

		e.invIxx, e.invIxy, e.invIxz = 0, 0, 0
		e.invIyy, e.invIyz, e.invIzz = 0, 0, 0
		return
	}

	r := [3][3]float64{{a00, a01, a02}, {a10, a11, a12}, {a20, a21, a22}}

	body := [3][3]float64{
		{e.bodyIxx, e.bodyIxy, e.bodyIxz},
		{e.bodyIxy, e.bodyIyy, e.bodyIyz},
		{e.bodyIxz, e.bodyIyz, e.bodyIzz},
	}

	bodyInv := [3][3]float64{
		{e.bodyInvIxx, e.bodyInvIxy, e.bodyInvIxz},
		{e.bodyInvIxy, e.bodyInvIyy, e.bodyInvIyz},
		{e.bodyInvIxz, e.bodyInvIyz, e.bodyInvIzz},
	}

	var world, worldInv [3][3]float64

	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			for k := 0; k < 3; k++ {
				for l := 0; l < 3; l++ {
					world[i][j] += r[k][i] * body[k][l] * r[l][j]
					worldInv[i][j] += r[k][i] * bodyInv[k][l] * r[l][j]
				}
			}
		}
	}

	e.ixx, e.ixy, e.ixz = world[0][0], world[0][1], world[0][2]
	e.iyy, e.iyz, e.izz = world[1][1], world[1][2], world[2][2]

	e.invIxx, e.invIxy, e.invIxz = worldInv[0][0], worldInv[0][1], worldInv[0][2]
	e.invIyy, e.invIyz, e.invIzz = worldInv[1][1], worldInv[1][2], worldInv[2][2]
}

// ApplyWorldInvInertia multiplies a world-space vector by the 3x3 world-space inverse inertia tensor I_world^-1.
func (e *Cinematic) ApplyWorldInvInertia(tx, ty, tz float64) (float64, float64, float64) {
	return e.invIxx*tx + e.invIxy*ty + e.invIxz*tz,
		e.invIxy*tx + e.invIyy*ty + e.invIyz*tz,
		e.invIxz*tx + e.invIyz*ty + e.invIzz*tz
}

// SetDt configures the time step, ground friction, and air friction values for the cinematic simulation.
func (e *Cinematic) SetDt(dt float64) {
	e.dt = dt
	e.groundYawDamping = math.Pow(0.02, e.dt)
	e.material.SetDt(e.dt)
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
	e.wx = wx
	e.wy = wy
	e.wz = wz
}

// AngularMomentum returns the world-space angular momentum L = I_world * omega.
func (e *Cinematic) AngularMomentum() (float64, float64, float64) {
	return e.ixx*e.wx + e.ixy*e.wy + e.ixz*e.wz,
		e.ixy*e.wx + e.iyy*e.wy + e.iyz*e.wz,
		e.ixz*e.wx + e.iyz*e.wy + e.izz*e.wz
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
	e.vx = vx
}

// SetVy sets the vertical velocity (vy) of the Cinematic object to the specified value.
func (e *Cinematic) SetVy(vy float64) {
	e.vy = vy
}

// SetVz sets the z-axis velocity (vz) for the Cinematic object.
func (e *Cinematic) SetVz(vz float64) {
	e.vz = vz
}

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
	e.ax += fx * e.invMass
	e.ay += fy * e.invMass
	e.az += fz * e.invMass
}

// AddForceAtPoint applies a world-space force at a point and accumulates the resulting force and torque.
func (e *Cinematic) AddForceAtPoint(fx, fy, fz float64, px, py, pz float64, cx, cy, cz float64) {
	if e.invMass == 0.0 {
		return
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

	if e.keepUpright {
		tiltSq := ax0*ax0 + ax1*ax1
		e.isTilted = tiltSq > 1e-7 || ax2 < 0.999999
	}

	if e.isTilted && e.keepUpright {
		stiffness := 140.0
		damping := 16.0
		if !e.onGround {
			stiffness = 60.0
			damping = 10.0
		}
		errX := ax1
		errY := -ax0
		if ax2 < 0.0 {
			lenXY := math.Hypot(errX, errY)
			if lenXY > 1e-6 {
				errX /= lenXY
				errY /= lenXY
			} else {
				errX = 1.0
			}
		}
		e.wx += (errX*stiffness - e.wx*damping) * e.dt
		e.wy += (errY*stiffness - e.wy*damping) * e.dt
	}

	// Angular momentum: L = I_world * omega
	lx, ly, lz := e.AngularMomentum()

	// Gyroscopic term: omega × L
	gx := e.wy*lz - e.wz*ly
	gy := e.wz*lx - e.wx*lz
	gz := e.wx*ly - e.wy*lx

	// Angular acceleration: alpha = I_world^-1 * (tau - omega × L)
	ax, ay, az := e.ApplyWorldInvInertia(e.torqueX-gx, e.torqueY-gy, e.torqueZ-gz)
	// Integrate angular velocity.
	e.wx += ax * e.dt
	e.wy += ay * e.dt
	e.wz += az * e.dt
	// Reset torque accumulators.
	e.torqueX = 0.0
	e.torqueY = 0.0
	e.torqueZ = 0.0

	// Angular damping.
	tiltDamping := e.airDamping
	yawDamping := e.airDamping
	groundDamping := e.material.GetGroundDamping()
	if e.onGround {
		yawDamping = e.groundYawDamping
		if !e.isTilted || e.keepUpright {
			if e.keepUpright {
				tiltDamping = groundDamping
			} else {
				// Flat face resting contact strongly damps residual tilt and yaw micro-vibrations.
				faceDamping := e.groundYawDamping * e.groundYawDamping
				tiltDamping = faceDamping
				yawDamping = faceDamping
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

	// Angular sleep (only when body is face-aligned and not tilted on an edge).
	if !e.isTilted {
		if math.Abs(e.wx) < angSleepEpsilon {
			e.wx = 0.0
		}
		if math.Abs(e.wy) < angSleepEpsilon {
			e.wy = 0.0
		}
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
	if groundedFlat {
		e.vx *= groundDamping
		e.vy *= groundDamping
		e.vz *= e.airDamping
		if math.Abs(e.vx) < e.vMin {
			e.vx = 0.0
		}
		if math.Abs(e.vy) < e.vMin {
			e.vy = 0.0
		}
		if e.vz <= 0.0 {
			e.vz = 0.0
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
	// Safe normalization before computing lever arms or normal thresholds.
	nLen := math.Sqrt(nx*nx + ny*ny + nz*nz)
	if nLen <= 0 {
		return //0, 0, 0, false
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

	// If bodies are separating at the contact point, no collision to resolve.
	if vRelDotN > 0.0 {
		return
	}

	invMass1 := e.invMass
	invMass2 := e2.invMass
	if e2.onGround && nz > floorNormalThreshold {
		invMass2 = 0.0
	} else if e.onGround && nz < -floorNormalThreshold {
		invMass1 = 0.0
	}

	var u1nx, u1ny, u1nz, rotMass1N float64
	if invMass1 > 0.0 {
		u1nx, u1ny, u1nz, rotMass1N = e.rotationalMass(r1x, r1y, r1z, nx, ny, nz)
	}

	var u2nx, u2ny, u2nz, rotMass2N float64
	if invMass2 > 0.0 {
		u2nx, u2ny, u2nz, rotMass2N = e2.rotationalMass(r2x, r2y, r2z, nx, ny, nz)
	}

	effMassN := invMass1 + invMass2 + rotMass1N + rotMass2N
	if effMassN <= 0.0 {
		return
	}

	// Restitution combining with threshold (micro-bounce slop).
	actualRestitution := e.material.CombineRestitution(e2.material)
	if math.Abs(actualRestitution) > 1e-6 {
		restitutionSlop := 2.0 * math.Max(e.gForce, e2.gForce) * e.dt
		if restitutionSlop < 2.0 {
			restitutionSlop = 2.0
		}
		if math.Abs(vRelDotN) < restitutionSlop || (e.onGround && nz > 0.25) || (e2.onGround && nz < -0.25) {
			actualRestitution = 0.0
		}
	} else {
		actualRestitution = 0.0
	}

	// Normal impulse including rotational inertia denominator K_n.
	impulse := -(1.0 + actualRestitution) * vRelDotN
	j := impulse / effMassN

	// Apply normal impulse to linear and angular velocities.
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
		// Unstable ridge equilibrium breaking when a tilted free body balances directly above its contact.
		if e.onGround && e.isTilted && !e.keepUpright && nz > floorNormalThreshold {
			if (r1x*r1x+r1y*r1y) < 0.04 && (e.wx*e.wx+e.wy*e.wy) < 1e-4 {
				e.wy += 0.1
			}
		}
	}
	if invMass2 > 0.0 {
		e2.wx = math.Max(-maxTiltVel, math.Min(maxTiltVel, e2.wx-j*u2nx))
		e2.wy = math.Max(-maxTiltVel, math.Min(maxTiltVel, e2.wy-j*u2ny))
		e2.wz = math.Max(-maxYawVel, math.Min(maxYawVel, e2.wz-j*u2nz))
		if e2.onGround && e2.isTilted && !e2.keepUpright && nz < -floorNormalThreshold {
			if (r2x*r2x+r2y*r2y) < 0.04 && (e2.wx*e2.wx+e2.wy*e2.wy) < 1e-4 {
				e2.wy += 0.1
			}
		}
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
	if tLen <= 1e-8 {
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

	effMassT := invMass1 + invMass2 + rotMass1T + rotMass2T
	if effMassT <= 0.0 {
		return
	}

	vRelDotT := vrx*tx + vry*ty + vrz*tz
	jt := -vRelDotT / effMassT
	// Friction mixing: explicit contact material properties.
	mu := e.material.CombineFriction(e2.material)
	maxFriction := j * mu
	// Clamp tangential impulse within Coulomb cone.
	if math.Abs(jt) > maxFriction {
		jt = math.Copysign(maxFriction, jt)
	}
	// Apply friction impulse to linear and angular velocities.
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
	if (otherFloor || other.IsOnGround()) && nz > floorNormalThreshold {
		invMass2 = 0.0
	} else if e.IsOnGround() && nz < -floorNormalThreshold {
		invMass1 = 0.0
	}
	invMassSum := invMass1 + invMass2
	if invMassSum <= 0.0 {
		return 0, 0, false
	}
	correction := depth
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

// ResolveInelasticPenetration computes positional correction for an inelastic collision with static geometry.
func (e *Cinematic) ResolveInelasticPenetration(nz float64, pen float64) (float64, bool) {
	pen -= defaultPenetrationSlop
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

func (e *Cinematic) rotationalMass(rx, ry, rz float64, dx, dy, dz float64) (float64, float64, float64, float64) {
	cx := ry*dz - rz*dy
	cy := rz*dx - rx*dz
	cz := rx*dy - ry*dx
	ux, uy, uz := e.ApplyWorldInvInertia(cx, cy, cz)
	mass := cx*ux + cy*uy + cz*uz
	return ux, uy, uz, mass
}
