package geometry

import "math"

// XYZ32 represents a 3D vector with single-precision floating-point components X, Y, and Z.
type XYZ32 struct {
	X float32 `json:"x"`
	Y float32 `json:"y"`
	Z float32 `json:"z"`
}

// Scale multiplies the X, Y, and Z fields of the current XYZ32 instance by the respective fields of the provided scale parameter.
func (v *XYZ32) Scale(scale XYZ32) {
	v.X *= scale.X
	v.Y *= scale.Y
	v.Z *= scale.Z
}

// SubXYZ subtracts the components of the given XYZ32 vector b from the current vector and returns the resulting XYZ32.
func (v *XYZ32) SubXYZ(b XYZ32) XYZ32 {
	return XYZ32{X: v.X - b.X, Y: v.Y - b.Y, Z: v.Z - b.Z}
}

// DotXYZ calculates and returns the dot product of the current XYZ32 vector and the provided XYZ32 vector.
func (v *XYZ32) DotXYZ(b XYZ32) float32 { return v.X*b.X + v.Y*b.Y + v.Z*b.Z }

// CrossXYZ computes the cross product of the current XYZ32 vector and the given XYZ32 vector b, returning a new XYZ32 vector.
func (v *XYZ32) CrossXYZ(b XYZ32) XYZ32 {
	return XYZ32{X: v.Y*b.Z - v.Z*b.Y, Y: v.Z*b.X - v.X*b.Z, Z: v.X*b.Y - v.Y*b.X}
}

// NormalizeXYZ normalizes the XYZ32 vector and returns a new normalized vector with a magnitude of 1.
func (v *XYZ32) NormalizeXYZ() XYZ32 {
	lenInv := float32(1.0 / math.Sqrt(float64(v.X*v.X+v.Y*v.Y+v.Z*v.Z)))
	return XYZ32{X: v.X * lenInv, Y: v.Y * lenInv, Z: v.Z * lenInv}
}
