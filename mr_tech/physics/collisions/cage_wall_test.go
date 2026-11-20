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
	edgeMask    uint8
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
		v0:       v0,
		v1:       v1,
		v2:       v2,
		nx:       nx,
		ny:       ny,
		nz:       nz,
		box:      box,
		edgeMask: EdgeAllActive,
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
func (f *testFace) ActiveEdgeMask() uint8             { return f.edgeMask }

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

func newTestDynamicObject(cx, cy, cz, w, h, d float64) *testObject {
	solver := NewSolver(true, 80.0, 0.0, 0.2, int(rigid_body.CombineAverage), int(rigid_body.CombineAverage), 0.0)
	ent := solver.GetEntity()
	ent.Rebuild(cx-w*0.5, cy-h*0.5, cz-d*0.5, w, h, d)
	ent.SetKeepUpright(false)
	obj := &testObject{
		solver: solver,
		ent:    ent,
		box:    aabb.NewAABB(),
	}
	_ = solver.Setup(obj)
	return obj
}

func TestActiveEdgeMaskSuppressesSeamDeflection(t *testing.T) {
	// Subcase A: Single half-quad wall triangle with inactive diagonal CA (edge 2) tested alone
	// in both 3D proximity and Z-slab crossing must keep flat wall normal (0,-1,0) without X kick.
	t1 := newTestFace([3]float64{0, 200, 0}, [3]float64{200, 200, 0}, [3]float64{200, 200, 100}, 0, -1, 0)
	t1.edgeMask = EdgeABActive | EdgeBCActive

	obj3D := newTestUprightObject(115.0, 188.0, 72.0, 40.0, 40.0, 60.0)
	cage3D := newTestCage(100, obj3D)
	cage3D.Rebuild(18.0, 1)
	cage3D.AddFace(t1, nil)
	cage3D.Commit(nil)
	if cage3D.GetSlotsLen() == 0 {
		t.Fatalf("subcase A (3D): expected wall contact on t1")
	}
	for i := 0; i < cage3D.GetSlotsLen(); i++ {
		nx, ny, _ := cage3D.GetSlot(i).GetNormal()
		if math.Abs(nx) > 1e-6 || ny > -0.999 {
			t.Fatalf("subcase A (3D): inactive diagonal leaked lateral push (%v, %v)", nx, ny)
		}
	}

	objSlab := newTestUprightObject(114.0, 195.0, 88.0, 40.0, 40.0, 60.0)
	objSlab.ent.SetV(0.0, 5.5*120.0, 0.0)
	cageSlab := newTestCage(101, objSlab)
	cageSlab.Rebuild(18.0, 1)
	cageSlab.AddFace(t1, nil)
	cageSlab.Commit(nil)
	if cageSlab.GetSlotsLen() == 0 {
		t.Fatalf("subcase A (Z-slab): expected wall contact on t1")
	}
	for i := 0; i < cageSlab.GetSlotsLen(); i++ {
		nx, ny, _ := cageSlab.GetSlot(i).GetNormal()
		if math.Abs(nx) > 1e-6 || ny > -0.999 {
			t.Fatalf("subcase A (Z-slab): inactive diagonal leaked lateral push (%v, %v)", nx, ny)
		}
	}

	// Subcase B: Walkable 30-deg ramp with bottom edge AB (y=100, z=0) marked inactive (concave floor crease)
	// must keep floor/ramp normal instead of converting into a horizontal -Y wall kick.
	sin30 := 0.5
	cos30 := math.Sqrt(3.0) * 0.5
	rampZ := 100.0 * (sin30 / cos30)
	ramp := newTestFace(
		[3]float64{0, 100, 0},
		[3]float64{200, 100, 0},
		[3]float64{0, 200, rampZ},
		0.0, -sin30, cos30,
	)
	ramp.edgeMask = EdgeBCActive | EdgeCAActive

	objRamp := newTestUprightObject(50.0, 89.0, 15.0, 40.0, 40.0, 60.0)
	cageRamp := newTestCage(102, objRamp)
	cageRamp.Rebuild(18.0, 1)
	cageRamp.AddFace(ramp, nil)
	cageRamp.Commit(nil)
	if cageRamp.GetSlotsLen() == 0 {
		t.Fatalf("subcase B: expected ramp floor contact")
	}
	_, rny, rnz := cageRamp.GetSlot(0).GetNormal()
	if rnz < 0.8 || rny < -0.55 {
		t.Fatalf("subcase B: inactive concave ramp crease produced horizontal kick normal (ny=%v, nz=%v)", rny, rnz)
	}
	if !cageRamp.HasFloor() {
		t.Fatalf("subcase B: expected HasFloor() to remain true on walkable ramp")
	}

	// Subcase C: Dynamic (!KeepUpright) rigid body slightly behind a single-sided wall's inactive diagonal seam
	// must be ignored rather than triggering a phantom backface seam catch.
	tBack := newTestFace([3]float64{0, 200, 0}, [3]float64{200, 200, 0}, [3]float64{200, 200, 100}, 0, -1, 0)
	tBack.singleSided = true
	tBack.edgeMask = EdgeABActive | EdgeBCActive

	objDyn := newTestDynamicObject(110.0, 202.0, 62.0, 20.0, 20.0, 20.0)
	objDyn.ent.SetV(0.0, 2.0*120.0, 0.0)
	cageDyn := newTestCage(103, objDyn)
	cageDyn.Rebuild(0.0, 1)
	cageDyn.AddFace(tBack, nil)
	cageDyn.Commit(nil)
	if n := cageDyn.GetSlotsLen(); n != 0 {
		t.Fatalf("subcase C: expected 0 backface seam contacts for !KeepUpright body on inactive diagonal, got %d", n)
	}
}

func TestKeepUprightDynamicColliderSwitching(t *testing.T) {
	floor := newTestFace(
		[3]float64{-200, -200, 0},
		[3]float64{600, -200, 0},
		[3]float64{100, 600, 0},
		0, 0, 1,
	)

	// Start upright: ignores yaw rotation and external render GetAABB() bloat/offset,
	// emitting a single closest contact point on static faces.
	obj := newTestDynamicObject(100.0, 100.0, 29.0, 40.0, 40.0, 60.0)
	obj.ent.SetKeepUpright(true)
	obj.ent.SetRotationAngleZ(math.Pi / 4.0)
	obj.ent.GetAABB().Rebuild(100.0-35.0, 100.0-10.0, 29.0-5.0, 100.0+65.0, 100.0+90.0, 29.0+75.0)

	cage := newTestCage(200, obj)
	cage.Rebuild(0.0, 1)

	cx, cy, cz := cage.GetC()
	if math.Abs(cx-100.0) > 1e-6 || math.Abs(cy-100.0) > 1e-6 || math.Abs(cz-29.0) > 1e-6 {
		t.Fatalf("upright phase: expected center (100,100,29) ignoring render AABB offset, got (%v,%v,%v)", cx, cy, cz)
	}
	rx, ry, rz := cage.GetRad()
	if math.Abs(rx-20.0) > 1e-6 || math.Abs(ry-20.0) > 1e-6 || math.Abs(rz-30.0) > 1e-6 {
		t.Fatalf("upright phase: expected upright half-extents (20,20,30) ignoring yaw and render AABB bloat, got (%v,%v,%v)", rx, ry, rz)
	}

	cage.AddFace(floor, nil)
	cage.Commit(nil)
	if n := cage.GetSlotsLen(); n != 1 {
		t.Fatalf("upright phase: expected 1 closest contact while KeepUpright=true, got %d", n)
	}
	if !cage.HasFloor() {
		t.Fatalf("upright phase: expected HasFloor() while grounded")
	}
	cage.CommitStaticClamp()

	// Disable KeepUpright dynamically mid-tick (before Rebuild): NeedsStaticClamp triggers
	// and BeginStaticClamp immediately refreshes extents to the rotated OBB.
	obj.ent.SetKeepUpright(false)
	if !cage.NeedsStaticClamp() {
		t.Fatalf("mid-tick disable: expected NeedsStaticClamp() after SetKeepUpright(false)")
	}
	oCX, oCY, oCZ, oDX, oDY, oDZ, oTX, oTY, oTZ, _ := cage.BeginStaticClamp()
	if clx, cly, clz := cage.GetRad(); clx <= 28.0 || cly <= 28.0 || math.Abs(clz-30.0) > 1e-6 {
		t.Fatalf("mid-tick disable: expected BeginStaticClamp to refresh rotated OBB half-extents (~28.28,~28.28,30), got (%v,%v,%v)", clx, cly, clz)
	}
	cage.EndStaticClamp(oCX, oCY, oCZ, oDX, oDY, oDZ, oTX, oTY, oTZ)
	cage.CommitStaticClamp()

	// Next tick with KeepUpright=false: uses rotated OBB extents and multi-point OBBManifold.
	cage.Rebuild(0.0, 2)
	if orx, ory, orz := cage.GetRad(); orx <= 28.0 || ory <= 28.0 || math.Abs(orz-30.0) > 1e-6 {
		t.Fatalf("OBB phase: expected rotated OBB half-extents (~28.28,~28.28,30), got (%v,%v,%v)", orx, ory, orz)
	}
	cage.AddFace(floor, nil)
	cage.Commit(nil)
	if n := cage.GetSlotsLen(); n <= 1 {
		t.Fatalf("OBB phase: expected multi-point OBBManifold (>1 contacts) after SetKeepUpright(false), got %d", n)
	}
	if !cage.HasFloor() {
		t.Fatalf("OBB phase: expected HasFloor() for grounded OBB")
	}
	for i := 0; i < cage.GetSlotsLen(); i++ {
		slot := cage.GetSlot(i)
		_, _, nz := slot.GetNormal()
		if nz < 0.999 || slot.GetPenetration() < 0.99 {
			t.Fatalf("OBB phase: unexpected slot %d normalZ=%v pen=%v", i, nz, slot.GetPenetration())
		}
	}

	// Re-enable KeepUpright dynamically: returns to yaw-invariant upright collider extents.
	obj.ent.SetKeepUpright(true)
	if !cage.NeedsStaticClamp() {
		t.Fatalf("re-enable: expected NeedsStaticClamp() after SetKeepUpright(true)")
	}
	cage.Rebuild(18.0, 3)
	if brx, bry, brz := cage.GetRad(); math.Abs(brx-20.0) > 1e-6 || math.Abs(bry-20.0) > 1e-6 || math.Abs(brz-30.0) > 1e-6 {
		t.Fatalf("re-enable phase: expected upright half-extents (20,20,30) restored, got (%v,%v,%v)", brx, bry, brz)
	}
}

func TestDynamicKeepUprightSymmetricCollision(t *testing.T) {
	rigid_body.SetFps(120)
	solidFace := newTestFace([3]float64{-1, -1, -1}, [3]float64{1, -1, -1}, [3]float64{0, 1, -1}, 0, 0, 1)

	// Subcase A: Two KeepUpright bodies collide horizontally as yaw-invariant cylinders,
	// avoiding OBB square-corner hits at diagonal offsets.
	objGapA := newTestUprightObject(0.0, 0.0, 30.0, 40.0, 40.0, 60.0)
	objGapB := newTestUprightObject(29.0, 29.0, 30.0, 40.0, 40.0, 60.0)
	cageGapA := newTestCage(300, objGapA)
	cageGapB := newTestCage(301, objGapB)
	cageGapA.Rebuild(18.0, 1)
	cageGapB.Rebuild(18.0, 1)
	cageGapA.AddFace(solidFace, solidFace)
	cageGapA.Commit(cageGapB)
	if n := cageGapA.GetSlotsLen(); n != 0 {
		t.Fatalf("subcase A (separated corners): expected 0 cylindrical contacts at dist=41.01 > sumR=40, got %d", n)
	}

	objCylA := newTestUprightObject(0.0, 0.0, 30.0, 40.0, 40.0, 60.0)
	objCylB := newTestUprightObject(28.0, 28.0, 30.0, 40.0, 40.0, 60.0)
	objCylA.ent.SetRotationAngleZ(math.Pi / 4.0)
	objCylB.ent.SetRotationAngleZ(math.Pi / 6.0)
	cageCylA := newTestCage(302, objCylA)
	cageCylB := newTestCage(303, objCylB)
	cageCylA.Rebuild(18.0, 2)
	cageCylB.Rebuild(18.0, 2)
	cageCylA.AddFace(solidFace, solidFace)
	cageCylA.Commit(cageCylB)
	if n := cageCylA.GetSlotsLen(); n != 1 {
		t.Fatalf("subcase A (diagonal overlap): expected 1 radial contact, got %d", n)
	}
	slotA := cageCylA.GetSlot(0)
	nxA, nyA, nzA := slotA.GetNormal()
	expectedPen := 40.0 - math.Hypot(28.0, 28.0)
	if math.Abs(nxA+math.Sqrt(0.5)) > 1e-4 || math.Abs(nyA+math.Sqrt(0.5)) > 1e-4 || math.Abs(nzA) > 1e-6 || math.Abs(slotA.GetPenetration()-expectedPen) > 1e-4 {
		t.Fatalf("subcase A: expected radial normal (-0.707,-0.707,0) and pen=%.4f, got (%v,%v,%v) pen=%v", expectedPen, nxA, nyA, nzA, slotA.GetPenetration())
	}
	if cageCylB.GetSlotsLen() != 1 {
		t.Fatalf("subcase A: expected reciprocal slot on cageCylB, got %d", cageCylB.GetSlotsLen())
	}
	nxB, nyB, _ := cageCylB.GetSlot(0).GetNormal()
	if math.Abs(nxB+nxA) > 1e-6 || math.Abs(nyB+nyA) > 1e-6 {
		t.Fatalf("subcase A: expected reciprocal opposite normal on B, got A=(%v,%v) B=(%v,%v)", nxA, nyA, nxB, nyB)
	}

	// Subcase B: Mixed KeepUpright cylinder vs rotated 6-DoF OBB produces smooth rounded corner contact
	// in both evaluation orders (cylinder->OBB and OBB->cylinder).
	objBox := newTestDynamicObject(0.0, 0.0, 30.0, 40.0, 40.0, 60.0)
	objBox.ent.SetRotationAngleZ(math.Pi / 4.0)
	objCyl := newTestUprightObject(47.0, 1.0, 30.0, 40.0, 40.0, 60.0)
	cageBox := newTestCage(304, objBox)
	cageCyl := newTestCage(305, objCyl)
	cageBox.Rebuild(0.0, 3)
	cageCyl.Rebuild(18.0, 3)
	cageCyl.AddFace(solidFace, solidFace)
	cageCyl.Commit(cageBox)
	if cageCyl.GetSlotsLen() != 1 || cageBox.GetSlotsLen() != 1 {
		t.Fatalf("subcase B: expected 1 mutual contact between cylinder and rotated OBB corner, got cyl=%d box=%d", cageCyl.GetSlotsLen(), cageBox.GetSlotsLen())
	}
	cSlot := cageCyl.GetSlot(0)
	cnx, cny, _ := cSlot.GetNormal()
	cornerX := 20.0 * math.Sqrt2
	expDist := math.Hypot(47.0-cornerX, 1.0)
	expNX := (47.0 - cornerX) / expDist
	expNY := 1.0 / expDist
	if math.Abs(cnx-expNX) > 1e-4 || math.Abs(cny-expNY) > 1e-4 || math.Abs(cSlot.GetPenetration()-(20.0-expDist)) > 1e-4 {
		t.Fatalf("subcase B (cyl->OBB): expected rounded corner normal (%v,%v) pen=%v, got (%v,%v) pen=%v",
			expNX, expNY, 20.0-expDist, cnx, cny, cSlot.GetPenetration())
	}

	// Reverse evaluation order (OBB->cyl) must yield exact opposite normal and matching penetration.
	cageBox.Rebuild(0.0, 4)
	cageCyl.Rebuild(18.0, 4)
	cageBox.AddFace(solidFace, solidFace)
	cageBox.Commit(cageCyl)
	if cageBox.GetSlotsLen() != 1 {
		t.Fatalf("subcase B (OBB->cyl): expected 1 contact on box, got %d", cageBox.GetSlotsLen())
	}
	bnx, bny, _ := cageBox.GetSlot(0).GetNormal()
	if math.Abs(bnx+expNX) > 1e-4 || math.Abs(bny+expNY) > 1e-4 {
		t.Fatalf("subcase B (OBB->cyl): expected negated radial normal (%v,%v), got (%v,%v)", -expNX, -expNY, bnx, bny)
	}

	// Subcase C: Dynamic 6-DoF box resting flat atop a KeepUpright cylinder receives multi-point cap support and hasFloor.
	objPillar := newTestUprightObject(0.0, 0.0, 10.0, 40.0, 40.0, 20.0)
	objCrate := newTestDynamicObject(0.0, 0.0, 29.5, 30.0, 30.0, 20.0)
	cagePillar := newTestCage(306, objPillar)
	cageCrate := newTestCage(307, objCrate)
	cagePillar.Rebuild(0.0, 5)
	cageCrate.Rebuild(0.0, 5)
	cageCrate.AddFace(solidFace, solidFace)
	cageCrate.Commit(cagePillar)
	if !cageCrate.HasFloor() || cageCrate.GetSlotsLen() <= 1 {
		t.Fatalf("subcase C: expected multi-point Z-cap support (>1 slots) and HasFloor() on crate atop cylinder, got slots=%d hasFloor=%v",
			cageCrate.GetSlotsLen(), cageCrate.HasFloor())
	}

	// Subcase D: High-speed swept KeepUpright tunneling check + 0 heap allocations.
	objBullet := newTestUprightObject(-25.0, 0.0, 30.0, 4.0, 4.0, 4.0)
	objBullet.ent.SetV(60.0*120.0, 0.0, 0.0)
	objTarget := newTestUprightObject(0.0, 0.0, 30.0, 40.0, 40.0, 60.0)
	cageBullet := newTestCage(308, objBullet)
	cageTarget := newTestCage(309, objTarget)
	cageBullet.Rebuild(0.0, 6)
	cageTarget.Rebuild(18.0, 6)
	cageBullet.AddFace(solidFace, solidFace)
	cageBullet.Commit(cageTarget)
	if cageBullet.GetSlotsLen() != 1 {
		t.Fatalf("subcase D: expected swept KeepUpright tunneling hit, got %d slots", cageBullet.GetSlotsLen())
	}
	sweptNX, _, _ := cageBullet.GetSlot(0).GetNormal()
	if sweptNX > -0.999 {
		t.Fatalf("subcase D: expected entry-side -X normal preventing forward tunnel, got nx=%v", sweptNX)
	}

	session := uint64(10)
	allocs := testing.AllocsPerRun(100, func() {
		cageCyl.Rebuild(18.0, session)
		cageBox.Rebuild(0.0, session)
		cageCyl.AddFace(solidFace, solidFace)
		cageCyl.Commit(cageBox)
		cageCylA.Rebuild(18.0, session)
		cageCylB.Rebuild(18.0, session)
		cageCylA.AddFace(solidFace, solidFace)
		cageCylA.Commit(cageCylB)
		session++
	})
	if allocs != 0 {
		t.Fatalf("subcase D: expected 0 heap allocs in dynamic KeepUpright manifold evaluation, got %v", allocs)
	}

	// Subcase E: Dynamic runtime SetKeepUpright toggle switches collider behavior immediately.
	objDynA := newTestDynamicObject(0.0, 0.0, 30.0, 40.0, 40.0, 60.0)
	objDynB := newTestDynamicObject(29.0, 29.0, 30.0, 40.0, 40.0, 60.0)
	objDynA.ent.SetKeepUpright(true)
	objDynB.ent.SetKeepUpright(true)
	cageDynA := newTestCage(310, objDynA)
	cageDynB := newTestCage(311, objDynB)
	cageDynA.Rebuild(18.0, 200)
	cageDynB.Rebuild(18.0, 200)
	cageDynA.AddFace(solidFace, solidFace)
	cageDynA.Commit(cageDynB)
	if cageDynA.GetSlotsLen() != 0 {
		t.Fatalf("subcase E (upright): expected 0 cylindrical contacts before SetKeepUpright(false), got %d", cageDynA.GetSlotsLen())
	}
	objDynA.ent.SetKeepUpright(false)
	objDynB.ent.SetKeepUpright(false)
	cageDynA.Rebuild(0.0, 201)
	cageDynB.Rebuild(0.0, 201)
	cageDynA.AddFace(solidFace, solidFace)
	cageDynA.Commit(cageDynB)
	if cageDynA.GetSlotsLen() == 0 {
		t.Fatalf("subcase E (OBB after toggle): expected OBB box overlap after disabling KeepUpright, got 0 contacts")
	}
}
