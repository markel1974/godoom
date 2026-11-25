package shaders

import (
	"fmt"

	"github.com/markel1974/godoom/mr_tech/model/things"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/gpu/draws"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/metrics"
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
	DepthLocInstanceMode
	DepthLocLast
)

const (
	depthDoubleBuffer = 2
	shadowAtlasSize   = int32(2048)

	// ShadowTileRoom is the atlas slot index for the orthographic room shadow tile.
	ShadowTileRoom = 0
	// ShadowTileFlash is the atlas slot index for the perspective flashlight shadow tile.
	ShadowTileFlash = 1
	// ShadowTileLightBase is the starting atlas slot index for dynamic spotlights.
	ShadowTileLightBase = 2
	// MaxShadowTiles is the total number of tiles managed inside the master shadow atlas.
	MaxShadowTiles = 10
)

// ShadowsTileData carries per-light culled draw commands and cache keys for a single atlas tile.
type ShadowsTileData struct {
	Active    bool
	RenderDC  *draws.DrawCommandsRender
	Matrix    [16]float32
	SceneHash uint64
	Reuse     bool
}

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

// Depth is responsible for managing depth shaders and shadow map framebuffers for rendering depth-based effects.
type Depth struct {
	ctx        api.IContext
	handler    *Handler
	prg        uint32
	table      [DepthLocLast]int32
	prgSolid   uint32
	tableSolid [DepthLocLast]int32
	atlas      *Atlas

	shadows             bool
	shadowLights        bool
	mapMetrics          *metrics.Map
	flashlightMetrics   *metrics.Flashlight
	shadowLightsMetrics *metrics.ShadowLights
	flash               *things.Flash
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
	return &Depth{
		maxShadowLights: maxShadowLights,
	}
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
	s.shadowLights = s.handler.GetCalibration().EnableShadowLights
	s.atlas = NewAtlas(s.ctx)
	s.atlas.SetShadows(s.shadows)

	s.matrixEmptyPtr = &s.matrixEmpty[0]
	s.viewMatrixPtr = s.matrixEmptyPtr
	s.roomSpaceMatrix = s.matrixEmptyPtr
	s.flashSpaceMatrix = s.matrixEmptyPtr
	return nil
}

// SetupSamplers initializes or configures the sampler bindings for the Depth program.
func (s *Depth) SetupSamplers() error {
	s.ctx.UseProgram(s.prg)
	diffuseUnits := []int32{api.TexUnitDiffuseBase, api.TexUnitDiffuseBase + 1, api.TexUnitDiffuseBase + 2}
	s.ctx.Uniform1iv(s.GetUniform(DepthLocTexture), 3, &diffuseUnits[0])
	return nil
}

// Init initializes the Depth instance by setting up necessary resources and ensuring its readiness for rendering.
func (s *Depth) Init() error {
	return nil
}

// SetShadowsEnabled toggles shadow rendering and synchronizes the underlying atlas cache state.
func (s *Depth) SetShadowsEnabled(shadows bool) {
	s.shadows = shadows
	if s.atlas != nil {
		s.atlas.SetShadows(shadows)
	}
}

// CanReuseTile reports whether slot in the next target atlas buffer already matches matrix and sceneHash.
func (s *Depth) CanReuseTile(slot int, matrix *[16]float32, sceneHash uint64) bool {
	if !s.shadows || s.atlas == nil {
		return false
	}
	return s.atlas.CanReuseTile(slot, matrix, sceneHash)
}

// InvalidateCache forces all cached shadow atlas tiles across both double-buffers to redraw on next use.
func (s *Depth) InvalidateCache() {
	if s.atlas != nil {
		s.atlas.InvalidateCache()
	}
}

// GetAtlas returns the underlying master shadow Atlas instance managed by Depth.
func (s *Depth) GetAtlas() *Atlas {
	return s.atlas
}

// Shutdown releases GPU resources owned by the Depth shadow atlas.
func (s *Depth) Shutdown() {
	if s.atlas != nil {
		s.atlas.Shutdown()
	}
}

// GetTileConfig returns whether the given tile slot is active this frame, whether it requires DEPTH_CLAMP, and its matrix pointer.
func (s *Depth) GetTileConfig(slot int) (active bool, orthoDepthClamp bool, matPtr *float32) {
	if !s.shadows {
		return false, false, nil
	}
	switch {
	case slot == ShadowTileRoom:
		return true, true, s.roomSpaceMatrix
	case slot == ShadowTileFlash:
		if s.flash == nil || s.flash.GetIntensity() <= 0.001 {
			return false, false, nil
		}
		return true, false, s.flashSpaceMatrix
	case slot >= ShadowTileLightBase && slot < MaxShadowTiles:
		x := slot - ShadowTileLightBase
		if !s.shadowLights || x >= s.maxShadowLights || uint32(x) >= s.shadowLightsMetrics.GetLightCount() {
			return false, false, nil
		}
		light, lMatrix := s.shadowLightsMetrics.GetLight(x)
		if light == nil || lMatrix == nil || light.Intensity <= 0.001 {
			return false, false, nil
		}
		return true, false, lMatrix
	default:
		return false, false, nil
	}
}

// GetProgram retrieves the OpenGL program ID associated with the Depth instance.
func (s *Depth) GetProgram() uint32 {
	return s.prg
}

// GetUniform retrieves the uniform location corresponding to the provided DepthLoc identifier from the uniform table.
func (s *Depth) GetUniform(id DepthLoc) int32 {
	return s.table[id]
}

// GetShadowLightCount returns the count of dynamic shadow lights currently managed by the Depth instance.
func (s *Depth) GetShadowLightCount() uint32 {
	return s.shadowLightsMetrics.GetLightCount()
}

// GetRoomShadowTextures retrieves the texture ID associated with the room shadow map (Master Shadow Atlas).
func (s *Depth) GetRoomShadowTextures() uint32 {
	return s.GetShadowAtlasTexture()
}

// GetFlashShadowTextures retrieves the OpenGL texture ID associated with the flashlight shadow map (Master Shadow Atlas).
func (s *Depth) GetFlashShadowTextures() uint32 {
	return s.GetShadowAtlasTexture()
}

// GetShadowAtlasTexture retrieves the OpenGL texture ID of the master shadow atlas.
func (s *Depth) GetShadowAtlasTexture() uint32 {
	if s.atlas == nil {
		return 0
	}
	return s.atlas.GetShadowAtlasTexture()
}

// GetShadowLightTextures retrieves the texture ID, framebuffer ID, and matrix pointer for a specific dynamic shadow light.
func (s *Depth) GetShadowLightTextures(idx uint32) (uint32, uint32, *float32) {
	_, lMatrix := s.shadowLightsMetrics.GetLight(int(idx))
	if s.atlas == nil {
		return 0, 0, lMatrix
	}
	dm := s.atlas.GetShadowAtlas()
	if dm == nil {
		return 0, 0, lMatrix
	}
	return dm.tex, dm.fbo, lMatrix
}

// Compile initializes and compiles the cutout and solid depth shader programs for shadow atlas rendering.
func (s *Depth) Compile(assets IAssets) error {
	var err error
	s.prg, s.table, err = s.compileDepthProgram(assets, "depth.vert", "depth.frag", "depth", true)
	if err != nil {
		return err
	}
	s.prgSolid, s.tableSolid, err = s.compileDepthProgram(assets, "depth.vert", "zprepass.frag", "depth_solid", false)
	if err != nil {
		return err
	}
	s.InvalidateCache()
	return nil
}

func (s *Depth) compileDepthProgram(assets IAssets, vertID, fragID, progName string, requireTexture bool) (uint32, [DepthLocLast]int32, error) {
	var table [DepthLocLast]int32
	vertexSrcBytes, fragmentSrc, err := assets.ReadMulti(vertID, fragID)
	if err != nil {
		return 0, table, err
	}
	vertexSrc := string(vertexSrcBytes)
	if requireTexture {
		vertexSrc = injectShaderDefine(vertexSrc, "#define ALPHA_CUTOUT 1")
	}

	vertexShader, err := ShaderCompile(s.ctx, vertID, vertexSrc, api.VERTEX_SHADER)
	if err != nil {
		return 0, table, err
	}
	fragmentShader, err := ShaderCompile(s.ctx, fragID, string(fragmentSrc), api.FRAGMENT_SHADER)
	if err != nil {
		s.ctx.DeleteShader(vertexShader)
		return 0, table, err
	}
	prg, err := ShaderCreateProgram(s.ctx, progName, vertexShader, fragmentShader)
	if err != nil {
		return 0, table, err
	}
	table[DepthLocLightSpaceMatrix] = s.ctx.GetUniformLocation(prg, s.ctx.Str("u_lightSpaceMatrix\x00"))
	table[DepthLocTime] = s.ctx.GetUniformLocation(prg, s.ctx.Str("u_time\x00"))
	table[DepthLocTexture] = s.ctx.GetUniformLocation(prg, s.ctx.Str("u_texture\x00"))
	table[DepthLocView] = s.ctx.GetUniformLocation(prg, s.ctx.Str("u_view\x00"))
	table[DepthLocInstanceMode] = s.ctx.GetUniformLocation(prg, s.ctx.Str("u_instanceMode\x00"))

	for idx, v := range table {
		if idx == int(DepthLocTexture) && !requireTexture {
			continue
		}
		if v < 0 {
			return 0, table, fmt.Errorf("invalid uniform location in %s: %d", progName, idx)
		}
	}
	return prg, table, nil
}

// Prepare updates the uniform matrix values for room and flashlight space transformations for the shader.
func (s *Depth) Prepare() {
	s.roomSpaceMatrix = s.mapMetrics.GetRoomSpacePtr()
	s.flashSpaceMatrix = s.flashlightMetrics.GetShadowSpacePtr()
	s.viewMatrixPtr = s.mapMetrics.GetViewPtr()
}

func shadowTileRect(slot int) (x, y, w, h int32, depthClamp bool, polyUnits, polyFactor float32) {
	if slot == ShadowTileRoom {
		return 0, 1024, 1024, 1024, true, 2.0, 4.0
	}
	if slot == ShadowTileFlash {
		return 1024, 1024, 1024, 1024, false, 1.5, 2.0
	}
	idx := slot - ShadowTileLightBase
	col := int32(idx % 4)
	row := int32(idx / 4)
	return col * 512, row * 512, 512, 512, false, 1.5, 2.0
}

// Render performs the depth render into the unified master shadow atlas using one shared command set.
func (s *Depth) Render(dc *draws.DrawCommandsRender) {
	s.RenderWithTiles(dc, nil)
}

// RenderWithTiles renders dirty per-light tiles into the double-buffered master shadow atlas, skipping cached unchanged tiles.
// When tiles is nil, it falls back to rendering fallbackDC across all active tiles.
func (s *Depth) RenderWithTiles(fallbackDC *draws.DrawCommandsRender, tiles *[MaxShadowTiles]ShadowsTileData) {
	if !s.shadows || s.atlas == nil {
		return
	}
	if tiles == nil {
		s.renderSharedAtlas(fallbackDC)
		return
	}

	idx, atlas := s.atlas.Update()
	if atlas == nil || atlas.fbo == 0 {
		return
	}

	var dirty [MaxShadowTiles]bool
	anyDirty := false
	anySolid := false
	anyCutout := false
	for t := range MaxShadowTiles {
		tile := &tiles[t]
		if !tile.Active {
			s.atlas.InvalidateTile(idx, t)
			continue
		}
		if tile.Reuse && s.atlas.IsTileCached(idx, t, tile, atlas.gen) {
			continue
		}
		dirty[t] = true
		anyDirty = true
		if tile.RenderDC != nil {
			if tile.RenderDC.HasSolid() {
				anySolid = true
			}
			if tile.RenderDC.HasCutout() {
				anyCutout = true
			}
		}
	}
	if !anyDirty {
		return
	}

	fbW, fbH := s.handler.GetSize()
	mainVao := s.main.GetVAO()
	s.ctx.BindVertexArray(mainVao)

	s.ctx.Disable(api.CULL_FACE)
	s.ctx.Disable(api.BLEND)
	s.ctx.Enable(api.DEPTH_TEST)
	s.ctx.DepthFunc(api.LESS)
	s.ctx.DepthMask(true)
	s.ctx.Enable(api.POLYGON_OFFSET_FILL)
	s.ctx.Enable(api.SCISSOR_TEST)

	s.ctx.BindFramebuffer(api.FRAMEBUFFER, atlas.fbo)
	for t := range MaxShadowTiles {
		if !dirty[t] {
			continue
		}
		tx, ty, tw, th, _, _, _ := shadowTileRect(t)
		s.ctx.Viewport(tx, ty, tw, th)
		s.ctx.Scissor(tx, ty, tw, th)
		s.ctx.Clear(api.DEPTH_BUFFER_BIT)
		s.atlas.SetTile(idx, t, &tiles[t], atlas.gen)
	}

	interval := float32(textures.GlobalTick()) * 0.05
	if anySolid {
		s.renderTilesPass(s.prgSolid, &s.tableSolid, interval, mainVao, tiles, &dirty, false)
	}
	if anyCutout {
		s.renderTilesPass(s.prg, &s.table, interval, mainVao, tiles, &dirty, true)
	}

	s.ctx.Disable(api.SCISSOR_TEST)
	s.ctx.Disable(api.DEPTH_CLAMP)
	s.ctx.Disable(api.POLYGON_OFFSET_FILL)
	s.ctx.DepthMask(true)
	s.ctx.DepthFunc(api.LESS)
	s.ctx.BindFramebuffer(api.FRAMEBUFFER, 0)
	s.ctx.Viewport(0, 0, fbW, fbH)
}

func (s *Depth) renderSharedAtlas(dc *draws.DrawCommandsRender) {
	if dc == nil || !dc.HasCommands() || s.atlas == nil {
		return
	}

	fbW, fbH := s.handler.GetSize()
	mainVao := s.main.GetVAO()
	idx, atlas := s.atlas.Update()
	if atlas == nil || atlas.fbo == 0 {
		return
	}
	s.atlas.InvalidateTiles(idx)

	s.ctx.BindVertexArray(mainVao)

	s.ctx.Disable(api.CULL_FACE)
	s.ctx.Disable(api.BLEND)
	s.ctx.Enable(api.DEPTH_TEST)
	s.ctx.DepthFunc(api.LESS)
	s.ctx.DepthMask(true)
	s.ctx.Enable(api.POLYGON_OFFSET_FILL)
	s.ctx.Enable(api.SCISSOR_TEST)

	// Bind single master shadow atlas FBO and clear entire depth atlas in one go
	s.ctx.BindFramebuffer(api.FRAMEBUFFER, atlas.fbo)
	s.ctx.Viewport(0, 0, s.atlas.width, s.atlas.height)
	s.ctx.Scissor(0, 0, s.atlas.width, s.atlas.height)
	s.ctx.Clear(api.DEPTH_BUFFER_BIT)

	interval := float32(textures.GlobalTick()) * 0.05
	if dc.HasSolid() {
		dc.SetPassState(mainVao, s.tableSolid[DepthLocInstanceMode])
		s.renderAtlasPass(s.prgSolid, &s.tableSolid, interval, dc, false)
	}
	if dc.HasCutout() {
		dc.SetPassState(mainVao, s.table[DepthLocInstanceMode])
		s.renderAtlasPass(s.prg, &s.table, interval, dc, true)
	}

	// Ripristiniamo lo stato di default per non influenzare il resto del rendering
	s.ctx.Disable(api.SCISSOR_TEST)
	s.ctx.Disable(api.DEPTH_CLAMP)
	s.ctx.Disable(api.POLYGON_OFFSET_FILL)
	s.ctx.DepthMask(true)
	s.ctx.DepthFunc(api.LESS)
	s.ctx.BindFramebuffer(api.FRAMEBUFFER, 0)
	s.ctx.Viewport(0, 0, fbW, fbH)
}

func (s *Depth) renderTilesPass(prg uint32, table *[DepthLocLast]int32, interval float32, mainVao uint32, tiles *[MaxShadowTiles]ShadowsTileData, dirty *[MaxShadowTiles]bool, cutout bool) {
	s.ctx.UseProgram(prg)
	s.ctx.Uniform1f(table[DepthLocTime], interval)
	s.ctx.UniformMatrix4fv(table[DepthLocView], 1, false, s.viewMatrixPtr)

	locLightSpace := table[DepthLocLightSpaceMatrix]
	locInstMode := table[DepthLocInstanceMode]

	curDepthClamp := false
	lastPolyUnits := float32(0)
	lastPolyFactor := float32(0)

	for t := range MaxShadowTiles {
		if !dirty[t] {
			continue
		}
		dc := tiles[t].RenderDC
		if dc == nil || (cutout && !dc.HasCutout()) || (!cutout && !dc.HasSolid()) {
			continue
		}
		tx, ty, tw, th, wantClamp, pUnits, pFactor := shadowTileRect(t)
		if wantClamp != curDepthClamp {
			if wantClamp {
				s.ctx.Enable(api.DEPTH_CLAMP)
			} else {
				s.ctx.Disable(api.DEPTH_CLAMP)
			}
			curDepthClamp = wantClamp
		}
		if pUnits != lastPolyUnits || pFactor != lastPolyFactor {
			s.ctx.PolygonOffset(pUnits, pFactor)
			lastPolyUnits = pUnits
			lastPolyFactor = pFactor
		}
		s.ctx.Viewport(tx, ty, tw, th)
		s.ctx.Scissor(tx, ty, tw, th)
		s.ctx.UniformMatrix4fv(locLightSpace, 1, false, &tiles[t].Matrix[0])
		dc.SetPassState(mainVao, locInstMode)
		dc.RenderCutoutFiltered(cutout)
	}
	if curDepthClamp {
		s.ctx.Disable(api.DEPTH_CLAMP)
	}
}

func (s *Depth) renderAtlasPass(prg uint32, table *[DepthLocLast]int32, interval float32, dc *draws.DrawCommandsRender, cutout bool) {
	s.ctx.UseProgram(prg)
	s.ctx.Uniform1f(table[DepthLocTime], interval)
	s.ctx.UniformMatrix4fv(table[DepthLocView], 1, false, s.viewMatrixPtr)

	locLightSpace := table[DepthLocLightSpaceMatrix]

	// OMBRA STANZA (Ortografica - Tile 1024x1024 in alto a sinistra: X: 0..1024, Y: 1024..2048)
	// DEPTH_CLAMP applies only to orthographic room shadows so near roofs are not clipped;
	// perspective tiles below rely on standard near-plane clipping.
	s.ctx.Enable(api.DEPTH_CLAMP)
	s.ctx.Viewport(0, 1024, 1024, 1024)
	s.ctx.Scissor(0, 1024, 1024, 1024)
	s.ctx.PolygonOffset(2.0, 4.0)
	s.ctx.UniformMatrix4fv(locLightSpace, 1, false, s.roomSpaceMatrix)
	dc.RenderCutoutFiltered(cutout)
	s.ctx.Disable(api.DEPTH_CLAMP)

	// OMBRE TORCIA (Prospettica - Tile 1024x1024 in alto a destra: X: 1024..2048, Y: 1024..2048)
	hasFlash := s.flash != nil && s.flash.GetIntensity() > 0.001
	if hasFlash {
		s.ctx.Viewport(1024, 1024, 1024, 1024)
		s.ctx.Scissor(1024, 1024, 1024, 1024)
		s.ctx.PolygonOffset(1.5, 2.0)
		s.ctx.UniformMatrix4fv(locLightSpace, 1, false, s.flashSpaceMatrix)
		dc.RenderCutoutFiltered(cutout)
	}

	// OMBRE LUCI DINAMICHE (Fino a 8 Tile da 512x512 nella metà inferiore: Y: 0..1024)
	if s.shadowLights {
		if shadowLightCount := s.shadowLightsMetrics.GetLightCount(); shadowLightCount > 0 {
			if !hasFlash {
				s.ctx.PolygonOffset(1.5, 2.0)
			}
			for x := 0; x < int(shadowLightCount); x++ {
				shadowLights, lMatrix := s.shadowLightsMetrics.GetLight(x)
				if shadowLights.Intensity <= 0.001 {
					continue
				}
				col := int32(x % 4)
				row := int32(x / 4)
				s.ctx.Viewport(col*512, row*512, 512, 512)
				s.ctx.Scissor(col*512, row*512, 512, 512)
				s.ctx.UniformMatrix4fv(locLightSpace, 1, false, lMatrix)
				dc.RenderCutoutFiltered(cutout)
			}
		}
	}
}
