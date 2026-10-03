package shaders

import (
	"fmt"

	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// DepthLoc represents the location identifiers for shader uniform variables used in depth shaders.
type DepthLoc int

// DepthLocLightSpaceMatrix represents the shader location for the light space matrix in depth shaders.
// DepthLocTexture represents the shader location for the texture in depth shaders.
// DepthLocLast is a sentinel value indicating the last shader depth location.
const (
	DepthLocLightSpaceMatrix = DepthLoc(iota)
	DepthLocTexture
	DepthLocView
	DepthLocTime
	DepthLocLast
)

// DepthMap represents a structure for managing depth framebuffers and textures for shadow mapping and depth rendering.
type DepthMap struct {
	ctx    api.IContext
	fbo    uint32
	tex    uint32
	matrix [16]float32
}

// NewDepthMap creates and returns a new instance of DepthMap with default uninitialized properties.
func NewDepthMap(ctx api.IContext) *DepthMap {
	return &DepthMap{
		ctx: ctx,
		fbo: 0,
		tex: 0,
	}
}

// Update initializes and configures the framebuffer and texture for depth rendering with the given dimensions.
func (d *DepthMap) Update(width, height int32) {
	d.Shutdown()
	var fbo, tex uint32
	borderColor := []float32{1.0, 1.0, 1.0, 1.0}
	d.ctx.GenFramebuffers(1, &fbo)
	d.ctx.GenTextures(1, &tex)
	d.ctx.BindTexture(api.TEXTURE_2D, tex)
	d.ctx.TexImage2D(api.TEXTURE_2D, 0, api.DEPTH_COMPONENT32F, width, height, 0, api.DEPTH_COMPONENT, api.FLOAT, nil)
	d.ctx.TexParameteri(api.TEXTURE_2D, api.TEXTURE_MIN_FILTER, api.LINEAR)
	d.ctx.TexParameteri(api.TEXTURE_2D, api.TEXTURE_MAG_FILTER, api.LINEAR)
	d.ctx.TexParameteri(api.TEXTURE_2D, api.TEXTURE_COMPARE_MODE, api.COMPARE_REF_TO_TEXTURE)
	d.ctx.TexParameteri(api.TEXTURE_2D, api.TEXTURE_COMPARE_FUNC, api.LEQUAL)

	d.ctx.TexParameteri(api.TEXTURE_2D, api.TEXTURE_WRAP_S, api.CLAMP_TO_BORDER)
	d.ctx.TexParameteri(api.TEXTURE_2D, api.TEXTURE_WRAP_T, api.CLAMP_TO_BORDER)
	d.ctx.TexParameterfv(api.TEXTURE_2D, api.TEXTURE_BORDER_COLOR, &borderColor[0])

	d.ctx.BindFramebuffer(api.FRAMEBUFFER, fbo)
	d.ctx.FramebufferTexture2D(api.FRAMEBUFFER, api.DEPTH_ATTACHMENT, api.TEXTURE_2D, tex, 0)
	d.ctx.DrawBuffer(api.NONE)
	d.ctx.ReadBuffer(api.NONE)
	d.ctx.BindFramebuffer(api.FRAMEBUFFER, 0)
	d.fbo = fbo
	d.tex = tex
}

// SetMatrix assigns a 4x4 transformation matrix to the DepthMap instance.
func (d *DepthMap) SetMatrix(matrix [16]float32) {
	d.matrix = matrix
}

// Shutdown releases OpenGL resources associated with the framebuffer and texture of the DepthMap.
func (d *DepthMap) Shutdown() {
	if d.fbo != 0 {
		d.ctx.DeleteFramebuffers(1, &d.fbo)
		return
	}
	if d.tex != 0 {
		d.ctx.DeleteTextures(1, &d.tex)
	}
}

// Depth is responsible for managing depth shaders and shadow map framebuffers for rendering depth-based effects.
type Depth struct {
	ctx              api.IContext
	prg              uint32
	table            [DepthLocLast]int32
	sWidth           int32
	sHeight          int32
	roomMap          *DepthMap
	flashMap         *DepthMap
	shadowLights     []*DepthMap
	viewMatrix       [16]float32
	shadows          bool
	metrics          *MapMetrics
	shadowLightCount uint32
}

// NewDepth initializes and returns a new instance of Depth with default uninitialized properties.
func NewDepth(ctx api.IContext, m *MapMetrics, shadowLights int) *Depth {
	d := &Depth{
		ctx:              ctx,
		metrics:          m,
		roomMap:          NewDepthMap(ctx),
		flashMap:         NewDepthMap(ctx),
		shadowLightCount: 0,
	}
	for i := 0; i < shadowLights; i++ {
		d.shadowLights = append(d.shadowLights, NewDepthMap(ctx))
	}
	return d
}

// SetupSamplers initializes or configures the sampler bindings for the Depth program.
func (s *Depth) SetupSamplers() error {
	s.ctx.UseProgram(s.prg)
	diffuseUnits := []int32{0, 1, 2, 3}
	s.ctx.Uniform1iv(s.GetUniform(DepthLocTexture), 4, &diffuseUnits[0])
	return nil
}

// Init initializes the Depth instance by setting up necessary resources and ensuring its readiness for rendering.
func (s *Depth) Init() error {
	return nil
}

// EnableShadows toggles shadow rendering by setting the internal shadows flag to the provided boolean value.
func (s *Depth) EnableShadows(e bool) {
	s.shadows = e
}

// GetProgram retrieves the OpenGL program ID associated with the Depth instance.
func (s *Depth) GetProgram() uint32 {
	return s.prg
}

// GetUniform retrieves the uniform location corresponding to the provided DepthLoc identifier from the uniform table.
func (s *Depth) GetUniform(id DepthLoc) int32 {
	return s.table[id]
}

// GetRoomShadowTextures retrieves the texture ID associated with the room shadow map.
func (s *Depth) GetRoomShadowTextures() uint32 {
	return s.roomMap.tex
}

// GetShadowLightCount returns the count of dynamic shadow lights currently managed by the Depth instance.
func (s *Depth) GetShadowLightCount() uint32 {
	return s.shadowLightCount
}

// GetFlashShadowTextures retrieves the OpenGL texture ID associated with the flashlight's shadow map.
func (s *Depth) GetFlashShadowTextures() uint32 {
	return s.flashMap.tex
}

// GetShadowLightTextures retrieves the texture ID for a specific dynamic shadow light by its index. Returns 0 if index is out of range.
func (s *Depth) GetShadowLightTextures(idx uint32) (uint32, uint32, [16]float32) {
	if idx >= s.shadowLightCount {
		return 0, 0, [16]float32{}
	}
	return s.shadowLights[idx].tex, s.shadowLights[idx].fbo, s.shadowLights[idx].matrix
}

// Compile initializes and compiles the shader program using vertex and fragment sources, and sets up uniform locations.
func (s *Depth) Compile(assets IAssets) error {
	const vertId = "depth.vert"
	const fragId = "depth.frag"
	vertexSrc, fragmentSrc, err := assets.ReadMulti(vertId, fragId)

	//s.roomShadowFbo, s.roomShadowTex = s.createDepthMap(s.shadowWidth, s.shadowHeight)
	//s.flashShadowFbo, s.flashShadowTex = s.createDepthMap(s.shadowWidth, s.shadowHeight)

	vertexShader, err := ShaderCompile(s.ctx, vertId, string(vertexSrc), api.VERTEX_SHADER)
	if err != nil {
		return err
	}
	fragmentShader, err := ShaderCompile(s.ctx, fragId, string(fragmentSrc), api.FRAGMENT_SHADER)
	if err != nil {
		s.ctx.DeleteShader(vertexShader)
		return err
	}
	s.prg, err = ShaderCreateProgram(s.ctx, "depth", vertexShader, fragmentShader)
	if err != nil {
		return err
	}
	s.table[DepthLocLightSpaceMatrix] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_lightSpaceMatrix\x00"))
	s.table[DepthLocTime] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_time\x00"))
	s.table[DepthLocTexture] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_texture\x00"))
	s.table[DepthLocView] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_view\x00"))

	for idx, v := range s.table {
		if v < 0 {
			return fmt.Errorf("invalid uniform location in depth: %d", idx)
		}
	}
	return nil
}

// UpdateUniforms updates the uniform matrix values for room and flashlight space transformations for the shader.
func (s *Depth) UpdateUniforms(roomSpaceMatrix [16]float32, flashSpaceMatrix [16]float32, viewMatrix [16]float32, dynaLight [][16]float32, dynaLightCount uint32) {
	s.viewMatrix = viewMatrix
	s.roomMap.SetMatrix(roomSpaceMatrix)
	s.flashMap.SetMatrix(flashSpaceMatrix)
	s.shadowLightCount = dynaLightCount //len(dynaLight))
	if s.shadowLightCount >= uint32(len(s.shadowLights)) {
		s.shadowLightCount = uint32(len(s.shadowLights)) - 1
	}
	for x := uint32(0); x < s.shadowLightCount; x++ {
		s.shadowLights[x].SetMatrix(dynaLight[x])
	}
}

// Render performs the depth pre-pass for shadow mapping by rendering the scene to multiple framebuffers for shadows.
func (s *Depth) Render(renderScene func(), mainVao uint32, fbw, fbh int32) {
	if !s.shadows {
		return
	}
	sWidth, sHeight := s.metrics.GetShadowSize()
	if sWidth != s.sWidth || sHeight != s.sHeight {
		s.allocate(sWidth, sHeight)
	}
	s.ctx.BindVertexArray(mainVao)

	s.ctx.Disable(api.CULL_FACE)
	s.ctx.Enable(api.POLYGON_OFFSET_FILL)

	// Attiviamo il clamp della profondità.
	// Impedisce che la geometria sparisca dalla mappa delle ombre
	// quando la telecamera ci finisce letteralmente addosso.
	s.ctx.Enable(api.DEPTH_CLAMP)

	s.ctx.Viewport(0, 0, sWidth, sHeight)

	// OMBRE STANZA (Ortografica)
	s.ctx.PolygonOffset(2.0, 4.0)
	s.ctx.BindFramebuffer(api.FRAMEBUFFER, s.roomMap.fbo)
	s.ctx.Clear(api.DEPTH_BUFFER_BIT)
	s.ctx.UseProgram(s.GetProgram())
	s.ctx.Uniform1f(s.GetUniform(DepthLocTime), float32(textures.GlobalTick())*0.05)
	// Invia la View Matrix del Player per i calcoli del Billboard degli Sprite
	s.ctx.UniformMatrix4fv(s.GetUniform(DepthLocView), 1, false, &s.viewMatrix[0])

	// ROOM
	s.ctx.UniformMatrix4fv(s.GetUniform(DepthLocLightSpaceMatrix), 1, false, &s.roomMap.matrix[0])
	//s.ctx.Uniform1i(s.GetUniform(DepthLocTexture), 0)
	renderScene()

	// OMBRE TORCIA (Prospettica)
	s.ctx.PolygonOffset(0.5, 1.0)
	s.ctx.BindFramebuffer(api.FRAMEBUFFER, s.flashMap.fbo)
	s.ctx.Clear(api.DEPTH_BUFFER_BIT)
	s.ctx.UniformMatrix4fv(s.GetUniform(DepthLocLightSpaceMatrix), 1, false, &s.flashMap.matrix[0])
	renderScene()

	for x := 0; x < int(s.shadowLightCount); x++ {
		s.ctx.PolygonOffset(0.5, 1.0)
		s.ctx.BindFramebuffer(api.FRAMEBUFFER, s.shadowLights[x].fbo)
		s.ctx.Clear(api.DEPTH_BUFFER_BIT)
		s.ctx.UniformMatrix4fv(s.GetUniform(DepthLocLightSpaceMatrix), 1, false, &s.shadowLights[x].matrix[0])
		renderScene()
	}

	// Ripristiniamo lo stato di default per non influenzare il resto del rendering
	s.ctx.Disable(api.DEPTH_CLAMP)
	s.ctx.Disable(api.POLYGON_OFFSET_FILL)
	s.ctx.Viewport(0, 0, fbw, fbh)
}

// allocate configures the internal shadow map dimensions and updates the associated depth maps for rendering.
func (s *Depth) allocate(width, height int32) {
	s.sWidth = width
	s.sHeight = height
	// roomShadowFbo e roomShadowTex gestiscono le ombre delle luci ambientali.
	s.roomMap.Update(s.sWidth, s.sHeight)
	// flashShadowFbo e flashShadowTex gestiscono l'ombra della torcia (ShadowLight).
	s.flashMap.Update(s.sWidth, s.sHeight)
	for _, sl := range s.shadowLights {
		sl.Update(s.sWidth, s.sHeight)
	}
}
