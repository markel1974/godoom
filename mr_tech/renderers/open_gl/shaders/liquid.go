package shaders

import (
	"math"
	"unsafe"

	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/metrics"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// LiquidType represents the type of liquid medium being rendered by the Liquid shader.
type LiquidType int32

const (
	LiquidTypeWater LiquidType = 0 // Acqua limpida e trasparente (default)
	LiquidTypeSea   LiquidType = 1 // Mare oceanico (con onde ampie e schiuma su creste e riva)
	LiquidTypeLava  LiquidType = 2 // Lava incandescente opaca con vene di magma
	LiquidTypeSlime LiquidType = 3 // Slime / Nukage acido radioattivo torbido
	LiquidTypeBlood LiquidType = 4 // Sangue denso e viscoso
	LiquidTypeFog   LiquidType = 5 // Nebbia volumetrica atmosferica
)

// ShaderLiquidLoc represents an enumerator for shader uniform locations specific to the Liquid shader system.
type ShaderLiquidLoc int

// ShaderLiquidLocProjection represents the location for the liquid shader's projection matrix.
// ShaderLiquidLocView represents the location for the liquid shader's view matrix.
// ShaderLiquidLocTime represents the location for the liquid shader's time uniform.
// ShaderLiquidLocScreenResolution represents the location for the liquid shader's screen resolution uniform.
// ShaderLiquidLocNear represents the location for the liquid shader's near clipping plane distance.
// ShaderLiquidLocFar represents the location for the liquid shader's far clipping plane distance.
// ShaderLiquidLocTexture represents the location for the liquid shader's texture sampler array.
// ShaderLiquidLocLightDir represents the location for the liquid shader's view-space light direction uniform.
// ShaderLiquidLocLiquidType represents the location for the liquid type uniform.
// ShaderLiquidLocLast signifies the last valid location for the liquid shader.
const (
	ShaderLiquidLocProjection = ShaderLiquidLoc(iota)
	ShaderLiquidLocView
	ShaderLiquidLocTime
	ShaderLiquidLocScreenResolution
	ShaderLiquidLocNear
	ShaderLiquidLocFar
	ShaderLiquidLocTexture
	ShaderLiquidLocLightDir
	ShaderLiquidLocLiquidType
	ShaderLiquidLocLast
)

// Liquid represents a shader program for rendering liquid effects within a 3D graphical context.
type Liquid struct {
	ctx   api.IContext
	prg   uint32
	table [ShaderLiquidLocLast]int32

	handler    *Handler
	main       *Main
	post       *Post
	metrics    *metrics.Map
	liquidType LiquidType
}

// NewLiquid initializes and returns a new Liquid instance with the provided graphics context.
func NewLiquid() *Liquid {
	return &Liquid{
		prg:        0,
		liquidType: LiquidTypeWater,
	}
}

// SetLiquidType sets the active liquid rendering type (Water, Sea, Lava, Slime, Blood, Fog).
func (s *Liquid) SetLiquidType(t LiquidType) {
	s.liquidType = t
}

// GetLiquidType returns the currently set liquid type.
func (s *Liquid) GetLiquidType() LiquidType {
	return s.liquidType
}

// Setup initializes the Lights instance by assigning the rendering context from the provided Handler.
func (s *Liquid) Setup(h *Handler) error {
	s.handler = h
	s.ctx = h.GetContext()
	s.main = h.GetMain()
	s.post = h.GetPost()
	s.metrics = h.GetMapMetrics()
	return nil
}

// SetupSamplers binds texture samplers to specific uniform locations in the shader program.
func (s *Liquid) SetupSamplers() error {
	s.ctx.UseProgram(s.prg)

	diffuseUnits := [4]int32{api.TexUnitDiffuseBase, api.TexUnitDiffuseBase + 1, api.TexUnitDiffuseBase + 2, api.TexUnitDiffuseBase + 3}
	s.ctx.Uniform1iv(s.GetUniform(ShaderLiquidLocTexture), 4, &diffuseUnits[0])
	locRefraction := s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_refractionTex\x00"))
	if locRefraction != -1 {
		s.ctx.Uniform1i(locRefraction, api.TexUnitRefraction)
	}

	locDepth := s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_depthTex\x00"))
	if locDepth != -1 {
		s.ctx.Uniform1i(locDepth, api.TexUnitShadowMap)
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
	s.table[ShaderLiquidLocLightDir] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_lightDir\x00"))
	s.table[ShaderLiquidLocLiquidType] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_liquidType\x00"))

	return nil
}

// Init initializes the Liquid shader, setting up necessary state or resources. Returns an error if initialization fails.
func (s *Liquid) Init() error {
	return nil
}

// Render handles the rendering logic for the liquid, applying shaders, setting up textures, and drawing geometry based on parameters.
func (s *Liquid) Render(renderGeometry func(), transparent bool) {
	vao := s.main.GetVAO()
	refractionTex := s.post.GetColorBuffer()
	depthTex := s.post.GetDepthBuffer()
	view := s.metrics.GetViewPtr()
	proj := s.metrics.GetProjPtr()
	fbW, fbH := s.handler.GetSize()

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
	s.ctx.Enable(api.DEPTH_TEST)
	s.ctx.DepthFunc(api.LEQUAL)

	s.ctx.UniformMatrix4fv(s.GetUniform(ShaderLiquidLocView), 1, false, view)
	s.ctx.UniformMatrix4fv(s.GetUniform(ShaderLiquidLocProjection), 1, false, proj)
	s.ctx.Uniform1f(s.GetUniform(ShaderLiquidLocTime), interval)
	s.ctx.Uniform2f(s.GetUniform(ShaderLiquidLocScreenResolution), float32(fbW), float32(fbH))
	s.ctx.Uniform1f(s.GetUniform(ShaderLiquidLocNear), s.metrics.GetRoomZNear())
	s.ctx.Uniform1f(s.GetUniform(ShaderLiquidLocFar), s.metrics.GetRoomZFar())
	s.ctx.Uniform1i(s.GetUniform(ShaderLiquidLocLiquidType), 10) //int32(s.liquidType))

	if loc := s.GetUniform(ShaderLiquidLocLightDir); loc != -1 {
		const (
			normLx float32 = 0.2977
			normLy float32 = 0.8437
			normLz float32 = 0.4467
		)
		vSlice := unsafe.Slice(view, 16)
		lvX := vSlice[0]*normLx + vSlice[4]*normLy + vSlice[8]*normLz
		lvY := vSlice[1]*normLx + vSlice[5]*normLy + vSlice[9]*normLz
		lvZ := vSlice[2]*normLx + vSlice[6]*normLy + vSlice[10]*normLz
		lenSq := lvX*lvX + lvY*lvY + lvZ*lvZ
		if lenSq > 1e-6 {
			invLen := float32(1.0 / math.Sqrt(float64(lenSq)))
			s.ctx.Uniform3f(loc, lvX*invLen, lvY*invLen, lvZ*invLen)
		} else {
			s.ctx.Uniform3f(loc, 0.35, 0.85, 0.40)
		}
	}

	s.ctx.BindVertexArray(vao)

	s.ctx.ActiveTexture(api.TEXTURE0 + api.TexUnitRefraction)
	s.ctx.BindTexture(api.TEXTURE_2D, refractionTex)
	s.ctx.ActiveTexture(api.TEXTURE0 + api.TexUnitShadowMap)
	s.ctx.BindTexture(api.TEXTURE_2D, depthTex)

	renderGeometry()

	// UNBIND TEXTURES TO PREVENT FEEDBACK LOOPS IN POST RESOLVE
	s.ctx.ActiveTexture(api.TEXTURE0 + api.TexUnitRefraction)
	s.ctx.BindTexture(api.TEXTURE_2D, 0)
	s.ctx.ActiveTexture(api.TEXTURE0 + api.TexUnitShadowMap)
	s.ctx.BindTexture(api.TEXTURE_2D, 0)

	s.ctx.Disable(api.BLEND)
	s.ctx.DepthMask(true)
	s.ctx.DepthFunc(api.LESS)
}
