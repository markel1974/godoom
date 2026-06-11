package physics

// Point represents a 3D point in space with x, y, and z coordinates as float64 values.
type Point struct {
	x float64
	y float64
	z float64
}

// NewPoint creates a new Point with the specified x, y, and z coordinates as float64 values.
func NewPoint(x float64, y float64, z float64) Point {
	return Point{
		x: x,
		y: y,
		z: z,
	}
}

// AddTo updates the x and y coordinates of the Point by adding the given values to the current coordinates.
func (p *Point) AddTo(x float64, y float64, z float64) {
	p.x += x
	p.y += y
	p.z += z
}

// AddToX increments the x-coordinate of the Point by the specified value.
func (p *Point) AddToX(x float64) {
	p.x += x
}

// AddToY increments the y-coordinate of the Point by the specified value.
func (p *Point) AddToY(y float64) {
	p.y += y
}

// AddToZ increments the z-coordinate of the Point by the given value.
func (p *Point) AddToZ(z float64) {
	p.z += z
}

// MoveTo updates the coordinates of the point to the specified x, y, and z values.
func (p *Point) MoveTo(x float64, y float64, z float64) {
	p.x = x
	p.y = y
	p.z = z
}

// MoveToX updates the x-coordinate of the Point to the specified value.
func (p *Point) MoveToX(x float64) {
	p.x = x
}

// MoveToY sets the y-coordinate of the Point to the specified value.
func (p *Point) MoveToY(y float64) {
	p.y = y
}

// MoveToZ sets the z-coordinate of the Point to the specified value.
func (p *Point) MoveToZ(z float64) {
	p.z = z
}

// GetX returns the x-coordinate of the Point.
func (p *Point) GetX() float64 {
	return p.x
}

// GetY returns the y-coordinate value of the Point.
func (p *Point) GetY() float64 {
	return p.y
}

// GetZ returns the z-coordinate value of the Point.
func (p *Point) GetZ() float64 {
	return p.z
}
