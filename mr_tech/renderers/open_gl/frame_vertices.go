package open_gl

// FrameVertices represents a structure for managing vertex and index data for rendering, with support for dynamic growth.
type FrameVertices struct {
	vertices      []float32
	indices       []uint32
	verticesCount int32
	verticesSlot  uint32
	indicesCount  int32
	stride        int32
}

// NewFrameVertices initializes and returns a new FrameVertices instance with preallocated space for vertices and indices.
func NewFrameVertices(maxVerts int) *FrameVertices {
	return &FrameVertices{
		vertices:      make([]float32, 0, maxVerts*5),
		indices:       make([]uint32, 0, maxVerts*6),
		indicesCount:  0,
		verticesCount: 0,
		verticesSlot:  0,
		stride:        5,
	}
}

// GetIndicesLen returns the current number of indices in the frame vertex buffer.
func (w *FrameVertices) GetIndicesLen() int32 {
	return w.indicesCount
}

// Stride returns the number of elements in each vertex's attribute group within the vertex buffer.
func (w *FrameVertices) Stride() int32 {
	return w.stride
}

// Reset clears the FrameVertices by resetting vertex and index counts, and vertex slot to their initial state.
func (w *FrameVertices) Reset() {
	w.verticesCount = 0
	w.indicesCount = 0
	w.verticesSlot = 0
}

// AddVertex adds a vertex with specified position (x, y, z) and texture coordinates (u, v) to the buffer and returns its slot.
func (w *FrameVertices) AddVertex(x, y, z, u, v float32) uint32 {
	head := w.verticesCount
	if w.verticesCount += w.stride; w.verticesCount >= int32(len(w.vertices)) {
		w.growVertices()
	}
	w.vertices[head] = x
	w.vertices[head+1] = y
	w.vertices[head+2] = z
	w.vertices[head+3] = u
	w.vertices[head+4] = v

	slot := w.verticesSlot
	w.verticesSlot++
	return slot
}

// AddTriangle appends three vertex indices (i0, i1, i2) to define a triangle in the index buffer.
func (w *FrameVertices) AddTriangle(i0, i1, i2 uint32) {
	head := w.indicesCount
	if w.indicesCount += 3; w.indicesCount >= int32(len(w.indices)) {
		w.growIndices()
	}
	w.indices[head] = i0
	w.indices[head+1] = i1
	w.indices[head+2] = i2
}

// GetVertices returns the slice of vertices and indices currently stored in the frame's vertex buffer.
func (w *FrameVertices) GetVertices() ([]float32, int32, []uint32, int32) {
	return w.vertices, w.verticesCount, w.indices, w.indicesCount
}

// VerticesStride calculates the byte stride of vertex data in the buffer by multiplying the attribute group size by 4.
func (w *FrameVertices) VerticesStride() int32 {
	return w.stride * 4
}

// growIndices dynamically doubles the size of the indices slice or initializes it to 128 if empty to accommodate new indices.
func (w *FrameVertices) growIndices() {
	newSize := len(w.indices) * 2
	if newSize == 0 {
		newSize = 128
	}
	newData := make([]uint32, newSize)
	copy(newData, w.indices)
	w.indices = newData
}

// growVertices allocates a larger slice for storing vertex data when the current capacity is exceeded.
func (w *FrameVertices) growVertices() {
	newSize := len(w.vertices) * 2
	if newSize == 0 {
		newSize = 128
	}
	newData := make([]float32, newSize)
	copy(newData, w.vertices)
	w.vertices = newData
}
