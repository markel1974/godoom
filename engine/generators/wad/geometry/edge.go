package geometry

import "math"

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

type QuantizedEdgeKey struct {
	X1, Y1, X2, Y2 int64
}

func NewQuantizedEdgeKey(x1, y1, x2, y2, q float64) QuantizedEdgeKey {
	return QuantizedEdgeKey{
		X1: quantize(x1, q),
		Y1: quantize(y1, q),
		X2: quantize(x2, q),
		Y2: quantize(y2, q),
	}
}

func quantize(v float64, q float64) int64 {
	return int64(math.Round(v * q))
}
