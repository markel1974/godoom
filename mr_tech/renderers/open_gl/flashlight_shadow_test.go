package open_gl

import (
	"math"
	"os"
	"testing"
	"unsafe"

	"github.com/markel1974/godoom/mr_tech/engine"
	"github.com/markel1974/godoom/mr_tech/generators/common"
	"github.com/markel1974/godoom/mr_tech/generators/quake"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/metrics"
)

type stubSolidRaycaster struct {
	hitDist float64
	hit     bool
}

func (s stubSolidRaycaster) RaycastSolid(_, _, _, _, _, _, _ float64) (float64, float64, float64, float64, bool) {
	return s.hitDist, 0, 1, 0, s.hit
}

func TestResolveFlashOriginPreservesConfigAndAvoidsWalls(t *testing.T) {
	invView := [16]float32{
		1, 0, 0, 0,
		0, 1, 0, 0,
		0, 0, 1, 0,
		0, 0, 0, 1,
	}

	// Open room (no hit): preserves full configured offset (50, -0.9).
	fx, fy := resolveFlashOrigin(stubSolidRaycaster{hit: false}, &invView, 50.0, -0.9)
	if fx != 50.0 || fy != -0.9 {
		t.Fatalf("resolveFlashOrigin open room = (%v,%v), want (50,-0.9)", fx, fy)
	}

	// Wall 18 units to the right: pulls back smoothly by wallClearance (4.0) to 14.0 along the ray.
	wx, wy := resolveFlashOrigin(stubSolidRaycaster{hitDist: 18.0, hit: true}, &invView, 50.0, -0.9)
	offLen := math.Sqrt(50.0*50.0 + 0.9*0.9)
	wantScale := float32((18.0 - 4.0) / offLen)
	if math.Abs(float64(wx-50.0*wantScale)) > 1e-3 || math.Abs(float64(wy+0.9*wantScale)) > 1e-3 {
		t.Fatalf("resolveFlashOrigin near wall = (%v,%v), want (%v,%v)", wx, wy, 50.0*wantScale, -0.9*wantScale)
	}

	// Flush against wall (dist <= clearance): clamps safely to eye origin (0,0).
	cx, cy := resolveFlashOrigin(stubSolidRaycaster{hitDist: 2.0, hit: true}, &invView, 50.0, -0.9)
	if cx != 0 || cy != 0 {
		t.Fatalf("resolveFlashOrigin flush wall = (%v,%v), want (0,0)", cx, cy)
	}
}

func TestQuakeFlashlightSwayAndShadowAlignment(t *testing.T) {
	cases := []struct {
		name string
		pak  string
	}{
		{"quake1", "../../../build/resources/quake/pak0.pak"},
		{"quake2", "../../../build/resources/quake2/pak0.pak"},
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
			vi := engine.NewViewMatrix()
			en.Compute(pl, vi)

			swayX, swayY, swaySens := vi.GetSway()
			if swayX < 40.0 {
				t.Fatalf("%s idle swayX=%v unexpectedly clamped/shrunk (expected ~50 configured offset)", tc.name, swayX)
			}

			cal := en.GetCalibration()
			mapM := metrics.NewMap()
			mapM.SetOrthoSize(float32(cal.OrthoSize), float32(cal.ZNear), float32(cal.ZFar)+4.0)
			mapM.SetCenter(float32(cal.MapCenterX), float32(cal.MapCenterZ), float32(cal.LightCamY)+2.0)
			sx, sy := mapM.GetScale3d(1280, 960, float32(cal.AspectRatio), float32(cal.FovVerticalDegrees))
			mapM.Update(vi, sx, sy)

			view := mapM.GetView()
			invView := mapM.GetInvView()
			prod := metrics.MatrixMultiply4x4(view, invView)
			for i := 0; i < 4; i++ {
				if math.Abs(float64(prod[i*4+i]-1.0)) > 1e-3 {
					t.Fatalf("%s MatrixInverse4x4 invalid on view matrix: prod=%v", tc.name, prod)
				}
			}

			rawX, rawY, sens := float32(swayX), float32(swayY), float32(swaySens)
			flashX, flashY := resolveFlashOrigin(en.GetVolumes(), &invView, rawX, rawY)
			flM := metrics.NewFlashlight(pl.GetFlash())
			flM.UpdateWithAim(mapM.GetViewPtr(), flashX, flashY, rawX, rawY, sens)

			// Verify spotlight center direction uses configured sway offset and matches shadow camera axis.
			dirX, dirY, dirZ := flM.ResolveAim(flashX, flashY, sens)
			if dirX >= 0 {
				t.Fatalf("%s expected angled left beam dirX < 0 with SwayOffsetX=50, got dirX=%v", tc.name, dirX)
			}
			dx, dy, dz := dirX*512.0-flashX, dirY*512.0-flashY, dirZ*512.0
			dLen := float32(math.Sqrt(float64(dx*dx + dy*dy + dz*dz)))
			spotX, spotY, spotZ := dx/dLen, dy/dLen, dz/dLen
			vx, vy, vz := flashX+spotX*250.0, flashY+spotY*250.0, spotZ*250.0

			wx := invView[0]*vx + invView[4]*vy + invView[8]*vz + invView[12]
			wy := invView[1]*vx + invView[5]*vy + invView[9]*vz + invView[13]
			wz := invView[2]*vx + invView[6]*vy + invView[10]*vz + invView[14]

			var fSpace [16]float32
			copy(fSpace[:], (*[16]float32)(unsafe.Pointer(flM.GetShadowSpacePtr()))[:])
			cx := fSpace[0]*wx + fSpace[4]*wy + fSpace[8]*wz + fSpace[12]
			cy := fSpace[1]*wx + fSpace[5]*wy + fSpace[9]*wz + fSpace[13]
			cz := fSpace[2]*wx + fSpace[6]*wy + fSpace[10]*wz + fSpace[14]
			cw := fSpace[3]*wx + fSpace[7]*wy + fSpace[11]*wz + fSpace[15]
			px := (cx/cw)*0.5 + 0.5
			py := (cy/cw)*0.5 + 0.5
			pz := (cz/cw)*0.5 + 0.5
			if cw <= 0 || math.Abs(float64(px-0.5)) > 1e-3 || math.Abs(float64(py-0.5)) > 1e-3 {
				t.Fatalf("%s shadow projection off-center along spotlight ray: (%.4f,%.4f, w=%.2f)", tc.name, px, py, cw)
			}
			if pz >= 0.994 {
				t.Fatalf("%s flashlight depth precision degraded at d=250: pz=%.6f", tc.name, pz)
			}

			// Verify visible floor region at z=-60..-480 has positive w, valid depth (0,1), and full frustum coverage across the beam.
			ex, ey, ez := vi.GetView()
			downDist, _, _, _, downHit := en.GetVolumes().RaycastSolid(ex, ey, ez, 0, 0, -1, 200)
			if !downHit {
				t.Fatalf("%s expected floor hit beneath spawn", tc.name)
			}
			floorY := -float32(downDist)
			coneStart := float32(pl.GetFlash().GetConeStart())
			for _, fz := range []float32{-60, -120, -240, -480} {
				fwx := invView[4]*floorY + invView[8]*fz + invView[12]
				fwy := invView[5]*floorY + invView[9]*fz + invView[13]
				fwz := invView[6]*floorY + invView[10]*fz + invView[14]
				fcx := fSpace[0]*fwx + fSpace[4]*fwy + fSpace[8]*fwz + fSpace[12]
				fcy := fSpace[1]*fwx + fSpace[5]*fwy + fSpace[9]*fwz + fSpace[13]
				fcz := fSpace[2]*fwx + fSpace[6]*fwy + fSpace[10]*fwz + fSpace[14]
				fcw := fSpace[3]*fwx + fSpace[7]*fwy + fSpace[11]*fwz + fSpace[15]
				fpx := (fcx/fcw)*0.5 + 0.5
				fpy := (fcy/fcw)*0.5 + 0.5
				fpz := (fcz/fcw)*0.5 + 0.5
				if fcw <= 0 || fpz <= 0 || fpz >= 1.0 {
					t.Fatalf("%s floor z=%.0f invalid clip depth: uv=(%.3f,%.3f,%.5f) w=%.2f",
						tc.name, fz, fpx, fpy, fpz, fcw)
				}
				toPtX, toPtY, toPtZ := -flashX, floorY-flashY, fz
				ptLen := float32(math.Sqrt(float64(toPtX*toPtX + toPtY*toPtY + toPtZ*toPtZ)))
				dotCone := (toPtX/ptLen)*spotX + (toPtY/ptLen)*spotY + (toPtZ/ptLen)*spotZ
				if dotCone >= coneStart && (fpx < 0.03 || fpx > 0.97 || fpy < 0.03 || fpy > 0.97) {
					t.Fatalf("%s in-cone floor z=%.0f projected near/outside frustum edge: uv=(%.3f,%.3f)",
						tc.name, fz, fpx, fpy)
				}
			}
		})
	}
}
