package shaders

import (
	"math"
	"strings"
	"testing"
	"unsafe"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
	"github.com/markel1974/godoom/mr_tech/textures"
)

type clusterStateCtx struct {
	*recordingCtx
	enableLog   []uint32
	disableLog  []uint32
	cullFaceLog []uint32
	drawStarts  []uintptr
}

func newClusterStateCtx() *clusterStateCtx {
	return &clusterStateCtx{recordingCtx: newRecordingCtx()}
}

func (c *clusterStateCtx) Enable(cap uint32) {
	c.recordingCtx.Enable(cap)
	c.enableLog = append(c.enableLog, cap)
}

func (c *clusterStateCtx) Disable(cap uint32) {
	c.recordingCtx.Disable(cap)
	c.disableLog = append(c.disableLog, cap)
}

func (c *clusterStateCtx) CullFace(mode uint32) {
	c.cullFaceLog = append(c.cullFaceLog, mode)
}

func (c *clusterStateCtx) DrawElements(_ uint32, _ int32, _ uint32, indices unsafe.Pointer) {
	c.drawStarts = append(c.drawStarts, uintptr(indices))
	c.multiDrawPrograms = append(c.multiDrawPrograms, c.boundProgram)
}

func (c *clusterStateCtx) DrawElementsInstanced(_ uint32, _ int32, _ uint32, indices unsafe.Pointer, _ int32) {
	c.drawStarts = append(c.drawStarts, uintptr(indices))
	c.multiDrawPrograms = append(c.multiDrawPrograms, c.boundProgram)
}

func (c *clusterStateCtx) Ptr(data interface{}) unsafe.Pointer {
	switch s := data.(type) {
	case []float32:
		if len(s) > 0 {
			return unsafe.Pointer(&s[0])
		}
	case []uint16:
		if len(s) > 0 {
			return unsafe.Pointer(&s[0])
		}
	case []uint32:
		if len(s) > 0 {
			return unsafe.Pointer(&s[0])
		}
	}
	return nil
}

func TestLightsDeadShaderCompileSkipped(t *testing.T) {
	ctx := newClusterStateCtx()
	l := NewLights(64)
	l.ctx = ctx

	if err := l.Compile(mockAssets{}); err != nil {
		t.Fatalf("Lights.Compile returned unexpected error: %v", err)
	}
	if l.prg != 0 {
		t.Fatalf("expected unused standalone lights.frag not compiled (prg=0), got %d", l.prg)
	}
	if err := l.Init(); err != nil {
		t.Fatalf("Lights.Init failed: %v", err)
	}
	if l.clusterTex == 0 {
		t.Fatalf("expected Lights.Init to allocate cluster lookup texture")
	}
	if err := l.SetupSamplers(); err != nil {
		t.Fatalf("Lights.SetupSamplers with prg=0 failed: %v", err)
	}
}

func TestClusteredForwardBinningAndZeroAllocs(t *testing.T) {
	ctx := newClusterStateCtx()
	l := NewLights(64)
	l.ctx = ctx
	if err := l.Init(); err != nil {
		t.Fatalf("Lights.Init failed: %v", err)
	}

	// Perspective projection with 90 deg FOV (scaleX = 1, scaleY = 1), zNear = 1, zFar = 1000.
	proj := [16]float32{
		1, 0, 0, 0,
		0, 1, 0, 0,
		0, 0, -1001.0 / 999.0, -1,
		0, 0, -2000.0 / 999.0, 0,
	}

	// Pack 4 lights directly in view space (mapMetrics == nil keeps view coords verbatim):
	// 0: Directional light (type=2) -> must go into global pool.
	// 1: Near-left point light at (-18, 0, -30) with small radius -> cx < 8 only.
	// 2: Far-right point light at (+40, 0, -200) -> cx >= 8 only, higher cz slice.
	// 3: Behind camera / beyond zFar light at (0, 0, +500) -> culled from all bins.
	lights := make([]float32, 4*16)
	// Light 0: Directional
	lights[0*16+3] = 2.0 // Directional
	lights[0*16+7] = 1.0 // Intensity
	lights[0*16+10] = -1.0
	// Light 1: Point left (effRadius = 4.605 * 2 * 1 = ~9.21, safely stays x < -8 -> cx < 8)
	lights[1*16+0] = -18.0
	lights[1*16+1] = 0.0
	lights[1*16+2] = -30.0
	lights[1*16+3] = 0.0
	lights[1*16+7] = 1.0
	lights[1*16+11] = 2.0
	// Light 2: Point right at (-? -> +40, 0, -200), radius ~9.21 -> x > +30 -> cx >= 8
	lights[2*16+0] = 40.0
	lights[2*16+1] = 0.0
	lights[2*16+2] = -200.0
	lights[2*16+3] = 0.0
	lights[2*16+7] = 1.0
	lights[2*16+11] = 2.0
	// Light 3: Behind near plane (vz = +500 -> depth = -500)
	lights[3*16+0] = 0.0
	lights[3*16+1] = 0.0
	lights[3*16+2] = 500.0
	lights[3*16+3] = 0.0
	lights[3*16+7] = 1.0
	lights[3*16+11] = 2.0

	l.Prepare(unsafe.Pointer(&lights[0]), int32(len(lights)*4), 4)
	usedPool := l.PrepareClustersWithProj(proj, 1.0, 1000.0)
	if usedPool <= 1 {
		t.Fatalf("expected global light + clustered bins in pool, got usedPool=%d", usedPool)
	}
	_, near, logScale, globalCount := l.GetClusterState()
	if near != 1.0 || logScale <= 0 || globalCount != 1 {
		t.Fatalf("unexpected cluster state: near=%v logScale=%v globalCount=%d", near, logScale, globalCount)
	}
	if l.GetClusterPoolEntry(0) != 0 {
		t.Fatalf("expected directional light 0 in global pool slot 0, got %d", l.GetClusterPoolEntry(0))
	}

	depthToSlice := func(d float64) int {
		s := int(math.Floor(math.Log(d/1.0) * float64(logScale)))
		if s < 0 {
			s = 0
		} else if s >= ClusterGridZ {
			s = ClusterGridZ - 1
		}
		return s
	}

	leftZ := depthToSlice(30.0)
	rightZ := depthToSlice(200.0)
	hasLight := func(cx, cy, cz int, target uint16) bool {
		off, cnt := l.GetClusterCell(cx, cy, cz)
		for i := uint32(0); i < cnt; i++ {
			if l.GetClusterPoolEntry(int(off+i)) == target {
				return true
			}
		}
		return false
	}

	// Left light (-18, 0, -30) -> ndcX = -18/30 = -0.6 -> uvX = 0.2 -> cx ~= 3 (< 8).
	if !hasLight(3, 4, leftZ, 1) {
		t.Fatalf("expected light 1 in left cluster (3, 4, %d)", leftZ)
	}
	if hasLight(12, 4, leftZ, 1) {
		t.Fatalf("did not expect light 1 on right half of frustum (12, 4, %d)", leftZ)
	}
	// Right light (+40, 0, -200) -> ndcX = +0.2 -> uvX = 0.6 -> cx ~= 9 (>= 8).
	if !hasLight(9, 4, rightZ, 2) {
		t.Fatalf("expected light 2 in right cluster (9, 4, %d)", rightZ)
	}
	if hasLight(2, 4, rightZ, 2) {
		t.Fatalf("did not expect light 2 on far-left cluster (2, 4, %d)", rightZ)
	}

	// Ensure zero heap allocations on hot cluster rebuild path.
	allocs := testing.AllocsPerRun(200, func() {
		l.PrepareClustersWithProj(proj, 1.0, 1000.0)
	})
	if allocs != 0 {
		t.Fatalf("expected 0 heap allocations in PrepareClustersWithProj, got %.2f", allocs)
	}
}

func TestPerMaterialCullStateAndShadowDoubleSidedAndTransparentSort(t *testing.T) {
	ctx := newClusterStateCtx()
	dcr := NewDrawCommandsRender(ctx, "cull_test", false)

	tex := textures.NewTexture("T", 1, 32, 32, false)
	for x := 0; x < 32; x++ {
		for y := 0; y < 32; y++ {
			tex.SetDiffuse(x, y, int(0xFFFFFFFF))
		}
	}
	frames := []*textures.Texture{tex}
	texAlpha := textures.NewTexture("T_CUTOUT", 1, 32, 32, true)
	alphaFrames := []*textures.Texture{texAlpha}

	matFront := textures.NewMaterial("front", frames, int(config.MaterialKindSolid))
	matFront.SetCullMode(config.CullFront)
	matFront.SetDepthWrite(true)

	matBack := textures.NewMaterial("back", frames, int(config.MaterialKindSolid))
	matBack.SetCullMode(config.CullBack)
	matBack.SetDepthWrite(true)

	matTwoSided := textures.NewMaterial("two", frames, int(config.MaterialKindSolid))
	matTwoSided.SetCullMode(config.CullNone)
	matTwoSided.SetDepthWrite(true)

	matCutout := textures.NewMaterial("cutout", alphaFrames, int(config.MaterialKindSolid))
	matCutout.SetCullMode(config.CullFront)
	matCutout.SetDepthWrite(true)

	cmds := make([]*DrawCommand, 4)
	for i := range cmds {
		cmds[i] = NewDrawCommand()
		cmds[i].SetFirstIndex(int32(i * 6))
		cmds[i].SetIndexCount(6)
	}
	cmds[0].SetMaterial(matFront)
	cmds[1].SetMaterial(matBack)
	cmds[2].SetMaterial(matTwoSided)
	cmds[3].SetMaterial(matCutout)

	dcr.Prepare(cmds)
	dcr.Render()

	if len(ctx.cullFaceLog) < 2 || ctx.cullFaceLog[0] != api.BACK || ctx.cullFaceLog[1] != api.FRONT {
		t.Fatalf("expected CullFront->BACK then CullBack->FRONT transitions, got %v", ctx.cullFaceLog)
	}
	// Last disabled state should include CULL_FACE when entering two-sided / cutout and at end of pass.
	sawDisableCull := false
	for _, cap := range ctx.disableLog {
		if cap == api.CULL_FACE {
			sawDisableCull = true
			break
		}
	}
	if !sawDisableCull {
		t.Fatalf("expected CULL_FACE disabled for two-sided/cutout materials, disableLog=%v", ctx.disableLog)
	}

	// Shadow atlas pass (RenderCutoutFiltered) must keep CULL_FACE disabled even for CullFront/CullBack.
	ctx.cullFaceLog = ctx.cullFaceLog[:0]
	ctx.enableLog = ctx.enableLog[:0]
	dcr.RenderCutoutFiltered(false)
	for _, cap := range ctx.enableLog {
		if cap == api.CULL_FACE {
			t.Fatalf("shadow atlas RenderCutoutFiltered must not enable CULL_FACE")
		}
	}
	if len(ctx.cullFaceLog) != 0 {
		t.Fatalf("shadow atlas RenderCutoutFiltered must not switch CullFace, got %v", ctx.cullFaceLog)
	}

	// Verify back-to-front transparent sorting (far before near, SortOrder ascending tie-breaker).
	mesh := NewResidentMesh(ctx)
	mesh.VAO = 77
	mesh.SubMeshes = []ResidentSubMesh{
		{FirstIndex: 30, IndexCount: 6, SortOrder: 2, CullMode: config.CullNone},
	}

	transItems := []TransparentDrawItem{
		{Kind: TransparentStreaming, DistSq: 100.0, SortOrder: 1, CullMode: config.CullNone, FirstIndex: 10, IndexCount: 6},
		{Kind: TransparentStreaming, DistSq: 900.0, SortOrder: 2, CullMode: config.CullNone, FirstIndex: 40, IndexCount: 6},
		{Kind: TransparentStatic, DistSq: 900.0, SortOrder: 0, CullMode: config.CullNone, Mesh: mesh, SubIdx: 0},
	}
	SortTransparentDrawItems(transItems)
	dcrTrans := NewDrawCommandsRender(ctx, "trans_test", false)
	dcrTrans.SetTransparentDraws(transItems)
	ctx.drawStarts = ctx.drawStarts[:0]
	dcrTrans.RenderTransparent()

	if len(ctx.drawStarts) != 3 {
		t.Fatalf("expected 3 sorted transparent draws, got %d", len(ctx.drawStarts))
	}
	// Expected draw index byte offsets: 30*4 (DistSq=900, SortOrder=0) -> 40*4 (DistSq=900, SortOrder=2) -> 10*4 (DistSq=100).
	if ctx.drawStarts[0] != 30*4 || ctx.drawStarts[1] != 40*4 || ctx.drawStarts[2] != 10*4 {
		t.Fatalf("unexpected transparent back-to-front order (byte offsets): %v", ctx.drawStarts)
	}
}

func TestLiquidAndAdditiveCompileWithUnifiedVertexAndInstanceMode(t *testing.T) {
	ctx := newClusterStateCtx()
	assets := diskAssets{}

	liq := NewLiquid()
	liq.ctx = ctx
	if err := liq.Compile(assets); err != nil {
		t.Fatalf("Liquid.Compile failed: %v", err)
	}
	if liq.GetUniform(ShaderLiquidLocInstanceMode) < 0 {
		t.Fatalf("expected u_instanceMode uniform resolved in Liquid shader")
	}

	add := NewAdditive()
	add.ctx = ctx
	if err := add.Compile(assets); err != nil {
		t.Fatalf("Additive.Compile failed: %v", err)
	}
	if add.GetUniform(ShaderAdditiveLocInstanceMode) < 0 {
		t.Fatalf("expected u_instanceMode uniform resolved in Additive shader")
	}
}

func TestClusterPoolHandlesHighDensityWithoutTruncationOrUint16Wrap(t *testing.T) {
	ctx := newClusterStateCtx()
	l := NewLights(64)
	l.ctx = ctx
	if err := l.Init(); err != nil {
		t.Fatalf("Lights.Init failed: %v", err)
	}

	proj := [16]float32{
		1, 0, 0, 0,
		0, 1, 0, 0,
		0, 0, -1001.0 / 999.0, -1,
		0, 0, -2000.0 / 999.0, 0,
	}

	// 200 large overlapping lights centered near origin cover all 2,304 clusters -> 460,800 pool entries
	// (exceeding both legacy 23,552 cap and 65,535 uint16 wrap boundary).
	const numLights = 200
	lights := make([]float32, numLights*16)
	for i := 0; i < numLights; i++ {
		off := i * 16
		lights[off+2] = -20.0
		lights[off+3] = 3.0   // LightKindAmbient
		lights[off+7] = 10.0  // Intensity
		lights[off+11] = 50.0 // Falloff -> effRadius = 4.605*500 = 2302.5 (covers zNear..zFar & all cx,cy)
	}

	l.Prepare(unsafe.Pointer(&lights[0]), int32(len(lights)*4), numLights)
	usedPool := l.PrepareClustersWithProj(proj, 1.0, 1000.0)
	wantPool := ClusterCount * numLights
	if usedPool != wantPool {
		t.Fatalf("expected full cluster pool %d without truncation, got %d", wantPool, usedPool)
	}

	off, cnt := l.GetClusterCell(ClusterGridX-1, ClusterGridY-1, ClusterGridZ-1)
	if cnt != numLights {
		t.Fatalf("last cluster cell dropped lights: got count=%d, want %d", cnt, numLights)
	}
	wantOff := uint32((ClusterCount - 1) * numLights)
	if off != wantOff {
		t.Fatalf("last cluster offset wrapped or mismatched: got %d, want %d", off, wantOff)
	}
	if first := l.GetClusterPoolEntry(int(off)); first != 0 {
		t.Fatalf("expected light 0 at slot %d, got %d", off, first)
	}
	if last := l.GetClusterPoolEntry(int(off + cnt - 1)); last != numLights-1 {
		t.Fatalf("expected light %d at tail slot, got %d", numLights-1, last)
	}

	fragSrc, err := (diskAssets{}).Read("main_opaque.frag")
	if err != nil {
		t.Fatalf("failed to read main_opaque.frag: %v", err)
	}
	if !strings.Contains(string(fragSrc), "header.r | (header.b << 16u)") {
		t.Fatalf("main_opaque.frag must decode 32-bit cluster pool offset from header.r and header.b")
	}
}

func TestLightsPrepareCapsAboveMaxUBOLightsByProximity(t *testing.T) {
	ctx := newClusterStateCtx()
	l := NewLights(64)
	l.ctx = ctx
	if err := l.Init(); err != nil {
		t.Fatalf("Lights.Init failed: %v", err)
	}

	const numLights = 300
	lights := make([]float32, numLights*16)
	for i := 0; i < numLights-1; i++ {
		off := i * 16
		lights[off+2] = -2000.0 - float32(i)
		lights[off+3] = 3.0
		lights[off+7] = 1.0
		lights[off+11] = 2.0
	}
	// Put a bright nearby light at the end of the slice (index 299) so naive truncation would drop it.
	nearOff := (numLights - 1) * 16
	lights[nearOff+2] = -15.0
	lights[nearOff+3] = 3.0
	lights[nearOff+7] = 5.0
	lights[nearOff+11] = 10.0

	l.Prepare(unsafe.Pointer(&lights[0]), int32(len(lights)*4), numLights)
	if got := l.GetActiveLightsCount(); got != maxUBOLights {
		t.Fatalf("expected activeLights clamped to %d, got %d", maxUBOLights, got)
	}
	foundNear := false
	for i := 0; i < maxUBOLights; i++ {
		if l.viewLights[i*16+2] == -15.0 {
			foundNear = true
			break
		}
	}
	if !foundNear {
		t.Fatalf("expected nearby light (vz=-15) prioritized inside UBO limit when >256 lights exist")
	}
}
