package shaders

import (
	"fmt"

	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
)

// ShaderSkyLoc represents the location index of shader uniforms specific to the Sky renderer.
type ShaderSkyLoc int

// ShaderSkyLocProjection represents the projection matrix location for the sky shader.
// ShaderSkyLocView represents the view matrix location for the sky shader.
// ShaderSkyLocSky represents the sky texture location for the sky shader.
// ShaderSkyLocLast represents the total count of sky shader locations.
const (
	ShaderSkyLocProjection = ShaderSkyLoc(iota)
	ShaderSkyLocView
	ShaderSkyLocSky
	ShaderSkyLocSkyLayer
	ShaderSkyLocScrollU
	ShaderSkyLocScrollV
	ShaderSkyLocLast
)

// Sky is a type that manages the state and rendering of the sky shader in a graphics application.
type Sky struct {
	ctx    api.IContext
	prg    uint32
	table  [ShaderSkyLocLast]int32
	skyVAO uint32
	skyVBO uint32
	view   [16]float32
	proj   [16]float32
}

// NewSky creates and returns a new instance of Sky with default uninitialized properties.
func NewSky(ctx api.IContext) *Sky {
	return &Sky{
		ctx: ctx,
		prg: 0,
	}
}

// SetupSamplers initializes the sky vertex array and buffer objects and prepares the sampler for rendering sky elements.
func (s *Sky) SetupSamplers() error {
	s.ctx.UseProgram(s.prg)

	skyUnits := []int32{0, 1, 2, 3}
	s.ctx.Uniform1iv(s.GetUniform(ShaderSkyLocSky), 4, &skyUnits[0])

	s.ctx.GenVertexArrays(1, &s.skyVAO)
	s.ctx.BindVertexArray(s.skyVAO)
	s.ctx.GenBuffers(1, &s.skyVBO)
	s.ctx.BindBuffer(api.ARRAY_BUFFER, s.skyVBO)
	skyQuadVertices := []float32{-1.0, -1.0, 1.0, -1.0, -1.0, 1.0, 1.0, 1.0}
	s.ctx.BufferData(api.ARRAY_BUFFER, len(skyQuadVertices)*4, s.ctx.Ptr(skyQuadVertices), api.STATIC_DRAW)
	s.ctx.VertexAttribPointer(0, 2, api.FLOAT, false, 2*4, s.ctx.PtrOffset(0))
	s.ctx.EnableVertexAttribArray(0)
	// Restore default state
	s.ctx.Enable(api.DEPTH_TEST)
	s.ctx.DepthFunc(api.LEQUAL)
	return nil
}

// Init initializes the Sky instance by setting up necessary resources and ensuring its readiness for rendering.
func (s *Sky) Init() error {
	return nil
}

// GetProgram returns the OpenGL program identifier associated with the Sky instance.
func (s *Sky) GetProgram() uint32 {
	return s.prg
}

// GetUniform retrieves the location of a shader uniform variable by its ID.
func (s *Sky) GetUniform(id ShaderSkyLoc) int32 {
	return s.table[id]
}

// UpdateUniforms updates the view and projection matrices for the shader.
func (s *Sky) UpdateUniforms(view, proj [16]float32) {
	s.view = view
	s.proj = proj
}

// GetVAO retrieves the vertex array object (VAO) associated with the Sky instance.
func (s *Sky) GetVAO() uint32 {
	return s.skyVAO
}

// Compile compiles the shader program for rendering the sky using provided vertex and fragment shaders from assets.
func (s *Sky) Compile(a IAssets) error {
	const vertId = "sky.vert"
	const fragId = "sky.frag"

	vertexSrc, fragmentSrc, err := a.ReadMulti(vertId, fragId)
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
	s.prg, err = ShaderCreateProgram(s.ctx, "sky", vertexShader, fragmentShader)
	if err != nil {
		return err
	}
	s.table[ShaderSkyLocProjection] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_projection\x00"))
	s.table[ShaderSkyLocView] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_view\x00"))
	s.table[ShaderSkyLocSky] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_sky\x00"))
	s.table[ShaderSkyLocSkyLayer] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_skyLayer\x00"))
	s.table[ShaderSkyLocScrollU] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_scrollU\x00"))
	s.table[ShaderSkyLocScrollV] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_scrollV\x00"))
	for idx, v := range s.table {
		if v < 0 {
			return fmt.Errorf("invalid uniform location in sky: %d", idx)
		}
	}
	return nil
}

// Render handles the rendering of the sky by setting up shaders, texture bindings, and drawing the vertex array.
func (s *Sky) Render(skyLayer float32, skyEnabled bool, u, v float32) {
	if !skyEnabled {
		return
	}
	s.ctx.UseProgram(s.GetProgram())

	s.ctx.DepthFunc(api.LEQUAL)
	s.ctx.DepthMask(false)

	s.ctx.UniformMatrix4fv(s.GetUniform(ShaderSkyLocProjection), 1, false, &s.proj[0])
	s.ctx.UniformMatrix4fv(s.GetUniform(ShaderSkyLocView), 1, false, &s.view[0])

	s.ctx.BindVertexArray(s.skyVAO)

	s.ctx.Uniform1f(s.GetUniform(ShaderSkyLocSkyLayer), skyLayer)
	s.ctx.Uniform1f(s.GetUniform(ShaderSkyLocScrollU), u)
	s.ctx.Uniform1f(s.GetUniform(ShaderSkyLocScrollV), v)

	s.ctx.DrawArrays(api.TRIANGLE_STRIP, 0, 4)

	s.ctx.DepthMask(true)
	s.ctx.DepthFunc(api.LESS)
}
