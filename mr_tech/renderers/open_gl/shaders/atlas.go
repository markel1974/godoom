package shaders

import (
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
)

// AtlasTileCache stores cached projection/scene keys for a single tile in one atlas buffer.
type AtlasTileCache struct {
	valid     bool
	atlasGen  uint64
	matrix    [16]float32
	sceneHash uint64
}

// Matches reports whether the cached entry is valid for atlasGen, matrix, and sceneHash.
func (c *AtlasTileCache) Matches(atlasGen uint64, matrix *[16]float32, sceneHash uint64) bool {
	return matrix != nil && c.valid && c.atlasGen == atlasGen && c.sceneHash == sceneHash && c.matrix == *matrix
}

// AtlasData represents a structure for managing depth framebuffers and textures for shadow mapping and depth rendering.
type AtlasData struct {
	ctx            api.IContext
	fbo            uint32
	tex            uint32
	width          int32
	height         int32
	gen            uint64
	matrixPtr      *float32
	emptyMatrix    [16]float32
	emptyMatrixPtr *float32
}

// NewAtlasData creates and returns a new instance of AtlasData with default uninitialized properties.
func NewAtlasData(ctx api.IContext) *AtlasData {
	dm := &AtlasData{
		ctx: ctx,
	}
	dm.emptyMatrixPtr = &dm.emptyMatrix[0]
	dm.matrixPtr = dm.emptyMatrixPtr
	return dm
}

// Update initializes and configures the framebuffer and texture for depth rendering with the given dimensions.
func (d *AtlasData) Update(width, height int32) {
	if d.ctx == nil || width <= 0 || height <= 0 {
		return
	}
	if d.fbo != 0 && d.width == width && d.height == height {
		return
	}
	nextGen := d.gen + 1
	d.Shutdown()
	d.gen = nextGen
	d.width = width
	d.height = height
	var fbo, tex uint32
	borderColor := [4]float32{1.0, 1.0, 1.0, 1.0}
	d.ctx.GenFramebuffers(1, &fbo)
	d.ctx.GenTextures(1, &tex)
	d.ctx.ActiveTexture(api.TEXTURE0 + api.TexUnitShadowAtlas)
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
	d.ctx.BindTexture(api.TEXTURE_2D, 0)
	d.fbo = fbo
	d.tex = tex
}

// SetMatrix assigns a 4x4 transformation matrix to the AtlasData instance.
func (d *AtlasData) SetMatrix(matrix *float32) {
	d.matrixPtr = matrix
}

// Shutdown releases OpenGL resources associated with the framebuffer and texture of the AtlasData.
func (d *AtlasData) Shutdown() {
	if d.ctx != nil {
		if d.fbo != 0 {
			d.ctx.DeleteFramebuffers(1, &d.fbo)
		}
		if d.tex != 0 {
			d.ctx.DeleteTextures(1, &d.tex)
		}
	}
	d.fbo = 0
	d.tex = 0
	d.width = 0
	d.height = 0
	d.gen = 0
}

// Atlas manages the double-buffered master shadow depth atlas and per-tile reuse cache.
type Atlas struct {
	ctx             api.IContext
	frameIdx        int
	width           int32
	height          int32
	shadowAtlasMaps [depthDoubleBuffer]*AtlasData
	tileCache       [depthDoubleBuffer][MaxShadowTiles]AtlasTileCache
	shadows         bool
}

// NewAtlas initializes a double-buffered master shadow atlas bound to ctx.
func NewAtlas(ctx api.IContext) *Atlas {
	a := &Atlas{
		ctx:      ctx,
		frameIdx: 0,
		width:    shadowAtlasSize,
		height:   shadowAtlasSize,
		shadows:  true,
	}
	for i := range depthDoubleBuffer {
		a.shadowAtlasMaps[i] = NewAtlasData(ctx)
	}
	a.InvalidateCache()
	return a
}

// SetShadows enables or disables tile cache reuse on the atlas, invalidating cached tiles when re-enabled.
func (s *Atlas) SetShadows(shadows bool) {
	if shadows && !s.shadows {
		s.InvalidateCache()
	}
	s.shadows = shadows
}

// GetShadowAtlasTexture returns the OpenGL texture ID of the currently bound master shadow atlas.
func (s *Atlas) GetShadowAtlasTexture() uint32 {
	if dm := s.GetShadowAtlas(); dm != nil {
		return dm.tex
	}
	return 0
}

// GetShadowAtlas returns the currently active AtlasData backing the master shadow atlas.
func (s *Atlas) GetShadowAtlas() *AtlasData {
	return s.shadowAtlasMaps[s.frameIdx]
}

// NextAtlasIndex returns the double-buffer atlas index that the next Update call will target.
func (s *Atlas) NextAtlasIndex() int {
	return (s.frameIdx + 1) % depthDoubleBuffer
}

// Update advances the double-buffer index, ensures backing depth textures are allocated, and returns the active slot and AtlasData.
func (s *Atlas) Update() (int, *AtlasData) {
	idx := s.NextAtlasIndex()
	s.frameIdx = idx

	atlas := s.shadowAtlasMaps[idx]
	atlas.Update(s.width, s.height)
	return idx, atlas
}

// IsTileCached reports whether tile slot t in buffer idx is already up to date for tile and gen.
func (s *Atlas) IsTileCached(idx, t int, tile *ShadowsTileData, gen uint64) bool {
	if idx < 0 || idx >= depthDoubleBuffer || t < 0 || t >= MaxShadowTiles || tile == nil || !s.shadows {
		return false
	}
	return s.tileCache[idx][t].Matches(gen, &tile.Matrix, tile.SceneHash)
}

// SetTile records tile's matrix, sceneHash, and atlas generation in the cache for buffer idx and slot t.
func (s *Atlas) SetTile(idx, t int, tile *ShadowsTileData, gen uint64) {
	if idx < 0 || idx >= depthDoubleBuffer || t < 0 || t >= MaxShadowTiles || tile == nil {
		return
	}
	c := &s.tileCache[idx][t]
	c.valid = true
	c.atlasGen = gen
	c.matrix = tile.Matrix
	c.sceneHash = tile.SceneHash
}

// Allocate configures the unified master shadow map atlas dimensions and invalidates cached tiles.
func (s *Atlas) Allocate(width, height int32) {
	if width <= 0 || height <= 0 {
		return
	}
	s.width = width
	s.height = height
	for i := range depthDoubleBuffer {
		s.shadowAtlasMaps[i].Update(width, height)
	}
	s.InvalidateCache()
}

// CanReuseTile reports whether slot in the next target atlas buffer already matches matrix and sceneHash.
func (s *Atlas) CanReuseTile(slot int, matrix *[16]float32, sceneHash uint64) bool {
	if slot < 0 || slot >= MaxShadowTiles || !s.shadows || matrix == nil {
		return false
	}
	nextIdx := s.NextAtlasIndex()
	atlas := s.shadowAtlasMaps[nextIdx]
	if atlas == nil || atlas.fbo == 0 || atlas.width != s.width || atlas.height != s.height {
		return false
	}
	return s.tileCache[nextIdx][slot].Matches(atlas.gen, matrix, sceneHash)
}

// InvalidateCache forces all cached shadow atlas tiles across both double-buffers to redraw on next use.
func (s *Atlas) InvalidateCache() {
	for b := range depthDoubleBuffer {
		s.InvalidateTiles(b)
	}
}

// InvalidateTile marks a single cached tile slot in buffer idx as stale.
func (s *Atlas) InvalidateTile(idx, t int) {
	if idx < 0 || idx >= depthDoubleBuffer || t < 0 || t >= MaxShadowTiles {
		return
	}
	s.tileCache[idx][t].valid = false
}

// InvalidateTiles marks all tile slots in buffer idx as stale.
func (s *Atlas) InvalidateTiles(idx int) {
	if idx < 0 || idx >= depthDoubleBuffer {
		return
	}
	for t := range MaxShadowTiles {
		s.tileCache[idx][t].valid = false
	}
}

// Shutdown releases OpenGL resources for all backing depth maps and clears cached tile state.
func (s *Atlas) Shutdown() {
	for i := range depthDoubleBuffer {
		if s.shadowAtlasMaps[i] != nil {
			s.shadowAtlasMaps[i].Shutdown()
		}
	}
	s.InvalidateCache()
}
