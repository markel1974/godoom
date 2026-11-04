package model

import "github.com/markel1974/godoom/mr_tech/geometry"

// _im is a predefined identity matrix array representing the standard basis vectors in 3D space.
var _im = [3]geometry.XYZ{
	{X: 1, Y: 0, Z: 0},
	{X: 0, Y: 1, Z: 0},
	{X: 0, Y: 0, Z: 1},
}

// Tag represents a spatial marker with an origin in 3D space and an associated transformation matrix (Axis).
type Tag struct {
	OriginX float64
	OriginY float64
	OriginZ float64
	Axis    *Axis
}

// NewTag creates a new Tag instance using the given origin coordinates and axis orientation in 3D space.
func NewTag(origin geometry.XYZ, axis [3]geometry.XYZ) *Tag {
	return &Tag{OriginX: origin.X, OriginY: origin.Y, OriginZ: origin.Z, Axis: NewAxis(axis)}
}

// Axis represents a 3x3 matrix stored as an array of 3 geometry.XYZ vectors, used for transformations in 3D space.
type Axis struct {
	data [3]geometry.XYZ
}

// NewAxis creates and returns a new Axis instance initialized with the provided 3D vectors as its basis data.
// If all provided vectors are zero vectors, the Axis is reset to the identity matrix.
func NewAxis(data [3]geometry.XYZ) *Axis {
	a := &Axis{
		data: data,
	}
	if a.data[0].X == 0 && a.data[0].Y == 0 && a.data[0].Z == 0 &&
		a.data[1].X == 0 && a.data[1].Y == 0 && a.data[1].Z == 0 &&
		a.data[2].X == 0 && a.data[2].Y == 0 && a.data[2].Z == 0 {
		a.Reset()
	}
	return a
}

// Reset resets the axis data to the predefined identity matrix (_im), representing standard basis vectors in 3D space.
func (axis *Axis) Reset() {
	axis.data[0].X, axis.data[0].Y, axis.data[0].Z = _im[0].X, _im[0].Y, _im[0].Z
	axis.data[1].X, axis.data[1].Y, axis.data[1].Z = _im[1].X, _im[1].Y, _im[1].Z
	axis.data[2].X, axis.data[2].Y, axis.data[2].Z = _im[2].X, _im[2].Y, _im[2].Z
}

// RotatePoint rotates a point (pX, pY, pZ) based on the Axis transformation matrix and returns the transformed coordinates.
func (axis *Axis) RotatePoint(pX, pY, pZ float64) (float64, float64, float64) {
	x := pX*axis.data[0].X + pY*axis.data[1].X + pZ*axis.data[2].X
	y := pX*axis.data[0].Y + pY*axis.data[1].Y + pZ*axis.data[2].Y
	z := pX*axis.data[0].Z + pY*axis.data[1].Z + pZ*axis.data[2].Z
	return x, y, z
}

// IsIdentity checks if the Axis represents the identity matrix for a 3D Cartesian coordinate system.
func (axis *Axis) IsIdentity() bool {
	return axis.data[0].X == 1 && axis.data[0].Y == 0 && axis.data[0].Z == 0 &&
		axis.data[1].X == 0 && axis.data[1].Y == 1 && axis.data[1].Z == 0 &&
		axis.data[2].X == 0 && axis.data[2].Y == 0 && axis.data[2].Z == 1
}

// TransformPointsWithMatrix applies a transformation matrix to points in the source face and stores the result in the destination face.
func (axis *Axis) TransformPointsWithMatrix(dst *Face, src *Face, originX, originY, originZ float64) {
	for k := 0; k < 3; k++ {
		rx := src.tri[k].X*axis.data[0].X + src.tri[k].Y*axis.data[1].X + src.tri[k].Z*axis.data[2].X
		ry := src.tri[k].X*axis.data[0].Y + src.tri[k].Y*axis.data[1].Y + src.tri[k].Z*axis.data[2].Y
		rz := src.tri[k].X*axis.data[0].Z + src.tri[k].Y*axis.data[1].Z + src.tri[k].Z*axis.data[2].Z

		dst.tri[k].X = rx + originX
		dst.tri[k].Y = ry + originY
		dst.tri[k].Z = rz + originZ
	}
}

// MultiplyMatrix performs matrix multiplication between the current axis and a child axis, storing the result in the target axis.
func (axis *Axis) MultiplyMatrix(child *Axis, target *Axis) {
	target.data[0].X, target.data[0].Y, target.data[0].Z = axis.RotatePoint(child.data[0].X, child.data[0].Y, child.data[0].Z)
	target.data[1].X, target.data[1].Y, target.data[1].Z = axis.RotatePoint(child.data[1].X, child.data[1].Y, child.data[1].Z)
	target.data[2].X, target.data[2].Y, target.data[2].Z = axis.RotatePoint(child.data[2].X, child.data[2].Y, child.data[2].Z)
}
