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

// Geometry represents a shader program used for geometry rendering in a graphics pipeline.
// This type holds the OpenGL program ID, uniform locations, and state for view and projection matrices.
type Geometry struct {
	prg   uint32
	table [GeometryLocLast]int32
	view  [16]float32
	proj  [16]float32
}

// NewGeometry initializes and returns a new Geometry instance with default properties.
func NewGeometry() *Geometry {
	return &Geometry{
		prg: 0,
	}
}

// Setup initializes the Geometry with the specified viewport width and height.
func (s *Geometry) Setup(width int32, height int32) error {
	return nil
}

// SetupSamplers initializes and configures texture samplers for the Geometry instance.
func (s *Geometry) SetupSamplers() error {
	return nil
}

// Init initializes the Geometry instance and prepares it for use. It ensures necessary internal state is configured.
func (s *Geometry) Init() error {
	return nil
}

// GetProgram returns the OpenGL program ID associated with the shader.
func (s *Geometry) GetProgram() uint32 {
	return s.prg
}

// GetUniform retrieves the uniform location associated with the given GeometryLoc identifier.
func (s *Geometry) GetUniform(id GeometryLoc) int32 {
	return s.table[id]
}

// Compile initializes and compiles shaders, links them into a program, sets uniform locations, and validates the program.
func (s *Geometry) Compile(assets IAssets) error {
	const vertId = "main.vert"
	const fragId = "geometry.frag"

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
	s.prg, err = ShaderCreateProgram("geometry", vertexShader, fragmentShader)
	if err != nil {
		return err
	}
	s.table[GeometryLocTexture] = gl.GetUniformLocation(s.prg, gl.Str("u_texture\x00"))
	s.table[GeometryLocView] = gl.GetUniformLocation(s.prg, gl.Str("u_view\x00"))
	s.table[GeometryLocProjection] = gl.GetUniformLocation(s.prg, gl.Str("u_projection\x00"))
	for idx, v := range s.table {
		if v < 0 {
			return fmt.Errorf("invalid uniform location in geometry: %d", idx)
		}
	}
	return nil
}

// UpdateUniforms updates the view and projection matrices used by the shader with the given values.
func (s *Geometry) UpdateUniforms(view, proj [16]float32) {
	s.view = view
	s.proj = proj
}

// Render applies shader program, updates uniform values, and executes the provided rendering function.
func (s *Geometry) Render(renderScene func()) {
	gl.UseProgram(s.GetProgram())
	gl.Uniform1i(s.GetUniform(GeometryLocTexture), 0)
	gl.UniformMatrix4fv(s.GetUniform(GeometryLocView), 1, false, &s.view[0])
	gl.UniformMatrix4fv(s.GetUniform(GeometryLocProjection), 1, false, &s.proj[0])
	renderScene()
}
