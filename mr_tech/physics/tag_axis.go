package physics

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
	data    *geometry.Triangle3D
}

func NewTagAxisIdentity() *TagAxis {
	return NewTagAxis(geometry.XYZ{}, _im)
}

// NewTagAxis creates and returns a new TagAxis instance initialized with the provided 3D vectors as its basis data.
// If all provided vectors are zero vectors, the TagAxis is reset to the identity matrix.
func NewTagAxis(origin geometry.XYZ, data [3]geometry.XYZ) *TagAxis {
	a := &TagAxis{
		originX: origin.X, originY: origin.Y, originZ: origin.Z,
		data: geometry.NewTriangle3D(data),
	}
	if a.data.IsEmpty() {
		a.Reset()
	}
	return a
}

// Reset resets the axis data to the predefined identity matrix (_im), representing standard basis vectors in 3D space.
func (ta *TagAxis) Reset() {
	ta.data.Points[0].X, ta.data.Points[0].Y, ta.data.Points[0].Z = _im[0].X, _im[0].Y, _im[0].Z
	ta.data.Points[1].X, ta.data.Points[1].Y, ta.data.Points[1].Z = _im[1].X, _im[1].Y, _im[1].Z
	ta.data.Points[2].X, ta.data.Points[2].Y, ta.data.Points[2].Z = _im[2].X, _im[2].Y, _im[2].Z
}

// SetFromQuaternion populates the 3x3 basis axes directly from the supplied orientation quaternion.
func (ta *TagAxis) SetFromQuaternion(q *Quaternion) {
	if q == nil {
		ta.Reset()
		return
	}
	ta.data.Points[0].X, ta.data.Points[0].Y, ta.data.Points[0].Z,
		ta.data.Points[1].X, ta.data.Points[1].Y, ta.data.Points[1].Z,
		ta.data.Points[2].X, ta.data.Points[2].Y, ta.data.Points[2].Z = q.ToAxes()
}

// ToQuaternion extracts the 3x3 basis orientation of the TagAxis into the supplied Quaternion without allocating.
func (ta *TagAxis) ToQuaternion(q *Quaternion) {
	if q == nil {
		return
	}
	q.SetFromAxes(
		ta.data.Points[0].X, ta.data.Points[0].Y, ta.data.Points[0].Z,
		ta.data.Points[1].X, ta.data.Points[1].Y, ta.data.Points[1].Z,
		ta.data.Points[2].X, ta.data.Points[2].Y, ta.data.Points[2].Z,
	)
}

// RotatePointByTag performs a 3D vector transformation using the matrix of the current TagAxis and the origin of the input TagAxis.
func (ta *TagAxis) RotatePointByTag(k *TagAxis) (float64, float64, float64) {
	x := k.originX*ta.data.Points[0].X + k.originY*ta.data.Points[1].X + k.originZ*ta.data.Points[2].X
	y := k.originX*ta.data.Points[0].Y + k.originY*ta.data.Points[1].Y + k.originZ*ta.data.Points[2].Y
	z := k.originX*ta.data.Points[0].Z + k.originY*ta.data.Points[1].Z + k.originZ*ta.data.Points[2].Z
	return x, y, z
}

// RotatePoint rotates a point (pX, pY, pZ) based on the TagAxis transformation matrix and returns the transformed coordinates.
func (ta *TagAxis) RotatePoint(pX, pY, pZ float64) (float64, float64, float64) {
	return ta.data.RotatePoint(pX, pY, pZ)
}

// RotatePointInverse transforms a vector (pX, pY, pZ) by the transpose (inverse) of the TagAxis 3x3 basis matrix.
func (ta *TagAxis) RotatePointInverse(pX, pY, pZ float64) (float64, float64, float64) {
	return ta.data.RotatePointInverse(pX, pY, pZ)
}

// IsIdentity checks if the TagAxis represents the identity matrix for a 3D Cartesian coordinate system.
func (ta *TagAxis) IsIdentity() bool {
	return ta.data.Points[0].X == 1 && ta.data.Points[0].Y == 0 && ta.data.Points[0].Z == 0 &&
		ta.data.Points[1].X == 0 && ta.data.Points[1].Y == 1 && ta.data.Points[1].Z == 0 &&
		ta.data.Points[2].X == 0 && ta.data.Points[2].Y == 0 && ta.data.Points[2].Z == 1
}

// TransformPointsWithMatrix applies a transformation matrix to points in the source face and stores the result in the destination face.
func (ta *TagAxis) TransformPointsWithMatrix(dst *geometry.Triangle3D, src *geometry.Triangle3D, originX, originY, originZ float64) {
	ta.data.TransformPointsWithMatrix(dst, src, originX, originY, originZ)
}

// MultiplyMatrix performs matrix multiplication between the current axis and a child axis, storing the result in the target axis.
func (ta *TagAxis) MultiplyMatrix(c *TagAxis, t *TagAxis) {
	child := c.data
	target := t.data
	target.Points[0].X, target.Points[0].Y, target.Points[0].Z = ta.data.RotatePoint(child.Points[0].X, child.Points[0].Y, child.Points[0].Z)
	target.Points[1].X, target.Points[1].Y, target.Points[1].Z = ta.data.RotatePoint(child.Points[1].X, child.Points[1].Y, child.Points[1].Z)
	target.Points[2].X, target.Points[2].Y, target.Points[2].Z = ta.data.RotatePoint(child.Points[2].X, child.Points[2].Y, child.Points[2].Z)
}
