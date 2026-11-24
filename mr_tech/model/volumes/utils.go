package volumes

import (
	"math"
)

// NormalizeXYZ normalizes a 3D float32 vector to unit length using float64 intermediate precision;
// returns fallback if the vector length is near zero.
func NormalizeXYZ(vX, vY, vZ, fX, fY, fZ float32) (float32, float32, float32) {
	dx, dy, dz := float64(vX), float64(vY), float64(vZ)
	l := math.Sqrt(dx*dx + dy*dy + dz*dz)
	if l > 1e-6 {
		inv := 1.0 / l
		return float32(dx * inv), float32(dy * inv), float32(dz * inv)
	}
	return fX, fY, fZ
}

// NormalizeXYZ64 normalizes a 3D float64 vector to unit length and converts it to float32,
// preserving direction for small magnitudes before float32 narrowing.
func NormalizeXYZ64(vX, vY, vZ float64, fX, fY, fZ float32) (float32, float32, float32) {
	l := math.Sqrt(vX*vX + vY*vY + vZ*vZ)
	if l > 1e-12 {
		inv := 1.0 / l
		return float32(vX * inv), float32(vY * inv), float32(vZ * inv)
	}
	return fX, fY, fZ
}

// FallbackTangent computes a fallback tangent vector orthogonal to the input normal vector n in 3D space.
func FallbackTangent(nX, nY, nZ float32) (float32, float32, float32) {
	var upX, upY, upZ float32
	if math.Abs(float64(nZ)) > 0.99 {
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
