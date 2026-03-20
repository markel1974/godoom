package open_gl

import (
	"fmt"

	"github.com/go-gl/gl/v3.3-core/gl"
)

type ShaderMainLoc int

const (
	ShaderMainLocView = ShaderMainLoc(iota)
	ShaderMainLocProj
	ShaderMainLocAmbientLight
	ShaderMainLocProjection
	ShaderMainLocScreenResolution
	ShaderMainLocFlashDir
	ShaderMainLocFlashIntensityFactor
	ShaderMainLocFlashConeStart
	ShaderMainLocFlashConeEnd
	ShaderMainLocTexture
	ShaderMainLocNormalMap
	ShaderMainLocSSAO
	ShaderMainLocLast
)

type ShaderMain struct {
	prg     uint32
	table   [ShaderMainLocLast]int32
	mainVao uint32
	mainVbo uint32
}

func NewShaderMain() *ShaderMain {
	return &ShaderMain{
		prg: 0,
	}
}

func (s *ShaderMain) Init() {
	gl.GenVertexArrays(1, &s.mainVao)
	gl.BindVertexArray(s.mainVao)
	gl.GenBuffers(1, &s.mainVbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, s.mainVbo)
	gl.BufferData(gl.ARRAY_BUFFER, vboMaxFloats*4, nil, gl.DYNAMIC_DRAW)
}

func (s *ShaderMain) Setup(width int32, height int32) {
}

func (s *ShaderMain) GetProgram() uint32 {
	return s.prg
}

func (s *ShaderMain) GetUniform(id ShaderMainLoc) int32 {
	return s.table[id]
}

func (s *ShaderMain) Compile(vertexSrc string, fragmentSrc string) error {
	vertexShader, err := ShaderCompile(vertexSrc, gl.VERTEX_SHADER)
	if err != nil {
		return err
	}
	fragmentShader, err := ShaderCompile(fragmentSrc, gl.FRAGMENT_SHADER)
	if err != nil {
		gl.DeleteShader(vertexShader)
		return err
	}
	s.prg, err = ShaderCreateProgram(vertexShader, fragmentShader)
	if err != nil {
		return err
	}
	s.table[ShaderMainLocView] = gl.GetUniformLocation(s.prg, gl.Str("u_view\x00"))
	s.table[ShaderMainLocProj] = gl.GetUniformLocation(s.prg, gl.Str("u_projection\x00"))
	s.table[ShaderMainLocAmbientLight] = gl.GetUniformLocation(s.prg, gl.Str("u_ambient_light\x00"))
	s.table[ShaderMainLocProjection] = gl.GetUniformLocation(s.prg, gl.Str("u_projection\x00"))
	s.table[ShaderMainLocScreenResolution] = gl.GetUniformLocation(s.prg, gl.Str("u_screenResolution\x00"))
	s.table[ShaderMainLocFlashDir] = gl.GetUniformLocation(s.prg, gl.Str("u_flashDir\x00"))
	s.table[ShaderMainLocFlashIntensityFactor] = gl.GetUniformLocation(s.prg, gl.Str("u_flashIntensityFactor\x00"))
	s.table[ShaderMainLocFlashConeStart] = gl.GetUniformLocation(s.prg, gl.Str("u_flashConeStart\x00"))
	s.table[ShaderMainLocFlashConeEnd] = gl.GetUniformLocation(s.prg, gl.Str("u_flashConeEnd\x00"))
	s.table[ShaderMainLocTexture] = gl.GetUniformLocation(s.prg, gl.Str("u_texture\x00"))
	s.table[ShaderMainLocNormalMap] = gl.GetUniformLocation(s.prg, gl.Str("u_normalMap\x00"))
	s.table[ShaderMainLocSSAO] = gl.GetUniformLocation(s.prg, gl.Str("u_ssao\x00"))
	for _, v := range s.table {
		if v < 0 {
			return fmt.Errorf("invalid uniform location: %d", v)
		}
	}
	return nil
}
