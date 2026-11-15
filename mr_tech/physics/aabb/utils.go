package aabb

import "math"

// ClampFloat returns v constrained to the inclusive range from minV to maxV.
func ClampFloat(v, minV, maxV float64) float64 {
	if v < minV {
		return minV
	}
	if v > maxV {
		return maxV
	}
	return v
}

// PointDistSq returns the squared Euclidean distance between two 3D points.
func PointDistSq(aX, aY, aZ, bX, bY, bZ float64) float64 {
	dx, dy, dz := aX-bX, aY-bY, aZ-bZ
	return dx*dx + dy*dy + dz*dz
}

// AxisHalfExtent returns the half extent for axis index idx: x for 0, y for 1, and z otherwise.
func AxisHalfExtent(x, y, z float64, idx int) float64 {
	switch idx {
	case 0:
		return x
	case 1:
		return y
	default:
		return z
	}
}

// EdgeMidpointAnchor returns an anchor point at the midpoint of a box edge selected by skipIdx and direction.
func EdgeMidpointAnchor(cx, cy, cz float64, axes *Axes, hx, hy, hz float64, skipIdx int, dirX, dirY, dirZ, tieX, tieY, tieZ float64) (float64, float64, float64) {
	px, py, pz := cx, cy, cz
	for k := 0; k < 3; k++ {
		if k == skipIdx {
			continue
		}
		ax, ay, az := axes.GetRow(k)
		step := -AxisHalfExtent(hx, hy, hz, k)
		dot := dirX*ax + dirY*ay + dirZ*az
		if math.Abs(dot) <= 1e-12 {
			dot = tieX*ax + tieY*ay + tieZ*az
		}
		if dot > 0.0 {
			step = -step
		}
		px += ax * step
		py += ay * step
		pz += az * step
	}
	return px, py, pz
}
