package shaders

import (
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// ShaderLiquidLoc represents an enumerator for shader uniform locations specific to the Liquid shader system.
type ShaderLiquidLoc int

// ShaderLiquidLocProjection represents the location for the liquid shader's projection matrix.
// ShaderLiquidLocView represents the location for the liquid shader's view matrix.
// ShaderLiquidLocTime represents the location for the liquid shader's time uniform.
// ShaderLiquidLocScreenResolution represents the location for the liquid shader's screen resolution uniform.
// ShaderLiquidLocNear represents the location for the liquid shader's near clipping plane distance.
// ShaderLiquidLocFar represents the location for the liquid shader's far clipping plane distance.
// ShaderLiquidLocLast signifies the last valid location for the liquid shader.
const (
	ShaderLiquidLocProjection = ShaderLiquidLoc(iota)
	ShaderLiquidLocView
	ShaderLiquidLocTime
	ShaderLiquidLocScreenResolution
	ShaderLiquidLocNear
	ShaderLiquidLocFar
	ShaderLiquidLocTexture
	ShaderLiquidLocLast
)

// Liquid represents a shader program for rendering liquid effects within a 3D graphical context.
type Liquid struct {
	ctx   api.IContext
	prg   uint32
	table [ShaderLiquidLocLast]int32
}

// NewLiquid initializes and returns a new Liquid instance with the provided graphics context.
func NewLiquid(ctx api.IContext) *Liquid {
	return &Liquid{
		ctx: ctx,
		prg: 0,
	}
}

// SetupSamplers binds texture samplers to specific uniform locations in the shader program.
func (s *Liquid) SetupSamplers() error {
	s.ctx.UseProgram(s.prg)

	diffuseUnits := [4]int32{0, 1, 2, 3}
	s.ctx.Uniform1iv(s.GetUniform(ShaderLiquidLocTexture), 4, &diffuseUnits[0])
	locRefraction := s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_refractionTex\x00"))
	if locRefraction != -1 {
		s.ctx.Uniform1i(locRefraction, 12)
	}

	locDepth := s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_depthTex\x00"))
	if locDepth != -1 {
		s.ctx.Uniform1i(locDepth, 13)
	}

	return nil
}

// GetProgram returns the program ID associated with the Liquid instance.
func (s *Liquid) GetProgram() uint32 {
	return s.prg
}

// GetUniform retrieves the uniform location for the given ShaderLiquidLoc identifier from the internal location table.
func (s *Liquid) GetUniform(u ShaderLiquidLoc) int32 {
	return s.table[u]
}

// Compile initializes and compiles vertex and fragment shaders, creates a shader program, and stores uniform locations.
func (s *Liquid) Compile(a IAssets) error {
	const vertId = "main.vert"
	const fragId = "main_liquid.frag"

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

	s.prg, err = ShaderCreateProgram(s.ctx, "main_liquid", vertexShader, fragmentShader)
	if err != nil {
		return err
	}

	s.table[ShaderLiquidLocView] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_view\x00"))
	s.table[ShaderLiquidLocProjection] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_projection\x00"))
	s.table[ShaderLiquidLocTime] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_time\x00"))
	s.table[ShaderLiquidLocScreenResolution] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_resolution\x00"))
	s.table[ShaderLiquidLocNear] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_near\x00"))
	s.table[ShaderLiquidLocFar] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_far\x00"))
	s.table[ShaderLiquidLocTexture] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_texture\x00"))

	return nil
}

// Init initializes the Liquid shader, setting up necessary state or resources. Returns an error if initialization fails.
func (s *Liquid) Init() error {
	return nil
}

// Render handles the rendering logic for the liquid, applying shaders, setting up textures, and drawing geometry based on parameters.
func (s *Liquid) Render(renderGeometry func(), vao uint32, refractionTex, depthTex uint32, transparent bool, fbW, fbH int32, viewMatrixPtr, projMatrixPtr *float32) {
	s.ctx.UseProgram(s.prg)

	//TODO from config
	const liquidTimeScale = 0.02
	interval := float32(textures.GlobalTick()) * liquidTimeScale
	//TODO from config
	if transparent {
		s.ctx.Enable(api.BLEND)
		s.ctx.BlendFunc(api.SRC_ALPHA, api.ONE_MINUS_SRC_ALPHA)
		s.ctx.DepthMask(false) // Do not write to depth buffer
	} else {
		s.ctx.Disable(api.BLEND)
		s.ctx.DepthMask(true)
	}
	s.ctx.UniformMatrix4fv(s.GetUniform(ShaderLiquidLocView), 1, false, viewMatrixPtr)
	s.ctx.UniformMatrix4fv(s.GetUniform(ShaderLiquidLocProjection), 1, false, projMatrixPtr)
	s.ctx.Uniform1f(s.GetUniform(ShaderLiquidLocTime), interval)
	s.ctx.Uniform2f(s.GetUniform(ShaderLiquidLocScreenResolution), float32(fbW), float32(fbH))
	s.ctx.Uniform1f(s.GetUniform(ShaderLiquidLocNear), 0.1)
	s.ctx.Uniform1f(s.GetUniform(ShaderLiquidLocFar), 1000.0)

	s.ctx.BindVertexArray(vao)

	s.ctx.ActiveTexture(api.TEXTURE12)
	s.ctx.BindTexture(api.TEXTURE_2D, refractionTex)
	s.ctx.ActiveTexture(api.TEXTURE13)
	s.ctx.BindTexture(api.TEXTURE_2D, depthTex)

	renderGeometry()

	// UNBIND TEXTURES TO PREVENT FEEDBACK LOOPS IN POST RESOLVE
	s.ctx.ActiveTexture(api.TEXTURE12)
	s.ctx.BindTexture(api.TEXTURE_2D, 0)
	s.ctx.ActiveTexture(api.TEXTURE13)
	s.ctx.BindTexture(api.TEXTURE_2D, 0)

	s.ctx.Disable(api.BLEND)
	s.ctx.DepthMask(true)
}
