package shaders

import (
	"fmt"

	"github.com/go-gl/gl/v3.3-core/gl"
)

// BlurLoc is an enumerated type representing uniform locations for the Blur GLSL program.
type BlurLoc int

// BlurLocSSAOInput specifies the location for the SSAO input in the blur shader.
// BlurLocLast marks the last location enumerator for the blur shader.
const (
	BlurLocSSAOInput = BlurLoc(iota)
	BlurLocLast
)

// Blur represents a shader program specifically designed for performing blur effects in the rendering pipeline.
type Blur struct {
	prg   uint32
	table [BlurLocLast]int32
}

// NewBlur initializes and returns a new instance of Blur with default uninitialized properties.
func NewBlur() *Blur {
	return &Blur{
		prg: 0,
	}
}

// Init initializes the Blur instance by setting up necessary resources and ensuring its readiness for rendering.
func (s *Blur) Init() error {
	return nil
}

// GetProgram retrieves the program ID associated with the Blur instance.
func (s *Blur) GetProgram() uint32 {
	return s.prg
}

// GetUniform retrieves the uniform location associated with the given BlurLoc identifier from the shader's uniform table.
func (s *Blur) GetUniform(id BlurLoc) int32 {
	return s.table[id]
}

// SetupSamplers binds the blur shader program and sets up the sampler for the SSAO input texture.
func (s *Blur) SetupSamplers() error {
	gl.UseProgram(s.prg)
	gl.Uniform1i(s.GetUniform(BlurLocSSAOInput), 0)
	return nil
}

// Compile compiles the necessary shaders, links them into a program, and assigns uniform locations for the Blur instance.
func (s *Blur) Compile(assets IAssets) error {
	const vertId = "ssao.vert"
	const fragId = "ssao_blur.frag"
	vertexSrc, fragmentSrc, err := assets.ReadMulti(vertId, fragId)
	if err != nil {
		return err
	}
	vertexShader, err := ShaderCompile(vertId, string(vertexSrc), gl.VERTEX_SHADER)
	if err != nil {
		return err
	}
	fragmentShader, err := ShaderCompile(fragId, string(fragmentSrc), gl.FRAGMENT_SHADER)
	if err != nil {
		gl.DeleteShader(vertexShader)
		return err
	}
	s.prg, err = ShaderCreateProgram("blur", vertexShader, fragmentShader)
	if err != nil {
		return err
	}
	s.table[BlurLocSSAOInput] = gl.GetUniformLocation(s.prg, gl.Str("ssaoInput\x00"))
	for idx, v := range s.table {
		if v < 0 {
			return fmt.Errorf("invalid uniform location in blur: %d", idx)
		}
	}
	return nil
}
