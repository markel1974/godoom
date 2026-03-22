package shaders

import (
	"fmt"

	"github.com/go-gl/gl/v3.3-core/gl"
)

// GeometryLoc represents a location identifier for shader geometry uniforms.
type GeometryLoc int

// GeometryLocTexture represents the texture location in shader geometry.
// GeometryLocView represents the view matrix location in shader geometry.
// GeometryLocProjection represents the projection matrix location in shader geometry.
// GeometryLocLast marks the end of the GeometryLoc constants.
const (
	GeometryLocTexture = GeometryLoc(iota)
	GeometryLocView
	GeometryLocProjection
	GeometryLocLast
)

// ShaderGeometry represents a shader program used for geometry rendering in a graphics pipeline.
// This type holds the OpenGL program ID, uniform locations, and state for view and projection matrices.
type ShaderGeometry struct {
	prg    uint32
	table  [GeometryLocLast]int32
	view   [16]float32
	proj   [16]float32
	width  int32
	height int32
}

// NewShaderGeometry initializes and returns a new ShaderGeometry instance with default properties.
func NewShaderGeometry() *ShaderGeometry {
	return &ShaderGeometry{
		prg: 0,
	}
}

// Setup initializes the ShaderGeometry with the specified viewport width and height.
func (s *ShaderGeometry) Setup(width int32, height int32) {
	s.width = width
	s.height = height
}

// SetupSamplers initializes and configures texture samplers for the ShaderGeometry instance.
func (s *ShaderGeometry) SetupSamplers() {
}

// GetProgram returns the OpenGL program ID associated with the shader.
func (s *ShaderGeometry) GetProgram() uint32 {
	return s.prg
}

// GetUniform retrieves the uniform location associated with the given GeometryLoc identifier.
func (s *ShaderGeometry) GetUniform(id GeometryLoc) int32 {
	return s.table[id]
}

// Compile initializes and compiles shaders, links them into a program, sets uniform locations, and validates the program.
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
	s.table[GeometryLocTexture] = gl.GetUniformLocation(s.prg, gl.Str("u_texture\x00"))
	s.table[GeometryLocView] = gl.GetUniformLocation(s.prg, gl.Str("u_view\x00"))
	s.table[GeometryLocProjection] = gl.GetUniformLocation(s.prg, gl.Str("u_projection\x00"))
	for _, v := range s.table {
		if v < 0 {
			return fmt.Errorf("invalid uniform location: %d", v)
		}
	}
	return nil
}

// UpdateUniforms updates the view and projection matrices used by the shader with the given values.
func (s *ShaderGeometry) UpdateUniforms(view, proj [16]float32) {
	s.view = view
	s.proj = proj
}

// Render applies shader program, updates uniform values, and executes the provided rendering function.
func (s *ShaderGeometry) Render(renderScene func()) {
	gl.UseProgram(s.GetProgram())
	gl.Uniform1i(s.GetUniform(GeometryLocTexture), 0)
	gl.UniformMatrix4fv(s.GetUniform(GeometryLocView), 1, false, &s.view[0])
	gl.UniformMatrix4fv(s.GetUniform(GeometryLocProjection), 1, false, &s.proj[0])
	renderScene()
}
