package shaders

import (
	"fmt"

	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
)

// GeometryLoc represents a location identifier for shader geometry uniforms.
type GeometryLoc int

// GeometryLocTexture represents the texture location in shader geometry.
// GeometryLocView represents the view matrix location in shader geometry.
// GeometryLocProjection represents the projection matrix location in shader geometry.
// GeometryLocLast marks the end of the GeometryLoc constants.
const (
	GeometryLocTexture = GeometryLoc(iota)
	GeometryLocNormalMap
	GeometryLocView
	GeometryLocProjection
	GeometryLocLast
)

// Geometry represents a shader program used for geometry rendering in a graphics pipeline.
// This type holds the OpenGL program ID, uniform locations, and state for view and projection matrices.
type Geometry struct {
	ctx   api.IContext
	prg   uint32
	table [GeometryLocLast]int32
	view  [16]float32
	proj  [16]float32
}

// NewGeometry initializes and returns a new Geometry instance with default properties.
func NewGeometry(ctx api.IContext) *Geometry {
	return &Geometry{
		ctx: ctx,
		prg: 0,
	}
}

// SetupSamplers initializes and configures texture samplers for the Geometry instance.
func (s *Geometry) SetupSamplers() error {
	s.ctx.UseProgram(s.prg)
	diffuseUnits := []int32{0, 1, 2, 3}
	normalUnits := []int32{4, 5, 6, 7}
	s.ctx.Uniform1iv(s.GetUniform(GeometryLocTexture), 4, &diffuseUnits[0])
	s.ctx.Uniform1iv(s.GetUniform(GeometryLocNormalMap), 4, &normalUnits[0])
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
	vertexShader, err := ShaderCompile(s.ctx, vertId, string(vertexSrc), api.VERTEX_SHADER)
	if err != nil {
		return err
	}
	fragmentShader, err := ShaderCompile(s.ctx, fragId, string(fragmentSrc), api.FRAGMENT_SHADER)
	if err != nil {
		s.ctx.DeleteShader(vertexShader)
		return err
	}
	s.prg, err = ShaderCreateProgram(s.ctx, "geometry", vertexShader, fragmentShader)
	if err != nil {
		return err
	}
	s.table[GeometryLocTexture] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_texture\x00"))
	s.table[GeometryLocNormalMap] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_normalMap\x00"))
	s.table[GeometryLocView] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_view\x00"))
	s.table[GeometryLocProjection] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_projection\x00"))
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
	s.ctx.UseProgram(s.GetProgram())
	// RIMOSSO: s.ctx.Uniform1i(s.GetUniform(GeometryLocTexture), 0) (Gestito ora da SetupSamplers)
	s.ctx.UniformMatrix4fv(s.GetUniform(GeometryLocView), 1, false, &s.view[0])
	s.ctx.UniformMatrix4fv(s.GetUniform(GeometryLocProjection), 1, false, &s.proj[0])
	renderScene()
}
