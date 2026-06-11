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
		// Nessun attrito atmosferico, la velocità terminale tenderebbe a infinito
		e.terminalZVelocity = -math.MaxFloat64
	} else {
		// Calcolo dell'asintoto dell'integratore
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
	e.vz += (e.az - gForce) * e.dt // La gravità agisce sempre e senza condizioni

	// RESET ACCUMULATORE
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
	// SLEEP PLANARE Azzeriamo solo XY per fermare i micro-scivolamenti (jittering).
	if math.Abs(e.vx) < e.vMin {
		e.vx = 0.0
	}
	if math.Abs(e.vy) < e.vMin {
		e.vy = 0.0
	}
	//CLAMPING VELOCITÀ TERMINALE
	if e.vz < e.terminalZVelocity {
		e.vz = e.terminalZVelocity
	}
}

// IsMoving returns true if the object has any non-zero velocity components (vx, vy, or vz), indicating it is in motion.
func (e *Cinematic) IsMoving() bool {
	return e.vx != 0 || e.vy != 0 || e.vz != 0
}

// ResolveImpact resolves a collision between two Cinematic objects by applying normal and tangential impulses for both.
func (e *Cinematic) ResolveImpact(e2 *Cinematic, nx, ny, nz float64, _ float64) {
	// NORMALIZZAZIONE SICURA (Anti-Esplosione)
	nLen := math.Sqrt(nx*nx + ny*ny + nz*nz)
	if nLen > 0.0 {
		nx /= nLen
		ny /= nLen
		nz /= nLen
	} else {
		return // Vettore nullo, impossibile risolvere
	}

	// VELOCITÀ RELATIVA LUNGO LA NORMALE
	vrx := e.vx - e2.vx
	vry := e.vy - e2.vy
	vrz := e.vz - e2.vz
	vRelDotN := vrx*nx + vry*ny + vrz*nz

	// Se i corpi si stanno allontanando, nessuna collisione da risolvere
	if vRelDotN > 0.0 {
		return
	}

	invMassSum := e.invMass + e2.invMass
	if invMassSum == 0.0 {
		return
	}

	// RESTITUZIONE CON SOGLIA (Micro-bounce slop)
	actualRestitution := e2.restitution
	if actualRestitution != 0 {
		const restitutionSlop = 0.5
		if math.Abs(vRelDotN) < restitutionSlop {
			actualRestitution = 0.0
		}
	}

	// BAUMGARTE STABILIZATION ---
	//const slop = 0.05
	//const percent = 0.2
	//bias := math.Max(penetration-slop, 0.0) * percent

	// IMPULSO NORMALE (Puro, senza Baumgarte bias)
	impulse := -(1.0 + actualRestitution) * vRelDotN
	j := impulse / invMassSum

	// Applica l'impulso normale
	e.vx += (j * nx) * e.invMass
	e.vy += (j * ny) * e.invMass
	e.vz += (j * nz) * e.invMass
	e2.vx -= (j * nx) * e2.invMass
	e2.vy -= (j * ny) * e2.invMass
	e2.vz -= (j * nz) * e2.invMass

	// IMPULSO TANGENZIALE (ATTRITO)
	// Ricalcoliamo la velocità relativa DOPO l'impulso normale
	vrx = e.vx - e2.vx
	vry = e.vy - e2.vy
	vrz = e.vz - e2.vz
	vRelDotNPost := vrx*nx + vry*ny + vrz*nz

	// Troviamo il vettore tangente
	tx := vrx - (vRelDotNPost * nx)
	ty := vry - (vRelDotNPost * ny)
	tz := vrz - (vRelDotNPost * nz)

	if tLen := math.Sqrt(tx*tx + ty*ty + tz*tz); tLen > 1e-8 {
		tx /= tLen
		ty /= tLen
		tz /= tLen

		vRelDotT := vrx*tx + vry*ty + vrz*tz
		jt := -vRelDotT / invMassSum

		// Friction Mixing: Media geometrica dei due coefficienti di attrito
		mu := math.Sqrt(e.frictionActive * e2.frictionActive)
		maxFriction := j * mu

		// Clamp dell'impulso tangenziale nel cono di Coulomb
		if math.Abs(jt) > maxFriction {
			jt = math.Copysign(maxFriction, jt)
		}

		// Applica l'impulso di attrito
		e.vx += (jt * tx) * e.invMass
		e.vy += (jt * ty) * e.invMass
		e.vz += (jt * tz) * e.invMass
		e2.vx -= (jt * tx) * e2.invMass
		e2.vy -= (jt * ty) * e2.invMass
		e2.vz -= (jt * tz) * e2.invMass
	}
}

// ClearForce resets the accumulated force components (ax, ay, az) of the entity to zero.
//func (e *Entity) ClearForce() {
//	e.ax, e.ay, e.az = 0.0, 0.0, 0.0
//}

/*
// ResolveImpact resolves the collision impact between two entities, applying forces based on restitution, friction, and penetration.
func (e *Entity) ResolveImpact(e2 *Entity, nx, ny, nz float64, penetration float64) {
	// NORMALIZZAZIONE SICURA (Anti-Esplosione)
	nLen := math.Sqrt(nx*nx + ny*ny + nz*nz)
	if nLen > 0.0 {
		nx /= nLen
		ny /= nLen
		nz /= nLen
	} else {
		return // Vettore nullo, impossibile risolvere
	}
	// VELOCITÀ RELATIVA E SOGLIE
	if e2.invMass == 0.0 {
		e2.vx, e2.vy, e2.vz = 0.0, 0.0, 0.0
	}
	vrx := e.vx - e2.vx
	vry := e.vy - e2.vy
	vrz := e.vz - e2.vz
	vRelDotN := vrx*nx + vry*ny + vrz*nz

	// Se i corpi si stanno allontanando
	if vRelDotN > 0.0 {
		return
	}
	invMassSum := e.invMass + e2.invMass
	if invMassSum == 0.0 {
		return
	}

	const restitutionSlop = 0.5
	actualRestitution := e2.restitution
	if math.Abs(vRelDotN) < restitutionSlop {
		actualRestitution = 0.0
	}

	// BAUMGARTE STABILIZATION ---
	//const slop = 0.05
	//const percent = 0.2
	//bias := math.Max(penetration-slop, 0.0) * percent

	//bias := 0.0
	//if e.ax != 0.0 || e.ay != 0.0 || e.az != 0.0 {
	const slop = 0.05
	const percent = 0.5 //0.2
	bias := math.Max(penetration-slop, 0.0) * percent

	// IMPULSO NORMALE (con bias applicato)
	impulse := -(1.0 + actualRestitution) * vRelDotN
	if bias > impulse {
		bias = impulse
	}
	j := impulse + bias
	//fmt.Println("Applying normal impulse with bias:", impulse, bias)
	j /= invMassSum

	e.vx += (j * nx) * e.invMass
	e.vy += (j * ny) * e.invMass
	e.vz += (j * nz) * e.invMass
	e2.vx -= (j * nx) * e2.invMass
	e2.vy -= (j * ny) * e2.invMass
	e2.vz -= (j * nz) * e2.invMass

	// IMPULSO TANGENZIALE (ATTRITO)
	vrx = e.vx - e2.vx
	vry = e.vy - e2.vy
	vrz = e.vz - e2.vz
	vRelDotNPost := vrx*nx + vry*ny + vrz*nz

	tx := vrx - (vRelDotNPost * nx)
	ty := vry - (vRelDotNPost * ny)
	tz := vrz - (vRelDotNPost * nz)

	tLen := math.Sqrt(tx*tx + ty*ty + tz*tz)
	if tLen > 1e-8 {
		tx /= tLen
		ty /= tLen
		tz /= tLen

		vRelDotT := vrx*tx + vry*ty + vrz*tz
		jt := -vRelDotT / invMassSum

		maxFriction := j * e2.frictionActive
		if math.Abs(jt) > maxFriction {
			// math.Copysign copia il segno di jt su maxFriction, evitando branch
			jt = math.Copysign(maxFriction, jt)
		}

		e.vx += (jt * tx) * e.invMass
		e.vy += (jt * ty) * e.invMass
		e.vz += (jt * tz) * e.invMass
		e2.vx -= (jt * tx) * e2.invMass
		e2.vy -= (jt * ty) * e2.invMass
		e2.vz -= (jt * tz) * e2.invMass
	}
}

*/

/*
// ComputeImpact determines the collision state and calculates the collision normal and penetration depth between two entities.
// Returns the normal vector (normX, normY, normZ), penetration depth, and a boolean indicating whether a collision occurred.
func (e *Entity) ComputeImpact(otherEnt *Entity) (float64, float64, float64, float64, bool) {
	x1Min, x1Max := e.GetXRange()
	x2Min, x2Max := otherEnt.GetXRange()
	y1Min, y1Max := e.GetYRange()
	y2Min, y2Max := otherEnt.GetYRange()
	// SAT: Collisione AABB Planare Veloce
	if x1Max > x2Min && x1Min < x2Max && y1Max > y2Min && y1Min < y2Max {
		z1Min, z1Max := e.GetZRange()
		z2Min, z2Max := otherEnt.GetZRange()
		// Supporto Swept Z per il Continuous Collision Detection verticale
		if math.Abs(e.GetVz()) >= e.GetGForce() {
			z1Min, z1Max = e.GetSweptZRange()
		}
		if math.Abs(otherEnt.GetVz()) >= otherEnt.GetGForce() {
			z2Min, z2Max = otherEnt.GetSweptZRange()
		}
		if z1Max > z2Min && z1Min < z2Max {
			pX1 := x1Max - x2Min
			pX2 := x2Max - x1Min
			pY1 := y1Max - y2Min
			pY2 := y2Max - y1Min
			pZ1 := z1Max - z2Min
			pZ2 := z2Max - z1Min
			minPenetration := pX1
			var normX, normY, normZ float64 = -1, 0, 0
			// Troviamo l'asse di minima compenetrazione
			if pX2 < minPenetration {
				minPenetration = pX2
				normX, normY, normZ = 1, 0, 0
			}
			if pY1 < minPenetration {
				minPenetration = pY1
				normX, normY, normZ = 0, -1, 0
			}
			if pY2 < minPenetration {
				minPenetration = pY2
				normX, normY, normZ = 0, 1, 0
			}
			if pZ1 < minPenetration {
				minPenetration = pZ1
				normX, normY, normZ = 0, 0, -1
			}
			if pZ2 < minPenetration {
				minPenetration = pZ2
				normX, normY, normZ = 0, 0, 1
			}
			if minPenetration > 0.001 {
				return normX, normY, normZ, minPenetration, true
			}
		}
	}
	return 0, 0, 0, 0, false
}

*/

/*
// ClipVelocity adjusts the velocity vector to prevent movement into a surface by negating the velocity component along the normal.
// The method applies a slight over-bounce to mitigate precision errors and ensure the entity does not get stuck on the surface.
func (e *Entity) ClipVelocity(vx, vy, vz, nx, ny, nz float64) (float64, float64, float64) {
	// V · N (How much velocity is pushing against the plane)
	backoff := vx*nx + vy*ny + vz*nz
	// If positive, we are already moving away from the plane, no correction needed
	if backoff > 0.0 {
		return vx, vy, vz
	}
	// Very slight over-bounce (1.001) to absorb FP64 precision error
	// and prevent the entity from getting stuck in the plane on the next frame.
	backoff *= 1.001
	// V_new = V - N * (V · N)
	return vx - (nx * backoff), vy - (ny * backoff), vz - (nz * backoff)
}

*/

/*
// DistanceSq calculates the squared distance between the center points of the current entity and another entity.
func (e *Entity) DistanceSq(other *Entity) float64 {
	dx := (e.rect.point.x + e.rect.size.w/2.0) - (other.rect.point.x + other.rect.size.w/2.0)
	dy := (e.rect.point.y + e.rect.size.h/2.0) - (other.rect.point.y + other.rect.size.h/2.0)
	dz := (e.rect.point.z + e.rect.size.d/2.0) - (other.rect.point.z + other.rect.size.d/2.0)
	return dx*dx + dy*dy + dz*dz
}
*/
