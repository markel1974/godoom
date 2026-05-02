package physics

// Rect represents a 3D rectangle in space, defined by its position, size, depth, elevation, and bounding box.
type Rect struct {
	point  Point
	size   Size
	center Point
	aabb   *AABB
}

// NewRect creates a new Rect instance with the specified position (x, y), size (w, h), base elevation (z), and depth (d).
func NewRect(x, y, w, h, z, d float64) Rect {
	r := Rect{
		aabb:  &AABB{},
		point: NewPoint(x, y, z),
		size:  NewSize(w, h, d),
	}
	r.rebuild()
	return r
}

// GetBottomLeft returns the bottom-left corner coordinates (x, y, z) of the Rect as float64 values.
func (r *Rect) GetBottomLeft() (float64, float64, float64) {
	return r.point.x, r.point.y, r.point.z
}

// Reset updates the position, size, and depth of the Rect to the specified values and recalculates its properties.
func (r *Rect) Reset(x, y, z, w, h, d float64) {
	r.point.x = x
	r.point.y = y
	r.point.z = z
	r.size.w = w
	r.size.h = h
	r.size.d = d
	r.rebuild()
}

// rebuild recalculates the rectangle's center, AABB bounds, and surface area based on its current position and size.
func (r *Rect) rebuild() {
	r.center.x = r.point.x + (r.size.w / 2)
	r.center.y = r.point.y + (r.size.h / 2)
	r.center.z = r.point.z + (r.size.d / 2)
	minX := r.point.x
	maxX := r.point.x + r.size.w
	minY := r.point.y
	maxY := r.point.y + r.size.h
	minZ := r.point.z
	maxZ := r.point.z + r.size.d
	r.aabb.Rebuild(minX, minY, minZ, maxX, maxY, maxZ)
}

// SetSize updates the width, height, and depth of the Rect and recalculates its properties by invoking rebuild.
func (r *Rect) SetSize(w, h, d float64) {
	r.size.w = w
	r.size.h = h
	r.size.d = d
	r.rebuild()
}

// AddSize adjusts the dimensions of the Rect by adding the given width, height, and depth values.
func (r *Rect) AddSize(w, h, d float64) {
	r.size.w += w
	r.size.h += h
	r.size.d += d
	r.rebuild()
}

// AddTo modifies the Rect's position and base elevation by adding the specified x, y, and z values, then rebuilds its state.
func (r *Rect) AddTo(x, y, z float64) {
	r.point.x += x
	r.point.y += y
	r.point.z += z
	r.rebuild()
}

// MoveTo updates the position of the Rect to the specified x, y, and z coordinates, and rebuilds its spatial properties.
func (r *Rect) MoveTo(x, y, z float64) {
	r.point.x = x
	r.point.y = y
	r.point.z = z
	r.rebuild()
}

// MoveTest calculates and returns the resulting coordinates after applying velocity values (vx, vy, vz) to the Rect position.
func (r *Rect) MoveTest(vx, vy, vz float64) (float64, float64, float64) {
	x := r.point.x + vx
	y := r.point.y + vy
	z := r.point.z + vz
	return x, y, z
}

// GetCenter returns the 3D center coordinates (x, y, z) of the Rect as float64 values.
func (r *Rect) GetCenter() (float64, float64, float64) {
	return r.center.x, r.center.y, r.center.z
}

// IntersectRect checks if the current rectangle intersects with another rectangle and returns true if they overlap.
func (r *Rect) IntersectRect(r2 Rect) bool {
	return r.Intersect(r2.point.x, r2.point.y, r2.point.z, r2.size.w, r2.size.h, r2.size.d)
}

// Intersect checks if the current rectangle intersects with another rectangle specified by its position and dimensions.
func (r *Rect) Intersect(x2, y2, z2, w2, h2, d2 float64) bool {
	// Separating Axis Theorem (SAT) per AABB
	if x2 > r.point.x+r.size.w || r.point.x > x2+w2 ||
		y2 > r.point.y+r.size.h || r.point.y > y2+h2 ||
		z2 > r.point.z+r.size.d || r.point.z > z2+d2 {
		return false
	}
	return true
}

// GetZ returns the base elevation (z-coordinate) of the Rect.
func (r *Rect) GetZ() float64 { return r.point.z }

// GetWidth retrieves the width dimension of the Rect structure.
func (r *Rect) GetWidth() float64 { return r.size.w }

// GetHeight returns the height of the Rect as a float64.
func (r *Rect) GetHeight() float64 { return r.size.h }

// GetDepth returns the depth (d) of the Rect object as a float64 value.
func (r *Rect) GetDepth() float64 { return r.size.d }

func (r *Rect) GetSize() (float64, float64, float64) { return r.size.w, r.size.h, r.size.d }

// GetAABB returns a pointer to the axis-aligned bounding box (AABB) associated with the Rect instance.
func (r *Rect) GetAABB() *AABB { return r.aabb }
