package shaders

import (
	"fmt"

	"github.com/markel1974/godoom/mr_tech/model/things"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/gpu/draws"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/metrics"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/shaders/atlas"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// DepthLoc identifies a uniform location used by depth and shadow rendering programs.
type DepthLoc int

// DepthLoc constants identify uniform locations used by the depth rendering shader.
const (
	DepthLocLightSpaceMatrix = DepthLoc(iota)
	DepthLocTexture
	DepthLocView
	DepthLocTime
	DepthLocInstanceMode
	DepthLocLast
)

// Depth manages depth rendering resources and shadow-related rendering state.
type Depth struct {
	ctx        api.IContext
	handler    *Handler
	prg        uint32
	table      [DepthLocLast]int32
	prgSolid   uint32
	tableSolid [DepthLocLast]int32
	atlas      *atlas.Atlas

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
	sunCSM          *metrics.SunCSM
}

// NewDepth creates a Depth renderer configured with the maximum number of shadow-casting lights.
func NewDepth(maxShadowLights int) *Depth {
	return &Depth{
		maxShadowLights: maxShadowLights,
	}
}

// Setup initializes depth rendering state from the handler and prepares the atlas and default matrix pointers.
func (s *Depth) Setup(h *Handler) error {
	s.handler = h
	s.ctx = h.GetContext()
	s.mapMetrics = h.GetMapMetrics()
	s.flashlightMetrics = h.GetFlashlightMetrics()
	s.shadowLightsMetrics = h.GetShadowLightsMetrics()
	s.sunCSM = h.GetSunCSM()
	s.flash = h.GetFlash()
	s.main = h.GetMain()
	s.shadowLights = s.handler.GetCalibration().EnableShadowLights
	s.atlas = atlas.NewAtlas(s.ctx)
	s.atlas.SetShadows(s.shadows)

	s.matrixEmptyPtr = &s.matrixEmpty[0]
	s.viewMatrixPtr = s.matrixEmptyPtr
	s.roomSpaceMatrix = s.matrixEmptyPtr
	s.flashSpaceMatrix = s.matrixEmptyPtr
	return nil
}

// SetupSamplers binds the depth program and assigns its texture samplers to diffuse texture units.
func (s *Depth) SetupSamplers() error {
	s.ctx.UseProgram(s.prg)
	diffuseUnits := []int32{api.TexUnitDiffuseBase, api.TexUnitDiffuseBase + 1, api.TexUnitDiffuseBase + 2}
	s.ctx.Uniform1iv(s.GetUniform(DepthLocTexture), 3, &diffuseUnits[0])
	return nil
}

// Init performs no initialization and always returns nil.
func (s *Depth) Init() error {
	return nil
}

// SetShadowsEnabled enables or disables shadow rendering and updates the shadow atlas when initialized.
func (s *Depth) SetShadowsEnabled(shadows bool) {
	s.shadows = shadows
	s.atlas.SetShadows(shadows)
}

// CanReuseTile reports whether the shadow atlas tile for slot can be reused for matrix and sceneHash.
// It returns false when shadows are disabled or no atlas is available.
func (s *Depth) CanReuseTile(slot int, matrix *float32, sceneHash uint64) bool {
	if !s.shadows {
		return false
	}
	return s.atlas.CanReuseTile(slot, matrix, sceneHash)
}

// PeekReusableTile returns the target buffer index, atlas generation, and cached sceneHash
// if slot in the next target atlas buffer already holds a valid render matching matrix.
func (s *Depth) PeekReusableTile(slot int, matPtr *float32) (int, uint64, uint64, bool) {
	if !s.shadows {
		return 0, 0, 0, false
	}
	return s.atlas.PeekReusableTile(slot, matPtr)
}

// TargetInfo returns the next writable double-buffer index, its current atlas generation, and whether its FBO is ready.
func (s *Depth) TargetInfo() (int, uint64, bool) {
	if !s.shadows {
		return 0, 0, false
	}
	return s.atlas.TargetInfo()
}

// InvalidateCache clears cached shadow atlas tiles, if an atlas is available.
func (s *Depth) InvalidateCache() {
	s.atlas.InvalidateCache()
}

// GetAtlas returns the depth shadow atlas managed by this Depth instance.
func (s *Depth) GetAtlas() *atlas.Atlas {
	return s.atlas
}

// Shutdown releases depth shadow atlas resources and resets depth shader state.
func (s *Depth) Shutdown() {
	s.atlas.Shutdown()
	s.shadows = false
	s.prg = 0
	s.prgSolid = 0
}

// GetTileConfig returns the active state, depth-clamp mode, and shadow matrix pointer for an atlas tile slot.
func (s *Depth) GetTileConfig(slot int) (bool, bool, *float32) {
	if !s.shadows {
		return false, false, nil
	}
	useSunCSM := s.sunCSM != nil && s.sunCSM.IsEnabled()
	switch {
	case slot == atlas.ShadowTileRoom:
		if useSunCSM {
			return false, false, nil
		}
		return true, true, s.roomSpaceMatrix
	case slot == atlas.ShadowTileFlash:
		if s.flash == nil || s.flash.GetIntensity() <= 0.001 {
			return false, false, nil
		}
		return true, false, s.flashSpaceMatrix
	case slot >= atlas.ShadowTileLightBase && slot < atlas.ShadowTileSunBase:
		x := slot - atlas.ShadowTileLightBase
		if !s.shadowLights || x >= s.maxShadowLights || uint32(x) >= s.shadowLightsMetrics.GetLightCount() {
			return false, false, nil
		}
		light, lMatrix := s.shadowLightsMetrics.GetLight(x)
		if light == nil || lMatrix == nil || light.Intensity <= 0.001 {
			return false, false, nil
		}
		return true, false, lMatrix
	case slot >= atlas.ShadowTileSunBase && slot < atlas.MaxShadowTiles:
		if !useSunCSM {
			return false, false, nil
		}
		c := slot - atlas.ShadowTileSunBase
		matPtr := s.sunCSM.GetCascadeMatrixPtr(c)
		if matPtr == nil {
			return false, false, nil
		}
		return true, true, matPtr
	default:
		return false, false, nil
	}
}

// GetProgram returns the depth shader program handle.
func (s *Depth) GetProgram() uint32 {
	return s.prg
}

// GetUniform returns the cached uniform location for the specified depth uniform identifier.
func (s *Depth) GetUniform(id DepthLoc) int32 {
	return s.table[id]
}

// GetShadowLightCount returns the number of active shadow lights.
func (s *Depth) GetShadowLightCount() uint32 {
	return s.shadowLightsMetrics.GetLightCount()
}

// GetShadowAtlasTexture returns the shadow atlas texture handle, or 0 if no atlas is available.
func (s *Depth) GetShadowAtlasTexture() uint32 {
	return s.atlas.GetShadowAtlasTexture()
}

// GetShadowLightTextures returns the shadow atlas texture, framebuffer, and light-space matrix for a light.
func (s *Depth) GetShadowLightTextures(idx uint32) (uint32, uint32, *float32) {
	_, lMatrix := s.shadowLightsMetrics.GetLight(int(idx))
	dm := s.atlas.GetShadowAtlas()
	if dm == nil {
		return 0, 0, lMatrix
	}
	return dm.GetTex(), dm.GetFBO(), lMatrix
}

// Compile builds the depth and solid z-prepass shader programs from assets and invalidates cached shadow data.
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

// compileDepthProgram reads depth shader assets, optionally enables alpha cutout, links the program, and resolves uniforms.
// It returns the program ID, uniform location table, or an error if loading, compilation, linking, or validation fails.
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

// Prepare refreshes cached matrix pointers used for depth and shadow rendering.
func (s *Depth) Prepare() {
	s.roomSpaceMatrix = s.mapMetrics.GetRoomSpacePtr()
	s.flashSpaceMatrix = s.flashlightMetrics.GetShadowSpacePtr()
	s.viewMatrixPtr = s.mapMetrics.GetViewPtr()
	width := s.atlas.GetWidth()
	if s.sunCSM != nil && s.atlas != nil && width >= 32 {
		s.sunCSM.SetTileResolution(float32(width / 4))
	}
}

// Render renders the depth pass using the provided draw commands and default shadow tile handling.
func (s *Depth) Render(dc *draws.DrawCommandsRender) {
	s.RenderWithTiles(dc, nil)
}

// RenderWithTiles renders dirty shadow atlas tiles, reusing cached tiles when possible.
// It falls back to rendering the shared atlas when tiles is nil.
func (s *Depth) RenderWithTiles(fallbackDC *draws.DrawCommandsRender, tiles *atlas.TilesHandler) {
	if !s.shadows {
		return
	}
	if tiles == nil {
		s.renderSharedAtlas(fallbackDC)
		return
	}

	idx, atl := s.atlas.Update()
	if atl == nil || atl.GetFBO() == 0 {
		return
	}

	var dirty [atlas.MaxShadowTiles]bool
	anyDirty := false
	anySolid := false
	anyCutout := false
	for t := range atlas.MaxShadowTiles {
		tile := tiles.Get(t)
		if !tile.IsActive() {
			s.atlas.InvalidateTile(idx, t)
			continue
		}
		if tile.CanReuse() && s.atlas.IsTileCached(idx, t, tile, atl.GetGen()) {
			continue
		}
		dirty[t] = true
		anyDirty = true
		renderDC := tile.GetRenderDC()
		if renderDC != nil {
			if renderDC.HasSolid() {
				anySolid = true
			}
			if renderDC.HasCutout() {
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

	s.ctx.BindFramebuffer(api.FRAMEBUFFER, atl.GetFBO())
	for t := range atlas.MaxShadowTiles {
		if !dirty[t] {
			continue
		}
		tx, ty, tw, th, _, _, _ := s.atlas.TileRect(t)
		s.ctx.Viewport(tx, ty, tw, th)
		s.ctx.Scissor(tx, ty, tw, th)
		s.ctx.Clear(api.DEPTH_BUFFER_BIT)
		s.atlas.SetTile(idx, t, tiles.Get(t), atl.GetGen())
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

// renderSharedAtlas renders depth commands into the shared shadow atlas and restores the framebuffer state.
func (s *Depth) renderSharedAtlas(dc *draws.DrawCommandsRender) {
	if dc == nil || !dc.HasCommands() || s.atlas == nil {
		return
	}

	fbW, fbH := s.handler.GetSize()
	mainVao := s.main.GetVAO()
	idx, atl := s.atlas.Update()
	if atl == nil || atl.GetFBO() == 0 {
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
	s.ctx.BindFramebuffer(api.FRAMEBUFFER, atl.GetFBO())
	w, h := s.atlas.GetSize()
	s.ctx.Viewport(0, 0, w, h)
	s.ctx.Scissor(0, 0, w, h)
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

// renderTilesPass renders dirty shadow atlas tiles for either solid or cutout geometry using the given program.
func (s *Depth) renderTilesPass(prg uint32, table *[DepthLocLast]int32, interval float32, mainVao uint32, tiles *atlas.TilesHandler, dirty *[atlas.MaxShadowTiles]bool, cutout bool) {
	s.ctx.UseProgram(prg)
	s.ctx.Uniform1f(table[DepthLocTime], interval)
	s.ctx.UniformMatrix4fv(table[DepthLocView], 1, false, s.viewMatrixPtr)

	locLightSpace := table[DepthLocLightSpaceMatrix]
	locInstMode := table[DepthLocInstanceMode]

	curDepthClamp := false
	lastPolyUnits := float32(0)
	lastPolyFactor := float32(0)

	for tileIdx := range atlas.MaxShadowTiles {
		if !dirty[tileIdx] {
			continue
		}
		tile := tiles.Get(tileIdx)
		dc := tile.GetRenderDC()
		if dc == nil || (cutout && !dc.HasCutout()) || (!cutout && !dc.HasSolid()) {
			continue
		}
		tx, ty, tw, th, wantClamp, pUnits, pFactor := s.atlas.TileRect(tileIdx)
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
		s.ctx.UniformMatrix4fv(locLightSpace, 1, false, tile.GetMatrixPtr())
		dc.SetPassState(mainVao, locInstMode)
		dc.RenderCutoutFiltered(cutout)
	}
	if curDepthClamp {
		s.ctx.Disable(api.DEPTH_CLAMP)
	}
}

// renderAtlasPass renders the shared shadow atlas for room, flashlight, and dynamic light tiles.
// It sets depth shader uniforms, tile viewports, scissors, and light-space matrices per pass.
func (s *Depth) renderAtlasPass(prg uint32, table *[DepthLocLast]int32, interval float32, dc *draws.DrawCommandsRender, cutout bool) {
	s.ctx.UseProgram(prg)
	s.ctx.Uniform1f(table[DepthLocTime], interval)
	s.ctx.UniformMatrix4fv(table[DepthLocView], 1, false, s.viewMatrixPtr)

	locLightSpace := table[DepthLocLightSpaceMatrix]

	useSunCSM := s.sunCSM != nil && s.sunCSM.IsEnabled()
	if useSunCSM {
		s.ctx.Enable(api.DEPTH_CLAMP)
		for c := range metrics.SunCascadeCount {
			rx, ry, rw, rh, _, rpUnits, rpFactor := s.atlas.TileRect(atlas.ShadowTileSunBase + c)
			s.ctx.Viewport(rx, ry, rw, rh)
			s.ctx.Scissor(rx, ry, rw, rh)
			s.ctx.PolygonOffset(rpUnits, rpFactor)
			s.ctx.UniformMatrix4fv(locLightSpace, 1, false, s.sunCSM.GetCascadeMatrixPtr(c))
			dc.RenderCutoutFiltered(cutout)
		}
		s.ctx.Disable(api.DEPTH_CLAMP)
	} else {
		// OMBRA STANZA (Ortografica - quadrant superiore sinistro)
		rx, ry, rw, rh, _, rpUnits, rpFactor := s.atlas.TileRect(atlas.ShadowTileRoom)
		s.ctx.Enable(api.DEPTH_CLAMP)
		s.ctx.Viewport(rx, ry, rw, rh)
		s.ctx.Scissor(rx, ry, rw, rh)
		s.ctx.PolygonOffset(rpUnits, rpFactor)
		s.ctx.UniformMatrix4fv(locLightSpace, 1, false, s.roomSpaceMatrix)
		dc.RenderCutoutFiltered(cutout)
		s.ctx.Disable(api.DEPTH_CLAMP)
	}

	// OMBRE TORCIA (Prospettica - quadrante superiore destro)
	hasFlash := s.flash != nil && s.flash.GetIntensity() > 0.001
	if hasFlash {
		fx, fy, fw, fh, _, fpUnits, fpFactor := s.atlas.TileRect(atlas.ShadowTileFlash)
		s.ctx.Viewport(fx, fy, fw, fh)
		s.ctx.Scissor(fx, fy, fw, fh)
		s.ctx.PolygonOffset(fpUnits, fpFactor)
		s.ctx.UniformMatrix4fv(locLightSpace, 1, false, s.flashSpaceMatrix)
		dc.RenderCutoutFiltered(cutout)
	}

	// OMBRE LUCI DINAMICHE (Fino a 8 tile nella metà inferiore)
	if s.shadowLights {
		if shadowLightCount := s.shadowLightsMetrics.GetLightCount(); shadowLightCount > 0 {
			polyOffsetSet := hasFlash
			for x := 0; x < int(shadowLightCount); x++ {
				shadowLights, lMatrix := s.shadowLightsMetrics.GetLight(x)
				if shadowLights.Intensity <= 0.001 {
					continue
				}
				lx, ly, lw, lh, _, lpUnits, lpFactor := s.atlas.TileRect(atlas.ShadowTileLightBase + x)
				if !polyOffsetSet {
					s.ctx.PolygonOffset(lpUnits, lpFactor)
					polyOffsetSet = true
				}
				s.ctx.Viewport(lx, ly, lw, lh)
				s.ctx.Scissor(lx, ly, lw, lh)
				s.ctx.UniformMatrix4fv(locLightSpace, 1, false, lMatrix)
				dc.RenderCutoutFiltered(cutout)
			}
		}
	}
}
