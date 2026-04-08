package physics

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/utils"
)

// vMin defines the minimum threshold for velocity components to be considered negligible during calculations.
const vMin = 0.001

// Entity represents a dynamic object with physical attributes, motion parameters, collision detection, and response data.
type Entity struct {
	rect               Rect
	id                 string
	mass               float64
	invMass            float64
	vx                 float64
	vy                 float64
	vz                 float64
	vxMin              float64
	vyMin              float64
	vzMin              float64
	defaultFriction    float64
	friction           float64
	defaultAirFriction float64
	airFriction        float64
	defaultGForce      float64
	gForce             float64
	impulse            float64
	restitution        float64
	collider           *Entity
}

// NewEntity initializes a new Entity with the specified position, dimensions, and mass, and sets default parameters.
func NewEntity(x, y, z, w, h, d, mass, restitution float64) *Entity {
	if restitution <= 0.0 {
		restitution = 0.2
	}
	a := &Entity{
		id:          utils.NextUUId(),
		rect:        NewRect(x, y, w, h, z, d),
		mass:        mass,
		invMass:     1.0 / mass,
		vx:          0.0,
		vy:          0.0,
		vxMin:       vMin,
		vyMin:       vMin,
		vzMin:       vMin,
		impulse:     vMin,
		restitution: restitution,
	}
	a.SetFriction(0.9)
	a.SetAirFriction(0.98)
	//TODO BUG impostare gForce a 0.1 [con valori diversi da 0 non funziona il motore delle collisioni] anche thingbase.go
	//a.SetGForce(0.1)
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
	e.gForce = e.defaultGForce
	e.vxMin = vMin
	e.vyMin = vMin
	e.impulse = vMin
	e.restitution = restitution
	e.rect.rebuild()
}

// Stop halts all movement by setting the velocities (vx, vy, vz) of the entity to zero.
func (e *Entity) Stop() {
	e.vx = 0.0
	e.vy = 0.0
	e.vz = 0.0
}

// Update adjusts the entity's velocity based on friction, gravity, and minimum velocity thresholds. Checks collision status.
func (e *Entity) Update() bool {
	// Clearance esatto tramite AABB
	if e.collider != nil {
		if !e.HasCollision(e.collider) {
			e.clearCollider()
		}
	}
	e.vx *= e.friction
	e.vy *= e.friction
	e.vz *= e.airFriction
	e.vz -= e.gForce
	// 4. CLAMPING DELLE VELOCITÀ MINIME
	if math.Abs(e.vx) < e.vxMin {
		e.vx = 0.0
	}
	if math.Abs(e.vy) < e.vyMin {
		e.vy = 0.0
	}
	if math.Abs(e.vz) < e.vzMin {
		e.vz = 0.0
	}
	return e.IsMoving()
}

//func (e *Entity) AddTo(x float64, y float64, z float64) {
//	e.rect.AddTo(x, y, z)
//}

// MoveTest calculates potential movement based on current velocities and returns the resulting position deltas (dx, dy, dz).
//func (e *Entity) MoveTest() (float64, float64, float64) {
//	return e.rect.MoveTest(e.vx, e.vy, e.vz)
//}

// Move updates the position of the entity by adding its velocity components (vx, vy, vz) to its current position.
//func (e *Entity) Move() {
//	e.rect.AddTo(e.vx, e.vy, e.vz)
//}

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

// SetGForce updates the gravitational force applied to the entity, resetting both gForce and defaultGForce.
func (e *Entity) SetGForce(gForce float64) {
	e.defaultGForce = gForce
	e.gForce = e.defaultGForce
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
	return e.vx != 0 || e.vy != 0 || e.vz != 0
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

/*
// SetupCollision establishes a collision relationship between the current entity and another entity.
func (e *Entity) SetupCollision(otherEnt *Entity) {
	e.collider = otherEnt
	otherEnt.collider = e

	c1x, c1y, c1z := e.rect.GetCenter()
	c2x, c2y, c2z := otherEnt.rect.GetCenter()

	dx := c2x - c1x
	dy := c2y - c1y
	dz := c2z - c1z

	// 1. Calcolo delle semi-estensioni (Extents)
	extX1, extY1, extZ1 := e.rect.GetWidth()/2.0, e.rect.GetHeight()/2.0, e.rect.GetDepth()/2.0
	extX2, extY2, extZ2 := otherEnt.rect.GetWidth()/2.0, otherEnt.rect.GetHeight()/2.0, otherEnt.rect.GetDepth()/2.0

	// 2. Calcolo delle penetrazioni assiali
	overlapX := (extX1 + extX2) - math.Abs(dx)
	overlapY := (extY1 + extY2) - math.Abs(dy)
	overlapZ := (extZ1 + extZ2) - math.Abs(dz)

	// Uscita anticipata di sicurezza (prevenzione edge-case se chiamati fuori fase)
	if overlapX <= 0 || overlapY <= 0 || overlapZ <= 0 {
		return
	}

	// 3. Determinazione dell'asse di minima penetrazione (Collision Normal)
	var nx, ny, nz float64
	var penetration float64

	if overlapX < overlapY && overlapX < overlapZ {
		nx = math.Copysign(1.0, dx)
		ny, nz = 0, 0
		penetration = overlapX
	} else if overlapY < overlapZ {
		nx, nz = 0, 0
		ny = math.Copysign(1.0, dy)
		penetration = overlapY
	} else {
		nx, ny = 0, 0
		nz = math.Copysign(1.0, dz)
		penetration = overlapZ
	}

	// 4. Prodotto scalare per la velocità di avvicinamento lungo la normale esatta
	relVx := otherEnt.vx - e.vx
	relVy := otherEnt.vy - e.vy
	relVz := otherEnt.vz - e.vz
	vRelDotN := relVx*nx + relVy*ny + relVz*nz

	// 5. Risoluzione Impulso (Newtonian Response)
	if vRelDotN < 0 {
		restitution := 1.0
		j := -(1.0 + restitution) * vRelDotN
		j /= (1.0 / e.mass) + (1.0 / otherEnt.mass)

		e.vx -= (j / e.mass) * nx
		e.vy -= (j / e.mass) * ny
		e.vz -= (j / e.mass) * nz

		otherEnt.vx += (j / otherEnt.mass) * nx
		otherEnt.vy += (j / otherEnt.mass) * ny
		otherEnt.vz += (j / otherEnt.mass) * nz
	}

	// 6. Stabilizzazione di Baumgarte (Positional Projection)
	const percent = 0.2
	const slop = 0.01
	if penetration > slop {
		correction := (math.Max(penetration-slop, 0.0) / (1.0/e.mass + 1.0/otherEnt.mass)) * percent
		cx, cy, cz := nx*correction, ny*correction, nz*correction
		e.rect.AddTo(-cx/e.mass, -cy/e.mass, -cz/e.mass)
		otherEnt.rect.AddTo(cx/otherEnt.mass, cy/otherEnt.mass, cz/otherEnt.mass)
	}
}

*/
