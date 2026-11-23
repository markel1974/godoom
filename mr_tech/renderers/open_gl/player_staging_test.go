package open_gl

import (
	"math"
	"os"
	"testing"
	"unsafe"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/engine"
	"github.com/markel1974/godoom/mr_tech/generators/common"
	"github.com/markel1974/godoom/mr_tech/generators/jedi"
	"github.com/markel1974/godoom/mr_tech/generators/quake"
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/model"
	"github.com/markel1974/godoom/mr_tech/model/things"
	"github.com/markel1974/godoom/mr_tech/physics/rigid_body"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/gpu"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/metrics"
	"github.com/markel1974/godoom/mr_tech/textures"
)

type stubTexHandler struct{ tex *textures.Texture }

func (m *stubTexHandler) GetNames() []string { return []string{"WALL"} }
func (m *stubTexHandler) GetFrames(_ []string) []*textures.Texture {
	return []*textures.Texture{m.tex}
}

type stubCtx struct{ api.IContext }

func (stubCtx) Ptr(_ interface{}) unsafe.Pointer    { return nil }
func (stubCtx) PtrOffset(offset int) unsafe.Pointer { return unsafe.Pointer(uintptr(offset)) }
func (stubCtx) GenVertexArrays(_ int32, a *uint32) {
	if a != nil {
		*a = 1
	}
}
func (stubCtx) GenBuffers(_ int32, b *uint32) {
	if b != nil {
		*b = 1
	}
}
func (stubCtx) DeleteVertexArrays(_ int32, _ *uint32)                  {}
func (stubCtx) DeleteBuffers(_ int32, _ *uint32)                       {}
func (stubCtx) BindVertexArray(_ uint32)                               {}
func (stubCtx) BindBuffer(_ uint32, _ uint32)                          {}
func (stubCtx) BufferData(_ uint32, _ int, _ unsafe.Pointer, _ uint32) {}
func (stubCtx) BufferSubData(_ uint32, _ int, _ int, _ unsafe.Pointer) {}
func (stubCtx) VertexAttribPointer(_ uint32, _ int32, _ uint32, _ bool, _ int32, _ unsafe.Pointer) {
}
func (stubCtx) VertexAttribDivisor(_ uint32, _ uint32) {}
func (stubCtx) EnableVertexAttribArray(_ uint32)       {}
func (stubCtx) GenTextures(_ int32, t *uint32) {
	if t != nil {
		*t = 1
	}
}
func (stubCtx) GenQueries(_ int32, _ *uint32)        {}
func (stubCtx) BindTexture(_ uint32, _ uint32)       {}
func (stubCtx) TexParameteri(_, _ uint32, _ int32)   {}
func (stubCtx) TexParameterf(_, _ uint32, _ float32) {}
func (stubCtx) GetFloatv(_ uint32, data *float32) {
	if data != nil {
		*data = 1.0
	}
}
func (stubCtx) TexImage3D(_ uint32, _ int32, _ int32, _, _, _, _ int32, _, _ uint32, _ unsafe.Pointer) {
}
func (stubCtx) TexSubImage3D(_ uint32, _ int32, _, _, _, _, _, _ int32, _, _ uint32, _ unsafe.Pointer) {
}
func (stubCtx) GenerateMipmap(_ uint32) {}

func newTestBuilderVolume(cal *model.Calibration, pl *things.Player) *BuilderVolume {
	dcOpaque := gpu.NewDrawCommands(config.BlendModeOpaque, 32768)
	dcAdditive := gpu.NewDrawCommands(config.BlendModeAdditive, 4096)
	dcLiquid := gpu.NewDrawCommands(config.BlendModeLiquid, 4096)
	modes := make([]*gpu.DrawCommands, config.BlendModeLatest)
	modes[dcOpaque.GetBlendMode()] = dcOpaque
	modes[dcAdditive.GetBlendMode()] = dcAdditive
	modes[dcLiquid.GetBlendMode()] = dcLiquid

	var plID uint64
	if pl != nil {
		plID = pl.GetEntity().GetId()
	}

	return &BuilderVolume{
		ctx:        stubCtx{},
		tex:        gpu.NewTextures(stubCtx{}),
		fv:         gpu.NewFrameVertices(stubCtx{}, 1048576),
		dcOpaque:   dcOpaque,
		dcAdditive: dcAdditive,
		dcLiquid:   dcLiquid,
		modes:      modes,
		cal:        cal,
		occlusion:  gpu.NewOcclusionHW(stubCtx{}, 8),
		player:     pl,
		playerId:   plID,
	}
}

func (w *BuilderVolume) resetTestFrame() {
	w.fv.Reset()
	w.dcOpaque.Reset()
	w.dcAdditive.Reset()
	w.dcLiquid.Reset()
	w.occlusion.Reset()
}

func TestPushQThingsAlwaysStagesPlayerWhenOccludedOrOutsideFrustum(t *testing.T) {
	rigid_body.SetFps(120)
	defer rigid_body.SetFps(120)

	texH := &stubTexHandler{tex: textures.NewTexture("WALL", 1, 64, 64, false)}
	cal := config.NewConfigCalibration()
	cal.AspectRatio = 1.1
	scale := geometry.XYZ{X: 1, Y: 1, Z: 1}

	mat := config.NewConfigMaterial("m", []string{"WALL"}, config.MaterialKindSolid)
	sec := config.NewConfigSector("0", 1.0, config.LightKindAmbient, 100)
	sec.FloorY = 0
	sec.CeilY = 200
	sec.Floor = mat
	sec.Ceil = mat
	sec.Segments = []*config.Segment{
		func() *config.Segment {
			s := config.NewConfigSegment("0", config.SegmentWall, geometry.XY{X: -500, Y: -500}, geometry.XY{X: 500, Y: -500})
			s.Middle = mat
			return s
		}(),
		func() *config.Segment {
			s := config.NewConfigSegment("0", config.SegmentWall, geometry.XY{X: 500, Y: -500}, geometry.XY{X: 500, Y: 500})
			s.Middle = mat
			return s
		}(),
		func() *config.Segment {
			s := config.NewConfigSegment("0", config.SegmentWall, geometry.XY{X: 500, Y: 500}, geometry.XY{X: -500, Y: 500})
			s.Middle = mat
			return s
		}(),
		func() *config.Segment {
			s := config.NewConfigSegment("0", config.SegmentWall, geometry.XY{X: -500, Y: 500}, geometry.XY{X: -500, Y: -500})
			s.Middle = mat
			return s
		}(),
	}

	p := config.NewConfigPlayer(geometry.XYZ{X: 0, Y: 0, Z: 4}, 0.0, 40, 1800, 2.5, 66.0)
	_ = common.NewPlayer(p)
	p.GForce = 9.8 * 50
	p.JumpForce = p.GForce * 8
	p.Impulse = 0.13

	root := config.NewConfigRoot(cal, []*config.Sector{sec}, p, nil, scale, texH)
	en := engine.NewEngine()
	if err := en.Setup(root); err != nil {
		t.Fatalf("engine setup failed: %v", err)
	}

	pl := en.GetPlayer()
	th := en.GetThings()
	vi := engine.NewViewMatrix()
	mCal := en.GetCalibration()

	bv := newTestBuilderVolume(mCal, pl)
	// Simulate OcclusionHW having marked the player invisible on an earlier readback.
	bv.occlusion.GetState(pl.GetEntity().GetId()).IsVisible = false

	// Place frustum far away so the player AABB is also outside QueryFrustum.
	ff := metrics.NewFrustum()
	ff.Setup(float32(mCal.AspectRatio), float32(mCal.ZFar), float32(mCal.ZNear), mCal.PrjScale, mCal.PrjScale)
	ff.Update(10000, 10000, 10000, 0, 0, 0)
	farFrustum := vi.GetFrontFrustum(ff.Get())

	startX, startY, _ := pl.GetEntity().GetCenter()
	for i := 0; i < 60; i++ {
		bv.resetTestFrame()
		pl.Move(0.13, true, false, false, false)
		bv.pushQThings(th, farFrustum)
		en.Compute(pl, vi)
	}
	endX, endY, _ := pl.GetEntity().GetCenter()
	moved := math.Hypot(endX-startX, endY-startY)
	if moved < 20.0 {
		t.Fatalf("player stalled when occluded/outside frustum: moved=%.4f (start=(%.2f,%.2f) end=(%.2f,%.2f))", moved, startX, startY, endX, endY)
	}
	if bv.occlusion.HasDrawTests() {
		t.Fatalf("expected invisible player not to queue OcclusionHW tests")
	}
}

func TestPlayerBobbingAndLandingBounceActiveThroughCoordinator(t *testing.T) {
	rigid_body.SetFps(120)
	defer rigid_body.SetFps(120)

	texH := &stubTexHandler{tex: textures.NewTexture("WALL", 1, 64, 64, false)}
	cal := config.NewConfigCalibration()
	cal.AspectRatio = 1.1
	scale := geometry.XYZ{X: 1, Y: 1, Z: 1}

	mat := config.NewConfigMaterial("m", []string{"WALL"}, config.MaterialKindSolid)
	sec := config.NewConfigSector("0", 1.0, config.LightKindAmbient, 100)
	sec.FloorY = 0
	sec.CeilY = 400
	sec.Floor = mat
	sec.Ceil = mat
	sec.Segments = []*config.Segment{
		func() *config.Segment {
			s := config.NewConfigSegment("0", config.SegmentWall, geometry.XY{X: -500, Y: -500}, geometry.XY{X: 500, Y: -500})
			s.Middle = mat
			return s
		}(),
		func() *config.Segment {
			s := config.NewConfigSegment("0", config.SegmentWall, geometry.XY{X: 500, Y: -500}, geometry.XY{X: 500, Y: 500})
			s.Middle = mat
			return s
		}(),
		func() *config.Segment {
			s := config.NewConfigSegment("0", config.SegmentWall, geometry.XY{X: 500, Y: 500}, geometry.XY{X: -500, Y: 500})
			s.Middle = mat
			return s
		}(),
		func() *config.Segment {
			s := config.NewConfigSegment("0", config.SegmentWall, geometry.XY{X: -500, Y: 500}, geometry.XY{X: -500, Y: -500})
			s.Middle = mat
			return s
		}(),
	}

	// Start player high above floor so landing triggers vertical spring bounce, then walk for head bobbing.
	p := config.NewConfigPlayer(geometry.XYZ{X: 0, Y: 0, Z: 150}, 0.0, 40, 1800, 16.0, 60.0)
	_ = common.NewPlayer(p)
	p.GForce = 9.8 * 50
	p.JumpForce = p.GForce * 8
	p.Impulse = 0.13
	p.Bobbing.MaxAmplitudeX = 5.0
	p.Bobbing.MaxAmplitudeY = 5.5
	p.Bobbing.StrideLength = 0.07
	p.Bobbing.SpeedLerp = 0.30
	p.Bobbing.AmpLerp = 0.20
	p.Bobbing.ImpactMax = 1000.0
	p.Bobbing.ImpactScale = 0.02
	p.Bobbing.SpringTension = 0.20
	p.Bobbing.SpringDamping = 0.80

	root := config.NewConfigRoot(cal, []*config.Sector{sec}, p, nil, scale, texH)
	en := engine.NewEngine()
	if err := en.Setup(root); err != nil {
		t.Fatalf("engine setup failed: %v", err)
	}

	pl := en.GetPlayer()
	th := en.GetThings()
	vi := engine.NewViewMatrix()
	bv := newTestBuilderVolume(en.GetCalibration(), pl)
	ff := metrics.NewFrustum()
	ff.Setup(float32(en.GetCalibration().AspectRatio), float32(en.GetCalibration().ZFar), float32(en.GetCalibration().ZNear), en.GetCalibration().PrjScale, en.GetCalibration().PrjScale)

	minEyeRelZ := math.MaxFloat64
	landed := false
	for i := 0; i < 180; i++ {
		bv.resetTestFrame()
		bv.pushQThings(th, vi.GetFrontFrustum(ff.Get()))
		en.Compute(pl, vi)
		if pl.GetEntity().IsOnGround() {
			landed = true
			_, _, cz := pl.GetEntity().GetCenter()
			_, _, vz := pl.GetVisualPosition()
			if relZ := vz - cz; relZ < minEyeRelZ {
				minEyeRelZ = relZ
			}
		}
	}
	if !landed {
		t.Fatalf("expected player to land on floor within 180 frames")
	}
	// Normal standing eye offset is (0.80 - 0.5)*depth = +18.0; landing spring must compress it downward.
	if minEyeRelZ >= 17.5 {
		t.Fatalf("expected landing bounce to dip eye offset below 17.5, got minEyeRelZ=%.4f", minEyeRelZ)
	}

	maxAbsBobX, maxBobY := 0.0, 0.0
	for i := 0; i < 120; i++ {
		bv.resetTestFrame()
		pl.Move(0.13, true, false, false, false)
		bv.pushQThings(th, vi.GetFrontFrustum(ff.Get()))
		en.Compute(pl, vi)
		bx, by, _ := pl.GetBob()
		if math.Abs(bx) > maxAbsBobX {
			maxAbsBobX = math.Abs(bx)
		}
		if by > maxBobY {
			maxBobY = by
		}
	}
	if maxAbsBobX < 0.5 || maxBobY < 0.5 {
		t.Fatalf("expected active walk bobbing, got maxAbsBobX=%.4f maxBobY=%.4f", maxAbsBobX, maxBobY)
	}
}

func TestJediBobbingWithinPlayerRadiusAndContinuousWalk(t *testing.T) {
	rigid_body.SetFps(120)
	defer rigid_body.SetFps(120)

	const jediDir = "../../../build/resources/jedi"
	const outlawsDir = "../../../build/resources/outlaws"
	cases := []struct {
		name string
		mode jedi.EngineMode
		dir  string
	}{
		{"jedi", jedi.EngineModeJedi, jediDir},
		{"outlaws", jedi.EngineModeOutlaws, outlawsDir},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := os.Stat(tc.dir); err != nil {
				t.Skipf("optional external resources not found at %s: %v", tc.dir, err)
			}
			res := common.NewResourcesOs()
			if err := res.Setup(); err != nil {
				t.Fatalf("res setup: %v", err)
			}
			root, err := jedi.NewBuilder().Build(res, tc.mode, tc.dir, 1)
			if err != nil {
				t.Fatalf("build %s failed: %v", tc.name, err)
			}
			if root.Player.Bobbing.MaxAmplitudeX >= root.Player.Radius {
				t.Fatalf("%s lateral bobbing %.2f must stay smaller than player radius %.2f",
					tc.name, root.Player.Bobbing.MaxAmplitudeX, root.Player.Radius)
			}

			en := engine.NewEngine()
			if err := en.Setup(root); err != nil {
				t.Fatalf("engine setup %s failed: %v", tc.name, err)
			}
			pl := en.GetPlayer()
			th := en.GetThings()
			vi := engine.NewViewMatrix()
			mCal := en.GetCalibration()
			ff := metrics.NewFrustum()
			ff.Setup(float32(mCal.AspectRatio), float32(mCal.ZFar), float32(mCal.ZNear), mCal.PrjScale, mCal.PrjScale)

			bv := newTestBuilderVolume(mCal, pl)
			bv.occlusion.GetState(pl.GetEntity().GetId()).IsVisible = false

			for i := 0; i < 300; i++ {
				bv.resetTestFrame()
				bv.pushQThings(th, vi.GetFrontFrustum(ff.Get()))
				en.Compute(pl, vi)
			}
			startX, startY, _ := pl.GetEntity().GetCenter()
			for i := 0; i < 120; i++ {
				bv.resetTestFrame()
				pl.Move(0.13, true, false, false, false)
				ex, ey, ez := vi.GetView()
				ff.Update(float32(ex), float32(ey), float32(ez), vi.GetAngle(), -vi.GetPitch(), vi.GetRoll())
				bv.pushQThings(th, vi.GetFrontFrustum(ff.Get()))
				en.Compute(pl, vi)
			}
			endX, endY, _ := pl.GetEntity().GetCenter()
			if dist := math.Hypot(endX-startX, endY-startY); dist < 20.0 {
				t.Fatalf("%s player stalled under pushQThings: moved=%.4f", tc.name, dist)
			}
		})
	}
}

func TestQuakeBootAndStepFirstFrames(t *testing.T) {
	rigid_body.SetFps(120)
	defer rigid_body.SetFps(120)

	cases := []struct {
		name string
		pak  string
	}{
		{"quake1", "../../../build/resources/quake/pak0.pak"},
		{"quake2", "../../../build/resources/quake2/pak0.pak"},
		{"quake3", "../../../build/resources/quake3arena/pak0.pk3"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := os.Stat(tc.pak); err != nil {
				t.Skipf("optional external resources not found at %s: %v", tc.pak, err)
			}
			res := common.NewResourcesOs()
			if err := res.Setup(); err != nil {
				t.Fatalf("res setup: %v", err)
			}
			root, err := quake.NewBuilder().Setup(res, tc.pak, 0)
			if err != nil {
				t.Fatalf("build %s failed: %v", tc.name, err)
			}
			en := engine.NewEngine()
			if err := en.Setup(root); err != nil {
				t.Fatalf("engine setup %s failed: %v", tc.name, err)
			}
			pl := en.GetPlayer()
			th := en.GetThings()
			vi := engine.NewViewMatrix()
			mCal := en.GetCalibration()
			ff := metrics.NewFrustum()
			ff.Setup(float32(mCal.AspectRatio), float32(mCal.ZFar), float32(mCal.ZNear), mCal.PrjScale, mCal.PrjScale)
			bv := newTestBuilderVolume(mCal, pl)

			for i := 0; i < 60; i++ {
				bv.resetTestFrame()
				ex, ey, ez := vi.GetView()
				ff.Update(float32(ex), float32(ey), float32(ez), vi.GetAngle(), -vi.GetPitch(), vi.GetRoll())
				bv.pushQThings(th, vi.GetFrontFrustum(ff.Get()))
				en.Compute(pl, vi)
			}
		})
	}
}
