package shaders

import (
	"math"
	"strings"
	"testing"
	"unsafe"

	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/metrics"
)

type clusterStateCtx struct {
	*recordingCtx
}

func newClusterStateCtx() *clusterStateCtx {
	return &clusterStateCtx{recordingCtx: newRecordingCtx()}
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
	l.mapMetrics = metrics.NewMap()
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

	// Pack 4 lights directly in view space (identity view matrix keeps view coords verbatim):
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

	// Turn off light 1 and verify previously active clusters clear back to (0, 0) via active-index tracking.
	lights[1*16+7] = 0.0
	l.Prepare(unsafe.Pointer(&lights[0]), int32(len(lights)*4), 4)
	l.PrepareClustersWithProj(proj, 1.0, 1000.0)
	if off, cnt := l.GetClusterCell(3, 4, leftZ); off != 0 || cnt != 0 {
		t.Fatalf("expected previously active left cluster (3, 4, %d) cleared to (0,0), got (%d,%d)", leftZ, off, cnt)
	}
	if !hasLight(9, 4, rightZ, 2) {
		t.Fatalf("expected light 2 to remain in right cluster (9, 4, %d)", rightZ)
	}

	// Zero active lights must also clear remaining previously active cluster headers.
	l.Prepare(nil, 0, 0)
	if usedEmpty := l.PrepareClustersWithProj(proj, 1.0, 1000.0); usedEmpty != 0 {
		t.Fatalf("expected usedPool=0 when activeLights=0, got %d", usedEmpty)
	}
	if off, cnt := l.GetClusterCell(9, 4, rightZ); off != 0 || cnt != 0 {
		t.Fatalf("expected right cluster (9, 4, %d) cleared to (0,0) on 0 lights, got (%d,%d)", rightZ, off, cnt)
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
	l.mapMetrics = metrics.NewMap()
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
	l.mapMetrics = metrics.NewMap()
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
