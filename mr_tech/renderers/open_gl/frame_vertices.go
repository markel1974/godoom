package open_gl

import (
	"unsafe"

	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/shaders"
	"github.com/markel1974/godoom/mr_tech/textures"
)

const materialsMax = 8

var _emptyMaterial = make([]*textures.Material, materialsMax)

// FrameBatch represents a single drawing batch associated with a blend mode and material.
type FrameBatch struct {
	blendMode  int
	indexCount int32
	material   *textures.Material
	centroidX  float32
	centroidY  float32
	centroidZ  float32
	centCount  int32
}

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
	containers          []*FrameVertices
	containerIdx        int
	batches             []*FrameBatch
	batchesLen          int
	materials           []*textures.Material
	animMats            []*textures.Material
	animTexs            []*textures.Texture
	animLen             int
}

func newFrameBatches(n int) []*FrameBatch {
	b := make([]*FrameBatch, n)
	for i := range b {
		b[i] = &FrameBatch{}
	}
	return b
}

// NewFrameVertices initializes a new FrameVertices instance with preallocated buffers for vertices and indices.
func NewFrameVertices(ctx api.IContext, maxVerts int) *FrameVertices {
	const maxStride = shaders.StreamingVertexFloats
	fv := &FrameVertices{
		ctx:          ctx,
		vertices:     make([]float32, maxVerts),
		indices:      make([]uint32, maxVerts),
		stride:       maxStride,
		containers:   make([]*FrameVertices, 16),
		containerIdx: 0,
		batches:      newFrameBatches(8),
		materials:    make([]*textures.Material, materialsMax),
	}
	if ctx != nil {
		fv.verticesPtr = ctx.Ptr(fv.vertices)
		fv.indicesPtr = ctx.Ptr(fv.indices)
	}
	return fv
}

// NewStagingFrameVertices initializes a CPU-only staging FrameVertices without registering pointers with the GPU context.
func NewStagingFrameVertices(maxVerts int) *FrameVertices {
	const maxStride = shaders.StreamingVertexFloats
	return &FrameVertices{
		vertices:  make([]float32, maxVerts),
		indices:   make([]uint32, maxVerts),
		stride:    maxStride,
		batches:   newFrameBatches(4),
		materials: make([]*textures.Material, materialsMax),
	}
}

// CreateMarker increments the container index and returns the updated value as an integer.
func (w *FrameVertices) CreateMarker() int {
	w.containerIdx++
	return w.containerIdx
}

// Stage ensures the container at the specified marker index exists, growing if needed, and returns it.
func (w *FrameVertices) Stage(marker int) *FrameVertices {
	for marker >= len(w.containers) {
		w.growContainers()
	}
	if w.containers[marker] == nil {
		w.containers[marker] = NewStagingFrameVertices(128)
	}
	return w.containers[marker]
}

// AddBatch appends or merges a draw batch of indices into the FrameVertices.
func (w *FrameVertices) AddBatch(blendMode int, count int32, mat *textures.Material) {
	w.AddBatchCentroid(blendMode, count, mat, 0, 0, 0)
}

// AddBatchCentroid appends or merges a draw batch while tracking an approximate world centroid for transparent sorting.
func (w *FrameVertices) AddBatchCentroid(blendMode int, count int32, mat *textures.Material, cx, cy, cz float32) {
	if count <= 0 {
		return
	}
	if w.batchesLen > 0 {
		b := w.batches[w.batchesLen-1]
		if b.blendMode == blendMode && b.material == mat {
			b.indexCount += count
			b.centroidX += cx
			b.centroidY += cy
			b.centroidZ += cz
			b.centCount++
			return
		}
	}
	if w.batchesLen >= len(w.batches) {
		oldLen := len(w.batches)
		newLen := max(oldLen*2, 4)
		grown := make([]*FrameBatch, newLen)
		copy(grown, w.batches)
		for i := oldLen; i < newLen; i++ {
			grown[i] = &FrameBatch{}
		}
		w.batches = grown
	}
	b := w.batches[w.batchesLen]
	b.blendMode = blendMode
	b.indexCount = count
	b.material = mat
	b.centroidX = cx
	b.centroidY = cy
	b.centroidZ = cz
	b.centCount = 1
	w.batchesLen++
}

// SetMaterial assigns a material to the specified index in the materials slice of the FrameVertices object.
func (w *FrameVertices) SetMaterial(index int, mat *textures.Material) {
	w.materials[index] = mat
}

// HasMaterial checks if there is a material assigned at the specified index in the materials slice.
func (w *FrameVertices) HasMaterial(index int) bool {
	return w.materials[index] != nil
}

// GetMaterial retrieves the material at the specified index from the materials slice in the FrameVertices instance.
func (w *FrameVertices) GetMaterial(index int) *textures.Material {
	return w.materials[index]
}

// Commit fast-copies the staged vertices, re-indexes the indices to the current vertex slot,
// and replays the draw commands into the appropriate blend modes.
func (w *FrameVertices) Commit(fv *FrameVertices, modes []*DrawCommands) {
	vCount := fv.verticesCount
	iCount := fv.indicesCount
	if vCount == 0 || iCount == 0 {
		return
	}

	for w.verticesCount+vCount >= int32(len(w.vertices)) {
		w.growVertices()
	}

	copy(w.vertices[w.verticesCount:], fv.vertices[:vCount])

	for w.indicesCount+iCount > int32(len(w.indices)) {
		w.growIndices()
	}

	baseSlot := w.verticesSlot
	dstIndices := w.indices[w.indicesCount : w.indicesCount+iCount]
	srcIndices := fv.indices[:iCount]
	for i := int32(0); i < iCount; i++ {
		dstIndices[i] = srcIndices[i] + baseSlot
	}

	startIdx := w.indicesCount
	for b := 0; b < fv.batchesLen; b++ {
		batch := fv.batches[b]
		endIdx := startIdx + batch.indexCount
		targetDc := modes[batch.blendMode]
		cx, cy, cz := batch.centroidX, batch.centroidY, batch.centroidZ
		if batch.centCount > 1 {
			inv := 1.0 / float32(batch.centCount)
			cx *= inv
			cy *= inv
			cz *= inv
		}
		targetDc.ComputeWithCentroid(startIdx, endIdx, batch.material, cx, cy, cz)
		startIdx = endIdx
	}

	w.verticesCount += vCount
	w.verticesSlot += fv.verticesSlot
	w.indicesCount += iCount
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
	for i := 0; i < w.batchesLen; i++ {
		w.batches[i].material = nil
	}
	w.batchesLen = 0
	for i := 0; i < w.animLen; i++ {
		w.animMats[i] = nil
		w.animTexs[i] = nil
	}
	w.animLen = 0
	copy(w.materials, _emptyMaterial)
	w.Reset()
}

// AddAnimRef records an animated material and the active texture frame used when building vertices.
func (w *FrameVertices) AddAnimRef(mat *textures.Material, tex *textures.Texture) {
	for i := 0; i < w.animLen; i++ {
		if w.animMats[i] == mat {
			return
		}
	}
	if w.animLen >= len(w.animMats) {
		newLen := max(len(w.animMats)*2, 4)
		newMats := make([]*textures.Material, newLen)
		copy(newMats, w.animMats)
		w.animMats = newMats
		newTexs := make([]*textures.Texture, newLen)
		copy(newTexs, w.animTexs)
		w.animTexs = newTexs
	}
	w.animMats[w.animLen] = mat
	w.animTexs[w.animLen] = tex
	w.animLen++
}

// NeedsAnimRebuild reports whether any tracked animated material has advanced to a different texture frame.
func (w *FrameVertices) NeedsAnimRebuild() bool {
	for i := 0; i < w.animLen; i++ {
		if w.animMats[i].GetCurrentFrame() != w.animTexs[i] {
			return true
		}
	}
	return false
}

// FrozenBytes returns the byte sizes of the frozen static prefix in the vertex and index buffers.
func (w *FrameVertices) FrozenBytes() (int, int) {
	return int(w.freezeVerticesCount * 4), int(w.freezeIndicesCount * 4)
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
	return w.AddVertex20(x, y, z, u, v, texLayer, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1.0, 0.0)
}

// AddVertex12 adds a vertex with position, texture coordinates, origin, billboard flag, specular boost, and shininess.
func (w *FrameVertices) AddVertex12(x, y, z, u, v, texLayer, ox, oy, oz, isBB float32, material, spec, shine float32) uint32 {
	return w.AddVertex20(x, y, z, u, v, texLayer, ox, oy, oz, isBB, 0, 0, 0, 0, 0, material, spec, shine, 1.0, 0.0)
}

// AddVertex14 adds a vertex with position, texture coordinates, origin, billboard flag, specular, and detail parameters.
func (w *FrameVertices) AddVertex14(x, y, z, u, v, texLayer, ox, oy, oz, isBB float32, material, spec, shine, detailScale, detailFactor float32) uint32 {
	return w.AddVertex20(x, y, z, u, v, texLayer, ox, oy, oz, isBB, 0, 0, 0, 0, 0, material, spec, shine, detailScale, detailFactor)
}

// AddVertex15 adds a vertex with 15 parameters.
func (w *FrameVertices) AddVertex15(x, y, z, u, v, texLayer, ox, oy, oz, isBB, nx, ny, nz, lerp, yaw float32) uint32 {
	return w.AddVertex20(x, y, z, u, v, texLayer, ox, oy, oz, isBB, nx, ny, nz, lerp, yaw, 0, 0, 0, 1.0, 0.0)
}

// AddVertex17 adds a vertex to the buffer with position, texture coordinates, animation, and specular attributes.
func (w *FrameVertices) AddVertex17(x, y, z, u, v, texLayer, ox, oy, oz, isBB, nx, ny, nz, lerp, yaw, material, specBoost, shininess float32) uint32 {
	return w.AddVertex20(x, y, z, u, v, texLayer, ox, oy, oz, isBB, nx, ny, nz, lerp, yaw, material, specBoost, shininess, 1.0, 0.0)
}

// AddVertex20 adds a streaming vertex using default flat normals and no lightmap/env reflection.
func (w *FrameVertices) AddVertex20(x, y, z, u, v, texLayer, ox, oy, oz, isBB, nx, ny, nz, lerp, yaw, material, specBoost, shininess, detailScale, detailFactor float32) uint32 {
	return w.AddVertexFull(
		x, y, z, u, v, texLayer,
		0, 1, 0, 1, 0, 0, 1,
		0, 0, -1,
		ox, oy, oz, isBB,
		nx, ny, nz, lerp, yaw,
		material, specBoost, shininess, detailScale, detailFactor, 0, 16777215,
	)
}

// AddVertexFull appends a complete 32-float streaming vertex with explicit normal, tangent, lightmap UV, and env reflection data.
func (w *FrameVertices) AddVertexFull(
	x, y, z, u, v, texLayer,
	nx, ny, nz, tx, ty, tz, tSign,
	lmU, lmV, lmLayer,
	ox, oy, oz, renderMode,
	pnx, pny, pnz, lerp, yaw,
	material, specBoost, shininess, detailScale, detailFactor, envStrength, envTint float32,
) uint32 {
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
	// Location 2: aNormal
	w.vertices[head+6] = nx
	w.vertices[head+7] = ny
	w.vertices[head+8] = nz
	// Location 3: aTangent
	w.vertices[head+9] = tx
	w.vertices[head+10] = ty
	w.vertices[head+11] = tz
	w.vertices[head+12] = tSign
	// Location 4: aLightmap
	w.vertices[head+13] = lmU
	w.vertices[head+14] = lmV
	w.vertices[head+15] = lmLayer
	// Location 5: aMaterial
	w.vertices[head+16] = material
	w.vertices[head+17] = specBoost
	w.vertices[head+18] = shininess
	w.vertices[head+19] = detailScale
	// Location 6: aExtra
	w.vertices[head+20] = detailFactor
	w.vertices[head+21] = envStrength
	w.vertices[head+22] = renderMode
	w.vertices[head+23] = envTint
	// Location 7: aOrigin
	w.vertices[head+24] = ox
	w.vertices[head+25] = oy
	w.vertices[head+26] = oz
	// Location 8: aPosNext
	w.vertices[head+27] = pnx
	w.vertices[head+28] = pny
	w.vertices[head+29] = pnz
	// Location 9: aAnimData
	w.vertices[head+30] = lerp
	w.vertices[head+31] = yaw

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
	if w.ctx != nil {
		w.indicesPtr = w.ctx.Ptr(w.indices)
	}
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
	if w.ctx != nil {
		w.verticesPtr = w.ctx.Ptr(w.vertices)
	}
}

func (w *FrameVertices) growContainers() {
	newSize := len(w.containers) * 2
	if newSize == 0 {
		newSize = 128
	}
	newData := make([]*FrameVertices, newSize)
	copy(newData, w.containers)
	w.containers = newData
}
