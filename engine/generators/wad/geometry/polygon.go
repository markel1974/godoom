package geometry

import "math"

// Polygon represents a slice of points defining a closed geometric shape in 2D space.
type Polygon []Point

// TraceLoops constructs closed polygon definitions (outers and holes) from a set of edges for a given Level.
// Handles self-intersecting topologies and shared vertices by enforcing maximum-angle left turns.
func (poly Polygon) TraceLoops(edges []Edge, count int) []ComplexPolygon {
	adj := make([][]Edge, len(poly))
	for _, e := range edges {
		adj[e.V1] = append(adj[e.V1], e)
	}

	// Bitmask for visited edges: (LDIdx << 1) | IsLeft
	visited := make([]bool, count*2)

	getVisitedIdx := func(e Edge) int {
		idx := e.LDIdx << 1
		if e.IsLeft {
			idx |= 1
		}
		return idx
	}

	var rawLoops []Polygon

	for _, startEdge := range edges {
		vIdx := getVisitedIdx(startEdge)
		if visited[vIdx] {
			continue
		}

		var currentLoop Polygon
		curr := startEdge

		for {
			visited[getVisitedIdx(curr)] = true
			v := poly[curr.V1]
			currentLoop = append(currentLoop, Point{X: v.X, Y: v.Y})

			nextOptions := adj[curr.V2]
			var nextEdge Edge
			found := false

			if len(nextOptions) == 1 {
				if !visited[getVisitedIdx(nextOptions[0])] {
					nextEdge = nextOptions[0]
					found = true
				}
			} else if len(nextOptions) > 1 {
				// Multiple outgoing edges: Calculate angles to perform the tightest possible turn.
				// We need the incoming vector to compute the relative deviation.
				inV1 := poly[curr.V1]
				inV2 := poly[curr.V2]
				inDx := inV2.X - inV1.X
				inDy := inV2.Y - inV1.Y
				inAngle := math.Atan2(inDy, inDx)

				minAngleDiff := math.MaxFloat64
				bestIdx := -1

				for i, o := range nextOptions {
					if visited[getVisitedIdx(o)] {
						continue
					}
					outV1 := poly[o.V1]
					outV2 := poly[o.V2]
					outDx := outV2.X - outV1.X
					outDy := outV2.Y - outV1.Y
					outAngle := math.Atan2(outDy, outDx)

					// Calculation of relative angular deviation (standard CCW orientation)
					// The incoming angle must be inverted (as if looking backward from the vertex)
					diff := outAngle - (inAngle + math.Pi)
					for diff < 0 {
						diff += 2 * math.Pi
					}
					for diff >= 2*math.Pi {
						diff -= 2 * math.Pi
					}

					// We look for the smallest angle (tightest right turn)
					// to close the local envelope consistently
					if diff < minAngleDiff {
						minAngleDiff = diff
						bestIdx = i
					}
				}

				if bestIdx != -1 {
					nextEdge = nextOptions[bestIdx]
					found = true
				}
			}

			if !found || nextEdge.V1 == startEdge.V1 {
				break
			}
			curr = nextEdge
		}

		if len(currentLoop) >= 3 {
			rawLoops = append(rawLoops, currentLoop)
		}
	}

	if len(rawLoops) == 0 {
		return nil
	}

	var outers []Polygon
	var holes []Polygon

	maxArea := 0.0
	outerSign := 1.0

	areas := make([]float64, len(rawLoops))
	for i, loop := range rawLoops {
		areas[i] = loop.SignedArea()
		absArea := math.Abs(areas[i])
		if absArea > maxArea {
			maxArea = absArea
			if areas[i] < 0 {
				outerSign = -1.0
			} else {
				outerSign = 1.0
			}
		}
	}

	for i, loop := range rawLoops {
		if (areas[i] < 0 && outerSign < 0) || (areas[i] > 0 && outerSign > 0) {
			outers = append(outers, loop)
		} else {
			holes = append(holes, loop)
		}
	}

	defs := make([]ComplexPolygon, len(outers))
	for i, o := range outers {
		defs[i] = ComplexPolygon{Outer: o}
	}

	for _, h := range holes {
		for i, def := range defs {
			if def.Outer.PointInPolygon(h[0]) {
				defs[i].Holes = append(defs[i].Holes, h)
				break
			}
		}
	}

	return defs
}

// Triangulate decomposes a polygon with possible holes into a set of non-overlapping triangles using PSLG processing.
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

	// TOPOLOGICAL PRE-PROCESSING (Vertex Injection for T-Junctions and Intersections)
	var rawConstraints [][2]Point
	rawConstraints = append(rawConstraints, outer.BuildConstraints()...)
	for _, hole := range holes {
		rawConstraints = append(rawConstraints, hole.BuildConstraints()...)
	}

	sanitizedPoints, sanitizedConstraints := points.SanitizePSLG(rawConstraints)

	if len(sanitizedPoints) < 3 {
		return nil
	}

	// 3. Unconstrained Delaunay Triangulation (Bowyer-Watson)
	// We pass the superset of vertices which now includes the injected nodes
	mesh := sanitizedPoints.BowyerWatson()

	// 4. Constraint Recovery (deterministic FIFO Lawson)
	// We use the fragmented constraint set to guarantee exact adjacencies
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

// SanitizePSLG ensures a planar straight-line graph (PSLG) is valid by resolving T-junctions and edge-to-edge intersections.
func (poly Polygon) SanitizePSLG(constraints [][2]Point) (Polygon, [][2]Point) {
	var orderedPoints []Point
	seenPoints := make(map[Point]bool)

	// Deterministic vertex initialization
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

			// 1. T-Junction Resolution: deterministic iteration over the slice
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

			// 2. Edge-to-Edge Intersection Resolution
			for j := i + 1; j < len(constraints); j++ {
				c2 := constraints[j]
				if c1[0] == c2[0] || c1[0] == c2[1] || c1[1] == c2[0] || c1[1] == c2[1] {
					continue
				}

				if SegmentsCross(c1[0], c1[1], c2[0], c2[1]) {
					ix, iy := LineIntersection(c1[0], c1[1], c2[0], c2[1])

					if math.IsNaN(ix) || math.IsNaN(iy) {
						continue // Singularity bypass, no injection
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

// MaxPointsX returns the maximum X-coordinate among all points in the polygon.
func (poly Polygon) MaxPointsX() float64 {
	max := poly[0].X
	for _, p := range poly {
		if p.X > max {
			max = p.X
		}
	}
	return max
}

// BuildConstraints generates a list of line segments representing the edges of the polygon in a cyclic order.
func (poly Polygon) BuildConstraints() [][2]Point {
	var c [][2]Point
	if len(poly) < 3 {
		return c
	}
	for i := 0; i < len(poly); i++ {
		c = append(c, [2]Point{poly[i], poly[(i+1)%len(poly)]})
	}
	return c
}

// BowyerWatson performs unconstrained Delaunay triangulation of the polygon's vertices using the Bowyer-Watson algorithm.
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

// PointInPolygon determines if a given point lies inside the polygon using the ray-casting algorithm.
func (poly Polygon) PointInPolygon(p Point) bool {
	inside := false
	for i, j := 0, len(poly)-1; i < len(poly); j, i = i, i+1 {
		vi, vj := poly[i], poly[j]
		// Does the horizontal ray intersect the Y-axis of the segment?
		if (vi.Y > p.Y) != (vj.Y > p.Y) {
			o := Orientation(vi, vj, p)
			// If the segment is ascending, intersection occurs if p is to the left (CCW).
			// If descending, intersection occurs if p is to the right (CW).
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

// SignedArea calculates the signed area of the polygon. Positive value indicates counterclockwise orientation.
func (poly Polygon) SignedArea() float64 {
	var area float64
	for i := 0; i < len(poly); i++ {
		p1, p2 := poly[i], poly[(i+1)%len(poly)]
		area += p1.X*p2.Y - p2.X*p1.Y
	}
	return area / 2.0
}
