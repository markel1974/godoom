package open_gl

// FrameVertices represents a structure for storing 3D vertex data, including positions, UV coordinates, and light values.
type FrameVertices struct {
	vertices []float32
}

// NewFrameVertices creates and returns a pointer to a new FrameVertices instance with an initial capacity for vertices.
func NewFrameVertices(s int) *FrameVertices {
	return &FrameVertices{
		vertices: make([]float32, 0, s),
	}
}

// Reset clears all vertices from the FrameVertices, preparing it for reuse without allocating new memory.
func (w *FrameVertices) Reset() {
	w.vertices = w.vertices[:0]
}

// Len returns the number of vertices currently stored in the FrameVertices struct.
func (w *FrameVertices) Len() int {
	return len(w.vertices)
}

// Alignment returns the alignment value of the vertices in the FrameVertices struct as an int32.
func (w *FrameVertices) Alignment() int32 {
	const vertexAlignment = 6
	return vertexAlignment
}

// AddVertex appends a new vertex defined by position (x, y, z), texture coordinates (u, v), and lighting intensity.
func (w *FrameVertices) AddVertex(x, y, z, u, v, light float32) {
	//remember to modify vertexAlignment if you change de signature
	w.vertices = append(w.vertices, x, y, z, u, v, light)
}
