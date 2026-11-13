package aabb

import (
	"math"
)

const maxTiltSin = 0.30

var _maxTiltAngle = math.Asin(maxTiltSin) //, in radians

// obbVector is a fixed-size 3D vector used by the SAT implementation.
type obbVector [3]float64

// BoundingBox represents a 3D oriented bounding box (OBB) and its enclosing axis-aligned bounding box (AABB).
type BoundingBox struct {
	bottomLeft       Point
	bottomCenter     Point
	center           Point
	size             Size
	aabb             *AABB
	axis             [3]obbVector // Local axes transformed into world space.
	qRotation        *Quaternion  // Orientation quaternion.
	qYaw             *Quaternion
	qTilt            *Quaternion
	qYawInverse      *Quaternion
	qYawRelative     *Quaternion
	ex               float64 // Oriented half-extents projected onto world axes.
	ey               float64 // Oriented half-extents projected onto world axes.
	ez               float64 // Oriented half-extents projected onto world axes.
	tiltLimitEnabled bool
}

// NewBoundingBox creates a bounding box with identity rotation.
func NewBoundingBox() *BoundingBox {
	r := &BoundingBox{
		bottomLeft:       NewPoint(0, 0, 0),
		bottomCenter:     NewPoint(0, 0, 0),
		center:           NewPoint(0, 0, 0),
		size:             NewSize(0, 0, 0),
		aabb:             NewAABB(),
		qRotation:        NewQuaternionEmpty(),
		qYaw:             NewQuaternionEmpty(),
		qTilt:            NewQuaternionEmpty(),
		qYawInverse:      NewQuaternionEmpty(),
		qYawRelative:     NewQuaternionEmpty(),
		tiltLimitEnabled: false, //true //MODIFICATO,
	}
	r.qRotation.SetIdentity()
	r.qYaw.SetIdentity()
	r.qTilt.SetIdentity()
	r.rebuild()
	return r
}

// Rebuild updates the reference position and dimensions,
// then recalculates the derived bounding-box data.
func (r *BoundingBox) Rebuild(blx, bly, blz, w, h, d float64) {
	r.bottomLeft.x, r.bottomLeft.y, r.bottomLeft.z = blx, bly, blz
	r.size.w, r.size.h, r.size.d = w, h, d
	r.rebuild()
}

// RebuildCenter updates the bounding box centered at (cx, cy, cz) with dimensions w, h, d, then recalculates the derived bounding-box data.
func (r *BoundingBox) RebuildCenter(cx, cy, cz, w, h, d float64) {
	r.bottomLeft.x, r.bottomLeft.y, r.bottomLeft.z = cx-w*0.5, cy-h*0.5, cz-d*0.5
	r.size.w, r.size.h, r.size.d = w, h, d
	r.rebuild()
}

// SetRotationQuaternion sets the orientation using a quaternion (x, y, z, w). The quaternion is normalized before being stored.
func (r *BoundingBox) SetRotationQuaternion(x, y, z, w float64) {
	r.qRotation.Set(x, y, z, w)
	r.qRotation.Normalize()
	r.syncYawTiltFromRotation()
	r.rebuild()
}

// SetRotationAngleZ sets the orientation to a rotation around the world Z axis by angleRad radians.
func (r *BoundingBox) SetRotationAngleZ(angleRad float64) {
	r.qYaw.SetAxisAngle(0, 0, 1, angleRad)
	r.qTilt.SetIdentity()
	r.composeRotation()
	r.rebuild()
}

// SetRotationEuler sets the orientation from roll (X), pitch (Y), and yaw (Z) angles in radians.
func (r *BoundingBox) SetRotationEuler(rollX, pitchY, yawZ float64) {
	r.qRotation.SetEulerZYX(rollX, pitchY, yawZ)
	r.qRotation.Normalize()
	r.syncYawTiltFromRotation()
	r.rebuild()
}

// SyncRotation recalculates the oriented axes and enclosing AABB after in-place quaternion updates.
func (r *BoundingBox) SyncRotation() {
	r.rebuild()
}

// GetRotationQuaternion returns the orientation in x, y, z, w order.
func (r *BoundingBox) GetRotationQuaternion() (float64, float64, float64, float64) {
	return r.qRotation.Get()
}

// GetQRotation returns a pointer to the internal orientation Quaternion.
func (r *BoundingBox) GetQRotation() *Quaternion {
	return r.qRotation
}

// GetQYaw returns the yaw rotation quaternion for the bounding box.
func (r *BoundingBox) GetQYaw() *Quaternion {
	return r.qYaw
}

// GetQTilt returns the tilt quaternion for the bounding box.
func (r *BoundingBox) GetQTilt() *Quaternion {
	return r.qTilt
}

// GetAxes returns the three orthonormal local axes (X, Y, Z) in world space.
func (r *BoundingBox) GetAxes() (float64, float64, float64, float64, float64, float64, float64, float64, float64) {
	return r.axis[0][0], r.axis[0][1], r.axis[0][2], r.axis[1][0], r.axis[1][1], r.axis[1][2], r.axis[2][0], r.axis[2][1], r.axis[2][2]
}

// GetAxes0 returns the x, y, and z components of the first bounding box axis.
func (r *BoundingBox) GetAxes0() (float64, float64, float64) {
	return r.axis[0][0], r.axis[0][1], r.axis[0][2]
}

// GetAxes1 returns the x, y, and z components of the bounding box's second axis.
func (r *BoundingBox) GetAxes1() (float64, float64, float64) {
	return r.axis[1][0], r.axis[1][1], r.axis[1][2]
}

// GetAxes2 returns the three components of the bounding box's third axis.
func (r *BoundingBox) GetAxes2() (float64, float64, float64) {
	return r.axis[2][0], r.axis[2][1], r.axis[2][2]
}

// GetYaw returns the current horizontal yaw angle in radians derived from the local X axis in world space.
func (r *BoundingBox) GetYaw() float64 {
	_, _, qz, qw := r.qYaw.Get()
	return 2.0 * math.Atan2(qz, qw)
}

// GetOrientedHalfExtents returns the half-extents of the oriented bounding box projected onto world X, Y, Z axes.
func (r *BoundingBox) GetOrientedHalfExtents() (float64, float64, float64) {
	return r.ex, r.ey, r.ez
}

// rebuild recalculates the center, local axes and enclosing AABB.
// The box rotates around its geometric center.
func (r *BoundingBox) rebuild() {
	hx := r.size.w * 0.5
	hy := r.size.h * 0.5
	hz := r.size.d * 0.5

	cx := r.bottomLeft.x + hx
	cy := r.bottomLeft.y + hy
	cz := r.bottomLeft.z + hz

	r.center.MoveTo(cx, cy, cz)
	r.bottomCenter.MoveTo(cx, cy, r.bottomLeft.z)

	r.updateAxes()

	// Project the oriented half-extents onto the world axes.
	r.ex = math.Abs(r.axis[0][0])*hx + math.Abs(r.axis[1][0])*hy + math.Abs(r.axis[2][0])*hz
	r.ey = math.Abs(r.axis[0][1])*hx + math.Abs(r.axis[1][1])*hy + math.Abs(r.axis[2][1])*hz
	r.ez = math.Abs(r.axis[0][2])*hx + math.Abs(r.axis[1][2])*hy + math.Abs(r.axis[2][2])*hz

	r.aabb.Rebuild(cx-r.ex, cy-r.ey, cz-r.ez, cx+r.ex, cy+r.ey, cz+r.ez)
}

// updateAxes transforms the local X, Y and Z axes into world space.
// Called only when the bounding box changes.
func (r *BoundingBox) updateAxes() {
	r.axis[0][0], r.axis[0][1], r.axis[0][2], r.axis[1][0], r.axis[1][1], r.axis[1][2], r.axis[2][0], r.axis[2][1], r.axis[2][2] = r.qRotation.ToAxes()
}

// GetAABB returns the enclosing axis-aligned bounding box.
func (r *BoundingBox) GetAABB() *AABB {
	return r.aabb
}

// GetWidth returns the width of the bounding box.
func (r *BoundingBox) GetWidth() float64 {
	return r.size.GetWidth()
}

// GetHeight returns the height of the bounding box.
func (r *BoundingBox) GetHeight() float64 {
	return r.size.GetHeight()
}

// GetDepth returns the depth of the bounding box.
func (r *BoundingBox) GetDepth() float64 {
	return r.size.GetDepth()
}

// GetSize returns the width, height and depth.
func (r *BoundingBox) GetSize() (float64, float64, float64) {
	return r.size.Get()
}

// GetSizeCenter returns the center of the dimensions.
func (r *BoundingBox) GetSizeCenter() (float64, float64, float64) {
	return r.size.GetCenter()
}

// GetZ returns the reference position's Z coordinate.
func (r *BoundingBox) GetZ() float64 {
	return r.bottomLeft.z
}

// GetBottomLeft returns the reference position.
func (r *BoundingBox) GetBottomLeft() (float64, float64, float64) {
	return r.bottomLeft.x, r.bottomLeft.y, r.bottomLeft.z
}

// GetBottomCenter returns the bottom-center coordinates.
func (r *BoundingBox) GetBottomCenter() (float64, float64, float64) {
	return r.bottomCenter.x, r.bottomCenter.y, r.bottomCenter.z
}

// GetCenter returns the geometric center.
func (r *BoundingBox) GetCenter() (float64, float64, float64) {
	return r.center.x, r.center.y, r.center.z
}

// SetTiltLimitEnabled sets whether tilt limit constraints are enabled for the bounding box.
func (r *BoundingBox) SetTiltLimitEnabled(enabled bool) {
	r.tiltLimitEnabled = enabled
}

// composeRotation rebuilds the complete orientation from the independent yaw and tilt states.
func (r *BoundingBox) composeRotation() {
	r.qYaw.Multiplies(r.qTilt, r.qRotation)
	r.qRotation.Normalize()
}

// setSwingFromDirection builds a pure tilt quaternion that rotates local Z onto dir.
// Any twist around local Z is intentionally discarded.
func (r *BoundingBox) setSwingFromDirection(dx, dy, dz float64) float64 {
	length := math.Sqrt(dx*dx + dy*dy + dz*dz)
	if length < 1e-12 {
		r.qTilt.SetIdentity()
		return 0
	}
	dx, dy, dz = dx/length, dy/length, dz/length
	if dz > 1.0 {
		dz = 1.0
	}
	if dz < -1.0 {
		dz = -1.0
	}
	angle := math.Acos(dz)
	if r.tiltLimitEnabled && angle > _maxTiltAngle {
		angle = _maxTiltAngle
	}
	axisX, axisY := -dy, dx // (0,0,1) x dir
	axisLength := math.Hypot(axisX, axisY)
	if axisLength < 1e-12 || angle < 1e-12 {
		r.qTilt.SetIdentity()
		return angle
	}
	r.qTilt.SetAxisAngle(axisX/axisLength, axisY/axisLength, 0, angle)
	return angle
}

// syncYawTiltFromRotation decomposes an externally supplied orientation
// into world-Z yaw and a pure swing (tilt).
func (r *BoundingBox) syncYawTiltFromRotation() {
	dx, dy, dz := r.extractSwingDirection()
	r.setSwingFromDirection(dx, dy, dz)
	r.composeRotation()
}

func (r *BoundingBox) Rotate(wx, wy, wz, dt float64) bool {
	if dt == 0 {
		return false
	}

	// Integrate the complete angular velocity in world space.
	r.qRotation.IntegrateAngular(wx, wy, wz, dt)

	if !r.tiltLimitEnabled {
		r.SyncRotation()
		return false
	}

	// Extract the yaw and pure swing from the complete orientation.
	dx, dy, dz := r.extractSwingDirection()

	// Angular velocity expressed in the yaw-local frame.
	lwx, lwy, _ := r.qYawInverse.Rotate(wx, wy, 0)

	angle := math.Acos(math.Max(-1, math.Min(1, dz)))

	// Detect outward angular motion beyond the tilt limit.
	reset := angle > _maxTiltAngle && (lwx*(-dy)+lwy*dx) > 0

	// Rebuild tilt, applying the limit only when enabled.
	r.setSwingFromDirection(dx, dy, dz)

	// Recompose the final orientation.
	r.composeRotation()
	r.SyncRotation()

	return reset
}

// supportAxisFactor smoothly maps an axis-normal alignment dot product to [-1, 1],
// suppressing discontinuous corner snaps within a small near-flat deadband.
func supportAxisFactor(d float64) float64 {
	const flatTol = 0.012
	const cornerTol = 0.06
	ad := math.Abs(d)
	if ad <= flatTol {
		return 0.0
	}
	if ad >= cornerTol {
		return math.Copysign(1.0, d)
	}
	t := (ad - flatTol) / (cornerTol - flatTol)
	s := t * t * (3.0 - 2.0*t)
	return math.Copysign(s, d)
}

// GetSupportPoint returns the extreme contact point of the OBB in direction (dirX, dirY, dirZ).
// For directions aligned with faces or edges, it returns the center of that face or edge.
func (r *BoundingBox) GetSupportPoint(dirX, dirY, dirZ float64) (float64, float64, float64) {
	cx, cy, cz := r.GetCenter()
	hw := r.size.w * 0.5
	hh := r.size.h * 0.5
	hd := r.size.d * 0.5

	px, py, pz := cx, cy, cz

	f0 := supportAxisFactor(r.axis[0][0]*dirX+r.axis[0][1]*dirY+r.axis[0][2]*dirZ) * hw
	if f0 != 0.0 {
		px += r.axis[0][0] * f0
		py += r.axis[0][1] * f0
		pz += r.axis[0][2] * f0
	}

	f1 := supportAxisFactor(r.axis[1][0]*dirX+r.axis[1][1]*dirY+r.axis[1][2]*dirZ) * hh
	if f1 != 0.0 {
		px += r.axis[1][0] * f1
		py += r.axis[1][1] * f1
		pz += r.axis[1][2] * f1
	}

	f2 := supportAxisFactor(r.axis[2][0]*dirX+r.axis[2][1]*dirY+r.axis[2][2]*dirZ) * hd
	if f2 != 0.0 {
		px += r.axis[2][0] * f2
		py += r.axis[2][1] * f2
		pz += r.axis[2][2] * f2
	}

	return px, py, pz
}

// IsFaceAligned returns whether any of the 6 box faces is aligned horizontally with world Z.
func (r *BoundingBox) IsFaceAligned() bool {
	maxZ := math.Max(math.Abs(r.axis[0][2]), math.Max(math.Abs(r.axis[1][2]), math.Abs(r.axis[2][2])))
	return maxZ > 0.99985
}

// SetSize updates the dimensions and derived data.
func (r *BoundingBox) SetSize(w, h, d float64) {
	r.size.w, r.size.h, r.size.d = w, h, d
	r.rebuild()
}

// AddSize increases the dimensions and updates derived data.
func (r *BoundingBox) AddSize(w, h, d float64) {
	r.size.w += w
	r.size.h += h
	r.size.d += d
	r.rebuild()
}

// AddTo translates the reference position.
func (r *BoundingBox) AddTo(x, y, z float64) {
	if x == 0.0 && y == 0.0 && z == 0.0 {
		return
	}
	r.bottomLeft.x += x
	r.bottomLeft.y += y
	r.bottomLeft.z += z
	r.rebuild()
}

// MoveTo changes the reference position.
func (r *BoundingBox) MoveTo(x, y, z float64) {
	r.bottomLeft.x = x
	r.bottomLeft.y = y
	r.bottomLeft.z = z
	r.rebuild()
}

// MoveToZ changes the reference position's Z coordinate.
func (r *BoundingBox) MoveToZ(z float64) {
	r.bottomLeft.z = z
	r.rebuild()
}

// MoveTest returns the reference position after applying a translation,
// without modifying the bounding box.
func (r *BoundingBox) MoveTest(vx, vy, vz float64) (float64, float64, float64) {
	return r.bottomLeft.x + vx,
		r.bottomLeft.y + vy,
		r.bottomLeft.z + vz
}

// Distance returns the distance between the geometric centers.
func (r *BoundingBox) Distance(target *BoundingBox) float64 {
	dx := target.center.x - r.center.x
	dy := target.center.y - r.center.y
	dz := target.center.z - r.center.z

	d := dx*dx + dy*dy + dz*dz
	if d < 0.0001 {
		return 0.01
	}
	return math.Sqrt(d)
}

var _oobId = [3]obbVector{{1, 0, 0}, {0, 1, 0}, {0, 0, 1}}

// Intersect tests this OBB against an axis-aligned box defined by
// its reference position and dimensions.
func (r *BoundingBox) Intersect(x, y, z, w, h, d float64) bool {
	return intersectsOBB(
		obbVector{r.center.x, r.center.y, r.center.z},
		r.axis,
		obbVector{r.size.w * 0.5, r.size.h * 0.5, r.size.d * 0.5},
		obbVector{x + w*0.5, y + h*0.5, z + d*0.5},
		_oobId,
		obbVector{w * 0.5, h * 0.5, d * 0.5},
	)
}

// IntersectBB tests this OBB against another OBB.
func (r *BoundingBox) IntersectBB(other *BoundingBox) bool {
	return intersectsOBB(
		obbVector{r.center.x, r.center.y, r.center.z},
		r.axis,
		obbVector{r.size.w * 0.5, r.size.h * 0.5, r.size.d * 0.5},
		obbVector{other.center.x, other.center.y, other.center.z},
		other.axis,
		obbVector{other.size.w * 0.5, other.size.h * 0.5, other.size.d * 0.5},
	)
}

// intersectsOBB performs the 15-axis Separating Axis Theorem test. It uses fixed-size arrays and does not create slices or heap objects.
func intersectsOBB(cA obbVector, a [3]obbVector, hA obbVector, cB obbVector, b [3]obbVector, hB obbVector) bool {
	dx := cB[0] - cA[0]
	dy := cB[1] - cA[1]
	dz := cB[2] - cA[2]
	var R [3][3]float64
	var AbsR [3][3]float64
	// Rotation matrix between the two box coordinate systems.
	// The epsilon stabilizes nearly parallel axes.
	const epsilon = 1e-12
	for i := 0; i < 3; i++ {
		ax := a[i][0]
		ay := a[i][1]
		az := a[i][2]
		for j := 0; j < 3; j++ {
			value := ax*b[j][0] + ay*b[j][1] + az*b[j][2]
			R[i][j] = value
			AbsR[i][j] = math.Abs(value) + epsilon
		}
	}

	// Center displacement in A's coordinate system.
	t0 := dx*a[0][0] + dy*a[0][1] + dz*a[0][2]
	t1 := dx*a[1][0] + dy*a[1][1] + dz*a[1][2]
	t2 := dx*a[2][0] + dy*a[2][1] + dz*a[2][2]

	// Test the three axes of A.
	for i := 0; i < 3; i++ {
		rb := hB[0]*AbsR[i][0] + hB[1]*AbsR[i][1] + hB[2]*AbsR[i][2]
		var distance float64
		switch i {
		case 0:
			distance = math.Abs(t0)
		case 1:
			distance = math.Abs(t1)
		default:
			distance = math.Abs(t2)
		}
		if distance > hA[i]+rb {
			return false
		}
	}

	// Test the three axes of B.
	for j := 0; j < 3; j++ {
		ra := hA[0]*AbsR[0][j] + hA[1]*AbsR[1][j] + hA[2]*AbsR[2][j]
		distance := math.Abs(t0*R[0][j] + t1*R[1][j] + t2*R[2][j])
		if distance > ra+hB[j] {
			return false
		}
	}

	// Test the nine cross-product axes A[i] x B[j].
	// These are evaluated directly without constructing vectors.
	for i := 0; i < 3; i++ {
		i1 := (i + 1) % 3
		i2 := (i + 2) % 3

		for j := 0; j < 3; j++ {
			j1 := (j + 1) % 3
			j2 := (j + 2) % 3
			ra := hA[i1]*AbsR[i2][j] + hA[i2]*AbsR[i1][j]
			rb := hB[j1]*AbsR[i][j2] + hB[j2]*AbsR[i][j1]
			var ti1, ti2 float64
			switch i1 {
			case 0:
				ti1 = t0
			case 1:
				ti1 = t1
			default:
				ti1 = t2
			}
			switch i2 {
			case 0:
				ti2 = t0
			case 1:
				ti2 = t1
			default:
				ti2 = t2
			}
			distance := math.Abs(ti2*R[i1][j] - ti1*R[i2][j])
			if distance > ra+rb {
				return false
			}
		}
	}

	// Preserve the previous contact semantics: touching is intersection.
	return true
}

// extractSwingDirection extracts the world-Z yaw from qRotation and returns the local-Z direction in the resulting yaw frame.
func (r *BoundingBox) extractSwingDirection() (float64, float64, float64) {
	_, _, qz, qw := r.qRotation.Get()

	twistLength := math.Hypot(qz, qw)
	if twistLength < 1e-12 {
		r.qYaw.SetIdentity()
	} else {
		r.qYaw.Set(0, 0, qz/twistLength, qw/twistLength)
	}

	r.qYawInverse.Conjugate(r.qYaw)
	r.qYawInverse.Multiplies(r.qRotation, r.qYawRelative)

	return r.qYawRelative.Rotate(0, 0, 1)
}
