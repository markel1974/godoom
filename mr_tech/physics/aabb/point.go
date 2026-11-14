package aabb

// Point represents a 3D point in space with x, y, and z coordinates as float64 values.
type Point struct {
	x1 float64
	y1 float64
	z1 float64
}

// NewPoint creates a new Point with the specified x, y, and z coordinates as float64 values.
func NewPoint(x float64, y float64, z float64) *Point {
	return &Point{
		x1: x,
		y1: y,
		z1: z,
	}
}

// Get returns the x, y, and z coordinates of the Point.
func (p *Point) Get() (float64, float64, float64) {
	return p.x1, p.y1, p.z1
}

// GetX returns the x-coordinate of the Point.
func (p *Point) GetX() float64 {
	return p.x1
}

// GetY returns the y-coordinate value of the Point.
func (p *Point) GetY() float64 {
	return p.y1
}

// GetZ returns the z-coordinate value of the Point.
func (p *Point) GetZ() float64 {
	return p.z1
}

// Set updates the point's x, y, and z coordinates to the specified values.
func (p *Point) Set(x float64, y float64, z float64) {
	p.x1 = x
	p.y1 = y
	p.z1 = z
}

// SetX sets the x-coordinate of the Point to the specified value.
func (p *Point) SetX(x float64) {
	p.x1 = x
}

// SetY sets the y-coordinate of the Point to the specified value.
func (p *Point) SetY(y float64) {
	p.y1 = y
}

// SetZ sets the z-coordinate of the Point to the specified value.
func (p *Point) SetZ(z float64) {
	p.z1 = z
}

// AddTo updates the x and y coordinates of the Point by adding the given values to the current coordinates.
func (p *Point) AddTo(x float64, y float64, z float64) {
	p.x1 += x
	p.y1 += y
	p.z1 += z
}

// AddToX increments the x-coordinate of the Point by the specified value.
func (p *Point) AddToX(x float64) {
	p.x1 += x
}

// AddToY increments the y-coordinate of the Point by the specified value.
func (p *Point) AddToY(y float64) {
	p.y1 += y
}

// AddToZ increments the z-coordinate of the Point by the given value.
func (p *Point) AddToZ(z float64) {
	p.z1 += z
}

// MoveTo updates the coordinates of the point to the specified x, y, and z values.
func (p *Point) MoveTo(x float64, y float64, z float64) {
	p.x1 = x
	p.y1 = y
	p.z1 = z
}

// MoveToX updates the x-coordinate of the Point to the specified value.
func (p *Point) MoveToX(x float64) {
	p.x1 = x
}

// MoveToY sets the y-coordinate of the Point to the specified value.
func (p *Point) MoveToY(y float64) {
	p.y1 = y
}

// MoveToZ sets the z-coordinate of the Point to the specified value.
func (p *Point) MoveToZ(z float64) {
	p.z1 = z
}
