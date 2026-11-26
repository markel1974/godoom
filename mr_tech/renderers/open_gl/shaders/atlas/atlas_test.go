package atlas

import (
	"testing"
	"unsafe"

	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
)

type atlasTestCtx struct {
	api.IContext
	nextID uint32
}

func newAtlasTestCtx() *atlasTestCtx {
	return &atlasTestCtx{nextID: 1}
}

func (c *atlasTestCtx) allocIDs(n int32, out *uint32) {
	if out == nil || n <= 0 {
		return
	}
	s := unsafe.Slice(out, int(n))
	for i := range s {
		s[i] = c.nextID
		c.nextID++
	}
}

func (c *atlasTestCtx) ActiveTexture(_ uint32)                                               {}
func (c *atlasTestCtx) BindFramebuffer(_ uint32, _ uint32)                                   {}
func (c *atlasTestCtx) BindTexture(_ uint32, _ uint32)                                       {}
func (c *atlasTestCtx) DeleteFramebuffers(_ int32, _ *uint32)                                {}
func (c *atlasTestCtx) DeleteTextures(_ int32, _ *uint32)                                    {}
func (c *atlasTestCtx) DrawBuffer(_ uint32)                                                  {}
func (c *atlasTestCtx) FramebufferTexture2D(_ uint32, _ uint32, _ uint32, _ uint32, _ int32) {}
func (c *atlasTestCtx) GenFramebuffers(n int32, fbos *uint32)                                { c.allocIDs(n, fbos) }
func (c *atlasTestCtx) GenTextures(n int32, textures *uint32)                                { c.allocIDs(n, textures) }
func (c *atlasTestCtx) ReadBuffer(_ uint32)                                                  {}
func (c *atlasTestCtx) TexImage2D(_ uint32, _ int32, _ int32, _ int32, _ int32, _ int32, _ uint32, _ uint32, _ unsafe.Pointer) {
}
func (c *atlasTestCtx) TexParameterfv(_ uint32, _ uint32, _ *float32) {}
func (c *atlasTestCtx) TexParameteri(_ uint32, _ uint32, _ int32)     {}

func TestAtlasLifecycleAndSetShadowsInvalidation(t *testing.T) {
	ctx := newAtlasTestCtx()
	a := NewAtlas(ctx)

	mat := [16]float32{
		1, 0, 0, 0,
		0, 1, 0, 0,
		0, 0, 1, 0,
		0, 0, 0, 1,
	}
	tile := NewTileShadow()
	tile.Assign(true, nil, &mat[0], 4242, false)
	if tile.GetMatrixPtr() != &tile.matrix[0] || tile.matrix != mat || tile.GetSceneHash() != 4242 {
		t.Fatalf("unexpected TileShadow getters: hash=%d mat=%v", tile.GetSceneHash(), tile.matrix)
	}
	var zeroTile TileShadow
	zeroTile.Assign(true, nil, &mat[0], 123, false)
	if zeroTile.GetMatrixPtr() != &zeroTile.matrix[0] || zeroTile.matrix != mat {
		t.Fatalf("expected zero-value TileShadow.Assign to bind matrixPtr and copy matrix")
	}

	// Cold buffers cannot be reused yet.
	if a.CanReuseTile(ShadowTileRoom, &mat[0], tile.GetSceneHash()) {
		t.Fatalf("expected cold Atlas buffer not reusable")
	}

	// Warm both double-buffers.
	for i := range DepthDoubleBuffer {
		idx, dm := a.Update()
		if dm == nil || dm.GetFBO() == 0 || dm.GetTex() == 0 {
			t.Fatalf("Update iteration %d produced invalid DepthMap: %+v", i, dm)
		}
		a.SetTile(idx, ShadowTileRoom, tile, dm.GetGen())
	}
	if a.GetShadowAtlasTexture() == 0 {
		t.Fatalf("expected valid shadow atlas texture after Update")
	}
	if !a.CanReuseTile(ShadowTileRoom, &mat[0], tile.GetSceneHash()) {
		t.Fatalf("expected warmed double-buffered Atlas tile to be reusable")
	}

	allocs := testing.AllocsPerRun(50, func() {
		if !a.CanReuseTile(ShadowTileRoom, tile.GetMatrixPtr(), tile.GetSceneHash()) || !a.IsTileCached(a.frameIdx, ShadowTileRoom, tile, a.GetShadowAtlas().GetGen()) {
			t.Fatalf("expected zero-alloc cached Atlas hit")
		}
	})
	if allocs > 0 {
		t.Fatalf("expected 0 allocs on Atlas tile cache query, got %.2f", allocs)
	}
	if _, _, _, ok := a.PeekReusableTile(ShadowTileRoom, nil); ok {
		t.Fatalf("expected PeekReusableTile(nil) to return false")
	}

	// Re-enabling shadows after disabling must invalidate prior tile entries.
	a.SetShadows(false)
	if a.CanReuseTile(ShadowTileRoom, &mat[0], tile.GetSceneHash()) {
		t.Fatalf("expected CanReuseTile=false when Atlas shadows disabled")
	}
	a.SetShadows(true)
	if a.CanReuseTile(ShadowTileRoom, &mat[0], tile.GetSceneHash()) {
		t.Fatalf("expected CanReuseTile=false after SetShadows(false->true) invalidation")
	}

	// Verify proportional TileRect coordinates at default 2048x2048.
	rx, ry, rw, rh, rClamp, _, _ := a.TileRect(ShadowTileRoom)
	fx, fy, fw, fh, fClamp, _, _ := a.TileRect(ShadowTileFlash)
	lx5, ly5, lw5, lh5, _, _, _ := a.TileRect(ShadowTileLightBase + 5)
	if rx != 0 || ry != 1024 || rw != 1024 || rh != 1024 || !rClamp ||
		fx != 1024 || fy != 1024 || fw != 1024 || fh != 1024 || fClamp ||
		lx5 != 512 || ly5 != 512 || lw5 != 512 || lh5 != 512 {
		t.Fatalf("unexpected 2048 TileRects: room=(%d,%d,%d,%d,%v) flash=(%d,%d,%d,%d,%v) light5=(%d,%d,%d,%d)",
			rx, ry, rw, rh, rClamp, fx, fy, fw, fh, fClamp, lx5, ly5, lw5, lh5)
	}

	// Re-warm and verify Allocate scales TileRects and invalidates cache for new dimensions.
	for range DepthDoubleBuffer {
		idx, dm := a.Update()
		a.SetTile(idx, ShadowTileRoom, tile, dm.GetGen())
	}
	a.Allocate(1024, 1024)
	if a.CanReuseTile(ShadowTileRoom, &mat[0], tile.GetSceneHash()) {
		t.Fatalf("expected Allocate resize to invalidate cached tiles")
	}
	srx, sry, srw, srh, _, _, _ := a.TileRect(ShadowTileRoom)
	slx, sly, slw, slh, _, _, _ := a.TileRect(ShadowTileLightBase + 5)
	if srx != 0 || sry != 512 || srw != 512 || srh != 512 ||
		slx != 256 || sly != 256 || slw != 256 || slh != 256 {
		t.Fatalf("unexpected 1024 TileRects after Allocate: room=(%d,%d,%d,%d) light5=(%d,%d,%d,%d)",
			srx, sry, srw, srh, slx, sly, slw, slh)
	}

	// Shutdown clears backing GPU handles, resets frameIdx, and clears cache safely.
	a.Update()
	a.Shutdown()
	a.Shutdown()
	if a.frameIdx != 0 {
		t.Fatalf("expected frameIdx=0 after Shutdown, got %d", a.frameIdx)
	}
	if a.GetShadowAtlasTexture() != 0 {
		t.Fatalf("expected GetShadowAtlasTexture=0 after Shutdown")
	}
	if a.CanReuseTile(ShadowTileRoom, &mat[0], tile.GetSceneHash()) {
		t.Fatalf("expected CanReuseTile=false after Shutdown")
	}
}

func TestTileCacheAssignAndReusable(t *testing.T) {
	tc := NewTileCache()
	if tc.valid {
		t.Fatalf("unexpected initial TileCache state: valid=%v", tc.valid)
	}

	mat := [16]float32{
		1, 0, 0, 0,
		0, 2, 0, 0,
		0, 0, 3, 0,
		4, 5, 6, 1,
	}
	tc.Assign(true, 7, &mat[0], 999)
	if !tc.Reusable(7, &mat[0], 999) || !tc.ReusablePeek(7, &mat[0]) {
		t.Fatalf("expected exact match on Reusable and ReusablePeek for gen=7 and sceneHash=999")
	}
	if tc.Reusable(7, nil, 999) || tc.Reusable(8, &mat[0], 999) || tc.Reusable(7, &mat[0], 1000) {
		t.Fatalf("expected Reusable mismatch on nil matrix, different gen, or different sceneHash")
	}
	if tc.ReusablePeek(7, nil) || tc.ReusablePeek(8, &mat[0]) {
		t.Fatalf("expected ReusablePeek mismatch on nil matrix or different gen")
	}
	otherMat := mat
	otherMat[12] = 99
	if tc.Reusable(7, &otherMat[0], 999) || tc.ReusablePeek(7, &otherMat[0]) {
		t.Fatalf("expected mismatch on modified matrix")
	}

	tc.Assign(false, 7, &mat[0], 999)
	if tc.Reusable(7, &mat[0], 999) || tc.ReusablePeek(7, &mat[0]) {
		t.Fatalf("expected mismatch when valid=false")
	}
}

func TestTilesHandlerInitGetAndDisableAllTiles(t *testing.T) {
	h := NewTilesHandler()
	var mat [16]float32
	mat[0], mat[15] = 1, 1

	for i := range MaxShadowTiles {
		tile := h.Get(i)
		if tile == nil || tile.GetMatrixPtr() == nil {
			t.Fatalf("expected non-nil initialized tile and matrixPtr at slot %d", i)
		}
		tile.Assign(true, nil, &mat[0], uint64(i+1), true)
	}

	h.DisableAllTiles()
	for i := range MaxShadowTiles {
		tile := h.Get(i)
		if tile.IsActive() || tile.CanReuse() {
			t.Fatalf("expected tile %d disabled and reuse cleared, got active=%v reuse=%v", i, tile.IsActive(), tile.CanReuse())
		}
	}
}
