package wad

// Package wad provides utilities for handling WAD-based geometry, specifically
// focusing on polygon manipulation, hole merging, and constrained Delaunay triangulation.
// It includes robust geometric predicates (Orient2D, InCircle) with exact arithmetic
// fallbacks to handle floating-point precision issues common in legacy map data.

import (
	"fmt"
	"math"
	"math/big"
	"sort"

	"github.com/markel1974/godoom/engine/model"
)

const (
	// Machine epsilon per float64 IEEE 754
	epsilon = 1.1102230246251565e-16
	// Bound di errore precalcolati per i determinanti
	errBoundOri = (3.0 + 16.0*epsilon) * epsilon
	errBoundInc = (10.0 + 96.0*epsilon) * epsilon
)

func BigFloat(f float64) *big.Float {
	return new(big.Float).SetPrec(256).SetFloat64(f)
}

type Point struct {
	X float64
	Y float64
}

func (p Point) ToModelXY() model.XY { return model.XY{X: p.X, Y: p.Y} }

type Triangle struct {
	A, B, C Point
}

func (t Triangle) HasVertex(p Point) bool {
	return t.A == p || t.B == p || t.C == p
}

func (t Triangle) HasEdge(e [2]Point) bool {
	// Check both directions of the edge
	eRev := [2]Point{e[1], e[0]}
	tEdges := [3][2]Point{{t.A, t.B}, {t.B, t.C}, {t.C, t.A}}
	for _, te := range tEdges {
		if te == e || te == eRev {
			return true
		}
	}
	return false
}

func (t Triangle) GetOppositeVertex(e [2]Point) Point {
	for _, p := range []Point{t.A, t.B, t.C} {
		if p != e[0] && p != e[1] {
			return p
		}
	}
	return Point{}
}

type Polygon []Point

func (poly Polygon) Triangulate(secIdx int) []Polygon {
	if len(poly) < 3 {
		return nil
	}

	// 1. Deserialization: extract Outer and Holes from flat array separated by NaN
	var outer Polygon
	var holes []Polygon
	var current Polygon

	for _, p := range poly {
		if math.IsNaN(p.X) || math.IsNaN(p.Y) {
			if outer == nil {
				outer = current
			} else {
				holes = append(holes, current)
			}
			current = nil
		} else {
			current = append(current, p)
		}
	}
	if outer == nil {
		outer = current
	} else if len(current) > 0 {
		holes = append(holes, current)
	}

	// 2. Collect all valid vertices
	var points Polygon
	points = append(points, outer...)
	for _, h := range holes {
		points = append(points, h...)
	}

	// PRE-PROCESSING TOPOLOGICO (Vertex Injection per T-Junctions e Intersezioni)
	var rawConstraints [][2]Point
	rawConstraints = append(rawConstraints, outer.buildConstraints()...)
	for _, hole := range holes {
		rawConstraints = append(rawConstraints, hole.buildConstraints()...)
	}

	sanitizedPoints, sanitizedConstraints := points.SanitizePSLG(rawConstraints)

	if len(sanitizedPoints) < 3 {
		return nil
	}

	// 3. Unconstrained Delaunay Triangulation (Bowyer-Watson)
	// Passiamo il superset di vertici che ora include i nodi iniettati
	mesh := sanitizedPoints.BowyerWatson()

	// 4. Constraint Recovery (Lawson FIFO deterministico)
	// Usiamo il set di vincoli frammentato per garantire adiacenze esatte
	mesh = RecoverConstraints(sanitizedConstraints, mesh, secIdx)

	// 5. Domain Culling
	var finalTriangles []Polygon
	for _, t := range mesh {
		// Calculate centroid for containment test
		centroid := Point{
			X: (t.A.X + t.B.X + t.C.X) / 3.0,
			Y: (t.A.Y + t.B.Y + t.C.Y) / 3.0,
		}

		// The triangle is valid if it's inside the perimeter and outside all holes
		if outer.PointInPolygon(centroid) {
			inHole := false
			for _, hole := range holes {
				if hole.PointInPolygon(centroid) {
					inHole = true
					break
				}
			}
			if !inHole {
				// Ensure counterclockwise (CCW) winding required by renderer
				if Orientation(t.A, t.B, t.C) == 2 {
					finalTriangles = append(finalTriangles, Polygon{t.A, t.C, t.B})
				} else {
					finalTriangles = append(finalTriangles, Polygon{t.A, t.B, t.C})
				}
			}
		}
	}

	return finalTriangles
}

func (poly Polygon) SanitizePSLG(constraints [][2]Point) (Polygon, [][2]Point) {
	var orderedPoints []Point
	seenPoints := make(map[Point]bool)

	// Inizializzazione deterministica dei vertici
	for _, p := range poly {
		if !seenPoints[p] {
			seenPoints[p] = true
			orderedPoints = append(orderedPoints, p)
		}
	}

	splitted := true
	for splitted {
		splitted = false
		var nextConstraints [][2]Point

		for i, c1 := range constraints {
			wasSplit := false

			// 1. Risoluzione T-Junctions: iterazione deterministica sullo slice
			for _, p := range orderedPoints {
				if p != c1[0] && p != c1[1] && OnSegmentStrict(c1[0], p, c1[1]) {
					nextConstraints = append(nextConstraints, [2]Point{c1[0], p}, [2]Point{p, c1[1]})
					wasSplit = true
					splitted = true
					break
				}
			}
			if wasSplit {
				continue
			}

			// 2. Risoluzione Intersezioni Edge-to-Edge
			for j := i + 1; j < len(constraints); j++ {
				c2 := constraints[j]
				if c1[0] == c2[0] || c1[0] == c2[1] || c1[1] == c2[0] || c1[1] == c2[1] {
					continue
				}

				if SegmentsCross(c1[0], c1[1], c2[0], c2[1]) {
					ix, iy := LineIntersection(c1[0], c1[1], c2[0], c2[1])

					if math.IsNaN(ix) || math.IsNaN(iy) {
						continue // Bypass della singolarità, nessuna iniezione
					}

					ip := Point{X: ix, Y: iy}

					if !seenPoints[ip] {
						seenPoints[ip] = true
						orderedPoints = append(orderedPoints, ip)
					}

					nextConstraints = append(nextConstraints, [2]Point{c1[0], ip}, [2]Point{ip, c1[1]})
					wasSplit = true
					splitted = true
					break
				}
			}

			if !wasSplit {
				nextConstraints = append(nextConstraints, c1)
			}
		}
		constraints = nextConstraints
	}

	return orderedPoints, constraints
}

func (poly Polygon) MaxPointsX() float64 {
	max := poly[0].X
	for _, p := range poly {
		if p.X > max {
			max = p.X
		}
	}
	return max
}

func (poly Polygon) buildConstraints() [][2]Point {
	var c [][2]Point
	if len(poly) < 3 {
		return c
	}
	for i := 0; i < len(poly); i++ {
		c = append(c, [2]Point{poly[i], poly[(i+1)%len(poly)]})
	}
	return c
}

func (poly Polygon) BowyerWatson() []Triangle {
	minX, minY := math.MaxFloat64, math.MaxFloat64
	maxX, maxY := -math.MaxFloat64, -math.MaxFloat64

	for _, p := range poly {
		if p.X < minX {
			minX = p.X
		}
		if p.X > maxX {
			maxX = p.X
		}
		if p.Y < minY {
			minY = p.Y
		}
		if p.Y > maxY {
			maxY = p.Y
		}
	}

	dx, dy := maxX-minX, maxY-minY
	deltaMax := math.Max(dx, dy)
	if deltaMax == 0 {
		deltaMax = 1.0
	}
	midX, midY := (minX+maxX)/2.0, (minY+maxY)/2.0

	// Bounding super-triangle that wraps the entire area
	stA := Point{X: midX - 20*deltaMax, Y: midY - deltaMax}
	stB := Point{X: midX, Y: midY + 20*deltaMax}
	stC := Point{X: midX + 20*deltaMax, Y: midY - deltaMax}

	triangles := []Triangle{{stA, stB, stC}}

	for _, p := range poly {
		var badTriangles []Triangle
		var polygon [][2]Point

		for _, t := range triangles {
			// Force CCW orientation for exact InCircle determinant
			if Orientation(t.A, t.B, t.C) == 2 {
				if InCircle2D(t.A, t.B, t.C, p) > 0 {
					badTriangles = append(badTriangles, t)
				}
			} else {
				if InCircle2D(t.A, t.C, t.B, p) > 0 {
					badTriangles = append(badTriangles, t)
				}
			}
		}

		for _, bt := range badTriangles {
			edges := [3][2]Point{{bt.A, bt.B}, {bt.B, bt.C}, {bt.C, bt.A}}
			for _, edge := range edges {
				shared := false
				for _, other := range badTriangles {
					if bt == other {
						continue
					}
					if other.HasEdge(edge) {
						shared = true
						break
					}
				}
				if !shared {
					polygon = append(polygon, edge)
				}
			}
		}

		var nextTriangles []Triangle
		for _, t := range triangles {
			isBad := false
			for _, bt := range badTriangles {
				if t == bt {
					isBad = true
					break
				}
			}
			if !isBad {
				nextTriangles = append(nextTriangles, t)
			}
		}
		triangles = nextTriangles

		for _, edge := range polygon {
			triangles = append(triangles, Triangle{edge[0], edge[1], p})
		}
	}

	return triangles
}

func (poly Polygon) PointInPolygon(p Point) bool {
	inside := false
	for i, j := 0, len(poly)-1; i < len(poly); j, i = i, i+1 {
		vi, vj := poly[i], poly[j]
		// Il raggio orizzontale interseca l'asse Y del segmento?
		if (vi.Y > p.Y) != (vj.Y > p.Y) {
			o := Orientation(vi, vj, p)
			// Se il segmento è ascendente, l'intersezione avviene se p è a sinistra (CCW).
			// Se discendente, l'intersezione avviene se p è a destra (CW).
			if vi.Y < vj.Y {
				if o == 2 {
					inside = !inside
				}
			} else {
				if o == 1 {
					inside = !inside
				}
			}
		}
	}
	return inside
}

func (poly Polygon) SignedArea() float64 {
	var area float64
	for i := 0; i < len(poly); i++ {
		p1, p2 := poly[i], poly[(i+1)%len(poly)]
		area += p1.X*p2.Y - p2.X*p1.Y
	}
	return area / 2.0
}

type ComplexPolygon struct {
	Outer Polygon
	Holes []Polygon
}

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

// RecoverConstraints deterministico. Richiede in input un PSLG puro.
// Elimina i loop infiniti processando le intersezioni tramite coda FIFO con failsafe per degenerazioni.
func RecoverConstraints(constraints [][2]Point, triangles []Triangle, secIdx int) []Triangle {
	for _, c := range constraints {
		var queue [][2]Point

		// 1. Accoda gli edge che intersecano il vincolo in modo stretto
		for _, t := range triangles {
			edges := [3][2]Point{{t.A, t.B}, {t.B, t.C}, {t.C, t.A}}
			for _, e := range edges {
				if SegmentsCross(e[0], e[1], c[0], c[1]) {
					queue = AppendUniqueEdge(queue, e)
				}
			}
		}

		// 2. Risoluzione topologica garantita con failsafe
		consecutiveFailures := 0

		for len(queue) > 0 {
			if consecutiveFailures >= len(queue) {
				// Rompiamo il ciclo per preservare l'esecuzione.
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

			// Solo le diagonali dei quadrilateri strettamente convessi possono essere flippate
			if IsConvexQuadrilateral(e[0], e[1], pOpp1, pOpp2) {
				consecutiveFailures = 0 // Reset in caso di successo
				triangles[t1Idx] = Triangle{pOpp1, pOpp2, e[0]}
				triangles[t2Idx] = Triangle{pOpp1, pOpp2, e[1]}

				newEdge := [2]Point{pOpp1, pOpp2}
				if SegmentsCross(newEdge[0], newEdge[1], c[0], c[1]) {
					queue = append(queue, newEdge)
				} else {
					// Valuta se i nuovi bordi del quadrilatero intersecano il vincolo
					for _, newBoundary := range [][2]Point{{pOpp1, e[0]}, {e[0], pOpp2}, {pOpp2, e[1]}, {e[1], pOpp1}} {
						if SegmentsCross(newBoundary[0], newBoundary[1], c[0], c[1]) {
							queue = AppendUniqueEdge(queue, newBoundary)
						}
					}
				}
			} else {
				// Il quadrilatero non è convesso. Lo reinseriamo in coda.
				consecutiveFailures++
				queue = append(queue, e)
			}
		}
	}
	return triangles
}

func DistanceSq(p1 Point, p2 Point) float64 {
	dx := p1.X - p2.X
	dy := p1.Y - p2.Y
	return dx*dx + dy*dy
}

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

func AppendUniqueEdge(queue [][2]Point, edge [2]Point) [][2]Point {
	eRev := [2]Point{edge[1], edge[0]}
	for _, qe := range queue {
		if qe == edge || qe == eRev {
			return queue
		}
	}
	return append(queue, edge)
}

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

func IsConvexQuadrilateral(p1, p2, p3, p4 Point) bool {
	o1 := Orientation(p3, p4, p1)
	o2 := Orientation(p3, p4, p2)
	o3 := Orientation(p1, p2, p3)
	o4 := Orientation(p1, p2, p4)
	return o1 != o2 && o3 != o4 && o1 != 0 && o2 != 0 && o3 != 0 && o4 != 0
}

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

	// Fallback in caso di segmenti perfettamente paralleli (collinearità)
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

func OnSegment(p, q, r Point) bool {
	return q.X <= math.Max(p.X, r.X) && q.X >= math.Min(p.X, r.X) &&
		q.Y <= math.Max(p.Y, r.Y) && q.Y >= math.Min(p.Y, r.Y)
}

func OnSegmentStrict(p, q, r Point) bool {
	return Orientation(p, q, r) == 0 &&
		q.X >= math.Min(p.X, r.X) && q.X <= math.Max(p.X, r.X) &&
		q.Y >= math.Min(p.Y, r.Y) && q.Y <= math.Max(p.Y, r.Y)
}

func SegmentsCross(p1, q1, p2, q2 Point) bool {
	o1 := Orientation(p1, q1, p2)
	o2 := Orientation(p1, q1, q2)
	o3 := Orientation(p2, q2, p1)
	o4 := Orientation(p2, q2, q1)
	return o1 != o2 && o3 != o4 && o1 != 0 && o2 != 0 && o3 != 0 && o4 != 0
}

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
	res, _ := det.Float64()
	return res
}

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
	res, _ := det.Float64()
	return res
}
