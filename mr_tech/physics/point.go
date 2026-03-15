package physics

// Point represents a 2D point in Cartesian coordinates with x and y as float64 values.
type Point struct {
	x float64
	y float64
}

// NewPointFloat creates and returns a new Point instance initialized with the specified x and y float64 values.
func NewPointFloat(x float64, y float64) Point {
	return Point{
		x: x,
		y: y,
	}
}

// AddTo adjusts the coordinates of the Point by adding the given x and y values to the current coordinates.
func (p *Point) AddTo(x float64, y float64) {
	p.x += x
	p.y += y
}

// AddToX increments the x-coordinate of the Point by the given value.
func (p *Point) AddToX(x float64) {
	p.x += x
}

// AddToY adds the given value to the y-coordinate of the Point.
func (p *Point) AddToY(y float64) {
	p.y += y
}

// MoveTo sets the x and y coordinates of the Point to the specified values.
func (p *Point) MoveTo(x float64, y float64) {
	p.x = x
	p.y = y
}

// MoveToX sets the x-coordinate of the Point to the specified value.
func (p *Point) MoveToX(x float64) {
	p.x = x
}

// MoveToY sets the y-coordinate of the Point to the specified value.
func (p *Point) MoveToY(y float64) {
	p.y = y
}

// GetX returns the x-coordinate of the Point.
func (p *Point) GetX() float64 {
	return p.x
}

// GetY returns the y-coordinate of the point.
func (p *Point) GetY() float64 {
	return p.y
}
