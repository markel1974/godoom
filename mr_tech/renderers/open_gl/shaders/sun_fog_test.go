package shaders

import (
	"strings"
	"testing"
	"unsafe"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/model"
	"github.com/markel1974/godoom/mr_tech/model/things"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/gpu/draws"
)

func TestAtlasSunCSMTileRectsAndDepthConfig(t *testing.T) {
	ctx := newRecordingCtx()
	calCfg := config.NewConfigCalibration()
	calCfg.EnableSunCSM = true
	calCfg.EnableHorizonFog = true
	cal := model.NewCalibration(calCfg, nil)
	cal.ZNear = 1.0
	cal.ZFar = 4096.0
	cal.MSAASamples = 0

	flash := things.NewFlash(config.NewFlash())
	h := NewHandler(80, 64)
	if err := h.Setup(ctx, cal, flash, 1280, 720); err != nil {
		t.Fatalf("Handler.Setup failed: %v", err)
	}
	if err := h.GetMain().Init(); err != nil {
		t.Fatalf("Main.Init failed: %v", err)
	}
	d := h.GetDepth()
	if err := d.Compile(diskAssets{}); err != nil {
		t.Fatalf("Depth.Compile failed: %v", err)
	}
	d.SetShadowsEnabled(true)

	a := d.GetAtlas()
	wantRects := [SunCascadeCount][4]int32{
		{0, 1024, 512, 512},
		{512, 1024, 512, 512},
		{0, 1536, 512, 512},
		{512, 1536, 512, 512},
	}
	for c := range SunCascadeCount {
		x, y, w, hh, clamp, _, _ := a.TileRect(ShadowTileSunBase + c)
		got := [4]int32{x, y, w, hh}
		if got != wantRects[c] || !clamp {
			t.Fatalf("Sun cascade %d TileRect = %v clamp=%v, want %v true", c, got, clamp, wantRects[c])
		}
	}

	h.GetMapMetrics().Update(makeRotatedViewMatrix(t), 1.2, 1.5)
	h.UpdateSunAndFog(1.2, 1.5)
	d.Prepare()

	// Sun CSM active: ShadowTileRoom must be inactive, cascades 0..3 must be active with depthClamp.
	if active, _, _ := d.GetTileConfig(ShadowTileRoom); active {
		t.Fatalf("expected ShadowTileRoom inactive when SunCSM is enabled")
	}

	solidMat, _ := makeSolidAndCutoutMaterials(t)
	cmd := draws.NewDrawCommand()
	cmd.SetFirstIndex(0)
	cmd.SetIndexCount(6)
	cmd.SetMaterial(solidMat)
	dc := draws.NewDrawCommandsRender(ctx, "sun_csm", false)
	dc.Prepare([]*draws.DrawCommand{cmd})

	var tiles [MaxShadowTiles]ShadowsTileData
	fillSunTiles := func(reuse bool) {
		for c := range SunCascadeCount {
			slot := ShadowTileSunBase + c
			active, clamp, matPtr := d.GetTileConfig(slot)
			if !active || !clamp || matPtr == nil {
				t.Fatalf("expected cascade %d active with clamp & matrix", slot)
			}
			var mat [16]float32
			copy(mat[:], (*[16]float32)(unsafe.Pointer(matPtr))[:])
			tiles[slot] = ShadowsTileData{
				Active:    true,
				RenderDC:  dc,
				Matrix:    mat,
				SceneHash: uint64(1000 + c),
				Reuse:     reuse && d.CanReuseTile(slot, &mat, uint64(1000+c)),
			}
		}
	}

	// Warm both double buffers -> 4 clears per buffer -> then steady frame hits 0 clears, 0 draws, 0 allocs!
	for range depthDoubleBuffer {
		fillSunTiles(false)
		ctx.clearCalls = 0
		ctx.multiDrawPrograms = nil
		d.RenderWithTiles(nil, &tiles)
		if ctx.clearCalls != SunCascadeCount || len(ctx.multiDrawPrograms) != SunCascadeCount {
			t.Fatalf("cold cascade render expected %d clears/%d draws, got clears=%d draws=%d",
				SunCascadeCount, SunCascadeCount, ctx.clearCalls, len(ctx.multiDrawPrograms))
		}
	}

	fillSunTiles(true)
	ctx.clearCalls = 0
	ctx.multiDrawPrograms = nil
	d.RenderWithTiles(nil, &tiles)
	if ctx.clearCalls != 0 || len(ctx.multiDrawPrograms) != 0 {
		t.Fatalf("expected cached CSM tiles to skip draws, got clears=%d draws=%d", ctx.clearCalls, len(ctx.multiDrawPrograms))
	}

	// Disabling SunCSM switches back cleanly to ShadowTileRoom and disables cascades.
	h.SetSunEnabled(false)
	d.Prepare()
	if active, _, _ := d.GetTileConfig(ShadowTileRoom); !active {
		t.Fatalf("expected ShadowTileRoom re-activated when SunCSM disabled")
	}
	if active, _, _ := d.GetTileConfig(ShadowTileSunBase); active {
		t.Fatalf("expected SunCSM cascade inactive when disabled")
	}
}

func TestSunCSMAndHorizonFogShadersCompileAndRenderPasses(t *testing.T) {
	assets := diskAssets{}

	opaqueBytes, err := assets.Read("main_opaque.frag")
	if err != nil {
		t.Fatalf("Read main_opaque.frag failed: %v", err)
	}
	opaque := string(opaqueBytes)
	for _, token := range []string{
		"sampleSunCSMShadow",
		"selectSunMatrix",
		"computeHorizonFog",
		"u_sunSpaceMatrices[4]",
		"u_sunCascadeSplits",
		"u_fogParams",
		"u_fogHeightParams",
	} {
		if !strings.Contains(opaque, token) {
			t.Fatalf("main_opaque.frag missing token %q", token)
		}
	}
	if strings.Contains(opaque, "u_sunSpaceMatrices[c]") {
		t.Fatalf("main_opaque.frag must use selectSunMatrix(c) instead of dynamic uniform matrix array indexing for WebGL2 parity")
	}

	skyBytes, err := assets.Read("sky.frag")
	if err != nil {
		t.Fatalf("Read sky.frag failed: %v", err)
	}
	for _, token := range []string{"u_hasSkyTex", "u_sunEnabled", "u_fogColor", "u_fogSunColor", "u_fogScatter"} {
		if !strings.Contains(string(skyBytes), token) {
			t.Fatalf("sky.frag missing token %q", token)
		}
	}

	liqBytes, err := assets.Read("main_liquid.frag")
	if err != nil {
		t.Fatalf("Read main_liquid.frag failed: %v", err)
	}
	if !strings.Contains(string(liqBytes), "emitLiquidOutput") || !strings.Contains(string(liqBytes), "u_fogParams") {
		t.Fatalf("main_liquid.frag missing horizon fog integration")
	}

	addBytes, err := assets.Read("main_additive.frag")
	if err != nil {
		t.Fatalf("Read main_additive.frag failed: %v", err)
	}
	if !strings.Contains(string(addBytes), "u_fogEnabled") || !strings.Contains(string(addBytes), "u_fogParams") {
		t.Fatalf("main_additive.frag missing horizon fog attenuation")
	}

	ctx := newRecordingCtx()
	calCfg := config.NewConfigCalibration()
	calCfg.EnableSunCSM = true
	calCfg.EnableHorizonFog = true
	calCfg.ZNear = 1.0
	calCfg.ZFar = 4096.0
	cal := model.NewCalibration(calCfg, nil)
	cal.MSAASamples = 0

	h := NewHandler(80, 64)
	if err := h.Setup(ctx, cal, things.NewFlash(config.NewFlash()), 1280, 720); err != nil {
		t.Fatalf("Handler.Setup failed: %v", err)
	}
	for _, s := range []struct {
		compile func(IAssets) error
		init    func() error
		sampl   func() error
	}{
		{h.GetMain().Compile, h.GetMain().Init, h.GetMain().SetupSamplers},
		{h.GetSky().Compile, h.GetSky().Init, h.GetSky().SetupSamplers},
		{h.GetLiquid().Compile, h.GetLiquid().Init, h.GetLiquid().SetupSamplers},
		{h.GetAdditive().Compile, h.GetAdditive().Init, h.GetAdditive().SetupSamplers},
	} {
		if err := s.compile(assets); err != nil {
			t.Fatalf("Compile failed: %v", err)
		}
		if err := s.init(); err != nil {
			t.Fatalf("Init failed: %v", err)
		}
		if err := s.sampl(); err != nil {
			t.Fatalf("SetupSamplers failed: %v", err)
		}
	}
	h.GetPost().EnsureAllocated()

	h.GetMapMetrics().Update(makeRotatedViewMatrix(t), 1.2, 1.5)
	h.UpdateSunAndFog(1.2, 1.5)

	ctx.drawArraysPrograms = nil
	h.GetSky().RenderProcedural()
	h.GetSky().Render(1.0, 0, 0)
	if len(ctx.drawArraysPrograms) != 2 {
		t.Fatalf("expected 2 sky draw calls (procedural + textured), got %d", len(ctx.drawArraysPrograms))
	}
	h.GetLiquid().Render(func() {})
	h.GetAdditive().RenderAdditive(func() {})
}
