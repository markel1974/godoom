package physics

// Rect represents a 2D rectangle with a position, size, and optional axis-aligned bounding box (AABB) for spatial calculations.
type Rect struct {
	point  Point
	size   Size
	center Point
	z      float64
	aabb   *AABB
}

// NewRect creates and returns a new Rect instance initialized with the given position, size, and depth values.
func NewRect(x float64, y float64, w float64, h float64, z float64) Rect {
	r := Rect{
		aabb:  &AABB{},
		point: NewPointFloat(x, y),
		size:  NewSize(w, h),
		z:     z,
	}
	r.rebuild()

	return r
}

// rebuild updates the Rect's center, AABB bounds, and AABB surface area based on its current position and size.
func (r *Rect) rebuild() {
	r.center.x = r.point.x + (r.size.w / 2)
	r.center.y = r.point.y + (r.size.h / 2)

	// L'AABB ora riflette le dimensioni ESATTE (Tight Bounding Box)
	r.aabb.minX = r.point.x
	r.aabb.maxX = r.point.x + r.size.w
	r.aabb.minY = r.point.y
	r.aabb.maxY = r.point.y + r.size.h
	r.aabb.minZ = 0
	r.aabb.maxZ = r.z
	r.aabb.surfaceArea = r.aabb.CalculateSurfaceArea()
}

// SetSize adjusts the width and height of the rectangle by the given values and updates its bounding box.
func (r *Rect) SetSize(w float64, h float64) {
	r.size.w += w
	r.size.h += h
	r.rebuild()
}

// AddTo adjusts the position of the Rect by adding the given x and y values to its current coordinates.
func (r *Rect) AddTo(x float64, y float64) {
	r.point.x += x
	r.point.y += y
	r.rebuild()
}

// AddToX increments the x-coordinate of the Rect's point by the specified value and recalculates its properties.
func (r *Rect) AddToX(x float64) {
	r.point.x += x
	r.rebuild()
}

// AddToY adds the specified value to the y-coordinate of the rectangle's point and updates its internal state.
func (r *Rect) AddToY(y float64) {
	r.point.y += y
	r.rebuild()
}

// MoveTo sets the top-left corner of the rectangle to the specified x and y coordinates and updates its properties.
func (r *Rect) MoveTo(x float64, y float64) {
	r.point.x = x
	r.point.y = y
	r.rebuild()
}

// MoveToX updates the x-coordinate of the rectangle's position and recalculates its derived properties.
func (r *Rect) MoveToX(x float64) {
	r.point.x = x
	r.rebuild()
}

// MoveToY updates the Rect's y-coordinate to the specified value and recalculates its center and bounding box.
func (r *Rect) MoveToY(y float64) {
	r.point.y = y
	r.rebuild()
}

// GetCenterXY returns the x and y coordinates of the rectangle's center as a pair of float64 values.
func (r *Rect) GetCenterXY() (float64, float64) {
	return r.center.x, r.center.y
}

// GetCenterX returns the x-coordinate of the center point of the rectangle.
func (r *Rect) GetCenterX() float64 {
	return r.center.x
}

// GetCenterY returns the y-coordinate of the center point of the Rect structure.
func (r *Rect) GetCenterY() float64 {
	return r.center.y
}

// GetX returns the x-coordinate of the rectangle's starting point.
func (r *Rect) GetX() float64 {
	return r.point.x
}

// GetY returns the Y-coordinate of the bottom-left corner of the rectangle.
func (r *Rect) GetY() float64 {
	return r.point.y
}

// GetWidth returns the width of the rectangle (Rect) as a float64 value.
func (r *Rect) GetWidth() float64 {
	return r.size.w
}

// GetHeight returns the height of the Rect structure.
func (r *Rect) GetHeight() float64 {
	return r.size.h
}

// GetAABB returns the axis-aligned bounding box (AABB) associated with the Rect.
func (r *Rect) GetAABB() *AABB {
	return r.aabb
}

// IntersectRect checks if the current rectangle intersects with another rectangle (r2) and returns true if they overlap.
func (r *Rect) IntersectRect(r2 Rect) bool {
	return r.Intersect(r2.point.x, r2.point.y, r2.size.w, r2.size.h)
}

// Intersect checks if the rectangle intersects with another rectangle defined by its top-left corner (x2, y2), width (w2), and height (h2). Returns true if an intersection exists, otherwise false.
func (r *Rect) Intersect(x2 float64, y2 float64, w2 float64, h2 float64) bool {
	if x2 > r.size.w+r.point.x || r.point.x > w2+x2 || y2 > r.size.h+r.point.y || r.point.y > h2+y2 {
		return false
	}
	return true
}
