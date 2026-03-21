package open_gl

import (
	"fmt"

	"github.com/go-gl/gl/v3.3-core/gl"
)

type ShaderBlurLoc int

const (
	ShaderBlurLocSSAOInput = ShaderBlurLoc(iota)
	ShaderBlurLocLast
)

type ShaderBlur struct {
	prg   uint32
	table [ShaderBlurLocLast]int32
}

func NewShaderBlur() *ShaderBlur {
	return &ShaderBlur{
		prg: 0,
	}
}

func (s *ShaderBlur) Setup(width int32, height int32) {
}

func (s *ShaderBlur) GetProgram() uint32 {
	return s.prg
}

func (s *ShaderBlur) GetUniform(id ShaderBlurLoc) int32 {
	return s.table[id]
}

func (s *ShaderBlur) SetupSamplers() {
	// Setup Blur Sampler
	gl.UseProgram(s.prg)
	gl.Uniform1i(s.GetUniform(ShaderBlurLocSSAOInput), 0)
}

func (s *ShaderBlur) Compile(assets IAssets) error {
	vertexSrc, fragmentSrc, err := assets.ReadMulti("ssao.vert", "ssao_blur.frag")
	if err != nil {
		return err
	}
	vertexShader, err := ShaderCompile(string(vertexSrc), gl.VERTEX_SHADER)
	if err != nil {
		return err
	}
	fragmentShader, err := ShaderCompile(string(fragmentSrc), gl.FRAGMENT_SHADER)
	if err != nil {
		gl.DeleteShader(vertexShader)
		return err
	}
	s.prg, err = ShaderCreateProgram(vertexShader, fragmentShader)
	if err != nil {
		return err
	}
	s.table[ShaderBlurLocSSAOInput] = gl.GetUniformLocation(s.prg, gl.Str("ssaoInput\x00"))
	for _, v := range s.table {
		if v < 0 {
			return fmt.Errorf("invalid uniform location: %d", v)
		}
	}
	return nil
}
