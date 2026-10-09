package physics

import (
	"math"

	aabb "github.com/markel1974/godoom/mr_tech/physics/aabb"
)

// BoundingBoxOBB represents a 3D oriented bounding box (OBB)
// and its enclosing axis-aligned bounding box (AABB).
type BoundingBoxOBB struct {
	bottomLeft   Point
	bottomCenter Point
	center       Point
	size         Size
	aabb         *aabb.AABB

	// Local axes transformed into world space.
	axis [3]obbVector

	// Rotation quaternion in x, y, z, w order.
	rotationX float64
	rotationY float64
	rotationZ float64
	rotationW float64
}

// obbVector is a fixed-size 3D vector used by the SAT implementation.
type obbVector [3]float64

// NewBoundingBoxOBB creates a bounding box with identity rotation.
func NewBoundingBoxOBB(x, y, w, h, z, d float64) *BoundingBoxOBB {
	r := &BoundingBoxOBB{
		bottomLeft:   NewPoint(x, y, z),
		bottomCenter: NewPoint(0, 0, 0),
		center:       NewPoint(0, 0, 0),
		size:         NewSize(w, h, d),
		aabb:         aabb.NewAABB(),
		rotationW:    1,
	}
	r.rebuild()
	return r
}

// Rebuild updates the reference position and dimensions,
// then recalculates the derived bounding-box data.
func (r *BoundingBoxOBB) Rebuild(x, y, z, w, h, d float64) {
	r.bottomLeft.x, r.bottomLeft.y, r.bottomLeft.z = x, y, z
	r.size.w, r.size.h, r.size.d = w, h, d
	r.rebuild()
}

// SetRotationQuaternion sets the orientation using a quaternion (x, y, z, w).
// The quaternion is normalized before being stored.
func (r *BoundingBoxOBB) SetRotationQuaternion(x, y, z, w float64) {
	length := x*x + y*y + z*z + w*w
	if length < 1e-24 {
		r.rotationX = 0
		r.rotationY = 0
		r.rotationZ = 0
		r.rotationW = 1
	} else {
		invLength := 1.0 / math.Sqrt(length)
		r.rotationX = x * invLength
		r.rotationY = y * invLength
		r.rotationZ = z * invLength
		r.rotationW = w * invLength
	}
	r.rebuild()
}

// GetRotationQuaternion returns the orientation in x, y, z, w order.
func (r *BoundingBoxOBB) GetRotationQuaternion() (x, y, z, w float64) {
	return r.rotationX, r.rotationY, r.rotationZ, r.rotationW
}

// rebuild recalculates the center, local axes and enclosing AABB.
// The box rotates around its geometric center.
func (r *BoundingBoxOBB) rebuild() {
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
	ex := math.Abs(r.axis[0][0])*hx + math.Abs(r.axis[1][0])*hy + math.Abs(r.axis[2][0])*hz
	ey := math.Abs(r.axis[0][1])*hx + math.Abs(r.axis[1][1])*hy + math.Abs(r.axis[2][1])*hz
	ez := math.Abs(r.axis[0][2])*hx + math.Abs(r.axis[1][2])*hy + math.Abs(r.axis[2][2])*hz

	r.aabb.Rebuild(cx-ex, cy-ey, cz-ez, cx+ex, cy+ey, cz+ez)
}

// updateAxes transforms the local X, Y and Z axes into world space.
// Called only when the bounding box changes.
func (r *BoundingBoxOBB) updateAxes() {
	x := r.rotationX
	y := r.rotationY
	z := r.rotationZ
	w := r.rotationW

	xx := x * x
	yy := y * y
	zz := z * z
	xy := x * y
	xz := x * z
	yz := y * z
	wx := w * x
	wy := w * y
	wz := w * z

	r.axis[0][0] = 1 - 2*(yy+zz)
	r.axis[0][1] = 2 * (xy + wz)
	r.axis[0][2] = 2 * (xz - wy)

	r.axis[1][0] = 2 * (xy - wz)
	r.axis[1][1] = 1 - 2*(xx+zz)
	r.axis[1][2] = 2 * (yz + wx)

	r.axis[2][0] = 2 * (xz + wy)
	r.axis[2][1] = 2 * (yz - wx)
	r.axis[2][2] = 1 - 2*(xx+yy)
}

// GetAABB returns the enclosing axis-aligned bounding box.
func (r *BoundingBoxOBB) GetAABB() *aabb.AABB {
	return r.aabb
}

// GetWidth returns the width of the bounding box.
func (r *BoundingBoxOBB) GetWidth() float64 {
	return r.size.GetWidth()
}

// GetHeight returns the height of the bounding box.
func (r *BoundingBoxOBB) GetHeight() float64 {
	return r.size.GetHeight()
}

// GetDepth returns the depth of the bounding box.
func (r *BoundingBoxOBB) GetDepth() float64 {
	return r.size.GetDepth()
}

// GetSize returns the width, height and depth.
func (r *BoundingBoxOBB) GetSize() (float64, float64, float64) {
	return r.size.Get()
}

// GetSizeCenter returns the center of the dimensions.
func (r *BoundingBoxOBB) GetSizeCenter() (float64, float64, float64) {
	return r.size.GetCenter()
}

// GetZ returns the reference position's Z coordinate.
func (r *BoundingBoxOBB) GetZ() float64 {
	return r.bottomLeft.z
}

// GetBottomLeft returns the reference position.
func (r *BoundingBoxOBB) GetBottomLeft() (float64, float64, float64) {
	return r.bottomLeft.x, r.bottomLeft.y, r.bottomLeft.z
}

// GetBottomCenter returns the bottom-center coordinates.
func (r *BoundingBoxOBB) GetBottomCenter() (float64, float64, float64) {
	return r.bottomCenter.x, r.bottomCenter.y, r.bottomCenter.z
}

// GetCenter returns the geometric center.
func (r *BoundingBoxOBB) GetCenter() (float64, float64, float64) {
	return r.center.x, r.center.y, r.center.z
}

// SetSize updates the dimensions and derived data.
func (r *BoundingBoxOBB) SetSize(w, h, d float64) {
	r.size.w, r.size.h, r.size.d = w, h, d
	r.rebuild()
}

// AddSize increases the dimensions and updates derived data.
func (r *BoundingBoxOBB) AddSize(w, h, d float64) {
	r.size.w += w
	r.size.h += h
	r.size.d += d
	r.rebuild()
}

// AddTo translates the reference position.
func (r *BoundingBoxOBB) AddTo(x, y, z float64) {
	r.bottomLeft.x += x
	r.bottomLeft.y += y
	r.bottomLeft.z += z
	r.rebuild()
}

// MoveTo changes the reference position.
func (r *BoundingBoxOBB) MoveTo(x, y, z float64) {
	r.bottomLeft.x = x
	r.bottomLeft.y = y
	r.bottomLeft.z = z
	r.rebuild()
}

// MoveToZ changes the reference position's Z coordinate.
func (r *BoundingBoxOBB) MoveToZ(z float64) {
	r.bottomLeft.z = z
	r.rebuild()
}

// MoveTest returns the reference position after applying a translation,
// without modifying the bounding box.
func (r *BoundingBoxOBB) MoveTest(vx, vy, vz float64) (float64, float64, float64) {
	return r.bottomLeft.x + vx,
		r.bottomLeft.y + vy,
		r.bottomLeft.z + vz
}

// Distance returns the distance between the geometric centers.
func (r *BoundingBoxOBB) Distance(target *BoundingBox) float64 {
	dx := target.center.x - r.center.x
	dy := target.center.y - r.center.y
	dz := target.center.z - r.center.z

	d := dx*dx + dy*dy + dz*dz
	if d < 0.0001 {
		return 0.01
	}
	return math.Sqrt(d)
}

// Intersect tests this OBB against an axis-aligned box defined by
// its reference position and dimensions.
func (r *BoundingBoxOBB) Intersect(x, y, z, w, h, d float64) bool {
	return intersectsOBB(
		obbVector{r.center.x, r.center.y, r.center.z},
		r.axis,
		obbVector{r.size.w * 0.5, r.size.h * 0.5, r.size.d * 0.5},
		obbVector{x + w*0.5, y + h*0.5, z + d*0.5},
		[3]obbVector{{1, 0, 0}, {0, 1, 0}, {0, 0, 1}},
		obbVector{w * 0.5, h * 0.5, d * 0.5},
	)
}

// IntersectBB tests this OBB against another OBB.
func (r *BoundingBoxOBB) IntersectBB(other *BoundingBoxOBB) bool {
	return intersectsOBB(
		obbVector{r.center.x, r.center.y, r.center.z},
		r.axis,
		obbVector{r.size.w * 0.5, r.size.h * 0.5, r.size.d * 0.5},
		obbVector{other.center.x, other.center.y, other.center.z},
		other.axis,
		obbVector{other.size.w * 0.5, other.size.h * 0.5, other.size.d * 0.5},
	)
}

// intersectsOBB performs the 15-axis Separating Axis Theorem test.
// It uses fixed-size arrays and does not create slices or heap objects.
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

			distance := math.Abs(
				ti2*R[i1][j] -
					ti1*R[i2][j],
			)

			if distance > ra+rb {
				return false
			}
		}
	}

	// Preserve the previous contact semantics: touching is intersection.
	return true
}
