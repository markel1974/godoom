package shaders

import (
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/metrics"
)

// ShaderAdditiveLoc represents uniform locations used in additive shader programs.
type ShaderAdditiveLoc int

// ShaderAdditiveLocProjection specifies the location index for the projection matrix in additive shader programs.
// ShaderAdditiveLocView specifies the location index for the view matrix in additive shader programs.
// ShaderAdditiveLocTime specifies the location index for the time parameter in additive shader programs.
// ShaderAdditiveLocTexture specifies the location index for the texture sampler in additive shader programs.
// ShaderAdditiveLocLast represents the last location index in additive shader programs.
const (
	ShaderAdditiveLocProjection = ShaderAdditiveLoc(iota)
	ShaderAdditiveLocView
	ShaderAdditiveLocTime
	ShaderAdditiveLocTexture
	ShaderAdditiveLocLast
)

// Additive represents a shader-based rendering effect supporting additive blending for dynamic rendering.
type Additive struct {
	ctx   api.IContext
	prg   uint32
	table [ShaderAdditiveLocLast]int32

	main    *Main
	metrics *metrics.Map
}

// NewAdditive creates and returns a new instance of Additive initialized with the provided graphics context.
func NewAdditive() *Additive {
	return &Additive{
		prg: 0,
	}
}

// Setup initializes the Additive instance by assigning the rendering context from the provided Handler.
func (s *Additive) Setup(h *Handler) error {
	s.ctx = h.GetContext()
	s.main = h.GetMain()
	s.metrics = h.GetMapMetrics()
	return nil
}

// Init initializes the Additive object, preparing it for rendering operations. Returns an error if initialization fails.
func (s *Additive) Init() error {
	return nil
}

// SetupSamplers assigns texture sampler units to the shader program and binds them to the appropriate uniform locations.
func (s *Additive) SetupSamplers() error {
	s.ctx.UseProgram(s.prg)
	diffuseUnits := [4]int32{api.TexUnitDiffuseBase, api.TexUnitDiffuseBase + 1, api.TexUnitDiffuseBase + 2, api.TexUnitDiffuseBase + 3}
	s.ctx.Uniform1iv(s.GetUniform(ShaderAdditiveLocTexture), 4, &diffuseUnits[0])
	return nil
}

// GetProgram returns the current shader program identifier stored in the Additive instance.
func (s *Additive) GetProgram() uint32 {
	return s.prg
}

// GetUniform retrieves the OpenGL uniform location for the specified ShaderAdditiveLoc from the cached table.
func (s *Additive) GetUniform(u ShaderAdditiveLoc) int32 {
	return s.table[u]
}

// Compile initializes and compiles the shaders, creates the shader program, and retrieves uniform locations.
func (s *Additive) Compile(a IAssets) error {
	const vertId = "main.vert"
	const fragId = "main_additive.frag"

	vertSrc, fragSrc, err := a.ReadMulti(vertId, fragId)
	if err != nil {
		return err
	}

	vertexShader, err := ShaderCompile(s.ctx, vertId, string(vertSrc), api.VERTEX_SHADER)
	if err != nil {
		return err
	}

	fragmentShader, err := ShaderCompile(s.ctx, fragId, string(fragSrc), api.FRAGMENT_SHADER)
	if err != nil {
		return err
	}

	s.prg, err = ShaderCreateProgram(s.ctx, "main_additive", vertexShader, fragmentShader)
	if err != nil {
		return err
	}

	s.table[ShaderAdditiveLocView] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_view\x00"))
	s.table[ShaderAdditiveLocProjection] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_projection\x00"))
	s.table[ShaderAdditiveLocTime] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_time\x00"))
	s.table[ShaderAdditiveLocTexture] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_texture\x00"))

	return nil
}

// RenderAdditive applies additive blending and manages OpenGL state for rendering with given geometry and matrices.
func (s *Additive) RenderAdditive(renderGeometry func()) {
	view := s.metrics.GetViewPtr()
	proj := s.metrics.GetProjPtr()

	vao := s.main.GetVAO()

	s.ctx.UseProgram(s.prg)

	interval := float32(0) //unused
	s.ctx.UniformMatrix4fv(s.GetUniform(ShaderAdditiveLocView), 1, false, view)
	s.ctx.UniformMatrix4fv(s.GetUniform(ShaderAdditiveLocProjection), 1, false, proj)
	s.ctx.Uniform1f(s.GetUniform(ShaderAdditiveLocTime), interval)

	s.ctx.DepthMask(false)
	s.ctx.Enable(api.DEPTH_TEST)
	s.ctx.DepthFunc(api.LEQUAL)
	s.ctx.Enable(api.BLEND)
	s.ctx.BlendFunc(api.ONE, api.ONE)

	s.ctx.BindVertexArray(vao)

	renderGeometry()

	s.ctx.Disable(api.BLEND)
	s.ctx.Enable(api.DEPTH_TEST)
	s.ctx.DepthFunc(api.LESS)
	s.ctx.DepthMask(true)
}
