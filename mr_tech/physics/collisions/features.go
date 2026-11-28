package collisions

import "math"

// ClosestSegmentToTriangleFeature finds the closest point pair between segment [0 -> u] and triangle (a, b, c),
// returning their squared distance, the start-point squared distance at the origin, whether the segment pierces the
// triangle interior, and the 3-bit triangle feature mask (0 for interior).
func ClosestSegmentToTriangleFeature(ux, uy, uz, ax, ay, az, bx, by, bz, cx, cy, cz float64) (sx, sy, sz, qx, qy, qz, bestSq, startSq float64, pierced bool, featureMask uint8) {
	q0x, q0y, q0z, bestFeat := ClosestPointOnTriangleFeature(ax, ay, az, bx, by, bz, cx, cy, cz)
	startSq = q0x*q0x + q0y*q0y + q0z*q0z
	bestSq = startSq
	sx, sy, sz = 0.0, 0.0, 0.0
	qx, qy, qz = q0x, q0y, q0z
	featureMask = bestFeat

	abx, aby, abz := bx-ax, by-ay, bz-az
	acx, acy, acz := cx-ax, cy-ay, cz-az
	nx := aby*acz - abz*acy
	ny := abz*acx - abx*acz
	nz := abx*acy - aby*acx
	if denom := nx*ux + ny*uy + nz*uz; math.Abs(denom) > 1e-12 {
		tau := (nx*ax + ny*ay + nz*az) / denom
		if tau >= -1e-9 && tau <= 1.0+1e-9 {
			if tau < 0.0 {
				tau = 0.0
			} else if tau > 1.0 {
				tau = 1.0
			}
			px, py, pz := tau*ux, tau*uy, tau*uz
			dx, dy, dz, featPlane := ClosestPointOnTriangleFeature(ax-px, ay-py, az-pz, bx-px, by-py, bz-pz, cx-px, cy-py, cz-pz)
			dSq := dx*dx + dy*dy + dz*dz
			if featPlane == 0 || dSq <= 1e-18 {
				return px, py, pz, px, py, pz, 0.0, startSq, true, 0
			}
			if dSq < bestSq {
				bestSq = dSq
				sx, sy, sz = px, py, pz
				qx, qy, qz = px+dx, py+dy, pz+dz
				featureMask = featPlane
			}
		}
	}

	q1x, q1y, q1z, feat1 := ClosestPointOnTriangleFeature(ax-ux, ay-uy, az-uz, bx-ux, by-uy, bz-uz, cx-ux, cy-uy, cz-uz)
	if dSq := q1x*q1x + q1y*q1y + q1z*q1z; dSq < bestSq {
		bestSq = dSq
		sx, sy, sz = ux, uy, uz
		qx, qy, qz = ux+q1x, uy+q1y, uz+q1z
		featureMask = feat1
	}

	if csx, csy, csz, cqx, cqy, cqz, dSq, tSeg := ClosestPointsSegmentOriginSegment(ux, uy, uz, ax, ay, az, bx, by, bz); dSq < bestSq {
		bestSq = dSq
		sx, sy, sz = csx, csy, csz
		qx, qy, qz = cqx, cqy, cqz
		featureMask = SegmentEndpointFeatureMask(tSeg, EdgeABActive, EdgeABActive|EdgeCAActive, EdgeABActive|EdgeBCActive)
	}
	if csx, csy, csz, cqx, cqy, cqz, dSq, tSeg := ClosestPointsSegmentOriginSegment(ux, uy, uz, bx, by, bz, cx, cy, cz); dSq < bestSq {
		bestSq = dSq
		sx, sy, sz = csx, csy, csz
		qx, qy, qz = cqx, cqy, cqz
		featureMask = SegmentEndpointFeatureMask(tSeg, EdgeBCActive, EdgeABActive|EdgeBCActive, EdgeBCActive|EdgeCAActive)
	}
	if csx, csy, csz, cqx, cqy, cqz, dSq, tSeg := ClosestPointsSegmentOriginSegment(ux, uy, uz, cx, cy, cz, ax, ay, az); dSq < bestSq {
		bestSq = dSq
		sx, sy, sz = csx, csy, csz
		qx, qy, qz = cqx, cqy, cqz
		featureMask = SegmentEndpointFeatureMask(tSeg, EdgeCAActive, EdgeBCActive|EdgeCAActive, EdgeABActive|EdgeCAActive)
	}
	return sx, sy, sz, qx, qy, qz, bestSq, startSq, false, featureMask
}

// ClosestPointOnTriangleFeature returns the closest point on 3D triangle (a, b, c) to the origin
// and a 3-bit mask of active triangle edges touched (0 means the triangle interior).
func ClosestPointOnTriangleFeature(ax, ay, az, bx, by, bz, cx, cy, cz float64) (qx, qy, qz float64, featureMask uint8) {
	abx, aby, abz := bx-ax, by-ay, bz-az
	acx, acy, acz := cx-ax, cy-ay, cz-az
	apx, apy, apz := -ax, -ay, -az
	d1 := abx*apx + aby*apy + abz*apz
	d2 := acx*apx + acy*apy + acz*apz
	if d1 <= 0.0 && d2 <= 0.0 {
		return ax, ay, az, EdgeABActive | EdgeCAActive
	}

	bpx, bpy, bpz := -bx, -by, -bz
	d3 := abx*bpx + aby*bpy + abz*bpz
	d4 := acx*bpx + acy*bpy + acz*bpz
	if d3 >= 0.0 && d4 <= d3 {
		return bx, by, bz, EdgeABActive | EdgeBCActive
	}

	vc := d1*d4 - d3*d2
	if vc <= 0.0 && d1 >= 0.0 && d3 <= 0.0 {
		v := d1 / (d1 - d3)
		return ax + v*abx, ay + v*aby, az + v*abz, EdgeABActive
	}

	cpx, cpy, cpz := -cx, -cy, -cz
	d5 := abx*cpx + aby*cpy + abz*cpz
	d6 := acx*cpx + acy*cpy + acz*cpz
	if d6 >= 0.0 && d5 <= d6 {
		return cx, cy, cz, EdgeBCActive | EdgeCAActive
	}

	vb := d5*d2 - d1*d6
	if vb <= 0.0 && d2 >= 0.0 && d6 <= 0.0 {
		w := d2 / (d2 - d6)
		return ax + w*acx, ay + w*acy, az + w*acz, EdgeCAActive
	}

	va := d3*d6 - d5*d4
	if va <= 0.0 && (d4-d3) >= 0.0 && (d5-d6) >= 0.0 {
		w := (d4 - d3) / ((d4 - d3) + (d5 - d6))
		return bx + w*(cx-bx), by + w*(cy-by), bz + w*(cz-bz), EdgeBCActive
	}

	denom := va + vb + vc
	if denom <= 1e-18 {
		q0x, q0y, q0z, f0 := ClosestPointOnSegmentFeature(ax, ay, az, bx, by, bz, EdgeABActive, EdgeABActive|EdgeCAActive, EdgeABActive|EdgeBCActive)
		bestSq := q0x*q0x + q0y*q0y + q0z*q0z
		q1x, q1y, q1z, f1 := ClosestPointOnSegmentFeature(ax, ay, az, cx, cy, cz, EdgeCAActive, EdgeABActive|EdgeCAActive, EdgeBCActive|EdgeCAActive)
		if dSq := q1x*q1x + q1y*q1y + q1z*q1z; dSq < bestSq {
			bestSq = dSq
			q0x, q0y, q0z, f0 = q1x, q1y, q1z, f1
		}
		q2x, q2y, q2z, f2 := ClosestPointOnSegmentFeature(bx, by, bz, cx, cy, cz, EdgeBCActive, EdgeABActive|EdgeBCActive, EdgeBCActive|EdgeCAActive)
		if dSq := q2x*q2x + q2y*q2y + q2z*q2z; dSq < bestSq {
			q0x, q0y, q0z, f0 = q2x, q2y, q2z, f2
		}
		return q0x, q0y, q0z, f0
	}

	invDenom := 1.0 / denom
	v := vb * invDenom
	w := vc * invDenom
	return ax + abx*v + acx*w, ay + aby*v + acy*w, az + abz*v + acz*w, 0
}

// IsInactiveTriangleFeature reports whether a nonzero triangle feature mask has no bits in the active mask.
func IsInactiveTriangleFeature(featureMask, activeMask uint8) bool {
	return featureMask != 0 && (featureMask&activeMask) == 0
}

// ClosestPointOnSegmentFeature returns the point on segment AB closest to the origin and its boundary feature mask.
func ClosestPointOnSegmentFeature(ax, ay, az, bx, by, bz float64, edgeBit, startMask, endMask uint8) (float64, float64, float64, uint8) {
	abx, aby, abz := bx-ax, by-ay, bz-az
	denom := abx*abx + aby*aby + abz*abz
	if denom <= 1e-18 {
		return ax, ay, az, startMask
	}
	t := -(ax*abx + ay*aby + az*abz) / denom
	if t <= 0.0 {
		return ax, ay, az, startMask
	}
	if t >= 1.0 {
		return bx, by, bz, endMask
	}
	return ax + t*abx, ay + t*aby, az + t*abz, edgeBit
}
