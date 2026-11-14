package aabb

import (
	"math"
)

const maxTiltSin = 0.30

var _maxTiltAngle = math.Asin(maxTiltSin) //, in radians

// BoundingBox represents a 3D oriented bounding box (OBB) and its enclosing axis-aligned bounding box (AABB).
type BoundingBox struct {
	aabb             *AABB
	bottomLeft       Point
	center           Point
	size             Size
	axis             Axis        // Local axes transformed into world space.
	qRotation        *Quaternion // Orientation quaternion.
	qYaw             *Quaternion
	qTilt            *Quaternion
	qYawInverse      *Quaternion
	qYawRelative     *Quaternion
	halfExtents      Point // Oriented half-extents projected onto world axes.
	tiltLimitEnabled bool
}

// NewBoundingBox creates a bounding box with identity rotation.
func NewBoundingBox() *BoundingBox {
	r := &BoundingBox{
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
	r.bottomLeft.Set(blx, bly, blz)
	r.size.Set(w, h, d)
	r.rebuild()
}

// RebuildCenter updates the bounding box centered at (cx, cy, cz) with dimensions w, h, d, then recalculates the derived bounding-box data.
func (r *BoundingBox) RebuildCenter(cx, cy, cz, w, h, d float64) {
	r.bottomLeft.Set(cx-w*0.5, cy-h*0.5, cz-d*0.5)
	r.size.Set(w, h, d)
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
	return r.axis.Get()
}

// GetAxes0 returns the x, y, and z components of the first bounding box axis.
func (r *BoundingBox) GetAxes0() (float64, float64, float64) {
	return r.axis.Get0()
}

// GetAxes1 returns the x, y, and z components of the bounding box's second axis.
func (r *BoundingBox) GetAxes1() (float64, float64, float64) {
	return r.axis.Get1()
}

// GetAxes2 returns the three components of the bounding box's third axis.
func (r *BoundingBox) GetAxes2() (float64, float64, float64) {
	return r.axis.Get2()
}

// GetYaw returns the current horizontal yaw angle in radians derived from the local X axis in world space.
func (r *BoundingBox) GetYaw() float64 {
	_, _, qz, qw := r.qYaw.Get()
	return 2.0 * math.Atan2(qz, qw)
}

// GetOrientedHalfExtents returns the half-extents of the oriented bounding box projected onto world X, Y, Z axes.
func (r *BoundingBox) GetOrientedHalfExtents() (float64, float64, float64) {
	return r.halfExtents.Get()
}

// ProjectExtent returns the half-extent of the OBB projected onto unit normal (nx, ny, nz).
func (r *BoundingBox) ProjectExtent(nx, ny, nz float64) float64 {
	hx, hy, hz := r.size.GetCenter()
	return r.axis.ProjectNormal(nx, ny, nz, hx, hy, hz)
}

// rebuild recalculates the center, local axes and enclosing AABB.
// The box rotates around its geometric center.
func (r *BoundingBox) rebuild() {
	hx, hy, hz := r.size.GetCenter()

	cx := r.bottomLeft.GetX() + hx
	cy := r.bottomLeft.GetY() + hy
	cz := r.bottomLeft.GetZ() + hz

	r.center.MoveTo(cx, cy, cz)

	r.axis.Set(r.qRotation.ToAxes())

	ex, ey, ez := r.axis.Project(hx, hy, hz)
	r.halfExtents.Set(ex, ey, ez)

	r.aabb.Rebuild(cx-ex, cy-ey, cz-ez, cx+ex, cy+ey, cz+ez)
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
	return r.bottomLeft.GetZ()
}

// GetBottomLeft returns the reference position.
func (r *BoundingBox) GetBottomLeft() (float64, float64, float64) {
	return r.bottomLeft.Get()
}

// GetCenter returns the geometric center.
func (r *BoundingBox) GetCenter() (float64, float64, float64) {
	return r.center.Get()
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

// Rotate integrates world-space angular velocity over dt, updates the orientation, and reports tilt-limit reset.
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

// GetSupportPoint returns the extreme contact point of the OBB in direction (dirX, dirY, dirZ).
// For directions aligned with faces or edges, it returns the center of that face or edge.
func (r *BoundingBox) GetSupportPoint(dirX, dirY, dirZ float64) (float64, float64, float64) {
	cx, cy, cz := r.GetCenter()
	hw, hh, hd := r.size.GetCenter()
	px, py, pz := r.axis.ComputeSupportPoint(cx, cy, cz, dirX, dirY, dirZ, hw, hh, hd)
	return px, py, pz
}

// IsFaceAligned returns whether any of the 6 box faces is aligned horizontally with world Z.
func (r *BoundingBox) IsFaceAligned() bool {
	maxZ := math.Max(math.Abs(r.axis.data[0][2]), math.Max(math.Abs(r.axis.data[1][2]), math.Abs(r.axis.data[2][2])))
	return maxZ > 0.99985
}

// SetSize updates the dimensions and derived data.
func (r *BoundingBox) SetSize(w, h, d float64) {
	r.size.Set(w, h, d)
	r.rebuild()
}

// AddSize increases the dimensions and updates derived data.
func (r *BoundingBox) AddSize(w, h, d float64) {
	r.size.Add(w, h, d)
	r.rebuild()
}

// AddTo translates the reference position.
func (r *BoundingBox) AddTo(x, y, z float64) {
	if x == 0.0 && y == 0.0 && z == 0.0 {
		return
	}
	r.bottomLeft.AddTo(x, y, z)
	r.rebuild()
}

// MoveTo changes the reference position.
func (r *BoundingBox) MoveTo(x, y, z float64) {
	r.bottomLeft.Set(x, y, z)
	r.rebuild()
}

// MoveToZ changes the reference position's Z coordinate.
func (r *BoundingBox) MoveToZ(z float64) {
	r.bottomLeft.SetZ(z)
	r.rebuild()
}

// Distance returns the distance between the geometric centers.
func (r *BoundingBox) Distance(target *BoundingBox) float64 {
	tx, ty, tz := target.center.Get()
	px, py, pz := r.center.Get()
	dx := tx - px
	dy := ty - py
	dz := tz - pz
	d := dx*dx + dy*dy + dz*dz
	if d < 0.0001 {
		return 0.01
	}
	return math.Sqrt(d)
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

// Intersect tests this OBB against an axis-aligned box defined by
// its reference position and dimensions.
func (r *BoundingBox) Intersect(x, y, z, w, h, d float64) bool {
	cx, cy, cz := r.center.Get()
	dx := (x + (w * 0.5)) - cx
	dy := (y + (h * 0.5)) - cy
	dz := (z + (d * 0.5)) - cz
	scx, scy, scz := r.size.GetCenter()
	ocx, ocy, ocz := w*0.5, h*0.5, d*0.5
	hit, _, _ := overlapOBB(dx, dy, dz, &r.axis, scx, scy, scz, r.axis.GetIdentity(), ocx, ocy, ocz)
	return hit
}

// IntersectBB tests this OBB against another OBB.
func (r *BoundingBox) IntersectBB(other *BoundingBox) bool {
	return r.IntersectBBAtOffset(other, 0, 0, 0)
}

// IntersectBBAtOffset tests this OBB against another OBB shifted by relative offset (offX, offY, offZ).
func (r *BoundingBox) IntersectBBAtOffset(other *BoundingBox, offX, offY, offZ float64) bool {
	hit, _, _ := r.OverlapBBAtOffset(other, offX, offY, offZ)
	return hit
}

// OverlapBBAtOffset tests this OBB against another OBB shifted by relative offset (offX, offY, offZ)
// and returns whether they overlap along with the minimum SAT penetration depth (when overlapping)
// or maximum SAT separation gap (when separated).
func (r *BoundingBox) OverlapBBAtOffset(other *BoundingBox, offX, offY, offZ float64) (bool, float64, float64) {
	tx, ty, tz := other.center.Get()
	px, py, pz := r.center.Get()
	dx := tx - px + offX
	dy := ty - py + offY
	dz := tz - pz + offZ
	scx, scy, scz := r.size.GetCenter()
	ocx, ocy, ocz := other.size.GetCenter()
	return overlapOBB(dx, dy, dz, &r.axis, scx, scy, scz, &other.axis, ocx, ocy, ocz)
}

// overlapOBB performs the 15-axis Separating Axis Theorem test and tracks Euclidean overlap/gap bounds.
// It uses fixed-size arrays and does not create slices or heap objects.
func overlapOBB(dx, dy, dz float64, a *Axis, scx, scy, scz float64, b *Axis, ocx, ocy, ocz float64) (bool, float64, float64) {
	var hA = [3]float64{scx, scy, scz}
	var hB = [3]float64{ocx, ocy, ocz}
	var R [3][3]float64
	var AbsR [3][3]float64
	// Rotation matrix between the two box coordinate systems.
	// The epsilon stabilizes nearly parallel axes.
	const epsilon = 1e-12
	const crossAxisLenEps = 1e-6
	for i := 0; i < 3; i++ {
		ax := a.data[i][0]
		ay := a.data[i][1]
		az := a.data[i][2]
		for j := 0; j < 3; j++ {
			value := ax*b.data[j][0] + ay*b.data[j][1] + az*b.data[j][2]
			R[i][j] = value
			AbsR[i][j] = math.Abs(value) + epsilon
		}
	}

	// Center displacement in A's coordinate system.
	t0 := dx*a.data[0][0] + dy*a.data[0][1] + dz*a.data[0][2]
	t1 := dx*a.data[1][0] + dy*a.data[1][1] + dz*a.data[1][2]
	t2 := dx*a.data[2][0] + dy*a.data[2][1] + dz*a.data[2][2]

	separated := false
	minPen := math.MaxFloat64
	maxGap := 0.0

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
			separated = true
			if g := distance - (hA[i] + rb); g > maxGap {
				maxGap = g
			}
		} else if !separated {
			if p := hA[i] + rb - distance; p < minPen {
				minPen = p
			}
		}
	}

	// Test the three axes of B.
	for j := 0; j < 3; j++ {
		ra := hA[0]*AbsR[0][j] + hA[1]*AbsR[1][j] + hA[2]*AbsR[2][j]
		distance := math.Abs(t0*R[0][j] + t1*R[1][j] + t2*R[2][j])
		if distance > ra+hB[j] {
			separated = true
			if g := distance - (ra + hB[j]); g > maxGap {
				maxGap = g
			}
		} else if !separated {
			if p := ra + hB[j] - distance; p < minPen {
				minPen = p
			}
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
			axisLen := math.Hypot(R[i1][j], R[i2][j])
			if distance > ra+rb {
				separated = true
				if axisLen > crossAxisLenEps {
					if g := (distance - (ra + rb)) / axisLen; g > maxGap {
						maxGap = g
					}
				}
			} else if !separated && axisLen > crossAxisLenEps {
				if p := (ra + rb - distance) / axisLen; p < minPen {
					minPen = p
				}
			}
		}
	}

	if separated {
		return false, 0.0, maxGap
	}
	// Preserve the previous contact semantics: touching is intersection.
	return true, minPen, 0.0
}
