package physics

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/utils"
)

const (
	airFriction = 0.99

	gForce = 9.8

	// vMin represents the minimum velocity threshold below which motion is considered negligible.
	vMin = 0.1

	// minThickness represents the minimum allowable thickness value to prevent calculations with near-zero dimensions.
	minThickness = 0.01

	// safetyMargin defines a proportional buffer value, typically used to ensure numerical stability in calculations.
	safetyMargin float64 = 0.90

	// dt60 represents the fixed time step duration in seconds, commonly used for 60 frames per second simulations.
	dt60 float64 = 1.0 / 60.0

	// dt120 represents the fixed time step duration equivalent to 1/120th of a second.
	dt120 float64 = 1.0 / 120.0
)

// Entity represents a physical object in a simulation with properties for position, velocity, acceleration, and collision.
type Entity struct {
	rect              Rect
	id                string
	mass              float64
	invMass           float64
	vx                float64
	vy                float64
	vz                float64
	ax                float64
	ay                float64
	az                float64
	vMin              float64
	frictionGround    float64
	frictionAir       float64
	frictionActive    float64
	dampingAir        float64
	dampingGround     float64
	dampingActive     float64
	gForce            float64
	restitution       float64
	maxVelocitySq     float64
	sleepThresholdSq  float64
	dt                float64
	terminalZVelocity float64
	onGround          bool
	collider          *Entity
}

// NewEntity creates and returns a pointer to a new Entity initialized with the given position, size, mass, and physical properties.
func NewEntity(x, y, z, w, h, d, mass, restitution, friction float64) *Entity {
	if restitution <= 0.0 {
		restitution = 0.2
	}
	invMass := 0.0
	if mass <= 0.0 {
		mass = 0.0
	} else {
		invMass = 1.0 / mass
	}
	a := &Entity{
		id:               utils.NextUUId(),
		rect:             NewRect(x, y, w, h, z, d),
		gForce:           gForce,
		vMin:             vMin,
		sleepThresholdSq: vMin * vMin,
		dt:               dt60,
		mass:             mass,
		invMass:          invMass,
		restitution:      restitution,
	}
	a.SetFriction(friction)
	a.SetAirFriction(airFriction)
	a.SetMaxVelocity(minThickness, safetyMargin)
	a.SetOnGround(true)
	return a
}

// Reset initializes or updates the Entity's state including position, size, mass, and restitution.
func (e *Entity) Reset(x, y, w, h, z, d, mass, restitution float64) {
	e.rect.Reset(x, y, w, h, z, d)
	e.mass = mass
	e.invMass = 1.0 / mass
	e.vx = 0.0
	e.vy = 0.0
	e.restitution = restitution
	e.rect.rebuild()
}

// Stop sets the entity's velocity components (vx, vy, vz) to zero, effectively halting its movement.
func (e *Entity) Stop() {
	e.vx = 0.0
	e.vy = 0.0
	e.vz = 0.0
}

// MoveTo sets the entity's position to the specified x, y, and z coordinates.
func (e *Entity) MoveTo(x float64, y float64, z float64) {
	e.rect.MoveTo(x, y, z)
}

// SetFriction sets the friction coefficient for the entity and updates its current friction value.
func (e *Entity) SetFriction(f float64) {
	e.frictionGround = f
	e.dampingGround = math.Pow(e.frictionGround, e.dt)
	e.SetOnGround(e.onGround)
}

// SetAirFriction sets the air friction value and updates the active air friction for the entity.
func (e *Entity) SetAirFriction(f float64) {
	e.frictionAir = f
	e.dampingAir = math.Pow(e.frictionAir, e.dt)
	if e.dampingAir >= 1.0 {
		// Nessun attrito atmosferico, la velocità terminale tenderebbe a infinito
		e.terminalZVelocity = -math.MaxFloat64
	} else {
		// Calcolo dell'asintoto dell'integratore
		e.terminalZVelocity = (-e.gForce * e.dt * e.dampingAir) / (1.0 - e.dampingAir)
	}
	e.SetOnGround(e.onGround)
}

// SetMaxVelocity calculates and sets the maximum velocity squared based on the entity's time step and given parameters.
func (e *Entity) SetMaxVelocity(minThickness float64, safetyMargin float64) {
	maxVelocity := (minThickness * safetyMargin) / e.dt
	e.maxVelocitySq = maxVelocity * maxVelocity
}

// SetOnGround sets the onGround state of the entity to the specified boolean value.
func (e *Entity) SetOnGround(onGround bool) {
	e.onGround = onGround
	if e.onGround {
		e.frictionActive = e.frictionGround
		e.dampingActive = e.dampingGround
		//e.vz = 0.0
	} else {
		e.frictionActive = e.frictionAir
		e.dampingActive = e.dampingAir
	}
}

// IsOnGround checks if the entity is currently on the ground and returns true if it is, otherwise false.
func (e *Entity) IsOnGround() bool {
	return e.onGround
}

// GetVx returns the current velocity component along the X-axis for the entity.
func (e *Entity) GetVx() float64 {
	return e.vx
}

// GetVy returns the current vertical velocity (Vy) of the entity.
func (e *Entity) GetVy() float64 {
	return e.vy
}

// GetVz returns the current Z-axis velocity of the entity.
func (e *Entity) GetVz() float64 { return e.vz }

// SetVx sets the velocity of the entity along the X axis.
func (e *Entity) SetVx(vx float64) {
	e.vx = vx
}

// SetVy updates the vertical velocity (vy) of the entity to the specified value.
func (e *Entity) SetVy(vy float64) {
	e.vy = vy
}

// SetVz sets the z-axis velocity (vz) of the entity to the specified value.
func (e *Entity) SetVz(vz float64) { e.vz = vz }

// SetV sets the velocity components of the Entity along the x, y, and z axes.
func (e *Entity) SetV(vx, vy, vz float64) {
	e.vx = vx
	e.vy = vy
	e.vz = vz
}

// AddV increments the entity's velocity components by the specified vx, vy, and vz values.
func (e *Entity) AddV(vx, vy, vz float64) {
	e.vx += vx
	e.vy += vy
	e.vz += vz
}

// SubV subtracts the given velocity components (vx, vy, vz) from the entity's velocity.
func (e *Entity) SubV(vx, vy, vz float64) {
	e.vx -= vx
	e.vy -= vy
	e.vz -= vz
}

// GetId returns the unique identifier of the entity as a string.
func (e *Entity) GetId() string {
	return e.id
}

// Invalidate clears the current collider of the entity by invoking the internal clearCollider method.
func (e *Entity) Invalidate() {
	e.clearCollider()
}

// GetWidth returns the width of the entity by querying its internal rectangular representation.
func (e *Entity) GetWidth() float64 {
	return e.rect.GetWidth()
}

// GetHeight returns the height of the entity by retrieving it from the associated rectangle object.
func (e *Entity) GetHeight() float64 {
	return e.rect.GetHeight()
}

// GetInvMass returns the inverse mass of the entity, which is the reciprocal of its mass.
func (e *Entity) GetInvMass() float64 {
	return e.invMass
}

// GetMass retrieves the mass of the entity. It returns the mass as a float64 value.
func (e *Entity) GetMass() float64 {
	return e.mass
}

// GetRestitution returns the restitution coefficient of the entity, which determines its bounciness upon collision.
func (e *Entity) GetRestitution() float64 {
	return e.restitution
}

// GetAABB returns the axis-aligned bounding box (AABB) of the entity.
func (e *Entity) GetAABB() *AABB {
	return e.rect.GetAABB()
}

// GetCenter returns the 3D center coordinates (x, y, z) of the Entity's bounding rectangle.
func (e *Entity) GetCenter() (float64, float64, float64) {
	return e.rect.GetCenter()
}

// GetDepth returns the depth of the entity's bounding rectangle.
func (e *Entity) GetDepth() float64 {
	return e.rect.GetDepth()
}

// GetGForce returns the gravitational force acting on the entity.
func (e *Entity) GetGForce() float64 {
	return e.gForce
}

// HasCollision checks if the current entity's rectangular boundaries intersect with the specified entity's boundaries.
func (e *Entity) HasCollision(obj2 *Entity) bool {
	return e.rect.IntersectRect(obj2.rect)
}

// Distance computes the Euclidean distance between the entity and a specified collider entity.
func (e *Entity) Distance(collider *Entity) float64 {
	x1, y1, z1 := e.rect.GetCenter()
	x2, y2, z2 := collider.rect.GetCenter()
	dx := x2 - x1
	dy := y2 - y1
	dz := z2 - z1
	d := dx*dx + dy*dy + dz*dz
	if d < 0.0001 {
		return 0.01
	}
	return math.Sqrt(d)
}

// DistanceSq calculates the squared distance between the center points of the current entity and another entity.
func (e *Entity) DistanceSq(other *Entity) float64 {
	dx := (e.rect.point.x + e.rect.size.w/2.0) - (other.rect.point.x + other.rect.size.w/2.0)
	dy := (e.rect.point.y + e.rect.size.h/2.0) - (other.rect.point.y + other.rect.size.h/2.0)
	dz := (e.rect.point.z + e.rect.size.d/2.0) - (other.rect.point.z + other.rect.size.d/2.0)
	return dx*dx + dy*dy + dz*dz
}

// GetXRange returns the min and max x-coordinates of the entity's rectangular bounds.
func (e *Entity) GetXRange() (float64, float64) {
	return e.rect.point.x, e.rect.point.x + e.rect.size.w
}

// GetYRange returns the minimum and maximum Y coordinates of the entity's bounding rectangle as a tuple.
func (e *Entity) GetYRange() (float64, float64) {
	return e.rect.point.y, e.rect.point.y + e.rect.size.h
}

// GetZRange returns the minimum and maximum Z-coordinates of the entity's rectangular bounding box.
func (e *Entity) GetZRange() (float64, float64) {
	return e.rect.point.z, e.rect.point.z + e.rect.size.d
}

// GetSweptZRange computes the range of the entity's Z-axis considering its velocity to account for potential motion effects.
func (e *Entity) GetSweptZRange() (float64, float64) {
	minZ, maxZ := e.GetZRange()
	// Se la velocità è quasi nulla (es. bloccata da wall.Compute),
	// restituisci il range statico per evitare ghost-collisions sotto i piedi
	if math.Abs(e.vz) < e.gForce {
		return minZ, maxZ
	}
	if e.vz > 0 {
		maxZ += e.vz
	} else {
		minZ += e.vz
	}
	return minZ, maxZ
}

// IsMoving checks if the entity is currently in motion based on its velocity or acceleration along any axis.
func (e *Entity) IsMoving() bool {
	return e.vx != 0 || e.vy != 0 || e.vz != 0 || e.ax != 0 || e.ay != 0 || e.az != 0
}

// clearCollider removes the association with the current collider, ensuring any mutual reference is also cleared.
func (e *Entity) clearCollider() {
	if e.collider != nil {
		if e.collider.collider == e {
			e.collider.collider = nil
		}
		e.collider = nil
	}
}

// GetVelocity retrieves the velocity components of the entity along the x, y, and z axes.
func (e *Entity) GetVelocity() (float64, float64, float64) {
	return e.vx, e.vy, e.vz
}

// GetDisplacement computes the displacement of an entity based on its velocity and time step. Returns dx, dy, dz.
func (e *Entity) GetDisplacement() (float64, float64, float64) {
	return e.vx * e.dt, e.vy * e.dt, e.vz * e.dt
}

// AddForce applies a force to the entity, altering its acceleration based on the inverse of its mass.
func (e *Entity) AddForce(fx, fy, fz float64) {
	e.ax += fx * e.invMass
	e.ay += fy * e.invMass
	e.az += fz * e.invMass
}

// Update updates the entity's velocity and position based on applied forces, damping, gravity, and constraints.
func (e *Entity) Update() bool {
	e.vx += e.ax * e.dt
	e.vy += e.ay * e.dt
	e.vz += (e.az - e.gForce) * e.dt // La gravità agisce sempre e senza condizioni

	e.vx *= e.dampingGround //e.dampingActive
	e.vy *= e.dampingGround //e.dampingActive
	e.vz *= e.dampingAir
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
	// 5. RESET ACCUMULATORE
	e.ax, e.ay, e.az = 0.0, 0.0, 0.0
	return e.IsMoving()
}

// ResolveImpact calculates and applies the collision response between two entities based on their velocities, masses, and properties.
func (e *Entity) ResolveImpact(e2 *Entity, nx, ny, nz float64) {
	// 1. NORMALIZZAZIONE SICURA (Anti-Esplosione)
	nLen := math.Sqrt(nx*nx + ny*ny + nz*nz)
	if nLen > 0.0 {
		nx /= nLen
		ny /= nLen
		nz /= nLen
	} else {
		return // Vettore nullo, impossibile risolvere
	}
	// 2. VELOCITÀ RELATIVA E SOGLIE
	// Assicuriamoci che un oggetto a massa infinita (es. il muro) non abbia mai velocità residua
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
		return // Masse infinite contro masse infinite
	}

	const restitutionSlop = 1.0
	actualRestitution := e2.restitution
	if math.Abs(vRelDotN) < restitutionSlop {
		actualRestitution = 0.0
	}

	// 3. IMPULSO NORMALE
	j := -(1.0 + actualRestitution) * vRelDotN
	j /= invMassSum

	e.vx += (j * nx) * e.invMass
	e.vy += (j * ny) * e.invMass
	e.vz += (j * nz) * e.invMass
	e2.vx -= (j * nx) * e2.invMass
	e2.vy -= (j * ny) * e2.invMass
	e2.vz -= (j * nz) * e2.invMass

	// 4. IMPULSO TANGENZIALE (ATTRITO)
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

// ComputeCollision checks for a collision between two entities and returns collision normal, penetration depth, and status.
func (e *Entity) ComputeCollision(otherEnt *Entity) (float64, float64, float64, float64, bool) {
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
