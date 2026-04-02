package geometry

import "math"

// Edge represents a connection between two vertices with additional metadata such as linedef index and orientation.
type Edge struct {
	V1Idx  int
	V2Idx  int
	LdIdx  int
	IsLeft bool
}

// GetVisitedIdx computes a unique bitmask for the edge based on its linedef index and orientation.
func (e Edge) GetVisitedIdx() int {
	idx := e.LdIdx << 1
	if e.IsLeft {
		idx |= 1
	}
	return idx
}

// EdgeKey uniquely identifies a 2D undirected edge using normalized coordinates independent of edge orientation.
type EdgeKey struct {
	X1, Y1 float64
	X2, Y2 float64
}

// QuantizedEdgeKey represents a unique spatial key for an edge, with coordinates quantized to fixed precision.
type QuantizedEdgeKey struct {
	X1, Y1, X2, Y2 int64
}

// NewQuantizedEdgeKey generates a QuantizedEdgeKey by quantizing the given coordinates using the specified quantization factor.
func NewQuantizedEdgeKey(x1, y1, x2, y2, q float64) QuantizedEdgeKey {
	return QuantizedEdgeKey{
		X1: quantize(x1, q),
		Y1: quantize(y1, q),
		X2: quantize(x2, q),
		Y2: quantize(y2, q),
	}
}

// quantize maps a floating-point value `v` to a quantized integer value based on the quantization factor `q`.
func quantize(v float64, q float64) int64 {
	return int64(math.Round(v * q))
}
