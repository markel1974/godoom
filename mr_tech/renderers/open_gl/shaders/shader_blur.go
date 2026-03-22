package shaders

import (
	"fmt"

	"github.com/go-gl/gl/v3.3-core/gl"
)

// ShaderBlurLoc is an enumerated type representing uniform locations for the ShaderBlur GLSL program.
type ShaderBlurLoc int

// ShaderBlurLocSSAOInput specifies the location for the SSAO input in the blur shader.
// ShaderBlurLocLast marks the last location enumerator for the blur shader.
const (
	ShaderBlurLocSSAOInput = ShaderBlurLoc(iota)
	ShaderBlurLocLast
)

// ShaderBlur represents a shader program specifically designed for performing blur effects in the rendering pipeline.
type ShaderBlur struct {
	prg   uint32
	table [ShaderBlurLocLast]int32
}

// NewShaderBlur initializes and returns a new instance of ShaderBlur with default uninitialized properties.
func NewShaderBlur() *ShaderBlur {
	return &ShaderBlur{
		prg: 0,
	}
}

// Setup initializes the ShaderBlur instance with specified render dimensions.
func (s *ShaderBlur) Setup(width int32, height int32) {
}

// GetProgram retrieves the program ID associated with the ShaderBlur instance.
func (s *ShaderBlur) GetProgram() uint32 {
	return s.prg
}

// GetUniform retrieves the uniform location associated with the given ShaderBlurLoc identifier from the shader's uniform table.
func (s *ShaderBlur) GetUniform(id ShaderBlurLoc) int32 {
	return s.table[id]
}

// SetupSamplers binds the blur shader program and sets up the sampler for the SSAO input texture.
func (s *ShaderBlur) SetupSamplers() {
	// Setup Blur Sampler
	gl.UseProgram(s.prg)
	gl.Uniform1i(s.GetUniform(ShaderBlurLocSSAOInput), 0)
}

// Compile compiles the necessary shaders, links them into a program, and assigns uniform locations for the ShaderBlur instance.
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
