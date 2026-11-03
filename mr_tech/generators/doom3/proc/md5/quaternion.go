package md5

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/geometry"
)

// QuatRotate rotates a 3D vector v by a quaternion q and returns the resulting vector as a new geometry.XYZ.
func QuatRotate(q [4]float64, v geometry.XYZ) geometry.XYZ {
	qx, qy, qz, qw := q[0], q[1], q[2], q[3]

	tx := 2.0 * (qy*v.Z - qz*v.Y)
	ty := 2.0 * (qz*v.X - qx*v.Z)
	tz := 2.0 * (qx*v.Y - qy*v.X)

	rx := v.X + qw*tx + (qy*tz - qz*ty)
	ry := v.Y + qw*ty + (qz*tx - qx*tz)
	rz := v.Z + qw*tz + (qx*ty - qy*tx)

	return geometry.XYZ{X: rx, Y: ry, Z: rz}
}

// QuatMultiplies computes the product of two quaternions q1 and q2, returning the resulting quaternion.
func QuatMultiplies(q1, q2 [4]float64) [4]float64 {
	x1, y1, z1, w1 := q1[0], q1[1], q1[2], q1[3]
	x2, y2, z2, w2 := q2[0], q2[1], q2[2], q2[3]

	return [4]float64{
		w1*x2 + x1*w2 + y1*z2 - z1*y2,
		w1*y2 - x1*z2 + y1*w2 + z1*x2,
		w1*z2 + x1*y2 - y1*x2 + z1*w2,
		w1*w2 - x1*x2 - y1*y2 - z1*z2,
	}
}

// QuatNormalize normalizes a quaternion to ensure it has a unit length, returning the normalized quaternion.
func QuatNormalize(q [4]float64) [4]float64 {
	lenSq := q[0]*q[0] + q[1]*q[1] + q[2]*q[2] + q[3]*q[3]
	if lenSq <= 0.0 {
		return [4]float64{0, 0, 0, 1}
	}
	l := math.Sqrt(lenSq)
	return [4]float64{q[0] / l, q[1] / l, q[2] / l, q[3] / l}
}

// QuatConjugate computes the conjugate of a quaternion, reversing the sign of the x, y, and z components.
func QuatConjugate(q [4]float64) [4]float64 {
	return [4]float64{-q[0], -q[1], -q[2], q[3]}
}
