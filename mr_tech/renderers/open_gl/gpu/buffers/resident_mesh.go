package buffers

import (
	"cmp"
	"slices"
	"unsafe"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/model/volumes"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// ResidentMesh stores GPU-resident geometry, render grouping, and rebuild state for a mesh.
type ResidentMesh struct {
	ctx                  api.IContext
	VAO                  uint32
	VAOInstanced         uint32
	VBO                  uint32
	EBO                  uint32
	boundInstVBO         uint32
	lastInstByteOffset   int32
	SubMeshes            []*ResidentSubMesh
	SubMeshesLen         int
	HasBlend             [config.BlendModeLatest]bool
	HasSolidOpaque       bool
	HasCutoutOpaque      bool
	HasSolidPrepass      bool
	HasCutoutPrepass     bool
	SkyMaterial          *textures.Material
	SrcFaces             *volumes.Faces
	built                bool
	Version              uint64
	GeomHash             uint64
	FaceCount            int
	LastSeenFrame        uint64
	RebuiltFrame         uint64
	InstGroupFrame       uint64
	InstGroupIdx         int
	ShadowInstGroupFrame uint64
	ShadowInstGroupIdx   int
	VertData             []uint32
	vertDataPtr          unsafe.Pointer
	vertWordsLen         int
	vertCount            int
	Indices              []uint32
	indicesPtr           unsafe.Pointer
	indicesLen           int
	vertHashSlots        []uint32
	vertHashGen          []uint32
	dedupGen             uint32
	AnimMats             []*textures.Material
	AnimFrames           []*textures.Texture
	AnimLen              int
	faceOrder            []int
	faceKeys             []*faceStateKey
}

// NewResidentMesh returns a ResidentMesh bound to ctx with its instance byte offset initialized.
func NewResidentMesh(ctx api.IContext) *ResidentMesh {
	return &ResidentMesh{
		ctx:                ctx,
		lastInstByteOffset: -1,
	}
}

// acquireSubMesh returns the next reusable submesh, growing the submesh pool when needed.
func (rm *ResidentMesh) acquireSubMesh() *ResidentSubMesh {
	if rm.SubMeshesLen >= len(rm.SubMeshes) {
		oldLen := len(rm.SubMeshes)
		newLen := max(oldLen*2, 4)
		grown := make([]*ResidentSubMesh, newLen)
		copy(grown, rm.SubMeshes)
		for i := oldLen; i < newLen; i++ {
			grown[i] = &ResidentSubMesh{}
		}
		rm.SubMeshes = grown
	}
	sub := rm.SubMeshes[rm.SubMeshesLen]
	rm.SubMeshesLen++
	return sub
}

// addAnimRef records an animated material and its current texture frame if the material is not already tracked.
func (rm *ResidentMesh) addAnimRef(mat *textures.Material, tex *textures.Texture) {
	for i := 0; i < rm.AnimLen; i++ {
		if rm.AnimMats[i] == mat {
			return
		}
	}
	if rm.AnimLen >= len(rm.AnimMats) {
		newLen := max(len(rm.AnimMats)*2, 4)
		newMats := make([]*textures.Material, newLen)
		copy(newMats, rm.AnimMats)
		rm.AnimMats = newMats
		newFrames := make([]*textures.Texture, newLen)
		copy(newFrames, rm.AnimFrames)
		rm.AnimFrames = newFrames
	}
	rm.AnimMats[rm.AnimLen] = mat
	rm.AnimFrames[rm.AnimLen] = tex
	rm.AnimLen++
}

// NeedsAnimRebuild reports whether any tracked animated material has advanced to a different frame.
func (rm *ResidentMesh) NeedsAnimRebuild() bool {
	for i := 0; i < rm.AnimLen; i++ {
		if rm.AnimMats[i].GetCurrentFrame() != rm.AnimFrames[i] {
			return true
		}
	}
	return false
}

// NeedsRebuild reports whether the mesh is unbuilt, out of sync with faces, or has stale animation frames.
func (rm *ResidentMesh) NeedsRebuild(faces *volumes.Faces, checkAnim bool) bool {
	if !rm.built || faces == nil {
		return true
	}
	if faces.GetVersion() != rm.Version || faces.Len() != rm.FaceCount {
		return true
	}
	if checkAnim && rm.NeedsAnimRebuild() {
		return true
	}
	return false
}

// VertexData returns the active packed 32-bit vertex words for the current build.
func (rm *ResidentMesh) VertexData() []uint32 {
	return rm.VertData[:rm.vertWordsLen]
}

// VertexCount returns the number of unique deduplicated vertices in the current build.
func (rm *ResidentMesh) VertexCount() int {
	return rm.vertCount
}

// IndexData returns the active triangle index slice for the current build.
func (rm *ResidentMesh) IndexData() []uint32 {
	return rm.Indices[:rm.indicesLen]
}

// IndexCount returns the number of triangle indices in the current build.
func (rm *ResidentMesh) IndexCount() int {
	return rm.indicesLen
}

// resetBuildState clears transient mesh build flags and length counters before rebuilding geometry.
func (rm *ResidentMesh) resetBuildState() {
	for i := range rm.HasBlend {
		rm.HasBlend[i] = false
	}
	rm.HasSolidOpaque = false
	rm.HasCutoutOpaque = false
	rm.HasSolidPrepass = false
	rm.HasCutoutPrepass = false
	rm.SkyMaterial = nil
	rm.SubMeshesLen = 0
	rm.AnimLen = 0
	rm.vertWordsLen = 0
	rm.vertCount = 0
	rm.indicesLen = 0
	rm.dedupGen++
	if rm.dedupGen == 0 {
		clear(rm.vertHashGen)
		rm.dedupGen = 1
	}
}

// Rebuild rebuilds the resident mesh from the provided faces and uploads valid geometry to the GPU.
// Rebuild returns true when at least one submesh is produced.
func (rm *ResidentMesh) Rebuild(faces *volumes.Faces, texLookup func(*textures.Texture) (float32, bool)) bool {
	rm.SrcFaces = faces
	prevAnimLen := rm.AnimLen
	rm.resetBuildState()

	if faces == nil || faces.Len() == 0 {
		for i := range prevAnimLen {
			rm.AnimMats[i] = nil
			rm.AnimFrames[i] = nil
		}
		if faces != nil {
			rm.Version = faces.GetVersion()
			rm.GeomHash = faces.GetGeomHash()
			rm.FaceCount = 0
		}
		rm.built = true
		return false
	}

	_, _, skyMat, _, ok := rm.appendFacesRange(faces, texLookup, false)
	for i := rm.AnimLen; i < prevAnimLen; i++ {
		rm.AnimMats[i] = nil
		rm.AnimFrames[i] = nil
	}
	rm.SkyMaterial = skyMat
	rm.Version = faces.GetVersion()
	rm.GeomHash = faces.GetGeomHash()
	rm.FaceCount = faces.Len()
	if !ok {
		rm.built = true
		return false
	}

	rm.uploadToGPU()
	rm.built = true
	return rm.SubMeshesLen > 0
}

func nextPow2(v int) int {
	if v <= 1 {
		return 1
	}
	v--
	v |= v >> 1
	v |= v >> 2
	v |= v >> 4
	v |= v >> 8
	v |= v >> 16
	return v + 1
}

func hashVertWords(v *[StaticVertexWords]uint32) uint32 {
	h := uint32(2166136261)
	for i := range v {
		h ^= v[i]
		h *= 16777619
	}
	h ^= h >> 16
	h *= 0x85ebca6b
	h ^= h >> 13
	h *= 0xc2b2ae35
	h ^= h >> 16
	return h
}

func (rm *ResidentMesh) ensureVertHashCapacity(maxVerts int) {
	needSlots := nextPow2(max(maxVerts*2, 64))
	if len(rm.vertHashSlots) >= needSlots {
		if rm.dedupGen == 0 {
			rm.dedupGen = 1
		}
		return
	}
	newSlots := make([]uint32, needSlots)
	newGens := make([]uint32, needSlots)
	rm.vertHashSlots = newSlots
	rm.vertHashGen = newGens
	rm.dedupGen = 1
	if rm.vertCount == 0 {
		return
	}
	mask := uint32(needSlots - 1)
	for idx := 0; idx < rm.vertCount; idx++ {
		base := idx * StaticVertexWords
		key := (*[StaticVertexWords]uint32)(rm.VertData[base : base+StaticVertexWords])
		pos := hashVertWords(key) & mask
		for newGens[pos] == 1 {
			pos = (pos + 1) & mask
		}
		newGens[pos] = 1
		newSlots[pos] = uint32(idx)
	}
}

func (rm *ResidentMesh) ensureVertexStorage(reqWords, reqIndices, reqVerts int) {
	if len(rm.VertData) < reqWords {
		newLen := max(len(rm.VertData)*2, reqWords)
		grown := make([]uint32, newLen)
		copy(grown, rm.VertData[:rm.vertWordsLen])
		rm.VertData = grown
		if rm.ctx != nil {
			rm.vertDataPtr = rm.ctx.Ptr(rm.VertData)
		}
	}
	if len(rm.Indices) < reqIndices {
		newLen := max(len(rm.Indices)*2, reqIndices)
		grown := make([]uint32, newLen)
		copy(grown, rm.Indices[:rm.indicesLen])
		rm.Indices = grown
		if rm.ctx != nil {
			rm.indicesPtr = rm.ctx.Ptr(rm.Indices)
		}
	}
	rm.ensureVertHashCapacity(reqVerts)
}

func (rm *ResidentMesh) dedupAppendVertex(cand *[StaticVertexWords]uint32) uint32 {
	slots := rm.vertHashSlots
	gens := rm.vertHashGen
	mask := uint32(len(slots) - 1)
	pos := hashVertWords(cand) & mask
	gen := rm.dedupGen
	for gens[pos] == gen {
		existingIdx := slots[pos]
		base := int(existingIdx) * StaticVertexWords
		if *(*[StaticVertexWords]uint32)(rm.VertData[base : base+StaticVertexWords]) == *cand {
			return existingIdx
		}
		pos = (pos + 1) & mask
	}
	idx := uint32(rm.vertCount)
	off := rm.vertWordsLen
	copy(rm.VertData[off:off+StaticVertexWords], cand[:])
	rm.vertWordsLen = off + StaticVertexWords
	rm.vertCount++
	gens[pos] = gen
	slots[pos] = idx
	return idx
}

// ReserveStaticAtlas ensures mesh vertex, index, and deduplication storage can hold totalFaces static triangles.
// ReserveStaticAtlas is a no-op when totalFaces is zero or negative.
func (rm *ResidentMesh) ReserveStaticAtlas(totalFaces int) {
	if totalFaces <= 0 {
		return
	}
	neededVerts := totalFaces * 3
	rm.ensureVertexStorage(neededVerts*StaticVertexWords, neededVerts, neededVerts)
}

// BeginStaticAtlas resets the mesh for static atlas construction and clears source face tracking.
// It marks the mesh unbuilt until the atlas is finalized.
func (rm *ResidentMesh) BeginStaticAtlas() {
	rm.SrcFaces = nil
	rm.resetBuildState()
	rm.built = false
}

// AppendStaticFaces appends a non-empty face set to the current static atlas build.
// It returns the added submesh range, any sky material, whether animation was found, and success.
func (rm *ResidentMesh) AppendStaticFaces(faces *volumes.Faces, texLookup func(*textures.Texture) (float32, bool)) (subStart int, subCount int, skyMat *textures.Material, hasAnim bool, ok bool) {
	if faces == nil || faces.Len() == 0 {
		return 0, 0, nil, false, false
	}
	return rm.appendFacesRange(faces, texLookup, true)
}

// EndStaticAtlas uploads the accumulated static atlas mesh data to the GPU and marks the mesh as built.
func (rm *ResidentMesh) EndStaticAtlas() {
	rm.uploadToGPU()
	rm.built = true
}

// appendFacesRange appends renderable faces to the resident mesh, grouping them into submeshes by render state.
// It skips unsupported faces, records sky material, and reports animated rejection when requested.
func (rm *ResidentMesh) appendFacesRange(faces *volumes.Faces, texLookup func(*textures.Texture) (float32, bool), rejectAnimated bool) (subStart int, subCount int, skyMat *textures.Material, hasAnim bool, ok bool) {
	nFaces := faces.Len()
	if len(rm.faceOrder) < nFaces {
		rm.faceOrder = make([]int, max(len(rm.faceOrder)*2, nFaces))
	}
	if len(rm.faceKeys) < nFaces {
		oldLen := len(rm.faceKeys)
		newLen := max(oldLen*2, nFaces)
		grown := make([]*faceStateKey, newLen)
		copy(grown, rm.faceKeys)
		for i := oldLen; i < newLen; i++ {
			grown[i] = &faceStateKey{}
		}
		rm.faceKeys = grown
	}

	validCount := 0
	for i := range nFaces {
		face := faces.At(i)
		if face == nil {
			continue
		}
		mat := face.GetMaterial()
		if mat == nil || mat.GetKind() == int(config.MaterialKindNone) {
			continue
		}
		if mat.GetKind() == int(config.MaterialKindSky) {
			skyMat = mat
			continue
		}
		tex := mat.GetCurrentFrame()
		if tex == nil {
			continue
		}
		if _, texOK := texLookup(tex); !texOK {
			continue
		}
		if mat.IsAnimated() {
			if rejectAnimated {
				return 0, 0, nil, true, false
			}
			rm.addAnimRef(mat, tex)
		}
		bm := mat.GetBlendMode()
		if bm < 0 || bm >= config.BlendModeLatest {
			bm = config.BlendModeOpaque
		}
		ptr := rm.faceKeys[i]
		ptr.blendMode = bm
		ptr.cullMode = mat.GetCullMode()
		ptr.hasCutout = mat.HasCutout()
		ptr.depthWrite = mat.GetDepthWrite()
		ptr.polygonOffset = mat.GetPolygonOffset()
		ptr.sortOrder = mat.GetSort()

		rm.faceOrder[validCount] = i
		validCount++
	}

	if skyMat != nil {
		rm.SkyMaterial = skyMat
	}
	if validCount == 0 {
		return 0, 0, skyMat, false, false
	}

	slices.SortStableFunc(rm.faceOrder[:validCount], func(a, b int) int {
		ka := rm.faceKeys[a]
		kb := rm.faceKeys[b]
		if ka.blendMode != kb.blendMode {
			return cmp.Compare(ka.blendMode, kb.blendMode)
		}
		if ka.hasCutout != kb.hasCutout {
			if !ka.hasCutout {
				return -1
			}
			return 1
		}
		if ka.depthWrite != kb.depthWrite {
			if ka.depthWrite {
				return -1
			}
			return 1
		}
		if ka.polygonOffset != kb.polygonOffset {
			if !ka.polygonOffset {
				return -1
			}
			return 1
		}
		if ka.cullMode != kb.cullMode {
			return cmp.Compare(ka.cullMode, kb.cullMode)
		}
		return cmp.Compare(ka.sortOrder, kb.sortOrder)
	})

	maxAddedVerts := validCount * 3
	rm.ensureVertexStorage(rm.vertWordsLen+maxAddedVerts*StaticVertexWords, rm.indicesLen+maxAddedVerts, rm.vertCount+maxAddedVerts)

	subStart = rm.SubMeshesLen
	var curSub *ResidentSubMesh
	var curKey faceStateKey
	var centSumX, centSumY, centSumZ float64
	var centCount int

	iOut := rm.indicesLen
	var c0, c1, c2 [StaticVertexWords]uint32
	for k := 0; k < validCount; k++ {
		fi := rm.faceOrder[k]
		face := faces.At(fi)
		mat := face.GetMaterial()
		fk := rm.faceKeys[fi]
		texLayer, _ := texLookup(mat.GetCurrentFrame())

		if curSub == nil || *fk != curKey {
			curSub.FinalizeCentroid(centSumX, centSumY, centSumZ, centCount)
			curKey = *fk
			centSumX, centSumY, centSumZ = 0, 0, 0
			centCount = 0
			curSub = rm.acquireSubMesh()
			curSub.BlendMode = fk.blendMode
			curSub.CullMode = fk.cullMode
			curSub.HasCutout = fk.hasCutout
			curSub.DepthWrite = fk.depthWrite
			curSub.PolygonOffset = fk.polygonOffset
			curSub.SortOrder = fk.sortOrder
			curSub.CentroidX = 0
			curSub.CentroidY = 0
			curSub.CentroidZ = 0
			curSub.FirstIndex = int32(iOut)
			curSub.IndexCount = 0
			if fk.blendMode >= 0 && fk.blendMode < config.BlendModeLatest {
				rm.HasBlend[fk.blendMode] = true
			}
			if fk.blendMode == config.BlendModeOpaque {
				if fk.hasCutout {
					rm.HasCutoutOpaque = true
					if fk.depthWrite && !fk.polygonOffset {
						rm.HasCutoutPrepass = true
					}
				} else {
					rm.HasSolidOpaque = true
					if fk.depthWrite && !fk.polygonOffset {
						rm.HasSolidPrepass = true
					}
				}
			}
		}

		renderMode := float32(0.0)
		if config.MaterialKind(mat.GetKind()).ToMaterialFamily() == config.MaterialFamilyLiquid {
			renderMode = 0.5
		}
		packed, spec, shininess, dScale, dFactor, envStr, envTint := mat.GetRenderParams()

		p0x, p0y, p0z, p1x, p1y, p1z, p2x, p2y, p2z := face.GetPointsFlat()
		u0, u1, u2, v0, v1, v2 := face.GetUVFlat()
		nx0, ny0, nz0, nx1, ny1, nz1, nx2, ny2, nz2 := face.GetVertexNormalsFlat()
		tx0, ty0, tz0, s0, tx1, ty1, tz1, s1, tx2, ty2, tz2, s2 := face.GetTangentsFlat()
		lmID, lmU, lmV := face.GetLightmap()
		lmIDFloat := float32(lmID)

		// GL coordinates centroid
		centSumX += float64(p0x+p1x+p2x) / 3.0
		centSumY += float64(p0z+p1z+p2z) / 3.0
		centSumZ += -float64(p0y+p1y+p2y) / 3.0
		centCount++

		WriteStaticVertex(c0[:], p0x, p0y, p0z, u0, v0, texLayer, nx0, ny0, nz0, tx0, ty0, tz0, s0, lmU[0], lmV[0], lmIDFloat, packed, spec, shininess, dScale, dFactor, envStr, renderMode, envTint)
		WriteStaticVertex(c1[:], p1x, p1y, p1z, u1, v1, texLayer, nx1, ny1, nz1, tx1, ty1, tz1, s1, lmU[1], lmV[1], lmIDFloat, packed, spec, shininess, dScale, dFactor, envStr, renderMode, envTint)
		WriteStaticVertex(c2[:], p2x, p2y, p2z, u2, v2, texLayer, nx2, ny2, nz2, tx2, ty2, tz2, s2, lmU[2], lmV[2], lmIDFloat, packed, spec, shininess, dScale, dFactor, envStr, renderMode, envTint)

		rm.Indices[iOut] = rm.dedupAppendVertex(&c0)
		rm.Indices[iOut+1] = rm.dedupAppendVertex(&c1)
		rm.Indices[iOut+2] = rm.dedupAppendVertex(&c2)
		iOut += 3
		curSub.IndexCount += 3
	}
	curSub.FinalizeCentroid(centSumX, centSumY, centSumZ, centCount)
	rm.indicesLen = iOut
	return subStart, rm.SubMeshesLen - subStart, skyMat, false, true
}

// uploadToGPU uploads mesh vertex and index data to GPU buffers and configures vertex attributes.
// It creates GPU objects on first use and invalidates cached instanced VAO binding state.
func (rm *ResidentMesh) uploadToGPU() {
	if rm.VAO == 0 {
		rm.ctx.GenVertexArrays(1, &rm.VAO)
		rm.ctx.GenBuffers(1, &rm.VBO)
		rm.ctx.GenBuffers(1, &rm.EBO)
	}
	rm.ctx.BindVertexArray(rm.VAO)
	rm.ctx.BindBuffer(api.ARRAY_BUFFER, rm.VBO)
	if rm.vertWordsLen > 0 {
		rm.ctx.BufferData(api.ARRAY_BUFFER, rm.vertWordsLen*4, rm.vertDataPtr, api.STATIC_DRAW)
	}
	rm.ctx.BindBuffer(api.ELEMENT_ARRAY_BUFFER, rm.EBO)
	if rm.indicesLen > 0 {
		rm.ctx.BufferData(api.ELEMENT_ARRAY_BUFFER, rm.indicesLen*4, rm.indicesPtr, api.STATIC_DRAW)
	}
	ConfigureVertexAttribs(rm.ctx, StaticVertexBytes, false)
	rm.ctx.BindVertexArray(0)
	// Invalidate instanced VAO bindings so next instanced bind refreshes cleanly.
	rm.lastInstByteOffset = -1
}

// BindInstanced binds the mesh for instanced rendering using instanceVBO matrix data at baseByteOffset.
func (rm *ResidentMesh) BindInstanced(instanceVBO uint32, baseByteOffset int32) {
	rm.BindInstancedWithCache(instanceVBO, baseByteOffset, 0)
}

// BindInstancedWithCache binds the instanced VAO, updating instance buffer attributes only when cached state changes.
// It returns the VAO used for instanced rendering.
func (rm *ResidentMesh) BindInstancedWithCache(instanceVBO uint32, baseByteOffset int32, boundVAO uint32) uint32 {
	if rm.VAOInstanced == 0 {
		rm.ctx.GenVertexArrays(1, &rm.VAOInstanced)
		rm.ctx.BindVertexArray(rm.VAOInstanced)
		rm.ctx.BindBuffer(api.ARRAY_BUFFER, rm.VBO)
		rm.ctx.BindBuffer(api.ELEMENT_ARRAY_BUFFER, rm.EBO)
		ConfigureVertexAttribs(rm.ctx, StaticVertexBytes, false)
		rm.boundInstVBO = 0
		rm.lastInstByteOffset = -1
	} else if boundVAO != rm.VAOInstanced {
		rm.ctx.BindVertexArray(rm.VAOInstanced)
	}

	if rm.boundInstVBO != instanceVBO || rm.lastInstByteOffset != baseByteOffset {
		rm.ctx.BindBuffer(api.ARRAY_BUFFER, instanceVBO)
		const matStride = InstanceMatrixFloats * 4
		for col := range uint32(4) {
			loc := 10 + col
			off := int(baseByteOffset) + int(col*16)
			rm.ctx.VertexAttribPointer(loc, 4, api.FLOAT, false, matStride, rm.ctx.PtrOffset(off))
			if rm.boundInstVBO == 0 {
				rm.ctx.EnableVertexAttribArray(loc)
				rm.ctx.VertexAttribDivisor(loc, 1)
			}
		}
		rm.boundInstVBO = instanceVBO
		rm.lastInstByteOffset = baseByteOffset
	}
	return rm.VAOInstanced
}

// Dispose releases GPU vertex array and buffer resources owned by the resident mesh.
func (rm *ResidentMesh) Dispose() {
	if rm.VAO != 0 {
		rm.ctx.DeleteVertexArrays(1, &rm.VAO)
		rm.VAO = 0
	}
	if rm.VAOInstanced != 0 {
		rm.ctx.DeleteVertexArrays(1, &rm.VAOInstanced)
		rm.VAOInstanced = 0
	}
	if rm.VBO != 0 {
		rm.ctx.DeleteBuffers(1, &rm.VBO)
		rm.VBO = 0
	}
	if rm.EBO != 0 {
		rm.ctx.DeleteBuffers(1, &rm.EBO)
		rm.EBO = 0
	}
}
