package atlas

import (
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
)

/*
Y=2048 +------------------------+------------------------+
       |   Sun2   |   Sun3     |                        |
       | (512x512)| (512x512)  |    FLASHLIGHT SHADOW   |
Y=1536 +----------+------------+      (1024 x 1024)     |
       |   Sun0   |   Sun1     |  UV: [0.5..1, 0.5..1]  |
       | (512x512)| (512x512)  |   (or Room 1024x1024)  |
Y=1024 +--------+--------+------+--+--------+------------+
       | Light4 | Light5 |  Light6 | Light7 | (512x512)  |  Row 1 (Y: 512..1023)
 Y=512 +--------+--------+---------+--------+------------+
       | Light0 | Light1 |  Light2 | Light3 | (512x512)  |  Row 0 (Y: 0..511)
   Y=0 +--------+--------+---------+--------+------------+
       X=0     X=512   X=1024    X=1536   X=2048
*/

const (
	DepthDoubleBuffer = 2

	ShadowAtlasSize = int32(2048)
)

// Atlas manages the double-buffered master shadow depth atlas and per-tile reuse cache.
type Atlas struct {
	ctx         api.IContext
	frameIdx    int
	width       int32
	height      int32
	shadowAtlas [DepthDoubleBuffer]*Element
	tileCache   [DepthDoubleBuffer][MaxShadowTiles]*TileCache
	shadows     bool
}

// NewAtlas initializes a double-buffered master shadow atlas bound to ctx.
func NewAtlas(ctx api.IContext) *Atlas {
	a := &Atlas{
		ctx:      ctx,
		frameIdx: 0,
		width:    ShadowAtlasSize,
		height:   ShadowAtlasSize,
		shadows:  true,
	}
	for i := range DepthDoubleBuffer {
		a.shadowAtlas[i] = NewElement(ctx)
		for x := range MaxShadowTiles {
			a.tileCache[i][x] = NewTileCache()
		}
	}
	a.InvalidateCache()
	return a
}

// GetTileRes returns the shadow tile resolution derived from the atlas width, defaulting to 1024.
func (s *Atlas) GetTileRes() float32 {
	if w := s.width; w > 64 {
		return float32(w / 2)
	}
	return 1024
}

// GetWidth returns the configured width of the shadow atlas in pixels.
func (s *Atlas) GetWidth() int32 {
	return s.width
}

// GetHeight returns the configured height of the shadow atlas in pixels.
func (s *Atlas) GetHeight() int32 {
	return s.height
}

// GetSize returns the configured width and height of the shadow atlas in pixels.
func (s *Atlas) GetSize() (int32, int32) {
	return s.width, s.height
}

// TileRect returns the pixel viewport/scissor rectangle and depth-raster state for tile slot at the active atlas size.
func (s *Atlas) TileRect(slot int) (x, y, w, h int32, depthClamp bool, polyUnits, polyFactor float32) {
	halfW := s.width / 2
	halfH := s.height / 2
	if slot == ShadowTileRoom {
		return 0, halfH, halfW, halfH, true, 2.0, 4.0
	}
	if slot == ShadowTileFlash {
		return halfW, halfH, halfW, halfH, false, 1.5, 2.0
	}
	tileW := s.width / 4
	tileH := s.height / 4
	if slot >= ShadowTileSunBase && slot < MaxShadowTiles {
		c := slot - ShadowTileSunBase
		col := int32(c & 1)
		row := int32(c >> 1)
		return col * tileW, halfH + row*tileH, tileW, tileH, true, 2.0, 4.0
	}
	idx := slot - ShadowTileLightBase
	col := int32(idx % 4)
	row := int32(idx / 4)
	return col * tileW, row * tileH, tileW, tileH, false, 1.5, 2.0
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

// GetShadowAtlas returns the currently active Element backing the master shadow atlas.
func (s *Atlas) GetShadowAtlas() *Element {
	return s.shadowAtlas[s.frameIdx]
}

// Update advances the double-buffer index, ensures backing depth textures are allocated, and returns the active slot and Element.
func (s *Atlas) Update() (int, *Element) {
	idx := s.nextIndex()
	s.frameIdx = idx

	atl := s.shadowAtlas[idx]
	atl.Update(s.width, s.height)
	return idx, atl
}

// IsTileCached reports whether tile slot t in buffer idx is already up to date for tile and gen.
func (s *Atlas) IsTileCached(idx, t int, tile *TileShadow, gen uint64) bool {
	if idx < 0 || idx >= DepthDoubleBuffer || t < 0 || t >= MaxShadowTiles || tile == nil || !s.shadows {
		return false
	}
	return s.tileCache[idx][t].Reusable(gen, &tile.matrix[0], tile.sceneHash)
}

// SetTile records tile's matrix, sceneHash, and atlas generation in the cache for buffer idx and slot t.
func (s *Atlas) SetTile(idx, t int, tile *TileShadow, gen uint64) {
	if idx < 0 || idx >= DepthDoubleBuffer || t < 0 || t >= MaxShadowTiles || tile == nil {
		return
	}
	c := s.tileCache[idx][t]
	c.Assign(true, gen, &tile.matrix[0], tile.sceneHash)
}

// Allocate configures the unified master shadow map atlas dimensions and invalidates cached tiles.
func (s *Atlas) Allocate(width, height int32) {
	if width <= 0 || height <= 0 {
		return
	}
	s.width = width
	s.height = height
	for i := range DepthDoubleBuffer {
		s.shadowAtlas[i].Update(width, height)
	}
	s.InvalidateCache()
}

// CanReuseTile reports whether slot in the next target atlas buffer already matches matrix and sceneHash.
func (s *Atlas) CanReuseTile(slot int, matrix *float32, sceneHash uint64) bool {
	if slot < 0 || slot >= MaxShadowTiles || !s.shadows || matrix == nil {
		return false
	}
	nextIdx := s.nextIndex()
	atlas := s.shadowAtlas[nextIdx]
	if atlas == nil || atlas.fbo == 0 || atlas.width != s.width || atlas.height != s.height {
		return false
	}
	c := s.tileCache[nextIdx][slot]
	return c.Reusable(atlas.gen, matrix, sceneHash)
}

// PeekReusableTile returns the target atlas buffer index, atlas generation, and cached sceneHash
// when slot in the next target buffer already holds a valid render for matrix.
func (s *Atlas) PeekReusableTile(slot int, matrixPtr *float32) (int, uint64, uint64, bool) {
	if slot < 0 || slot >= MaxShadowTiles || !s.shadows || matrixPtr == nil {
		return 0, 0, 0, false
	}
	nextIdx := s.nextIndex()
	atl := s.shadowAtlas[nextIdx]
	if atl == nil || atl.fbo == 0 || atl.width != s.width || atl.height != s.height {
		return 0, 0, 0, false
	}
	c := s.tileCache[nextIdx][slot]
	if !c.ReusablePeek(atl.gen, matrixPtr) {
		return 0, 0, 0, false
	}
	return nextIdx, atl.gen, c.sceneHash, true
}

// TargetInfo returns the next writable double-buffer index, its current atlas generation, and whether its FBO is ready.
func (s *Atlas) TargetInfo() (int, uint64, bool) {
	if !s.shadows {
		return 0, 0, false
	}
	nextIdx := s.nextIndex()
	atl := s.shadowAtlas[nextIdx]
	if atl == nil || atl.fbo == 0 || atl.width != s.width || atl.height != s.height {
		return 0, 0, false
	}
	return nextIdx, atl.gen, true
}

// InvalidateCache forces all cached shadow atlas tiles across both double-buffers to redraw on next use.
func (s *Atlas) InvalidateCache() {
	for b := range DepthDoubleBuffer {
		s.InvalidateTiles(b)
	}
}

// InvalidateTile marks a single cached tile slot in buffer idx as stale.
func (s *Atlas) InvalidateTile(idx, t int) {
	if idx < 0 || idx >= DepthDoubleBuffer || t < 0 || t >= MaxShadowTiles {
		return
	}
	s.tileCache[idx][t].valid = false
}

// InvalidateTiles marks all tile slots in buffer idx as stale.
func (s *Atlas) InvalidateTiles(idx int) {
	if idx < 0 || idx >= DepthDoubleBuffer {
		return
	}
	for t := range MaxShadowTiles {
		s.tileCache[idx][t].valid = false
	}
}

// Shutdown releases OpenGL resources for all backing depth maps, resets the frame index, and clears cached tile state.
func (s *Atlas) Shutdown() {
	for i := range DepthDoubleBuffer {
		if s.shadowAtlas[i] != nil {
			s.shadowAtlas[i].Shutdown()
		}
	}
	s.frameIdx = 0
	s.InvalidateCache()
}

// nextIndex returns the double-buffer atlas index that the next Update call will target.
func (s *Atlas) nextIndex() int {
	return (s.frameIdx + 1) % DepthDoubleBuffer
}
