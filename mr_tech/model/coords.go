package model

// XY represents a 2D point or vector with X and Y coordinates in floating-point precision.
type XY struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// Scale adjusts the X and Y components of the XY instance by dividing them by the given scale factor.
func (xy *XY) Scale(scale float64) {
	xy.X /= scale
	xy.Y /= scale
}

// XYZ represents a point or vector in 3D space with X, Y, and Z coordinates.
type XYZ struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

// ScaleXY divides the X and Y components of the XYZ struct by the given scale factor.
func (xyz *XYZ) ScaleXY(scale float64) {
	xyz.X /= scale
	xyz.Y /= scale
}
