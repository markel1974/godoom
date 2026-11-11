package geometry

// Triangle2D represents a geometric figure defined by three vertices A, B, and C represented as Point.
type Triangle2D struct {
	A, B, C XY
}

// HasVertex checks if a given point p is one of the vertices of the triangle t.
func (t Triangle2D) HasVertex(p XY) bool {
	return t.A == p || t.B == p || t.C == p
}

// HasEdge checks if the given edge, in either direction, exists in the triangle.
func (t Triangle2D) HasEdge(e [2]XY) bool {
	// Check both directions of the edge
	eRev := [2]XY{e[1], e[0]}
	tEdges := [3][2]XY{{t.A, t.B}, {t.B, t.C}, {t.C, t.A}}
	for _, te := range tEdges {
		if te == e || te == eRev {
			return true
		}
	}
	return false
}

// GetOppositeVertex returns the vertex of the triangle that is not part of the specified edge.
func (t Triangle2D) GetOppositeVertex(e [2]XY) XY {
	for _, p := range []XY{t.A, t.B, t.C} {
		if p != e[0] && p != e[1] {
			return p
		}
	}
	return XY{}
}
