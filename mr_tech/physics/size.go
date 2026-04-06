package physics

// Size represents the dimensions of an object with width and height as float64 fields.
type Size struct {
	w float64
	h float64
	d float64
}

// NewSize creates and returns a new Size instance initialized with the specified width and height.
func NewSize(w float64, h float64, d float64) Size {
	return Size{w: w, h: h, d: d}
}

// GetWidth returns the width value of the Size structure.
func (s *Size) GetWidth() float64 {
	return s.w
}

// GetHeight returns the height (h) of the Size structure.
func (s *Size) GetHeight() float64 {
	return s.h
}

// GetDepth returns the depth (d) value of the Size structure.
func (s *Size) GetDepth() float64 {
	return s.d
}
