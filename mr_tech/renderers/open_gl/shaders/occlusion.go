package shaders

import (
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/metrics"
)

// ShaderOccLoc represents uniform locations for shader program configuration in occlusion rendering contexts.
type ShaderOccLoc int

// ShaderOccLocProjection represents the projection matrix location in the shader.
// ShaderOccLocView represents the view matrix location in the shader.
// ShaderOccLocDebug represents the debug mode toggle location in the shader.
// ShaderOccLocLast marks the end of ShaderOccLoc values.
const (
	ShaderOccLocProjection = ShaderOccLoc(iota)
	ShaderOccLocView
	ShaderOccLocDebug
	ShaderOccLocLast
)

// Occlusion represents a shader-based rendering system for managing occlusion queries in a graphics pipeline.
// It encapsulates context, compiled program ID, and uniform location mappings for shader operations.
type Occlusion struct {
	ctx        api.IContext
	prg        uint32
	table      [ShaderOccLocLast]int32
	metrics    *metrics.Map
	debugBoxes int
}

// NewOcclusion creates and initializes a new Occlusion instance using the provided graphics context.
func NewOcclusion() *Occlusion {
	return &Occlusion{
		prg:        0,
		debugBoxes: 0,
	}
}

// Setup initializes the Occlusion instance by assigning the rendering context from the provided Handler.
func (s *Occlusion) Setup(h *Handler) error {
	s.ctx = h.GetContext()
	s.metrics = h.GetMapMetrics()
	return nil
}

// SetupSamplers initializes the texture samplers required for the occlusion shader and binds them to the appropriate locations.
func (s *Occlusion) SetupSamplers() error {
	return nil
}

// GetProgram retrieves the OpenGL program identifier associated with the Occlusion instance.
func (s *Occlusion) GetProgram() uint32 {
	return s.prg
}

// GetUniform retrieves the uniform location for the given shader uniform identifier from the internal table.
func (s *Occlusion) GetUniform(u ShaderOccLoc) int32 {
	return s.table[u]
}

// Compile initializes the shader program for occlusion rendering by loading, compiling, and linking vertex and fragment shaders.
func (s *Occlusion) Compile(a IAssets) error {
	const vertId = "main_occlusion.vert"
	const fragId = "main_occlusion.frag"

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

	s.prg, err = ShaderCreateProgram(s.ctx, "occlusion", vertexShader, fragmentShader)
	if err != nil {
		return err
	}

	s.table[ShaderOccLocView] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_view\x00"))
	s.table[ShaderOccLocProjection] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_projection\x00"))
	s.table[ShaderOccLocDebug] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_debug\x00"))

	return nil
}

// SetDebug enables or disables the visual wireframe debug mode for occlusion bounding boxes.
func (s *Occlusion) SetDebug(debug bool) {
	if debug {
		s.debugBoxes = 1
	} else {
		s.debugBoxes = 0
	}
}

// GetDebug reports whether visual wireframe debug mode is enabled.
func (s *Occlusion) GetDebug() bool {
	return s.debugBoxes != 0
}

// Init initializes the Occlusion instance and prepares it for use. Returns an error if initialization fails.
func (s *Occlusion) Init() error {
	return nil
}

// Render executes occlusion queries using provided view and projection matrices and a render callback.
func (s *Occlusion) Render(renderQueries func()) {
	view := s.metrics.GetViewPtr()
	proj := s.metrics.GetProjPtr()

	if s.debugBoxes != 0 {
		s.ctx.ColorMask(true, true, true, true)
		s.ctx.Enable(api.BLEND)
		s.ctx.BlendFunc(api.SRC_ALPHA, api.ONE_MINUS_SRC_ALPHA)
		s.ctx.Disable(api.CULL_FACE)
	} else {
		s.ctx.ColorMask(false, false, false, false)
		s.ctx.Enable(api.CULL_FACE)
	}
	s.ctx.DepthMask(false)
	s.ctx.DepthFunc(api.LEQUAL)

	s.ctx.UseProgram(s.prg)
	s.ctx.UniformMatrix4fv(s.GetUniform(ShaderOccLocView), 1, false, view)
	s.ctx.UniformMatrix4fv(s.GetUniform(ShaderOccLocProjection), 1, false, proj)
	s.ctx.Uniform1i(s.GetUniform(ShaderOccLocDebug), int32(s.debugBoxes))

	renderQueries()

	if s.debugBoxes != 0 {
		s.ctx.Disable(api.BLEND)
	}
	s.ctx.Disable(api.CULL_FACE)
	s.ctx.ColorMask(true, true, true, true)
	s.ctx.DepthMask(true)
	s.ctx.DepthFunc(api.LESS)
}
