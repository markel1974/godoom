package shaders

import (
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
)

type ShaderAdditiveLoc int

const (
	ShaderAdditiveLocProjection = ShaderAdditiveLoc(iota)
	ShaderAdditiveLocView
	ShaderAdditiveLocTime
	ShaderAdditiveLocTexture
	ShaderAdditiveLocLast
)

type Additive struct {
	ctx   api.IContext
	prg   uint32
	table [ShaderAdditiveLocLast]int32
}

func NewAdditive(ctx api.IContext) *Additive {
	return &Additive{
		ctx: ctx,
		prg: 0,
	}
}

func (s *Additive) SetupSamplers() error {
	s.ctx.UseProgram(s.prg)
	diffuseUnits := [4]int32{0, 1, 2, 3}
	s.ctx.Uniform1iv(s.GetUniform(ShaderAdditiveLocTexture), 4, &diffuseUnits[0])
	return nil
}

func (s *Additive) GetProgram() uint32 {
	return s.prg
}

func (s *Additive) GetUniform(u ShaderAdditiveLoc) int32 {
	return s.table[u]
}

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

func (s *Additive) Init() error {
	return nil
}

func (s *Additive) RenderAdditive(renderGeometry func(), vao uint32, viewMatrixPtr, projMatrixPtr *float32) {
	s.ctx.UseProgram(s.prg)

	interval := float32(0) //unused
	s.ctx.UniformMatrix4fv(s.GetUniform(ShaderAdditiveLocView), 1, false, viewMatrixPtr)
	s.ctx.UniformMatrix4fv(s.GetUniform(ShaderAdditiveLocProjection), 1, false, projMatrixPtr)
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
