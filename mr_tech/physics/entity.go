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
	e.Cinematic.SyncWorldInertia(e.BoundingBox.GetAxes())
}

// GetSize returns the width, height, and depth of the entity's bounding box as three float64 values.
func (e *Entity) GetSize() (float64, float64, float64) {
	return e.BoundingBox.GetSize()
}

// SetSize updates the bounding box dimensions and refreshes the cuboid inertia tensor.
func (e *Entity) SetSize(w, h, d float64) {
	e.BoundingBox.SetSize(w, h, d)
	e.Cinematic.UpdateInertia(w, h, d)
	e.Cinematic.SyncWorldInertia(e.BoundingBox.GetAxes())
}

// AddSize increments the bounding box dimensions and refreshes the cuboid inertia tensor.
func (e *Entity) AddSize(w, h, d float64) {
	e.BoundingBox.AddSize(w, h, d)
	e.Cinematic.UpdateInertia(e.GetWidth(), e.GetHeight(), e.GetDepth())
	e.Cinematic.SyncWorldInertia(e.BoundingBox.GetAxes())
}

// SetRotationAngleZ sets the OBB orientation to a yaw rotation around Z and synchronizes the world inertia tensor.
func (e *Entity) SetRotationAngleZ(angleRad float64) {
	e.BoundingBox.SetRotationAngleZ(angleRad)
	e.Cinematic.SyncWorldInertia(e.BoundingBox.GetAxes())
}

// SetRotationEuler sets the OBB orientation from Euler angles and synchronizes the world inertia tensor.
func (e *Entity) SetRotationEuler(rollX, pitchY, yawZ float64) {
	e.BoundingBox.SetRotationEuler(rollX, pitchY, yawZ)
	e.Cinematic.SyncWorldInertia(e.BoundingBox.GetAxes())
}

// SyncRotation updates the OBB world axes and synchronizes the world-space inverse inertia tensor.
func (e *Entity) SyncRotation() {
	e.BoundingBox.SyncRotation()
	e.Cinematic.SyncWorldInertia(e.BoundingBox.GetAxes())
}

// AddForceAtPoint applies a linear force (fx, fy, fz) at world point (px, py, pz) and generates 3D torque τ = r × F on the OBB
// using exact Ray-OBB slab intersection and the world-space inverse inertia tensor I_world^-1 = R * I_body^-1 * R^T.
func (e *Entity) AddForceAtPoint(fx, fy, fz, px, py, pz float64) {
	e.AddForce(fx, fy, fz)
	if e.GetInvMass() == 0.0 {
		return
	}
	fMag := math.Sqrt(fx*fx + fy*fy + fz*fz)
	if fMag <= 1e-9 {
		return
	}
	invF := 1.0 / fMag

	cx, cy, cz := e.GetCenter()
	r0x, r0y, r0z := px-cx, py-cy, pz-cz

	hx, hy, hz := e.GetSizeCenter()
	a00, a01, a02, a10, a11, a12, a20, a21, a22 := e.GetAxes()

	// Project impact point and ray direction into OBB local frame
	plx := r0x*a00 + r0y*a01 + r0z*a02
	ply := r0x*a10 + r0y*a11 + r0z*a12
	plz := r0x*a20 + r0y*a21 + r0z*a22

	dlx := (fx*a00 + fy*a01 + fz*a02) * invF
	dly := (fx*a10 + fy*a11 + fz*a12) * invF
	dlz := (fx*a20 + fy*a21 + fz*a22) * invF

	// Shift to closest point on the ray to OBB center, clamped within the OBB interior
	tProj := plx*dlx + ply*dly + plz*dlz
	plx = math.Max(-hx*0.98, math.Min(hx*0.98, plx-tProj*dlx))
	ply = math.Max(-hy*0.98, math.Min(hy*0.98, ply-tProj*dly))
	plz = math.Max(-hz*0.98, math.Min(hz*0.98, plz-tProj*dlz))

	// March backward along -dl to find the exact entry point on the struck OBB face
	tBack := math.MaxFloat64
	hitAxis := 0
	hitSign := 1.0
	if math.Abs(dlx) > 1e-6 {
		s := -1.0
		if dlx < 0 {
			s = 1.0
		}
		if tb := (plx - s*hx) / dlx; tb >= 0.0 && tb < tBack {
			tBack = tb
			hitAxis = 0
			hitSign = s
		}
	}
	if math.Abs(dly) > 1e-6 {
		s := -1.0
		if dly < 0 {
			s = 1.0
		}
		if tb := (ply - s*hy) / dly; tb >= 0.0 && tb < tBack {
			tBack = tb
			hitAxis = 1
			hitSign = s
		}
	}
	if math.Abs(dlz) > 1e-6 {
		s := -1.0
		if dlz < 0 {
			s = 1.0
		}
		if tb := (plz - s*hz) / dlz; tb >= 0.0 && tb < tBack {
			tBack = tb
			hitAxis = 2
			hitSign = s
		}
	}
	if tBack < math.MaxFloat64 {
		plx -= tBack * dlx
		ply -= tBack * dly
		plz -= tBack * dlz
	}

	// Reconstruct world-space lever arm r from OBB center to surface entry point
	rx := a00*plx + a10*ply + a20*plz
	ry := a01*plx + a11*ply + a21*plz
	rz := (a02*plx + a12*ply + a22*plz) * 0.25

	// Direct moment of force: τ = r × F
	tx := ry*fz - rz*fy
	ty := rz*fx - rx*fz
	tz := rx*fy - ry*fx

	// Normal pressure moment on the struck OBB face: τ_n = r × F_n
	var fnX, fnY, fnZ float64
	switch hitAxis {
	case 0:
		fnX, fnY, fnZ = hitSign*a00, hitSign*a01, hitSign*a02
	case 1:
		fnX, fnY, fnZ = hitSign*a10, hitSign*a11, hitSign*a12
	default:
		fnX, fnY, fnZ = hitSign*a20, hitSign*a21, hitSign*a22
	}
	if fDotN := fx*fnX + fy*fnY + fz*fnZ; fDotN < 0.0 {
		fnFx, fnFy, fnFz := fDotN*fnX, fDotN*fnY, fDotN*fnZ
		tx += (ry*fnFz - rz*fnFy) * 0.25
		ty += (rz*fnFx - rx*fnFz) * 0.25
		tz += rx*fnFy - ry*fnFx
	}

	e.AddTorque(tx, ty, tz)
}

// Update integrates linear and angular velocities, applies righting torque on Roll/Pitch, and updates the OBB quaternion.
func (e *Entity) Update() {
	if e.GetInvMass() == 0.0 {
		e.Cinematic.Update()
		return
	}

	a20, a21, a22 := e.GetAxes2()
	tiltSq := a20*a20 + a21*a21
	isTilted := tiltSq > 1e-7 || a22 < 0.999999

	if isTilted {
		stiffness := 140.0
		damping := 16.0
		if !e.IsOnGround() {
			stiffness = 60.0
			damping = 10.0
		}
		errX := a21
		errY := -a20
		if a22 < 0.0 {
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
		e.SyncRotation()

		// Clamp maximum tilt angle (~18 degrees) so objects rock dynamically when hit
		// without ever flipping sideways/upside-down or getting stuck in an unnatural pose.
		a2Post0, a2Post1, a2Post2 := e.GetAxes2()
		tiltSin := math.Hypot(a2Post0, a2Post1)
		const maxTiltSin = 0.30
		if tiltSin > maxTiltSin || a2Post2 < 0.95 {
			yaw := e.GetYaw()
			var qYaw, qTilt Quaternion
			qYaw.SetAxisAngle(0.0, 0.0, 1.0, yaw)
			var txAxis, tyAxis float64
			if tiltSin > 1e-6 {
				txAxis = -a2Post1 / tiltSin
				tyAxis = a2Post0 / tiltSin
			} else {
				txAxis = 1.0
			}
			qTilt.SetAxisAngle(txAxis, tyAxis, 0.0, math.Asin(maxTiltSin))
			qTilt.Multiplies(&qYaw, q)
			q.Normalize()
			e.SyncRotation()
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

// ResolveImpact resolves a 6-DoF collision impact with another entity using the supplied collision normal,
// computing the contact manifold point on the OBBs and applying both linear and rotational impulses.
func (e *Entity) ResolveImpact(e2 *Entity, nx, ny, nz float64) {
	cx1, cy1, cz1 := e.GetCenter()
	ex1, ey1, ez1 := e.GetOrientedHalfExtents()

	var r1x, r1y, r1z float64
	var r2x, r2y, r2z float64

	if e2.GetInvMass() > 0.0 {
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

		r1x, r1y, r1z = px-cx1, py-cy1, (pz-cz1)*0.25
		r2x, r2y, r2z = px-cx2, py-cy2, (pz-cz2)*0.25
	} else if math.Abs(nz) < 0.7 {
		// Impact against static wall/obstacle: contact point on OBB perimeter along -n
		r1x = -nx * ex1
		r1y = -ny * ey1
		r1z = -nz * ez1 * 0.25
	}

	e.Cinematic.ResolveImpact(e2.Cinematic, nx, ny, nz, r1x, r1y, r1z, r2x, r2y, r2z)
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
