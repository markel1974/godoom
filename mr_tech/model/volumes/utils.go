package volumes

import (
	"math"
)

// NormalizeXYZ normalizes a given 3D vector to unit length; returns fallback if the vector's length is near zero.
func NormalizeXYZ(vX, vY, vZ float64, fX, fY, fZ float64) (float64, float64, float64) {
	l := math.Sqrt(vX*vX + vY*vY + vZ*vZ)
	if l > 1e-12 {
		inv := 1.0 / l
		return vX * inv, vY * inv, vZ * inv
	}
	return fX, fY, fZ
}

// FallbackTangent computes a fallback tangent vector orthogonal to the input normal vector n in 3D space.
func FallbackTangent(nX, nY, nZ float64) (float64, float64, float64) {
	var upX, upY, upZ float64
	if math.Abs(nZ) > 0.99 {
		upX, upY, upZ = 1, 0, 0
	} else {
		upX, upY, upZ = 0, 0, 1
	}
	// up x n
	tx := upY*nZ - upZ*nY
	ty := upZ*nX - upX*nZ
	tz := upX*nY - upY*nX
	return NormalizeXYZ(tx, ty, tz, 1, 0, 0)
}
