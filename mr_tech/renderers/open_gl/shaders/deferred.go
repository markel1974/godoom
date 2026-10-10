package shaders

import (
	"fmt"

	"github.com/markel1974/godoom/mr_tech/model"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
)

// deferredDoubleBuffer defines the number of buffers utilized in a double buffering mechanism for deferred rendering.
const (
	deferredDoubleBuffer = 2
)

// DeferredLoc represents a unique identifier for a specific uniform location within a deferred rendering pipeline.
type DeferredLoc int

// DeferredLocView represents the view matrix location in a deferred rendering pipeline.
// DeferredLocGPositionDepth represents the position and depth buffer location.
// DeferredLocGNormal represents the normal buffer location.
// DeferredLocGAlbedoSpec represents the albedo and specular buffer location.
// DeferredLocGEmissive represents the emissive buffer location.
// DeferredLocSSAO represents the screen space ambient occlusion (SSAO) buffer location.
// DeferredLocScreenResolution represents the screen resolution location.
// DeferredLocAoFactor represents the ambient occlusion factor location.
// DeferredLocAmbientLight represents the ambient light factor location.
// DeferredLocNumLights represents the number of active lights location.
// DeferredLocShininessWall represents the shininess value for wall materials.
// DeferredLocShininessFloor represents the shininess value for floor materials.
// DeferredLocSpecBoostWall represents the specular boost factor for wall materials.
// DeferredLocSpecBoostFloor represents the specular boost factor for floor materials.
// DeferredLocFlashSpaceMatrix represents the space transformation matrix for a flashlight.
// DeferredLocFlashPosView represents the view-space position of the flashlight.
// DeferredLocFlashSpotDir represents the direction of the flashlight spotlight.
// DeferredLocFlashCutOff represents the inner cutoff angle of the flashlight spotlight.
// DeferredLocFlashOuterCutOff represents the outer cutoff angle of the flashlight spotlight.
// DeferredLocFlashIntensityFactor represents the intensity factor of the flashlight.
// DeferredLocFlashShadowMap represents the shadow map associated with the flashlight.
// DeferredLocRoomShadowMap represents the shadow map associated with the room environment.
// DeferredLocLast represents the last enumeration value for DeferredLoc, used for boundary validation.
const (
	DeferredLocView = DeferredLoc(iota)
	DeferredLocGPositionDepth
	DeferredLocGNormal
	DeferredLocGAlbedoSpec
	DeferredLocGEmissive
	DeferredLocSSAO
	DeferredLocScreenResolution
	DeferredLocAoFactor
	DeferredLocAmbientLight
	DeferredLocNumLights
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
	DeferredLocFlashShadowMap
	DeferredLocRoomShadowMap
	DeferredLocLast
)

// Deferred represents a structure for managing deferred rendering operations in a 3D rendering pipeline.
type Deferred struct {
	ctx          api.IContext
	prg          uint32
	table        [DeferredLocLast]int32
	uboLights    [deferredDoubleBuffer]uint32
	frameIdx     int
	activeLights int32
	stride       int32
	cal          *model.Calibration

	vao uint32
	vbo uint32
}

// NewDeferred creates a new Deferred object configured with a rendering context, stride, and calibration data.
func NewDeferred(ctx api.IContext, stride int32, cal *model.Calibration) *Deferred {
	return &Deferred{
		ctx:      ctx,
		cal:      cal,
		stride:   stride,
		frameIdx: 0,
	}
}

// Init initializes the uniform buffer objects used for lighting with dynamic draw usage and prepares them for rendering.
func (s *Deferred) Init() error {
	size := 1024 * int(s.stride)
	s.ctx.GenBuffers(deferredDoubleBuffer, &s.uboLights[0])
	for i := 0; i < deferredDoubleBuffer; i++ {
		s.ctx.BindBuffer(api.UNIFORM_BUFFER, s.uboLights[i])
		s.ctx.BufferData(api.UNIFORM_BUFFER, size, s.ctx.Ptr(nil), api.DYNAMIC_DRAW)
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

	s.table[DeferredLocView] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_view\x00"))
	s.table[DeferredLocGPositionDepth] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_gPositionDepth\x00"))
	s.table[DeferredLocGNormal] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_gNormal\x00"))
	s.table[DeferredLocGAlbedoSpec] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_gAlbedoSpec\x00"))
	s.table[DeferredLocGEmissive] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_gEmissive\x00"))
	s.table[DeferredLocSSAO] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_ssao\x00"))
	s.table[DeferredLocScreenResolution] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_screenResolution\x00"))
	s.table[DeferredLocAoFactor] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_aoFactor\x00"))
	s.table[DeferredLocAmbientLight] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_ambient_light\x00"))
	s.table[DeferredLocNumLights] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_numLights\x00"))
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
	s.table[DeferredLocFlashShadowMap] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_flashShadowMap\x00"))
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

	return nil
}

// Prepare initializes the lighting data for the current frame and updates the uniform buffer with the provided light data.
func (s *Deferred) Prepare(frameLights []float32, numLights int32) {
	s.activeLights = numLights
	s.frameIdx = (s.frameIdx + 1) % deferredDoubleBuffer
	if numLights > 0 {
		s.ctx.BindBuffer(api.UNIFORM_BUFFER, s.uboLights[s.frameIdx])
		s.ctx.BufferSubData(api.UNIFORM_BUFFER, 0, len(frameLights)*4, s.ctx.Ptr(frameLights))
		s.ctx.BindBuffer(api.UNIFORM_BUFFER, 0)
	}
}

// Render performs the deferred rendering process, applying lighting and shading effects using multiple input textures.
func (s *Deferred) Render(view *float32, ambient, aoFactor float32, screenW, screenH float32,
	gPosDepth, gNormal, gAlbedo, gEmissive, ssaoTex, flashShadowTex, roomShadowTex uint32,
	flashMatrix *float32, flashPosView, flashSpotDir []float32, flashCutOff, flashOuterCutOff, flashIntensity float32) {

	s.ctx.Disable(api.DEPTH_TEST)
	s.ctx.UseProgram(s.prg)

	s.ctx.ActiveTexture(api.TEXTURE0)
	s.ctx.BindTexture(api.TEXTURE_2D, gPosDepth)
	s.ctx.ActiveTexture(api.TEXTURE1)
	s.ctx.BindTexture(api.TEXTURE_2D, gNormal)
	s.ctx.ActiveTexture(api.TEXTURE2)
	s.ctx.BindTexture(api.TEXTURE_2D, gAlbedo)
	s.ctx.ActiveTexture(api.TEXTURE3)
	s.ctx.BindTexture(api.TEXTURE_2D, gEmissive)
	s.ctx.ActiveTexture(api.TEXTURE4)
	s.ctx.BindTexture(api.TEXTURE_2D, ssaoTex)
	s.ctx.ActiveTexture(api.TEXTURE5)
	s.ctx.BindTexture(api.TEXTURE_2D, flashShadowTex)
	s.ctx.ActiveTexture(api.TEXTURE6)
	s.ctx.BindTexture(api.TEXTURE_2D, roomShadowTex)

	s.ctx.UniformMatrix4fv(s.GetUniform(DeferredLocView), 1, false, view)
	s.ctx.Uniform1i(s.GetUniform(DeferredLocNumLights), s.activeLights)
	s.ctx.Uniform2f(s.GetUniform(DeferredLocScreenResolution), screenW, screenH)
	s.ctx.Uniform1f(s.GetUniform(DeferredLocAmbientLight), ambient)
	s.ctx.Uniform1f(s.GetUniform(DeferredLocAoFactor), aoFactor)

	s.ctx.Uniform1f(s.GetUniform(DeferredLocShininessWall), float32(s.cal.ShininessWall))
	s.ctx.Uniform1f(s.GetUniform(DeferredLocShininessFloor), float32(s.cal.ShininessFloor))
	s.ctx.Uniform1f(s.GetUniform(DeferredLocSpecBoostWall), float32(s.cal.SpecBoostWall))
	s.ctx.Uniform1f(s.GetUniform(DeferredLocSpecBoostFloor), float32(s.cal.SpecBoostFloor))

	// Flashlight
	s.ctx.UniformMatrix4fv(s.GetUniform(DeferredLocFlashSpaceMatrix), 1, false, flashMatrix)
	s.ctx.Uniform3f(s.GetUniform(DeferredLocFlashPosView), flashPosView[0], flashPosView[1], flashPosView[2])
	s.ctx.Uniform3f(s.GetUniform(DeferredLocFlashSpotDir), flashSpotDir[0], flashSpotDir[1], flashSpotDir[2])
	s.ctx.Uniform1f(s.GetUniform(DeferredLocFlashCutOff), flashCutOff)
	s.ctx.Uniform1f(s.GetUniform(DeferredLocFlashOuterCutOff), flashOuterCutOff)
	s.ctx.Uniform1f(s.GetUniform(DeferredLocFlashIntensityFactor), flashIntensity)

	s.ctx.BindBufferBase(api.UNIFORM_BUFFER, 0, s.uboLights[s.frameIdx])

	s.ctx.BindVertexArray(s.vao)
	s.ctx.DrawArrays(api.TRIANGLE_STRIP, 0, 4)
	s.ctx.Enable(api.DEPTH_TEST)
}
