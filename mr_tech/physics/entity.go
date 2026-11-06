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

const defaultPenetrationSlop = 0.02
const defaultPositionalPercent = 0.3

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

// ResolveImpact resolves a collision impact with another entity using the supplied collision normal.
func (e *Entity) ResolveImpact(e2 *Entity, nx, ny, nz float64) {
	e.Cinematic.ResolveImpact(e2.Cinematic, nx, ny, nz)
}

// ResolveElasticPenetration separates overlapping entities using an elastic, inverse-mass-weighted correction.
// It honors penetration slop and avoids pushing grounded entities downward.
func (e *Entity) ResolveElasticPenetration(other *Entity, otherFloor bool, nx, ny, nz float64, penetration float64) {
	depth := penetration - defaultPenetrationSlop
	if depth <= 0.0 {
		return
	}
	if nz > defaultPenetrationDepth {
		// 'other' è sotto 'e': se 'other' poggia a terra, non può scendere verso il basso
		if otherFloor || other.IsOnGround() {
			e.AddTo(nx*depth, ny*depth, nz*depth)
			//other.AddTo(-nx*0.0, -ny*0.0, -nz*0.0)
			return
		}
	} else if nz < -defaultPenetrationDepth {
		// 'other' è sopra 'e': se 'e' poggia a terra, non deve essere compresso in basso
		if e.IsOnGround() {
			//e.AddTo(nx*0.0, ny*0.0, nz*0.0
			other.AddTo(-nx*depth, -ny*depth, -nz*depth)
			return
		}
	}
	correction := depth * defaultPositionalPercent
	invMass1 := e.GetInvMass()
	invMass2 := other.GetInvMass()
	invMassSum := invMass1 + invMass2
	if invMassSum <= 0.0 {
		return
	}
	p1 := correction * (invMass1 / invMassSum)
	e.AddTo(nx*p1, ny*p1, nz*p1)
	p2 := correction * (invMass2 / invMassSum)
	other.AddTo(-nx*p2, -ny*p2, -nz*p2)
}

// ResolveInelasticPenetration moves the entity along the collision normal to clear an inelastic overlap.
// It ignores nonpositive penetration and applies vertical slop for upward corrections.
func (e *Entity) ResolveInelasticPenetration(nx, ny, nz float64, pen float64) {
	if pen <= 0.0 {
		return
	}
	if nz > defaultPenetrationDepth {
		if pen -= defaultPenetrationSlop; pen <= 0.0 {
			return
		}
	}
	e.AddTo(nx*pen, ny*pen, nz*pen)
}
