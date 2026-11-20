package collisions

import "math"

// sweptAxisInterval calculates the time interval of penetration in a swept axis-aligned scenario.
// Returns start and end of interval, and a boolean indicating if penetration occurs.
func sweptAxisInterval(rel, sumRad, imp float64) (float64, float64, bool) {
	if math.Abs(imp) <= 1e-9 {
		if math.Abs(rel) > sumRad+planeEpsilon {
			return 0, 0, false
		}
		return -math.MaxFloat64, math.MaxFloat64, true
	}
	t1 := (-sumRad - rel) / imp
	t2 := (sumRad - rel) / imp
	if t1 < t2 {
		return t1, t2, true
	}
	return t2, t1, true
}

// computeTOI calculates the time of intersection (TOI) and related intervals for swept AABB collision on three axes.
// Returns entry time, exit time, axis-specific entry times, and a boolean indicating if a valid TOI exists.
func computeTOI(relX, relY, relZ, sumRadX, sumRadY, sumRadZ, impX, impY, impZ float64) (float64, float64, float64, float64, float64, bool) {
	// Intervalli temporali completi [tEnter_i, tExit_i] su tutti e tre gli assi (inclusi gli assi già sovrapposti a t = 0)
	tEnterX, tExitX, okX := sweptAxisInterval(relX, sumRadX, impX)
	if !okX {
		return 0, 0, 0, 0, 0, false
	}
	tEnterY, tExitY, okY := sweptAxisInterval(relY, sumRadY, impY)
	if !okY {
		return 0, 0, 0, 0, 0, false
	}
	tEnterZ, tExitZ, okZ := sweptAxisInterval(relZ, sumRadZ, impZ)
	if !okZ {
		return 0, 0, 0, 0, 0, false
	}
	tEnter := math.Max(tEnterX, math.Max(tEnterY, tEnterZ))
	tExit := math.Min(tExitX, math.Min(tExitY, tExitZ))
	// Condizione necessaria e sufficiente di intersezione temporale AABB in t in [0, 1]
	if tEnter > tExit+toiTieEpsilon || tExit < -toiTieEpsilon || tEnter > 1.0+toiTieEpsilon {
		return 0, 0, 0, 0, 0, false
	}
	return tEnter, tExit, tEnterX, tEnterY, tEnterZ, true
}

func segmentEndpointFeatureMask(t float64, edgeBit, startMask, endMask uint8) uint8 {
	if t <= 0.0 {
		return startMask
	}
	if t >= 1.0 {
		return endMask
	}
	return edgeBit
}

// closestSegmentToTriangleFeature finds the closest point pair between segment [0 -> u] and triangle (a, b, c),
// returning their squared distance, the start-point squared distance at the origin, whether the segment pierces the
// triangle interior, and the 3-bit triangle feature mask (0 for interior).
func closestSegmentToTriangleFeature(ux, uy, uz, ax, ay, az, bx, by, bz, cx, cy, cz float64) (sx, sy, sz, qx, qy, qz, bestSq, startSq float64, pierced bool, featureMask uint8) {
	q0x, q0y, q0z, bestFeat := closestPointOnTriangleFeature(ax, ay, az, bx, by, bz, cx, cy, cz)
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
			dx, dy, dz, featPlane := closestPointOnTriangleFeature(ax-px, ay-py, az-pz, bx-px, by-py, bz-pz, cx-px, cy-py, cz-pz)
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

	q1x, q1y, q1z, feat1 := closestPointOnTriangleFeature(ax-ux, ay-uy, az-uz, bx-ux, by-uy, bz-uz, cx-ux, cy-uy, cz-uz)
	if dSq := q1x*q1x + q1y*q1y + q1z*q1z; dSq < bestSq {
		bestSq = dSq
		sx, sy, sz = ux, uy, uz
		qx, qy, qz = ux+q1x, uy+q1y, uz+q1z
		featureMask = feat1
	}

	if csx, csy, csz, cqx, cqy, cqz, dSq, tSeg := closestPointsSegmentOriginSegment(ux, uy, uz, ax, ay, az, bx, by, bz); dSq < bestSq {
		bestSq = dSq
		sx, sy, sz = csx, csy, csz
		qx, qy, qz = cqx, cqy, cqz
		featureMask = segmentEndpointFeatureMask(tSeg, EdgeABActive, EdgeABActive|EdgeCAActive, EdgeABActive|EdgeBCActive)
	}
	if csx, csy, csz, cqx, cqy, cqz, dSq, tSeg := closestPointsSegmentOriginSegment(ux, uy, uz, bx, by, bz, cx, cy, cz); dSq < bestSq {
		bestSq = dSq
		sx, sy, sz = csx, csy, csz
		qx, qy, qz = cqx, cqy, cqz
		featureMask = segmentEndpointFeatureMask(tSeg, EdgeBCActive, EdgeABActive|EdgeBCActive, EdgeBCActive|EdgeCAActive)
	}
	if csx, csy, csz, cqx, cqy, cqz, dSq, tSeg := closestPointsSegmentOriginSegment(ux, uy, uz, cx, cy, cz, ax, ay, az); dSq < bestSq {
		bestSq = dSq
		sx, sy, sz = csx, csy, csz
		qx, qy, qz = cqx, cqy, cqz
		featureMask = segmentEndpointFeatureMask(tSeg, EdgeCAActive, EdgeBCActive|EdgeCAActive, EdgeABActive|EdgeCAActive)
	}
	return sx, sy, sz, qx, qy, qz, bestSq, startSq, false, featureMask
}

// closestPointOnTriangleFeature returns the closest point on 3D triangle (a, b, c) to the origin
// and a 3-bit mask of active triangle edges touched (0 means the triangle interior).
func closestPointOnTriangleFeature(ax, ay, az, bx, by, bz, cx, cy, cz float64) (qx, qy, qz float64, featureMask uint8) {
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
		q0x, q0y, q0z, f0 := closestPointOnSegmentFeature(ax, ay, az, bx, by, bz, EdgeABActive, EdgeABActive|EdgeCAActive, EdgeABActive|EdgeBCActive)
		bestSq := q0x*q0x + q0y*q0y + q0z*q0z
		q1x, q1y, q1z, f1 := closestPointOnSegmentFeature(ax, ay, az, cx, cy, cz, EdgeCAActive, EdgeABActive|EdgeCAActive, EdgeBCActive|EdgeCAActive)
		if dSq := q1x*q1x + q1y*q1y + q1z*q1z; dSq < bestSq {
			bestSq = dSq
			q0x, q0y, q0z, f0 = q1x, q1y, q1z, f1
		}
		q2x, q2y, q2z, f2 := closestPointOnSegmentFeature(bx, by, bz, cx, cy, cz, EdgeBCActive, EdgeABActive|EdgeBCActive, EdgeBCActive|EdgeCAActive)
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

func isInactiveTriangleFeature(featureMask, activeMask uint8) bool {
	return featureMask != 0 && (featureMask&activeMask) == 0
}

// closestPointOnSegmentFeature returns the point on segment AB closest to the origin and its boundary feature mask.
func closestPointOnSegmentFeature(ax, ay, az, bx, by, bz float64, edgeBit, startMask, endMask uint8) (float64, float64, float64, uint8) {
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

// closestPointsSegmentOriginSegment returns the closest point pair between segment [0 -> u] and segment [a -> b]
// along with their squared distance and clamped segment parameter t in [0, 1] along AB.
func closestPointsSegmentOriginSegment(ux, uy, uz, ax, ay, az, bx, by, bz float64) (sx, sy, sz, qx, qy, qz, distSq, t float64) {
	a := ux*ux + uy*uy + uz*uz
	d2x, d2y, d2z := bx-ax, by-ay, bz-az
	e := d2x*d2x + d2y*d2y + d2z*d2z
	f := -(d2x*ax + d2y*ay + d2z*az)

	var s float64
	if a <= 1e-18 && e <= 1e-18 {
		s, t = 0.0, 0.0
	} else if a <= 1e-18 {
		s = 0.0
		t = math.Max(0.0, math.Min(1.0, f/e))
	} else {
		c := -(ux*ax + uy*ay + uz*az)
		if e <= 1e-18 {
			t = 0.0
			s = math.Max(0.0, math.Min(1.0, -c/a))
		} else {
			b := ux*d2x + uy*d2y + uz*d2z
			if denom := a*e - b*b; denom > 1e-18 {
				s = math.Max(0.0, math.Min(1.0, (b*f-c*e)/denom))
			} else {
				s = 0.0
			}
			t = (b*s + f) / e
			if t < 0.0 {
				t = 0.0
				s = math.Max(0.0, math.Min(1.0, -c/a))
			} else if t > 1.0 {
				t = 1.0
				s = math.Max(0.0, math.Min(1.0, (b-c)/a))
			}
		}
	}

	sx, sy, sz = s*ux, s*uy, s*uz
	qx, qy, qz = ax+t*d2x, ay+t*d2y, az+t*d2z
	dx, dy, dz := sx-qx, sy-qy, sz-qz
	distSq = dx*dx + dy*dy + dz*dz
	return sx, sy, sz, qx, qy, qz, distSq, t
}
