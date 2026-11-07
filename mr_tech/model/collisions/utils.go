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

// isDiscreteMTV determines if a box satisfies discrete Minimum Translation Vector (MTV) criteria under SAT constraints.
func isDiscreteMTV(prevA, boxA, relA, impA, prevB, boxB, prevC, boxC float64) bool {
	if prevA > prevB+satFilterEpsilon || prevA > prevC+satFilterEpsilon {
		return false
	}
	tieB := math.Abs(prevA-prevB) <= satFilterEpsilon
	tieC := math.Abs(prevA-prevC) <= satFilterEpsilon
	if tieB && boxA > boxB+satFilterEpsilon && relA*impA >= -planeEpsilon {
		return false
	}
	if tieC && boxA > boxC+satFilterEpsilon && relA*impA >= -planeEpsilon {
		return false
	}
	return true
}
