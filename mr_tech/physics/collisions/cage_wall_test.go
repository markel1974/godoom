package collisions

import (
	"math"
	"testing"

	"github.com/markel1974/godoom/mr_tech/physics/aabb"
	"github.com/markel1974/godoom/mr_tech/physics/rigid_body"
)

type testObject struct {
	solver *Solver
	ent    *rigid_body.Entity
	box    *aabb.AABB
}

func newTestUprightObject(cx, cy, cz, w, h, d float64) *testObject {
	solver := NewSolver(true, 80.0, 0.0, 0.2, int(rigid_body.CombineAverage), int(rigid_body.CombineAverage), 0.0)
	ent := solver.GetEntity()
	ent.Rebuild(cx-w*0.5, cy-h*0.5, cz-d*0.5, w, h, d)
	ent.SetKeepUpright(true)
	ent.LockUpright()
	obj := &testObject{
		solver: solver,
		ent:    ent,
		box:    aabb.NewAABB(),
	}
	_ = solver.Setup(obj)
	return obj
}

func (o *testObject) GetAABB() *aabb.AABB {
	cx, cy, cz := o.ent.GetCenter()
	w, h, d := o.ent.GetSize()
	hx, hy, hz := w*0.5, h*0.5, d*0.5
	o.box.Rebuild(cx-hx, cy-hy, cz-hz, cx+hx, cy+hy, cz+hz)
	return o.box
}

func (o *testObject) GetEntity() *rigid_body.Entity {
	return o.ent
}

type testFace struct {
	v0, v1, v2  [3]float64
	nx, ny, nz  float64
	box         *aabb.AABB
	singleSided bool
}

func newTestFace(v0, v1, v2 [3]float64, nx, ny, nz float64) *testFace {
	minX := math.Min(v0[0], math.Min(v1[0], v2[0]))
	minY := math.Min(v0[1], math.Min(v1[1], v2[1]))
	minZ := math.Min(v0[2], math.Min(v1[2], v2[2]))
	maxX := math.Max(v0[0], math.Max(v1[0], v2[0]))
	maxY := math.Max(v0[1], math.Max(v1[1], v2[1]))
	maxZ := math.Max(v0[2], math.Max(v1[2], v2[2]))
	box := aabb.NewAABB()
	box.Rebuild(minX, minY, minZ, maxX, maxY, maxZ)
	return &testFace{
		v0:  v0,
		v1:  v1,
		v2:  v2,
		nx:  nx,
		ny:  ny,
		nz:  nz,
		box: box,
	}
}

func (f *testFace) GetAABB() *aabb.AABB { return f.box }
func (f *testFace) GetNormal() (float64, float64, float64) {
	return f.nx, f.ny, f.nz
}
func (f *testFace) GetNormalAbs() (float64, float64, float64) {
	return math.Abs(f.nx), math.Abs(f.ny), math.Abs(f.nz)
}
func (f *testFace) Translate(int, float64, float64, float64) (float64, float64, float64) {
	return 0, 0, 0
}
func (f *testFace) GetParent() *Solver                { return nil }
func (f *testFace) Get0() (float64, float64, float64) { return f.v0[0], f.v0[1], f.v0[2] }
func (f *testFace) Get1() (float64, float64, float64) { return f.v1[0], f.v1[1], f.v1[2] }
func (f *testFace) Get2() (float64, float64, float64) { return f.v2[0], f.v2[1], f.v2[2] }
func (f *testFace) IsSolid() bool                     { return true }
func (f *testFace) IsSingleSided() bool               { return f.singleSided }

func newTestCage(_ uint64, obj *testObject) *Cage {
	return obj.solver.GetCage()
}

func TestKeepUprightPillarCornerNoPhantomBounce(t *testing.T) {
	for _, ny := range []float64{1.0, -1.0} {
		obj := newTestUprightObject(80.5, 202.0, 30.0, 40.0, 40.0, 60.0)
		cage := newTestCage(1, obj)
		cage.Rebuild(18.0, 1)

		t1 := newTestFace([3]float64{100, 200, 0}, [3]float64{120, 200, 0}, [3]float64{120, 200, 100}, 0, ny, 0)
		t2 := newTestFace([3]float64{100, 200, 0}, [3]float64{120, 200, 100}, [3]float64{100, 200, 100}, 0, ny, 0)
		cage.AddFace(t1, nil)
		cage.AddFace(t2, nil)
		cage.Commit(nil)

		if cage.GetSlotsLen() == 0 {
			t.Fatalf("ny=%v: expected slight corner contact at (80.5, 202), got 0 slots", ny)
		}
		for i := 0; i < cage.GetSlotsLen(); i++ {
			slot := cage.GetSlot(i)
			pen := slot.GetPenetration()
			cnx, _, _ := slot.GetNormal()
			if pen > 1.0 {
				t.Fatalf("ny=%v: phantom deep bounce on pillar corner: pen=%v", ny, pen)
			}
			if cnx >= -0.9 {
				t.Fatalf("ny=%v: expected smooth -X pushout on pillar X-corner, got normal x=%v", ny, cnx)
			}
		}
	}
}

func TestKeepUprightBevelCornerNoPhantomKick(t *testing.T) {
	invSqrt2 := 1.0 / math.Sqrt2
	t1 := newTestFace([3]float64{100, 200, 0}, [3]float64{120, 180, 0}, [3]float64{120, 180, 100}, -invSqrt2, -invSqrt2, 0)
	t2 := newTestFace([3]float64{100, 200, 0}, [3]float64{120, 180, 100}, [3]float64{100, 200, 100}, -invSqrt2, -invSqrt2, 0)

	objMiss := newTestUprightObject(80.1, 205.0, 30.0, 40.0, 40.0, 60.0)
	cageMiss := newTestCage(2, objMiss)
	cageMiss.Rebuild(18.0, 1)
	cageMiss.AddFace(t1, nil)
	cageMiss.AddFace(t2, nil)
	cageMiss.Commit(nil)
	if n := cageMiss.GetSlotsLen(); n != 0 {
		t.Fatalf("expected phantom bevel miss at (80.1, 205) to produce 0 contacts, got %d (pen=%v)", n, cageMiss.GetSlot(0).GetPenetration())
	}

	objHit := newTestUprightObject(80.5, 202.0, 30.0, 40.0, 40.0, 60.0)
	cageHit := newTestCage(3, objHit)
	cageHit.Rebuild(18.0, 2)
	cageHit.AddFace(t1, nil)
	cageHit.AddFace(t2, nil)
	cageHit.Commit(nil)
	if n := cageHit.GetSlotsLen(); n == 0 {
		t.Fatalf("expected grazing bevel hit at (80.5, 202) to register contact")
	}
	for i := 0; i < cageHit.GetSlotsLen(); i++ {
		if pen := cageHit.GetSlot(i).GetPenetration(); pen > 0.6 {
			t.Fatalf("expected small smooth corner pen <= 0.6, got %v", pen)
		}
	}
}

func TestKeepUprightFlatWallDiagonalSeamNoSnag(t *testing.T) {
	// Flat wall quad y=200, x=[0..200], z=[0..100] split diagonally from (0,200,0) to (200,200,100).
	// With player at (115, 185, 90) (Z slab [60..120]), t1 covers [120..200] (phantom seam corner at x=120)
	// while t2 covers [0..200] across the diagonal and contains x=115 squarely.
	t1 := newTestFace([3]float64{0, 200, 0}, [3]float64{200, 200, 0}, [3]float64{200, 200, 100}, 0, -1, 0)
	t2 := newTestFace([3]float64{0, 200, 0}, [3]float64{200, 200, 100}, [3]float64{0, 200, 100}, 0, -1, 0)

	for _, cornerFirst := range []bool{true, false} {
		obj := newTestUprightObject(115.0, 185.0, 90.0, 40.0, 40.0, 60.0)
		cage := newTestCage(10, obj)
		cage.Rebuild(18.0, 1)
		if cornerFirst {
			cage.AddFace(t1, nil)
			cage.AddFace(t2, nil)
		} else {
			cage.AddFace(t2, nil)
			cage.AddFace(t1, nil)
		}
		cage.Commit(nil)

		if n := cage.GetSlotsLen(); n != 1 {
			t.Fatalf("cornerFirst=%v: expected 1 clean flat contact, got %d", cornerFirst, n)
		}
		nx, ny, _ := cage.GetSlot(0).GetNormal()
		if math.Abs(nx) > 1e-6 || ny > -0.999 {
			t.Fatalf("cornerFirst=%v: internal seam leaked lateral push (%v, %v)", cornerFirst, nx, ny)
		}
	}

	// Verify static clamp ordering with t1 before t2 also produces zero lateral X push.
	objClamp := newTestUprightObject(115.0, 185.0, 90.0, 40.0, 40.0, 60.0)
	objClamp.ent.SetV(100.0, 0.0, 0.0)
	cageClamp := newTestCage(11, objClamp)
	cageClamp.Rebuild(18.0, 1)
	origCX, origCY, origCZ, origDX, origDY, origDZ, origTX, origTY, origTZ, lAABB := cageClamp.BeginStaticClamp()
	cageClamp.ClampStaticFace(lAABB, t1)
	cageClamp.ClampStaticFace(lAABB, t2)
	cageClamp.FlushStaticClamp(lAABB)
	cageClamp.EndStaticClamp(origCX, origCY, origCZ, origDX, origDY, origDZ, origTX, origTY, origTZ)
	cx, cy, _ := objClamp.ent.GetCenter()
	if math.Abs(cx-115.0) > 1e-9 || math.Abs(cy-180.0) > 0.01 {
		t.Fatalf("expected static clamp to push purely along -Y near (115, 180), got (%v, %v)", cx, cy)
	}
	if vx := objClamp.ent.GetVx(); math.Abs(vx-100.0) > 1e-9 {
		t.Fatalf("expected tangential slide velocity vx=100 preserved across internal seam, got %v", vx)
	}
}

func TestKeepUprightFrontWallOvershootNotFlipped(t *testing.T) {
	// Player starts strictly on the front side of wall y=200 (n=(0,-1,0)) at c=(110, 195, 30)
	// and steps +6 along Y to t=(110, 201, 30), crossing 1 unit past y=200.
	obj := newTestUprightObject(110.0, 195.0, 30.0, 40.0, 40.0, 60.0)
	obj.ent.SetV(0.0, 6.0*120.0, 0.0)
	cage := newTestCage(12, obj)
	cage.Rebuild(18.0, 1)

	t1 := newTestFace([3]float64{100, 200, 0}, [3]float64{120, 200, 0}, [3]float64{120, 200, 100}, 0, -1, 0)
	t2 := newTestFace([3]float64{100, 200, 0}, [3]float64{120, 200, 100}, [3]float64{100, 200, 100}, 0, -1, 0)
	cage.AddFace(t1, nil)
	cage.AddFace(t2, nil)
	cage.Commit(nil)

	if cage.GetSlotsLen() == 0 {
		t.Fatalf("expected contact against front wall")
	}
	slot := cage.GetSlot(0)
	_, ny, _ := slot.GetNormal()
	if ny > -0.99 {
		t.Fatalf("wall normal inverted backward to %v instead of pushing back to -Y", ny)
	}
	if pen := slot.GetPenetration(); math.Abs(pen-21.0) > 0.05 {
		t.Fatalf("expected penetration ~21.0, got %v", pen)
	}
}

func TestKeepUprightOutOfZSlabNotResurrectedByTunnel(t *testing.T) {
	// Overhead lintel at z=[65..120], y=200; player drops from cZ=50 to tZ=30 (tZ+eRadZ=60 < 65)
	// while sweeping across y=200 (distStart >= rayEff, distTarget < 0 => tunneled=true).
	obj := newTestUprightObject(110.0, 179.0, 50.0, 40.0, 40.0, 60.0)
	obj.ent.SetV(0.0, 22.0*120.0, -20.0*120.0)
	cage := newTestCage(13, obj)
	cage.Rebuild(18.0, 1)

	t1 := newTestFace([3]float64{100, 200, 65}, [3]float64{120, 200, 65}, [3]float64{120, 200, 120}, 0, -1, 0)
	t2 := newTestFace([3]float64{100, 200, 65}, [3]float64{120, 200, 120}, [3]float64{100, 200, 120}, 0, -1, 0)
	cage.AddFace(t1, nil)
	cage.AddFace(t2, nil)
	cage.Commit(nil)

	if n := cage.GetSlotsLen(); n != 0 {
		t.Fatalf("expected overhead lintel outside target Z slab [0..60] to produce 0 contacts, got %d", n)
	}
}

func TestSingleSidedBackfacingWallIgnoredAtConcaveCorner(t *testing.T) {
	// Two-sided wall (e.g. Quake/mid-portal) approached from -normal side must still block.
	objTwo := newTestUprightObject(122.0, 100.0, 30.0, 40.0, 40.0, 60.0)
	objTwo.ent.SetV(-4.0*120.0, 0.0, 0.0)
	cageTwo := newTestCage(3, objTwo)
	cageTwo.Rebuild(18.0, 1)
	w1 := newTestFace([3]float64{100, 0, 0}, [3]float64{100, 200, 0}, [3]float64{100, 200, 100}, -1, 0, 0)
	w2 := newTestFace([3]float64{100, 0, 0}, [3]float64{100, 200, 100}, [3]float64{100, 0, 100}, -1, 0, 0)
	cageTwo.AddFace(w1, nil)
	cageTwo.AddFace(w2, nil)
	cageTwo.Commit(nil)
	if cageTwo.GetSlotsLen() == 0 {
		t.Fatalf("expected two-sided wall approach from -normal side to block")
	}
	if nx, _, _ := cageTwo.GetSlot(0).GetNormal(); nx <= 0.9 {
		t.Fatalf("expected flipped normal (+1,0,0) pushing back to +X, got nx=%v", nx)
	}

	// Single-sided sector wall starting behind the backface (-rayEff or non-corner backside -2.499)
	// must be culled rather than flipped into a reverse vacuum that pulls through adjacent walls.
	objDeep := newTestUprightObject(122.0, 100.0, 30.0, 40.0, 40.0, 60.0)
	objDeep.ent.SetV(-4.0*120.0, 0.0, 0.0)
	cageDeep := newTestCage(4, objDeep)
	cageDeep.Rebuild(18.0, 1)

	t1 := newTestFace([3]float64{100, 0, 0}, [3]float64{100, 200, 0}, [3]float64{100, 200, 100}, -1, 0, 0)
	t2 := newTestFace([3]float64{100, 0, 0}, [3]float64{100, 200, 100}, [3]float64{100, 0, 100}, -1, 0, 0)
	t1.singleSided = true
	t2.singleSided = true
	cageDeep.AddFace(t1, nil)
	cageDeep.AddFace(t2, nil)
	cageDeep.Commit(nil)
	if n := cageDeep.GetSlotsLen(); n != 0 {
		t.Fatalf("expected deep backfacing single-sided wall (distStart <= -rayEff) to produce 0 contacts, got %d", n)
	}

	// Curved sector wall seam (-2.499 behind face[3] while moving out to open space +46.25) must not flip -n.
	objSeam := newTestUprightObject(3159.966, 3171.839, 54.999, 5.0, 5.0, 66.0)
	cageSeam := newTestCage(44, objSeam)
	cageSeam.Rebuild(18.0, 1)
	cageSeam.CommitStaticClamp()
	seam1 := newTestFace([3]float64{3161.3, 3178.3, -22}, [3]float64{3142.3, 3149.7, -22}, [3]float64{3142.3, 3149.7, 154}, -0.833, 0.554, 0)
	seam2 := newTestFace([3]float64{3161.3, 3178.3, -22}, [3]float64{3142.3, 3149.7, 154}, [3]float64{3161.3, 3178.3, 154}, -0.833, 0.554, 0)
	seam1.singleSided = true
	seam2.singleSided = true
	objSeam.ent.AddTo(3154.435-3159.966, 3251.452-3171.839, 0.0)
	oCX, oCY, oCZ, oDX, oDY, oDZ, oTX, oTY, oTZ, box := cageSeam.BeginStaticClamp()
	if cageSeam.ClampStaticFace(box, seam1) || cageSeam.ClampStaticFace(box, seam2) || cageSeam.FlushStaticClamp(box) {
		px, py, _ := objSeam.ent.GetCenter()
		t.Fatalf("expected exiting backfacing concave facet not to pull inward, got (%.3f,%.3f)", px, py)
	}
	cageSeam.EndStaticClamp(oCX, oCY, oCZ, oDX, oDY, oDZ, oTX, oTY, oTZ)
}

func TestEvaluateStaticFaceAllocs(t *testing.T) {
	obj := newTestUprightObject(80.5, 202.0, 30.0, 40.0, 40.0, 60.0)
	cage := newTestCage(5, obj)
	t1 := newTestFace([3]float64{100, 200, 0}, [3]float64{120, 200, 0}, [3]float64{120, 200, 100}, 0, 1, 0)
	t2 := newTestFace([3]float64{100, 200, 0}, [3]float64{120, 200, 100}, [3]float64{100, 200, 100}, 0, 1, 0)

	cage.Rebuild(18.0, 1)
	cage.AddFace(t1, nil)
	cage.AddFace(t2, nil)
	cage.Commit(nil)

	var session uint64 = 2
	allocs := testing.AllocsPerRun(100, func() {
		cage.Rebuild(18.0, session)
		cage.AddFace(t1, nil)
		cage.AddFace(t2, nil)
		cage.Commit(nil)
		session++
	})
	if allocs != 0 {
		t.Fatalf("expected 0 heap allocs per Rebuild+AddFace+Commit frame, got %v", allocs)
	}
}

func TestKeepUprightDiagonalYawNoBloat(t *testing.T) {
	obj := newTestUprightObject(100.0, 100.0, 30.0, 40.0, 40.0, 60.0)
	obj.ent.SetRotationAngleZ(math.Pi / 4.0)
	cage := newTestCage(20, obj)
	cage.Rebuild(18.0, 1)

	box := cage.GetAABB()
	hx := 0.5 * (box.GetMaxX() - box.GetMinX())
	hy := 0.5 * (box.GetMaxY() - box.GetMinY())
	hz := 0.5 * (box.GetMaxZ() - box.GetMinZ())
	if math.Abs(hx-20.0) > 1e-6 || math.Abs(hy-20.0) > 1e-6 || math.Abs(hz-30.0) > 1e-6 {
		t.Fatalf("expected unbloated upright AABB half-extents (20,20,30) under 45deg yaw, got (%.3f,%.3f,%.3f)", hx, hy, hz)
	}
}

func TestKeepUprightSlopedBevelTrimNoPhantomKick(t *testing.T) {
	obj := newTestUprightObject(0.0, 0.0, 30.0, 40.0, 40.0, 60.0)
	cage := newTestCage(21, obj)
	cage.Rebuild(18.0, 1)

	// Sloped floor/ceiling bevel at AABB corner strictly outside the 3D ellipsoid:
	// (16/20)^2 + (16/20)^2 + ((20-30)/30)^2 = 1.391 > 1.0
	lenN := math.Sqrt(32.0*32.0 + 32.0*32.0 + 64.0*64.0)
	bevel := newTestFace(
		[3]float64{16.0, 16.0, 20.0},
		[3]float64{24.0, 16.0, 16.0},
		[3]float64{16.0, 24.0, 16.0},
		32.0/lenN, 32.0/lenN, 64.0/lenN,
	)
	cage.AddFace(bevel, nil)
	cage.Commit(nil)

	if n := cage.GetSlotsLen(); n != 0 {
		t.Fatalf("expected corner bevel outside 3D ellipsoid to produce 0 contacts, got %d (pen=%v)", n, cage.GetSlot(0).GetPenetration())
	}
}

func TestKeepUprightDomeCapAndTiltedWallExact3D(t *testing.T) {
	// Subcase A: Upper dome-cap lip (rMinZ > tZ=30) must push by 3D cap depth (< 3.5), not full cylinder (~13).
	objCap := newTestUprightObject(0.0, 0.0, 30.0, 40.0, 40.0, 60.0)
	cageCap := newTestCage(22, objCap)
	cageCap.Rebuild(18.0, 1)
	invSqrt2 := 1.0 / math.Sqrt2
	capLip := newTestFace(
		[3]float64{5.0, 5.0, 56.0},
		[3]float64{5.0, 5.0, 62.0},
		[3]float64{12.0, -2.0, 58.0},
		-invSqrt2, -invSqrt2, 0.0,
	)
	cageCap.AddFace(capLip, nil)
	cageCap.Commit(nil)
	if cageCap.GetSlotsLen() == 0 {
		t.Fatalf("expected dome-cap graze to register 3D contact")
	}
	if pen := cageCap.GetSlot(0).GetPenetration(); pen > 3.5 {
		t.Fatalf("expected 3D dome-cap penetration <= 3.5 (not Z-slab cylinder ~13), got %v", pen)
	}

	// Subcase B: Tilted bevel wall spanning tZ=222 (exact Doom 3 repro) must not Z-slab flatten to ~11.4.
	objTilt := newTestUprightObject(-1075.34, 632.11, 222.0, 40.0, 40.0, 60.0)
	cageTilt := newTestCage(23, objTilt)
	cageTilt.Rebuild(18.0, 1)
	fnLen := math.Hypot(0.894427, 0.447214)
	tilted := newTestFace(
		[3]float64{-1072.0, 640.0, 192.0},
		[3]float64{-1056.0, 640.0, 192.0},
		[3]float64{-1072.0, 656.0, 224.0},
		0.0, -0.894427/fnLen, 0.447214/fnLen,
	)
	cageTilt.AddFace(tilted, nil)
	cageTilt.Commit(nil)
	if cageTilt.GetSlotsLen() == 0 {
		t.Fatalf("expected tilted wall graze to register 3D contact")
	}
	if pen := cageTilt.GetSlot(0).GetPenetration(); pen > 2.5 {
		t.Fatalf("expected tilted wall 3D penetration <= 2.5 (not Z-slab ~11.4), got %v", pen)
	}
}

func TestKeepUprightCoplanarStepSuppressedOnFlatWall(t *testing.T) {
	lower := newTestFace([3]float64{-100, 0, 0}, [3]float64{100, 0, 0}, [3]float64{-10, 0, 55}, 0, -1, 0)
	upper := newTestFace([3]float64{100, 0, 0}, [3]float64{100, 0, 120}, [3]float64{-100, 0, 120}, 0, -1, 0)

	for _, lowerFirst := range []bool{true, false} {
		obj := newTestUprightObject(-4.0, -18.5, 60.0, 40.0, 40.0, 60.0)
		cage := newTestCage(24, obj)
		cage.Rebuild(30.0, 1)
		if lowerFirst {
			cage.AddFace(lower, nil)
			cage.AddFace(upper, nil)
		} else {
			cage.AddFace(upper, nil)
			cage.AddFace(lower, nil)
		}
		cage.Commit(nil)

		if cage.GetSlotsLen() == 0 {
			t.Fatalf("lowerFirst=%v: expected inelastic wall contact", lowerFirst)
		}
		for i := 0; i < cage.GetSlotsLen(); i++ {
			if mode := cage.GetSlot(i).GetImpactMode(); mode == ImpactStep {
				t.Fatalf("lowerFirst=%v: lower coplanar triangle leaked ImpactStep on sheer flat wall", lowerFirst)
			}
		}
	}
}

func TestKeepUprightHighSpeedSweptWallsAndZeroAllocs(t *testing.T) {
	// Lintel z=[55..200] at x=100, large sweep dX=300 (Jedi/normal and extreme speeds).
	lintel1 := newTestFace([3]float64{100, -200, 55}, [3]float64{100, 200, 55}, [3]float64{100, 200, 200}, -1, 0, 0)
	lintel2 := newTestFace([3]float64{100, -200, 55}, [3]float64{100, 200, 200}, [3]float64{100, -200, 200}, -1, 0, 0)
	objLintel := newTestUprightObject(50.0, 0.0, 50.0, 40.0, 40.0, 100.0)
	objLintel.ent.SetV(300.0*120.0, 0.0, 0.0)
	cageLintel := newTestCage(30, objLintel)
	cageLintel.Rebuild(18.0, 1)
	cageLintel.AddFace(lintel1, nil)
	cageLintel.AddFace(lintel2, nil)
	cageLintel.Commit(nil)
	if cageLintel.GetSlotsLen() == 0 {
		t.Fatalf("expected swept lintel contact when crossing x=100 with dX=300")
	}

	// Jedi small-radius player (r=2.5) moving dX=30 past lintel x=100.
	objJedi := newTestUprightObject(90.0, 0.0, 50.0, 5.0, 5.0, 100.0)
	objJedi.ent.SetV(30.0*120.0, 0.0, 0.0)
	cageJedi := newTestCage(31, objJedi)
	cageJedi.Rebuild(18.0, 1)
	cageJedi.AddFace(lintel1, nil)
	cageJedi.AddFace(lintel2, nil)
	cageJedi.Commit(nil)
	if cageJedi.GetSlotsLen() == 0 {
		t.Fatalf("expected swept lintel contact for small-radius body (r=2.5, dX=30)")
	}

	// Slightly tilted wall crossed at dX=300 must register contact at plane crossing.
	tLen := math.Sqrt(1.0 + 0.05*0.05)
	tilted1 := newTestFace([3]float64{105, -200, 0}, [3]float64{105, 200, 0}, [3]float64{95, 200, 200}, -1.0/tLen, 0, -0.05/tLen)
	tilted2 := newTestFace([3]float64{105, -200, 0}, [3]float64{95, 200, 200}, [3]float64{95, -200, 200}, -1.0/tLen, 0, -0.05/tLen)
	objTilt := newTestUprightObject(50.0, 0.0, 50.0, 40.0, 40.0, 100.0)
	objTilt.ent.SetV(300.0*120.0, 0.0, 0.0)
	cageTilt := newTestCage(32, objTilt)
	cageTilt.Rebuild(18.0, 1)
	cageTilt.AddFace(tilted1, nil)
	cageTilt.AddFace(tilted2, nil)
	cageTilt.Commit(nil)
	if cageTilt.GetSlotsLen() == 0 {
		t.Fatalf("expected swept contact against slightly tilted wall with dX=300")
	}

	// Finite-Z band z=[20..100] crossed mid-sweep while falling below z=20 by t=1.
	band1 := newTestFace([3]float64{100, -200, 20}, [3]float64{100, 200, 20}, [3]float64{100, 200, 100}, -1, 0, 0)
	band2 := newTestFace([3]float64{100, -200, 20}, [3]float64{100, 200, 100}, [3]float64{100, -200, 100}, -1, 0, 0)
	objFall := newTestUprightObject(20.0, 0.0, 100.0, 40.0, 40.0, 60.0)
	objFall.ent.SetV(200.0*120.0, 0.0, -200.0*120.0)
	cageFall := newTestCage(33, objFall)
	cageFall.Rebuild(18.0, 1)
	cageFall.AddFace(band1, nil)
	cageFall.AddFace(band2, nil)
	cageFall.Commit(nil)
	if cageFall.GetSlotsLen() == 0 {
		t.Fatalf("expected contact against finite-Z band z=[20..100] crossed mid-flight while falling")
	}

	// Phantom infinite-plane extension miss outside finite diagonal triangle must produce 0 contacts.
	dLen := math.Sqrt2 * 0.5
	diag1 := newTestFace([3]float64{100, 100, 0}, [3]float64{200, 200, 0}, [3]float64{200, 200, 200}, -dLen, dLen, 0)
	diag2 := newTestFace([3]float64{100, 100, 0}, [3]float64{200, 200, 200}, [3]float64{100, 100, 200}, -dLen, dLen, 0)
	objDiag := newTestUprightObject(50.0, 80.0, 50.0, 40.0, 40.0, 60.0)
	objDiag.ent.SetV(70.0*120.0, -40.0*120.0, 0.0)
	cageDiag := newTestCage(34, objDiag)
	cageDiag.Rebuild(18.0, 1)
	cageDiag.AddFace(diag1, nil)
	cageDiag.AddFace(diag2, nil)
	cageDiag.Commit(nil)
	if n := cageDiag.GetSlotsLen(); n != 0 {
		t.Fatalf("expected 0 contacts outside finite diagonal wall extension, got %d", n)
	}

	// Upward launch crossing wall z=[0..100] must not be skipped by clampStaticWallShouldSkip.
	wall1 := newTestFace([3]float64{100, -200, 0}, [3]float64{100, 200, 0}, [3]float64{100, 200, 100}, -1, 0, 0)
	wall2 := newTestFace([3]float64{100, -200, 0}, [3]float64{100, 200, 100}, [3]float64{100, -200, 100}, -1, 0, 0)
	objUp := newTestUprightObject(20.0, 0.0, 50.0, 40.0, 40.0, 60.0)
	cageUp := newTestCage(35, objUp)
	cageUp.Rebuild(18.0, 1)
	cageUp.CommitStaticClamp()
	objUp.ent.AddTo(200.0, 0.0, 160.0)
	oCX, oCY, oCZ, oDX, oDY, oDZ, oTX, oTY, oTZ, lAABB := cageUp.BeginStaticClamp()
	cageUp.ClampStaticFace(lAABB, wall1)
	cageUp.ClampStaticFace(lAABB, wall2)
	cageUp.FlushStaticClamp(lAABB)
	cageUp.EndStaticClamp(oCX, oCY, oCZ, oDX, oDY, oDZ, oTX, oTY, oTZ)
	if px, _, _ := objUp.ent.GetCenter(); px > 80.5 {
		t.Fatalf("expected static clamp to stop upward launch at x<=80.5, got x=%v", px)
	}

	// Zero heap allocations in high-speed swept Commit and ClampStaticFace hot paths.
	allocs := testing.AllocsPerRun(20, func() {
		cageLintel.Rebuild(18.0, 1)
		cageLintel.AddFace(lintel1, nil)
		cageLintel.AddFace(lintel2, nil)
		cageLintel.Commit(nil)
		ocx, ocy, ocz, odx, ody, odz, otx, oty, otz, box := cageUp.BeginStaticClamp()
		cageUp.ClampStaticFace(box, wall1)
		cageUp.FlushStaticClamp(box)
		cageUp.EndStaticClamp(ocx, ocy, ocz, odx, ody, odz, otx, oty, otz)
	})
	if allocs > 0 {
		t.Fatalf("expected 0 heap allocations in swept wall Commit/ClampStaticFace hot paths, got %v", allocs)
	}
}

func TestKeepUprightPressedCornerGrazingClamps(t *testing.T) {
	// Pressed slightly against curved corner wall at grazing angle crossing near endpoint
	// must be caught by closestSegmentToTriangleFromOrigin fallback and never miss between tauPlane and endpoint.
	rigid_body.SetFps(60)
	defer rigid_body.SetFps(120)
	f2a := newTestFace([3]float64{3168.00, 3212.00, -22}, [3]float64{3161.29, 3178.34, -22}, [3]float64{3161.29, 3178.34, 154}, -0.9807, 0.1955, 0)
	f2b := newTestFace([3]float64{3168.00, 3212.00, -22}, [3]float64{3161.29, 3178.34, 154}, [3]float64{3168.00, 3212.00, 154}, -0.9807, 0.1955, 0)
	obj58 := newTestUprightObject(3159.966, 3171.839, 54.999, 5.0, 5.0, 66.0)
	cage58 := newTestCage(99, obj58)
	cage58.Rebuild(18.0, 1)
	cage58.CommitStaticClamp()
	obj58.ent.AddTo(3228.837-3159.966, 3338.109-3171.839, 0.0)
	oCX2, oCY2, oCZ2, oDX2, oDY2, oDZ2, oTX2, oTY2, oTZ2, box58 := cage58.BeginStaticClamp()
	m1 := cage58.ClampStaticFace(box58, f2a)
	m2 := cage58.ClampStaticFace(box58, f2b)
	mf := cage58.FlushStaticClamp(box58)
	cage58.EndStaticClamp(oCX2, oCY2, oCZ2, oDX2, oDY2, oDZ2, oTX2, oTY2, oTZ2)
	if !m1 && !m2 && !mf {
		t.Fatalf("expected pressed-corner crossing across (3161.29,3178.34) to clamp")
	}
	px, py, _ := obj58.ent.GetCenter()
	if d := (px-3168.00)*(-0.9807) + (py-3212.00)*0.1955; d < 2.45 {
		t.Fatalf("expected corner sweep clamped to front +n side (d >= 2.45), got d=%.3f at (%.3f,%.3f)", d, px, py)
	}
}
