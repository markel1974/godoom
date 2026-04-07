package physics

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/utils"
)

// vMin defines the minimum velocity threshold below which motion is considered negligible for an entity.
const vMin = 0.00001

// CalcDistance calculates the Euclidean distance between two points in 3D space defined by their coordinates.
func CalcDistance(x1, y1, z1, x2, y2, z2 float64) float64 {
	dx := x2 - x1
	dy := y2 - y1
	dz := z2 - z1
	d := dx*dx + dy*dy + dz*dz
	if d < 0.0001 {
		return 0.01
	}
	return math.Sqrt(d)
}

// Entity represents a physical object in a 3D space with properties like position, velocity, mass, and collision handling.
type Entity struct {
	rect               Rect
	id                 string
	mass               float64
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
	collider           *Entity
}

// NewEntity creates a new Entity instance with the specified position, dimensions, and mass. It initializes default properties.
func NewEntity(x float64, y float64, z float64, w float64, h float64, d float64, mass float64) *Entity {
	a := &Entity{
		id:      utils.NextUUId(),
		rect:    NewRect(x, y, w, h, z, d),
		mass:    mass,
		vx:      0.0,
		vy:      0.0,
		vxMin:   vMin,
		vyMin:   vMin,
		vzMin:   vMin,
		impulse: vMin,
	}
	a.SetFriction(0.9)
	a.SetAirFriction(0.2)
	a.SetGForce(0.3)
	return a
}

// Reset reinitializes the Entity's position, size, depth, mass, and resets velocity, friction, and related properties to defaults.
func (e *Entity) Reset(x float64, y float64, w float64, h float64, z float64, d float64, mass float64) {
	e.rect.Reset(x, y, w, h, z, d)
	e.mass = mass
	e.vx = 0.0
	e.vy = 0.0
	e.friction = e.defaultFriction
	e.airFriction = e.defaultAirFriction
	e.gForce = e.defaultGForce
	e.vxMin = vMin
	e.vyMin = vMin
	e.impulse = vMin
	e.rect.rebuild()
}

// Stop halts all movement of the entity by setting its velocity components (vx, vy, vz) to zero.
func (e *Entity) Stop() {
	e.vx = 0.0
	e.vy = 0.0
	e.vz = 0.0
}

func (e *Entity) Update() bool {
	// Clearance esatto tramite AABB
	if e.collider != nil {
		if !e.HasCollision(e.collider) {
			e.clearCollider()
		}
	}
	e.vx *= e.friction
	e.vy *= e.friction
	e.vz *= e.defaultAirFriction
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
	return e.isMoving()
}

// MoveTest calculates the new x, y, z coordinates of the Entity based on its current velocity (vx, vy, vz).
func (e *Entity) MoveTest() (float64, float64, float64) {
	return e.rect.MoveTest(e.vx, e.vy, e.vz)
}

// Move adjusts the position of the Entity by adding its velocity components (vx, vy, vz) to its current Rect position.
func (e *Entity) Move() {
	e.rect.AddTo(e.vx, e.vy, e.vz)
}

// SetFriction sets the default friction value for the entity, affecting its velocity decay over time.
func (e *Entity) SetFriction(f float64) {
	e.defaultFriction = f
	e.friction = e.defaultFriction
}

func (e *Entity) SetAirFriction(f float64) {
	e.defaultAirFriction = f
	e.airFriction = e.defaultAirFriction
}

// SetGForce sets the gravitational force (G-Force) acting on the entity to the specified value.
func (e *Entity) SetGForce(gForce float64) {
	e.defaultGForce = gForce
	e.gForce = e.defaultGForce
}

// GetVx returns the current velocity along the X-axis (horizontal movement) for the entity.
func (e *Entity) GetVx() float64 {
	return e.vx
}

// GetVy retrieves the current vertical velocity (Vy) of the entity.
func (e *Entity) GetVy() float64 {
	return e.vy
}

// GetVz retrieves the current velocity of the entity along the Z-axis.
func (e *Entity) GetVz() float64 { return e.vz }

// SetVx sets the horizontal velocity (Vx) of the entity to the specified value.
func (e *Entity) SetVx(vx float64) {
	e.vx = vx
}

// SetVy sets the vertical velocity (vy) of the entity to the specified value.
func (e *Entity) SetVy(vy float64) {
	e.vy = vy
}

// SetVz sets the z-axis velocity (vz) of the Entity to the specified value.
func (e *Entity) SetVz(vz float64) { e.vz = vz }

// GetId returns the unique identifier of the entity as a string.
func (e *Entity) GetId() string {
	return e.id
}

// Invalidate clears the current active collider associated with the entity.
func (e *Entity) Invalidate() {
	e.clearCollider()
}

// GetWidth returns the width of the Entity by retrieving the width of its Rect.
func (e *Entity) GetWidth() float64 {
	return e.rect.GetWidth()
}

// GetAABB returns the Axis-Aligned Bounding Box (AABB) of the entity's rectangular bounds.
func (e *Entity) GetAABB() *AABB {
	return e.rect.GetAABB()
}

// MoveTo sets the Entity's position to the specified x, y, and z coordinates and updates its spatial data.
func (e *Entity) MoveTo(x float64, y float64, z float64) {
	e.rect.MoveTo(x, y, z)
}

// GetCenter returns the 3D center coordinates (x, y, z) of the Entity based on its Rect's center.
func (e *Entity) GetCenter() (float64, float64, float64) {
	return e.rect.GetCenter()
}

// GetDepth returns the depth of the Entity by delegating to the underlying Rect's GetDepth method.
func (e *Entity) GetDepth() float64 {
	return e.rect.GetDepth()
}

// HasCollision checks if the current entity's rectangle intersects with another entity's rectangle and returns true if they overlap.
func (e *Entity) HasCollision(obj2 *Entity) bool {
	return e.rect.IntersectRect(obj2.rect)
}

// Distance calculates the 3D Euclidean distance between the calling Entity and the provided collider Entity.
func (e *Entity) Distance(collider *Entity) float64 {
	c1x, c1y, c1z := e.rect.GetCenter()
	c2x, c2y, c2z := collider.rect.GetCenter()
	return CalcDistance(c1x, c1y, c1z, c2x, c2y, c2z)
}

func (e *Entity) DistanceSq(other *Entity) float64 {
	dx := e.rect.point.x + (e.rect.size.w / 2.0) - (other.rect.point.x + (other.rect.size.w / 2.0))
	dy := e.rect.point.y + (e.rect.size.h / 2.0) - (other.rect.point.y + (other.rect.size.h / 2.0))
	return dx*dx + dy*dy
}

// GetZRange restituisce la quota minima (piedi) e massima (testa) dell'entità.
func (e *Entity) GetZRange() (float64, float64) {
	return e.rect.point.z, e.rect.point.z + e.rect.size.d
}

// SetupCollision establishes a collision relationship between the current entity and another entity, resolving overlaps and forces.
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

// isMoving determines if the entity is in motion by checking if any of its velocity components (vx, vy, vz) are non-zero.
func (e *Entity) isMoving() bool {
	return e.vx != 0 || e.vy != 0 || e.vz != 0
}

// clearCollider removes the current collision reference from the entity and unlinks it bi-directionally if necessary.
func (e *Entity) clearCollider() {
	if e.collider != nil {
		if e.collider.collider == e {
			e.collider.collider = nil
		}
		e.collider = nil
	}
}
