package shaders

import (
	"math"
	"sort"
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
	cullStateUnset = -1
	cullStateOff   = 0
	cullStateBack  = 1
	cullStateFront = 2
)

// ApplyMaterialCullState updates GL CULL_FACE and CullFace only when the desired state changes.
// Returns the updated current state token.
func ApplyMaterialCullState(ctx api.IContext, curState int, cullMode int, hasCutout bool, forceDisable bool) int {
	want := cullStateOff
	if !forceDisable && !hasCutout {
		switch cullMode {
		case config.CullFront:
			want = cullStateBack
		case config.CullBack:
			want = cullStateFront
		case config.CullNone:
			want = cullStateOff
		default:
			want = cullStateBack
		}
	}
	if want == curState {
		return curState
	}
	switch want {
	case cullStateOff:
		ctx.Disable(api.CULL_FACE)
	case cullStateBack:
		if curState <= cullStateOff {
			ctx.Enable(api.CULL_FACE)
		}
		ctx.CullFace(api.BACK)
	case cullStateFront:
		if curState <= cullStateOff {
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

type faceStateKey struct {
	blendMode     int
	cullMode      int
	hasCutout     bool
	depthWrite    bool
	polygonOffset bool
	sortOrder     int
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
	SubMeshes          []ResidentSubMesh
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
	VertFloats         []float32
	Indices            []uint32
	AnimMats           []*textures.Material
	AnimFrames         []*textures.Texture
	faceOrder          []int
	faceKeys           []faceStateKey
}

// NewResidentMesh creates an unpopulated ResidentMesh bound to ctx.
func NewResidentMesh(ctx api.IContext) *ResidentMesh {
	return &ResidentMesh{
		ctx:                ctx,
		lastInstByteOffset: -1,
	}
}

// NeedsAnimRebuild reports whether any animated material on this resident mesh changed frame.
func (rm *ResidentMesh) NeedsAnimRebuild() bool {
	for i, mat := range rm.AnimMats {
		if mat.GetCurrentFrame() != rm.AnimFrames[i] {
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
	rm.SubMeshes = rm.SubMeshes[:0]
	rm.AnimMats = rm.AnimMats[:0]
	rm.AnimFrames = rm.AnimFrames[:0]

	if faces == nil || faces.Len() == 0 {
		if faces != nil {
			rm.Version = faces.GetVersion()
			rm.GeomHash = faces.GetGeomHash()
			rm.FaceCount = 0
		}
		rm.built = true
		return false
	}

	nFaces := faces.Len()
	if cap(rm.faceOrder) < nFaces {
		rm.faceOrder = make([]int, nFaces)
		rm.faceKeys = make([]faceStateKey, nFaces)
	} else {
		rm.faceOrder = rm.faceOrder[:nFaces]
		rm.faceKeys = rm.faceKeys[:nFaces]
	}

	validCount := 0
	seenAnim := make(map[*textures.Material]struct{}, 8)
	for i := 0; i < nFaces; i++ {
		face := faces.At(i)
		if face == nil {
			continue
		}
		mat := face.GetMaterial()
		if mat == nil || mat.GetKind() == int(config.MaterialKindNone) {
			continue
		}
		if mat.GetKind() == int(config.MaterialKindSky) {
			rm.SkyMaterial = mat
			continue
		}
		tex := mat.GetCurrentFrame()
		if tex == nil {
			continue
		}
		if _, ok := texLookup(tex); !ok {
			continue
		}
		if mat.IsAnimated() {
			if _, seen := seenAnim[mat]; !seen {
				seenAnim[mat] = struct{}{}
				rm.AnimMats = append(rm.AnimMats, mat)
				rm.AnimFrames = append(rm.AnimFrames, tex)
			}
		}
		bm := mat.GetBlendMode()
		if bm < 0 || bm >= config.BlendModeLatest {
			bm = config.BlendModeOpaque
		}
		rm.faceKeys[i] = faceStateKey{
			blendMode:     bm,
			cullMode:      mat.GetCullMode(),
			hasCutout:     mat.HasCutout(),
			depthWrite:    mat.GetDepthWrite(),
			polygonOffset: mat.GetPolygonOffset(),
			sortOrder:     mat.GetSort(),
		}
		rm.faceOrder[validCount] = i
		validCount++
	}

	if validCount == 0 {
		rm.Version = faces.GetVersion()
		rm.GeomHash = faces.GetGeomHash()
		rm.FaceCount = nFaces
		rm.built = true
		return false
	}

	sort.SliceStable(rm.faceOrder[:validCount], func(a, b int) bool {
		ka := &rm.faceKeys[rm.faceOrder[a]]
		kb := &rm.faceKeys[rm.faceOrder[b]]
		if ka.blendMode != kb.blendMode {
			return ka.blendMode < kb.blendMode
		}
		if ka.hasCutout != kb.hasCutout {
			return !ka.hasCutout && kb.hasCutout
		}
		if ka.depthWrite != kb.depthWrite {
			return ka.depthWrite && !kb.depthWrite
		}
		if ka.polygonOffset != kb.polygonOffset {
			return !ka.polygonOffset && kb.polygonOffset
		}
		if ka.cullMode != kb.cullMode {
			return ka.cullMode < kb.cullMode
		}
		return ka.sortOrder < kb.sortOrder
	})

	neededVerts := validCount * 3
	neededFloats := neededVerts * StaticVertexFloats
	if cap(rm.VertFloats) < neededFloats {
		rm.VertFloats = make([]float32, neededFloats)
	} else {
		rm.VertFloats = rm.VertFloats[:neededFloats]
	}
	if cap(rm.Indices) < neededVerts {
		rm.Indices = make([]uint32, neededVerts)
	} else {
		rm.Indices = rm.Indices[:neededVerts]
	}

	var curSub *ResidentSubMesh
	var curKey faceStateKey
	var centSumX, centSumY, centSumZ float64
	var centCount int

	flushSubMesh := func() {
		if curSub != nil && centCount > 0 {
			inv := 1.0 / float64(centCount)
			curSub.CentroidX = float32(centSumX * inv)
			curSub.CentroidY = float32(centSumY * inv)
			curSub.CentroidZ = float32(centSumZ * inv)
		}
	}

	vIdx := uint32(0)
	fOut := 0
	for k := 0; k < validCount; k++ {
		fi := rm.faceOrder[k]
		face := faces.At(fi)
		mat := face.GetMaterial()
		fk := rm.faceKeys[fi]
		texLayer, _ := texLookup(mat.GetCurrentFrame())

		if curSub == nil || fk != curKey {
			flushSubMesh()
			curKey = fk
			centSumX, centSumY, centSumZ = 0, 0, 0
			centCount = 0
			rm.SubMeshes = append(rm.SubMeshes, ResidentSubMesh{
				BlendMode:     fk.blendMode,
				CullMode:      fk.cullMode,
				HasCutout:     fk.hasCutout,
				DepthWrite:    fk.depthWrite,
				PolygonOffset: fk.polygonOffset,
				SortOrder:     fk.sortOrder,
				FirstIndex:    int32(vIdx),
				IndexCount:    0,
			})
			curSub = &rm.SubMeshes[len(rm.SubMeshes)-1]
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

		writeVert := func(px, py, pz, u, v, nx, ny, nz, tx, ty, tz, tSign, lmu, lmv float64) {
			rm.VertFloats[fOut] = float32(px)
			rm.VertFloats[fOut+1] = float32(pz)
			rm.VertFloats[fOut+2] = float32(-py)
			rm.VertFloats[fOut+3] = float32(u)
			rm.VertFloats[fOut+4] = float32(-v)
			rm.VertFloats[fOut+5] = texLayer
			rm.VertFloats[fOut+6] = float32(nx)
			rm.VertFloats[fOut+7] = float32(nz)
			rm.VertFloats[fOut+8] = float32(-ny)
			rm.VertFloats[fOut+9] = float32(tx)
			rm.VertFloats[fOut+10] = float32(tz)
			rm.VertFloats[fOut+11] = float32(-ty)
			rm.VertFloats[fOut+12] = float32(tSign)
			rm.VertFloats[fOut+13] = float32(lmu)
			rm.VertFloats[fOut+14] = float32(lmv)
			rm.VertFloats[fOut+15] = lmIDFloat
			rm.VertFloats[fOut+16] = packed
			rm.VertFloats[fOut+17] = spec
			rm.VertFloats[fOut+18] = shininess
			rm.VertFloats[fOut+19] = dScale
			rm.VertFloats[fOut+20] = dFactor
			rm.VertFloats[fOut+21] = envStr
			rm.VertFloats[fOut+22] = renderMode
			rm.VertFloats[fOut+23] = envTint
			fOut += StaticVertexFloats
		}

		writeVert(p0x, p0y, p0z, u0, v0, nx0, ny0, nz0, tx0, ty0, tz0, s0, lmU[0], lmV[0])
		writeVert(p1x, p1y, p1z, u1, v1, nx1, ny1, nz1, tx1, ty1, tz1, s1, lmU[1], lmV[1])
		writeVert(p2x, p2y, p2z, u2, v2, nx2, ny2, nz2, tx2, ty2, tz2, s2, lmU[2], lmV[2])

		rm.Indices[vIdx] = vIdx
		rm.Indices[vIdx+1] = vIdx + 1
		rm.Indices[vIdx+2] = vIdx + 2
		vIdx += 3
		curSub.IndexCount += 3
	}
	flushSubMesh()

	rm.uploadToGPU()
	rm.Version = faces.GetVersion()
	rm.GeomHash = faces.GetGeomHash()
	rm.FaceCount = nFaces
	rm.built = true
	return len(rm.SubMeshes) > 0
}

func (rm *ResidentMesh) uploadToGPU() {
	if rm.VAO == 0 {
		rm.ctx.GenVertexArrays(1, &rm.VAO)
		rm.ctx.GenBuffers(1, &rm.VBO)
		rm.ctx.GenBuffers(1, &rm.EBO)
	}
	rm.ctx.BindVertexArray(rm.VAO)
	rm.ctx.BindBuffer(api.ARRAY_BUFFER, rm.VBO)
	if len(rm.VertFloats) > 0 {
		rm.ctx.BufferData(api.ARRAY_BUFFER, len(rm.VertFloats)*4, rm.ctx.Ptr(rm.VertFloats), api.STATIC_DRAW)
	}
	rm.ctx.BindBuffer(api.ELEMENT_ARRAY_BUFFER, rm.EBO)
	if len(rm.Indices) > 0 {
		rm.ctx.BufferData(api.ELEMENT_ARRAY_BUFFER, len(rm.Indices)*4, rm.ctx.Ptr(rm.Indices), api.STATIC_DRAW)
	}
	ConfigureVertexAttribs(rm.ctx, StaticVertexFloats*4, false)
	rm.ctx.BindVertexArray(0)
	// Invalidate instanced VAO bindings so next instanced bind refreshes cleanly.
	rm.lastInstByteOffset = -1
}

// BindInstanced binds rm.VAOInstanced with per-instance mat4 attributes (locations 10..13) pointing into instanceVBO.
func (rm *ResidentMesh) BindInstanced(instanceVBO uint32, baseByteOffset int32) {
	if rm.VAOInstanced == 0 {
		rm.ctx.GenVertexArrays(1, &rm.VAOInstanced)
		rm.ctx.BindVertexArray(rm.VAOInstanced)
		rm.ctx.BindBuffer(api.ARRAY_BUFFER, rm.VBO)
		rm.ctx.BindBuffer(api.ELEMENT_ARRAY_BUFFER, rm.EBO)
		ConfigureVertexAttribs(rm.ctx, StaticVertexFloats*4, false)
		rm.boundInstVBO = 0
		rm.lastInstByteOffset = -1
	} else {
		rm.ctx.BindVertexArray(rm.VAOInstanced)
	}

	if rm.boundInstVBO != instanceVBO || rm.lastInstByteOffset != baseByteOffset {
		rm.ctx.BindBuffer(api.ARRAY_BUFFER, instanceVBO)
		const matStride = InstanceMatrixFloats * 4
		for col := uint32(0); col < 4; col++ {
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

// InstancePool accumulates per-instance 4x4 transform matrices for the active frame
// and uploads them once per frame into a double-buffered GPU VBO.
type InstancePool struct {
	ctx      api.IContext
	data     []float32
	ptr      unsafe.Pointer
	count    int32
	vbo      [2]uint32
	vboCap   [2]int
	frameIdx int
}

// NewInstancePool allocates an InstancePool with initial instance matrix capacity.
func NewInstancePool(ctx api.IContext, initInstances int) *InstancePool {
	if initInstances < 64 {
		initInstances = 64
	}
	data := make([]float32, initInstances*InstanceMatrixFloats)
	return &InstancePool{
		ctx:  ctx,
		data: data,
		ptr:  unsafe.Pointer(&data[0]),
	}
}

// Reset clears the instance list at the start of a frame while advancing the double-buffer slot.
func (p *InstancePool) Reset() {
	p.count = 0
	p.frameIdx = (p.frameIdx + 1) & 1
}

// Reserve allocates consecutive matrix slots and returns the base byte offset and base float index.
func (p *InstancePool) Reserve(numInstances int) (baseByteOffset int32, baseFloat int) {
	needed := (int(p.count) + numInstances) * InstanceMatrixFloats
	if needed > len(p.data) {
		newLen := len(p.data) * 2
		for newLen < needed {
			newLen *= 2
		}
		grown := make([]float32, newLen)
		copy(grown, p.data[:int(p.count)*InstanceMatrixFloats])
		p.data = grown
		p.ptr = unsafe.Pointer(&p.data[0])
	}
	baseByteOffset = p.count * (InstanceMatrixFloats * 4)
	baseFloat = int(p.count) * InstanceMatrixFloats
	p.count += int32(numInstances)
	return baseByteOffset, baseFloat
}

// WriteTransformAt converts world position (cx, cy, cz) and physics quaternion (qx, qy, qz, qw)
// into a column-major GL mat4 at float offset baseFloat.
func (p *InstancePool) WriteTransformAt(baseFloat int, cx, cy, cz, qx, qy, qz, qw float64) {
	// Normalize quaternion defensively
	n2 := qx*qx + qy*qy + qz*qz + qw*qw
	if n2 > 1e-12 && math.Abs(n2-1.0) > 1e-6 {
		inv := 1.0 / math.Sqrt(n2)
		qx, qy, qz, qw = qx*inv, qy*inv, qz*inv, qw*inv
	} else if n2 <= 1e-12 {
		qx, qy, qz, qw = 0, 0, 0, 1
	}

	xx, yy, zz := qx*qx, qy*qy, qz*qz
	xy, xz, yz := qx*qy, qx*qz, qy*qz
	wx, wy, wz := qw*qx, qw*qy, qw*qz

	r00 := 1.0 - 2.0*(yy+zz)
	r01 := 2.0 * (xy - wz)
	r02 := 2.0 * (xz + wy)

	r10 := 2.0 * (xy + wz)
	r11 := 1.0 - 2.0*(xx+zz)
	r12 := 2.0 * (yz - wx)

	r20 := 2.0 * (xz - wy)
	r21 := 2.0 * (yz + wx)
	r22 := 1.0 - 2.0*(xx+yy)

	d := p.data[baseFloat : baseFloat+16]
	// Col 0
	d[0] = float32(r00)
	d[1] = float32(r20)
	d[2] = float32(-r10)
	d[3] = 0
	// Col 1
	d[4] = float32(r02)
	d[5] = float32(r22)
	d[6] = float32(-r12)
	d[7] = 0
	// Col 2
	d[8] = float32(-r01)
	d[9] = float32(-r21)
	d[10] = float32(r11)
	d[11] = 0
	// Col 3
	d[12] = float32(cx)
	d[13] = float32(cz)
	d[14] = float32(-cy)
	d[15] = 1
}

// ActiveVBO returns the current double-buffered GPU buffer ID containing this frame's instance matrices.
func (p *InstancePool) ActiveVBO() uint32 {
	return p.vbo[p.frameIdx]
}

// Upload transfers accumulated instance matrices to the active GPU VBO in one buffer call.
func (p *InstancePool) Upload() uint32 {
	if p.count == 0 {
		return 0
	}
	idx := p.frameIdx
	if p.vbo[idx] == 0 {
		p.ctx.GenBuffers(1, &p.vbo[idx])
	}
	bytesNeeded := int(p.count) * (InstanceMatrixFloats * 4)
	p.ctx.BindBuffer(api.ARRAY_BUFFER, p.vbo[idx])
	if bytesNeeded > p.vboCap[idx] {
		newCap := bytesNeeded * 2
		if newCap < 64*InstanceMatrixFloats*4 {
			newCap = 64 * InstanceMatrixFloats * 4
		}
		p.ctx.BufferData(api.ARRAY_BUFFER, newCap, nil, api.DYNAMIC_DRAW)
		p.vboCap[idx] = newCap
	}
	p.ctx.BufferSubData(api.ARRAY_BUFFER, 0, bytesNeeded, p.ptr)
	p.ctx.BindBuffer(api.ARRAY_BUFFER, 0)
	return p.vbo[idx]
}

// StaticDrawItem binds a GPU-resident static submesh for direct drawing.
type StaticDrawItem struct {
	Mesh   *ResidentMesh
	SubIdx int
}

// InstancedDrawItem binds a contiguous slice of instances sharing one GPU-resident submesh.
type InstancedDrawItem struct {
	Mesh           *ResidentMesh
	SubIdx         int
	InstanceVBO    uint32
	BaseByteOffset int32
	InstanceCount  int32
}

// TransparentKind identifies the source pipeline for a sorted translucent item.
type TransparentKind uint8

const (
	TransparentStreaming TransparentKind = iota
	TransparentStatic
	TransparentInstanced
)

// TransparentDrawItem holds one translucent submesh/command sorted back-to-front per frame.
type TransparentDrawItem struct {
	Kind           TransparentKind
	DistSq         float32
	SortOrder      int
	CullMode       int
	HasCutout      bool
	PolygonOffset  bool
	FirstIndex     int32
	IndexCount     int32
	Mesh           *ResidentMesh
	SubIdx         int
	InstanceVBO    uint32
	BaseByteOffset int32
	InstanceCount  int32
}

// SortTransparentDrawItems sorts translucent items strictly back-to-front (largest DistSq first),
// using SortOrder ascending as a deterministic co-planar tie-breaker.
func SortTransparentDrawItems(items []TransparentDrawItem) {
	if len(items) <= 1 {
		return
	}
	sort.SliceStable(items, func(i, j int) bool {
		di, dj := items[i].DistSq, items[j].DistSq
		if math.Abs(float64(di-dj)) > 1e-2 {
			return di > dj
		}
		return items[i].SortOrder < items[j].SortOrder
	})
}
