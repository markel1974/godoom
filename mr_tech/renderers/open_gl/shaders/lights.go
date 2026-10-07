package shaders

import (
	"fmt"

	"github.com/markel1974/godoom/mr_tech/model"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
)

const (
	lightsDoubleBuffer = 2
)

// LightLoc represents an identifier for light-related uniform locations in a shader program.
type LightLoc int

// LightLocProjection specifies the light's projection matrix location.
// LightLocView specifies the light's view matrix location.
// LightLocInvView specifies the light's inverse view matrix location.
// LightLocRoomSpaceMatrix specifies the light's room-space matrix location.
// LightLocTexture specifies the light's texture resource location.
// LightLocNormalMap specifies the light's normal map resource location.
// LightLocRoomShadowMap specifies the light's room shadow map resource location.
// LightLocScreenResolution specifies the screen resolution location.
// LightLocAmbientLight specifies the ambient light intensity location.
// LightLocEnableShadows specifies whether shadows are enabled location.
// LightLocVolumetricSteps specifies the number of volumetric lighting steps location.
// LightLocBeamRatioFactor specifies the beam-to-light ratio factor location.
// LightLocNumLights specifies the number of active lights location.
// LightLocLast specifies the marker for the last LightLoc value.
const (
	LightLocProjection = LightLoc(iota)
	LightLocView
	LightLocInvView
	LightLocRoomSpaceMatrix
	LightLocTexture
	LightLocNormalMap
	LightLocRoomShadowMap
	LightLocScreenResolution
	LightLocAmbientLight
	LightLocEnableShadows
	LightLocVolumetricSteps
	LightLocBeamRatioFactor
	LightLocNumLights
	LightLocShininessWall
	LightLocShininessFloor
	LightLocSpecBoostWall
	LightLocSpecBoostFloor
	LightLocDebugLights
	LightLocLast
)

// Lights represents a collection of light data and OpenGL resources for managing and rendering dynamic scene lighting.
type Lights struct {
	ctx          api.IContext
	prg          uint32
	table        [LightLocLast]int32
	uboLights    [lightsDoubleBuffer]uint32
	frameIdx     int
	activeLights int32
	shadows      int32
	stride       int32
	cal          *model.Calibration
	debugLights  int
}

// NewLights initializes and returns a new instance of Lights with default settings.
func NewLights(ctx api.IContext, stride int32, cal *model.Calibration) *Lights {
	return &Lights{
		ctx:         ctx,
		cal:         cal,
		stride:      stride,
		frameIdx:    0,
		debugLights: 0,
	}
}

// EnableShadows enables or disables shadow rendering based on the provided boolean value.
func (s *Lights) EnableShadows(e bool) {
	if e {
		s.shadows = 1
	} else {
		s.shadows = 0
	}
}

// Init initializes the uniform buffer object (UBO) for storing light data with the specified stride size.
func (s *Lights) Init() error {
	size := 1024 * int(s.stride)

	s.ctx.GenBuffers(lightsDoubleBuffer, &s.uboLights[0])

	for i := 0; i < lightsDoubleBuffer; i++ {
		s.ctx.BindBuffer(api.UNIFORM_BUFFER, s.uboLights[i])
		s.ctx.BufferData(api.UNIFORM_BUFFER, size, s.ctx.Ptr(nil), api.DYNAMIC_DRAW)
	}

	s.ctx.BindBuffer(api.UNIFORM_BUFFER, 0)
	return nil
}

// SetupSamplers configures shader samplers for texture, normal map, and room shadow map locations.
func (s *Lights) SetupSamplers() error {
	s.ctx.UseProgram(s.prg)
	diffuseUnits := []int32{0, 1, 2, 3}
	normalUnits := []int32{4, 5, 6, 7}
	s.ctx.Uniform1iv(s.GetUniform(LightLocTexture), 4, &diffuseUnits[0]) // FlashLocTexture in flashlight.go
	s.ctx.Uniform1iv(s.GetUniform(LightLocNormalMap), 4, &normalUnits[0])
	s.ctx.Uniform1i(s.GetUniform(LightLocRoomShadowMap), 12) // gl_api.TEXTURE12 per Room, gl_api.TEXTURE13 per Flash
	return nil
}

// GetUniform retrieves the uniform location for a given LightLoc identifier from the internal table.
func (s *Lights) GetUniform(id LightLoc) int32 {
	return s.table[id]
}

// Compile compiles the shaders for the Lights object and initializes its uniform locations and shader program.
func (s *Lights) Compile(a IAssets) error {
	const vertId = "main.vert"
	const fragId = "lights.frag"

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
	s.prg, err = ShaderCreateProgram(s.ctx, "lights", vSh, fSh)
	if err != nil {
		return err
	}

	s.table[LightLocProjection] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_projection\x00"))
	s.table[LightLocView] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_view\x00"))
	s.table[LightLocInvView] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_invView\x00"))
	s.table[LightLocRoomSpaceMatrix] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_roomSpaceMatrix\x00"))
	s.table[LightLocTexture] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_texture\x00"))
	s.table[LightLocNormalMap] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_normalMap\x00"))
	s.table[LightLocRoomShadowMap] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_roomShadowMap\x00"))
	s.table[LightLocScreenResolution] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_screenResolution\x00"))
	s.table[LightLocAmbientLight] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_ambient_light\x00"))
	s.table[LightLocEnableShadows] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_enableShadows\x00"))
	s.table[LightLocVolumetricSteps] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_volumetricSteps\x00"))
	s.table[LightLocBeamRatioFactor] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_beamRatioFactor\x00"))
	s.table[LightLocNumLights] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_numLights\x00"))
	s.table[LightLocShininessWall] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_shininessWall\x00"))
	s.table[LightLocShininessFloor] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_shininessFloor\x00"))
	s.table[LightLocSpecBoostWall] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_specBoostWall\x00"))
	s.table[LightLocSpecBoostFloor] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_specBoostFloor\x00"))
	s.table[LightLocDebugLights] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_debugLights\x00"))

	for idx, v := range s.table {
		if v < 0 {
			return fmt.Errorf("unused uniform location in lights: %d\n", idx)
		}
	}

	blockIndex := s.ctx.GetUniformBlockIndex(s.prg, s.ctx.Str("LightsBlock\x00"))
	if blockIndex != api.INVALID_INDEX {
		s.ctx.UniformBlockBinding(s.prg, blockIndex, 0)
	}

	return nil
}

// Prepare updates the uniform buffer object (UBO) with lighting data for rendering and sets the active lights count.
func (s *Lights) Prepare(frameLights []float32, numLights int32) {
	s.activeLights = numLights
	s.frameIdx = (s.frameIdx + 1) % lightsDoubleBuffer
	if numLights > 0 {
		s.ctx.BindBuffer(api.UNIFORM_BUFFER, s.uboLights[s.frameIdx])
		// Scrittura asincrona garantita sull'UBO inattivo
		s.ctx.BufferSubData(api.UNIFORM_BUFFER, 0, len(frameLights)*4, s.ctx.Ptr(frameLights))
		s.ctx.BindBuffer(api.UNIFORM_BUFFER, 0)
	}
}

// Render configures the shader program and draws geometry with lighting, shadows, and volumetric effects applied.
func (s *Lights) Render(renderGeometry func(), roomShadowTex uint32, view, proj, invView, roomSpace *float32, ambient float32, screenW, screenH float32) {
	s.ctx.UseProgram(s.prg)

	s.ctx.UniformMatrix4fv(s.GetUniform(LightLocProjection), 1, false, proj)
	s.ctx.UniformMatrix4fv(s.GetUniform(LightLocView), 1, false, view)
	s.ctx.UniformMatrix4fv(s.GetUniform(LightLocInvView), 1, false, invView)
	s.ctx.UniformMatrix4fv(s.GetUniform(LightLocRoomSpaceMatrix), 1, false, roomSpace)
	s.ctx.Uniform1i(s.GetUniform(LightLocNumLights), s.activeLights)
	s.ctx.Uniform2f(s.GetUniform(LightLocScreenResolution), screenW, screenH)
	s.ctx.Uniform1f(s.GetUniform(LightLocAmbientLight), ambient)
	s.ctx.Uniform1i(s.GetUniform(LightLocEnableShadows), s.shadows)
	s.ctx.Uniform1i(s.GetUniform(LightLocVolumetricSteps), int32(s.cal.VolSteps))
	s.ctx.Uniform1f(s.GetUniform(LightLocBeamRatioFactor), float32(s.cal.BeamRatio))
	s.ctx.Uniform1f(s.GetUniform(LightLocShininessWall), float32(s.cal.ShininessWall))
	s.ctx.Uniform1f(s.GetUniform(LightLocShininessFloor), float32(s.cal.ShininessFloor))
	s.ctx.Uniform1f(s.GetUniform(LightLocSpecBoostWall), float32(s.cal.SpecBoostWall))
	s.ctx.Uniform1f(s.GetUniform(LightLocSpecBoostFloor), float32(s.cal.SpecBoostFloor))
	s.ctx.Uniform1i(s.GetUniform(LightLocDebugLights), int32(s.debugLights))
	s.ctx.BindBufferBase(api.UNIFORM_BUFFER, 0, s.uboLights[s.frameIdx])
	if s.shadows != 0 {
		s.ctx.ActiveTexture(api.TEXTURE12)
		s.ctx.BindTexture(api.TEXTURE_2D, roomShadowTex)
	}
	renderGeometry()
}
