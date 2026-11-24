package volumes

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/geometry"
)

// normalizeXYZ normalizes a given 3D vector to unit length; returns fallback if the vector's length is near zero.
func normalizeXYZ(v, fallback geometry.XYZ) geometry.XYZ {
	l := math.Sqrt(v.X*v.X + v.Y*v.Y + v.Z*v.Z)
	if l > 1e-12 {
		inv := 1.0 / l
		return geometry.XYZ{X: v.X * inv, Y: v.Y * inv, Z: v.Z * inv}
	}
	return fallback
}

// fallbackTangent computes a fallback tangent vector orthogonal to the input normal vector n in 3D space.
func fallbackTangent(n geometry.XYZ) geometry.XYZ {
	up := geometry.XYZ{X: 0, Y: 0, Z: 1}
	if math.Abs(n.Z) > 0.99 {
		up = geometry.XYZ{X: 1, Y: 0, Z: 0}
	}
	// up x n
	tx := up.Y*n.Z - up.Z*n.Y
	ty := up.Z*n.X - up.X*n.Z
	tz := up.X*n.Y - up.Y*n.X
	return normalizeXYZ(geometry.XYZ{X: tx, Y: ty, Z: tz}, geometry.XYZ{X: 1, Y: 0, Z: 0})
}
