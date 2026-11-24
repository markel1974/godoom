package geometry

import "math"

// XYZ represents a point or vector in 3D space with X, Y, and Z coordinates.
type XYZ struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

// To32 converts the XYZ coordinates to single-precision XYZ32 values.
func (v *XYZ) To32() XYZ32 {
	return XYZ32{X: float32(v.X), Y: float32(v.Y), Z: float32(v.Z)}
}

// ToFloat64 returns the X, Y, and Z components as float64 values.
func (v *XYZ) ToFloat64() (float64, float64, float64) {
	return v.X, v.Y, v.Z
}

// Scale divides the X, Y and Z components of the XYZ struct by the given scale factor.
func (v *XYZ) Scale(scale XYZ) {
	v.X *= scale.X
	v.Y *= scale.Y
	v.Z *= scale.Z
}

// SubXYZ subtracts two geometry.XYZ vectors component-wise and returns the resulting vector.
func (v *XYZ) SubXYZ(b XYZ) XYZ {
	return XYZ{X: v.X - b.X, Y: v.Y - b.Y, Z: v.Z - b.Z}
}

// DotXYZ computes the dot product of two 3D vectors represented by geometry.XYZ structs.
func (v *XYZ) DotXYZ(b XYZ) float64 { return v.X*b.X + v.Y*b.Y + v.Z*b.Z }

// CrossXYZ computes the cross product of two 3D vectors a and b, returning the resulting vector as geometry.XYZ.
func (v *XYZ) CrossXYZ(b XYZ) XYZ {
	return XYZ{X: v.Y*b.Z - v.Z*b.Y, Y: v.Z*b.X - v.X*b.Z, Z: v.X*b.Y - v.Y*b.X}
}

// NormalizeXYZ normalizes a 3D vector to unit length and returns the resulting vector.
func (v *XYZ) NormalizeXYZ() XYZ {
	lenInv := 1.0 / math.Sqrt(v.X*v.X+v.Y*v.Y+v.Z*v.Z)
	return XYZ{X: v.X * lenInv, Y: v.Y * lenInv, Z: v.Z * lenInv}
}
