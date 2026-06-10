package physics

// Size represents the dimensions of a 3D object with width, height, and depth as float64 values.
type Size struct {
	w float64
	h float64
	d float64
}

// NewSize initializes and returns a pointer to a Size with the specified width, height, and depth.
func NewSize(w float64, h float64, d float64) *Size {
	return &Size{w: w, h: h, d: d}
}

// Get returns the width, height, and depth of the Size instance.
func (s *Size) Get() (float64, float64, float64) {
	return s.w, s.h, s.d
}

// GetWidth returns the width value stored in the Size object.
func (s *Size) GetWidth() float64 {
	return s.w
}

// GetHeight returns the height (h) of the Size.
func (s *Size) GetHeight() float64 {
	return s.h
}

// GetDepth returns the depth component of the Size struct.
func (s *Size) GetDepth() float64 {
	return s.d
}

// GetCenter calculates and returns the center coordinates (x, y, z) of the Size object as half of its dimensions.
func (s *Size) GetCenter() (float64, float64, float64) {
	return s.w * 0.5, s.h * 0.5, s.d * 0.5
}
