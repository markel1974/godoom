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
	rect     Rect
	id       string
	mass     float64
	vxMin    float64
	vx       float64
	vyMin    float64
	vy       float64
	friction float64
	g        float64
	gForce   float64
	impulse  float64
	collider *Entity
}

// NewEntity creates and returns a new Entity instance with the specified position, dimensions, and mass.
func NewEntity(x float64, y float64, w float64, h float64, mass float64) *Entity {
	a := &Entity{
		id:       utils.NextUUId(),
		rect:     NewRect(x, y, w, h, 1.0),
		mass:     mass,
		vx:       0.0,
		vy:       0.0,
		friction: friction,
		g:        0.0,
		gForce:   0.0,
		vxMin:    0.001,
		vyMin:    0.001,
		impulse:  0.001,
	}
	return a
}

// Reset reinitializes the Entity's position, size, and z-index with the given parameters.
func (e *Entity) Reset(x float64, y float64, w float64, h float64, z float64, mass float64) {
	e.rect.point.x = x
	e.rect.point.y = y
	e.rect.size.w = w
	e.rect.size.h = h
	e.rect.z = z
	e.mass = mass
	e.vx = 0.0
	e.vy = 0.0
	e.friction = friction
	e.g = 0.0
	e.gForce = 0.0
	e.vxMin = 0.001
	e.vyMin = 0.001
	e.impulse = 0.001
	e.rect.rebuild()
}

// SetFriction sets the friction coefficient for the entity, which affects its velocity reduction over time.
func (e *Entity) SetFriction(friction float64) {
	e.friction = friction
}

// SetGForce sets the gravitational force multiplier affecting the entity, influencing its vertical acceleration.
func (e *Entity) SetGForce(gForce float64) {
	e.gForce = gForce
}

// SetG updates the gravitational value (g) of the Entity, affecting its behavior in physics-based calculations.
func (e *Entity) SetG(g float64) {
	e.g = g
}

// GetVx returns the current horizontal velocity (Vx) of the Entity as a float64.
func (e *Entity) GetVx() float64 {
	return e.vx
}

// GetVy returns the vertical velocity (vy) of the Entity as a float64.
func (e *Entity) GetVy() float64 {
	return e.vy
}

// SetVx sets the horizontal velocity (Vx) of the Entity to the specified value.
func (e *Entity) SetVx(vx float64) {
	e.vx = vx
}

// SetVy sets the vertical velocity (Vy) of the Entity to the specified value.
func (e *Entity) SetVy(vy float64) {
	e.vy = vy
}

// GetId returns the unique identifier (Id) of the Entity as a string.
func (e *Entity) GetId() string {
	return e.id
}

// Invalidate clears the collider association for the entity by resetting its collider reference.
func (e *Entity) Invalidate() {
	e.clearCollider()
}

func (e *Entity) GetWidth() float64 {
	return e.rect.GetWidth()
}

func (e *Entity) GetAABB() *AABB {
	return e.rect.GetAABB()
}

func (e *Entity) MoveTo(x float64, y float64) {
	e.rect.MoveTo(x, y)
}

func (e *Entity) GetCenterXY() (float64, float64) {
	return e.rect.GetCenterXY()
}

// HasCollision checks if the current entity's rectangle intersects with another entity's rectangle and returns true if they overlap.
func (e *Entity) HasCollision(obj2 *Entity) bool {
	return e.rect.IntersectRect(obj2.rect)
	//return e.rectIntersect(e.rect.point.x, e.rect.point.y, e.rect.size.w, e.rect.size.h, obj2.rect.point.x, obj2.rect.point.y, obj2.rect.size.w, obj2.rect.size.h)
}

// rectIntersect checks if two rectangles, defined by their top-left coordinates and dimensions, intersect. Returns true if overlapping.
//func (e *Entity) rectIntersect(x1 float64, y1 float64, w1 float64, h1 float64, x2 float64, y2 float64, w2 float64, h2 float64) bool {
//	if x2 > w1+x1 || x1 > w2+x2 || y2 > h1+y1 || y1 > h2+y2 {
//		return false
//	}
//	return true
//}

// Distance calculates and returns the Euclidean distance between the calling Entity and the provided collider Entity.
func (e *Entity) Distance(collider *Entity) float64 {
	distance := CalcDistance(e.rect.center.x, e.rect.center.y, collider.rect.center.x, collider.rect.center.y)
	return distance
}

// SetupCollision resolves the collision between the current entity and another by adjusting velocities based on impulse and mass.
func (e *Entity) SetupCollision(otherEnt *Entity) {
	e.collider = otherEnt
	otherEnt.collider = e

	// 1. Collision vector and normal
	distance := e.Distance(otherEnt)
	vecCollision := Point{x: otherEnt.rect.center.x - e.rect.center.x, y: otherEnt.rect.center.y - e.rect.center.y}
	vecCollisionNorm := Point{x: vecCollision.x / distance, y: vecCollision.y / distance}

	// 2. Exact calculation of relative velocity
	relVx := otherEnt.vx - e.vx
	relVy := otherEnt.vy - e.vy

	// 3. Dot product between relative velocity and collision normal
	vRelDotN := relVx*vecCollisionNorm.x + relVy*vecCollisionNorm.y

	// 4. Early exit: If vRelDotN > 0, entities are already moving apart.
	// We only resolve positional penetration, no impulse transfer!
	if vRelDotN < 0 {
		// Coefficient of restitution (1.0 = perfectly elastic, 0.0 = inelastic)
		restitution := 1.0 // Can be modulated based on material

		// Impulse magnitude according to Newtonian dynamics
		j := -(1.0 + restitution) * vRelDotN
		j /= (1.0 / e.mass) + (1.0 / otherEnt.mass)

		// Apply impulsive forces
		e.vx -= (j / e.mass) * vecCollisionNorm.x
		e.vy -= (j / e.mass) * vecCollisionNorm.y
		otherEnt.vx += (j / otherEnt.mass) * vecCollisionNorm.x
		otherEnt.vy += (j / otherEnt.mass) * vecCollisionNorm.y
	}

	// 5. Positional Projection (Baumgarte Stabilization)
	if penetrationDepth := (e.rect.GetWidth()/2 + otherEnt.rect.GetWidth()/2) - distance; penetrationDepth > 0 {
		percent := 0.2
		slop := 0.01
		correction := (math.Max(penetrationDepth-slop, 0.0) / (e.mass + otherEnt.mass)) * percent
		e.rect.AddTo(-vecCollisionNorm.x*correction*otherEnt.mass, -vecCollisionNorm.y*correction*otherEnt.mass)
		otherEnt.rect.AddTo(vecCollisionNorm.x*correction*e.mass, vecCollisionNorm.y*correction*e.mass)
	}
}

// isMoving determines if the entity is currently in motion by checking if its velocity along both axes is non-zero.
func (e *Entity) isMoving() bool {
	if e.vx == 0 && e.vy == 0 {
		return false
	}
	return true
}

// hit determines if the current entity is in collision with another entity based on their centers and widths.
func (e *Entity) hit(collider *Entity) bool {
	if collider == nil {
		return false
	}
	distance := CalcDistance(e.rect.center.x, e.rect.center.y, collider.rect.center.x, collider.rect.center.y)
	if distance > collider.rect.GetWidth() {
		return false
	}
	return true
}

// clearCollider removes the reference to the current entity's collider and clears the reciprocal reference if it exists.
func (e *Entity) clearCollider() {
	if e.collider != nil {
		if e.collider.collider == e {
			e.collider.collider = nil
		}
		e.collider = nil
	}
}

// Update updates the entity's velocity and state based on collisions, friction, and movement, returning its active state.
func (e *Entity) Update() bool {
	if e.collider != nil {
		if distance := e.Distance(e.collider); distance >= e.rect.GetWidth()/2+e.collider.rect.GetWidth()/2 {
			e.clearCollider()
		}
	}
	if !e.isMoving() {
		return false
	}
	e.vx *= e.friction
	e.vy *= e.friction
	if math.Abs(e.vx) < e.vxMin {
		e.vx = 0.0
	}
	if math.Abs(e.vy) < e.vyMin {
		e.vy = 0.0
	}
	if !e.isMoving() {
		return false
	}
	e.g = e.calcG()
	return true
}

// MoveTest calculates the new position of the Entity by adding its velocity to its current coordinates and returns the result.
func (e *Entity) MoveTest() (float64, float64) {
	x := e.rect.point.x + e.vx
	y := e.rect.point.y + e.vy
	return x, y
}

// Move updates the position of the entity by adding its velocity components (Vx, Vy) to its current coordinates.
func (e *Entity) Move() {
	e.rect.AddTo(e.vx, e.vy)
}

// calcG calculates a derived value based on the absolute velocities (Vx, Vy) and GForce of the given Entity.
// Returns 0.0 if GForce is zero or the computed value otherwise.
func (e *Entity) calcG() float64 {
	if e.gForce == 0.0 {
		return 0.0
	}
	return (math.Abs(e.vx) + math.Abs(e.vy)) * e.gForce
}
