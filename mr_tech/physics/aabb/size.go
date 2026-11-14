package aabb

// Size represents the dimensions of a 3D object with width, height, and depth as float64 values.
type Size struct {
	w1 float64
	h1 float64
	d1 float64
}

// NewSize initializes and returns a pointer to a Size with the specified width, height, and depth.
func NewSize(w float64, h float64, d float64) *Size {
	return &Size{w1: w, h1: h, d1: d}
}

// Set updates the width, height, and depth values of the Size.
func (s *Size) Set(w, h, d float64) {
	s.w1, s.h1, s.d1 = w, h, d
}

// Add increases the width, height, and depth by the given values.
func (s *Size) Add(w, h, d float64) {
	s.w1 += w
	s.h1 += h
	s.d1 += d
}

// Get returns the width, height, and depth of the Size instance.
func (s *Size) Get() (float64, float64, float64) {
	return s.w1, s.h1, s.d1
}

// GetWidth returns the width value stored in the Size object.
func (s *Size) GetWidth() float64 {
	return s.w1
}

// GetHeight returns the height (h) of the Size.
func (s *Size) GetHeight() float64 {
	return s.h1
}

// GetDepth returns the depth component of the Size struct.
func (s *Size) GetDepth() float64 {
	return s.d1
}

// GetCenter calculates and returns the center coordinates (x, y, z) of the Size object as half of its dimensions.
func (s *Size) GetCenter() (float64, float64, float64) {
	return s.w1 * 0.5, s.h1 * 0.5, s.d1 * 0.5
}
