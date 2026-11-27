package metrics

import (
	"math"
	"testing"
	"unsafe"
)

func extractSpaceParams(ptr *float32) (fovDeg, far float64) {
	m := (*[16]float32)(unsafe.Pointer(ptr))
	fovDeg = 2.0 * math.Atan(1.0/float64(m[0])) * 180.0 / math.Pi
	far = 0.1 * (float64(m[10]) - 1.0) / (float64(m[10]) + 1.0)
	return fovDeg, far
}

func TestShadowLightsClampingAndWideFOV(t *testing.T) {
	sl := NewShadowLights()
	var inputs [ShadowLightsCount]*Light
	for i := range inputs {
		inputs[i] = NewLight()
	}

	// Light 0: narrow spotlight with tiny falloff/intensity that must clamp to match ComputeLightAttenuation (>= 1.0, >= 0.1) and >= near+1.0
	inputs[0].Assign(0, 0, 0, 1, 1, 1, 1, 0.01, 0, 0, -1, 0.2, float32(math.Cos(30*math.Pi/180)), float32(math.Cos(40*math.Pi/180)), 1)
	// Light 1: wide 60-deg half-angle spotlight (outer cone 120 deg) must widen shadow FOV beyond 90 deg
	inputs[1].Assign(0, 0, 0, 1, 1, 1, 1, 1.0, 0, 0, -1, 100.0, float32(math.Cos(45*math.Pi/180)), float32(math.Cos(60*math.Pi/180)), 1)

	sl.Update(inputs, 2)

	_, p0 := sl.GetLight(0)
	fov0, far0 := extractSpaceParams(p0)
	if math.Abs(fov0-90.0) > 0.01 {
		t.Fatalf("narrow cone expected fov 90, got %.4f", fov0)
	}
	if far0 < 1.09 {
		t.Fatalf("clamped effective radius too small: %.4f", far0)
	}

	_, p1 := sl.GetLight(1)
	fov1, far1 := extractSpaceParams(p1)
	if fov1 < 131.0 || fov1 > 133.0 {
		t.Fatalf("wide cone expected fov ~132, got %.4f", fov1)
	}
	if math.Abs(far1-460.5) > 0.1 {
		t.Fatalf("expected far 460.5, got %.4f", far1)
	}

	allocs := testing.AllocsPerRun(100, func() {
		sl.Update(inputs, 2)
	})
	if allocs > 0 {
		t.Fatalf("expected 0 allocs in ShadowLights.Update, got %.1f", allocs)
	}
}
