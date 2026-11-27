package builder

import (
	"testing"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/engine"
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/model"
	"github.com/markel1974/godoom/mr_tech/model/lights"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/metrics"
	"github.com/markel1974/godoom/mr_tech/textures"
)

func makeSpotLight(x, y, z, intensity, falloff float64, style []float64) *lights.Light {
	cfg := config.NewConfigLight(geometry.XYZ{X: x, Y: y, Z: z}, intensity, config.LightKindSpot, falloff)
	if len(style) > 0 {
		cfg.Style = style
	}
	l := lights.NewLight()
	l.Setup(cfg, cfg.Pos)
	return l
}

func findShadowSlotByX(shadowLights [ShadowLightNumber]*metrics.Light, count int32, x float32) int {
	for i := 0; i < int(count); i++ {
		if shadowLights[i].X == x {
			return i
		}
	}
	return -1
}

func TestFrameLightsSpotPersistentSlotsAndHysteresis(t *testing.T) {
	fl := NewFrameLights(nil, 64, true)
	spots := make([]*lights.Light, 9)
	for i := 0; i < 8; i++ {
		spots[i] = makeSpotLight(float64((i+1)*100), 0, 0, 1.0, 100.0, nil)
	}
	// Contender #9
	spots[8] = makeSpotLight(900, 0, 0, 1.0, 100.0, nil)

	// Frame 1: fill all 8 shadow slots
	fl.DeepReset()
	fl.Prepare()
	for i := 0; i < 8; i++ {
		fl.Create(spots[i], float64((i+1)*100*(i+1)*100), true)
	}
	fl.Finalize()

	sl, slCount := fl.GetShadowLights()
	if slCount != ShadowLightNumber {
		t.Fatalf("expected %d shadow lights, got %d", ShadowLightNumber, slCount)
	}
	var initialSlot [8]int
	for i := 0; i < 8; i++ {
		initialSlot[i] = findShadowSlotByX(sl, slCount, float32((i+1)*100))
		if initialSlot[i] < 0 {
			t.Fatalf("spot %d missing from shadow lights", i)
		}
	}

	// Frame 2: reverse player proximity order, send spot0 temporarily !isFront, and add 9th spot closer than spot0 (dist=400 vs 800).
	// Incumbent hysteresis (4x score / ~2x distance) and no !isFront penalty for incumbents must keep all 8 slots unchanged.
	fl.DeepReset()
	fl.Prepare()
	fl.Create(spots[0], 800*800, false)
	for i := 1; i < 8; i++ {
		d := float64((8 - i) * 100)
		fl.Create(spots[i], d*d, true)
	}
	fl.Create(spots[8], 400*400, true)
	fl.Finalize()

	sl, slCount = fl.GetShadowLights()
	_, _, unshadowedCount := fl.GetLights()
	if slCount != ShadowLightNumber || unshadowedCount != 1 {
		t.Fatalf("expected 8 shadowed + 1 unshadowed, got %d shadowed + %d unshadowed", slCount, unshadowedCount)
	}
	for i := 0; i < 8; i++ {
		slot := findShadowSlotByX(sl, slCount, float32((i+1)*100))
		if slot != initialSlot[i] {
			t.Fatalf("slot scrambled for spot %d: expected %d, got %d", i, initialSlot[i], slot)
		}
	}

	// Frame 3: spot0 moves far away (dist=1400) while 9th spotlight is very close (dist=50), exceeding 4x score hysteresis.
	// Only spot0 should be evicted and its slot reused by spot8; the other 7 must stay in their exact slots.
	fl.DeepReset()
	fl.Prepare()
	fl.Create(spots[0], 1400*1400, true)
	for i := 1; i < 8; i++ {
		d := float64(100 + i*20)
		fl.Create(spots[i], d*d, true)
	}
	fl.Create(spots[8], 50*50, true)
	fl.Finalize()

	sl, slCount = fl.GetShadowLights()
	_, _, unshadowedCount = fl.GetLights()
	if slCount != ShadowLightNumber || unshadowedCount != 1 {
		t.Fatalf("expected 8 shadowed + 1 unshadowed, got %d shadowed + %d unshadowed", slCount, unshadowedCount)
	}
	if slot8 := findShadowSlotByX(sl, slCount, 900); slot8 != initialSlot[0] {
		t.Fatalf("expected new winner spot8 to reuse evicted slot %d, got %d", initialSlot[0], slot8)
	}
	for i := 1; i < 8; i++ {
		slot := findShadowSlotByX(sl, slCount, float32((i+1)*100))
		if slot != initialSlot[i] {
			t.Fatalf("surviving spot %d moved from slot %d to %d", i, initialSlot[i], slot)
		}
	}
}

func TestFrameLightsNearSphereAndRangeHysteresis(t *testing.T) {
	fl := NewFrameLights(nil, 64, true)
	// falloff=15, intensity=1 -> effRadius = 4.605 * 15 = 69.075 (enterRange=2048, exitRange=2560; nearRadius=369.075 -> 489.075)
	spot := makeSpotLight(123, 0, 0, 1.0, 15.0, nil)
	effRadius := 4.605 * 15.0

	// Within near-camera sphere (effRadius + 250) while isFront==false: must stay shadowed.
	nearIn := effRadius + 250.0
	fl.DeepReset()
	fl.Prepare()
	fl.Create(spot, nearIn*nearIn, false)
	fl.Finalize()
	if _, cnt := fl.GetShadowLights(); cnt != 1 {
		t.Fatalf("expected near !isFront spotlight to cast shadows, got count %d", cnt)
	}

	// Step beyond enter nearRadius (effRadius + 360 <= exit nearRadius effRadius + 420): must remain shadowed via hysteresis.
	nearHold := effRadius + 360.0
	fl.DeepReset()
	fl.Prepare()
	fl.Create(spot, nearHold*nearHold, false)
	fl.Finalize()
	if _, cnt := fl.GetShadowLights(); cnt != 1 {
		t.Fatalf("expected hysteresis to retain !isFront spotlight at effRadius+360, got count %d", cnt)
	}

	// Far beyond exit nearRadius while !isFront: must fall back cleanly to unshadowed.
	nearOut := effRadius + 500.0
	fl.DeepReset()
	fl.Prepare()
	fl.Create(spot, nearOut*nearOut, false)
	fl.Finalize()
	_, shadowCnt := fl.GetShadowLights()
	_, _, unshadowedCnt := fl.GetLights()
	if shadowCnt != 0 || unshadowedCnt != 1 {
		t.Fatalf("expected !isFront out-of-near-sphere spotlight to fall back unshadowed, got shadow=%d unshadowed=%d", shadowCnt, unshadowedCnt)
	}

	// Small radius spotlight at d=1600 with isFront=true must cast shadows up to 2048,
	// hold up to 2300 once active, and demote at 2700.
	fl.DeepReset()
	fl.Prepare()
	fl.Create(spot, 1600*1600, true)
	fl.Finalize()
	if _, cnt := fl.GetShadowLights(); cnt != 1 {
		t.Fatalf("expected small spotlight at d=1600 to stay shadowed, got %d", cnt)
	}

	fl.DeepReset()
	fl.Prepare()
	fl.Create(spot, 2300*2300, true)
	fl.Finalize()
	if _, cnt := fl.GetShadowLights(); cnt != 1 {
		t.Fatalf("expected range hysteresis to keep spotlight at d=2300, got %d", cnt)
	}

	fl.DeepReset()
	fl.Prepare()
	fl.Create(spot, 2700*2700, true)
	fl.Finalize()
	_, shadowCnt = fl.GetShadowLights()
	_, _, unshadowedCnt = fl.GetLights()
	if shadowCnt != 0 || unshadowedCnt != 1 {
		t.Fatalf("expected spotlight at d=2700 to demote to unshadowed, got shadow=%d unshadowed=%d", shadowCnt, unshadowedCnt)
	}
	// Styled flickering spotlight should use its unstyled base intensity for ranking so dim ticks don't drop it from top-8.
	fl = NewFrameLights(nil, 64, true)
	styledSpots := make([]*lights.Light, 9)
	for i := 0; i < 8; i++ {
		var style []float64
		if i == 7 {
			style = []float64{1.0, 0.2}
		}
		styledSpots[i] = makeSpotLight(float64((i+1)*100), 0, 0, 1.0, 100.0, style)
	}
	styledSpots[8] = makeSpotLight(900, 0, 0, 0.9, 100.0, nil)

	for (textures.GlobalTick()/6)%2 != 0 {
		textures.Tick()
	}
	for phase := 0; phase < 2; phase++ {
		if phase == 1 {
			for k := 0; k < 6; k++ {
				textures.Tick()
			}
			if styledSpots[7].GetIntensityStyled(textures.GlobalTick()) > 0.3 {
				t.Fatalf("expected dim style tick, got %f", styledSpots[7].GetIntensityStyled(textures.GlobalTick()))
			}
		}
		fl.DeepReset()
		fl.Prepare()
		for i := 0; i < 7; i++ {
			d := float64((i + 1) * 10)
			fl.Create(styledSpots[i], d*d, true)
		}
		// Flickering spot7 and steady rival spot8 at equal distance 400; base intensity (1.0 vs 0.9) keeps spot7 in top-8 on dim tick.
		fl.Create(styledSpots[7], 400*400, true)
		fl.Create(styledSpots[8], 400*400, true)
		fl.Finalize()

		sl, slCount := fl.GetShadowLights()
		slot := findShadowSlotByX(sl, slCount, 800)
		if slot < 0 {
			t.Fatalf("styled spotlight dropped from shadow slots at phase=%d (tick=%d)", phase, textures.GlobalTick())
		}
		if phase == 1 && sl[slot].Intensity > 0.3 {
			t.Fatalf("expected output visual intensity to reflect dim tick, got %f", sl[slot].Intensity)
		}
	}
}

func TestFrameLightsProcessNilSafeAndZeroAllocs(t *testing.T) {
	ctx := &Context{}
	bl := NewLights(ctx)
	bl.fl.useShadowLight = true

	ls := model.NewLights(nil)
	spots := make([]*lights.Light, 12)
	for i := range spots {
		spots[i] = makeSpotLight(float64(i*80), 0, 0, 1.0, 100.0, nil)
		ls.AddLight(spots[i])
	}

	// nil Lights clears cleanly
	bl.Process(nil, nil, 0, 0, 0, 2000, 2000, 2000)
	if _, cnt := bl.GetFrameLights().GetShadowLights(); cnt != 0 {
		t.Fatalf("expected 0 shadow lights when model.Lights is nil, got %d", cnt)
	}

	// Steady state frame allocation check (use Process over model.Lights with nil frustum -> tests near-sphere invariance + zero allocs)
	bl.Process(ls, nil, 0, 0, 0, 2000, 2000, 2000)
	allocs := testing.AllocsPerRun(100, func() {
		bl.Process(ls, nil, 10, 0, 0, 2000, 2000, 2000)
		_, _ = bl.GetFrameLights().GetShadowLights()
		_, _, _ = bl.GetFrameLights().GetLights()
	})
	if allocs > 0 {
		t.Fatalf("expected 0 steady-state allocations per frame, got %.1f", allocs)
	}
}

func TestBuilderFrustumFrontTracksWidescreenAspect(t *testing.T) {
	calCfg := config.NewConfigCalibration()
	calCfg.AspectRatio = 1.0
	calCfg.FovVerticalDegrees = 90.0
	calCfg.ZNear = 0.1
	calCfg.ZFar = 4096.0
	calCfg.ZLightFar = 4096.0

	cal := model.NewCalibration(calCfg, nil)
	cal.EnableShadowLights = true
	cal.PrjScale = 0.85

	bv := NewBuilder(nil, nil, cal, nil)
	vi := engine.NewViewMatrix()

	// Update frustum with 16:9 framebuffer (1280x720): edge spotlight at ~52 deg off center must remain inside frustumFront.
	bv.setupFrustumFront(1280, 720)
	px, py, pz := vi.GetView()
	bv.frustumFront.Update(float32(px), float32(py), float32(pz), vi.GetAngle(), -vi.GetPitch(), vi.GetRoll())

	edgeSpot := makeSpotLight(1000, -1250, 0, 1.0, 30.0, nil)
	frustum := vi.GetFrontFrustum(bv.frustumFront.Get())
	if !edgeSpot.GetAABB().IntersectFrustum(frustum) {
		t.Fatalf("expected widescreen 16:9 frustumFront to include lateral spotlight AABB (distX=1000, distY=1250)")
	}
}
