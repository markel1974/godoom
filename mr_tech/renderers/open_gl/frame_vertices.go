package open_gl

import (
	"unsafe"

	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
)

// FrameVertices represents a structure for managing vertex and index buffers for rendering geometries in frames.
type FrameVertices struct {
	ctx                 api.IContext
	vertices            []float32
	indices             []uint32
	verticesPtr         unsafe.Pointer
	indicesPtr          unsafe.Pointer
	verticesCount       int32
	verticesSlot        uint32
	indicesCount        int32
	freezeIndicesCount  int32
	freezeVerticesCount int32
	freezeVerticesSlot  uint32
	stride              int32
}

// NewFrameVertices initializes a new FrameVertices instance with preallocated buffers for vertices and indices.
func NewFrameVertices(ctx api.IContext, maxVerts int) *FrameVertices {
	const maxStride = 19
	fv := &FrameVertices{
		ctx:      ctx,
		vertices: make([]float32, maxVerts),
		indices:  make([]uint32, maxVerts),
		stride:   maxStride,
	}
	fv.verticesPtr = ctx.Ptr(fv.vertices)
	fv.indicesPtr = ctx.Ptr(fv.indices)
	return fv
}

// Freeze locks the current state of verticesCount, indicesCount, and verticesSlot for subsequent resets.
func (w *FrameVertices) Freeze() {
	w.freezeVerticesCount = w.verticesCount
	w.freezeIndicesCount = w.indicesCount
	w.freezeVerticesSlot = w.verticesSlot
}

// DeepReset clears all frozen state and resets the FrameVertices instance completely, including frozen counts and slots.
func (w *FrameVertices) DeepReset() {
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

// AddVertex6 adds a vertex with position (x, y, z), texture coordinates (u, v), and texture layer (texLayer) to the buffer.
func (w *FrameVertices) AddVertex6(x, y, z, u, v, texLayer float32) uint32 {
	return w.AddVertex19(x, y, z, u, v, texLayer, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1.0, 0.0)
}

// AddVertex12 adds a vertex with position, texture coordinates, origin, billboard flag, specular boost, and shininess.
func (w *FrameVertices) AddVertex12(x, y, z, u, v, texLayer, ox, oy, oz, isBB float32, spec, shine float32) uint32 {
	return w.AddVertex19(x, y, z, u, v, texLayer, ox, oy, oz, isBB, 0, 0, 0, 0, 0, spec, shine, 1.0, 0.0)
}

// AddVertex14 adds a vertex with position, texture coordinates, origin, billboard flag, specular, and detail parameters.
func (w *FrameVertices) AddVertex14(x, y, z, u, v, texLayer, ox, oy, oz, isBB float32, spec, shine, detailScale, detailFactor float32) uint32 {
	return w.AddVertex19(x, y, z, u, v, texLayer, ox, oy, oz, isBB, 0, 0, 0, 0, 0, spec, shine, detailScale, detailFactor)
}

// AddVertex15 adds a vertex with 15 parameters.
func (w *FrameVertices) AddVertex15(x, y, z, u, v, texLayer, ox, oy, oz, isBB, nx, ny, nz, lerp, yaw float32) uint32 {
	return w.AddVertex19(x, y, z, u, v, texLayer, ox, oy, oz, isBB, nx, ny, nz, lerp, yaw, 0, 0, 1.0, 0.0)
}

// AddVertex17 adds a vertex to the buffer with position, texture coordinates, animation, and specular attributes.
func (w *FrameVertices) AddVertex17(x, y, z, u, v, texLayer, ox, oy, oz, isBB, nx, ny, nz, lerp, yaw, specBoost, shininess float32) uint32 {
	return w.AddVertex19(x, y, z, u, v, texLayer, ox, oy, oz, isBB, nx, ny, nz, lerp, yaw, specBoost, shininess, 1.0, 0.0)
}

// AddVertex19 adds a vertex to the buffer with 19 parameters, including detail scale and detail factor.
func (w *FrameVertices) AddVertex19(x, y, z, u, v, texLayer, ox, oy, oz, isBB, nx, ny, nz, lerp, yaw, specBoost, shininess, detailScale, detailFactor float32) uint32 {
	head := w.verticesCount
	w.verticesCount += w.stride
	if w.verticesCount >= int32(len(w.vertices)) {
		w.growVertices()
	}
	// Location 0: aPos
	w.vertices[head] = x
	w.vertices[head+1] = y
	w.vertices[head+2] = z
	// Location 1: aTexCoords
	w.vertices[head+3] = u
	w.vertices[head+4] = v
	w.vertices[head+5] = texLayer
	// Location 2: aOrigin
	w.vertices[head+6] = ox
	w.vertices[head+7] = oy
	w.vertices[head+8] = oz
	// Location 3: aIsBillboard
	w.vertices[head+9] = isBB
	// Location 4: aPosNext
	w.vertices[head+10] = nx
	w.vertices[head+11] = ny
	w.vertices[head+12] = nz
	// Location 5: aLerp
	w.vertices[head+13] = lerp
	// Location 6: aYaw
	w.vertices[head+14] = yaw
	// Location 7: specBoost
	w.vertices[head+15] = specBoost
	// Location 8: shininess
	w.vertices[head+16] = shininess
	// Location 9: aDetailParams
	w.vertices[head+17] = detailScale
	w.vertices[head+18] = detailFactor
	slot := w.verticesSlot
	w.verticesSlot++
	return slot
}

// AddTriangle adds a new triangle to the index buffer using the specified vertex indices i0, i1, and i2.
func (w *FrameVertices) AddTriangle(i0, i1, i2 uint32) {
	head := w.indicesCount
	w.indicesCount += 3
	if w.indicesCount > int32(len(w.indices)) {
		w.growIndices()
	}
	w.indices[head] = i0
	w.indices[head+1] = i1
	w.indices[head+2] = i2
}

// GetVertices retrieves the vertex buffer, vertex count, index buffer, and index count from the FrameVertices.
func (w *FrameVertices) GetVertices() (unsafe.Pointer, int, int32, unsafe.Pointer, int, int32) {
	return w.verticesPtr, int(w.verticesCount * 4), w.verticesCount, w.indicesPtr, int(w.indicesCount * 4), w.indicesCount
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
	w.indicesPtr = w.ctx.Ptr(w.indices)
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
	w.verticesPtr = w.ctx.Ptr(w.vertices)
}
