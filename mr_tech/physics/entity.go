package physics

import "sync/atomic"

// dt60 defines the fixed time step duration (in seconds) for 60 frames per second simulations.
// dt120 defines the fixed time step duration (in seconds) for 120 frames per second simulations.
const (
	// dt60 represents the fixed time step duration in seconds, commonly used for 60 frames per second simulations.
	dt60 float64 = 1.0 / 60.0

	// dt120 represents the fixed time step duration equivalent to 1/120th of a second.
	dt120 float64 = 1.0 / 120.0
)

// _globalId is an internal counter used to generate unique identifiers in a thread-safe manner.
var _globalId int64 = -1

// GetGlobalId generates and returns a unique globally incremental identifier using atomic operations.
func GetGlobalId() int64 {
	return atomic.AddInt64(&_globalId, 1)
}

// Entity represents a game object with a unique identifier, bounding box, and cinematic behaviors.
type Entity struct {
	id uint64
	*BoundingBox
	*Cinematic
}

// NewEntity creates and returns a pointer to a new Entity with specified mass, restitution, ground friction, and gravity force.
func NewEntity(mass, restitution, groundFriction, gForce float64) *Entity {
	e := &Entity{
		id:          uint64(GetGlobalId()),
		BoundingBox: NewBoundingBox(0, 0, 0, 0, 0, 0),
		Cinematic:   NewCinematic(dt60, mass, restitution, groundFriction, gForce),
	}
	return e
}

// GetId returns the unique identifier of the Entity as a uint64.
func (e *Entity) GetId() uint64 {
	return e.id
}

// ResolveImpact resolves a collision between two entities by applying impact forces based on their cinematic properties.
func (e *Entity) ResolveImpact(e2 *Entity, nx, ny, nz float64, penetration float64) {
	e.Cinematic.ResolveImpact(e2.Cinematic, nx, ny, nz, penetration)
}
