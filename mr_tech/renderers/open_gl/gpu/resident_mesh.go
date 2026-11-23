package gpu

import (
	"cmp"
	"slices"
	"unsafe"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/model/volumes"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
	"github.com/markel1974/godoom/mr_tech/textures"
)

const (
	// StaticVertexFloats is the float32 stride for GPU-resident static/instanced meshes (24 floats = 96 bytes).
	StaticVertexFloats = 24
	// StreamingVertexFloats is the float32 stride for CPU-streamed morphing/billboard/UI meshes (32 floats = 128 bytes).
	StreamingVertexFloats = 32
	// InstanceMatrixFloats is the float32 stride per rigid instance mat4 transform (16 floats = 64 bytes).
	InstanceMatrixFloats = 16
)

const (
	// InstanceModeStatic selects direct resident static geometry (locations 0..6, u_instanceMode = 0).
	InstanceModeStatic = int32(0)
	// InstanceModeInstanced selects GPU instanced geometry (locations 0..6 + mat4 at 10..13, u_instanceMode = 1).
	InstanceModeInstanced = int32(1)
	// InstanceModeStreaming selects dynamic CPU-streamed morphing/billboard/UI geometry (locations 0..9, u_instanceMode = 2).
	InstanceModeStreaming = int32(2)
)

const (
	CullStateUnset = -1
	CullStateOff   = 0
	CullStateBack  = 1
	CullStateFront = 2
)

// ApplyMaterialCullState updates GL CULL_FACE and CullFace only when the desired state changes.
// Returns the updated current state token.
func ApplyMaterialCullState(ctx api.IContext, curState int, cullMode int, hasCutout bool, forceDisable bool) int {
	want := CullStateOff
	if !forceDisable && !hasCutout {
		switch cullMode {
		case config.CullFront:
			want = CullStateBack
		case config.CullBack:
			want = CullStateFront
		case config.CullNone:
			want = CullStateOff
		default:
			want = CullStateBack
		}
	}
	if want == curState {
		return curState
	}
	switch want {
	case CullStateOff:
		ctx.Disable(api.CULL_FACE)
	case CullStateBack:
		if curState <= CullStateOff {
			ctx.Enable(api.CULL_FACE)
		}
		ctx.CullFace(api.BACK)
	case CullStateFront:
		if curState <= CullStateOff {
			ctx.Enable(api.CULL_FACE)
		}
		ctx.CullFace(api.FRONT)
	}
	return want
}

// ConfigureVertexAttribs configures vertex attribute pointers 0..6 (and optionally 7..9 for streaming) on the active VAO.
func ConfigureVertexAttribs(ctx api.IContext, strideBytes int32, includeDynamic bool) {
	// Location 0: aPos (vec3)
	ctx.VertexAttribPointer(0, 3, api.FLOAT, false, strideBytes, ctx.PtrOffset(0))
	ctx.EnableVertexAttribArray(0)
	// Location 1: aTexCoords (u, v_gl, texLayer) (vec3)
	ctx.VertexAttribPointer(1, 3, api.FLOAT, false, strideBytes, ctx.PtrOffset(3*4))
	ctx.EnableVertexAttribArray(1)
	// Location 2: aNormal (vec3 GL space)
	ctx.VertexAttribPointer(2, 3, api.FLOAT, false, strideBytes, ctx.PtrOffset(6*4))
	ctx.EnableVertexAttribArray(2)
	// Location 3: aTangent (tx, ty, tz, sign) (vec4 GL space)
	ctx.VertexAttribPointer(3, 4, api.FLOAT, false, strideBytes, ctx.PtrOffset(9*4))
	ctx.EnableVertexAttribArray(3)
	// Location 4: aLightmap (lmU, lmV, lmLayer) (vec3)
	ctx.VertexAttribPointer(4, 3, api.FLOAT, false, strideBytes, ctx.PtrOffset(13*4))
	ctx.EnableVertexAttribArray(4)
	// Location 5: aMaterial (packed, specBoost, shininess, detailScale) (vec4)
	ctx.VertexAttribPointer(5, 4, api.FLOAT, false, strideBytes, ctx.PtrOffset(16*4))
	ctx.EnableVertexAttribArray(5)
	// Location 6: aExtra (detailFactor, envStrength, renderMode, envTintPacked) (vec4)
	ctx.VertexAttribPointer(6, 4, api.FLOAT, false, strideBytes, ctx.PtrOffset(20*4))
	ctx.EnableVertexAttribArray(6)

	if includeDynamic {
		// Location 7: aOrigin (vec3)
		ctx.VertexAttribPointer(7, 3, api.FLOAT, false, strideBytes, ctx.PtrOffset(24*4))
		ctx.EnableVertexAttribArray(7)
		// Location 8: aPosNext (vec3)
		ctx.VertexAttribPointer(8, 3, api.FLOAT, false, strideBytes, ctx.PtrOffset(27*4))
		ctx.EnableVertexAttribArray(8)
		// Location 9: aAnimData (lerp, yaw) (vec2)
		ctx.VertexAttribPointer(9, 2, api.FLOAT, false, strideBytes, ctx.PtrOffset(30*4))
		ctx.EnableVertexAttribArray(9)
	}
}

func WriteStaticVertex(dst []float32, px, py, pz, u, v float64, texLayer float32, nx, ny, nz, tx, ty, tz, tSign, lmu, lmv float64, lmIDFloat, packed, spec, shininess, dScale, dFactor, envStr, renderMode, envTint float32) {
	dst[0] = float32(px)
	dst[1] = float32(pz)
	dst[2] = float32(-py)
	dst[3] = float32(u)
	dst[4] = float32(-v)
	dst[5] = texLayer
	dst[6] = float32(nx)
	dst[7] = float32(nz)
	dst[8] = float32(-ny)
	dst[9] = float32(tx)
	dst[10] = float32(tz)
	dst[11] = float32(-ty)
	dst[12] = float32(tSign)
	dst[13] = float32(lmu)
	dst[14] = float32(lmv)
	dst[15] = lmIDFloat
	dst[16] = packed
	dst[17] = spec
	dst[18] = shininess
	dst[19] = dScale
	dst[20] = dFactor
	dst[21] = envStr
	dst[22] = renderMode
	dst[23] = envTint
}

type faceStateKey struct {
	blendMode     int
	cullMode      int
	hasCutout     bool
	depthWrite    bool
	polygonOffset bool
	sortOrder     int
}

// ResidentSubMesh describes a contiguous index slice inside a ResidentMesh grouped by render state.
type ResidentSubMesh struct {
	BlendMode     int
	CullMode      int
	HasCutout     bool
	DepthWrite    bool
	PolygonOffset bool
	SortOrder     int
	CentroidX     float32
	CentroidY     float32
	CentroidZ     float32
	FirstIndex    int32
	IndexCount    int32
}

func (sub *ResidentSubMesh) FinalizeCentroid(sumX, sumY, sumZ float64, count int) {
	if sub != nil && count > 0 {
		inv := 1.0 / float64(count)
		sub.CentroidX = float32(sumX * inv)
		sub.CentroidY = float32(sumY * inv)
		sub.CentroidZ = float32(sumZ * inv)
	}
}

// ResidentMesh stores GPU-resident static VBO/EBO/VAO buffers and grouped submesh slices
// shared by static world volumes and instanced rigid 3D things.
type ResidentMesh struct {
	ctx                api.IContext
	VAO                uint32
	VAOInstanced       uint32
	VBO                uint32
	EBO                uint32
	boundInstVBO       uint32
	lastInstByteOffset int32
	SubMeshes          []*ResidentSubMesh
	SubMeshesLen       int
	HasBlend           [config.BlendModeLatest]bool
	HasSolidOpaque     bool
	HasCutoutOpaque    bool
	HasSolidPrepass    bool
	HasCutoutPrepass   bool
	SkyMaterial        *textures.Material
	SrcFaces           *volumes.Faces
	built              bool
	Version            uint64
	GeomHash           uint64
	FaceCount          int
	LastSeenFrame      uint64
	RebuiltFrame       uint64
	InstGroupFrame     uint64
	InstGroupIdx       int
	VertFloats         []float32
	vertFloatsPtr      unsafe.Pointer
	vertFloatsLen      int
	Indices            []uint32
	indicesPtr         unsafe.Pointer
	indicesLen         int
	AnimMats           []*textures.Material
	AnimFrames         []*textures.Texture
	AnimLen            int
	faceOrder          []int
	faceKeys           []*faceStateKey
}

// NewResidentMesh creates an unpopulated ResidentMesh bound to ctx.
func NewResidentMesh(ctx api.IContext) *ResidentMesh {
	return &ResidentMesh{
		ctx:                ctx,
		lastInstByteOffset: -1,
	}
}

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

// NeedsAnimRebuild reports whether any animated material on this resident mesh changed frame.
func (rm *ResidentMesh) NeedsAnimRebuild() bool {
	for i := 0; i < rm.AnimLen; i++ {
		if rm.AnimMats[i].GetCurrentFrame() != rm.AnimFrames[i] {
			return true
		}
	}
	return false
}

// NeedsRebuild reports whether the mesh needs re-uploading due to Faces mutation or animated texture frame changes.
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

// Rebuild uploads faces into resident static VBO/EBO buffers and builds contiguous ResidentSubMesh ranges.
func (rm *ResidentMesh) Rebuild(faces *volumes.Faces, texLookup func(*textures.Texture) (float32, bool)) bool {
	rm.SrcFaces = faces
	for i := range rm.HasBlend {
		rm.HasBlend[i] = false
	}
	rm.HasSolidOpaque = false
	rm.HasCutoutOpaque = false
	rm.HasSolidPrepass = false
	rm.HasCutoutPrepass = false
	rm.SkyMaterial = nil
	prevAnimLen := rm.AnimLen
	rm.SubMeshesLen = 0
	rm.AnimLen = 0
	rm.vertFloatsLen = 0
	rm.indicesLen = 0

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

// ReserveStaticAtlas preallocates vertex, index, and submesh capacity for static atlas batching.
func (rm *ResidentMesh) ReserveStaticAtlas(totalFaces int) {
	if totalFaces <= 0 {
		return
	}
	neededVerts := totalFaces * 3
	neededFloats := neededVerts * StaticVertexFloats
	if len(rm.VertFloats) < neededFloats {
		rm.VertFloats = make([]float32, neededFloats)
		if rm.ctx != nil {
			rm.vertFloatsPtr = rm.ctx.Ptr(rm.VertFloats)
		}
	}
	if len(rm.Indices) < neededVerts {
		rm.Indices = make([]uint32, neededVerts)
		if rm.ctx != nil {
			rm.indicesPtr = rm.ctx.Ptr(rm.Indices)
		}
	}
}

// BeginStaticAtlas resets accumulated static geometry ranges before packing static map volumes.
func (rm *ResidentMesh) BeginStaticAtlas() {
	rm.SrcFaces = nil
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
	rm.vertFloatsLen = 0
	rm.indicesLen = 0
	rm.built = false
}

// AppendStaticFaces appends a non-animated Faces volume into this static atlas mesh and returns its submesh range.
// If any valid face uses an animated material, no geometry is committed and hasAnim is returned as true.
func (rm *ResidentMesh) AppendStaticFaces(faces *volumes.Faces, texLookup func(*textures.Texture) (float32, bool)) (subStart int, subCount int, skyMat *textures.Material, hasAnim bool, ok bool) {
	if faces == nil || faces.Len() == 0 {
		return 0, 0, nil, false, false
	}
	return rm.appendFacesRange(faces, texLookup, true)
}

// EndStaticAtlas uploads accumulated static atlas vertices and indices into one shared VAO/VBO/EBO.
func (rm *ResidentMesh) EndStaticAtlas() {
	rm.uploadToGPU()
	rm.built = true
}

// appendFacesRange appends renderable faces to the resident mesh, grouping them into submeshes by render state.
// It skips invalid, sky, or missing-texture faces and records the sky material when present.
// It returns the added submesh range, sky material, animation rejection state, and success flag.
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

	neededVerts := validCount * 3
	neededFloats := neededVerts * StaticVertexFloats
	reqFloats := rm.vertFloatsLen + neededFloats
	reqVerts := rm.indicesLen + neededVerts
	if len(rm.VertFloats) < reqFloats {
		newLen := max(len(rm.VertFloats)*2, reqFloats)
		grown := make([]float32, newLen)
		copy(grown, rm.VertFloats[:rm.vertFloatsLen])
		rm.VertFloats = grown
		if rm.ctx != nil {
			rm.vertFloatsPtr = rm.ctx.Ptr(rm.VertFloats)
		}
	}
	if len(rm.Indices) < reqVerts {
		newLen := max(len(rm.Indices)*2, reqVerts)
		grown := make([]uint32, newLen)
		copy(grown, rm.Indices[:rm.indicesLen])
		rm.Indices = grown
		if rm.ctx != nil {
			rm.indicesPtr = rm.ctx.Ptr(rm.Indices)
		}
	}

	subStart = rm.SubMeshesLen
	var curSub *ResidentSubMesh
	var curKey faceStateKey
	var centSumX, centSumY, centSumZ float64
	var centCount int

	vIdx := uint32(rm.indicesLen)
	fOut := rm.vertFloatsLen
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
			curSub.FirstIndex = int32(vIdx)
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
		centSumX += (p0x + p1x + p2x) / 3.0
		centSumY += (p0z + p1z + p2z) / 3.0
		centSumZ += -(p0y + p1y + p2y) / 3.0
		centCount++

		WriteStaticVertex(rm.VertFloats[fOut:], p0x, p0y, p0z, u0, v0, texLayer, nx0, ny0, nz0, tx0, ty0, tz0, s0, lmU[0], lmV[0], lmIDFloat, packed, spec, shininess, dScale, dFactor, envStr, renderMode, envTint)
		fOut += StaticVertexFloats
		WriteStaticVertex(rm.VertFloats[fOut:], p1x, p1y, p1z, u1, v1, texLayer, nx1, ny1, nz1, tx1, ty1, tz1, s1, lmU[1], lmV[1], lmIDFloat, packed, spec, shininess, dScale, dFactor, envStr, renderMode, envTint)
		fOut += StaticVertexFloats
		WriteStaticVertex(rm.VertFloats[fOut:], p2x, p2y, p2z, u2, v2, texLayer, nx2, ny2, nz2, tx2, ty2, tz2, s2, lmU[2], lmV[2], lmIDFloat, packed, spec, shininess, dScale, dFactor, envStr, renderMode, envTint)
		fOut += StaticVertexFloats

		rm.Indices[vIdx] = vIdx
		rm.Indices[vIdx+1] = vIdx + 1
		rm.Indices[vIdx+2] = vIdx + 2
		vIdx += 3
		curSub.IndexCount += 3
	}
	curSub.FinalizeCentroid(centSumX, centSumY, centSumZ, centCount)
	rm.vertFloatsLen = fOut
	rm.indicesLen = int(vIdx)
	return subStart, rm.SubMeshesLen - subStart, skyMat, false, true
}

// uploadToGPU creates GPU buffers when needed, uploads mesh vertex and index data, and resets instanced bindings.
func (rm *ResidentMesh) uploadToGPU() {
	if rm.VAO == 0 {
		rm.ctx.GenVertexArrays(1, &rm.VAO)
		rm.ctx.GenBuffers(1, &rm.VBO)
		rm.ctx.GenBuffers(1, &rm.EBO)
	}
	rm.ctx.BindVertexArray(rm.VAO)
	rm.ctx.BindBuffer(api.ARRAY_BUFFER, rm.VBO)
	if rm.vertFloatsLen > 0 {
		rm.ctx.BufferData(api.ARRAY_BUFFER, rm.vertFloatsLen*4, rm.vertFloatsPtr, api.STATIC_DRAW)
	}
	rm.ctx.BindBuffer(api.ELEMENT_ARRAY_BUFFER, rm.EBO)
	if rm.indicesLen > 0 {
		rm.ctx.BufferData(api.ELEMENT_ARRAY_BUFFER, rm.indicesLen*4, rm.indicesPtr, api.STATIC_DRAW)
	}
	ConfigureVertexAttribs(rm.ctx, StaticVertexFloats*4, false)
	rm.ctx.BindVertexArray(0)
	// Invalidate instanced VAO bindings so next instanced bind refreshes cleanly.
	rm.lastInstByteOffset = -1
}

// BindInstanced binds rm.VAOInstanced with per-instance mat4 attributes (locations 10..13) pointing into instanceVBO.
func (rm *ResidentMesh) BindInstanced(instanceVBO uint32, baseByteOffset int32) {
	rm.BindInstancedWithCache(instanceVBO, baseByteOffset, 0)
}

// BindInstancedWithCache binds rm.VAOInstanced only when it differs from boundVAO, returning rm.VAOInstanced.
func (rm *ResidentMesh) BindInstancedWithCache(instanceVBO uint32, baseByteOffset int32, boundVAO uint32) uint32 {
	if rm.VAOInstanced == 0 {
		rm.ctx.GenVertexArrays(1, &rm.VAOInstanced)
		rm.ctx.BindVertexArray(rm.VAOInstanced)
		rm.ctx.BindBuffer(api.ARRAY_BUFFER, rm.VBO)
		rm.ctx.BindBuffer(api.ELEMENT_ARRAY_BUFFER, rm.EBO)
		ConfigureVertexAttribs(rm.ctx, StaticVertexFloats*4, false)
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

// Dispose releases GPU buffers owned by this ResidentMesh.
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
