package physics

import "math"

// BoundingBox represents a 3D rectangular region defined by its position, dimensions, and axis-aligned bounding box (AABB).
type BoundingBox struct {
	bottomLeft   *Point
	bottomCenter *Point
	center       *Point
	size         *Size
	aabb         *AABB
}

// NewBoundingBox initializes and returns a pointer to a BoundingBox with the specified position, dimensions, and depth.
func NewBoundingBox(x, y, w, h, z, d float64) *BoundingBox {
	r := &BoundingBox{
		bottomLeft:   NewPoint(x, y, z),
		bottomCenter: NewPoint(0, 0, 0),
		center:       NewPoint(0, 0, 0),
		size:         NewSize(w, h, d),
		aabb:         NewAABB(),
	}
	r.rebuild()
	return r
}

// Rebuild updates the bounding box's bottom-left position and size, and triggers a rebuild of its dependent properties.
func (r *BoundingBox) Rebuild(x, y, z, w, h, d float64) {
	r.bottomLeft.x, r.bottomLeft.y, r.bottomLeft.z = x, y, z
	r.size.w, r.size.h, r.size.d = w, h, d
	r.rebuild()
}

// rebuild recalculates the center, bottom-center, and AABB boundaries of the bounding box based on its current position and size.
func (r *BoundingBox) rebuild() {
	cw, ch, cd := r.size.GetCenter()
	r.center.MoveTo(r.bottomLeft.x+cw, r.bottomLeft.y+ch, r.bottomLeft.z+cd)
	r.bottomCenter.MoveTo(r.center.x, r.center.y, r.bottomLeft.z)
	minX := r.bottomLeft.x
	minY := r.bottomLeft.y
	minZ := r.bottomLeft.z
	maxX := r.bottomLeft.x + r.size.w
	maxY := r.bottomLeft.y + r.size.h
	maxZ := r.bottomLeft.z + r.size.d
	r.aabb.Rebuild(minX, minY, minZ, maxX, maxY, maxZ)
}

// GetBottomLeft retrieves the x, y, and z coordinates of the bottom-left corner of the bounding box.
func (r *BoundingBox) GetBottomLeft() (float64, float64, float64) {
	return r.bottomLeft.x, r.bottomLeft.y, r.bottomLeft.z
}

// GetBottomCenter returns the x, y, and z coordinates of the bottom-center point of the bounding box.
func (r *BoundingBox) GetBottomCenter() (float64, float64, float64) {
	return r.bottomCenter.x, r.bottomCenter.y, r.bottomCenter.z
}

// GetCenter returns the x, y, and z coordinates of the center point of the bounding box.
func (r *BoundingBox) GetCenter() (float64, float64, float64) {
	return r.center.x, r.center.y, r.center.z
}

// SetSize updates the width, height, and depth of the bounding box and triggers a rebuild of its derived properties.
func (r *BoundingBox) SetSize(w, h, d float64) {
	r.size.w = w
	r.size.h = h
	r.size.d = d
	r.rebuild()
}

// AddSize increases the width, height, and depth of the bounding box by the specified values and updates its state.
func (r *BoundingBox) AddSize(w, h, d float64) {
	r.size.w += w
	r.size.h += h
	r.size.d += d
	r.rebuild()
}

// AddTo adjusts the position of the BoundingBox by adding the specified x, y, and z offsets to its bottom left corner.
func (r *BoundingBox) AddTo(x, y, z float64) {
	r.bottomLeft.x += x
	r.bottomLeft.y += y
	r.bottomLeft.z += z
	r.rebuild()
}

// MoveTo updates the bottom-left corner of the BoundingBox to the specified x, y, and z coordinates and recalculates derived properties.
func (r *BoundingBox) MoveTo(x, y, z float64) {
	r.bottomLeft.x = x
	r.bottomLeft.y = y
	r.bottomLeft.z = z
	r.rebuild()
}

// MoveToZ updates the z-coordinate of the bottom-left point of the BoundingBox and recalculates dependent properties.
func (r *BoundingBox) MoveToZ(z float64) {
	r.bottomLeft.z = z
	r.rebuild()
}

// MoveTest computes new coordinates by adding the given offsets to the BoundingBox's bottom-left corner position.
func (r *BoundingBox) MoveTest(vx, vy, vz float64) (float64, float64, float64) {
	x := r.bottomLeft.x + vx
	y := r.bottomLeft.y + vy
	z := r.bottomLeft.z + vz
	return x, y, z
}

// IntersectBB checks if the current bounding box intersects with another bounding box.
func (r *BoundingBox) IntersectBB(r2 *BoundingBox) bool {
	return r.Intersect(r2.bottomLeft.x, r2.bottomLeft.y, r2.bottomLeft.z, r2.size.w, r2.size.h, r2.size.d)
}

// Distance computes the Euclidean distance between the entity and a specified collider entity.
func (r *BoundingBox) Distance(target *BoundingBox) float64 {
	x1, y1, z1 := r.GetCenter()
	x2, y2, z2 := target.GetCenter()
	dx := x2 - x1
	dy := y2 - y1
	dz := z2 - z1
	d := dx*dx + dy*dy + dz*dz
	if d < 0.0001 {
		return 0.01
	}
	return math.Sqrt(d)
}

// Intersect checks if the current bounding box intersects with another AABB defined by its position and dimensions.
func (r *BoundingBox) Intersect(x2, y2, z2, w2, h2, d2 float64) bool {
	// Separating Axis Theorem (SAT) per AABB
	if x2 > r.bottomLeft.x+r.size.w || r.bottomLeft.x > x2+w2 ||
		y2 > r.bottomLeft.y+r.size.h || r.bottomLeft.y > y2+h2 ||
		z2 > r.bottomLeft.z+r.size.d || r.bottomLeft.z > z2+d2 {
		return false
	}
	return true
}

// GetZ returns the z-coordinate of the bottom-left point of the BoundingBox.
func (r *BoundingBox) GetZ() float64 { return r.bottomLeft.z }

// GetWidth returns the width of the bounding box as a float64 value.
func (r *BoundingBox) GetWidth() float64 { return r.size.GetWidth() }

// GetHeight returns the height of the BoundingBox.
func (r *BoundingBox) GetHeight() float64 { return r.size.GetHeight() }

// GetDepth returns the depth value of the bounding box.
func (r *BoundingBox) GetDepth() float64 { return r.size.GetDepth() }

// GetSize returns the width, height, and depth of the bounding box as a tuple of three float64 values.
func (r *BoundingBox) GetSize() (float64, float64, float64) { return r.size.Get() }

// GetAABB returns the axis-aligned bounding box (AABB) of the BoundingBox.
func (r *BoundingBox) GetAABB() *AABB { return r.aabb }
