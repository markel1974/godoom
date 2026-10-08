package shaders

import (
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
)

// ShaderOccLoc represents uniform locations for shader program configuration in occlusion rendering contexts.
type ShaderOccLoc int

// ShaderOccLocProjection represents the projection matrix location in the shader.
// ShaderOccLocView represents the view matrix location in the shader.
// ShaderOccLocLast marks the end of ShaderOccLoc values.
const (
	ShaderOccLocProjection = ShaderOccLoc(iota)
	ShaderOccLocView
	ShaderOccLocLast
)

// Occlusion represents a shader-based rendering system for managing occlusion queries in a graphics pipeline.
// It encapsulates context, compiled program ID, and uniform location mappings for shader operations.
type Occlusion struct {
	ctx   api.IContext
	prg   uint32
	table [ShaderOccLocLast]int32
}

// NewOcclusion creates and initializes a new Occlusion instance using the provided graphics context.
func NewOcclusion(ctx api.IContext) *Occlusion {
	return &Occlusion{
		ctx: ctx,
		prg: 0,
	}
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

	return nil
}

// Init initializes the Occlusion instance and prepares it for use. Returns an error if initialization fails.
func (s *Occlusion) Init() error {
	return nil
}

// Render executes occlusion queries using provided view and projection matrices and a render callback.
func (s *Occlusion) Render(renderQueries func(), viewMatrixPtr, projMatrixPtr *float32) {
	s.ctx.ColorMask(false, false, false, false)
	s.ctx.DepthMask(false)
	s.ctx.DepthFunc(api.LEQUAL)

	s.ctx.UseProgram(s.prg)
	s.ctx.UniformMatrix4fv(s.GetUniform(ShaderOccLocView), 1, false, viewMatrixPtr)
	s.ctx.UniformMatrix4fv(s.GetUniform(ShaderOccLocProjection), 1, false, projMatrixPtr)

	renderQueries()

	s.ctx.ColorMask(true, true, true, true)
	s.ctx.DepthMask(true)
	s.ctx.DepthFunc(api.LESS)
}
