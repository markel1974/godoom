package shaders

import (
	"fmt"

	"github.com/markel1974/godoom/mr_tech/model"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
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
	MainLocShininessWall
	MainLocShininessFloor
	MainLocSpecBoostWall
	MainLocSpecBoostFloor
	MainLocDebugLights

	// Flashlight
	MainLocEnableFlashlight
	MainLocFlashPosView
	MainLocFlashDir
	MainLocFlashIntensityFactor
	MainLocFlashFalloff
	MainLocFlashConeStart
	MainLocFlashConeEnd

	// Dynamic Shadow Lights
	MainLocNumShadowLights

	// Dynamic UBO Lights
	MainLocNumLights

	MainLocLast
)

type shadowLightLocs struct {
	posIntensity int32
	dirFalloff   int32
	colorType    int32
	spotParams   int32
	lightSpace   int32
}

// Main represents the central structure for managing rendering states and operations.
type Main struct {
	ctx                 api.IContext
	prgOpaque           uint32
	tableOpaque         [MainLocLast]int32
	shadowLightLocs     [8]shadowLightLocs
	mainVAO             [mainDoubleBuffer]uint32
	mainVBO             [mainDoubleBuffer]uint32
	mainEBO             [mainDoubleBuffer]uint32
	vboBytesCap         [mainDoubleBuffer]int
	eboBytesCap         [mainDoubleBuffer]int
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
	flash               *model.Flash
	depth               *Depth
	lights              *Lights
	post                *Post
}

// NewMain initializes and returns a new instance of Main with the provided context, stride, and metrics.
func NewMain(stride int32) *Main {
	m := &Main{
		prgOpaque:   0,
		stride:      stride,
		shadows:     0,
		debugLights: 0,
	}
	return m
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

// EnableShadows enables or disables shadow evaluation in the main shader.
func (s *Main) EnableShadows(e bool) {
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

		strideBytes := s.stride

		// Location 0: aPos (x, y, z) - 3 float
		s.ctx.VertexAttribPointer(0, 3, api.FLOAT, false, strideBytes, s.ctx.PtrOffset(0))
		s.ctx.EnableVertexAttribArray(0)

		// Location 1: aTexCoords (u, v, layer) - 3 float
		s.ctx.VertexAttribPointer(1, 3, api.FLOAT, false, strideBytes, s.ctx.PtrOffset(3*4))
		s.ctx.EnableVertexAttribArray(1)

		// Location 2: aOrigin (worldX, worldY, worldZ) - 3 float
		s.ctx.VertexAttribPointer(2, 3, api.FLOAT, false, strideBytes, s.ctx.PtrOffset(6*4))
		s.ctx.EnableVertexAttribArray(2)

		// Location 3: aRenderMode (flag) - 1 float
		s.ctx.VertexAttribPointer(3, 1, api.FLOAT, false, strideBytes, s.ctx.PtrOffset(9*4))
		s.ctx.EnableVertexAttribArray(3)

		// Location 4: nextPos (x, y, z) - 3 float
		s.ctx.VertexAttribPointer(4, 3, api.FLOAT, false, strideBytes, s.ctx.PtrOffset(10*4))
		s.ctx.EnableVertexAttribArray(4)

		// Location 5: aLerp (t) - 1 float
		s.ctx.VertexAttribPointer(5, 1, api.FLOAT, false, strideBytes, s.ctx.PtrOffset(13*4))
		s.ctx.EnableVertexAttribArray(5)

		// Location 6: aYaw (angle) - 1 float
		s.ctx.VertexAttribPointer(6, 1, api.FLOAT, false, strideBytes, s.ctx.PtrOffset(14*4))
		s.ctx.EnableVertexAttribArray(6)
	}

	s.ctx.Enable(api.DEPTH_TEST)
	s.ctx.DepthFunc(api.LEQUAL)
	return nil
}

// GetProgramOpaque returns the program ID used for rendering opaque objects.
func (s *Main) GetProgramOpaque() uint32 {
	return s.prgOpaque
}

// GetUniformOpaque retrieves the uniform location for the opaque shader program using the given MainLoc identifier.
func (s *Main) GetUniformOpaque(id MainLoc) int32 {
	return s.tableOpaque[id]
}

// GetVAO returns the Vertex Array Object (VAO) associated with the current frame index.
func (s *Main) GetVAO() uint32 {
	return s.mainVAO[s.frameIdx]
}

func (s *Main) GetLocView() int32 { return s.tableOpaque[MainLocView] }

func (s *Main) GetLocProj() int32 { return s.tableOpaque[MainLocProjection] }

// SetupSamplers initializes sampler uniforms for opaque forward shader.
func (s *Main) SetupSamplers() error {
	diffuseUnits := []int32{api.TexUnitDiffuseBase, api.TexUnitDiffuseBase + 1, api.TexUnitDiffuseBase + 2, api.TexUnitDiffuseBase + 3}
	normalUnits := []int32{api.TexUnitNormalBase, api.TexUnitNormalBase + 1, api.TexUnitNormalBase + 2, api.TexUnitNormalBase + 3}
	emissiveUnits := []int32{api.TexUnitEmissiveBase, api.TexUnitEmissiveBase + 1, api.TexUnitEmissiveBase + 2, api.TexUnitEmissiveBase + 3}

	s.ctx.UseProgram(s.prgOpaque)
	s.ctx.Uniform1iv(s.GetUniformOpaque(MainLocTexture), 4, &diffuseUnits[0])
	s.ctx.Uniform1iv(s.GetUniformOpaque(MainLocNormalMap), 4, &normalUnits[0])
	s.ctx.Uniform1iv(s.GetUniformOpaque(MainLocEmissiveMap), 4, &emissiveUnits[0])
	s.ctx.Uniform1i(s.GetUniformOpaque(MainLocShadowAtlas), api.TexUnitShadowAtlas)

	return nil
}

// Compile initializes and compiles shader programs for opaque forward rendering using provided asset sources.
func (s *Main) Compile(a IAssets) error {
	const vertId = "main.vert"
	const fragOpaqueId = "main_opaque.frag"

	vertexSrc, fragmentOpaqueSrc, err := a.ReadMulti(vertId, fragOpaqueId)
	if err != nil {
		return err
	}

	vertexShader, err := ShaderCompile(s.ctx, vertId, string(vertexSrc), api.VERTEX_SHADER)
	if err != nil {
		return err
	}

	// Compile Opaque Program
	fragOpaqueShader, err := ShaderCompile(s.ctx, fragOpaqueId, string(fragmentOpaqueSrc), api.FRAGMENT_SHADER)
	if err != nil {
		s.ctx.DeleteShader(vertexShader)
		return err
	}
	s.prgOpaque, err = ShaderCreateProgram(s.ctx, "main_opaque", vertexShader, fragOpaqueShader)
	if err != nil {
		return err
	}

	// Setup Uniforms Opaque
	s.tableOpaque[MainLocView] = s.ctx.GetUniformLocation(s.prgOpaque, s.ctx.Str("u_view\x00"))
	s.tableOpaque[MainLocInvView] = s.ctx.GetUniformLocation(s.prgOpaque, s.ctx.Str("u_invView\x00"))
	s.tableOpaque[MainLocProjection] = s.ctx.GetUniformLocation(s.prgOpaque, s.ctx.Str("u_projection\x00"))
	s.tableOpaque[MainLocRoomSpaceMatrix] = s.ctx.GetUniformLocation(s.prgOpaque, s.ctx.Str("u_roomSpaceMatrix\x00"))
	s.tableOpaque[MainLocFlashSpaceMatrix] = s.ctx.GetUniformLocation(s.prgOpaque, s.ctx.Str("u_flashSpaceMatrix\x00"))
	s.tableOpaque[MainLocTime] = s.ctx.GetUniformLocation(s.prgOpaque, s.ctx.Str("u_time\x00"))
	s.tableOpaque[MainLocScreenResolution] = s.ctx.GetUniformLocation(s.prgOpaque, s.ctx.Str("u_screenResolution\x00"))
	s.tableOpaque[MainLocTexture] = s.ctx.GetUniformLocation(s.prgOpaque, s.ctx.Str("u_texture\x00"))
	s.tableOpaque[MainLocNormalMap] = s.ctx.GetUniformLocation(s.prgOpaque, s.ctx.Str("u_normalMap\x00"))
	s.tableOpaque[MainLocEmissiveMap] = s.ctx.GetUniformLocation(s.prgOpaque, s.ctx.Str("u_emissiveMap\x00"))
	s.tableOpaque[MainLocRoomShadowMap] = -1
	s.tableOpaque[MainLocFlashShadowMap] = -1
	s.tableOpaque[MainLocShadowAtlas] = s.ctx.GetUniformLocation(s.prgOpaque, s.ctx.Str("u_shadowAtlas\x00"))
	s.tableOpaque[MainLocEmissiveIntensity] = s.ctx.GetUniformLocation(s.prgOpaque, s.ctx.Str("u_emissiveIntensity\x00"))
	s.tableOpaque[MainLocAmbientLight] = s.ctx.GetUniformLocation(s.prgOpaque, s.ctx.Str("u_ambient_light\x00"))
	s.tableOpaque[MainLocEnableShadows] = s.ctx.GetUniformLocation(s.prgOpaque, s.ctx.Str("u_enableShadows\x00"))
	s.tableOpaque[MainLocShininessWall] = s.ctx.GetUniformLocation(s.prgOpaque, s.ctx.Str("u_shininessWall\x00"))
	s.tableOpaque[MainLocShininessFloor] = s.ctx.GetUniformLocation(s.prgOpaque, s.ctx.Str("u_shininessFloor\x00"))
	s.tableOpaque[MainLocSpecBoostWall] = s.ctx.GetUniformLocation(s.prgOpaque, s.ctx.Str("u_specBoostWall\x00"))
	s.tableOpaque[MainLocSpecBoostFloor] = s.ctx.GetUniformLocation(s.prgOpaque, s.ctx.Str("u_specBoostFloor\x00"))
	s.tableOpaque[MainLocDebugLights] = s.ctx.GetUniformLocation(s.prgOpaque, s.ctx.Str("u_debugLights\x00"))

	// Flashlight
	s.tableOpaque[MainLocEnableFlashlight] = s.ctx.GetUniformLocation(s.prgOpaque, s.ctx.Str("u_enableFlashlight\x00"))
	s.tableOpaque[MainLocFlashPosView] = s.ctx.GetUniformLocation(s.prgOpaque, s.ctx.Str("u_flashPosView\x00"))
	s.tableOpaque[MainLocFlashDir] = s.ctx.GetUniformLocation(s.prgOpaque, s.ctx.Str("u_flashDir\x00"))
	s.tableOpaque[MainLocFlashIntensityFactor] = s.ctx.GetUniformLocation(s.prgOpaque, s.ctx.Str("u_flashIntensityFactor\x00"))
	s.tableOpaque[MainLocFlashFalloff] = s.ctx.GetUniformLocation(s.prgOpaque, s.ctx.Str("u_flashFalloff\x00"))
	s.tableOpaque[MainLocFlashConeStart] = s.ctx.GetUniformLocation(s.prgOpaque, s.ctx.Str("u_flashConeStart\x00"))
	s.tableOpaque[MainLocFlashConeEnd] = s.ctx.GetUniformLocation(s.prgOpaque, s.ctx.Str("u_flashConeEnd\x00"))

	// Dynamic Shadow Lights
	s.tableOpaque[MainLocNumShadowLights] = s.ctx.GetUniformLocation(s.prgOpaque, s.ctx.Str("u_numShadowLights\x00"))
	for i := 0; i < 8; i++ {
		s.shadowLightLocs[i].posIntensity = s.ctx.GetUniformLocation(s.prgOpaque, s.ctx.Str(fmt.Sprintf("u_shadowLights[%d].pos_intensity\x00", i)))
		s.shadowLightLocs[i].dirFalloff = s.ctx.GetUniformLocation(s.prgOpaque, s.ctx.Str(fmt.Sprintf("u_shadowLights[%d].dir_falloff\x00", i)))
		s.shadowLightLocs[i].colorType = s.ctx.GetUniformLocation(s.prgOpaque, s.ctx.Str(fmt.Sprintf("u_shadowLights[%d].color_type\x00", i)))
		s.shadowLightLocs[i].spotParams = s.ctx.GetUniformLocation(s.prgOpaque, s.ctx.Str(fmt.Sprintf("u_shadowLights[%d].spot_params\x00", i)))
		s.shadowLightLocs[i].lightSpace = s.ctx.GetUniformLocation(s.prgOpaque, s.ctx.Str(fmt.Sprintf("u_shadowLights[%d].lightSpace\x00", i)))
	}

	// Dynamic UBO Lights
	s.tableOpaque[MainLocNumLights] = s.ctx.GetUniformLocation(s.prgOpaque, s.ctx.Str("u_numLights\x00"))

	blockIndex := s.ctx.GetUniformBlockIndex(s.prgOpaque, s.ctx.Str("LightsBlock\x00"))
	if blockIndex != api.INVALID_INDEX {
		s.ctx.UniformBlockBinding(s.prgOpaque, blockIndex, 0)
	}

	return nil
}

// Prepare configures the rendering pipeline, updates buffer data, and manages double buffering for rendering operations.
func (s *Main) Prepare(vertices []float32, verticesLen int32, indices []uint32, indicesLen int32) {
	fbW, fbH := s.handler.GetSize()
	s.ctx.Viewport(0, 0, fbW, fbH)
	s.ctx.ClearColor(0.0, 0.0, 0.0, 1.0)
	s.ctx.Clear(api.COLOR_BUFFER_BIT | api.DEPTH_BUFFER_BIT)

	s.frameIdx = (s.frameIdx + 1) % mainDoubleBuffer

	vTotal := int(verticesLen) * 4
	iTotal := int(indicesLen) * 4

	s.ctx.BindBuffer(api.ARRAY_BUFFER, s.mainVBO[s.frameIdx])
	if vTotal > s.vboBytesCap[s.frameIdx] {
		newCap := vTotal * 2
		s.ctx.BufferData(api.ARRAY_BUFFER, newCap, nil, api.DYNAMIC_DRAW)
		s.vboBytesCap[s.frameIdx] = newCap
	}
	s.ctx.BufferSubData(api.ARRAY_BUFFER, 0, vTotal, s.ctx.Ptr(vertices))

	s.ctx.BindBuffer(api.ELEMENT_ARRAY_BUFFER, s.mainEBO[s.frameIdx])
	if iTotal > s.eboBytesCap[s.frameIdx] {
		newCap := iTotal * 2
		s.ctx.BufferData(api.ELEMENT_ARRAY_BUFFER, newCap, nil, api.DYNAMIC_DRAW)
		s.eboBytesCap[s.frameIdx] = newCap
	}
	s.ctx.BufferSubData(api.ELEMENT_ARRAY_BUFFER, 0, iTotal, s.ctx.Ptr(indices))
}

// Render executes the single-pass forward rendering pipeline, combining all lights and shadows directly onto geometry.
func (s *Main) Render(renderGeometry func(), ambient float32, flashX, flashY, flashSensitivity float32) {
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

	s.ctx.UseProgram(s.GetProgramOpaque())

	// Matrices
	s.ctx.UniformMatrix4fv(s.GetUniformOpaque(MainLocView), 1, false, view)
	s.ctx.UniformMatrix4fv(s.GetUniformOpaque(MainLocInvView), 1, false, invView)
	s.ctx.UniformMatrix4fv(s.GetUniformOpaque(MainLocProjection), 1, false, proj)
	s.ctx.UniformMatrix4fv(s.GetUniformOpaque(MainLocRoomSpaceMatrix), 1, false, roomSpace)
	s.ctx.UniformMatrix4fv(s.GetUniformOpaque(MainLocFlashSpaceMatrix), 1, false, flashSpace)
	s.ctx.Uniform1f(s.GetUniformOpaque(MainLocTime), interval)

	// Environment & Material
	s.ctx.Uniform2f(s.GetUniformOpaque(MainLocScreenResolution), float32(fbW), float32(fbH))
	s.ctx.Uniform1f(s.GetUniformOpaque(MainLocEmissiveIntensity), s.emissiveIntensity)
	s.ctx.Uniform1f(s.GetUniformOpaque(MainLocAmbientLight), ambient)
	s.ctx.Uniform1i(s.GetUniformOpaque(MainLocEnableShadows), s.shadows)
	s.ctx.Uniform1f(s.GetUniformOpaque(MainLocShininessWall), float32(s.cal.ShininessWall))
	s.ctx.Uniform1f(s.GetUniformOpaque(MainLocShininessFloor), float32(s.cal.ShininessFloor))
	s.ctx.Uniform1f(s.GetUniformOpaque(MainLocSpecBoostWall), float32(s.cal.SpecBoostWall))
	s.ctx.Uniform1f(s.GetUniformOpaque(MainLocSpecBoostFloor), float32(s.cal.SpecBoostFloor))
	s.ctx.Uniform1i(s.GetUniformOpaque(MainLocDebugLights), int32(s.debugLights))

	// Flashlight
	flashIntensity := float32(s.flash.GetIntensity())
	if flashIntensity > 0.001 {
		s.ctx.Uniform1i(s.GetUniformOpaque(MainLocEnableFlashlight), 1)
		s.ctx.Uniform3f(s.GetUniformOpaque(MainLocFlashPosView), flashX, flashY, 0.0)
		dirX := float32(s.flash.GetOffsetX()) - (flashX * flashSensitivity)
		dirY := float32(s.flash.GetOffsetY()) + (flashY * flashSensitivity)
		s.ctx.Uniform3f(s.GetUniformOpaque(MainLocFlashDir), dirX, dirY, -1.0)
		s.ctx.Uniform1f(s.GetUniformOpaque(MainLocFlashIntensityFactor), flashIntensity)
		s.ctx.Uniform1f(s.GetUniformOpaque(MainLocFlashFalloff), float32(s.flash.GetFalloff()))
		s.ctx.Uniform1f(s.GetUniformOpaque(MainLocFlashConeStart), float32(s.flash.GetConeStart()))
		s.ctx.Uniform1f(s.GetUniformOpaque(MainLocFlashConeEnd), float32(s.flash.GetConeEnd()))
	} else {
		s.ctx.Uniform1i(s.GetUniformOpaque(MainLocEnableFlashlight), 0)
	}

	// Dynamic Shadow Lights
	shadowLights, numShadowLights := s.shadowLightsMetrics.GetLights()
	s.ctx.Uniform1i(s.GetUniformOpaque(MainLocNumShadowLights), int32(numShadowLights))
	for i := 0; i < int(numShadowLights); i++ {
		l := shadowLights[i]
		loc := &s.shadowLightLocs[i]
		s.ctx.Uniform4f(loc.posIntensity, l.X, l.Y, l.Z, l.Intensity)
		s.ctx.Uniform4f(loc.dirFalloff, l.DirX, l.DirY, l.DirZ, l.Falloff)
		s.ctx.Uniform4f(loc.colorType, l.R, l.G, l.B, l.Kind)
		s.ctx.Uniform4f(loc.spotParams, l.OuterCutOff, l.CutOff, 0.0, 0.0)
		s.ctx.UniformMatrix4fv(loc.lightSpace, 1, false, s.shadowLightsMetrics.GetSpacePtr(uint32(i)))
	}

	// Dynamic UBO Lights
	s.ctx.Uniform1i(s.GetUniformOpaque(MainLocNumLights), s.lights.GetActiveLightsCount())
	s.ctx.BindBufferBase(api.UNIFORM_BUFFER, 0, s.lights.GetActiveUBO())

	// Shadow Texture (Master Shadow Atlas)
	if s.shadows != 0 {
		s.ctx.ActiveTexture(api.TEXTURE0 + api.TexUnitShadowAtlas)
		s.ctx.BindTexture(api.TEXTURE_2D, s.depth.GetShadowAtlasTexture())
	}

	s.ctx.DepthMask(true)
	s.ctx.DepthFunc(api.LESS)

	s.ctx.BindVertexArray(s.mainVAO[s.frameIdx])

	// Enable Alpha To Coverage only for the main geometry
	s.ctx.Enable(api.SAMPLE_ALPHA_TO_COVERAGE)
	renderGeometry()
	// Disable it immediately so subsequent passes are not affected
	s.ctx.Disable(api.SAMPLE_ALPHA_TO_COVERAGE)
}
