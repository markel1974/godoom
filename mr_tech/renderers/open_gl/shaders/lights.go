package shaders

import (
	"fmt"

	"github.com/go-gl/gl/v3.3-core/gl"
)

type LightLoc int

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

type Lights struct {
	prg   uint32
	table [LightLocLast]int32
}

func NewLights() *Lights {
	return &Lights{}
}

func (s *Lights) Setup(width, height int32) {}

func (s *Lights) SetupSamplers() {
	gl.UseProgram(s.prg)
	gl.Uniform1i(s.GetUniform(LightLocTexture), 0)
	gl.Uniform1i(s.GetUniform(LightLocNormalMap), 1)
	gl.Uniform1i(s.GetUniform(LightLocRoomShadowMap), 3)
}

func (s *Lights) GetUniform(id LightLoc) int32 {
	return s.table[id]
}

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

func (s *Lights) Render(proj, view, invView, roomSpace [16]float32, ambient, spotIntensity, beamRatio float32, enableShadows int32, volSteps int32, screenW, screenH float32, uboLights uint32) {
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

	gl.BindBufferBase(gl.UNIFORM_BUFFER, 0, uboLights)
}
