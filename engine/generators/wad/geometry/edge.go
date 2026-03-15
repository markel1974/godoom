package geometry

// Edge represents a connection between two vertices in a graph or geometric structure.
// V1 and V2 are the indices of the vertices connected by the edge.
// LDIdx is an index associated with a line definition in the context of the level structure.
// IsLeft indicates whether the edge corresponds to the left side of a line definition.
type Edge struct {
	V1, V2 uint16
	LDIdx  int
	IsLeft bool
}

// EdgeKey represents a unique key for an edge defined by its start and end points in 2D space.
type EdgeKey struct {
	X1, Y1, X2, Y2 float64
}
