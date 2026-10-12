package shaders

import (
	"fmt"

	"github.com/markel1974/godoom/mr_tech/model"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/metrics"
)

// deferredDoubleBuffer defines the number of buffers utilized in a double buffering mechanism for deferred rendering.
const (
	deferredDoubleBuffer = 2
)

// DeferredLoc represents a unique identifier for a specific uniform location within a deferred rendering pipeline.
type DeferredLoc int

const (
	DeferredLocProjection = DeferredLoc(iota)
	DeferredLocView
	DeferredLocInvView
	DeferredLocGPositionDepth
	DeferredLocGNormal
	DeferredLocGAlbedoSpec
	DeferredLocGEmissive
	DeferredLocSSAO
	DeferredLocScreenResolution
	DeferredLocAoFactor
	DeferredLocAmbientLight
	DeferredLocNumLights
	DeferredLocNumShadowLights
	DeferredLocShininessWall
	DeferredLocShininessFloor
	DeferredLocSpecBoostWall
	DeferredLocSpecBoostFloor
	DeferredLocFlashSpaceMatrix
	DeferredLocFlashPosView
	DeferredLocFlashSpotDir
	DeferredLocFlashCutOff
	DeferredLocFlashOuterCutOff
	DeferredLocFlashIntensityFactor
	DeferredLocFlashFalloff
	DeferredLocFlashShadowMap
	DeferredLocEnableShadows
	DeferredLocRoomShadowMap
	DeferredLocLast
)

// Deferred represents a structure for managing deferred rendering operations in a 3D rendering pipeline.
type Deferred struct {
	ctx               api.IContext
	handler           *Handler
	depth             *Depth
	ssao              *SSAO
	mapMetrics        *metrics.Map
	flashlightMetrics *metrics.Flashlight
	flash             *model.Flash
	post              *Post
	prg               uint32
	table             [DeferredLocLast]int32
	uboLights         [deferredDoubleBuffer]uint32
	uboShadowLights   [deferredDoubleBuffer]uint32
	frameIdx          int
	activeLights      int32
	numShadowLights   int32
	stride            int32
	cal               *model.Calibration
	shadows           bool
	shadowsInt        int32

	vao uint32
	vbo uint32
}

// NewDeferred creates a new Deferred object configured with a rendering context, stride, and calibration data.
func NewDeferred(stride int32) *Deferred {
	d := &Deferred{
		stride:   stride,
		frameIdx: 0,
	}
	d.EnableShadows(true)
	return d
}

// EnableShadows updates the shadows state.
func (s *Deferred) EnableShadows(e bool) {
	s.shadows = e
	if s.shadows {
		s.shadowsInt = 1
	} else {
		s.shadowsInt = 0
	}
}

// Setup initializes the Deferred instance by assigning rendering resources from the provided Handler.
func (s *Deferred) Setup(h *Handler) error {
	s.handler = h
	s.ctx = h.GetContext()
	s.cal = h.GetCalibration()
	s.depth = h.GetDepth()
	s.ssao = h.GetSSAO()
	s.mapMetrics = h.GetMapMetrics()
	s.flashlightMetrics = h.GetFlashlightMetrics()
	s.flash = h.GetFlash()
	s.post = h.GetPost()
	return nil
}

// Init initializes the uniform buffer objects used for lighting with dynamic draw usage and prepares them for rendering.
func (s *Deferred) Init() error {
	size := 1024 * int(s.stride)
	s.ctx.GenBuffers(deferredDoubleBuffer, &s.uboLights[0])
	s.ctx.GenBuffers(deferredDoubleBuffer, &s.uboShadowLights[0])
	for i := 0; i < deferredDoubleBuffer; i++ {
		s.ctx.BindBuffer(api.UNIFORM_BUFFER, s.uboLights[i])
		s.ctx.BufferData(api.UNIFORM_BUFFER, size, s.ctx.Ptr(nil), api.DYNAMIC_DRAW)

		s.ctx.BindBuffer(api.UNIFORM_BUFFER, s.uboShadowLights[i])
		s.ctx.BufferData(api.UNIFORM_BUFFER, 8*128, s.ctx.Ptr(nil), api.DYNAMIC_DRAW)
	}
	s.ctx.BindBuffer(api.UNIFORM_BUFFER, 0)
	return nil
}

// SetupSamplers initializes OpenGL samplers, binds buffer objects, and configures attributes for rendering a fullscreen quad.
func (s *Deferred) SetupSamplers() error {
	s.ctx.Disable(api.DEPTH_TEST)
	s.ctx.UseProgram(s.prg)

	s.ctx.Uniform1i(s.table[DeferredLocGPositionDepth], 0)
	s.ctx.Uniform1i(s.table[DeferredLocGNormal], 1)
	s.ctx.Uniform1i(s.table[DeferredLocGAlbedoSpec], 2)
	s.ctx.Uniform1i(s.table[DeferredLocGEmissive], 3)
	s.ctx.Uniform1i(s.table[DeferredLocSSAO], 4)
	s.ctx.Uniform1i(s.table[DeferredLocFlashShadowMap], 5)
	s.ctx.Uniform1i(s.table[DeferredLocRoomShadowMap], 6)

	for i := 0; i < 8; i++ {
		name := fmt.Sprintf("u_dynamicShadowMaps[%d]\x00", i)
		loc := s.ctx.GetUniformLocation(s.prg, s.ctx.Str(name))
		if loc >= 0 {
			s.ctx.Uniform1i(loc, int32(7+i))
		}
	}

	s.ctx.GenVertexArrays(1, &s.vao)
	s.ctx.BindVertexArray(s.vao)
	s.ctx.GenBuffers(1, &s.vbo)
	s.ctx.BindBuffer(api.ARRAY_BUFFER, s.vbo)
	quad := []float32{-1, -1, 1, -1, -1, 1, 1, 1}
	s.ctx.BufferData(api.ARRAY_BUFFER, len(quad)*4, s.ctx.Ptr(quad), api.STATIC_DRAW)
	s.ctx.VertexAttribPointer(0, 2, api.FLOAT, false, 0, nil)
	s.ctx.EnableVertexAttribArray(0)

	return nil
}

// GetUniform retrieves the uniform location for the specified identifier from the internal table.
func (s *Deferred) GetUniform(id DeferredLoc) int32 {
	return s.table[id]
}

// Compile compiles and links shader programs using provided assets and initializes uniform locations for deferred rendering.
func (s *Deferred) Compile(a IAssets) error {
	const vertId = "deferred.vert"
	const fragId = "deferred.frag"

	vSrc, fSrc, err := a.ReadMulti(vertId, fragId)
	if err != nil {
		return err
	}
	vSh, err := ShaderCompile(s.ctx, vertId, string(vSrc), api.VERTEX_SHADER)
	if err != nil {
		return err
	}
	fSh, err := ShaderCompile(s.ctx, fragId, string(fSrc), api.FRAGMENT_SHADER)
	if err != nil {
		s.ctx.DeleteShader(vSh)
		return err
	}
	s.prg, err = ShaderCreateProgram(s.ctx, "deferred", vSh, fSh)
	if err != nil {
		return err
	}

	s.table[DeferredLocProjection] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_projection\x00"))
	s.table[DeferredLocView] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_view\x00"))
	s.table[DeferredLocInvView] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_invView\x00"))
	s.table[DeferredLocGPositionDepth] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_gPositionDepth\x00"))
	s.table[DeferredLocGNormal] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_gNormal\x00"))
	s.table[DeferredLocGAlbedoSpec] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_gAlbedoSpec\x00"))
	s.table[DeferredLocGEmissive] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_gEmissive\x00"))
	s.table[DeferredLocSSAO] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_ssao\x00"))
	s.table[DeferredLocScreenResolution] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_screenResolution\x00"))
	s.table[DeferredLocAoFactor] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_aoFactor\x00"))
	s.table[DeferredLocAmbientLight] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_ambient_light\x00"))
	s.table[DeferredLocNumLights] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_numLights\x00"))
	s.table[DeferredLocNumShadowLights] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_numShadowLights\x00"))
	s.table[DeferredLocShininessWall] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_shininessWall\x00"))
	s.table[DeferredLocShininessFloor] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_shininessFloor\x00"))
	s.table[DeferredLocSpecBoostWall] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_specBoostWall\x00"))
	s.table[DeferredLocSpecBoostFloor] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_specBoostFloor\x00"))

	s.table[DeferredLocFlashSpaceMatrix] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_flashSpaceMatrix\x00"))
	s.table[DeferredLocFlashPosView] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_flashPosView\x00"))
	s.table[DeferredLocFlashSpotDir] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_flashSpotDir\x00"))
	s.table[DeferredLocFlashCutOff] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_flashCutOff\x00"))
	s.table[DeferredLocFlashOuterCutOff] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_flashOuterCutOff\x00"))
	s.table[DeferredLocFlashIntensityFactor] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_flashIntensityFactor\x00"))
	s.table[DeferredLocFlashFalloff] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_flashFalloff\x00"))
	s.table[DeferredLocFlashShadowMap] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_flashShadowMap\x00"))
	s.table[DeferredLocEnableShadows] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_enableShadows\x00"))
	s.table[DeferredLocRoomShadowMap] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_roomShadowMap\x00"))

	for idx, v := range s.table {
		if v < 0 {
			fmt.Printf("[warning] unused uniform location in deferred [table index: %d]\n", idx)
		}
	}

	blockIndex := s.ctx.GetUniformBlockIndex(s.prg, s.ctx.Str("LightsBlock\x00"))
	if blockIndex != api.INVALID_INDEX {
		s.ctx.UniformBlockBinding(s.prg, blockIndex, 0)
	}

	blockShadow := s.ctx.GetUniformBlockIndex(s.prg, s.ctx.Str("ShadowLightsBlock\x00"))
	if blockShadow != api.INVALID_INDEX {
		s.ctx.UniformBlockBinding(s.prg, blockShadow, 1)
	}

	return nil
}

// Prepare initializes lighting data for the current frame and updates the uniform buffer with light data.
func (s *Deferred) Prepare(frameLights []float32, numLights int32, shadowLightsData []float32, numShadowLights int32) {
	s.activeLights = numLights
	s.numShadowLights = numShadowLights
	s.frameIdx = (s.frameIdx + 1) % deferredDoubleBuffer
	if numLights > 0 {
		s.ctx.BindBuffer(api.UNIFORM_BUFFER, s.uboLights[s.frameIdx])
		s.ctx.BufferSubData(api.UNIFORM_BUFFER, 0, len(frameLights)*4, s.ctx.Ptr(frameLights))
	}
	if numShadowLights > 0 && len(shadowLightsData) > 0 {
		s.ctx.BindBuffer(api.UNIFORM_BUFFER, s.uboShadowLights[s.frameIdx])
		s.ctx.BufferSubData(api.UNIFORM_BUFFER, 0, len(shadowLightsData)*4, s.ctx.Ptr(shadowLightsData))
	}
	s.ctx.BindBuffer(api.UNIFORM_BUFFER, 0)
}

// Render performs the deferred rendering pass, applying lighting and shadows to the screen quad.
func (s *Deferred) Render(ambient float32, flashX, flashY, flashDirX, flashDirY float32) {
	fbW, fbH := s.handler.GetSize()
	view := s.mapMetrics.GetViewPtr()
	invView := s.mapMetrics.GetInvViewPtr()
	proj := s.mapMetrics.GetProjPtr()
	postFBO := s.post.GetFBO()

	s.ctx.BindFramebuffer(api.FRAMEBUFFER, postFBO)
	drawBuffers := []uint32{api.COLOR_ATTACHMENT0, api.COLOR_ATTACHMENT1}
	s.ctx.DrawBuffers(2, &drawBuffers[0])

	s.ctx.Enable(api.DEPTH_TEST)
	s.ctx.DepthFunc(api.ALWAYS)
	s.ctx.DepthMask(true)
	s.ctx.UseProgram(s.prg)

	s.ctx.ActiveTexture(api.TEXTURE0)
	s.ctx.BindTexture(api.TEXTURE_2D, s.ssao.GetPositionDepthTexture())
	s.ctx.ActiveTexture(api.TEXTURE1)
	s.ctx.BindTexture(api.TEXTURE_2D, s.ssao.GetNormalTexture())
	s.ctx.ActiveTexture(api.TEXTURE2)
	s.ctx.BindTexture(api.TEXTURE_2D, s.ssao.GetAlbedoTexture())
	s.ctx.ActiveTexture(api.TEXTURE3)
	s.ctx.BindTexture(api.TEXTURE_2D, s.ssao.GetEmissiveTexture())
	s.ctx.ActiveTexture(api.TEXTURE4)
	s.ctx.BindTexture(api.TEXTURE_2D, s.ssao.GetSSAOBlurTexture())
	s.ctx.ActiveTexture(api.TEXTURE5)
	s.ctx.BindTexture(api.TEXTURE_2D, s.depth.GetFlashShadowTextures())
	s.ctx.ActiveTexture(api.TEXTURE6)
	s.ctx.BindTexture(api.TEXTURE_2D, s.depth.GetRoomShadowTextures())

	// Dynamic shadow textures on units 7..14
	for i := int32(0); i < s.numShadowLights; i++ {
		lTex, _, _ := s.depth.GetShadowLightTextures(uint32(i))
		s.ctx.ActiveTexture(api.TEXTURE0 + uint32(7+i))
		s.ctx.BindTexture(api.TEXTURE_2D, lTex)
	}

	s.ctx.UniformMatrix4fv(s.GetUniform(DeferredLocProjection), 1, false, proj)
	s.ctx.UniformMatrix4fv(s.GetUniform(DeferredLocView), 1, false, view)
	s.ctx.UniformMatrix4fv(s.GetUniform(DeferredLocInvView), 1, false, invView)
	s.ctx.Uniform1i(s.GetUniform(DeferredLocNumLights), s.activeLights)
	s.ctx.Uniform1i(s.GetUniform(DeferredLocNumShadowLights), s.numShadowLights)
	s.ctx.Uniform2f(s.GetUniform(DeferredLocScreenResolution), float32(fbW), float32(fbH))
	s.ctx.Uniform1f(s.GetUniform(DeferredLocAmbientLight), ambient)
	s.ctx.Uniform1f(s.GetUniform(DeferredLocAoFactor), s.handler.GetAoFactor())

	s.ctx.Uniform1f(s.GetUniform(DeferredLocShininessWall), float32(s.cal.ShininessWall))
	s.ctx.Uniform1f(s.GetUniform(DeferredLocShininessFloor), float32(s.cal.ShininessFloor))
	s.ctx.Uniform1f(s.GetUniform(DeferredLocSpecBoostWall), float32(s.cal.SpecBoostWall))
	s.ctx.Uniform1f(s.GetUniform(DeferredLocSpecBoostFloor), float32(s.cal.SpecBoostFloor))

	// Flashlight
	s.ctx.UniformMatrix4fv(s.GetUniform(DeferredLocFlashSpaceMatrix), 1, false, s.flashlightMetrics.GetShadowSpacePtr())
	s.ctx.Uniform3f(s.GetUniform(DeferredLocFlashPosView), flashX, flashY, 0.0)
	s.ctx.Uniform3f(s.GetUniform(DeferredLocFlashSpotDir), flashDirX, flashDirY, -1.0)
	s.ctx.Uniform1f(s.GetUniform(DeferredLocFlashCutOff), float32(s.flash.GetConeStart()))
	s.ctx.Uniform1f(s.GetUniform(DeferredLocFlashOuterCutOff), float32(s.flash.GetConeEnd()))
	s.ctx.Uniform1f(s.GetUniform(DeferredLocFlashIntensityFactor), float32(s.flash.GetIntensity()))
	s.ctx.Uniform1f(s.GetUniform(DeferredLocFlashFalloff), float32(s.flash.GetFalloff()))
	s.ctx.Uniform1i(s.GetUniform(DeferredLocEnableShadows), s.shadowsInt)

	s.ctx.BindBufferBase(api.UNIFORM_BUFFER, 0, s.uboLights[s.frameIdx])
	s.ctx.BindBufferBase(api.UNIFORM_BUFFER, 1, s.uboShadowLights[s.frameIdx])

	s.ctx.BindVertexArray(s.vao)
	s.ctx.DrawArrays(api.TRIANGLE_STRIP, 0, 4)
	s.ctx.DepthMask(false)
}
