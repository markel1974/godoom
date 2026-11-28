package collisions

import "math"

// SweptAxisInterval calculates the time interval of penetration in a swept axis-aligned scenario.
// Returns start and end of interval, and a boolean indicating if penetration occurs.
func SweptAxisInterval(rel, sumRad, imp float64) (float64, float64, bool) {
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

// ComputeTOI calculates the time of intersection (TOI) and related intervals for swept AABB collision on three axes.
// Returns entry time, exit time, axis-specific entry times, and a boolean indicating if a valid TOI exists.
func ComputeTOI(relX, relY, relZ, sumRadX, sumRadY, sumRadZ, impX, impY, impZ float64) (float64, float64, float64, float64, float64, bool) {
	// Intervalli temporali completi [tEnter_i, tExit_i] su tutti e tre gli assi (inclusi gli assi già sovrapposti a t = 0)
	tEnterX, tExitX, okX := SweptAxisInterval(relX, sumRadX, impX)
	if !okX {
		return 0, 0, 0, 0, 0, false
	}
	tEnterY, tExitY, okY := SweptAxisInterval(relY, sumRadY, impY)
	if !okY {
		return 0, 0, 0, 0, 0, false
	}
	tEnterZ, tExitZ, okZ := SweptAxisInterval(relZ, sumRadZ, impZ)
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

// SegmentEndpointFeatureMask returns the endpoint or edge feature mask selected by clamped segment parameter t.
func SegmentEndpointFeatureMask(t float64, edgeBit, startMask, endMask uint8) uint8 {
	if t <= 0.0 {
		return startMask
	}
	if t >= 1.0 {
		return endMask
	}
	return edgeBit
}

// ClosestPointsSegmentOriginSegment returns the closest point pair between segment [0 -> u] and segment [a -> b]
// along with their squared distance and clamped segment parameter t in [0, 1] along AB.
func ClosestPointsSegmentOriginSegment(ux, uy, uz, ax, ay, az, bx, by, bz float64) (sx, sy, sz, qx, qy, qz, distSq, t float64) {
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

func OverlapsTriangleEdgeAxes(hx, hy, hz, p0x, p0y, p0z, p1x, p1y, p1z, p2x, p2y, p2z, ex, ey, ez float64) bool {
	absX, absY, absZ := math.Abs(ex), math.Abs(ey), math.Abs(ez)

	if rad := hy*absZ + hz*absY; rad > 1e-12 {
		a0 := -p0y*ez + p0z*ey
		a1 := -p1y*ez + p1z*ey
		a2 := -p2y*ez + p2z*ey
		if math.Min(a0, math.Min(a1, a2)) > rad || math.Max(a0, math.Max(a1, a2)) < -rad {
			return false
		}
	}
	if rad := hx*absZ + hz*absX; rad > 1e-12 {
		a0 := p0x*ez - p0z*ex
		a1 := p1x*ez - p1z*ex
		a2 := p2x*ez - p2z*ex
		if math.Min(a0, math.Min(a1, a2)) > rad || math.Max(a0, math.Max(a1, a2)) < -rad {
			return false
		}
	}
	if rad := hx*absY + hy*absX; rad > 1e-12 {
		a0 := -p0x*ey + p0y*ex
		a1 := -p1x*ey + p1y*ex
		a2 := -p2x*ey + p2y*ex
		if math.Min(a0, math.Min(a1, a2)) > rad || math.Max(a0, math.Max(a1, a2)) < -rad {
			return false
		}
	}
	return true
}
