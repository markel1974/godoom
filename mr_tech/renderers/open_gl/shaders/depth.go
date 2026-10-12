package shaders

import (
	"fmt"

	"github.com/markel1974/godoom/mr_tech/model"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/metrics"
	"github.com/markel1974/godoom/mr_tech/textures"
)

/*
Y=2048 +------------------------+------------------------+
       |                        |                        |
       |      ROOM SHADOW       |    FLASHLIGHT SHADOW   |
       |      (1024 x 1024)     |      (1024 x 1024)     |
       |   UV: [0..0.5, 0.5..1] |  UV: [0.5..1, 0.5..1]  |
       |                        |                        |
Y=1024 +--------+--------+------+--+--------+------------+
       | Light4 | Light5 |  Light6 | Light7 | (512x512)  |  Row 1 (Y: 512..1023)
 Y=512 +--------+--------+---------+--------+------------+
       | Light0 | Light1 |  Light2 | Light3 | (512x512)  |  Row 0 (Y: 0..511)
   Y=0 +--------+--------+---------+--------+------------+
       X=0     X=512   X=1024    X=1536   X=2048
*/

/*
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
	ctx            api.IContext
	fbo            uint32
	tex            uint32
	matrixPtr      *float32
	emptyMatrix    [16]float32
	emptyMatrixPtr *float32
}

// NewDepthMap creates and returns a new instance of DepthMap with default uninitialized properties.
func NewDepthMap(ctx api.IContext) *DepthMap {
	dm := &DepthMap{
		ctx:       ctx,
		fbo:       0,
		tex:       0,
		matrixPtr: nil,
	}
	dm.emptyMatrixPtr = &dm.emptyMatrix[0]
	dm.matrixPtr = dm.emptyMatrixPtr
	return dm
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
func (d *DepthMap) SetMatrix(matrix *float32) {
	d.matrixPtr = matrix
}

// Shutdown releases OpenGL resources associated with the framebuffer and texture of the DepthMap.
func (d *DepthMap) Shutdown() {
	if d.fbo != 0 {
		d.ctx.DeleteFramebuffers(1, &d.fbo)
		d.fbo = 0
	}
	if d.tex != 0 {
		d.ctx.DeleteTextures(1, &d.tex)
		d.tex = 0
	}
}

// Depth is responsible for managing depth shaders and shadow map framebuffers for rendering depth-based effects.
type Depth struct {
	ctx                 api.IContext
	handler             *Handler
	prg                 uint32
	table               [DepthLocLast]int32
	sWidth              int32
	sHeight             int32
	shadowAtlasMap      *DepthMap
	shadows             bool
	mapMetrics          *metrics.Map
	flashlightMetrics   *metrics.Flashlight
	shadowLightsMetrics *metrics.ShadowLights
	flash               *model.Flash
	matrixEmpty         [16]float32
	matrixEmptyPtr      *float32
	viewMatrixPtr       *float32
	roomSpaceMatrix     *float32
	flashSpaceMatrix    *float32

	maxShadowLights int
	main            *Main
}

// NewDepth initializes and returns a new instance of Depth with default uninitialized properties.
func NewDepth(maxShadowLights int) *Depth {
	d := &Depth{
		maxShadowLights: maxShadowLights,
	}

	return d
}

// Setup initializes the Depth instance by assigning the rendering context from the provided Handler.
func (s *Depth) Setup(h *Handler) error {
	s.handler = h
	s.ctx = h.GetContext()
	s.mapMetrics = h.GetMapMetrics()
	s.flashlightMetrics = h.GetFlashlightMetrics()
	s.shadowLightsMetrics = h.GetShadowLightsMetrics()
	s.flash = h.GetFlash()
	s.main = h.GetMain()
	s.shadowAtlasMap = NewDepthMap(s.ctx)
	s.matrixEmptyPtr = &s.matrixEmpty[0]
	s.viewMatrixPtr = s.matrixEmptyPtr
	s.roomSpaceMatrix = s.matrixEmptyPtr
	s.flashSpaceMatrix = s.matrixEmptyPtr
	return nil
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

// GetRoomShadowTextures retrieves the texture ID associated with the room shadow map (Master Shadow Atlas).
func (s *Depth) GetRoomShadowTextures() uint32 {
	return s.shadowAtlasMap.tex
}

// GetShadowLightCount returns the count of dynamic shadow lights currently managed by the Depth instance.
func (s *Depth) GetShadowLightCount() uint32 {
	return s.shadowLightsMetrics.GetLightCount()
}

// GetFlashShadowTextures retrieves the OpenGL texture ID associated with the flashlight shadow map (Master Shadow Atlas).
func (s *Depth) GetFlashShadowTextures() uint32 {
	return s.shadowAtlasMap.tex
}

// GetShadowAtlasTexture retrieves the OpenGL texture ID of the master shadow atlas.
func (s *Depth) GetShadowAtlasTexture() uint32 {
	return s.shadowAtlasMap.tex
}

// GetShadowLightTextures retrieves the texture ID for a specific dynamic shadow light by its index. Returns 0 if index is out of range.
func (s *Depth) GetShadowLightTextures(idx int) (uint32, uint32, *float32) {
	_, lMatrix := s.shadowLightsMetrics.GetLight(idx)
	return s.shadowAtlasMap.tex, s.shadowAtlasMap.fbo, lMatrix
}

// Compile initializes and compiles the shader program using vertex and fragment sources, and sets up uniform locations.
func (s *Depth) Compile(assets IAssets) error {
	const vertId = "depth.vert"
	const fragId = "depth.frag"
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

// Prepare updates the uniform matrix values for room and flashlight space transformations for the shader.
func (s *Depth) Prepare() {
	s.roomSpaceMatrix = s.mapMetrics.GetRoomSpacePtr()
	s.flashSpaceMatrix = s.flashlightMetrics.GetShadowSpacePtr()
	s.viewMatrixPtr = s.mapMetrics.GetViewPtr()
	//_, shadowLightCount := s.shadowLightsMetrics.GetShadowLightMatrices()
	//s.shadowLightCount = shadowLightCount
}

// Render performs the depth render into the unified master shadow atlas.
func (s *Depth) Render(renderScene func()) {
	if !s.shadows {
		return
	}
	fbW, fbH := s.handler.GetSize()
	mainVao := s.main.GetVAO()
	const atlasSize = int32(2048)
	if s.sWidth != atlasSize || s.sHeight != atlasSize {
		s.allocate(atlasSize, atlasSize)
	}
	s.ctx.BindVertexArray(mainVao)

	s.ctx.Disable(api.CULL_FACE)
	s.ctx.Enable(api.POLYGON_OFFSET_FILL)
	s.ctx.Enable(api.DEPTH_CLAMP)

	// Bind single master shadow atlas FBO and clear entire depth atlas in one go
	s.ctx.BindFramebuffer(api.FRAMEBUFFER, s.shadowAtlasMap.fbo)
	s.ctx.Viewport(0, 0, atlasSize, atlasSize)
	s.ctx.Clear(api.DEPTH_BUFFER_BIT)

	s.ctx.UseProgram(s.GetProgram())
	s.ctx.Uniform1f(s.GetUniform(DepthLocTime), float32(textures.GlobalTick())*0.05)
	// Player view matrix
	s.ctx.UniformMatrix4fv(s.GetUniform(DepthLocView), 1, false, s.viewMatrixPtr)

	// OMBRA STANZA (Ortografica - Tile 1024x1024 in alto a sinistra: X: 0..1024, Y: 1024..2048)
	s.ctx.Viewport(0, 1024, 1024, 1024)
	s.ctx.PolygonOffset(2.0, 4.0)
	s.ctx.UniformMatrix4fv(s.GetUniform(DepthLocLightSpaceMatrix), 1, false, s.roomSpaceMatrix)
	renderScene()

	// OMBRE TORCIA (Prospettica - Tile 1024x1024 in alto a destra: X: 1024..2048, Y: 1024..2048)
	// Renderizza solo se la torcia è effettivamente accesa
	if s.flash != nil && s.flash.GetIntensity() > 0.001 {
		s.ctx.Viewport(1024, 1024, 1024, 1024)
		s.ctx.PolygonOffset(0.5, 1.0)
		s.ctx.UniformMatrix4fv(s.GetUniform(DepthLocLightSpaceMatrix), 1, false, s.flashSpaceMatrix)
		renderScene()
	}

	// OMBRE LUCI DINAMICHE (Fino a 8 Tile da 512x512 nella metà inferiore: Y: 0..1024)

	if shadowLightCount := s.shadowLightsMetrics.GetLightCount(); shadowLightCount > 0 {
		s.ctx.PolygonOffset(0.5, 1.0)
		for x := 0; x < int(shadowLightCount); x++ {
			light, lightMatrix := s.shadowLightsMetrics.GetLight(x)
			if light.Intensity > 0.001 {
				col := int32(x % 4)
				row := int32(x / 4)
				s.ctx.Viewport(col*512, row*512, 512, 512)
				s.ctx.UniformMatrix4fv(s.GetUniform(DepthLocLightSpaceMatrix), 1, false, lightMatrix)
				renderScene()
			}
		}
	}

	// Ripristiniamo lo stato di default per non influenzare il resto del rendering
	s.ctx.Disable(api.DEPTH_CLAMP)
	s.ctx.Disable(api.POLYGON_OFFSET_FILL)
	s.ctx.Viewport(0, 0, fbW, fbH)
}

// allocate configures the unified master shadow map atlas dimensions.
func (s *Depth) allocate(width, height int32) {
	s.sWidth = width
	s.sHeight = height
	s.shadowAtlasMap.Update(width, height)
}

*/

/*
Y=2048 +------------------------+------------------------+
       |                        |                        |
       |      ROOM SHADOW       |    FLASHLIGHT SHADOW   |
       |      (1024 x 1024)     |      (1024 x 1024)     |
       |   UV: [0..0.5, 0.5..1] |  UV: [0.5..1, 0.5..1]  |
       |                        |                        |
Y=1024 +--------+--------+------+--+--------+------------+
       | Light4 | Light5 |  Light6 | Light7 | (512x512)  |  Row 1 (Y: 512..1023)
 Y=512 +--------+--------+---------+--------+------------+
       | Light0 | Light1 |  Light2 | Light3 | (512x512)  |  Row 0 (Y: 0..511)
   Y=0 +--------+--------+---------+--------+------------+
       X=0     X=512   X=1024    X=1536   X=2048
*/

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
	ctx            api.IContext
	fbo            uint32
	tex            uint32
	matrixPtr      *float32
	emptyMatrix    [16]float32
	emptyMatrixPtr *float32
}

// NewDepthMap creates and returns a new instance of DepthMap with default uninitialized properties.
func NewDepthMap(ctx api.IContext) *DepthMap {
	dm := &DepthMap{
		ctx:       ctx,
		fbo:       0,
		tex:       0,
		matrixPtr: nil,
	}
	dm.emptyMatrixPtr = &dm.emptyMatrix[0]
	dm.matrixPtr = dm.emptyMatrixPtr
	return dm
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
func (d *DepthMap) SetMatrix(matrix *float32) {
	d.matrixPtr = matrix
}

// Shutdown releases OpenGL resources associated with the framebuffer and texture of the DepthMap.
func (d *DepthMap) Shutdown() {
	if d.fbo != 0 {
		d.ctx.DeleteFramebuffers(1, &d.fbo)
		d.fbo = 0
	}
	if d.tex != 0 {
		d.ctx.DeleteTextures(1, &d.tex)
		d.tex = 0
	}
}

// Depth is responsible for managing depth shaders and shadow map framebuffers for rendering depth-based effects.
type Depth struct {
	ctx                 api.IContext
	handler             *Handler
	prg                 uint32
	table               [DepthLocLast]int32
	sWidth              int32
	sHeight             int32
	shadowAtlasMap      *DepthMap
	shadows             bool
	mapMetrics          *metrics.Map
	flashlightMetrics   *metrics.Flashlight
	shadowLightsMetrics *metrics.ShadowLights
	flash               *model.Flash
	shadowLightCount    uint32
	matrixEmpty         [16]float32
	matrixEmptyPtr      *float32
	viewMatrixPtr       *float32
	roomSpaceMatrix     *float32
	flashSpaceMatrix    *float32

	maxShadowLights int
	main            *Main
}

// NewDepth initializes and returns a new instance of Depth with default uninitialized properties.
func NewDepth(maxShadowLights int) *Depth {
	d := &Depth{
		maxShadowLights:  maxShadowLights,
		shadowLightCount: 0,
	}

	return d
}

// Setup initializes the Depth instance by assigning the rendering context from the provided Handler.
func (s *Depth) Setup(h *Handler) error {
	s.handler = h
	s.ctx = h.GetContext()
	s.mapMetrics = h.GetMapMetrics()
	s.flashlightMetrics = h.GetFlashlightMetrics()
	s.shadowLightsMetrics = h.GetShadowLightsMetrics()
	s.flash = h.GetFlash()
	s.main = h.GetMain()
	s.shadowAtlasMap = NewDepthMap(s.ctx)
	s.matrixEmptyPtr = &s.matrixEmpty[0]
	s.viewMatrixPtr = s.matrixEmptyPtr
	s.roomSpaceMatrix = s.matrixEmptyPtr
	s.flashSpaceMatrix = s.matrixEmptyPtr
	return nil
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

// GetRoomShadowTextures retrieves the texture ID associated with the room shadow map (Master Shadow Atlas).
func (s *Depth) GetRoomShadowTextures() uint32 {
	return s.shadowAtlasMap.tex
}

// GetShadowLightCount returns the count of dynamic shadow lights currently managed by the Depth instance.
func (s *Depth) GetShadowLightCount() uint32 {
	return s.shadowLightCount
}

// GetFlashShadowTextures retrieves the OpenGL texture ID associated with the flashlight shadow map (Master Shadow Atlas).
func (s *Depth) GetFlashShadowTextures() uint32 {
	return s.shadowAtlasMap.tex
}

// GetShadowAtlasTexture retrieves the OpenGL texture ID of the master shadow atlas.
func (s *Depth) GetShadowAtlasTexture() uint32 {
	return s.shadowAtlasMap.tex
}

// GetShadowLightTextures retrieves the texture ID for a specific dynamic shadow light by its index. Returns 0 if index is out of range.
func (s *Depth) GetShadowLightTextures(idx uint32) (uint32, uint32, *float32) {
	if idx >= s.shadowLightCount {
		return 0, 0, s.matrixEmptyPtr
	}
	return s.shadowAtlasMap.tex, s.shadowAtlasMap.fbo, s.shadowLightsMetrics.GetSpacePtr(idx)
}

// Compile initializes and compiles the shader program using vertex and fragment sources, and sets up uniform locations.
func (s *Depth) Compile(assets IAssets) error {
	const vertId = "depth.vert"
	const fragId = "depth.frag"
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

// Prepare updates the uniform matrix values for room and flashlight space transformations for the shader.
func (s *Depth) Prepare() {
	s.roomSpaceMatrix = s.mapMetrics.GetRoomSpacePtr()
	s.flashSpaceMatrix = s.flashlightMetrics.GetShadowSpacePtr()
	s.viewMatrixPtr = s.mapMetrics.GetViewPtr()
	_, shadowLightCount := s.shadowLightsMetrics.GetShadowLightMatrices()
	s.shadowLightCount = shadowLightCount
}

// Render performs the depth render into the unified master shadow atlas.
func (s *Depth) Render(renderScene func()) {
	if !s.shadows {
		return
	}
	fbW, fbH := s.handler.GetSize()
	mainVao := s.main.GetVAO()
	const atlasSize = int32(2048)
	if s.sWidth != atlasSize || s.sHeight != atlasSize {
		s.allocate(atlasSize, atlasSize)
	}
	s.ctx.BindVertexArray(mainVao)

	s.ctx.Disable(api.CULL_FACE)
	s.ctx.Enable(api.POLYGON_OFFSET_FILL)
	s.ctx.Enable(api.DEPTH_CLAMP)

	// Bind single master shadow atlas FBO and clear entire depth atlas in one go
	s.ctx.BindFramebuffer(api.FRAMEBUFFER, s.shadowAtlasMap.fbo)
	s.ctx.Viewport(0, 0, atlasSize, atlasSize)
	s.ctx.Clear(api.DEPTH_BUFFER_BIT)

	s.ctx.UseProgram(s.GetProgram())
	s.ctx.Uniform1f(s.GetUniform(DepthLocTime), float32(textures.GlobalTick())*0.05)
	// Player view matrix
	s.ctx.UniformMatrix4fv(s.GetUniform(DepthLocView), 1, false, s.viewMatrixPtr)

	// OMBRA STANZA (Ortografica - Tile 1024x1024 in alto a sinistra: X: 0..1024, Y: 1024..2048)
	s.ctx.Viewport(0, 1024, 1024, 1024)
	s.ctx.PolygonOffset(2.0, 4.0)
	s.ctx.UniformMatrix4fv(s.GetUniform(DepthLocLightSpaceMatrix), 1, false, s.roomSpaceMatrix)
	renderScene()

	// OMBRE TORCIA (Prospettica - Tile 1024x1024 in alto a destra: X: 1024..2048, Y: 1024..2048)
	// Renderizza solo se la torcia è effettivamente accesa
	if s.flash != nil && s.flash.GetIntensity() > 0.001 {
		s.ctx.Viewport(1024, 1024, 1024, 1024)
		s.ctx.PolygonOffset(0.5, 1.0)
		s.ctx.UniformMatrix4fv(s.GetUniform(DepthLocLightSpaceMatrix), 1, false, s.flashSpaceMatrix)
		renderScene()
	}

	// OMBRE LUCI DINAMICHE (Fino a 8 Tile da 512x512 nella metà inferiore: Y: 0..1024)
	if s.shadowLightCount > 0 {
		s.ctx.PolygonOffset(0.5, 1.0)
		shadowLights, _ := s.shadowLightsMetrics.GetLights()
		for x := 0; x < int(s.shadowLightCount); x++ {
			if shadowLights[x] != nil && shadowLights[x].Intensity <= 0.001 {
				continue
			}
			col := int32(x % 4)
			row := int32(x / 4)
			s.ctx.Viewport(col*512, row*512, 512, 512)
			s.ctx.UniformMatrix4fv(s.GetUniform(DepthLocLightSpaceMatrix), 1, false, s.shadowLightsMetrics.GetSpacePtr(uint32(x)))
			renderScene()
		}
	}

	// Ripristiniamo lo stato di default per non influenzare il resto del rendering
	s.ctx.Disable(api.DEPTH_CLAMP)
	s.ctx.Disable(api.POLYGON_OFFSET_FILL)
	s.ctx.Viewport(0, 0, fbW, fbH)
}

// allocate configures the unified master shadow map atlas dimensions.
func (s *Depth) allocate(width, height int32) {
	s.sWidth = width
	s.sHeight = height
	s.shadowAtlasMap.Update(width, height)
}
