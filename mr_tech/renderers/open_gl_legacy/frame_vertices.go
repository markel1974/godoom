package open_gl_legacy

const vertexAlignment = 12

// FrameVertices represents a structure for storing 3D vertex data, including positions, UV coordinates, and light values.
type FrameVertices struct {
	vertices []float32
	len      int
}

// NewFrameVertices creates and returns a pointer to a new FrameVertices instance with an initial capacity for vertices.
func NewFrameVertices(s int) *FrameVertices {
	return &FrameVertices{
		vertices: make([]float32, s),
		len:      0,
	}
}

// Reset clears all vertices from the FrameVertices, preparing it for reuse without allocating new memory.
func (w *FrameVertices) Reset() {
	w.len = 0
}

// Len returns the number of vertices currently stored in the FrameVertices struct.
func (w *FrameVertices) Len() int {
	return w.len
}

// Alignment returns the alignment value of the vertices in the FrameVertices struct as an int32.
func (w *FrameVertices) Alignment() int32 {
	return vertexAlignment
}

// AddVertex appends a new vertex defined by position (x, y, z), texture coordinates (u, v), and lighting intensity.
// AddVertex appends a new vertex defined by position (x, y, z), texture coordinates (u, v), and lighting intensity.
func (w *FrameVertices) AddVertex(x, y, z, u, v, light, lcX, lcY, lcZ, nX, nY, nZ float32) {
	if w.len+vertexAlignment > len(w.vertices) {
		w.Grow()
	}

	idx := w.len
	w.vertices[idx] = x
	w.vertices[idx+1] = y
	w.vertices[idx+2] = z
	w.vertices[idx+3] = u
	w.vertices[idx+4] = v
	w.vertices[idx+5] = light
	w.vertices[idx+6] = lcX
	w.vertices[idx+7] = lcY
	w.vertices[idx+8] = lcZ
	w.vertices[idx+9] = nX
	w.vertices[idx+10] = nY
	w.vertices[idx+11] = nZ
	w.len += vertexAlignment
}

// Get returns the slice of float32 vertices stored in the FrameVertices instance.
func (w *FrameVertices) Get() []float32 {
	return w.vertices[:w.len]
}

// Grow increases the capacity of the vertices slice to accommodate additional vertex data.
func (w *FrameVertices) Grow() {
	newSize := len(w.vertices) * 2
	if newSize == 0 {
		newSize = vertexAlignment * 128
	}
	newVertices := make([]float32, newSize)
	copy(newVertices, w.vertices)
	w.vertices = newVertices
}
