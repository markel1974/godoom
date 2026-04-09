package physics

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/utils"
)

// vMin defines the minimum threshold for velocity components to be considered negligible during calculations.
const vMin = 0.001
const minThickness = 0.01
const safetyMargin float64 = 0.90
const dt60 float64 = 1.0 / 60.0
const dt120 float64 = 1.0 / 120.0

// Entity represents a dynamic object with physical attributes, motion parameters, collision detection, and response data.
type Entity struct {
	rect               Rect
	id                 string
	mass               float64
	invMass            float64
	vx                 float64
	vy                 float64
	vz                 float64
	ax                 float64
	ay                 float64
	az                 float64
	vMin               float64
	defaultFriction    float64
	defaultAirFriction float64
	airFriction        float64
	gForce             float64
	restitution        float64
	maxVelocitySq      float64
	sleepThresholdSq   float64
	dt                 float64
	//depth              float64
	terminalZVelocity float64
	friction          float64
	collider          *Entity
}

// NewEntity initializes a new Entity with the specified position, dimensions, and mass, and sets default parameters.
func NewEntity(x, y, z, w, h, d, mass, restitution, friction float64) *Entity {
	if restitution <= 0.0 {
		restitution = 0.2
	}
	a := &Entity{
		id:               utils.NextUUId(),
		rect:             NewRect(x, y, w, h, z, d),
		mass:             0.0,
		invMass:          0.0,
		vx:               0.0,
		vy:               0.0,
		gForce:           0.2,
		vMin:             vMin,
		sleepThresholdSq: vMin * vMin,
		dt:               dt60,
		friction:         friction,
		restitution:      restitution,
	}
	if mass <= 0.0 {
		a.mass = 0.0
		a.invMass = 0.0
	} else {
		a.mass = mass
		a.invMass = 1.0 / a.mass
	}

	a.SetFriction(0.9)
	a.SetAirFriction(0.98)
	a.SetMaxVelocity(minThickness, safetyMargin)

	if damping := math.Pow(a.airFriction, a.dt); damping >= 1.0 {
		// Nessun attrito atmosferico, la velocità terminale tenderebbe a infinito
		a.terminalZVelocity = -math.MaxFloat64
	} else {
		// Calcolo dell'asintoto dell'integratore
		a.terminalZVelocity = (-a.gForce * a.dt * damping) / (1.0 - damping)
	}
	return a
}

// Reset initializes the entity's position, dimensions, velocity, and physics properties to their default states.
func (e *Entity) Reset(x, y, w, h, z, d, mass, restitution float64) {
	e.rect.Reset(x, y, w, h, z, d)
	e.mass = mass
	e.invMass = 1.0 / mass
	e.vx = 0.0
	e.vy = 0.0
	e.friction = e.defaultFriction
	e.airFriction = e.defaultAirFriction
	e.restitution = restitution
	e.rect.rebuild()
}

// Stop halts all movement by setting the velocities (vx, vy, vz) of the entity to zero.
func (e *Entity) Stop() {
	e.vx = 0.0
	e.vy = 0.0
	e.vz = 0.0
}

/*
// Update adjusts the entity's velocity based on friction, gravity, and minimum velocity thresholds. Checks collision status.
func (e *Entity) Update2() bool {
	// Clearance esatto tramite AABB
	if e.collider != nil {
		if !e.HasCollision(e.collider) {
			e.clearCollider()
		}
	}
	e.vx *= e.friction
	e.vy *= e.friction
	if e.vz > 0.0 {
		e.vz = math.Max(0.0, e.vz-e.gForce)
	} else if e.vz < 0.0 {
		e.vz = math.Min(0.0, e.vz+e.gForce)
	}
	e.vz *= e.airFriction
	// 4. CLAMPING DELLE VELOCITÀ MINIME
	if math.Abs(e.vx) < e.vMin {
		e.vx = 0.0
	}
	if math.Abs(e.vy) < e.vMin {
		e.vy = 0.0
	}
	if math.Abs(e.vz) < e.vMin {
		e.vz = 0.0
	}
	if e.IsMoving() {
		return true
	}
	return false
}
*/

// MoveTo sets the Entity's position to the specified x, y, and z coordinates.
func (e *Entity) MoveTo(x float64, y float64, z float64) {
	e.rect.MoveTo(x, y, z)
}

// SetFriction updates the friction value of the entity and resets it to the default friction value.
func (e *Entity) SetFriction(f float64) {
	e.defaultFriction = f
	e.friction = e.defaultFriction
}

// SetAirFriction updates the air friction values for the entity, affecting the rate of velocity reduction in the Z axis.
func (e *Entity) SetAirFriction(f float64) {
	e.defaultAirFriction = f
	e.airFriction = e.defaultAirFriction
}

func (e *Entity) SetMaxVelocity(minThickness float64, safetyMargin float64) {
	maxVelocity := (minThickness * safetyMargin) / e.dt
	e.maxVelocitySq = maxVelocity * maxVelocity
}

// GetVx retrieves the current velocity of the entity along the x-axis.
func (e *Entity) GetVx() float64 {
	return e.vx
}

// GetVy returns the current vertical velocity (vy) of the entity.
func (e *Entity) GetVy() float64 {
	return e.vy
}

// GetVz returns the current velocity of the entity along the Z-axis.
func (e *Entity) GetVz() float64 { return e.vz }

// SetVx sets the velocity along the x-axis for the entity.
func (e *Entity) SetVx(vx float64) {
	e.vx = vx
}

// SetVy updates the vertical velocity (vy) of the entity.
func (e *Entity) SetVy(vy float64) {
	e.vy = vy
}

// SetVz sets the entity's velocity along the Z-axis.
func (e *Entity) SetVz(vz float64) { e.vz = vz }

// SetV updates the velocity components vx, vy, and vz of the entity.
func (e *Entity) SetV(vx, vy, vz float64) {
	e.vx = vx
	e.vy = vy
	e.vz = vz
}

// AddV adjusts the velocity of the entity by adding the provided vx, vy, and vz values to its current velocity components.
func (e *Entity) AddV(vx, vy, vz float64) {
	e.vx += vx
	e.vy += vy
	e.vz += vz
}

// SubV subtracts the given vx, vy, and vz values from the Entity's velocity components.
func (e *Entity) SubV(vx, vy, vz float64) {
	e.vx -= vx
	e.vy -= vy
	e.vz -= vz
}

// GetId retrieves the unique identifier of the Entity.
func (e *Entity) GetId() string {
	return e.id
}

// Invalidate clears the currently associated collider of the entity.
func (e *Entity) Invalidate() {
	e.clearCollider()
}

// GetWidth returns the width of the entity based on its rectangular bounds.
func (e *Entity) GetWidth() float64 {
	return e.rect.GetWidth()
}

// GetInvMass returns the inverse mass of the entity, a precomputed value used in physics calculations.
func (e *Entity) GetInvMass() float64 {
	return e.invMass
}

// GetRestitution returns the restitution coefficient of the entity, representing its bounciness during collisions.
func (e *Entity) GetRestitution() float64 {
	return e.restitution
}

// GetAABB returns the axis-aligned bounding box (AABB) of the entity.
func (e *Entity) GetAABB() *AABB {
	return e.rect.GetAABB()
}

// GetCenter returns the center point of the entity as (x, y, z) coordinates.
func (e *Entity) GetCenter() (float64, float64, float64) {
	return e.rect.GetCenter()
}

// GetDepth returns the depth dimension of the entity, as defined by its rectangular bounds.
func (e *Entity) GetDepth() float64 {
	return e.rect.GetDepth()
}

// GetGForce returns the current gravitational force value acting on the Entity.
func (e *Entity) GetGForce() float64 {
	return e.gForce
}

// HasCollision checks if the current entity's bounding box intersects with the bounding box of the provided entity.
func (e *Entity) HasCollision(obj2 *Entity) bool {
	return e.rect.IntersectRect(obj2.rect)
}

// Distance computes the Euclidean distance between the entity and another specified entity based on their center coordinates.
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

// DistanceSq computes the squared distance between the centers of the current entity and another entity in 3D space.
func (e *Entity) DistanceSq(other *Entity) float64 {
	// Centro X, Y
	dx := (e.rect.point.x + e.rect.size.w/2.0) - (other.rect.point.x + other.rect.size.w/2.0)
	dy := (e.rect.point.y + e.rect.size.h/2.0) - (other.rect.point.y + other.rect.size.h/2.0)
	// Centro Z
	dz := (e.rect.point.z + e.rect.size.d/2.0) - (other.rect.point.z + other.rect.size.d/2.0)
	return dx*dx + dy*dy + dz*dz
}

// GetXRange returns the minimum and maximum x-coordinates of the entity's rectangular bounds as a range.
func (e *Entity) GetXRange() (float64, float64) {
	return e.rect.point.x, e.rect.point.x + e.rect.size.w
}

// GetYRange returns the minimum and maximum Y values of the entity's rectangular bounds as a tuple.
func (e *Entity) GetYRange() (float64, float64) {
	return e.rect.point.y, e.rect.point.y + e.rect.size.h
}

// GetZRange returns the minimum and maximum Z values of the entity's rectangular bounds.
func (e *Entity) GetZRange() (float64, float64) {
	return e.rect.point.z, e.rect.point.z + e.rect.size.d
}

// GetSweptZRange calculates the swept Z-axis range based on current vertical velocity and gravity force.
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

// IsMoving checks if the Entity is currently in motion by determining if any of its velocity components are non-zero.
func (e *Entity) IsMoving() bool {
	return e.vx != 0 || e.vy != 0 || e.vz != 0 || e.ax != 0 || e.ay != 0 || e.az != 0
}

// clearCollider removes the current collider reference from the entity and ensures mutual disassociation between colliders.
func (e *Entity) clearCollider() {
	if e.collider != nil {
		if e.collider.collider == e {
			e.collider.collider = nil
		}
		e.collider = nil
	}
}

func (e *Entity) GetVelocity() (float64, float64, float64) {
	return e.vx, e.vy, e.vz
}

func (e *Entity) GetDisplacement() (float64, float64, float64) {
	return e.vx * e.dt, e.vy * e.dt, e.vz * e.dt
}

// AddForce applies a force to the entity by adjusting its acceleration based on the given force components and its inverse mass.
func (e *Entity) AddForce(fx, fy, fz float64) {
	e.ax += fx * e.invMass
	e.ay += fy * e.invMass
	e.az += fz * e.invMass
}

// Update esegue unicamente l'integrazione balistica (Eulero Semi-Implicito).
// L'attrito radente e l'assorbimento dell'impatto sono delegati al risolutore esterno.
func (e *Entity) Update() bool {
	// 1. INTEGRAZIONE
	e.vx += e.ax * e.dt
	e.vy += e.ay * e.dt
	e.vz += (e.az - e.gForce) * e.dt // La gravità agisce sempre e senza condizioni

	// 2. DAMPING (Smorzamento atmosferico)
	airDamping := math.Pow(e.airFriction, e.dt)
	e.vx *= airDamping
	e.vy *= airDamping
	e.vz *= airDamping

	// 3. SLEEP PLANARE
	// Azzeriamo solo XY per fermare i micro-scivolamenti (jittering).
	if math.Abs(e.vx) < e.vMin {
		e.vx = 0.0
	}
	if math.Abs(e.vy) < e.vMin {
		e.vy = 0.0
	}

	// 4. CLAMPING VELOCITÀ TERMINALE (Asse Z)
	if e.vz < e.terminalZVelocity {
		e.vz = e.terminalZVelocity
	}

	//if e.vz != 0 {
	//	fmt.Println("VZ", e.vz) // Ora vedrai la gravità accumularsi correttamente
	//}

	// 5. RESET ACCUMULATORE
	e.ax, e.ay, e.az = 0.0, 0.0, 0.0

	return e.IsMoving()
}

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

		maxFriction := j * e2.friction
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

/*
// GetCollisionManifold calcola la normale di impatto e la profondità di compenetrazione tra due AABB.
// La normale restituita punterà da 'other' verso 'e' (come richiesto da ResolveImpact).
func (e *Entity) GetCollisionManifold(other *Entity) (nx, ny, nz, depth float64, hit bool) {
	// Calcolo delle distanze tra i centri
	cx1, cy1, cz1 := e.rect.GetCenter()
	cx2, cy2, cz2 := other.rect.GetCenter()

	dx := cx1 - cx2
	dy := cy1 - cy2
	dz := cz1 - cz2

	// Calcolo delle semi-estensioni (extents)
	extX1, extY1, extZ1 := e.rect.size.w/2, e.rect.size.h/2, e.rect.size.d/2
	extX2, extY2, extZ2 := other.rect.size.w/2, other.rect.size.h/2, other.rect.size.d/2

	// Calcolo degli overlap sugli assi
	overlapX := (extX1 + extX2) - math.Abs(dx)
	if overlapX <= 0 {
		return 0, 0, 0, 0, false
	} // Separazione su X

	overlapY := (extY1 + extY2) - math.Abs(dy)
	if overlapY <= 0 {
		return 0, 0, 0, 0, false
	} // Separazione su Y

	overlapZ := (extZ1 + extZ2) - math.Abs(dz)
	if overlapZ <= 0 {
		return 0, 0, 0, 0, false
	} // Separazione su Z

	// Trova l'asse di minima penetrazione per risolvere la collisione spingendo fuori il meno possibile
	hit = true
	if overlapX < overlapY && overlapX < overlapZ {
		depth = overlapX
		if dx > 0 {
			nx = 1.0
		} else {
			nx = -1.0
		}
	} else if overlapY < overlapZ {
		depth = overlapY
		if dy > 0 {
			ny = 1.0
		} else {
			ny = -1.0
		}
	} else {
		depth = overlapZ
		if dz > 0 {
			nz = 1.0
		} else {
			nz = -1.0
		}
	}

	return nx, ny, nz, depth, hit
}


*/
