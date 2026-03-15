package geometry

// Package wad provides utilities for handling WAD-based geometry, specifically
// focusing on polygon manipulation, hole merging, and constrained Delaunay triangulation.
// It includes robust geometric predicates (Orient2D, InCircle) with exact arithmetic
// fallbacks to handle floating-point precision issues common in legacy map data.

import (
	"fmt"
	"math"
	"math/big"
)

// epsilon represents the machine epsilon for float64 based on IEEE 754.
// errBoundOri is the precomputed error bound for orientation-based determinant calculations.
// errBoundInc is the precomputed error bound for incircle determinant calculations.
const (
	// Machine epsilon for float64 IEEE 754
	epsilon = 1.1102230246251565e-16
	// Precomputed error bounds for determinants
	errBoundOri = (3.0 + 16.0*epsilon) * epsilon
	errBoundInc = (10.0 + 96.0*epsilon) * epsilon
)

// BigFloat creates a new *big.Float with 256 bits of precision and sets its value to the given float64.
func BigFloat(f float64) *big.Float {
	return new(big.Float).SetPrec(256).SetFloat64(f)
}

// RecoverConstraints ensures that a set of constraints is respected in a Delaunay triangulated mesh by edge flipping.
func RecoverConstraints(constraints [][2]Point, triangles []Triangle, secIdx int) []Triangle {
	for _, c := range constraints {
		var queue [][2]Point

		// 1. Enqueue edges that strictly intersect the constraint
		for _, t := range triangles {
			edges := [3][2]Point{{t.A, t.B}, {t.B, t.C}, {t.C, t.A}}
			for _, e := range edges {
				if SegmentsCross(e[0], e[1], c[0], c[1]) {
					queue = AppendUniqueEdge(queue, e)
				}
			}
		}

		// 2. Guaranteed topological resolution with failsafe
		consecutiveFailures := 0

		for len(queue) > 0 {
			if consecutiveFailures >= len(queue) {
				// Break the loop to preserve execution.
				fmt.Println("WARNING Topological deadlock: queue contains only non-flippable edges")
				break
			}

			e := queue[0]
			queue = queue[1:]

			t1Idx, t2Idx := FindAdjacentTriangles(triangles, e)
			if t1Idx == -1 || t2Idx == -1 {
				continue
			}

			t1, t2 := triangles[t1Idx], triangles[t2Idx]
			pOpp1 := t1.GetOppositeVertex(e)
			pOpp2 := t2.GetOppositeVertex(e)

			// Only diagonals of strictly convex quadrilaterals can be flipped
			if IsConvexQuadrilateral(e[0], e[1], pOpp1, pOpp2) {
				consecutiveFailures = 0 // Reset on success
				triangles[t1Idx] = Triangle{pOpp1, pOpp2, e[0]}
				triangles[t2Idx] = Triangle{pOpp1, pOpp2, e[1]}

				newEdge := [2]Point{pOpp1, pOpp2}
				if SegmentsCross(newEdge[0], newEdge[1], c[0], c[1]) {
					queue = append(queue, newEdge)
				} else {
					// Evaluate if the new quadrilateral edges intersect the constraint
					for _, newBoundary := range [][2]Point{{pOpp1, e[0]}, {e[0], pOpp2}, {pOpp2, e[1]}, {e[1], pOpp1}} {
						if SegmentsCross(newBoundary[0], newBoundary[1], c[0], c[1]) {
							queue = AppendUniqueEdge(queue, newBoundary)
						}
					}
				}
			} else {
				// The quadrilateral is not convex. Reinsert it at the back of the queue.
				consecutiveFailures++
				queue = append(queue, e)
			}
		}
	}
	return triangles
}

// DistanceSq calculates the squared distance between two points p1 and p2 in a 2D Cartesian plane.
func DistanceSq(p1 Point, p2 Point) float64 {
	dx := p1.X - p2.X
	dy := p1.Y - p2.Y
	return dx*dx + dy*dy
}

// HasLineOfSight determines if two points have a clear line of sight, considering obstacles in the form of polygons.
func HasLineOfSight(p1 Point, p2 Point, hole Polygon, outer Polygon) bool {
	for i := 0; i < len(outer); i++ {
		e1, e2 := outer[i], outer[(i+1)%len(outer)]
		if e1 == p1 || e1 == p2 || e2 == p1 || e2 == p2 {
			continue
		}
		if SegmentsIntersect(p1, p2, e1, e2) {
			return false
		}
	}
	for i := 0; i < len(hole); i++ {
		e1, e2 := hole[i], hole[(i+1)%len(hole)]
		if e1 == p1 || e1 == p2 || e2 == p1 || e2 == p2 {
			continue
		}
		if SegmentsIntersect(p1, p2, e1, e2) {
			return false
		}
	}
	return true
}

// AppendUniqueEdge appends the given edge to the queue if it is not already present, considering both orientations of the edge.
func AppendUniqueEdge(queue [][2]Point, edge [2]Point) [][2]Point {
	eRev := [2]Point{edge[1], edge[0]}
	for _, qe := range queue {
		if qe == edge || qe == eRev {
			return queue
		}
	}
	return append(queue, edge)
}

// FindAdjacentTriangles finds the indices of two triangles sharing the specified edge in a given list of triangles.
// Returns -1 for an index if no triangle with the edge is found.
func FindAdjacentTriangles(triangles []Triangle, e [2]Point) (int, int) {
	idx1, idx2 := -1, -1
	for i, t := range triangles {
		if t.HasEdge(e) {
			if idx1 == -1 {
				idx1 = i
			} else {
				idx2 = i
				break
			}
		}
	}
	return idx1, idx2
}

// IsConvexQuadrilateral determines if four points form a strictly convex quadrilateral using orientation checks.
func IsConvexQuadrilateral(p1, p2, p3, p4 Point) bool {
	o1 := Orientation(p3, p4, p1)
	o2 := Orientation(p3, p4, p2)
	o3 := Orientation(p1, p2, p3)
	o4 := Orientation(p1, p2, p4)
	return o1 != o2 && o3 != o4 && o1 != 0 && o2 != 0 && o3 != 0 && o4 != 0
}

// LineIntersection calculates the intersection point of two lines defined by points (p1, q1) and (p2, q2).
// Returns NaN, NaN if the lines are parallel or overlapping.
func LineIntersection(p1, q1, p2, q2 Point) (float64, float64) {
	a1 := new(big.Float).Sub(BigFloat(q1.Y), BigFloat(p1.Y))
	b1 := new(big.Float).Sub(BigFloat(p1.X), BigFloat(q1.X))
	c1 := new(big.Float).Add(
		new(big.Float).Mul(a1, BigFloat(p1.X)),
		new(big.Float).Mul(b1, BigFloat(p1.Y)),
	)

	a2 := new(big.Float).Sub(BigFloat(q2.Y), BigFloat(p2.Y))
	b2 := new(big.Float).Sub(BigFloat(p2.X), BigFloat(q2.X))
	c2 := new(big.Float).Add(
		new(big.Float).Mul(a2, BigFloat(p2.X)),
		new(big.Float).Mul(b2, BigFloat(p2.Y)),
	)

	det := new(big.Float).Sub(
		new(big.Float).Mul(a1, b2),
		new(big.Float).Mul(a2, b1),
	)

	// Fallback in case of perfectly parallel segments (collinearity)
	if det.Sign() == 0 {
		return math.NaN(), math.NaN()
	}

	xNum := new(big.Float).Sub(
		new(big.Float).Mul(b2, c1),
		new(big.Float).Mul(b1, c2),
	)
	yNum := new(big.Float).Sub(
		new(big.Float).Mul(a1, c2),
		new(big.Float).Mul(a2, c1),
	)

	xRes, _ := new(big.Float).Quo(xNum, det).Float64()
	yRes, _ := new(big.Float).Quo(yNum, det).Float64()

	return xRes, yRes
}

// OnSegment checks if point q lies on the line segment defined by points p and r, assuming p, q, and r are collinear.
func OnSegment(p, q, r Point) bool {
	return q.X <= math.Max(p.X, r.X) && q.X >= math.Min(p.X, r.X) &&
		q.Y <= math.Max(p.Y, r.Y) && q.Y >= math.Min(p.Y, r.Y)
}

// OnSegmentStrict checks if point q lies strictly on the line segment formed by points p and r.
// This function ensures q is collinear with p and r and lies within the bounded box formed by p and r.
func OnSegmentStrict(p, q, r Point) bool {
	return Orientation(p, q, r) == 0 &&
		q.X >= math.Min(p.X, r.X) && q.X <= math.Max(p.X, r.X) &&
		q.Y >= math.Min(p.Y, r.Y) && q.Y <= math.Max(p.Y, r.Y)
}

// SegmentsCross determines if two line segments, defined by points (p1, q1) and (p2, q2), intersect each other strictly.
func SegmentsCross(p1, q1, p2, q2 Point) bool {
	o1 := Orientation(p1, q1, p2)
	o2 := Orientation(p1, q1, q2)
	o3 := Orientation(p2, q2, p1)
	o4 := Orientation(p2, q2, q1)
	return o1 != o2 && o3 != o4 && o1 != 0 && o2 != 0 && o3 != 0 && o4 != 0
}

// SegmentsIntersect determines if two line segments, defined by points p1-q1 and p2-q2, intersect in 2D space.
func SegmentsIntersect(p1, q1, p2, q2 Point) bool {
	o1 := Orientation(p1, q1, p2)
	o2 := Orientation(p1, q1, q2)
	o3 := Orientation(p2, q2, p1)
	o4 := Orientation(p2, q2, q1)

	if o1 != o2 && o3 != o4 {
		return true
	}
	if o1 == 0 && OnSegment(p1, p2, q1) {
		return true
	}
	if o2 == 0 && OnSegment(p1, q2, q1) {
		return true
	}
	if o3 == 0 && OnSegment(p2, p1, q2) {
		return true
	}
	if o4 == 0 && OnSegment(p2, q1, q2) {
		return true
	}

	return false
}

// Orientation determines the orientation of the triplet (p, q, r): 0 = collinear, 1 = clockwise, 2 = counterclockwise.
func Orientation(p, q, r Point) int {
	det := Orient2D(p, q, r)
	if det > 0 {
		return 2
	}
	if det < 0 {
		return 1
	}
	return 0
}

// Orient2D calculates the 2D orientation determinant for three points to determine their relative orientation in a plane.
func Orient2D(pa, pb, pc Point) float64 {
	detLeft := (pa.X - pc.X) * (pb.Y - pc.Y)
	detRight := (pa.Y - pc.Y) * (pb.X - pc.X)
	det := detLeft - detRight

	detSum := math.Abs(detLeft) + math.Abs(detRight)
	if math.Abs(det) >= errBoundOri*detSum {
		return det
	}
	return Orient2DExact(pa, pb, pc)
}

// Orient2DExact computes the determinant to exactly determine the orientation of three 2D points (pa, pb, pc).
func Orient2DExact(pa, pb, pc Point) float64 {
	ax, ay := BigFloat(pa.X), BigFloat(pa.Y)
	bx, by := BigFloat(pb.X), BigFloat(pb.Y)
	cx, cy := BigFloat(pc.X), BigFloat(pc.Y)

	acx := new(big.Float).Sub(ax, cx)
	bcy := new(big.Float).Sub(by, cy)
	acy := new(big.Float).Sub(ay, cy)
	bcx := new(big.Float).Sub(bx, cx)

	left := new(big.Float).Mul(acx, bcy)
	right := new(big.Float).Mul(acy, bcx)

	det := new(big.Float).Sub(left, right)

	// Prevents silent underflow of Float64() for infinitesimal non-zero determinants
	return float64(det.Sign())
}

// InCircle2D computes the determinant to determine if point `pd` lies inside the circumcircle of triangle formed by `pa`, `pb`, `pc`.
func InCircle2D(pa, pb, pc, pd Point) float64 {
	adx, ady := pa.X-pd.X, pa.Y-pd.Y
	bdx, bdy := pb.X-pd.X, pb.Y-pd.Y
	cdx, cdy := pc.X-pd.X, pc.Y-pd.Y

	abDet := adx*bdy - bdx*ady
	bcDet := bdx*cdy - cdx*bdy
	caDet := cdx*ady - adx*cdy

	aLift := adx*adx + ady*ady
	bLift := bdx*bdx + bdy*bdy
	cLift := cdx*cdx + cdy*cdy

	det := aLift*bcDet + bLift*caDet + cLift*abDet

	perman := (math.Abs(adx*bdy)+math.Abs(bdx*ady))*cLift +
		(math.Abs(bdx*cdy)+math.Abs(cdx*bdy))*aLift +
		(math.Abs(cdx*ady)+math.Abs(adx*cdy))*bLift

	if math.Abs(det) >= errBoundInc*perman {
		return det
	}
	return InCircle2DExact(pa, pb, pc, pd)
}

// InCircle2DExact computes the determinant to determine if a point lies inside, on, or outside the circumcircle of three other points.
// Uses exact arithmetic based on arbitrary-precision floating-point computations to ensure robustness.
// The input points pa, pb, pc, pd are expected in a 2D plane with their coordinates provided as float64 values.
// Returns a positive value if pd is inside, zero if on, and negative if outside the circumcircle of pa, pb, and pc.
func InCircle2DExact(pa, pb, pc, pd Point) float64 {
	ax, ay := BigFloat(pa.X), BigFloat(pa.Y)
	bx, by := BigFloat(pb.X), BigFloat(pb.Y)
	cx, cy := BigFloat(pc.X), BigFloat(pc.Y)
	dx, dy := BigFloat(pd.X), BigFloat(pd.Y)

	adx, ady := new(big.Float).Sub(ax, dx), new(big.Float).Sub(ay, dy)
	bdx, bdy := new(big.Float).Sub(bx, dx), new(big.Float).Sub(by, dy)
	cdx, cdy := new(big.Float).Sub(cx, dx), new(big.Float).Sub(cy, dy)

	abDet := new(big.Float).Sub(new(big.Float).Mul(adx, bdy), new(big.Float).Mul(bdx, ady))
	bcDet := new(big.Float).Sub(new(big.Float).Mul(bdx, cdy), new(big.Float).Mul(cdx, bdy))
	caDet := new(big.Float).Sub(new(big.Float).Mul(cdx, ady), new(big.Float).Mul(adx, cdy))

	aLift := new(big.Float).Add(new(big.Float).Mul(adx, adx), new(big.Float).Mul(ady, ady))
	bLift := new(big.Float).Add(new(big.Float).Mul(bdx, bdx), new(big.Float).Mul(bdy, bdy))
	cLift := new(big.Float).Add(new(big.Float).Mul(cdx, cdx), new(big.Float).Mul(cdy, cdy))

	term1 := new(big.Float).Mul(aLift, bcDet)
	term2 := new(big.Float).Mul(bLift, caDet)
	term3 := new(big.Float).Mul(cLift, abDet)

	det := new(big.Float).Add(new(big.Float).Add(term1, term2), term3)

	// Prevents silent underflow of Float64() for infinitesimal non-zero determinants
	return float64(det.Sign())
}
