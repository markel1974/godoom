package open_gl

// FrameVertices represents a structure for managing vertex and index buffers for rendering geometries in frames.
type FrameVertices struct {
	vertices            []float32
	indices             []uint32
	verticesCount       int32
	verticesSlot        uint32
	indicesCount        int32
	freezeIndicesCount  int32
	freezeVerticesCount int32
	freezeVerticesSlot  uint32
	stride              int32
}

// NewFrameVertices initializes a new FrameVertices instance with preallocated buffers for vertices and indices.
func NewFrameVertices(maxVerts int) *FrameVertices {
	return &FrameVertices{
		vertices: make([]float32, maxVerts*5),
		indices:  make([]uint32, maxVerts*6),
		stride:   5,
	}
}

// Freeze locks the current state of verticesCount, indicesCount, and verticesSlot for subsequent resets.
func (w *FrameVertices) Freeze() {
	w.freezeVerticesCount = w.verticesCount
	w.freezeIndicesCount = w.indicesCount
	w.freezeVerticesSlot = w.verticesSlot
}

// DeepReset clears all frozen state and resets the FrameVertices instance completely, including frozen counts and slots.
func (w *FrameVertices) DeepReset() {
	// FIX: Corretto il copia-incolla
	w.freezeVerticesCount = 0
	w.freezeIndicesCount = 0
	w.freezeVerticesSlot = 0
	w.Reset()
}

// Reset restores the vertices count, indices count, and vertices slot to their previously frozen states.
func (w *FrameVertices) Reset() {
	w.verticesCount = w.freezeVerticesCount
	w.indicesCount = w.freezeIndicesCount
	w.verticesSlot = w.freezeVerticesSlot
}

// GetIndicesLen returns the current count of indices in the FrameVertices structure.
func (w *FrameVertices) GetIndicesLen() int32 {
	return w.indicesCount
}

// VerticesStride returns the byte stride of the vertex data, calculated as the stride value multiplied by 4.
func (w *FrameVertices) VerticesStride() int32 {
	return w.stride * 4
}

// AddVertex adds a vertex to the FrameVertices at the specified coordinates and texture mapping values.
// It returns the index of the added vertex.
func (w *FrameVertices) AddVertex(x, y, z, u, v float32) uint32 {
	head := w.verticesCount
	w.verticesCount += w.stride
	if w.verticesCount > int32(len(w.vertices)) {
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

// AddTriangle adds a new triangle to the index buffer using the specified vertex indices i0, i1, and i2.
func (w *FrameVertices) AddTriangle(i0, i1, i2 uint32) {
	head := w.indicesCount
	w.indicesCount += 3
	// FIX: Controllo di overflow corretto
	if w.indicesCount > int32(len(w.indices)) {
		w.growIndices()
	}
	w.indices[head] = i0
	w.indices[head+1] = i1
	w.indices[head+2] = i2
}

// GetVertices retrieves the vertex buffer, vertex count, index buffer, and index count from the FrameVertices.
func (w *FrameVertices) GetVertices() ([]float32, int32, []uint32, int32) {
	return w.vertices, w.verticesCount, w.indices, w.indicesCount
}

// growIndices dynamically increases the capacity of the indices slice, doubling its size or initializing it to 128 if empty.
func (w *FrameVertices) growIndices() {
	newSize := len(w.indices) * 2
	if newSize == 0 {
		newSize = 128
	}
	newData := make([]uint32, newSize)
	copy(newData, w.indices)
	w.indices = newData
}

// growVertices doubles the capacity of the vertices slice or initializes it with a default size of 128 if empty.
func (w *FrameVertices) growVertices() {
	newSize := len(w.vertices) * 2
	if newSize == 0 {
		newSize = 128
	}
	newData := make([]float32, newSize)
	copy(newData, w.vertices)
	w.vertices = newData
}
