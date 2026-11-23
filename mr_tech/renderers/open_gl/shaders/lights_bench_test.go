package shaders

import (
	"math"
	"testing"
	"unsafe"
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
