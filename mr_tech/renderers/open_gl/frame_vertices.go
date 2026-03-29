package open_gl

// FrameVertices represents a structure for storing and managing vertex and index data with a defined vertex stride.
type FrameVertices struct {
	vertices []float32
	indices  []uint32
	stride   int32
}

// NewFrameVertices initializes and returns a pointer to a new FrameVertices with space for maxVerts vertices and associated indices.
func NewFrameVertices(maxVerts int) *FrameVertices {
	return &FrameVertices{
		vertices: make([]float32, 0, maxVerts*5),
		indices:  make([]uint32, 0, maxVerts*6),
		stride:   5, // Solo X, Y, Z, U, V
	}
}

func (w *FrameVertices) Stride() int32 {
	return w.stride
}

// Reset clears the vertices and indices of the FrameVertices, resetting them to an empty state.
func (w *FrameVertices) Reset() {
	w.vertices = w.vertices[:0]
	w.indices = w.indices[:0]
}

// AddVertex appends a vertex with given position (x, y, z) and texture coordinates (u, v) to the vertex buffer.
// Returns the index of the added vertex in the vertex buffer.
func (w *FrameVertices) AddVertex(x, y, z, u, v float32) uint32 {
	idx := uint32(len(w.vertices) / int(w.stride))
	w.vertices = append(w.vertices, x, y, z, u, v)
	return idx
}

// AddTriangle appends three vertex indices (i0, i1, i2) to the indices slice, forming a new triangle.
func (w *FrameVertices) AddTriangle(i0, i1, i2 uint32) {
	w.indices = append(w.indices, i0, i1, i2)
}

// Get retrieves the vertex and index buffer data as slices of float32 and uint32, respectively.
func (w *FrameVertices) Get() ([]float32, []uint32) {
	return w.vertices, w.indices
}
