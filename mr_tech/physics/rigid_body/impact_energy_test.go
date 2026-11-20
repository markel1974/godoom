package rigid_body

import (
	"math"
	"testing"
)

func entityTotalKE(e *Entity) float64 {
	vx, vy, vz := e.GetVelocity()
	wx, wy, wz := e.GetAngularVelocity()
	m := e.GetMass()
	lx, ly, lz := e.Cinematic.world.MultiplySymmetric(wx, wy, wz)
	return 0.5*m*(vx*vx+vy*vy+vz*vz) + 0.5*(wx*lx+wy*ly+wz*lz)
}

func TestKineticEnergyNonIncreasingUnderSpinClampAndWarmStart(t *testing.T) {
	SetFps(120)
	cases := []struct {
		name     string
		rest     float64
		upright2 bool
		points   int
	}{
		{name: "single_pt_spin_clamp", rest: 0.0, upright2: false, points: 1},
		{name: "4pt_box_crash_inelastic", rest: 0.0, upright2: true, points: 4},
		{name: "4pt_box_crash_elastic", rest: 0.5, upright2: true, points: 4},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e1 := NewEntity(30.0, tc.rest, 0.2, int(CombineAverage), int(CombineAverage), 0.0)
			e1.Rebuild(0.0, 8.0, 8.0, 16.0, 16.0, 16.0)
			e1.SetKeepUpright(false)
			e1.SetV(1500.0, 0.0, 0.0)

			e2 := NewEntity(30.0, tc.rest, 0.2, int(CombineAverage), int(CombineAverage), 0.0)
			e2.Rebuild(20.0, 0.0, 0.0, 32.0, 32.0, 64.0)
			e2.SetKeepUpright(tc.upright2)

			var accN, accTx, accTy, accTz, restT [4]float64
			var hasRest [4]bool
			pts := [4][3]float64{
				{20.0, 0.0, 0.0},
				{20.0, 8.0, 0.0},
				{20.0, 8.0, 8.0},
				{20.0, 0.0, 8.0},
			}

			ke0 := entityTotalKE(e1) + entityTotalKE(e2)
			prevKE := ke0
			for frame := 0; frame < 6; frame++ {
				if frame > 0 {
					for p := 0; p < tc.points; p++ {
						accN[p] *= 0.80
						accTx[p] *= 0.80
						accTy[p] *= 0.80
						accTz[p] *= 0.80
						hasRest[p] = false
						restT[p] = 0.0
						e1.WarmStartImpact(e2, -1.0, 0.0, 0.0, pts[p][0], pts[p][1], pts[p][2], &accN[p], &accTx[p], &accTy[p], &accTz[p])
					}
				}
				for iter := 0; iter < 8; iter++ {
					for p := 0; p < tc.points; p++ {
						e1.ResolveImpactAccumulated(e2, -1.0, 0.0, 0.0, pts[p][0], pts[p][1], pts[p][2], &accN[p], &accTx[p], &accTy[p], &accTz[p], &restT[p], &hasRest[p])
					}
				}
				keNow := entityTotalKE(e1) + entityTotalKE(e2)
				if keNow > ke0*(1.0+1e-9) {
					t.Fatalf("frame %d exceeded initial KE: ke0=%g keNow=%g", frame, ke0, keNow)
				}
				if frame > 0 && keNow > prevKE*(1.0+1e-9) {
					t.Fatalf("frame %d injected KE across warm-start: prevKE=%g keNow=%g", frame, prevKE, keNow)
				}
				prevKE = keNow
			}
		})
	}
}

func TestComputeKineticImpactNormalPreSolveAndScaling(t *testing.T) {
	SetFps(120)

	// 1. Head-on inelastic hit (rest=0) with pre-solve snapshot retains exact SI impact Joules post-solve.
	proj := NewEntity(30.0, 0.0, 0.2, int(CombineAverage), int(CombineAverage), 0.0)
	proj.Rebuild(0.0, 0.0, 0.0, 16.0, 16.0, 16.0)
	proj.SetKeepUpright(true)
	proj.SetV(1500.0, 0.0, 0.0)

	target := NewEntity(30.0, 0.0, 0.2, int(CombineAverage), int(CombineAverage), 0.0)
	target.Rebuild(16.0, 0.0, 0.0, 16.0, 16.0, 16.0)
	target.SetKeepUpright(true)

	proj.SnapshotPreSolve()
	target.SnapshotPreSolve()
	proj.ResolveImpact(target, -1.0, 0.0, 0.0, 16.0, 8.0, 8.0)

	proj.SetActiveContact(-1.0, 0.0, 0.0, 16.0, 8.0, 8.0)
	eHeadOn := proj.ComputeKineticImpact(target)
	proj.ClearActiveContact()

	// Reduced mass = 15 kg; 1500 u/s = 30 m/s -> 0.5 * 15 * 30^2 = 6750 J.
	if math.Abs(eHeadOn-6750.0) > 1e-6 {
		t.Fatalf("expected 6750 J for 30kg head-on impact at 1500 u/s (30 m/s), got %v", eHeadOn)
	}

	// 2. Tangential grazing contact (n perpendicular to velocity) yields 0 impact energy.
	proj.SetV(1500.0, 0.0, 0.0)
	target.SetV(0.0, 0.0, 0.0)
	proj.SnapshotPreSolve()
	target.SnapshotPreSolve()
	proj.SetActiveContact(0.0, -1.0, 0.0, 8.0, 16.0, 8.0)
	eGraze := proj.ComputeKineticImpact(target)
	proj.ClearActiveContact()
	if eGraze != 0.0 {
		t.Fatalf("expected 0 J for tangential fly-by contact, got %v", eGraze)
	}

	// 3. Side-by-side monsters where one drops vertically (vz = -24.5 u/s) yields 0 J along horizontal normal.
	m1 := NewEntity(150.0, 0.0, 0.2, int(CombineAverage), int(CombineAverage), 490.0)
	m1.Rebuild(0.0, 0.0, 0.0, 40.0, 40.0, 60.0)
	m1.SetKeepUpright(true)
	m1.SetV(0.0, 0.0, -24.5)

	m2 := NewEntity(150.0, 0.0, 0.2, int(CombineAverage), int(CombineAverage), 490.0)
	m2.Rebuild(40.0, 0.0, 0.0, 40.0, 40.0, 60.0)
	m2.SetKeepUpright(true)
	m2.SetV(0.0, 0.0, 0.0)

	m1.SnapshotPreSolve()
	m2.SnapshotPreSolve()
	m1.SetActiveContact(-1.0, 0.0, 0.0, 40.0, 20.0, 30.0)
	eBrush := m1.ComputeKineticImpact(m2)
	m1.ClearActiveContact()
	if eBrush != 0.0 {
		t.Fatalf("expected 0 J for vertical gravity slide alongside horizontal wall partner, got %v", eBrush)
	}

	// 4. Static partner (mass=0) returns 0 J.
	wall := NewEntity(0.0, 0.0, 0.2, int(CombineAverage), int(CombineAverage), 0.0)
	wall.Rebuild(40.0, 0.0, 0.0, 20.0, 40.0, 60.0)
	if eWall := proj.ComputeKineticImpact(wall); eWall != 0.0 {
		t.Fatalf("expected 0 J for zero-mass partner, got %v", eWall)
	}
}

func TestImpactEnergyZeroAllocs(t *testing.T) {
	a := NewEntity(30.0, 0.0, 0.2, int(CombineAverage), int(CombineAverage), 0.0)
	a.Rebuild(0.0, 0.0, 0.0, 16.0, 16.0, 16.0)
	a.SetV(1500.0, 0.0, 0.0)

	b := NewEntity(30.0, 0.0, 0.2, int(CombineAverage), int(CombineAverage), 0.0)
	b.Rebuild(16.0, 0.0, 0.0, 16.0, 16.0, 16.0)

	allocs := testing.AllocsPerRun(100, func() {
		a.SnapshotPreSolve()
		b.SnapshotPreSolve()
		a.SetActiveContact(-1.0, 0.0, 0.0, 16.0, 8.0, 8.0)
		_ = a.ComputeKineticImpact(b)
		a.ClearActiveContact()
		a.ClearPreSolve()
		b.ClearPreSolve()
	})
	if allocs != 0 {
		t.Fatalf("expected 0 heap allocs per impact evaluation, got %v", allocs)
	}
}

func TestComputeStaticNormalKineticImpactAt(t *testing.T) {
	// 1. Upright body slamming downward onto a horizontal static floor (50 units/m):
	// m = 80 kg, vz = -500 units/s (10 m/s) -> 0.5 * 80 * 10^2 = 4000 J.
	body := NewEntity(80.0, 0.0, 0.2, int(CombineAverage), int(CombineAverage), 0.0)
	body.Rebuild(0.0, 0.0, 0.0, 40.0, 40.0, 60.0)
	body.SetKeepUpright(true)
	body.SetV(0.0, 0.0, -500.0)
	body.SnapshotPreSolve()

	eFloor := body.ComputeStaticNormalKineticImpactAt(0, 0, 1, 20, 20, 0)
	if math.Abs(eFloor-4000.0) > 1e-6 {
		t.Fatalf("expected 4000 J against static floor, got %v", eFloor)
	}

	// 2. Tangential motion along a vertical wall returns 0 J.
	if eWall := body.ComputeStaticNormalKineticImpactAt(-1, 0, 0, 40, 20, 30); eWall != 0.0 {
		t.Fatalf("expected 0 J for tangential slide along static wall, got %v", eWall)
	}

	// 3. Rotating 6DoF box hitting a static floor off-center includes lever-arm closing velocity.
	rotBox := NewEntity(80.0, 0.0, 0.2, int(CombineAverage), int(CombineAverage), 0.0)
	rotBox.Rebuild(0.0, 0.0, 0.0, 40.0, 40.0, 40.0)
	rotBox.SetKeepUpright(false)
	rotBox.SetAngularVelocity(0.0, -10.0, 0.0)
	rotBox.SnapshotPreSolve()
	eEdge := rotBox.ComputeStaticNormalKineticImpactAt(0, 0, 1, 0, 20, 0)
	eCenter := rotBox.ComputeStaticNormalKineticImpactAt(0, 0, 1, 20, 20, 0)
	if eEdge <= 0.0 || eCenter != 0.0 {
		t.Fatalf("expected positive static rotational edge energy and 0 center energy, got edge=%v center=%v", eEdge, eCenter)
	}

	allocs := testing.AllocsPerRun(100, func() {
		_ = rotBox.ComputeStaticNormalKineticImpactAt(0, 0, 1, 0, 20, 0)
	})
	if allocs != 0 {
		t.Fatalf("expected 0 heap allocs per static impact evaluation, got %v", allocs)
	}
}
