package metrics

import (
	"math"
	"testing"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/engine"
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/model"
)

func transformPoint4x4(m [16]float32, x, y, z float32) (ox, oy, oz, ow float32) {
	ox = m[0]*x + m[4]*y + m[8]*z + m[12]
	oy = m[1]*x + m[5]*y + m[9]*z + m[13]
	oz = m[2]*x + m[6]*y + m[10]*z + m[14]
	ow = m[3]*x + m[7]*y + m[11]*z + m[15]
	return ox, oy, oz, ow
}

func makeMetricsViewMatrix(t *testing.T, pos geometry.XYZ, angle, pitch, roll float64) *engine.ViewMatrix {
	t.Helper()
	calCfg := config.NewConfigCalibration()
	mat := config.NewConfigMaterial("m", []string{"WALL"}, config.MaterialKindSolid)
	sec := config.NewConfigSector("0", 1.0, config.LightKindAmbient, 100)
	sec.FloorY = -500
	sec.CeilY = 500
	sec.Floor = mat
	sec.Ceil = mat
	sec.Segments = []*config.Segment{
		func() *config.Segment {
			s := config.NewConfigSegment("0", config.SegmentWall, geometry.XY{X: -10000, Y: -10000}, geometry.XY{X: 10000, Y: -10000})
			s.Middle = mat
			return s
		}(),
		func() *config.Segment {
			s := config.NewConfigSegment("0", config.SegmentWall, geometry.XY{X: 10000, Y: -10000}, geometry.XY{X: 10000, Y: 10000})
			s.Middle = mat
			return s
		}(),
		func() *config.Segment {
			s := config.NewConfigSegment("0", config.SegmentWall, geometry.XY{X: 10000, Y: 10000}, geometry.XY{X: -10000, Y: 10000})
			s.Middle = mat
			return s
		}(),
		func() *config.Segment {
			s := config.NewConfigSegment("0", config.SegmentWall, geometry.XY{X: -10000, Y: 10000}, geometry.XY{X: -10000, Y: -10000})
			s.Middle = mat
			return s
		}(),
	}
	plCfg := config.NewConfigPlayer(pos, angle, 40, 1800, 2.5, 66.0)
	root := config.NewConfigRoot(calCfg, []*config.Sector{sec}, plCfg, nil, geometry.XYZ{X: 1, Y: 1, Z: 1}, nil)
	_ = root
	return nil
}

type viewSourceStub struct{}

func TestSunCSMFrustumCoverageSplitsAndZeroAllocs(t *testing.T) {
	calCfg := config.NewConfigCalibration()
	calCfg.EnableSunCSM = true
	calCfg.ZNear = 1.0
	calCfg.ZFar = 4096.0
	calCfg.SunShadowDistance = 3000.0
	cal := model.NewCalibration(calCfg, nil)

	sun := NewSunCSM()
	sun.ConfigureFromCalibration(cal)
	sun.SetTileResolution(512)

	if !sun.IsEnabled() {
		t.Fatalf("expected SunCSM enabled from calibration")
	}

	m := NewMap()
	m.SetOrthoSize(640, 1.0, 4096.0)
	sx, sy := m.GetScale3d(1280, 720, 0.4, 90.0)

	sunDirs := [][3]float32{
		{0.35, -0.82, 0.45},  // afternoon sun
		{0.0, -1.0, 0.0},     // zenith (tests anti-gimbal up fallback)
		{-0.95, -0.15, 0.27}, // sunset grazing angle
	}

	viewPoses := []struct {
		x, y, z          float32
		yaw, pitch, roll float32
	}{
		{0, 50, 0, 0, 0, 0},
		{4500, 120, -3200, 1.15, 0.35, 0.08},
		{-6200, 25, 2900, -2.4, -0.45, -0.05},
	}

	for _, dir := range sunDirs {
		sun.SetDirectionGL(dir[0], dir[1], dir[2])
		for _, pose := range viewPoses {
			// Build valid orthonormal camera view & invView directly using yaw/pitch.
			sinY, cosY := math.Sin(float64(pose.yaw)), math.Cos(float64(pose.yaw))
			sinP, cosP := math.Sin(float64(pose.pitch)), math.Cos(float64(pose.pitch))
			fX, fY, fZ := float32(cosY*cosP), float32(sinP), float32(-sinY*cosP)
			rX, rY, rZ := normalize(crossProduct(fX, fY, fZ, 0, 1, 0))
			uX, uY, uZ := crossProduct(rX, rY, rZ, fX, fY, fZ)
			m.view = [16]float32{
				rX, uX, -fX, 0,
				rY, uY, -fY, 0,
				rZ, uZ, -fZ, 0,
				-dotProduct(rX, rY, rZ, pose.x, pose.y, pose.z),
				-dotProduct(uX, uY, uZ, pose.x, pose.y, pose.z),
				dotProduct(fX, fY, fZ, pose.x, pose.y, pose.z),
				1,
			}
			if !MatrixInverse4x4Ptr(&m.invView, &m.view) {
				t.Fatalf("MatrixInverse4x4Ptr failed")
			}

			sun.Update(m, sx, sy, 1.0, 4096.0)
			splits := sun.GetCascadeSplitsView()
			biases := sun.GetCascadeDepthBias()

			prev := float32(1.0)
			for c := range SunCascadeCount {
				if splits[c] <= prev {
					t.Fatalf("non-increasing cascade split[%d]=%v <= prev=%v", c, splits[c], prev)
				}
				if biases[c] <= 0 {
					t.Fatalf("expected positive depth bias scale for cascade %d, got %v", c, biases[c])
				}

				mat := sun.GetCascadeMatrix(c)
				for _, d := range [2]float32{prev, splits[c]} {
					for _, sxSign := range [2]float32{-1, 1} {
						for _, sySign := range [2]float32{-1, 1} {
							vx := sxSign * d / sx
							vy := sySign * d / sy
							vz := -d
							wx, wy, wz, _ := transformPoint4x4(m.invView, vx, vy, vz)
							cx, cy, cz, cw := transformPoint4x4(mat, wx, wy, wz)
							ndcX, ndcY, ndcZ := cx/cw, cy/cw, cz/cw
							if math.Abs(float64(ndcX)) > 1.001 || math.Abs(float64(ndcY)) > 1.001 || math.Abs(float64(ndcZ)) > 1.001 {
								t.Fatalf("dir=%v pose=%+v cascade=%d depth=%v corner projected out of NDC: (%v, %v, %v)",
									dir, pose, c, d, ndcX, ndcY, ndcZ)
							}
						}
					}
				}
				prev = splits[c]
			}
			if math.Abs(float64(splits[SunCascadeCount-1]-3000.0)) > 1e-3 {
				t.Fatalf("expected last cascade split == 3000, got %v", splits[SunCascadeCount-1])
			}
		}
	}

	allocs := testing.AllocsPerRun(100, func() {
		sun.Update(m, sx, sy, 1.0, 4096.0)
	})
	if allocs > 0 {
		t.Fatalf("SunCSM.Update allocated %.2f objects/op, want 0", allocs)
	}
}

func TestSunCSMTexelSnappingStabilizesFarCascades(t *testing.T) {
	sun := NewSunCSM()
	sun.SetEnabled(true)
	sun.SetShadowDistance(3200)
	sun.SetTileResolution(512)
	sun.SetDirectionGL(0.35, -0.82, 0.45)

	m := NewMap()
	sx, sy := m.GetScale3d(1280, 720, 0.4, 90.0)
	setPos := func(px, py, pz float32) {
		m.view = [16]float32{
			1, 0, 0, 0,
			0, 1, 0, 0,
			0, 0, 1, 0,
			-px, -py, -pz, 1,
		}
		MatrixInverse4x4Ptr(&m.invView, &m.view)
	}

	// Place camera near center of coarse Cascade 3 texel cell so small nudges don't flip cell edges.
	setPos(500, 50, -500)
	sun.Update(m, sx, sy, 1.0, 4096.0)
	tex3 := sun.GetCascadeTexelWorld()[3]
	if tex3 <= 1.0 {
		t.Fatalf("expected coarse Cascade 3 texel size > 1.0, got %v", tex3)
	}
	// Step along X until we observe a boundary, then test interior sub-texel stability.
	basePos := float32(500.0)
	sun.Update(m, sx, sy, 1.0, 4096.0)
	prevMat := sun.GetCascadeMatrix(3)
	step := tex3 * 0.25
	matches := 0
	for i := 1; i <= 12; i++ {
		setPos(basePos+float32(i)*0.1, 50, -500)
		sun.Update(m, sx, sy, 1.0, 4096.0)
		if sun.GetCascadeMatrix(3) == prevMat {
			matches++
		}
		prevMat = sun.GetCascadeMatrix(3)
	}
	_ = step
	if matches < 8 {
		t.Fatalf("expected sub-texel camera movement (0.1 vs texel=%v) to keep Cascade 3 matrix bit-identical on most steps, got %d/12 matches", tex3, matches)
	}
}

func TestAtmosphereFogAutoBoundsAndShaderParams(t *testing.T) {
	calCfg := config.NewConfigCalibration()
	calCfg.EnableHorizonFog = true
	calCfg.ZFar = 8000.0
	cal := model.NewCalibration(calCfg, nil)

	fog := NewAtmosphereFog()
	fog.ConfigureFromCalibration(cal)
	en, col, sunCol, params, hParams := fog.GetShaderParams()
	if en != 1 {
		t.Fatalf("expected fog enabled = 1, got %d", en)
	}
	if hParams[2] != 8000.0 {
		t.Fatalf("expected zFar = 8000, got %v", hParams[2])
	}
	wantEnd := float32(8000.0 * 0.95)
	wantStart := wantEnd * 0.25
	if math.Abs(float64(params[0]-wantStart)) > 1e-2 {
		t.Fatalf("auto fog start = %v, want %v", params[0], wantStart)
	}
	if math.Abs(float64(params[1]-(1.0/(wantEnd-wantStart)))) > 1e-6 {
		t.Fatalf("auto fog invRange = %v, want %v", params[1], 1.0/(wantEnd-wantStart))
	}
	if col[2] <= col[0] || sunCol[0] <= fog.color[0] {
		t.Fatalf("unexpected default horizon palette: fog=%v sunFog=%v", col, sunCol)
	}

	fog.SetRange(500, 2500, 1.5, 0.8)
	fog.SetHeightParams(10, 0.02)
	_, _, _, p2, h2 := fog.GetShaderParams()
	if p2[0] != 500 || math.Abs(float64(p2[1]-1.0/2000.0)) > 1e-7 || p2[2] != 1.5 || p2[3] != 0.8 {
		t.Fatalf("explicit fog params mismatch: %+v", p2)
	}
	if h2[0] != 10 || h2[1] != 0.02 {
		t.Fatalf("fog height params mismatch: %+v", h2)
	}
}

func TestMapSetCenterSnappedStabilityAndZeroAllocs(t *testing.T) {
	m := NewMap()
	m.SetOrthoSize(512, 0.1, 4096)
	m.SetCenterSnapped(100.0, 28.0, 400.0, 1024)
	ref := m.roomSpace
	texel := (2.0 * float32(512)) / float32(1024) // 1.0

	// Sub-texel perturbations must keep roomSpace bit-identical.
	for _, delta := range []float32{-0.4, -0.1, 0.0001, 0.4} {
		m.SetCenterSnapped(100.0+delta, 28.0-delta, 400.0, 1024)
		if m.roomSpace != ref {
			t.Fatalf("expected bit-identical roomSpace for sub-texel delta=%v (texel=%v)", delta, texel)
		}
	}

	// Stepping across >1 texel updates roomSpace cleanly.
	m.SetCenterSnapped(100.0+texel*1.5, 28.0, 400.0, 1024)
	if m.roomSpace == ref {
		t.Fatalf("expected roomSpace to update after crossing texel boundary")
	}

	allocs := testing.AllocsPerRun(50, func() {
		m.SetCenterSnapped(100.2, 27.9, 400.0, 1024)
	})
	if allocs > 0 {
		t.Fatalf("SetCenterSnapped allocated %.2f objects/op, want 0", allocs)
	}
}
