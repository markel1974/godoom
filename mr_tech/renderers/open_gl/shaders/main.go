package shaders

import (
	"math"
	"strings"
	"unsafe"

	"github.com/markel1974/godoom/mr_tech/model"
	"github.com/markel1974/godoom/mr_tech/model/things"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/gpu/buffers"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/gpu/draws"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/metrics"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// MainLoc represents an integer-based enumerator used as an identifier for uniform locations within shaders.
type MainLoc int

const (
	mainDoubleBuffer = 2

	uboBindingLights       = 0
	uboBindingPerFrame     = 1
	uboBindingShadowLights = 2

	// PerFrameBlock std140 word offsets (every slot is mat4, vec4, or ivec4 -> 16-byte aligned).
	pfOffView             = 0
	pfOffInvView          = 16
	pfOffProjection       = 32
	pfOffRoomSpace        = 48
	pfOffFlashSpace       = 64
	pfOffSunSpaces        = 80
	pfOffScreenTime       = 144
	pfOffLightingParams   = 148
	pfOffFlashPosInt      = 152
	pfOffFlashDirFall     = 156
	pfOffFlashConeCluster = 160
	pfOffFlags            = 164
	pfOffCounts           = 168
	pfOffSunDirView       = 172
	pfOffSunColorInt      = 176
	pfOffSunSkyColor      = 180
	pfOffSunGroundColor   = 184
	pfOffSunSplits        = 188
	pfOffSunBias          = 192
	pfOffFogColor         = 196
	pfOffFogSunColor      = 200
	pfOffFogParams        = 204
	pfOffFogHeightParams  = 208
	perFrameBlockWords    = 212
	perFrameBlockBytes    = perFrameBlockWords * 4

	// ShadowLightsBlock std140 layout (8 entries * (4 vec4 + 1 mat4) = 8 * 32 words = 1024 bytes).
	shadowLightEntryWords  = 32
	shadowLightEntryBytes  = shadowLightEntryWords * 4
	shadowLightsBlockWords = metrics.ShadowLightsCount * shadowLightEntryWords
	shadowLightsBlockBytes = shadowLightsBlockWords * 4
)

const (
	MainLocTexture = MainLoc(iota)
	MainLocNormalMap
	MainLocEmissiveMap
	MainLocShadowAtlas
	MainLocInstanceMode
	MainLocEnvMap
	MainLocLightmaps
	MainLocClusterData
	MainLocSSAO
	MainLocEnableSSAO
	MainLocLast
)

type mainProgram struct {
	prg   uint32
	table [MainLocLast]int32
}

type zPrepassProgram struct {
	prg             uint32
	locTex          int32
	locInstanceMode int32
}

// Main represents the central structure for managing rendering states and operations.
type Main struct {
	ctx                 api.IContext
	solidPrg            mainProgram
	cutoutPrg           mainProgram
	zPrepassSolid       zPrepassProgram
	zPrepassCutout      zPrepassProgram
	mainVAO             [mainDoubleBuffer]uint32
	mainVBO             [mainDoubleBuffer]uint32
	mainEBO             [mainDoubleBuffer]uint32
	uboPerFrame         [mainDoubleBuffer]uint32
	uboShadowLights     [mainDoubleBuffer]uint32
	perFrameBuf         []uint32
	perFramePtr         unsafe.Pointer
	shadowLightsBuf     []uint32
	shadowLightsPtr     unsafe.Pointer
	perFrameReady       bool
	perFrameTick        uint64
	perFrameAmbient     float32
	perFrameFlashX      float32
	perFrameFlashY      float32
	perFrameFlashSens   float32
	vboBytesCap         [mainDoubleBuffer]int
	eboBytesCap         [mainDoubleBuffer]int
	vboStaticGen        [mainDoubleBuffer]uint64
	vboStaticBytes      [mainDoubleBuffer]int
	vboFrameGen         [mainDoubleBuffer]uint64
	eboStaticGen        [mainDoubleBuffer]uint64
	eboStaticBytes      [mainDoubleBuffer]int
	eboFrameGen         [mainDoubleBuffer]uint64
	frameIdx            int
	emissiveIntensity   float32
	stride              int32
	w                   int32
	h                   int32
	shadows             int32
	debugLights         int
	handler             *Handler
	cal                 *model.Calibration
	metrics             *metrics.Map
	flashlightMetrics   *metrics.Flashlight
	shadowLightsMetrics *metrics.ShadowLights
	sunCSM              *metrics.SunCSM
	atmosphereFog       *metrics.AtmosphereFog
	flash               *things.Flash
	depth               *Depth
	lights              *Lights
	post                *Post
	ssao                *SSAO
	aoFactor            float32
	envMaxLod           float32
}

// NewMain initializes and returns a new instance of Main with the provided context, stride, and metrics.
func NewMain(stride int32) *Main {
	perFrameBuf := make([]uint32, perFrameBlockWords)
	shadowLightsBuf := make([]uint32, shadowLightsBlockWords)
	return &Main{
		stride:          stride,
		shadows:         0,
		aoFactor:        0.8,
		envMaxLod:       5.0,
		perFrameBuf:     perFrameBuf,
		perFramePtr:     unsafe.Pointer(&perFrameBuf[0]),
		shadowLightsBuf: shadowLightsBuf,
		shadowLightsPtr: unsafe.Pointer(&shadowLightsBuf[0]),
	}
}

func (s *Main) ensureUBOBuffers() {
	if len(s.perFrameBuf) < perFrameBlockWords {
		s.perFrameBuf = make([]uint32, perFrameBlockWords)
		s.perFramePtr = unsafe.Pointer(&s.perFrameBuf[0])
	}
	if len(s.shadowLightsBuf) < shadowLightsBlockWords {
		s.shadowLightsBuf = make([]uint32, shadowLightsBlockWords)
		s.shadowLightsPtr = unsafe.Pointer(&s.shadowLightsBuf[0])
	}
}

// Setup initializes the Main instance by setting its context using the provided Handler and returns an error if any.
func (s *Main) Setup(h *Handler) error {
	s.handler = h
	s.ctx = h.GetContext()
	s.cal = h.GetCalibration()
	s.metrics = h.GetMapMetrics()
	s.flashlightMetrics = h.GetFlashlightMetrics()
	s.shadowLightsMetrics = h.GetShadowLightsMetrics()
	s.sunCSM = h.GetSunCSM()
	s.atmosphereFog = h.GetAtmosphereFog()
	s.flash = h.GetFlash()
	s.depth = h.GetDepth()
	s.lights = h.GetLights()
	s.post = h.GetPost()
	s.ssao = h.GetSSAO()
	s.aoFactor = h.GetAoFactor()
	s.envMaxLod = 5.0
	s.emissiveIntensity = h.GetEmissiveIntensity()
	s.shadows = 0

	return nil
}

// SetEnvMaxLod updates the maximum mipmap LOD index available on the environment cubemap.
func (s *Main) SetEnvMaxLod(lod float32) {
	if lod < 0 {
		lod = 0
	}
	s.envMaxLod = lod
	s.perFrameReady = false
}

// SetShadowsEnabled enables or disables shadow evaluation in the main shader.
func (s *Main) SetShadowsEnabled(e bool) {
	if e {
		s.shadows = 1
	} else {
		s.shadows = 0
	}
	s.perFrameReady = false
}

// Init initializes the necessary OpenGL resources, such as VAOs, VBOs, EBOs, and per-frame UBOs, and configures the vertex attributes.
func (s *Main) Init() error {
	vboBytesSize := 131072 * int(s.stride)
	eboBytesSize := 262144 * 4

	s.ctx.GenVertexArrays(mainDoubleBuffer, &s.mainVAO[0])
	s.ctx.GenBuffers(mainDoubleBuffer, &s.mainVBO[0])
	s.ctx.GenBuffers(mainDoubleBuffer, &s.mainEBO[0])
	s.ctx.GenBuffers(mainDoubleBuffer, &s.uboPerFrame[0])
	s.ctx.GenBuffers(mainDoubleBuffer, &s.uboShadowLights[0])

	for i := 0; i < mainDoubleBuffer; i++ {
		s.vboBytesCap[i] = vboBytesSize
		s.eboBytesCap[i] = eboBytesSize

		s.ctx.BindVertexArray(s.mainVAO[i])

		s.ctx.BindBuffer(api.ARRAY_BUFFER, s.mainVBO[i])
		s.ctx.BufferData(api.ARRAY_BUFFER, vboBytesSize, nil, api.DYNAMIC_DRAW)

		s.ctx.BindBuffer(api.ELEMENT_ARRAY_BUFFER, s.mainEBO[i])
		s.ctx.BufferData(api.ELEMENT_ARRAY_BUFFER, eboBytesSize, nil, api.DYNAMIC_DRAW)

		buffers.ConfigureVertexAttribs(s.ctx, s.stride, true)

		s.ctx.BindBuffer(api.UNIFORM_BUFFER, s.uboPerFrame[i])
		s.ctx.BufferData(api.UNIFORM_BUFFER, perFrameBlockBytes, nil, api.DYNAMIC_DRAW)

		s.ctx.BindBuffer(api.UNIFORM_BUFFER, s.uboShadowLights[i])
		s.ctx.BufferData(api.UNIFORM_BUFFER, shadowLightsBlockBytes, nil, api.DYNAMIC_DRAW)
	}

	s.ctx.BindBuffer(api.UNIFORM_BUFFER, 0)
	s.ctx.BindVertexArray(0)
	s.ctx.Enable(api.DEPTH_TEST)
	s.ctx.DepthFunc(api.LEQUAL)
	return nil
}

// GetProgramOpaque returns the program ID used for rendering solid opaque objects without discard.
func (s *Main) GetProgramOpaque() uint32 {
	return s.solidPrg.prg
}

// GetProgramCutout returns the program ID used for rendering alpha-cutout opaque objects.
func (s *Main) GetProgramCutout() uint32 {
	return s.cutoutPrg.prg
}

// GetProgramZPrepassSolid returns the program ID used for the solid Z-prepass.
func (s *Main) GetProgramZPrepassSolid() uint32 {
	return s.zPrepassSolid.prg
}

// GetProgramZPrepassCutout returns the program ID used for the cutout Z-prepass.
func (s *Main) GetProgramZPrepassCutout() uint32 {
	return s.zPrepassCutout.prg
}

// GetUniformOpaque retrieves the uniform location for the opaque shader program using the given MainLoc identifier.
func (s *Main) GetUniformOpaque(id MainLoc) int32 {
	return s.solidPrg.table[id]
}

// GetVAO returns the Vertex Array Object (VAO) associated with the current frame index.
func (s *Main) GetVAO() uint32 {
	return s.mainVAO[s.frameIdx]
}

// SetupSamplers initializes sampler uniforms for opaque, cutout, and Z-prepass shader programs.
func (s *Main) SetupSamplers() error {
	diffuseUnits := []int32{api.TexUnitDiffuseBase, api.TexUnitDiffuseBase + 1, api.TexUnitDiffuseBase + 2}
	normalUnits := []int32{api.TexUnitNormalBase, api.TexUnitNormalBase + 1, api.TexUnitNormalBase + 2}
	emissiveUnits := []int32{api.TexUnitEmissiveBase, api.TexUnitEmissiveBase + 1, api.TexUnitEmissiveBase + 2}

	programs := [2]*mainProgram{&s.solidPrg, &s.cutoutPrg}
	for _, p := range programs {
		s.ctx.UseProgram(p.prg)
		s.ctx.Uniform1iv(p.table[MainLocTexture], 3, &diffuseUnits[0])
		s.ctx.Uniform1iv(p.table[MainLocNormalMap], 3, &normalUnits[0])
		s.ctx.Uniform1iv(p.table[MainLocEmissiveMap], 3, &emissiveUnits[0])
		s.ctx.Uniform1i(p.table[MainLocShadowAtlas], api.TexUnitShadowAtlas)
		if p.table[MainLocEnvMap] >= 0 {
			s.ctx.Uniform1i(p.table[MainLocEnvMap], api.TexUnitEnvMap)
		}
		if p.table[MainLocLightmaps] >= 0 {
			s.ctx.Uniform1i(p.table[MainLocLightmaps], api.TexUnitLightmap)
		}
		if p.table[MainLocClusterData] >= 0 {
			s.ctx.Uniform1i(p.table[MainLocClusterData], api.TexUnitClusterData)
		}
		if p.table[MainLocSSAO] >= 0 {
			s.ctx.Uniform1i(p.table[MainLocSSAO], api.TexUnitSSAO)
		}
	}

	if s.zPrepassCutout.prg != 0 && s.zPrepassCutout.locTex >= 0 {
		s.ctx.UseProgram(s.zPrepassCutout.prg)
		s.ctx.Uniform1iv(s.zPrepassCutout.locTex, 3, &diffuseUnits[0])
	}

	return nil
}

func injectShaderDefine(src string, defineLine string) string {
	const header = "#version 330 core"
	if strings.Contains(src, header) {
		return strings.Replace(src, header, header+"\n"+defineLine, 1)
	}
	return defineLine + "\n" + src
}

// Compile initializes and compiles shader programs for Z-prepass and opaque/cutout forward rendering.
func (s *Main) Compile(a IAssets) error {
	const vertId = "main.vert"
	const fragOpaqueId = "main_opaque.frag"
	const fragZSolidId = "zprepass.frag"
	const fragZCutoutId = "depth.frag"

	vertexSrc, fragmentOpaqueSrc, err := a.ReadMulti(vertId, fragOpaqueId)
	if err != nil {
		return err
	}
	zSolidSrc, err := a.Read(fragZSolidId)
	if err != nil {
		return err
	}
	zCutoutSrc, err := a.Read(fragZCutoutId)
	if err != nil {
		return err
	}

	vertUBOSrc := []byte(injectShaderDefine(string(vertexSrc), "#define USE_PERFRAME_UBO 1"))

	if err := s.compileForwardProgram(&s.solidPrg, "main_opaque", vertId, vertUBOSrc, fragOpaqueId, fragmentOpaqueSrc); err != nil {
		return err
	}
	cutoutSrc := injectShaderDefine(string(fragmentOpaqueSrc), "#define ALPHA_CUTOUT 1")
	if err := s.compileForwardProgram(&s.cutoutPrg, "main_cutout", vertId, vertUBOSrc, fragOpaqueId, []byte(cutoutSrc)); err != nil {
		return err
	}
	if err := s.compileZPrepassProgram(&s.zPrepassSolid, "zprepass_solid", vertId, vertUBOSrc, fragZSolidId, zSolidSrc, false); err != nil {
		return err
	}
	if err := s.compileZPrepassProgram(&s.zPrepassCutout, "zprepass_cutout", vertId, vertUBOSrc, fragZCutoutId, zCutoutSrc, true); err != nil {
		return err
	}
	return nil
}

func (s *Main) bindUniformBlock(prg uint32, name string, binding uint32) {
	blockIndex := s.ctx.GetUniformBlockIndex(prg, s.ctx.Str(name))
	if blockIndex != api.INVALID_INDEX {
		s.ctx.UniformBlockBinding(prg, blockIndex, binding)
	}
}

func (s *Main) compileForwardProgram(dst *mainProgram, progName, vertID string, vertSrc []byte, fragID string, fragSrc []byte) error {
	vertexShader, err := ShaderCompile(s.ctx, vertID, string(vertSrc), api.VERTEX_SHADER)
	if err != nil {
		return err
	}
	fragShader, err := ShaderCompile(s.ctx, fragID, string(fragSrc), api.FRAGMENT_SHADER)
	if err != nil {
		s.ctx.DeleteShader(vertexShader)
		return err
	}
	dst.prg, err = ShaderCreateProgram(s.ctx, progName, vertexShader, fragShader)
	if err != nil {
		return err
	}

	prg := dst.prg
	dst.table[MainLocTexture] = s.ctx.GetUniformLocation(prg, s.ctx.Str("u_texture\x00"))
	dst.table[MainLocNormalMap] = s.ctx.GetUniformLocation(prg, s.ctx.Str("u_normalMap\x00"))
	dst.table[MainLocEmissiveMap] = s.ctx.GetUniformLocation(prg, s.ctx.Str("u_emissiveMap\x00"))
	dst.table[MainLocShadowAtlas] = s.ctx.GetUniformLocation(prg, s.ctx.Str("u_shadowAtlas\x00"))
	dst.table[MainLocInstanceMode] = s.ctx.GetUniformLocation(prg, s.ctx.Str("u_instanceMode\x00"))
	dst.table[MainLocEnvMap] = s.ctx.GetUniformLocation(prg, s.ctx.Str("u_envMap\x00"))
	dst.table[MainLocLightmaps] = s.ctx.GetUniformLocation(prg, s.ctx.Str("u_lightmaps\x00"))
	dst.table[MainLocClusterData] = s.ctx.GetUniformLocation(prg, s.ctx.Str("u_clusterData\x00"))
	dst.table[MainLocSSAO] = s.ctx.GetUniformLocation(prg, s.ctx.Str("u_ssao\x00"))
	dst.table[MainLocEnableSSAO] = s.ctx.GetUniformLocation(prg, s.ctx.Str("u_enableSSAO\x00"))

	s.bindUniformBlock(prg, "LightsBlock\x00", uboBindingLights)
	s.bindUniformBlock(prg, "PerFrameBlock\x00", uboBindingPerFrame)
	s.bindUniformBlock(prg, "ShadowLightsBlock\x00", uboBindingShadowLights)
	return nil
}

func (s *Main) compileZPrepassProgram(dst *zPrepassProgram, progName, vertID string, vertSrc []byte, fragID string, fragSrc []byte, withTexture bool) error {
	vertexShader, err := ShaderCompile(s.ctx, vertID, string(vertSrc), api.VERTEX_SHADER)
	if err != nil {
		return err
	}
	fragShader, err := ShaderCompile(s.ctx, fragID, string(fragSrc), api.FRAGMENT_SHADER)
	if err != nil {
		s.ctx.DeleteShader(vertexShader)
		return err
	}
	dst.prg, err = ShaderCreateProgram(s.ctx, progName, vertexShader, fragShader)
	if err != nil {
		return err
	}
	dst.locInstanceMode = s.ctx.GetUniformLocation(dst.prg, s.ctx.Str("u_instanceMode\x00"))
	dst.locTex = -1
	if withTexture {
		dst.locTex = s.ctx.GetUniformLocation(dst.prg, s.ctx.Str("u_texture\x00"))
	}
	s.bindUniformBlock(dst.prg, "PerFrameBlock\x00", uboBindingPerFrame)
	return nil
}

// Prepare configures the rendering pipeline, updates buffer data, and manages double buffering for rendering operations.
func (s *Main) Prepare(vPtr unsafe.Pointer, vSize int, vCount int32, iPtr unsafe.Pointer, iSize int, iCount int32) {
	s.PrepareFrame(vPtr, vSize, 0, iPtr, iSize, 0, 0, 0)
}

// PrepareFrame uploads vertices and indices into the active double-buffered VAO/VBO/EBO, skipping unchanged static prefixes or identical frames.
func (s *Main) PrepareFrame(vPtr unsafe.Pointer, vSize int, staticVertBytes int, iPtr unsafe.Pointer, iSize int, staticIdxBytes int, staticGen uint64, frameGen uint64) {
	idx := (s.frameIdx + 1) % mainDoubleBuffer
	s.frameIdx = idx
	s.perFrameReady = false

	s.ctx.BindVertexArray(s.mainVAO[idx])

	s.ctx.BindBuffer(api.ARRAY_BUFFER, s.mainVBO[idx])
	if vSize > s.vboBytesCap[idx] {
		newCap := vSize * 2
		s.ctx.BufferData(api.ARRAY_BUFFER, newCap, nil, api.DYNAMIC_DRAW)
		s.vboBytesCap[idx] = newCap
		s.vboStaticGen[idx] = 0
		s.vboStaticBytes[idx] = 0
		s.vboFrameGen[idx] = 0
	}
	if vSize > 0 {
		if frameGen != 0 && s.vboFrameGen[idx] == frameGen {
			// Entire vertex buffer in this double-buffer slot is already current.
		} else if staticGen != 0 && s.vboStaticGen[idx] == staticGen && s.vboStaticBytes[idx] == staticVertBytes && staticVertBytes <= vSize {
			if dynBytes := vSize - staticVertBytes; dynBytes > 0 {
				s.ctx.BufferSubData(api.ARRAY_BUFFER, staticVertBytes, dynBytes, unsafe.Add(vPtr, staticVertBytes))
			}
			s.vboFrameGen[idx] = frameGen
		} else {
			s.ctx.BufferSubData(api.ARRAY_BUFFER, 0, vSize, vPtr)
			s.vboStaticGen[idx] = staticGen
			s.vboStaticBytes[idx] = staticVertBytes
			s.vboFrameGen[idx] = frameGen
		}
	} else {
		s.vboStaticGen[idx] = 0
		s.vboStaticBytes[idx] = 0
		s.vboFrameGen[idx] = frameGen
	}

	s.ctx.BindBuffer(api.ELEMENT_ARRAY_BUFFER, s.mainEBO[idx])
	if iSize > s.eboBytesCap[idx] {
		newCap := iSize * 2
		s.ctx.BufferData(api.ELEMENT_ARRAY_BUFFER, newCap, nil, api.DYNAMIC_DRAW)
		s.eboBytesCap[idx] = newCap
		s.eboStaticGen[idx] = 0
		s.eboStaticBytes[idx] = 0
		s.eboFrameGen[idx] = 0
	}
	if iSize > 0 {
		if frameGen != 0 && s.eboFrameGen[idx] == frameGen {
			// Entire index buffer in this double-buffer slot is already current.
		} else if staticGen != 0 && s.eboStaticGen[idx] == staticGen && s.eboStaticBytes[idx] == staticIdxBytes && staticIdxBytes <= iSize {
			if dynBytes := iSize - staticIdxBytes; dynBytes > 0 {
				s.ctx.BufferSubData(api.ELEMENT_ARRAY_BUFFER, staticIdxBytes, dynBytes, unsafe.Add(iPtr, staticIdxBytes))
			}
			s.eboFrameGen[idx] = frameGen
		} else {
			s.ctx.BufferSubData(api.ELEMENT_ARRAY_BUFFER, 0, iSize, iPtr)
			s.eboStaticGen[idx] = staticGen
			s.eboStaticBytes[idx] = staticIdxBytes
			s.eboFrameGen[idx] = frameGen
		}
	} else {
		s.eboStaticGen[idx] = 0
		s.eboStaticBytes[idx] = 0
		s.eboFrameGen[idx] = frameGen
	}

	s.ctx.BindVertexArray(0)
}

func writeMat4Words(dst []uint32, off int, ptr *float32) {
	if ptr == nil {
		clear(dst[off : off+16])
		dst[off+0] = math.Float32bits(1)
		dst[off+5] = math.Float32bits(1)
		dst[off+10] = math.Float32bits(1)
		dst[off+15] = math.Float32bits(1)
		return
	}
	copy(dst[off:off+16], unsafe.Slice((*uint32)(unsafe.Pointer(ptr)), 16))
}

func writeVec4Floats(dst []uint32, off int, x, y, z, w float32) {
	s := dst[off : off+4 : off+4]
	s[0] = math.Float32bits(x)
	s[1] = math.Float32bits(y)
	s[2] = math.Float32bits(z)
	s[3] = math.Float32bits(w)
}

func writeVec4Ints(dst []uint32, off int, x, y, z, w int32) {
	s := dst[off : off+4 : off+4]
	s[0] = uint32(x)
	s[1] = uint32(y)
	s[2] = uint32(z)
	s[3] = uint32(w)
}

func (s *Main) packShadowLightsUBO() int32 {
	if s.shadowLightsMetrics == nil || s.metrics == nil {
		return 0
	}
	numShadowLights := int(s.shadowLightsMetrics.GetLightCount())
	if numShadowLights > metrics.ShadowLightsCount {
		numShadowLights = metrics.ShadowLightsCount
	}
	buf := s.shadowLightsBuf
	for i := 0; i < numShadowLights; i++ {
		l, lMatrix := s.shadowLightsMetrics.GetLight(i)
		vx, vy, vz := s.metrics.TransformPointToView(l.X, l.Y, l.Z)
		dvx, dvy, dvz := s.metrics.TransformDirectionNormalizedToView(l.DirX, l.DirY, l.DirZ)
		effRadiusSq, invScale := metrics.ComputeLightAttenuation(l.Intensity, l.Falloff)
		base := i * shadowLightEntryWords
		writeVec4Floats(buf, base+0, vx, vy, vz, l.Intensity)
		writeVec4Floats(buf, base+4, dvx, dvy, dvz, l.Falloff)
		writeVec4Floats(buf, base+8, l.R, l.G, l.B, l.Kind)
		writeVec4Floats(buf, base+12, l.OuterCutOff, l.CutOff, effRadiusSq, invScale)
		writeMat4Words(buf, base+16, lMatrix)
	}
	return int32(numShadowLights)
}

func (s *Main) needsPerFrameUpload(tick uint64, ambient, flashX, flashY, flashSensitivity float32) bool {
	return !s.perFrameReady ||
		s.perFrameTick != tick ||
		s.perFrameAmbient != ambient ||
		s.perFrameFlashX != flashX ||
		s.perFrameFlashY != flashY ||
		s.perFrameFlashSens != flashSensitivity
}

func (s *Main) uploadPerFrameUBOs(
	tick uint64,
	fbW, fbH int32,
	interval, ambient, flashX, flashY, flashSensitivity float32,
	clusterNear, clusterLogScale float32,
	numGlobalLights int32,
) {
	s.ensureUBOBuffers()
	numShadowLights := s.packShadowLightsUBO()
	buf := s.perFrameBuf

	var viewPtr, invViewPtr, projPtr, roomSpacePtr, flashSpacePtr *float32
	if s.metrics != nil {
		viewPtr = s.metrics.GetViewPtr()
		invViewPtr = s.metrics.GetInvViewPtr()
		projPtr = s.metrics.GetProjPtr()
		roomSpacePtr = s.metrics.GetRoomSpacePtr()
	}
	if s.flashlightMetrics != nil {
		flashSpacePtr = s.flashlightMetrics.GetShadowSpacePtr()
	}
	writeMat4Words(buf, pfOffView, viewPtr)
	writeMat4Words(buf, pfOffInvView, invViewPtr)
	writeMat4Words(buf, pfOffProjection, projPtr)
	writeMat4Words(buf, pfOffRoomSpace, roomSpacePtr)
	writeMat4Words(buf, pfOffFlashSpace, flashSpacePtr)

	sunEnabled := int32(0)
	if s.sunCSM != nil && s.sunCSM.IsEnabled() {
		sunEnabled = 1
		for c := range metrics.SunCascadeCount {
			writeMat4Words(buf, pfOffSunSpaces+c*16, s.sunCSM.GetCascadeMatrixPtr(c))
		}
		dvx, dvy, dvz := s.sunCSM.GetDirectionView(s.metrics)
		sr, sg, sb, si := s.sunCSM.GetColorIntensity()
		skyR, skyG, skyB, grR, grG, grB := s.sunCSM.GetSkyAmbient()
		splits := s.sunCSM.GetCascadeSplitsView()
		texelBias := s.sunCSM.GetCascadeDepthBias()

		writeVec4Floats(buf, pfOffSunDirView, dvx, dvy, dvz, 0)
		writeVec4Floats(buf, pfOffSunColorInt, sr, sg, sb, si)
		writeVec4Floats(buf, pfOffSunSkyColor, skyR, skyG, skyB, 0)
		writeVec4Floats(buf, pfOffSunGroundColor, grR, grG, grB, 0)
		writeVec4Floats(buf, pfOffSunSplits, splits[0], splits[1], splits[2], splits[3])
		writeVec4Floats(buf, pfOffSunBias, texelBias[0], texelBias[1], texelBias[2], texelBias[3])
	}

	writeVec4Floats(buf, pfOffScreenTime, float32(fbW), float32(fbH), interval, s.emissiveIntensity)
	writeVec4Floats(buf, pfOffLightingParams, ambient, s.aoFactor, s.envMaxLod, clusterNear)

	enableFlash := int32(0)
	var flashIntensity float32
	if s.flash != nil {
		flashIntensity = float32(s.flash.GetIntensity())
	}
	if flashIntensity > 0.001 && s.flashlightMetrics != nil {
		enableFlash = 1
		dirX, dirY, dirZ := s.flashlightMetrics.ResolveAim(flashX, flashY, flashSensitivity)
		writeVec4Floats(buf, pfOffFlashPosInt, flashX, flashY, 0.0, flashIntensity)
		writeVec4Floats(buf, pfOffFlashDirFall, dirX, dirY, dirZ, float32(s.flash.GetFalloff()))
		writeVec4Floats(buf, pfOffFlashConeCluster, float32(s.flash.GetConeStart()), float32(s.flash.GetConeEnd()), clusterLogScale, 0)
	} else {
		writeVec4Floats(buf, pfOffFlashPosInt, 0, 0, 0, 0)
		writeVec4Floats(buf, pfOffFlashDirFall, 0, 0, -1, 0)
		writeVec4Floats(buf, pfOffFlashConeCluster, 0, 0, clusterLogScale, 0)
	}

	fogEnabled := int32(0)
	if s.atmosphereFog != nil {
		var fogCol, fogSunCol [3]float32
		var fogParams [4]float32
		var fogHeight [3]float32
		fogEnabled, fogCol, fogSunCol, fogParams, fogHeight = s.atmosphereFog.GetShaderParams()
		if fogEnabled == 1 {
			writeVec4Floats(buf, pfOffFogColor, fogCol[0], fogCol[1], fogCol[2], 0)
			writeVec4Floats(buf, pfOffFogSunColor, fogSunCol[0], fogSunCol[1], fogSunCol[2], 0)
			writeVec4Floats(buf, pfOffFogParams, fogParams[0], fogParams[1], fogParams[2], fogParams[3])
			writeVec4Floats(buf, pfOffFogHeightParams, fogHeight[0], fogHeight[1], fogHeight[2], 0)
		}
	}

	writeVec4Ints(buf, pfOffFlags, s.shadows, int32(s.debugLights), enableFlash, numShadowLights)
	writeVec4Ints(buf, pfOffCounts, s.lights.GetActiveLightsCount(), numGlobalLights, sunEnabled, fogEnabled)

	if ubo := s.uboPerFrame[s.frameIdx]; ubo != 0 {
		s.ctx.BindBuffer(api.UNIFORM_BUFFER, ubo)
		s.ctx.BufferSubData(api.UNIFORM_BUFFER, 0, perFrameBlockBytes, s.perFramePtr)
	}
	if numShadowLights > 0 {
		if ubo := s.uboShadowLights[s.frameIdx]; ubo != 0 {
			s.ctx.BindBuffer(api.UNIFORM_BUFFER, ubo)
			s.ctx.BufferSubData(api.UNIFORM_BUFFER, 0, int(numShadowLights)*shadowLightEntryBytes, s.shadowLightsPtr)
		}
	}
	s.ctx.BindBuffer(api.UNIFORM_BUFFER, 0)

	s.perFrameReady = true
	s.perFrameTick = tick
	s.perFrameAmbient = ambient
	s.perFrameFlashX = flashX
	s.perFrameFlashY = flashY
	s.perFrameFlashSens = flashSensitivity
}

func (s *Main) bindZPrepassProgram(zp *zPrepassProgram) {
	s.ctx.UseProgram(zp.prg)
	s.ctx.BindBufferBase(api.UNIFORM_BUFFER, uboBindingPerFrame, s.uboPerFrame[s.frameIdx])
	if zp.locInstanceMode >= 0 {
		s.ctx.Uniform1i(zp.locInstanceMode, buffers.InstanceModeStreaming)
	}
}

func (s *Main) bindForwardPass(p *mainProgram, enableSSAO int32) {
	s.ctx.UseProgram(p.prg)
	s.ctx.BindBufferBase(api.UNIFORM_BUFFER, uboBindingLights, s.lights.GetActiveUBO())
	s.ctx.BindBufferBase(api.UNIFORM_BUFFER, uboBindingPerFrame, s.uboPerFrame[s.frameIdx])
	s.ctx.BindBufferBase(api.UNIFORM_BUFFER, uboBindingShadowLights, s.uboShadowLights[s.frameIdx])
	if loc := p.table[MainLocEnableSSAO]; loc >= 0 {
		s.ctx.Uniform1i(loc, enableSSAO)
	}
	if loc := p.table[MainLocInstanceMode]; loc >= 0 {
		s.ctx.Uniform1i(loc, buffers.InstanceModeStreaming)
	}
}

// Render executes a Z-prepass followed by Early-Z forward opaque shading and conditional alpha-cutout shading.
func (s *Main) Render(dc *draws.DrawCommandsRender, ambient float32, flashX, flashY, flashSensitivity float32) {
	targetFbo := s.post.GetFBO()
	fbW, fbH := s.handler.GetSize()
	tick := textures.GlobalTick()
	interval := float32(tick) * 0.05

	// Target FBO preparation
	s.ctx.BindFramebuffer(api.FRAMEBUFFER, targetFbo)
	s.ctx.Viewport(0, 0, fbW, fbH)
	s.ctx.ClearColor(0.0, 0.0, 0.0, 1.0)
	s.ctx.Clear(api.COLOR_BUFFER_BIT | api.DEPTH_BUFFER_BIT)

	if dc == nil || !dc.HasCommands() {
		s.perFrameReady = false
		return
	}

	clusterTex, clusterNear, clusterLogScale, numGlobalLights := s.lights.PrepareClusters()
	s.uploadPerFrameUBOs(tick, fbW, fbH, interval, ambient, flashX, flashY, flashSensitivity, clusterNear, clusterLogScale, numGlobalLights)

	streamVAO := s.mainVAO[s.frameIdx]
	s.ctx.BindVertexArray(streamVAO)

	// Pass 1: Z-Prepass populates front-most depth before running heavy lighting/shadows.
	if dc.HasSolidPrepass() || dc.HasCutoutPrepass() {
		s.ctx.ColorMask(false, false, false, false)
		s.ctx.DepthMask(true)
		s.ctx.DepthFunc(api.LESS)
		s.ctx.Disable(api.SAMPLE_ALPHA_TO_COVERAGE)
		s.ctx.Disable(api.POLYGON_OFFSET_FILL)

		if dc.HasSolidPrepass() {
			s.bindZPrepassProgram(&s.zPrepassSolid)
			dc.SetPassState(streamVAO, s.zPrepassSolid.locInstanceMode)
			dc.RenderPrepass(false)
		}
		if dc.HasCutoutPrepass() {
			s.bindZPrepassProgram(&s.zPrepassCutout)
			dc.SetPassState(streamVAO, s.zPrepassCutout.locInstanceMode)
			dc.RenderPrepass(true)
		}
		s.ctx.ColorMask(true, true, true, true)
	}

	// Half-res Depth SSAO between Z-Prepass and Early-Z forward shading.
	enableSSAO := int32(0)
	var ssaoTex uint32
	if s.ssao != nil && s.ssao.IsReady() {
		if s.post.GetSamples() > 0 {
			s.post.ResolveMSAA(fbW, fbH, ResolveDepth)
		}
		s.ssao.RenderDepth(s.post.GetDepthBuffer(), fbW, fbH)
		ssaoTex = s.ssao.GetSSAOBlurTexture()
		if ssaoTex != 0 {
			enableSSAO = 1
		}
		s.ctx.BindFramebuffer(api.FRAMEBUFFER, targetFbo)
		s.ctx.Viewport(0, 0, fbW, fbH)
		s.ctx.BindVertexArray(streamVAO)
	} else if s.ssao != nil {
		ssaoTex = s.ssao.GetFallbackTexture()
	}

	// Shadow Texture (Master Shadow Atlas)
	if s.shadows != 0 {
		s.ctx.ActiveTexture(api.TEXTURE0 + api.TexUnitShadowAtlas)
		s.ctx.BindTexture(api.TEXTURE_2D, s.depth.GetShadowAtlasTexture())
	}
	if ssaoTex != 0 {
		s.ctx.ActiveTexture(api.TEXTURE0 + api.TexUnitSSAO)
		s.ctx.BindTexture(api.TEXTURE_2D, ssaoTex)
	}
	if clusterTex != 0 {
		s.ctx.ActiveTexture(api.TEXTURE0 + api.TexUnitClusterData)
		s.ctx.BindTexture(api.TEXTURE_2D, clusterTex)
	}

	// Pass 2a: Solid geometry (no discard -> hardware Early-Z enabled; DepthFunc(EQUAL) on prepassed batches)
	if dc.HasSolid() {
		s.bindForwardPass(&s.solidPrg, enableSSAO)
		dc.SetPassState(streamVAO, s.solidPrg.table[MainLocInstanceMode])
		s.ctx.Disable(api.SAMPLE_ALPHA_TO_COVERAGE)
		dc.RenderShade(false)
	}

	// Pass 2b: Alpha-cutout geometry (discard enabled; Alpha-to-Coverage only when MSAA > 0)
	if dc.HasCutout() {
		s.bindForwardPass(&s.cutoutPrg, enableSSAO)
		dc.SetPassState(streamVAO, s.cutoutPrg.table[MainLocInstanceMode])
		useAlphaToCoverage := s.post.GetSamples() > 0
		if useAlphaToCoverage {
			s.ctx.Enable(api.SAMPLE_ALPHA_TO_COVERAGE)
		}
		dc.RenderShade(true)
		if useAlphaToCoverage {
			s.ctx.Disable(api.SAMPLE_ALPHA_TO_COVERAGE)
		}
	}

	if s.shadows != 0 {
		s.ctx.ActiveTexture(api.TEXTURE0 + api.TexUnitShadowAtlas)
		s.ctx.BindTexture(api.TEXTURE_2D, 0)
	}
	if ssaoTex != 0 {
		s.ctx.ActiveTexture(api.TEXTURE0 + api.TexUnitSSAO)
		s.ctx.BindTexture(api.TEXTURE_2D, 0)
	}
	if clusterTex != 0 {
		s.ctx.ActiveTexture(api.TEXTURE0 + api.TexUnitClusterData)
		s.ctx.BindTexture(api.TEXTURE_2D, 0)
	}

	s.ctx.BindVertexArray(0)
}

// RenderTransparent executes the back-to-front depth-sorted translucent pass over the resolved opaque/liquid scene.
func (s *Main) RenderTransparent(dc *draws.DrawCommandsRender, ambient float32, flashX, flashY, flashSensitivity float32) {
	if dc == nil || !dc.HasCommands() {
		return
	}
	fbW, fbH := s.handler.GetSize()
	tick := textures.GlobalTick()
	interval := float32(tick) * 0.05

	clusterTex, clusterNear, clusterLogScale, numGlobalLights := s.lights.PrepareClusters()
	if s.needsPerFrameUpload(tick, ambient, flashX, flashY, flashSensitivity) {
		s.uploadPerFrameUBOs(tick, fbW, fbH, interval, ambient, flashX, flashY, flashSensitivity, clusterNear, clusterLogScale, numGlobalLights)
	}

	var ssaoFallback uint32
	if s.ssao != nil {
		ssaoFallback = s.ssao.GetFallbackTexture()
	}

	if s.shadows != 0 {
		s.ctx.ActiveTexture(api.TEXTURE0 + api.TexUnitShadowAtlas)
		s.ctx.BindTexture(api.TEXTURE_2D, s.depth.GetShadowAtlasTexture())
	}
	if ssaoFallback != 0 {
		s.ctx.ActiveTexture(api.TEXTURE0 + api.TexUnitSSAO)
		s.ctx.BindTexture(api.TEXTURE_2D, ssaoFallback)
	}
	if clusterTex != 0 {
		s.ctx.ActiveTexture(api.TEXTURE0 + api.TexUnitClusterData)
		s.ctx.BindTexture(api.TEXTURE_2D, clusterTex)
	}

	streamVAO := s.mainVAO[s.frameIdx]
	s.ctx.BindVertexArray(streamVAO)
	s.ctx.Enable(api.BLEND)
	s.ctx.BlendFunc(api.SRC_ALPHA, api.ONE_MINUS_SRC_ALPHA)
	s.ctx.DepthMask(false)
	s.ctx.Enable(api.DEPTH_TEST)
	s.ctx.DepthFunc(api.LEQUAL)
	s.ctx.Disable(api.SAMPLE_ALPHA_TO_COVERAGE)

	s.bindForwardPass(&s.solidPrg, 0)
	dc.SetPassState(streamVAO, s.solidPrg.table[MainLocInstanceMode])
	dc.RenderTransparent()

	if s.shadows != 0 {
		s.ctx.ActiveTexture(api.TEXTURE0 + api.TexUnitShadowAtlas)
		s.ctx.BindTexture(api.TEXTURE_2D, 0)
	}
	if ssaoFallback != 0 {
		s.ctx.ActiveTexture(api.TEXTURE0 + api.TexUnitSSAO)
		s.ctx.BindTexture(api.TEXTURE_2D, 0)
	}
	if clusterTex != 0 {
		s.ctx.ActiveTexture(api.TEXTURE0 + api.TexUnitClusterData)
		s.ctx.BindTexture(api.TEXTURE_2D, 0)
	}

	s.ctx.BindVertexArray(0)
	s.ctx.Disable(api.BLEND)
	s.ctx.DepthMask(true)
	s.ctx.DepthFunc(api.LESS)
}
