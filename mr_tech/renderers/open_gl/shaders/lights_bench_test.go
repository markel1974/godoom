package shaders

import (
	"math"
	"testing"
	"unsafe"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/model"
	"github.com/markel1974/godoom/mr_tech/model/things"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/gpu"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/metrics"
)

func TestLightsPrepareZeroAllocsLargeBatchAndTransforms(t *testing.T) {
	ctx := newRecordingCtx()
	h := newTestHandlerWithMSAA(t, ctx, 0)
	l := h.GetLights()
	m := h.GetMapMetrics()
	if err := l.Init(); err != nil {
		t.Fatalf("Lights.Init: %v", err)
	}
	vi := makeRotatedViewMatrix(t)
	m.SetCenter(15, 25, 0)
	m.Update(vi, 1.2, 1.5)

	const total = 1000
	raw := make([]float32, total*16)
	for i := 0; i < total; i++ {
		off := i * 16
		raw[off+0] = float32(i*10) + 1000.0
		raw[off+1] = 20.0
		raw[off+2] = float32(i*5) + 1000.0
		raw[off+3] = float32(i % 4)
		raw[off+4], raw[off+5], raw[off+6], raw[off+7] = 1.0, 0.9, 0.8, 2.0
		raw[off+8], raw[off+9], raw[off+10], raw[off+11] = 1.0, 0.0, 0.0, 40.0
		raw[off+12], raw[off+13] = 0.9, 0.7
	}
	// Place a spotlight right at camera world origin (~view 0,0,0) with non-unit direction at index 777.
	vx, vy, vz := vi.GetView()
	ex, ey, ez := float32(vx), float32(vz), float32(-vy)
	hotOff := 777 * 16
	raw[hotOff+0], raw[hotOff+1], raw[hotOff+2], raw[hotOff+3] = ex, ey, ez, 1.0
	raw[hotOff+7] = 3.0
	raw[hotOff+8], raw[hotOff+9], raw[hotOff+10], raw[hotOff+11] = 3.0, 4.0, 0.0, 60.0

	l.Prepare(unsafe.Pointer(&raw[0]), int32(len(raw)*4), total)
	if l.GetActiveLightsCount() != maxUBOLights {
		t.Fatalf("expected activeLights=%d, got %d", maxUBOLights, l.GetActiveLightsCount())
	}

	wantVX, wantVY, wantVZ := m.TransformPointToView(ex, ey, ez)
	wantDX, wantDY, wantDZ := m.TransformDirectionNormalizedToView(3.0, 4.0, 0.0)
	foundHot := false
	for i := 0; i < maxUBOLights; i++ {
		off := i * 16
		if math.Abs(float64(l.viewLights[off]-wantVX)) < 1e-4 &&
			math.Abs(float64(l.viewLights[off+1]-wantVY)) < 1e-4 &&
			math.Abs(float64(l.viewLights[off+2]-wantVZ)) < 1e-4 &&
			l.viewLights[off+7] == 3.0 {
			if math.Abs(float64(l.viewLights[off+8]-wantDX)) > 1e-4 ||
				math.Abs(float64(l.viewLights[off+9]-wantDY)) > 1e-4 ||
				math.Abs(float64(l.viewLights[off+10]-wantDZ)) > 1e-4 {
				t.Fatalf("spotlight dir mismatch: got (%v,%v,%v) want (%v,%v,%v)",
					l.viewLights[off+8], l.viewLights[off+9], l.viewLights[off+10], wantDX, wantDY, wantDZ)
			}
			foundHot = true
			break
		}
	}
	if !foundHot {
		t.Fatalf("expected camera-local light at index 777 to be retained in top-%d UBO heap", maxUBOLights)
	}

	allocs := testing.AllocsPerRun(20, func() {
		l.Prepare(unsafe.Pointer(&raw[0]), int32(len(raw)*4), total)
	})
	if allocs > 0 {
		t.Fatalf("Lights.Prepare (>256) allocated %.2f objects/op, want 0", allocs)
	}
}

func BenchmarkLightsPrepare256(b *testing.B) {
	benchmarkLightsPrepare(b, 256)
}

func BenchmarkLightsPrepare1024(b *testing.B) {
	benchmarkLightsPrepare(b, 1024)
}

func benchmarkLightsPrepare(b *testing.B, count int) {
	ctx := newRecordingCtx()
	h := newTestHandlerWithMSAA(&testing.T{}, ctx, 0)
	l := h.GetLights()
	m := h.GetMapMetrics()
	_ = l.Init()
	vi := makeRotatedViewMatrix(&testing.T{})
	m.SetCenter(15, 25, 0)
	m.Update(vi, 1.2, 1.5)

	raw := make([]float32, count*16)
	for i := 0; i < count; i++ {
		off := i * 16
		raw[off+0] = float32(i%50) * 10
		raw[off+1] = 10
		raw[off+2] = float32(i/50) * 10
		raw[off+3] = 0
		raw[off+4], raw[off+5], raw[off+6], raw[off+7] = 1, 1, 1, 2
		raw[off+11] = 40
	}
	ptr := unsafe.Pointer(&raw[0])
	size := int32(len(raw) * 4)
	n := int32(count)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l.Prepare(ptr, size, n)
	}
}

func BenchmarkLightsPrepareClustersSparse(b *testing.B) {
	ctx := newRecordingCtx()
	h := newTestHandlerWithMSAA(&testing.T{}, ctx, 0)
	l := h.GetLights()
	_ = l.Init()

	const count = 32
	raw := make([]float32, count*16)
	for i := 0; i < count; i++ {
		off := i * 16
		raw[off+0] = float32((i%8)-4) * 5.0
		raw[off+1] = 0.0
		raw[off+2] = -20.0 - float32(i)*4.0
		raw[off+3] = 0.0
		raw[off+4], raw[off+5], raw[off+6], raw[off+7] = 1, 1, 1, 1.0
		raw[off+11] = 2.0
	}
	l.Prepare(unsafe.Pointer(&raw[0]), int32(len(raw)*4), count)
	proj := [16]float32{
		1, 0, 0, 0,
		0, 1, 0, 0,
		0, 0, -1001.0 / 999.0, -1,
		0, 0, -2000.0 / 999.0, 0,
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l.PrepareClustersWithProjPtr(&proj, 1.0, 1000.0)
	}
}

func setupDepthAtlas10TileTest(t testing.TB, staticCount int) (*recordingCtx, *Depth, *gpu.DrawCommandsRender) {
	t.Helper()
	ctx := newRecordingCtx()
	calCfg := config.NewConfigCalibration()
	cal := model.NewCalibration(calCfg, nil)
	cal.EnableShadowLights = true
	cal.MSAASamples = 0
	flashCfg := config.NewFlash()
	flashCfg.Intensity = 0.05
	flash := things.NewFlash(flashCfg)
	h := NewHandler(80, 64)
	if err := h.Setup(ctx, cal, flash, 640, 480); err != nil {
		t.Fatalf("Handler.Setup: %v", err)
	}
	if err := h.GetMain().Init(); err != nil {
		t.Fatalf("Main.Init: %v", err)
	}
	d := h.GetDepth()
	if err := d.Compile(diskAssets{}); err != nil {
		t.Fatalf("Depth.Compile: %v", err)
	}
	d.SetShadowsEnabled(true)

	var shadowLights [metrics.ShadowLightsCount]*metrics.Light
	for i := range metrics.ShadowLightsCount {
		lt := metrics.NewLight()
		lt.Assign(float32(i*40), 50, -30, 1, 1, 1, 1, 2.0, 0, -1, 0, 10, 0.8, 0.7, 1.0)
		shadowLights[i] = lt
	}
	h.GetShadowLightsMetrics().Update(shadowLights, metrics.ShadowLightsCount)
	d.Prepare()

	staticAtlas := gpu.NewResidentMesh(ctx)
	staticAtlas.VAO = 9001
	staticItems := make([]*gpu.StaticDrawItem, staticCount)
	for i := range staticCount {
		staticItems[i] = &gpu.StaticDrawItem{
			Mesh: staticAtlas,
			Sub: &gpu.ResidentSubMesh{
				BlendMode:  config.BlendModeOpaque,
				CullMode:   config.CullFront,
				DepthWrite: true,
				FirstIndex: int32(i * 6),
				IndexCount: 6,
			},
		}
	}

	instMesh := gpu.NewResidentMesh(ctx)
	instMesh.VAO = 9002
	instMesh.VAOInstanced = 9003
	instItems := make([]*gpu.InstancedDrawItem, 4)
	for i := range 4 {
		instItems[i] = &gpu.InstancedDrawItem{
			Mesh: instMesh,
			Sub: &gpu.ResidentSubMesh{
				BlendMode:  config.BlendModeOpaque,
				CullMode:   config.CullFront,
				DepthWrite: true,
				FirstIndex: int32(i * 12),
				IndexCount: 12,
			},
			InstanceVBO:    77,
			BaseByteOffset: 0,
			InstanceCount:  8,
		}
	}

	dcr := gpu.NewDrawCommandsRender(ctx, "depth_bench", false)
	dcr.Prepare(nil)
	dcr.SetResidentDraws(staticItems, len(staticItems), instItems, len(instItems))
	return ctx, d, dcr
}

func TestDepthAtlasPassStaticCoalescedZeroAllocs(t *testing.T) {
	ctx, d, dcr := setupDepthAtlas10TileTest(t, 1000)

	// Warm up atlas FBO + instanced VAO once before measuring steady-state calls and allocs.
	d.Render(dcr)

	ctx.resetLogs()
	d.Render(dcr)

	// Across 10 tiles (Room + Flash + 8 ShadowLights):
	// 1000 contiguous static submeshes merge into 1 DrawElements per tile = 10 static draws,
	// plus 4 instanced submeshes per tile = 40 instanced draws -> 50 draws total instead of 10,040!
	if got := len(ctx.multiDrawPrograms); got != 50 {
		t.Fatalf("expected 50 coalesced draws across 10 shadow tiles (1000 static + 4 instanced), got %d", got)
	}
	// Each tile binds staticAtlas VAO (1) + instMesh VAOInstanced (1) once, plus Depth.Render initial mainVAO (1) = 21.
	if got := len(ctx.vaoHistory); got != 21 {
		t.Fatalf("expected 21 BindVertexArray calls across 10 tiles (no per-submesh VAO thrash), got %d", got)
	}

	allocs := testing.AllocsPerRun(50, func() {
		ctx.resetLogs()
		d.Render(dcr)
	})
	if allocs != 0 {
		t.Fatalf("expected 0 heap allocations in Depth.Render steady-state atlas pass, got %.2f", allocs)
	}
}

func BenchmarkDepthRenderAtlasPass1000Static(b *testing.B) {
	ctx, d, dcr := setupDepthAtlas10TileTest(b, 1000)
	d.Render(dcr)
	ctx.resetLogs()
	d.Render(dcr)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx.resetLogs()
		d.Render(dcr)
	}
}
