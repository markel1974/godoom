package physics

import (
	"sync/atomic"
)

var _dt float64
var _fps uint64

// SetFps sets the desired frames per second (fps) and adjusts the internal frame time interval (_dt) accordingly.
func SetFps(fps uint64) {
	_fps = fps
	_dt = 1.0 / float64(fps) //_dt = 1.0 / (float64(fps) * 0.5)
}

// GetFps returns the current frames-per-second (FPS) value as an unsigned 64-bit integer.
func GetFps() uint64 {
	return _fps
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

const defaultPenetrationSlop = 0.02
const defaultPositionalPercent = 0.3

// ResolvePenetration resolves positional overlap between two entities or against a static obstacle.
func (e *Entity) ResolvePenetration(other *Entity, nx, ny, nz float64, penetration float64) {
	depth := penetration - defaultPenetrationSlop
	if depth <= 0.0 {
		return
	}
	correction := depth * defaultPositionalPercent

	if other != nil {
		invMass1 := e.GetInvMass()
		invMass2 := other.GetInvMass()
		invMassSum := invMass1 + invMass2
		if invMassSum <= 0.0 {
			return
		}

		p1 := correction * (invMass1 / invMassSum)
		p2 := correction * (invMass2 / invMassSum)

		// Se l'entità sottostante è appoggiata al suolo e stiamo sopra di essa (+Z),
		// scarichiamo l'intera correzione verso l'alto evitando di spingerla nel pavimento.
		if other.IsOnGround() && nz > 0 {
			p1 = correction
			p2 = 0.0
		}

		e.AddTo(nx*p1, ny*p1, nz*p1)
		other.AddTo(-nx*p2, -ny*p2, -nz*p2)
	} else {
		e.AddTo(nx*correction, ny*correction, nz*correction)
	}
}
