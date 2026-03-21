package open_gl

import (
	"fmt"

	"github.com/go-gl/gl/v3.3-core/gl"
)

type ShaderGeometryLoc int

const (
	ShaderGeometryLocTexture = ShaderGeometryLoc(iota)
	ShaderGeometryLocView
	ShaderGeometryLocProjection
	ShaderGeometryLocLast
)

type ShaderGeometry struct {
	prg   uint32
	table [ShaderGeometryLocLast]int32
	view  [16]float32
	proj  [16]float32
}

func NewShaderGeometry() *ShaderGeometry {
	return &ShaderGeometry{
		prg: 0,
	}
}

func (s *ShaderGeometry) Setup(width int32, height int32) {
}

func (s *ShaderGeometry) SetupSamplers() {
}

func (s *ShaderGeometry) GetProgram() uint32 {
	return s.prg
}

func (s *ShaderGeometry) GetUniform(id ShaderGeometryLoc) int32 {
	return s.table[id]
}

func (s *ShaderGeometry) Compile(assets IAssets) error {
	vertexSrc, fragmentSrc, err := assets.ReadMulti("main.vert", "geometry.frag")
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
	s.table[ShaderGeometryLocTexture] = gl.GetUniformLocation(s.prg, gl.Str("u_texture\x00"))
	s.table[ShaderGeometryLocView] = gl.GetUniformLocation(s.prg, gl.Str("u_view\x00"))
	s.table[ShaderGeometryLocProjection] = gl.GetUniformLocation(s.prg, gl.Str("u_projection\x00"))
	for _, v := range s.table {
		if v < 0 {
			return fmt.Errorf("invalid uniform location: %d", v)
		}
	}
	return nil
}

func (s *ShaderGeometry) UpdateUniforms(view, proj [16]float32) {
	s.view = view
	s.proj = proj
}

func (s *ShaderGeometry) Render(renderScene func()) {
	gl.UseProgram(s.GetProgram())
	gl.Uniform1i(s.GetUniform(ShaderGeometryLocTexture), 0)
	gl.UniformMatrix4fv(s.GetUniform(ShaderGeometryLocView), 1, false, &s.view[0])
	gl.UniformMatrix4fv(s.GetUniform(ShaderGeometryLocProjection), 1, false, &s.proj[0])
	renderScene()
}
