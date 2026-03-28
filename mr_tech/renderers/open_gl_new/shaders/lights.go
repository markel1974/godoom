package shaders

import (
	"fmt"

	"github.com/go-gl/gl/v3.3-core/gl"
)

// LightLoc is a type representing uniform variable locations in a shader program.
type LightLoc int

// LightLocProjection represents the projection matrix location for the light.
// LightLocView represents the view matrix location for the light.
// LightLocInvView represents the inverse view matrix location for the light.
// LightLocRoomSpaceMatrix represents the room-space matrix location for the light.
// LightLocTexture represents the texture location for the light.
// LightLocNormalMap represents the normal map location for the light.
// LightLocRoomShadowMap represents the room shadow map location for the light.
// LightLocScreenResolution represents the screen resolution location for the light.
// LightLocAmbientLight represents the ambient light parameter location.
// LightLocEnableShadows represents the toggle for enabling or disabling shadows.
// LightLocVolumetricSteps represents the number of volumetric steps for rendering light effects.
// LightLocBeamRatioFactor represents the beam ratio factor for volumetric light effects.
// LightLocLast is a marker for the last element in the LightLoc enumeration.
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
	LightLocLast
)

// Lights manages a shader program and its associated uniform locations for rendering lighting effects.
type Lights struct {
	prg       uint32
	table     [LightLocLast]int32
	uboLights uint32
}

// NewLights initializes and returns a new instance of Lights with default settings.
func NewLights() *Lights {
	return &Lights{}
}

// Setup initializes light configurations with the specified screen width and height.
func (s *Lights) Setup(width, height int32) {}

func (s *Lights) Init() {
	// CREAZIONE UBO LIGHTS
	gl.GenBuffers(1, &s.uboLights)
	gl.BindBuffer(gl.UNIFORM_BUFFER, s.uboLights)
	gl.BufferData(gl.UNIFORM_BUFFER, (256*16)+16, nil, gl.DYNAMIC_DRAW)
	gl.BindBufferBase(gl.UNIFORM_BUFFER, 0, s.uboLights)
	gl.BindBuffer(gl.UNIFORM_BUFFER, 0)
}

// SetupSamplers initializes texture samplers for the shader by binding uniform locations to specific texture units.
func (s *Lights) SetupSamplers() {
	gl.UseProgram(s.prg)
	gl.Uniform1i(s.GetUniform(LightLocTexture), 0)
	gl.Uniform1i(s.GetUniform(LightLocNormalMap), 1)
	gl.Uniform1i(s.GetUniform(LightLocRoomShadowMap), 3)
}

// GetUniform retrieves the location of a specified uniform variable from the precomputed table.
func (s *Lights) GetUniform(id LightLoc) int32 {
	return s.table[id]
}

// Compile initializes and builds shaders for the Lights object, setting up uniform locations and validating their presence.
func (s *Lights) Compile(a IAssets) error {
	vSrc, fSrc, err := a.ReadMulti("main.vert", "lights.frag")
	if err != nil {
		return err
	}
	vSh, err := ShaderCompile(string(vSrc), gl.VERTEX_SHADER)
	if err != nil {
		return err
	}
	fSh, err := ShaderCompile(string(fSrc), gl.FRAGMENT_SHADER)
	if err != nil {
		gl.DeleteShader(vSh)
		return err
	}
	s.prg, err = ShaderCreateProgram(vSh, fSh)
	if err != nil {
		return err
	}

	s.table[LightLocProjection] = gl.GetUniformLocation(s.prg, gl.Str("u_projection\x00"))
	s.table[LightLocView] = gl.GetUniformLocation(s.prg, gl.Str("u_view\x00"))
	s.table[LightLocInvView] = gl.GetUniformLocation(s.prg, gl.Str("u_invView\x00"))
	s.table[LightLocRoomSpaceMatrix] = gl.GetUniformLocation(s.prg, gl.Str("u_roomSpaceMatrix\x00"))
	s.table[LightLocTexture] = gl.GetUniformLocation(s.prg, gl.Str("u_texture\x00"))
	s.table[LightLocNormalMap] = gl.GetUniformLocation(s.prg, gl.Str("u_normalMap\x00"))
	s.table[LightLocRoomShadowMap] = gl.GetUniformLocation(s.prg, gl.Str("u_roomShadowMap\x00"))
	s.table[LightLocScreenResolution] = gl.GetUniformLocation(s.prg, gl.Str("u_screenResolution\x00"))
	s.table[LightLocAmbientLight] = gl.GetUniformLocation(s.prg, gl.Str("u_ambient_light\x00"))
	s.table[LightLocEnableShadows] = gl.GetUniformLocation(s.prg, gl.Str("u_enableShadows\x00"))
	s.table[LightLocVolumetricSteps] = gl.GetUniformLocation(s.prg, gl.Str("u_volumetricSteps\x00"))
	s.table[LightLocBeamRatioFactor] = gl.GetUniformLocation(s.prg, gl.Str("u_beamRatioFactor\x00"))

	for idx, v := range s.table {
		if v < 0 {
			return fmt.Errorf("invalid uniform location in lights: %d\n", idx)
		}
	}

	blockIndex := gl.GetUniformBlockIndex(s.prg, gl.Str("LightsBlock\x00"))
	if blockIndex != gl.INVALID_INDEX {
		gl.UniformBlockBinding(s.prg, blockIndex, 0)
	}

	return nil
}

func (s *Lights) Prepare(frameLights []float32, numLights int32) {
	// UPLOAD UBO LIGHTS
	gl.BindBuffer(gl.UNIFORM_BUFFER, s.uboLights)
	if numLights > 0 {
		gl.BufferSubData(gl.UNIFORM_BUFFER, 0, int(numLights)*16, gl.Ptr(frameLights))
	}
	gl.BufferSubData(gl.UNIFORM_BUFFER, 256*16, 4, gl.Ptr(&numLights))
	gl.BindBuffer(gl.UNIFORM_BUFFER, 0)
}

// Render configures and dispatches the shader program to render lights with provided matrices, ambient light, and settings.
func (s *Lights) Render(view, proj, invView, roomSpace [16]float32, ambient float32, enableShadows int32, screenW, screenH float32) {
	const spotIntensity = 10.0
	const beamRatio = 0.05
	const volSteps = 32

	gl.UseProgram(s.prg)

	gl.UniformMatrix4fv(s.GetUniform(LightLocProjection), 1, false, &proj[0])
	gl.UniformMatrix4fv(s.GetUniform(LightLocView), 1, false, &view[0])
	gl.UniformMatrix4fv(s.GetUniform(LightLocInvView), 1, false, &invView[0])
	gl.UniformMatrix4fv(s.GetUniform(LightLocRoomSpaceMatrix), 1, false, &roomSpace[0])

	gl.Uniform2f(s.GetUniform(LightLocScreenResolution), screenW, screenH)
	gl.Uniform1f(s.GetUniform(LightLocAmbientLight), ambient)
	gl.Uniform1i(s.GetUniform(LightLocEnableShadows), enableShadows)
	gl.Uniform1i(s.GetUniform(LightLocVolumetricSteps), volSteps)
	gl.Uniform1f(s.GetUniform(LightLocBeamRatioFactor), beamRatio)

	gl.BindBufferBase(gl.UNIFORM_BUFFER, 0, s.uboLights)
}
