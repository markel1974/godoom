package physics

import (
	"math"
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

// Rebuild updates the bounding box position and dimensions and refreshes the cuboid inertia tensor.
func (e *Entity) Rebuild(x, y, z, w, h, d float64) {
	e.BoundingBox.Rebuild(x, y, z, w, h, d)
	e.Cinematic.UpdateInertia(w, h, d)
}

// SetSize updates the bounding box dimensions and refreshes the cuboid inertia tensor.
func (e *Entity) SetSize(w, h, d float64) {
	e.BoundingBox.SetSize(w, h, d)
	e.Cinematic.UpdateInertia(w, h, d)
}

// AddSize increments the bounding box dimensions and refreshes the cuboid inertia tensor.
func (e *Entity) AddSize(w, h, d float64) {
	e.BoundingBox.AddSize(w, h, d)
	e.Cinematic.UpdateInertia(e.GetWidth(), e.GetHeight(), e.GetDepth())
}

// AddForceAtPoint applies a linear force (fx, fy, fz) at world point (px, py, pz) and generates 3D torque on the OBB.
func (e *Entity) AddForceAtPoint(fx, fy, fz, px, py, pz float64) {
	e.AddForce(fx, fy, fz)
	if e.GetInvMass() == 0.0 {
		return
	}
	cx, cy, cz := e.GetCenter()
	rx, ry, rz := px-cx, py-cy, pz-cz

	hx, hy, hz := e.GetSizeCenter()
	a0, a1, a2 := e.GetAxes()

	lx := math.Max(-hx, math.Min(hx, rx*a0[0]+ry*a0[1]+rz*a0[2]))
	ly := math.Max(-hy, math.Min(hy, rx*a1[0]+ry*a1[1]+rz*a1[2]))
	lz := math.Max(-hz, math.Min(hz, rx*a2[0]+ry*a2[1]+rz*a2[2]))

	rx = a0[0]*lx + a1[0]*ly + a2[0]*lz
	ry = a0[1]*lx + a1[1]*ly + a2[1]*lz
	rz = (a0[2]*lx + a1[2]*ly + a2[2]*lz) * 0.20

	tx := ry*fz - rz*fy
	ty := rz*fx - rx*fz
	tz := rx*fy - ry*fx

	// Determine the struck horizontal OBB face normal to include surface normal pressure torque
	nxRatio, nyRatio := 0.0, 0.0
	if hx > 1e-6 {
		nxRatio = math.Abs(lx) / hx
	}
	if hy > 1e-6 {
		nyRatio = math.Abs(ly) / hy
	}

	var fnX, fnY, fnZ float64
	if nxRatio >= nyRatio {
		s := 1.0
		if lx < 0 {
			s = -1.0
		}
		fnX, fnY, fnZ = s*a0[0], s*a0[1], s*a0[2]
	} else {
		s := 1.0
		if ly < 0 {
			s = -1.0
		}
		fnX, fnY, fnZ = s*a1[0], s*a1[1], s*a1[2]
	}

	if fDotN := fx*fnX + fy*fnY + fz*fnZ; fDotN < 0.0 {
		fnFx, fnFy, fnFz := fDotN*fnX, fDotN*fnY, fDotN*fnZ
		tx += (ry*fnFz - rz*fnFy) * 0.25
		ty += (rz*fnFx - rx*fnFz) * 0.25
		tz += rx*fnFy - ry*fnFx
	}

	// Ensure even near-center hits produce a subtle, natural Yaw twist without spinning wildly
	fHoriz := math.Hypot(fx, fy)
	rHoriz := math.Max(hx, hy)
	minYawTorque := 0.12 * rHoriz * fHoriz
	if math.Abs(tz) < minYawTorque {
		sign := 1.0
		if math.Abs(tz) > 1e-6 {
			if tz < 0.0 {
				sign = -1.0
			}
		} else if fx*a0[1]-fy*a0[0] < 0.0 {
			sign = -1.0
		}
		tz = sign * minYawTorque
	}

	const torqueScale = 0.12
	e.AddTorque(tx*torqueScale, ty*torqueScale, tz*torqueScale)
}

// Update integrates linear and angular velocities, applies righting torque on Roll/Pitch, and updates the OBB quaternion.
func (e *Entity) Update() {
	if e.GetInvMass() == 0.0 {
		e.Cinematic.Update()
		return
	}

	_, _, a2 := e.GetAxes()
	tiltSq := a2[0]*a2[0] + a2[1]*a2[1]
	isTilted := tiltSq > 1e-7 || a2[2] < 0.999999

	if isTilted {
		stiffness := 140.0
		damping := 16.0
		if !e.IsOnGround() {
			stiffness = 60.0
			damping = 10.0
		}
		errX := a2[1]
		errY := -a2[0]
		if a2[2] < 0.0 {
			lenXY := math.Hypot(errX, errY)
			if lenXY > 1e-6 {
				errX /= lenXY
				errY /= lenXY
			} else {
				errX = 1.0
			}
		}
		dt := e.GetDt()
		e.wx += (errX*stiffness - e.wx*damping) * dt
		e.wy += (errY*stiffness - e.wy*damping) * dt
	}

	e.Cinematic.Update()

	if e.IsRotating() || isTilted {
		q := e.BoundingBox.GetQuaternion()
		q.IntegrateAngular(e.wx, e.wy, e.wz, e.GetDt())
		e.BoundingBox.SyncRotation()

		// Clamp maximum tilt angle (~18 degrees) so objects rock dynamically when hit
		// without ever flipping sideways/upside-down or getting stuck in an unnatural pose.
		_, _, a2Post := e.GetAxes()
		tiltSin := math.Hypot(a2Post[0], a2Post[1])
		const maxTiltSin = 0.30
		if tiltSin > maxTiltSin || a2Post[2] < 0.95 {
			yaw := e.GetYaw()
			var qYaw, qTilt Quaternion
			qYaw.SetAxisAngle(0.0, 0.0, 1.0, yaw)
			var txAxis, tyAxis float64
			if tiltSin > 1e-6 {
				txAxis = -a2Post[1] / tiltSin
				tyAxis = a2Post[0] / tiltSin
			} else {
				txAxis = 1.0
			}
			qTilt.SetAxisAngle(txAxis, tyAxis, 0.0, math.Asin(maxTiltSin))
			qTilt.Multiplies(&qYaw, q)
			q.Normalize()
			e.BoundingBox.SyncRotation()
			// Zero outward Roll/Pitch angular velocity
			if e.wx*txAxis+e.wy*tyAxis > 0.0 {
				e.wx = 0.0
				e.wy = 0.0
			}
		} else if tiltSin < 0.005 && math.Abs(e.wx) < 0.05 && math.Abs(e.wy) < 0.05 {
			e.wx = 0.0
			e.wy = 0.0
			e.SetRotationAngleZ(e.GetYaw())
		}
	}
}

// ResolveImpact resolves a collision impact with another entity using the supplied collision normal,
// applying both linear and rotational impulses.
func (e *Entity) ResolveImpact(e2 *Entity, nx, ny, nz float64) {
	jx, jy, jz, ok := e.Cinematic.ResolveImpact(e2.Cinematic, nx, ny, nz)
	if !ok {
		return
	}
	cx1, cy1, cz1 := e.GetCenter()
	ex1, ey1, ez1 := e.GetOrientedHalfExtents()
	cx2, cy2, cz2 := e2.GetCenter()
	ex2, ey2, ez2 := e2.GetOrientedHalfExtents()

	minX := math.Max(cx1-ex1, cx2-ex2)
	maxX := math.Min(cx1+ex1, cx2+ex2)
	minY := math.Max(cy1-ey1, cy2-ey2)
	maxY := math.Min(cy1+ey1, cy2+ey2)
	minZ := math.Max(cz1-ez1, cz2-ez2)
	maxZ := math.Min(cz1+ez1, cz2+ez2)

	px := math.Max(cx1-ex1, math.Min(cx1+ex1, (minX+maxX)*0.5))
	py := math.Max(cy1-ey1, math.Min(cy1+ey1, (minY+maxY)*0.5))
	pz := math.Max(cz1-ez1, math.Min(cz1+ez1, (minZ+maxZ)*0.5))

	if e.GetInvMass() > 0.0 && e2.GetInvMass() > 0.0 {
		e.AddImpulseAtOffset(jx, jy, jz, px-cx1, py-cy1, pz-cz1)
		e2.AddImpulseAtOffset(-jx, -jy, -jz, px-cx2, py-cy2, pz-cz2)
	}
}

// ResolveElasticPenetration separates overlapping entities using an elastic, inverse-mass-weighted correction.
// It honors penetration slop and avoids pushing grounded entities downward.
func (e *Entity) ResolveElasticPenetration(other *Entity, otherFloor bool, nx, ny, nz float64, penetration float64) {
	p1, p2, ok := e.Cinematic.ResolveElasticPenetration(other.Cinematic, otherFloor, nx, ny, nz, penetration)
	if !ok {
		return
	}
	e.AddTo(nx*p1, ny*p1, nz*p1)
	other.AddTo(-nx*p2, -ny*p2, -nz*p2)
}

// ResolveInelasticPenetration moves the entity along the collision normal to clear an inelastic overlap.
// It ignores nonpositive penetration and applies vertical slop for upward corrections.
func (e *Entity) ResolveInelasticPenetration(nx, ny, nz float64, pen float64) {
	p, ok := e.Cinematic.ResolveInelasticPenetration(nz, pen)
	if !ok {
		return
	}
	e.AddTo(nx*p, ny*p, nz*p)
}
