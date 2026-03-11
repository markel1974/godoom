package model

// XY represents a point in 2D space with X and Y coordinates as floating-point numbers.
type XY struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

func (xy *XY) Scale(scale float64) {
	xy.X /= scale
	xy.Y /= scale
}

// XYZ represents a 3D coordinate with x, y, and z components typically used for spatial data.
type XYZ struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

func (xyz *XYZ) ScaleXY(scale float64) {
	xyz.X /= scale
	xyz.Y /= scale
}
