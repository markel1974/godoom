package builder

import (
	"math"
	"testing"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/engine"
	"github.com/markel1974/godoom/mr_tech/generators/common"
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/model"
	"github.com/markel1974/godoom/mr_tech/model/lights"
	"github.com/markel1974/godoom/mr_tech/physics/rigid_body"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/gpu/vramtex"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/metrics"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/shaders"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/shaders/atlas"
	"github.com/markel1974/godoom/mr_tech/textures"
)

func TestDirectionalLightsGlobalPickupAndCSMSuppression(t *testing.T) {
	sunCfg := config.NewConfigLight(geometry.XYZ{X: 0, Y: 0, Z: 0}, 2.4, config.LightKindDirectional, 0)
	sunCfg.R, sunCfg.G, sunCfg.B = 1.0, 0.92, 0.80
	sunCfg.DirX, sunCfg.DirY, sunCfg.DirZ = 0.35, -0.45, -0.82

	sunLight := lights.NewLight()
	sunLight.Setup(sunCfg, sunCfg.Pos)
	ls := model.NewLights([]*lights.Light{sunLight})

	// Case 1: SunCSM disabled -> directional light MUST still be picked up far from (0,0,0).
	calOff := &model.Calibration{EnableSunCSM: false}
	bcOff := NewContextWithOcclusionMax(&shadowTestCtx{}, calOff, nil, nil, 0)
	blOff := NewLights(bcOff)
	blOff.Process(ls, nil, 12000, -9000, 150, 2000, 2000, 2000)

	_, _, countOff := blOff.GetFrameLights().GetLights()
	if countOff != 1 {
		t.Fatalf("expected directional light collected globally far from origin when CSM off, got count=%d", countOff)
	}
	if _, _, ok := blOff.GetFrameLights().GetSceneDirectional(); !ok {
		t.Fatalf("expected GetSceneDirectional ok=true")
	}

	// Case 2: SunCSM enabled -> omitted from clustered UBO to avoid double-lighting, exposed via GetSceneDirectional.
	calOn := &model.Calibration{EnableSunCSM: true}
	bcOn := NewContextWithOcclusionMax(&shadowTestCtx{}, calOn, nil, nil, 0)
	blOn := NewLights(bcOn)
	blOn.Process(ls, nil, 12000, -9000, 150, 2000, 2000, 2000)

	_, _, countOn := blOn.GetFrameLights().GetLights()
	if countOn != 0 {
		t.Fatalf("expected directional light suppressed in UBO when CSM owns sun shading, got count=%d", countOn)
	}
	dirGL, colInt, ok := blOn.GetFrameLights().GetSceneDirectional()
	if !ok {
		t.Fatalf("expected GetSceneDirectional ok=true when CSM on")
	}
	// Engine Z-up (0.35, -0.45, -0.82) -> GL Y-up (0.35, -0.82, +0.45)
	if math.Abs(float64(dirGL[0]-0.35)) > 1e-4 || math.Abs(float64(dirGL[1]+0.82)) > 1e-4 || math.Abs(float64(dirGL[2]-0.45)) > 1e-4 {
		t.Fatalf("unexpected GL sun dir %v", dirGL)
	}
	if math.Abs(float64(colInt[3]-2.4)) > 1e-4 {
		t.Fatalf("unexpected sun intensity %v", colInt[3])
	}
}

func TestShadowsBuilderUsesSunCSMTilesWhenEnabled(t *testing.T) {
	rigid_body.SetFps(60)
	textures.SetFps(60)

	tex := textures.NewTexture("SUN_T", 1, 8, 8, false)
	for x := range 8 {
		for y := range 8 {
			tex.SetDiffuse(x, y, 0x999999ff)
		}
	}
	texH := &multiTexHandler{frames: []*textures.Texture{tex}}

	calCfg := config.NewConfigCalibration()
	calCfg.AspectRatio = 1.0
	calCfg.ZNear = 1.0
	calCfg.ZFar = 2048.0
	calCfg.EnableSunCSM = true
	calCfg.EnableHorizonFog = true

	matSolid := config.NewConfigMaterial("solid", []string{"SUN_T"}, config.MaterialKindSolid)
	secs := []*config.Sector{
		makeBoxSector("street", 0, 0, 400, 0, 180, matSolid),
	}
	p := config.NewConfigPlayer(geometry.XYZ{X: 0, Y: 0, Z: 20}, 0, 40, 1800, 16, 60)
	_ = common.NewPlayer(p)

	crate := config.NewConfigItem("car", geometry.XYZ{X: 120, Y: 10, Z: 0}, 0, 10, 0)
	crate.Radius, crate.Height, crate.GForce = 16, 24, 9.8*50
	crate.Element3D = makeRigidCrateModel(matSolid)

	root := config.NewConfigRoot(calCfg, secs, p, []*config.Thing{crate}, geometry.XYZ{X: 1, Y: 1, Z: 1}, texH)
	en := engine.NewEngine()
	if err := en.Setup(root); err != nil {
		t.Fatalf("engine setup failed: %v", err)
	}

	pl := en.GetPlayer()
	vi := engine.NewViewMatrix()
	vi.Update(pl)

	rCtx := &shadowTestCtx{}
	glTex := vramtex.NewTextures(rCtx)
	if err := glTex.Setup(en.GetTextures()); err != nil {
		t.Fatalf("textures setup failed: %v", err)
	}
	mCal := en.GetCalibration()
	mCal.EnableShadows = true
	mCal.MSAASamples = 0

	bv := NewBuilder(rCtx, glTex, mCal, pl)
	h := shaders.NewHandler(bv.GetVerticesStride(), bv.GetLightsStride())
	if err := h.Setup(rCtx, mCal, pl.GetFlash(), 800, 600); err != nil {
		t.Fatalf("Handler.Setup: %v", err)
	}
	if err := h.GetMain().Init(); err != nil {
		t.Fatalf("Main.Init: %v", err)
	}
	d := h.GetDepth()
	if err := d.Compile(testShaderAssets{}); err != nil {
		t.Fatalf("Depth.Compile: %v", err)
	}
	d.SetShadowsEnabled(true)

	mm := h.GetMapMetrics()
	sx, sy := mm.GetScale3d(800, 600, float32(mCal.AspectRatio), float32(mCal.FovVerticalDegrees))
	mm.Update(vi, sx, sy)
	h.UpdateSunAndFog(sx, sy)

	// Warm both atlas double buffers.
	for range 2 {
		bv.Compute(800, 600, vi, en)
		d.Prepare()
		bv.ComputeShadows(en, d)
		d.RenderWithTiles(bv.GetDrawCommandsOpaque(), bv.GetShadowTiles())
	}

	tiles := bv.GetShadowTiles()
	if tiles.Get(atlas.ShadowTileRoom).IsActive() {
		t.Fatalf("expected ShadowTileRoom inactive when EnableSunCSM=true")
	}
	for c := range metrics.SunCascadeCount {
		slot := atlas.ShadowTileSunBase + c
		tile := tiles.Get(slot)
		if !tile.IsActive() || tile.GetRenderDC() == nil {
			t.Fatalf("expected active SunCSM cascade tile %d with RenderDC", slot)
		}
	}

	// Next identical frame must hit cached reuse across all 4 CSM cascades.
	bv.Compute(800, 600, vi, en)
	d.Prepare()
	bv.ComputeShadows(en, d)
	d.RenderWithTiles(bv.GetDrawCommandsOpaque(), bv.GetShadowTiles())

	for c := range metrics.SunCascadeCount {
		slot := atlas.ShadowTileSunBase + c
		tile := tiles.Get(slot)
		if !tile.IsActive() || !tile.CanReuse() {
			t.Fatalf("expected cached Reuse=true on steady SunCSM cascade %d", slot)
		}
	}
}
