package physics

import (
	"sync/atomic"
)

var _dt float64

func SetFps(fps uint64) {
	_dt = 1.0 / (float64(fps) / 2.0)
}

// _globalId is an internal counter used to generate unique identifiers in a thread-safe manner.
var _globalId int64

// GetGlobalId generates and returns a unique globally incremental identifier using atomic operations.
func GetGlobalId() int64 {
	return atomic.AddInt64(&_globalId, 1)
}

func init() {
	_globalId = -1
	SetFps(120)
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
		Cinematic:   NewCinematic(_dt, mass, restitution, groundFriction, gForce),
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
