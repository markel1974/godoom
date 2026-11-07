package volumes

import (
	"github.com/markel1974/godoom/mr_tech/geometry"
)

// _im is a predefined identity matrix array representing the standard basis vectors in 3D space.
var _im = [3]geometry.XYZ{
	{X: 1, Y: 0, Z: 0},
	{X: 0, Y: 1, Z: 0},
	{X: 0, Y: 0, Z: 1},
}

// TagAxis represents a 3x3 matrix stored as an array of 3 geometry.XYZ vectors, used for transformations in 3D space.
type TagAxis struct {
	originX float64
	originY float64
	originZ float64
	data    [3]geometry.XYZ
}

// NewTagAxis creates and returns a new TagAxis instance initialized with the provided 3D vectors as its basis data.
// If all provided vectors are zero vectors, the TagAxis is reset to the identity matrix.
func NewTagAxis(origin geometry.XYZ, data [3]geometry.XYZ) *TagAxis {
	a := &TagAxis{
		originX: origin.X, originY: origin.Y, originZ: origin.Z,
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
func (ta *TagAxis) Reset() {
	ta.data[0].X, ta.data[0].Y, ta.data[0].Z = _im[0].X, _im[0].Y, _im[0].Z
	ta.data[1].X, ta.data[1].Y, ta.data[1].Z = _im[1].X, _im[1].Y, _im[1].Z
	ta.data[2].X, ta.data[2].Y, ta.data[2].Z = _im[2].X, _im[2].Y, _im[2].Z
}

// RotatePointByTag performs a 3D vector transformation using the matrix of the current TagAxis and the origin of the input TagAxis.
func (ta *TagAxis) RotatePointByTag(k *TagAxis) (float64, float64, float64) {
	x := k.originX*ta.data[0].X + k.originY*ta.data[1].X + k.originZ*ta.data[2].X
	y := k.originX*ta.data[0].Y + k.originY*ta.data[1].Y + k.originZ*ta.data[2].Y
	z := k.originX*ta.data[0].Z + k.originY*ta.data[1].Z + k.originZ*ta.data[2].Z
	return x, y, z
}

// RotatePoint rotates a point (pX, pY, pZ) based on the TagAxis transformation matrix and returns the transformed coordinates.
func (ta *TagAxis) RotatePoint(pX, pY, pZ float64) (float64, float64, float64) {
	x := pX*ta.data[0].X + pY*ta.data[1].X + pZ*ta.data[2].X
	y := pX*ta.data[0].Y + pY*ta.data[1].Y + pZ*ta.data[2].Y
	z := pX*ta.data[0].Z + pY*ta.data[1].Z + pZ*ta.data[2].Z
	return x, y, z
}

// IsIdentity checks if the TagAxis represents the identity matrix for a 3D Cartesian coordinate system.
func (ta *TagAxis) IsIdentity() bool {
	return ta.data[0].X == 1 && ta.data[0].Y == 0 && ta.data[0].Z == 0 &&
		ta.data[1].X == 0 && ta.data[1].Y == 1 && ta.data[1].Z == 0 &&
		ta.data[2].X == 0 && ta.data[2].Y == 0 && ta.data[2].Z == 1
}

// TransformPointsWithMatrix applies a transformation matrix to points in the source face and stores the result in the destination face.
func (ta *TagAxis) TransformPointsWithMatrix(dst *Face, src *Face, originX, originY, originZ float64) {
	for k := 0; k < 3; k++ {
		rx := src.tri[k].X*ta.data[0].X + src.tri[k].Y*ta.data[1].X + src.tri[k].Z*ta.data[2].X
		ry := src.tri[k].X*ta.data[0].Y + src.tri[k].Y*ta.data[1].Y + src.tri[k].Z*ta.data[2].Y
		rz := src.tri[k].X*ta.data[0].Z + src.tri[k].Y*ta.data[1].Z + src.tri[k].Z*ta.data[2].Z

		dst.tri[k].X = rx + originX
		dst.tri[k].Y = ry + originY
		dst.tri[k].Z = rz + originZ
	}
}

// MultiplyMatrix performs matrix multiplication between the current axis and a child axis, storing the result in the target axis.
func (ta *TagAxis) MultiplyMatrix(child *TagAxis, target *TagAxis) {
	target.data[0].X, target.data[0].Y, target.data[0].Z = ta.RotatePoint(child.data[0].X, child.data[0].Y, child.data[0].Z)
	target.data[1].X, target.data[1].Y, target.data[1].Z = ta.RotatePoint(child.data[1].X, child.data[1].Y, child.data[1].Z)
	target.data[2].X, target.data[2].Y, target.data[2].Z = ta.RotatePoint(child.data[2].X, child.data[2].Y, child.data[2].Z)
}
