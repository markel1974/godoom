package geometry

import (
	"math"
	"sort"
)

// ComplexPolygon represents a polygon with an outer boundary and optional inner holes.
// The outer boundary is defined by the Outer field, while the holes are defined by the Holes field.
type ComplexPolygon struct {
	Outer Polygon
	Holes []Polygon
}

// BridgeHoles bridges interior polygon holes to form a single, contiguous outer boundary, preserving topology.
func (cp *ComplexPolygon) BridgeHoles() Polygon {
	if len(cp.Holes) == 0 {
		return cp.Outer
	}

	// Optimization 1: Exact calculation of final capacity to eliminate dynamic reallocations
	totalLen := len(cp.Outer)
	for _, h := range cp.Holes {
		totalLen += len(h) + 2 // +2 for bridge vertices (forward and return)
	}

	outer := make(Polygon, len(cp.Outer), totalLen)
	copy(outer, cp.Outer)

	// Sort holes from right to left to ensure topological consistency in bridging
	sort.Slice(cp.Holes, func(i, j int) bool {
		return cp.Holes[i].MaxPointsX() > cp.Holes[j].MaxPointsX()
	})

	for _, hole := range cp.Holes {
		holeIdx := 0
		mX := hole[0].X
		for i := 1; i < len(hole); i++ {
			if hole[i].X > mX {
				mX = hole[i].X
				holeIdx = i
			}
		}
		holePoint := hole[holeIdx]
		bestOuterIdx := -1
		minDist := math.MaxFloat64

		for i, op := range outer {
			if op.X < holePoint.X {
				continue
			}

			// Optimization 2: Fast rejection. Calculate distance in O(1) before
			// launching isVisible (which is O(N) for each segment).
			if dist := DistanceSq(holePoint, op); dist < minDist {
				if HasLineOfSight(holePoint, op, hole, outer) {
					minDist = dist
					bestOuterIdx = i
				}
			}
		}

		// Topological fallback for non-manifold sectors or anomalous intersections
		if bestOuterIdx == -1 {
			bestOuterIdx = 0
			for i, op := range outer {
				if dist := DistanceSq(holePoint, op); dist < minDist {
					minDist = dist
					bestOuterIdx = i
				}
			}
		}

		// Optimization 3: In-place memory shifting leveraging pre-allocated capacity.
		// No additional heap allocation for new bridges.
		oldLen := len(outer)
		spliceLen := len(hole) + 2
		outer = append(outer, make(Polygon, spliceLen)...)

		// Forward shift of elements to the right of the insertion point
		copy(outer[bestOuterIdx+1+spliceLen:], outer[bestOuterIdx+1:oldLen])

		// Linear reconstruction of the bridge
		insertPos := bestOuterIdx + 1
		for i := 0; i < len(hole); i++ {
			outer[insertPos+i] = hole[(holeIdx+i)%len(hole)]
		}
		outer[insertPos+len(hole)] = holePoint
		outer[insertPos+len(hole)+1] = outer[bestOuterIdx]
	}

	return outer
}
