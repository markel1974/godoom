package geometry

import "math"

// XYZ represents a point or vector in 3D space with X, Y, and Z coordinates.
type XYZ struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
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
