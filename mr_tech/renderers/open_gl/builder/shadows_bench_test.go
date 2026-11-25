package builder

import (
	"fmt"
	"os"
	"testing"
	"unsafe"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/engine"
	"github.com/markel1974/godoom/mr_tech/generators/common"
	"github.com/markel1974/godoom/mr_tech/generators/doom3"
	"github.com/markel1974/godoom/mr_tech/generators/quake"
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/model/things"
	"github.com/markel1974/godoom/mr_tech/physics/rigid_body"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/gpu/vramtex"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/metrics"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/shaders"
	"github.com/markel1974/godoom/mr_tech/textures"
)

type benchShadowCtx struct {
	shadowTestCtx
	bufferDataCalls    int
	bufferSubDataCalls int
	bufferUploadBytes  int
}

func (c *benchShadowCtx) resetUploadCounts() {
	c.bufferDataCalls = 0
	c.bufferSubDataCalls = 0
	c.bufferUploadBytes = 0
}

func (c *benchShadowCtx) BufferData(target uint32, size int, data unsafe.Pointer, usage uint32) {
	c.bufferDataCalls++
	if data != nil {
		c.bufferUploadBytes += size
	}
	c.shadowTestCtx.BufferData(target, size, data, usage)
}

func (c *benchShadowCtx) BufferSubData(target uint32, offset int, size int, data unsafe.Pointer) {
	c.bufferSubDataCalls++
	c.bufferUploadBytes += size
	c.shadowTestCtx.BufferSubData(target, offset, size, data)
}

func setupShadowBenchScene(b *testing.B, crateCount int) (*benchShadowCtx, *engine.Engine, *things.Player, *engine.ViewMatrix, *Builder, *shaders.Handler, *shaders.Depth) {
	b.Helper()
	rigid_body.SetFps(120)
	textures.SetFps(120)

	tA := textures.NewTexture("BENCH_W", 1, 8, 8, false)
	for x := range 8 {
		for y := range 8 {
			tA.SetDiffuse(x, y, 0x7f7f7fff)
		}
	}
	texH := &multiTexHandler{frames: []*textures.Texture{tA}}
	calCfg := config.NewConfigCalibration()
	calCfg.AspectRatio = 1.1
	calCfg.OrthoSize = 640

	matSolid := config.NewConfigMaterial("solid", []string{"BENCH_W"}, config.MaterialKindSolid)
	matCrate := config.NewConfigMaterial("crate", []string{"BENCH_W"}, config.MaterialKindSolid)

	secs := []*config.Sector{
		makeBoxSector("center", 0, 0, 600, 0, 160, matSolid),
	}

	p := config.NewConfigPlayer(geometry.XYZ{X: 0, Y: 0, Z: 20}, 0.0, 40, 1800, 16.0, 60.0)
	_ = common.NewPlayer(p)
	p.GForce = 9.8 * 50

	var ths []*config.Thing
	crateModel := makeRigidCrateModel(matCrate)
	for i := range crateCount {
		row := i / 16
		col := i % 16
		px := float64(80 + col*28)
		py := float64(-220 + row*28)
		crate := config.NewConfigItem(fmt.Sprintf("crate_%d", i), geometry.XYZ{X: px, Y: py, Z: 0}, float64(i)*0.1, 10, 0)
		crate.Radius, crate.Height, crate.GForce = 12, 24, 9.8*50
		crate.Element3D = crateModel
		ths = append(ths, crate)
	}

	root := config.NewConfigRoot(calCfg, secs, p, ths, geometry.XYZ{X: 1, Y: 1, Z: 1}, texH)
	en := engine.NewEngine()
	if err := en.Setup(root); err != nil {
		b.Fatalf("engine setup failed: %v", err)
	}

	pl := en.GetPlayer()
	vi := engine.NewViewMatrix()
	vi.Update(pl)

	rCtx := &benchShadowCtx{}
	glTex := vramtex.NewTextures(rCtx)
	if err := glTex.Setup(en.GetTextures()); err != nil {
		b.Fatalf("textures setup failed: %v", err)
	}

	mCal := en.GetCalibration()
	mCal.EnableShadows = true
	mCal.EnableShadowLights = true
	mCal.MSAASamples = 0

	bv := NewBuilder(rCtx, glTex, mCal, pl)
	h := shaders.NewHandler(bv.GetVerticesStride(), bv.GetLightsStride())
	if err := h.Setup(rCtx, mCal, pl.GetFlash(), 640, 480); err != nil {
		b.Fatalf("Handler.Setup: %v", err)
	}
	if err := h.GetMain().Init(); err != nil {
		b.Fatalf("Main.Init: %v", err)
	}
	d := h.GetDepth()
	if err := d.Compile(testShaderAssets{}); err != nil {
		b.Fatalf("Depth.Compile: %v", err)
	}
	d.SetShadowsEnabled(true)

	mm := h.GetMapMetrics()
	mm.SetOrthoSize(640, 0.1, 4096)
	mm.SetCenter(0, 0, 0)
	mm.Update(vi, 1.0, 1.0)
	h.GetFlashlightMetrics().UpdateWithAim(mm.GetViewPtr(), 0, 0, 0, 0, 1)

	var sl [metrics.ShadowLightsCount]*metrics.Light
	for i := range metrics.ShadowLightsCount {
		sl[i] = metrics.NewLight()
		sl[i].Assign(float32(150+i*25), 60, float32(-60+i*15), 1, 1, 1, 1, 2.0, 1, -0.35, 0, 100, 0.9, 0.7, 1.0)
	}
	h.GetShadowLightsMetrics().Update(sl, metrics.ShadowLightsCount)

	// Warm up both double-buffered atlas targets.
	for range 2 {
		bv.Compute(640, 480, vi, en)
		d.Prepare()
		bv.ComputeShadows(en, d)
		d.RenderWithTiles(bv.GetDrawCommandsOpaque(), bv.GetShadowTiles())
	}
	return rCtx, en, pl, vi, bv, h, d
}

func BenchmarkComputeShadowsRigid256(b *testing.B) {
	rCtx, en, _, vi, bv, h, d := setupShadowBenchScene(b, 256)
	mm := h.GetMapMetrics()
	fm := h.GetFlashlightMetrics()

	b.Run("SteadyCached", func(b *testing.B) {
		b.ReportAllocs()
		rCtx.resetUploadCounts()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			d.Prepare()
			bv.ComputeShadows(en, d)
			d.RenderWithTiles(bv.GetDrawCommandsOpaque(), bv.GetShadowTiles())
		}
		b.ReportMetric(float64(rCtx.bufferSubDataCalls)/float64(b.N), "subdata/op")
		b.ReportMetric(float64(rCtx.bufferUploadBytes)/float64(b.N), "upBytes/op")
	})

	b.Run("FlashRotateOnly", func(b *testing.B) {
		b.ReportAllocs()
		rCtx.resetUploadCounts()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			aim := float32((i%16)-8) * 0.002
			fm.UpdateWithAim(mm.GetViewPtr(), aim, 0, aim, 0, 1)
			d.Prepare()
			bv.ComputeShadows(en, d)
			d.RenderWithTiles(bv.GetDrawCommandsOpaque(), bv.GetShadowTiles())
		}
		b.ReportMetric(float64(rCtx.bufferSubDataCalls)/float64(b.N), "subdata/op")
		b.ReportMetric(float64(rCtx.bufferUploadBytes)/float64(b.N), "upBytes/op")
	})

	b.Run("RoomAndFlashMove", func(b *testing.B) {
		b.ReportAllocs()
		rCtx.resetUploadCounts()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			cx := float32((i%32)-16) * 0.1
			mm.SetCenter(cx, 0, 0)
			mm.Update(vi, 1.0, 1.0)
			fm.UpdateWithAim(mm.GetViewPtr(), cx*0.01, 0, cx*0.01, 0, 1)
			d.Prepare()
			bv.ComputeShadows(en, d)
			d.RenderWithTiles(bv.GetDrawCommandsOpaque(), bv.GetShadowTiles())
		}
		b.ReportMetric(float64(rCtx.bufferSubDataCalls)/float64(b.N), "subdata/op")
		b.ReportMetric(float64(rCtx.bufferUploadBytes)/float64(b.N), "upBytes/op")
	})
}

func BenchmarkComputeShadowsRealMaps(b *testing.B) {
	cases := []struct {
		name string
		pak  string
		kind string
	}{
		{"quake2", "../../../../build/resources/quake2/pak0.pak", "q2"},
		{"doom3", "../../../../build/resources/doom3/pak000.pk4", "d3"},
	}

	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			if _, err := os.Stat(tc.pak); err != nil {
				b.Skipf("optional external resources not found at %s: %v", tc.pak, err)
			}
			rigid_body.SetFps(120)
			textures.SetFps(120)
			res := common.NewResourcesOs()
			if err := res.Setup(); err != nil {
				b.Fatalf("res setup: %v", err)
			}
			var (
				root *config.Root
				err  error
			)
			if tc.kind == "q2" {
				root, err = quake.NewBuilder().Setup(res, tc.pak, 0)
			} else {
				root, err = doom3.NewBuilder().Setup(res, tc.pak, 0)
			}
			if err != nil {
				b.Fatalf("build %s failed: %v", tc.name, err)
			}
			en := engine.NewEngine()
			if err := en.Setup(root); err != nil {
				b.Fatalf("engine setup %s failed: %v", tc.name, err)
			}
			pl := en.GetPlayer()
			vi := engine.NewViewMatrix()
			en.Compute(pl, vi)

			rCtx := &benchShadowCtx{}
			glTex := vramtex.NewTextures(rCtx)
			if err := glTex.Setup(en.GetTextures()); err != nil {
				b.Fatalf("textures setup: %v", err)
			}
			mCal := en.GetCalibration()
			mCal.EnableShadows = true
			mCal.EnableShadowLights = true
			mCal.MSAASamples = 0

			bv := NewBuilder(rCtx, glTex, mCal, pl)
			h := shaders.NewHandler(bv.GetVerticesStride(), bv.GetLightsStride())
			if err := h.Setup(rCtx, mCal, pl.GetFlash(), 640, 480); err != nil {
				b.Fatalf("Handler.Setup: %v", err)
			}
			if err := h.GetMain().Init(); err != nil {
				b.Fatalf("Main.Init: %v", err)
			}
			d := h.GetDepth()
			if err := d.Compile(testShaderAssets{}); err != nil {
				b.Fatalf("Depth.Compile: %v", err)
			}
			d.SetShadowsEnabled(true)

			mm := h.GetMapMetrics()
			fm := h.GetFlashlightMetrics()
			mm.SetOrthoSize(float32(mCal.OrthoSize), float32(mCal.ZNear), float32(mCal.ZFar)+4.0)
			mm.SetCenter(float32(mCal.MapCenterX), float32(mCal.MapCenterZ), float32(mCal.LightCamY)+2.0)
			sx, sy := mm.GetScale3d(640, 480, float32(mCal.AspectRatio), float32(mCal.FovVerticalDegrees))
			mm.Update(vi, sx, sy)
			fm.UpdateWithAim(mm.GetViewPtr(), 0, 0, 0, 0, 1)

			bv.Compute(640, 480, vi, en)
			shadowLights, shadowLightsCount := bv.GetShadowLights()
			h.GetShadowLightsMetrics().Update(shadowLights, uint32(shadowLightsCount))
			for range 2 {
				d.Prepare()
				bv.ComputeShadows(en, d)
				d.RenderWithTiles(bv.GetDrawCommandsOpaque(), bv.GetShadowTiles())
			}

			b.ReportAllocs()
			rCtx.resetUploadCounts()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				aim := float32((i%16)-8) * 0.002
				fm.UpdateWithAim(mm.GetViewPtr(), aim, 0, aim, 0, 1)
				d.Prepare()
				bv.ComputeShadows(en, d)
				d.RenderWithTiles(bv.GetDrawCommandsOpaque(), bv.GetShadowTiles())
			}
			b.ReportMetric(float64(rCtx.bufferSubDataCalls)/float64(b.N), "subdata/op")
			b.ReportMetric(float64(rCtx.bufferUploadBytes)/float64(b.N), "upBytes/op")
		})
	}
}
