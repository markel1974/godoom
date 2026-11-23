package shaders

import (
	"fmt"
	"strings"
	"unsafe"

	"github.com/markel1974/godoom/mr_tech/model"
	"github.com/markel1974/godoom/mr_tech/model/things"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/gpu"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/metrics"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// MainLoc represents an integer-based enumerator used as an identifier for uniform locations within shaders.
type MainLoc int

// mainDoubleBuffer defines the number of buffer sets used for double buffering operations in rendering pipelines.
const (
	mainDoubleBuffer = 2
)

const (
	MainLocView = MainLoc(iota)
	MainLocInvView
	MainLocProjection
	MainLocRoomSpaceMatrix
	MainLocFlashSpaceMatrix
	MainLocTime
	MainLocScreenResolution
	MainLocTexture
	MainLocNormalMap
	MainLocEmissiveMap
	MainLocRoomShadowMap
	MainLocFlashShadowMap
	MainLocShadowAtlas
	MainLocEmissiveIntensity
	MainLocAmbientLight
	MainLocEnableShadows
	MainLocDebugLights
	// MainLocEnableFlashlight Flashlight
	MainLocEnableFlashlight
	MainLocFlashPosView
	MainLocFlashDir
	MainLocFlashIntensityFactor
	MainLocFlashFalloff
	MainLocFlashConeStart
	MainLocFlashConeEnd
	// MainLocNumShadowLights Dynamic Shadow Lights
	MainLocNumShadowLights
	// MainLocNumLights Dynamic UBO Lights
	MainLocNumLights
	MainLocInstanceMode
	MainLocEnvMap
	MainLocLightmaps
	MainLocClusterData
	MainLocClusterNear
	MainLocClusterLogScale
	MainLocNumGlobalLights
	MainLocLast
)

type shadowLightLocs struct {
	posIntensity int32
	dirFalloff   int32
	colorType    int32
	spotParams   int32
	lightSpace   int32
}

type mainProgram struct {
	prg             uint32
	table           [MainLocLast]int32
	shadowLightLocs [metrics.ShadowLightsCount]shadowLightLocs
}

type zPrepassProgram struct {
	prg             uint32
	locView         int32
	locProj         int32
	locTime         int32
	locTex          int32
	locInstanceMode int32
}

type transformedShadowLight struct {
	vx, vy, vz    float32
	intensity     float32
	dvx, dvy, dvz float32
	falloff       float32
	r, g, b       float32
	kind          float32
	outerCutOff   float32
	cutOff        float32
	effRadiusSq   float32
	invScale      float32
	lightSpace    *float32
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
	flash               *things.Flash
	depth               *Depth
	lights              *Lights
	post                *Post
}

// NewMain initializes and returns a new instance of Main with the provided context, stride, and metrics.
func NewMain(stride int32) *Main {
	return &Main{
		stride:      stride,
		shadows:     0,
		debugLights: 0,
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
	s.flash = h.GetFlash()
	s.depth = h.GetDepth()
	s.lights = h.GetLights()
	s.post = h.GetPost()
	s.emissiveIntensity = h.GetEmissiveIntensity()
	s.shadows = 0

	return nil
}

// SetShadowsEnabled enables or disables shadow evaluation in the main shader.
func (s *Main) SetShadowsEnabled(e bool) {
	if e {
		s.shadows = 1
	} else {
		s.shadows = 0
	}
}

// Init initializes the necessary OpenGL resources, such as VAOs, VBOs, and EBOs, and configures the vertex attributes.
func (s *Main) Init() error {
	vboBytesSize := 131072 * int(s.stride)
	eboBytesSize := 262144 * 4

	s.ctx.GenVertexArrays(mainDoubleBuffer, &s.mainVAO[0])
	s.ctx.GenBuffers(mainDoubleBuffer, &s.mainVBO[0])
	s.ctx.GenBuffers(mainDoubleBuffer, &s.mainEBO[0])

	for i := 0; i < mainDoubleBuffer; i++ {
		s.vboBytesCap[i] = vboBytesSize
		s.eboBytesCap[i] = eboBytesSize

		s.ctx.BindVertexArray(s.mainVAO[i])

		s.ctx.BindBuffer(api.ARRAY_BUFFER, s.mainVBO[i])
		s.ctx.BufferData(api.ARRAY_BUFFER, vboBytesSize, nil, api.DYNAMIC_DRAW)

		s.ctx.BindBuffer(api.ELEMENT_ARRAY_BUFFER, s.mainEBO[i])
		s.ctx.BufferData(api.ELEMENT_ARRAY_BUFFER, eboBytesSize, nil, api.DYNAMIC_DRAW)

		gpu.ConfigureVertexAttribs(s.ctx, s.stride, true)
	}

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

func (s *Main) GetLocView() int32 { return s.solidPrg.table[MainLocView] }

func (s *Main) GetLocProj() int32 { return s.solidPrg.table[MainLocProjection] }

// SetupSamplers initializes sampler uniforms for opaque, cutout, and Z-prepass shader programs.
func (s *Main) SetupSamplers() error {
	diffuseUnits := []int32{api.TexUnitDiffuseBase, api.TexUnitDiffuseBase + 1, api.TexUnitDiffuseBase + 2, api.TexUnitDiffuseBase + 3}
	normalUnits := []int32{api.TexUnitNormalBase, api.TexUnitNormalBase + 1, api.TexUnitNormalBase + 2, api.TexUnitNormalBase + 3}
	emissiveUnits := []int32{api.TexUnitEmissiveBase, api.TexUnitEmissiveBase + 1, api.TexUnitEmissiveBase + 2, api.TexUnitEmissiveBase + 3}

	programs := [2]*mainProgram{&s.solidPrg, &s.cutoutPrg}
	for _, p := range programs {
		s.ctx.UseProgram(p.prg)
		s.ctx.Uniform1iv(p.table[MainLocTexture], 4, &diffuseUnits[0])
		s.ctx.Uniform1iv(p.table[MainLocNormalMap], 4, &normalUnits[0])
		s.ctx.Uniform1iv(p.table[MainLocEmissiveMap], 4, &emissiveUnits[0])
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
	}

	if s.zPrepassCutout.prg != 0 && s.zPrepassCutout.locTex >= 0 {
		s.ctx.UseProgram(s.zPrepassCutout.prg)
		s.ctx.Uniform1iv(s.zPrepassCutout.locTex, 4, &diffuseUnits[0])
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

	if err := s.compileForwardProgram(&s.solidPrg, "main_opaque", vertId, vertexSrc, fragOpaqueId, fragmentOpaqueSrc); err != nil {
		return err
	}
	cutoutSrc := injectShaderDefine(string(fragmentOpaqueSrc), "#define ALPHA_CUTOUT 1")
	if err := s.compileForwardProgram(&s.cutoutPrg, "main_cutout", vertId, vertexSrc, fragOpaqueId, []byte(cutoutSrc)); err != nil {
		return err
	}
	if err := s.compileZPrepassProgram(&s.zPrepassSolid, "zprepass_solid", vertId, vertexSrc, fragZSolidId, zSolidSrc, false); err != nil {
		return err
	}
	if err := s.compileZPrepassProgram(&s.zPrepassCutout, "zprepass_cutout", vertId, vertexSrc, fragZCutoutId, zCutoutSrc, true); err != nil {
		return err
	}
	return nil
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
	dst.table[MainLocView] = s.ctx.GetUniformLocation(prg, s.ctx.Str("u_view\x00"))
	dst.table[MainLocInvView] = s.ctx.GetUniformLocation(prg, s.ctx.Str("u_invView\x00"))
	dst.table[MainLocProjection] = s.ctx.GetUniformLocation(prg, s.ctx.Str("u_projection\x00"))
	dst.table[MainLocRoomSpaceMatrix] = s.ctx.GetUniformLocation(prg, s.ctx.Str("u_roomSpaceMatrix\x00"))
	dst.table[MainLocFlashSpaceMatrix] = s.ctx.GetUniformLocation(prg, s.ctx.Str("u_flashSpaceMatrix\x00"))
	dst.table[MainLocTime] = s.ctx.GetUniformLocation(prg, s.ctx.Str("u_time\x00"))
	dst.table[MainLocScreenResolution] = s.ctx.GetUniformLocation(prg, s.ctx.Str("u_screenResolution\x00"))
	dst.table[MainLocTexture] = s.ctx.GetUniformLocation(prg, s.ctx.Str("u_texture\x00"))
	dst.table[MainLocNormalMap] = s.ctx.GetUniformLocation(prg, s.ctx.Str("u_normalMap\x00"))
	dst.table[MainLocEmissiveMap] = s.ctx.GetUniformLocation(prg, s.ctx.Str("u_emissiveMap\x00"))
	dst.table[MainLocRoomShadowMap] = -1
	dst.table[MainLocFlashShadowMap] = -1
	dst.table[MainLocShadowAtlas] = s.ctx.GetUniformLocation(prg, s.ctx.Str("u_shadowAtlas\x00"))
	dst.table[MainLocEmissiveIntensity] = s.ctx.GetUniformLocation(prg, s.ctx.Str("u_emissiveIntensity\x00"))
	dst.table[MainLocAmbientLight] = s.ctx.GetUniformLocation(prg, s.ctx.Str("u_ambient_light\x00"))
	dst.table[MainLocEnableShadows] = s.ctx.GetUniformLocation(prg, s.ctx.Str("u_enableShadows\x00"))
	dst.table[MainLocDebugLights] = s.ctx.GetUniformLocation(prg, s.ctx.Str("u_debugLights\x00"))

	dst.table[MainLocEnableFlashlight] = s.ctx.GetUniformLocation(prg, s.ctx.Str("u_enableFlashlight\x00"))
	dst.table[MainLocFlashPosView] = s.ctx.GetUniformLocation(prg, s.ctx.Str("u_flashPosView\x00"))
	dst.table[MainLocFlashDir] = s.ctx.GetUniformLocation(prg, s.ctx.Str("u_flashDir\x00"))
	dst.table[MainLocFlashIntensityFactor] = s.ctx.GetUniformLocation(prg, s.ctx.Str("u_flashIntensityFactor\x00"))
	dst.table[MainLocFlashFalloff] = s.ctx.GetUniformLocation(prg, s.ctx.Str("u_flashFalloff\x00"))
	dst.table[MainLocFlashConeStart] = s.ctx.GetUniformLocation(prg, s.ctx.Str("u_flashConeStart\x00"))
	dst.table[MainLocFlashConeEnd] = s.ctx.GetUniformLocation(prg, s.ctx.Str("u_flashConeEnd\x00"))

	dst.table[MainLocNumShadowLights] = s.ctx.GetUniformLocation(prg, s.ctx.Str("u_numShadowLights\x00"))
	for i := 0; i < metrics.ShadowLightsCount; i++ {
		dst.shadowLightLocs[i].posIntensity = s.ctx.GetUniformLocation(prg, s.ctx.Str(fmt.Sprintf("u_shadowLights[%d].pos_intensity\x00", i)))
		dst.shadowLightLocs[i].dirFalloff = s.ctx.GetUniformLocation(prg, s.ctx.Str(fmt.Sprintf("u_shadowLights[%d].dir_falloff\x00", i)))
		dst.shadowLightLocs[i].colorType = s.ctx.GetUniformLocation(prg, s.ctx.Str(fmt.Sprintf("u_shadowLights[%d].color_type\x00", i)))
		dst.shadowLightLocs[i].spotParams = s.ctx.GetUniformLocation(prg, s.ctx.Str(fmt.Sprintf("u_shadowLights[%d].spot_params\x00", i)))
		dst.shadowLightLocs[i].lightSpace = s.ctx.GetUniformLocation(prg, s.ctx.Str(fmt.Sprintf("u_shadowLights[%d].lightSpace\x00", i)))
	}

	dst.table[MainLocNumLights] = s.ctx.GetUniformLocation(prg, s.ctx.Str("u_numLights\x00"))
	dst.table[MainLocInstanceMode] = s.ctx.GetUniformLocation(prg, s.ctx.Str("u_instanceMode\x00"))
	dst.table[MainLocEnvMap] = s.ctx.GetUniformLocation(prg, s.ctx.Str("u_envMap\x00"))
	dst.table[MainLocLightmaps] = s.ctx.GetUniformLocation(prg, s.ctx.Str("u_lightmaps\x00"))
	dst.table[MainLocClusterData] = s.ctx.GetUniformLocation(prg, s.ctx.Str("u_clusterData\x00"))
	dst.table[MainLocClusterNear] = s.ctx.GetUniformLocation(prg, s.ctx.Str("u_clusterNear\x00"))
	dst.table[MainLocClusterLogScale] = s.ctx.GetUniformLocation(prg, s.ctx.Str("u_clusterLogScale\x00"))
	dst.table[MainLocNumGlobalLights] = s.ctx.GetUniformLocation(prg, s.ctx.Str("u_numGlobalLights\x00"))

	blockIndex := s.ctx.GetUniformBlockIndex(prg, s.ctx.Str("LightsBlock\x00"))
	if blockIndex != api.INVALID_INDEX {
		s.ctx.UniformBlockBinding(prg, blockIndex, 0)
	}
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
	dst.locView = s.ctx.GetUniformLocation(dst.prg, s.ctx.Str("u_view\x00"))
	dst.locProj = s.ctx.GetUniformLocation(dst.prg, s.ctx.Str("u_projection\x00"))
	dst.locTime = s.ctx.GetUniformLocation(dst.prg, s.ctx.Str("u_time\x00"))
	dst.locInstanceMode = s.ctx.GetUniformLocation(dst.prg, s.ctx.Str("u_instanceMode\x00"))
	dst.locTex = -1
	if withTexture {
		dst.locTex = s.ctx.GetUniformLocation(dst.prg, s.ctx.Str("u_texture\x00"))
	}
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

func (s *Main) bindZPrepassProgram(zp *zPrepassProgram, view, proj *float32, interval float32) {
	s.ctx.UseProgram(zp.prg)
	s.ctx.UniformMatrix4fv(zp.locView, 1, false, view)
	s.ctx.UniformMatrix4fv(zp.locProj, 1, false, proj)
	s.ctx.Uniform1f(zp.locTime, interval)
	if zp.locInstanceMode >= 0 {
		s.ctx.Uniform1i(zp.locInstanceMode, gpu.InstanceModeStreaming)
	}
}

func (s *Main) prepareShadowLights(transformed *[metrics.ShadowLightsCount]transformedShadowLight) int {
	numShadowLights := int(s.shadowLightsMetrics.GetLightCount())
	if numShadowLights > metrics.ShadowLightsCount {
		numShadowLights = metrics.ShadowLightsCount
	}
	for i := 0; i < numShadowLights; i++ {
		l, lMatrix := s.shadowLightsMetrics.GetLight(i)
		vx, vy, vz := s.metrics.TransformPointToView(l.X, l.Y, l.Z)
		dvx, dvy, dvz := s.metrics.TransformDirectionNormalizedToView(l.DirX, l.DirY, l.DirZ)
		effRadiusSq, invScale := metrics.ComputeLightAttenuation(l.Intensity, l.Falloff)
		transformed[i] = transformedShadowLight{
			vx: vx, vy: vy, vz: vz,
			intensity: l.Intensity,
			dvx:       dvx, dvy: dvy, dvz: dvz,
			falloff: l.Falloff,
			r:       l.R, g: l.G, b: l.B,
			kind:        l.Kind,
			outerCutOff: l.OuterCutOff,
			cutOff:      l.CutOff,
			effRadiusSq: effRadiusSq,
			invScale:    invScale,
			lightSpace:  lMatrix,
		}
	}
	return numShadowLights
}

func (s *Main) bindForwardUniforms(
	p *mainProgram,
	view, invView, proj, roomSpace, flashSpace *float32,
	fbW, fbH int32,
	interval, ambient, flashX, flashY, flashSensitivity float32,
	numShadowLights int,
	shadowLights *[metrics.ShadowLightsCount]transformedShadowLight,
	clusterNear, clusterLogScale float32,
	numGlobalLights int32,
) {
	s.ctx.UseProgram(p.prg)

	// Matrices
	s.ctx.UniformMatrix4fv(p.table[MainLocView], 1, false, view)
	s.ctx.UniformMatrix4fv(p.table[MainLocInvView], 1, false, invView)
	s.ctx.UniformMatrix4fv(p.table[MainLocProjection], 1, false, proj)
	s.ctx.UniformMatrix4fv(p.table[MainLocRoomSpaceMatrix], 1, false, roomSpace)
	s.ctx.UniformMatrix4fv(p.table[MainLocFlashSpaceMatrix], 1, false, flashSpace)
	s.ctx.Uniform1f(p.table[MainLocTime], interval)

	// Environment & Material
	s.ctx.Uniform2f(p.table[MainLocScreenResolution], float32(fbW), float32(fbH))
	s.ctx.Uniform1f(p.table[MainLocEmissiveIntensity], s.emissiveIntensity)
	s.ctx.Uniform1f(p.table[MainLocAmbientLight], ambient)
	s.ctx.Uniform1i(p.table[MainLocEnableShadows], s.shadows)
	s.ctx.Uniform1i(p.table[MainLocDebugLights], int32(s.debugLights))

	// Flashlight
	flashIntensity := float32(s.flash.GetIntensity())
	if flashIntensity > 0.001 {
		s.ctx.Uniform1i(p.table[MainLocEnableFlashlight], 1)
		s.ctx.Uniform3f(p.table[MainLocFlashPosView], flashX, flashY, 0.0)
		dirX, dirY, dirZ := s.flashlightMetrics.ResolveAim(flashX, flashY, flashSensitivity)
		s.ctx.Uniform3f(p.table[MainLocFlashDir], dirX, dirY, dirZ)
		s.ctx.Uniform1f(p.table[MainLocFlashIntensityFactor], flashIntensity)
		s.ctx.Uniform1f(p.table[MainLocFlashFalloff], float32(s.flash.GetFalloff()))
		s.ctx.Uniform1f(p.table[MainLocFlashConeStart], float32(s.flash.GetConeStart()))
		s.ctx.Uniform1f(p.table[MainLocFlashConeEnd], float32(s.flash.GetConeEnd()))
	} else {
		s.ctx.Uniform1i(p.table[MainLocEnableFlashlight], 0)
	}

	// Dynamic Shadow Lights (already view-space transformed on CPU)
	s.ctx.Uniform1i(p.table[MainLocNumShadowLights], int32(numShadowLights))
	for i := 0; i < numShadowLights; i++ {
		tl := &shadowLights[i]
		loc := &p.shadowLightLocs[i]
		s.ctx.Uniform4f(loc.posIntensity, tl.vx, tl.vy, tl.vz, tl.intensity)
		s.ctx.Uniform4f(loc.dirFalloff, tl.dvx, tl.dvy, tl.dvz, tl.falloff)
		s.ctx.Uniform4f(loc.colorType, tl.r, tl.g, tl.b, tl.kind)
		s.ctx.Uniform4f(loc.spotParams, tl.outerCutOff, tl.cutOff, tl.effRadiusSq, tl.invScale)
		s.ctx.UniformMatrix4fv(loc.lightSpace, 1, false, tl.lightSpace)
	}

	// Dynamic UBO Lights & Clustered Lookup
	s.ctx.Uniform1i(p.table[MainLocNumLights], s.lights.GetActiveLightsCount())
	if loc := p.table[MainLocClusterNear]; loc >= 0 {
		s.ctx.Uniform1f(loc, clusterNear)
	}
	if loc := p.table[MainLocClusterLogScale]; loc >= 0 {
		s.ctx.Uniform1f(loc, clusterLogScale)
	}
	if loc := p.table[MainLocNumGlobalLights]; loc >= 0 {
		s.ctx.Uniform1i(loc, numGlobalLights)
	}
	if loc := p.table[MainLocInstanceMode]; loc >= 0 {
		s.ctx.Uniform1i(loc, gpu.InstanceModeStreaming)
	}
	s.ctx.BindBufferBase(api.UNIFORM_BUFFER, 0, s.lights.GetActiveUBO())
}

// Render executes a Z-prepass followed by Early-Z forward opaque shading and conditional alpha-cutout shading.
func (s *Main) Render(dc *gpu.DrawCommandsRender, ambient float32, flashX, flashY, flashSensitivity float32) {
	targetFbo := s.post.GetFBO()
	view := s.metrics.GetViewPtr()
	invView := s.metrics.GetInvViewPtr()
	proj := s.metrics.GetProjPtr()
	roomSpace := s.metrics.GetRoomSpacePtr()
	flashSpace := s.flashlightMetrics.GetShadowSpacePtr()
	fbW, fbH := s.handler.GetSize()
	interval := float32(textures.GlobalTick()) * 0.05

	// Target FBO preparation
	s.ctx.BindFramebuffer(api.FRAMEBUFFER, targetFbo)
	s.ctx.Viewport(0, 0, fbW, fbH)
	s.ctx.ClearColor(0.0, 0.0, 0.0, 1.0)
	s.ctx.Clear(api.COLOR_BUFFER_BIT | api.DEPTH_BUFFER_BIT)

	if dc == nil || !dc.HasCommands() {
		return
	}

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
			s.bindZPrepassProgram(&s.zPrepassSolid, view, proj, interval)
			dc.SetPassState(streamVAO, s.zPrepassSolid.locInstanceMode)
			dc.RenderPrepass(false)
		}
		if dc.HasCutoutPrepass() {
			s.bindZPrepassProgram(&s.zPrepassCutout, view, proj, interval)
			dc.SetPassState(streamVAO, s.zPrepassCutout.locInstanceMode)
			dc.RenderPrepass(true)
		}
		s.ctx.ColorMask(true, true, true, true)
	}

	var transformed [metrics.ShadowLightsCount]transformedShadowLight
	numShadowLights := s.prepareShadowLights(&transformed)
	clusterTex, clusterNear, clusterLogScale, numGlobalLights := s.lights.PrepareClusters()

	// Shadow Texture (Master Shadow Atlas)
	if s.shadows != 0 {
		s.ctx.ActiveTexture(api.TEXTURE0 + api.TexUnitShadowAtlas)
		s.ctx.BindTexture(api.TEXTURE_2D, s.depth.GetShadowAtlasTexture())
	}
	if clusterTex != 0 {
		s.ctx.ActiveTexture(api.TEXTURE0 + api.TexUnitClusterData)
		s.ctx.BindTexture(api.TEXTURE_2D, clusterTex)
	}

	// Pass 2a: Solid geometry (no discard -> hardware Early-Z enabled; DepthFunc(EQUAL) on prepassed batches)
	if dc.HasSolid() {
		s.bindForwardUniforms(
			&s.solidPrg,
			view, invView, proj, roomSpace, flashSpace,
			fbW, fbH, interval, ambient, flashX, flashY, flashSensitivity,
			numShadowLights, &transformed,
			clusterNear, clusterLogScale, numGlobalLights,
		)
		dc.SetPassState(streamVAO, s.solidPrg.table[MainLocInstanceMode])
		s.ctx.Disable(api.SAMPLE_ALPHA_TO_COVERAGE)
		dc.RenderShade(false)
	}

	// Pass 2b: Alpha-cutout geometry (discard enabled; Alpha-to-Coverage only when MSAA > 0)
	if dc.HasCutout() {
		s.bindForwardUniforms(
			&s.cutoutPrg,
			view, invView, proj, roomSpace, flashSpace,
			fbW, fbH, interval, ambient, flashX, flashY, flashSensitivity,
			numShadowLights, &transformed,
			clusterNear, clusterLogScale, numGlobalLights,
		)
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
	if clusterTex != 0 {
		s.ctx.ActiveTexture(api.TEXTURE0 + api.TexUnitClusterData)
		s.ctx.BindTexture(api.TEXTURE_2D, 0)
	}

	s.ctx.BindVertexArray(0)
}

// RenderTransparent executes the back-to-front depth-sorted translucent pass over the resolved opaque/liquid scene.
func (s *Main) RenderTransparent(dc *gpu.DrawCommandsRender, ambient float32, flashX, flashY, flashSensitivity float32) {
	if dc == nil || !dc.HasCommands() {
		return
	}
	view := s.metrics.GetViewPtr()
	invView := s.metrics.GetInvViewPtr()
	proj := s.metrics.GetProjPtr()
	roomSpace := s.metrics.GetRoomSpacePtr()
	flashSpace := s.flashlightMetrics.GetShadowSpacePtr()
	fbW, fbH := s.handler.GetSize()
	interval := float32(textures.GlobalTick()) * 0.05

	var transformed [metrics.ShadowLightsCount]transformedShadowLight
	numShadowLights := s.prepareShadowLights(&transformed)
	clusterTex, clusterNear, clusterLogScale, numGlobalLights := s.lights.PrepareClusters()

	if s.shadows != 0 {
		s.ctx.ActiveTexture(api.TEXTURE0 + api.TexUnitShadowAtlas)
		s.ctx.BindTexture(api.TEXTURE_2D, s.depth.GetShadowAtlasTexture())
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

	s.bindForwardUniforms(
		&s.solidPrg,
		view, invView, proj, roomSpace, flashSpace,
		fbW, fbH, interval, ambient, flashX, flashY, flashSensitivity,
		numShadowLights, &transformed,
		clusterNear, clusterLogScale, numGlobalLights,
	)
	dc.SetPassState(streamVAO, s.solidPrg.table[MainLocInstanceMode])
	dc.RenderTransparent()

	if s.shadows != 0 {
		s.ctx.ActiveTexture(api.TEXTURE0 + api.TexUnitShadowAtlas)
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
