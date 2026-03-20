package open_gl

import (
	"fmt"

	"github.com/go-gl/gl/v3.3-core/gl"
)

type ShaderSkyLoc int

const (
	ShaderSkyLocProjection = ShaderSkyLoc(iota)
	ShaderSkyLocView
	ShaderSkyLocSky
	ShaderSkyLocLast
)

type ShaderSky struct {
	prg    uint32
	table  [ShaderSkyLocLast]int32
	skyVao uint32
	skyVbo uint32
}

func NewShaderSky() *ShaderSky {
	return &ShaderSky{
		prg: 0,
	}
}

func (s *ShaderSky) SetupSamplers() {
	// sky
	gl.GenVertexArrays(1, &s.skyVao)
	gl.BindVertexArray(s.skyVao)
	gl.GenBuffers(1, &s.skyVbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, s.skyVbo)
	skyQuadVertices := []float32{-1.0, -1.0, 1.0, -1.0, -1.0, 1.0, 1.0, 1.0}
	gl.BufferData(gl.ARRAY_BUFFER, len(skyQuadVertices)*4, gl.Ptr(skyQuadVertices), gl.STATIC_DRAW)
	gl.VertexAttribPointer(0, 2, gl.FLOAT, false, 2*4, gl.PtrOffset(0))
	gl.EnableVertexAttribArray(0)
	// Restore default state
	gl.Enable(gl.DEPTH_TEST)
	gl.DepthFunc(gl.LEQUAL)
}

func (s *ShaderSky) Setup(width int32, height int32) {
}

func (s *ShaderSky) GetProgram() uint32 {
	return s.prg
}

func (s *ShaderSky) GetUniform(id ShaderSkyLoc) int32 {
	return s.table[id]
}

func (s *ShaderSky) Compile(vertexSrc string, fragmentSrc string) error {
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
	s.table[ShaderSkyLocProjection] = gl.GetUniformLocation(s.prg, gl.Str("u_projection\x00"))
	s.table[ShaderSkyLocView] = gl.GetUniformLocation(s.prg, gl.Str("u_view\x00"))
	s.table[ShaderSkyLocSky] = gl.GetUniformLocation(s.prg, gl.Str("u_sky\x00"))
	for _, v := range s.table {
		if v < 0 {
			return fmt.Errorf("invalid uniform location: %d", v)
		}
	}
	return nil
}
