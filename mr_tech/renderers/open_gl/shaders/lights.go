package shaders

import (
	"fmt"

	"github.com/go-gl/gl/v3.3-core/gl"
)

type LightLoc int

const (
	LightLocGAlbedoSpec = LightLoc(iota)
	LightLocGNormalEmiss
	LightLocGPositionDepth
	LightLocView
	LightLocInvView
	LightLocAmbientLight
	LightLocLast
)

type Lights struct {
	prg   uint32
	table [LightLocLast]int32
	ubo   uint32
}

func NewShaderLight() *Lights {
	return &Lights{}
}

func (s *Lights) Setup(width, height int32) {}

func (s *Lights) SetupSamplers() {
	gl.UseProgram(s.prg)
	gl.Uniform1i(s.GetUniform(LightLocGAlbedoSpec), 0)
	gl.Uniform1i(s.GetUniform(LightLocGNormalEmiss), 1)
	gl.Uniform1i(s.GetUniform(LightLocGPositionDepth), 2)
}

func (s *Lights) GetUniform(id LightLoc) int32 {
	return s.table[id]
}

func (s *Lights) Compile(a IAssets) error {
	vSrc, fSrc, err := a.ReadMulti("post.vert", "light.frag") // Usa il vertex shader del full-screen quad
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

	s.table[LightLocGAlbedoSpec] = gl.GetUniformLocation(s.prg, gl.Str("gAlbedoSpec\x00"))
	s.table[LightLocGNormalEmiss] = gl.GetUniformLocation(s.prg, gl.Str("gNormalEmiss\x00"))
	s.table[LightLocGPositionDepth] = gl.GetUniformLocation(s.prg, gl.Str("gPositionDepth\x00"))
	s.table[LightLocView] = gl.GetUniformLocation(s.prg, gl.Str("u_view\x00"))
	s.table[LightLocInvView] = gl.GetUniformLocation(s.prg, gl.Str("u_invView\x00"))
	s.table[LightLocAmbientLight] = gl.GetUniformLocation(s.prg, gl.Str("u_ambient_light\x00"))

	for _, v := range s.table {
		if v < 0 {
			return fmt.Errorf("invalid uniform location in Lights: %d", v)
		}
	}

	// Setup UBO Binding
	blockIndex := gl.GetUniformBlockIndex(s.prg, gl.Str("LightsBlock\x00"))
	if blockIndex != gl.INVALID_INDEX {
		gl.UniformBlockBinding(s.prg, blockIndex, 0) // Bind al point 0
	}

	return nil
}

func (s *Lights) Render(drawScreenQuad func(), view, invView [16]float32, ambient float32, uboLights uint32) {
	gl.UseProgram(s.prg)

	// Additive blending per accumulo radianza
	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.ONE, gl.ONE)
	gl.DepthMask(false)
	gl.Disable(gl.DEPTH_TEST)

	gl.UniformMatrix4fv(s.GetUniform(LightLocView), 1, false, &view[0])
	gl.UniformMatrix4fv(s.GetUniform(LightLocInvView), 1, false, &invView[0])
	gl.Uniform1f(s.GetUniform(LightLocAmbientLight), ambient)

	// Bind dell'UBO popolato dal BatchBuilder
	gl.BindBufferBase(gl.UNIFORM_BUFFER, 0, uboLights)

	drawScreenQuad()

	gl.Disable(gl.BLEND)
	gl.DepthMask(true)
	gl.Enable(gl.DEPTH_TEST)
}
