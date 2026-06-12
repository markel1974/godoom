package physics

import "math"

/*
3. I Limiti Matematici dell'Implementazione Attuale
L'architettura è robusta, ma presenta un limite matematico fondamentale
se l'obiettivo (come avevi accennato prima) è un motore fisico puramente generico: Assenza di Dinamica Rotazionale (Torque e Inerzia)
Il modulo Cinematic descrive il moto traslazionale di un punto materiale, non la dinamica di un corpo rigido vero e proprio.
Manca totalmente il concetto di Momento d'Inerzia (tensore d'inerzia $I$), Velocità Angolare ($\omega$) e Momento Torcente ($\tau$).
Attualmente, quando calcoli gli impulsi in ResolveImpact, assumi che la forza colpisca sempre esattamente il centro di massa dell'oggetto.
In un motore fisico generico, l'impulso applicato a una distanza $\vec{r}$ dal centro di massa deve generare
una rotazione $\Delta\omega = I^{-1} (\vec{r} \times \vec{J})$.Essendo il tuo sistema basato su AABB
(che per definizione non possono ruotare, altrimenti diventerebbero OBB - Oriented Bounding Boxes),
questo non è un bug, ma un vincolo intrinseco del design.Conclusione:Dal punto di vista matematico, le equazioni lineari,
i calcoli tensoriali sui vettori normali e gli integratori esponenziali sono implementati alla perfezione.
Architetturalmente l'ECS-lite (Cinematic + BoundingBox) ti garantisce stabilità spaziale.
Se il motore è destinato a muovere entità orientate assialmente (come giocatori FPS, mostri, proiettili o veicoli hover),
il core matematico è completo e corretto. Se invece in futuro vorrai simulare scatole che ruzzolano o ragdoll,
l'architettura dovrà espandere il modulo Cinematic per includere i quaternioni e la matrice del tensore d'inerzia.
*/

const (
	airFriction = 0.99

	// vMin represents the minimum velocity threshold below which motion is considered negligible.
	vMin = 0.1

	// minThickness represents the minimum allowable thickness value to prevent calculations with near-zero dimensions.
	minThickness = 0.01

	// safetyMargin defines a proportional buffer value, typically used to ensure numerical stability in calculations.
	safetyMargin float64 = 0.90
)

// Cinematic represents the physical state and behavior of an object in a simulation with forces and motion dynamics.
type Cinematic struct {
	mass              float64
	invMass           float64
	vx                float64
	vy                float64
	vz                float64
	ax                float64
	ay                float64
	az                float64
	vMin              float64
	groundFriction    float64
	airFriction       float64
	frictionActive    float64
	airDamping        float64
	groundDamping     float64
	dampingActive     float64
	gForce            float64
	restitution       float64
	maxVelocitySq     float64
	sleepThresholdSq  float64
	dt                float64
	terminalZVelocity float64
	onGround          bool
}

// NewCinematic initializes a new Cinematic instance with specified mass, restitution, ground friction, and gravitational force.
// Mass and restitution values are clamped to ensure physical validity. Returns a pointer to the initialized Cinematic.
func NewCinematic(dt float64, mass, restitution, groundFriction, gForce float64) *Cinematic {
	if restitution <= 0.0 {
		restitution = 0.2
	}
	invMass := 0.0
	if mass <= 0.0 {
		mass = 0.0
	} else {
		invMass = 1.0 / mass
	}
	a := &Cinematic{
		gForce:           gForce,
		vMin:             vMin,
		sleepThresholdSq: vMin * vMin,
		mass:             mass,
		invMass:          invMass,
		restitution:      restitution,
		onGround:         false,
	}
	a.SetOptions(dt, groundFriction, airFriction)
	a.SetOnGround(false)
	return a
}

// SetOptions configures the time step, ground friction, and air friction values for the cinematic simulation.
func (e *Cinematic) SetOptions(dt, gFriction, aFriction float64) {
	e.dt = dt
	e.groundFriction = gFriction
	e.groundDamping = math.Pow(e.groundFriction, e.dt)
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

// Stop halts the entity's motion by setting its velocity components (vx, vy, vz) to zero.
func (e *Cinematic) Stop() {
	e.vx = 0.0
	e.vy = 0.0
	e.vz = 0.0
}

// SetDt updates the time step (dt) and recalculates damping and velocity limits based on current friction values.
func (e *Cinematic) SetDt(dt float64) {
	e.SetOptions(dt, e.groundFriction, e.airFriction)
}

// SetOnGround sets the entity's ground state and adjusts active friction and damping values based on the state.
func (e *Cinematic) SetOnGround(onGround bool) {
	e.onGround = onGround
	if e.onGround {
		e.frictionActive = e.groundFriction
		e.dampingActive = e.groundDamping
		//e.vz = 0.0
	} else {
		e.frictionActive = e.airFriction
		e.dampingActive = e.airDamping
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
func (e *Cinematic) GetRestitution() float64 {
	return e.restitution
}

// GetGForce returns the value of the gravitational force acting on the Cinematic entity.
func (e *Cinematic) GetGForce() float64 {
	return e.gForce
}

// GetVelocity returns the current velocity components (vx, vy, vz) of the cinematic object.
func (e *Cinematic) GetVelocity() (float64, float64, float64) {
	return e.vx, e.vy, e.vz
}

// GetDisplacement calculates and returns the displacement in the x, y, and z directions based on velocity and time step.
func (e *Cinematic) GetDisplacement() (float64, float64, float64) {
	return e.vx * e.dt, e.vy * e.dt, e.vz * e.dt
}

// IsMoving returns true if the object has any non-zero velocity components (vx, vy, or vz), indicating it is in motion.
func (e *Cinematic) IsMoving() bool {
	return e.vx != 0 || e.vy != 0 || e.vz != 0
}

// AddForce applies a force to the object by modifying its acceleration components (ax, ay, az) using the force and inverse mass.
func (e *Cinematic) AddForce(fx, fy, fz float64) {
	e.ax += fx * e.invMass
	e.ay += fy * e.invMass
	e.az += fz * e.invMass
}

// Update updates the state of the Cinematic object by integrating acceleration, velocity, and damping forces over time.
func (e *Cinematic) Update() {
	const sleepEpsilon = 0.005
	e.vx += e.ax * e.dt
	e.vy += e.ay * e.dt
	gForce := e.gForce
	if e.onGround {
		gForce = 0.0
	}
	e.vz += (e.az - gForce) * e.dt // gravity always acts unconditionally

	// reset accumulator
	e.ax, e.ay, e.az = 0.0, 0.0, 0.0

	dx, dy, dz := e.vx*e.dt, e.vy*e.dt, e.vz*e.dt
	if math.Abs(dx) < sleepEpsilon && math.Abs(dy) < sleepEpsilon && math.Abs(dz) < sleepEpsilon {
		e.vx = 0.0
		e.vy = 0.0
		if e.onGround {
			e.vz = 0.0
		}
		return
	}

	e.vx *= e.groundDamping //e.dampingActive
	e.vy *= e.groundDamping //e.dampingActive
	e.vz *= e.airDamping
	// planar sleep - zero only xy to stop micro-sliding (jittering).
	if math.Abs(e.vx) < e.vMin {
		e.vx = 0.0
	}
	if math.Abs(e.vy) < e.vMin {
		e.vy = 0.0
	}
	// terminal velocity clamping
	if e.vz < e.terminalZVelocity {
		e.vz = e.terminalZVelocity
	}
}

// ResolveImpact resolves a collision between two Cinematic objects by applying normal and tangential impulses for both.
func (e *Cinematic) ResolveImpact(e2 *Cinematic, nx, ny, nz float64, _ float64) {
	// safe normalization (anti-explosion)
	nLen := math.Sqrt(nx*nx + ny*ny + nz*nz)
	if nLen > 0.0 {
		nx /= nLen
		ny /= nLen
		nz /= nLen
	} else {
		return // null vector, cannot resolve
	}

	// relative velocity along the normal
	vrx := e.vx - e2.vx
	vry := e.vy - e2.vy
	vrz := e.vz - e2.vz
	vRelDotN := vrx*nx + vry*ny + vrz*nz

	// if bodies are separating, no collision to resolve
	if vRelDotN > 0.0 {
		return
	}

	invMassSum := e.invMass + e2.invMass
	if invMassSum == 0.0 {
		return
	}

	// restitution with threshold (micro-bounce slop)
	actualRestitution := e2.restitution
	if actualRestitution != 0 {
		const restitutionSlop = 0.5
		if math.Abs(vRelDotN) < restitutionSlop {
			actualRestitution = 0.0
		}
	}

	// baumgarte stabilization ---
	//const slop = 0.05
	//const percent = 0.2
	//bias := max(penetration-slop, 0.0) * percent

	// normal impulse (pure, without baumgarte bias)
	impulse := -(1.0 + actualRestitution) * vRelDotN
	j := impulse / invMassSum

	// apply normal impulse
	e.vx += (j * nx) * e.invMass
	e.vy += (j * ny) * e.invMass
	e.vz += (j * nz) * e.invMass
	e2.vx -= (j * nx) * e2.invMass
	e2.vy -= (j * ny) * e2.invMass
	e2.vz -= (j * nz) * e2.invMass

	// tangential impulse (friction)
	// recalculates relative velocity after normal impulse
	vrx = e.vx - e2.vx
	vry = e.vy - e2.vy
	vrz = e.vz - e2.vz
	vRelDotNPost := vrx*nx + vry*ny + vrz*nz

	// find tangent vector
	tx := vrx - (vRelDotNPost * nx)
	ty := vry - (vRelDotNPost * ny)
	tz := vrz - (vRelDotNPost * nz)

	if tLen := math.Sqrt(tx*tx + ty*ty + tz*tz); tLen > 1e-8 {
		tx /= tLen
		ty /= tLen
		tz /= tLen

		vRelDotT := vrx*tx + vry*ty + vrz*tz
		jt := -vRelDotT / invMassSum

		// friction mixing: geometric mean of the two friction coefficients
		mu := math.Sqrt(e.frictionActive * e2.frictionActive)
		maxFriction := j * mu

		// clamp tangential impulse within coulomb cone
		if math.Abs(jt) > maxFriction {
			jt = math.Copysign(maxFriction, jt)
		}

		// apply friction impulse
		e.vx += (jt * tx) * e.invMass
		e.vy += (jt * ty) * e.invMass
		e.vz += (jt * tz) * e.invMass
		e2.vx -= (jt * tx) * e2.invMass
		e2.vy -= (jt * ty) * e2.invMass
		e2.vz -= (jt * tz) * e2.invMass
	}
}
