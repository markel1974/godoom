package physics

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/utils"
)

// friction represents the default friction coefficient used to simulate resistance in motion calculations.
const (
	friction = 0.9
)

// CalcDistance calculates the Euclidean distance between two points (x1, y1) and (x2, y2)
func CalcDistance(x1, y1, x2, y2 float64) float64 {
	dx := x2 - x1
	dy := y2 - y1
	d := dx*dx + dy*dy
	// Valuta la singolarità spaziale (compenetrazione perfetta) PRIMA di invocare l'istruzione hardware
	if d < 0.0001 {
		return 0.01
	}
	// Sfrutta l'intrinseco hardware (SQRTSD su amd64/arm64) senza branch successivi
	return math.Sqrt(d)
}

// Entity represents a physics-based entity with properties for position, velocity, mass, and collision handling.
type Entity struct {
	Rect
	Id       string
	Mass     float64
	VxMin    float64
	Vx       float64
	VyMin    float64
	Vy       float64
	Friction float64
	G        float64
	GForce   float64
	impulse  float64
	Collider *Entity
}

// NewEntity creates and returns a new Entity instance with the specified position, dimensions, and mass.
func NewEntity(x float64, y float64, w float64, h float64, mass float64) *Entity {
	a := &Entity{
		Id:       utils.NextUUId(),
		Rect:     NewRect(x, y, w, h, 1.0),
		Mass:     mass,
		Vx:       0.0,
		Vy:       0.0,
		Friction: friction,
		G:        0.0,
		GForce:   0.0,
		VxMin:    0.001,
		VyMin:    0.001,
		impulse:  0.001,
	}
	return a
}

// Reset reinitializes the Entity's position, size, and z-index with the given parameters.
func (e *Entity) Reset(x float64, y float64, w float64, h float64, z float64, mass float64) {
	e.point.x = x
	e.point.y = y
	e.size.w = w
	e.size.h = h
	e.z = z
	e.Mass = mass
	e.Vx = 0.0
	e.Vy = 0.0
	e.Friction = friction
	e.G = 0.0
	e.GForce = 0.0
	e.VxMin = 0.001
	e.VyMin = 0.001
	e.impulse = 0.001
}

// GetId returns the unique identifier (Id) of the Entity as a string.
func (e *Entity) GetId() string {
	return e.Id
}

// Invalidate clears the collider association for the entity by resetting its collider reference.
func (e *Entity) Invalidate() {
	e.clearCollider()
}

// HasCollision checks if the current entity's rectangle intersects with another entity's rectangle and returns true if they overlap.
func (e *Entity) HasCollision(obj2 *Entity) bool {
	return e.rectIntersect(e.point.x, e.point.y, e.size.w, e.size.h, obj2.point.x, obj2.point.y, obj2.size.w, obj2.size.h)
}

// rectIntersect checks if two rectangles, defined by their top-left coordinates and dimensions, intersect. Returns true if overlapping.
func (e *Entity) rectIntersect(x1 float64, y1 float64, w1 float64, h1 float64, x2 float64, y2 float64, w2 float64, h2 float64) bool {
	if x2 > w1+x1 || x1 > w2+x2 || y2 > h1+y1 || y1 > h2+y2 {
		return false
	}
	return true
}

// Distance calculates and returns the Euclidean distance between the calling Entity and the provided collider Entity.
func (e *Entity) Distance(collider *Entity) float64 {
	distance := CalcDistance(e.center.x, e.center.y, collider.center.x, collider.center.y)
	return distance
}

// SetupCollision resolves the collision between the current entity and another by adjusting velocities based on impulse and mass.
func (e *Entity) SetupCollision(collider *Entity) {
	e.Collider = collider
	collider.Collider = e

	// 1. Collision vector and normal
	distance := e.Distance(collider)
	vecCollision := Point{x: collider.center.x - e.center.x, y: collider.center.y - e.center.y}
	vecCollisionNorm := Point{x: vecCollision.x / distance, y: vecCollision.y / distance}

	// 2. Exact calculation of relative velocity
	relVx := collider.Vx - e.Vx
	relVy := collider.Vy - e.Vy

	// 3. Dot product between relative velocity and collision normal
	vRelDotN := relVx*vecCollisionNorm.x + relVy*vecCollisionNorm.y

	// 4. Early exit: If vRelDotN > 0, entities are already moving apart.
	// We only resolve positional penetration, no impulse transfer!
	if vRelDotN < 0 {
		// Coefficient of restitution (1.0 = perfectly elastic, 0.0 = inelastic)
		restitution := 1.0 // Can be modulated based on material

		// Impulse magnitude according to Newtonian dynamics
		j := -(1.0 + restitution) * vRelDotN
		j /= (1.0 / e.Mass) + (1.0 / collider.Mass)

		// Apply impulsive forces
		e.Vx -= (j / e.Mass) * vecCollisionNorm.x
		e.Vy -= (j / e.Mass) * vecCollisionNorm.y
		collider.Vx += (j / collider.Mass) * vecCollisionNorm.x
		collider.Vy += (j / collider.Mass) * vecCollisionNorm.y
	}

	// 5. Positional Projection (Baumgarte Stabilization)
	if penetrationDepth := (e.GetWidth()/2 + collider.GetWidth()/2) - distance; penetrationDepth > 0 {
		percent := 0.2
		slop := 0.01
		correction := (math.Max(penetrationDepth-slop, 0.0) / (e.Mass + collider.Mass)) * percent
		e.AddTo(-vecCollisionNorm.x*correction*collider.Mass, -vecCollisionNorm.y*correction*collider.Mass)
		collider.AddTo(vecCollisionNorm.x*correction*e.Mass, vecCollisionNorm.y*correction*e.Mass)
	}
}

// SetupInelasticCollision configures inelastic collision properties for the entity with the given collider.
func (e *Entity) SetupInelasticCollision(collider *Entity) {
	e.Collider = collider
	if collider != nil {
		collider.Collider = e
	}
	e.Friction = 0.7
	e.Vx = -e.Vx
	e.Vy = -e.Vy
	e.G = 0.0
}

// isMoving determines if the entity is currently in motion by checking if its velocity along both axes is non-zero.
func (e *Entity) isMoving() bool {
	if e.Vx == 0 && e.Vy == 0 {
		return false
	}
	return true
}

// hit determines if the current entity is in collision with another entity based on their centers and widths.
func (e *Entity) hit(collider *Entity) bool {
	if collider == nil {
		return false
	}
	distance := CalcDistance(e.center.x, e.center.y, collider.center.x, collider.center.y)
	if distance > collider.GetWidth() {
		return false
	}
	return true
}

// clearCollider removes the reference to the current entity's collider and clears the reciprocal reference if it exists.
func (e *Entity) clearCollider() {
	if e.Collider != nil {
		if e.Collider.Collider == e {
			e.Collider.Collider = nil
		}
		e.Collider = nil
	}
}

// Update updates the entity's velocity and state based on collisions, friction, and movement, returning its active state.
func (e *Entity) Update() bool {
	if e.Collider != nil {
		if distance := e.Distance(e.Collider); distance >= e.GetWidth()/2+e.Collider.GetWidth()/2 {
			e.clearCollider()
		}
	}
	if !e.isMoving() {
		e.G = 0.0
		return false
	}
	e.Vx *= e.Friction
	e.Vy *= e.Friction
	if math.Abs(e.Vx) < e.VxMin {
		e.Vx = 0.0
	}
	if math.Abs(e.Vy) < e.VyMin {
		e.Vy = 0.0
	}
	if !e.isMoving() {
		e.G = 0.0
		return false
	}
	e.G = e.calcG()
	return true
}

// MoveTest calculates the new position of the Entity by adding its velocity to its current coordinates and returns the result.
func (e *Entity) MoveTest() (float64, float64) {
	x := e.point.x + e.Vx
	y := e.point.y + e.Vy
	return x, y
}

// Move updates the position of the entity by adding its velocity components (Vx, Vy) to its current coordinates.
func (e *Entity) Move() {
	e.AddTo(e.Vx, e.Vy)
}

// calcG calculates a derived value based on the absolute velocities (Vx, Vy) and GForce of the given Entity.
// Returns 0.0 if GForce is zero or the computed value otherwise.
func (e *Entity) calcG() float64 {
	if e.GForce == 0.0 {
		return 0.0
	}
	return (math.Abs(e.Vx) + math.Abs(e.Vy)) * e.GForce
}
