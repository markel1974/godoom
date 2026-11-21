package collisions

import (
	"fmt"
	"math"
	"testing"

	"github.com/markel1974/godoom/mr_tech/physics/aabb"
	"github.com/markel1974/godoom/mr_tech/physics/rigid_body"
)

func newDynamicCoordinatorSolver(cx, cy, cz, w, h, d float64, canSleep bool) (*Solver, *Cage) {
	solver := NewSolver(true, 80.0, 0.0, 0.2, int(rigid_body.CombineAverage), int(rigid_body.CombineAverage), 490.0)
	ent := solver.GetEntity()
	ent.Rebuild(cx-w*0.5, cy-h*0.5, cz-d*0.5, w, h, d)
	ent.SetKeepUpright(true)
	ent.LockUpright()
	ent.SetOnGround(false)
	obj := &testObject{solver: solver, ent: ent, box: aabb.NewAABB()}
	_ = solver.Setup(obj)
	solver.SetMaxStep(0)
	solver.SetCanSleep(canSleep)
	solver.SetElasticCollision(noopCollision)
	solver.SetInelasticCollision(noopCollision)
	return solver, solver.GetCage()
}

func TestCoordinatorIslandSleepAndWakeCascade(t *testing.T) {
	rigid_body.SetFps(60)
	defer rigid_body.SetFps(120)

	floorObj := newTestUprightObject(0, 0, -5, 400, 400, 10)
	floor := &solverParentFace{
		testFace: newTestFace([3]float64{-200, -200, 0}, [3]float64{200, -200, 0}, [3]float64{0, 200, 0}, 0, 0, 1),
		parent:   floorObj.solver,
	}

	sBot, cBot := newDynamicCoordinatorSolver(0, 0, 10, 20, 20, 20, true)
	sTop, cTop := newDynamicCoordinatorSolver(0, 0, 30, 20, 20, 20, true)

	// Top face of bottom box (local coords, z=+10) and bottom face of top box (local coords, z=-10).
	botTopFace := newTestFace([3]float64{-10, -10, 10}, [3]float64{10, -10, 10}, [3]float64{0, 10, 10}, 0, 0, 1)
	topBotFace := newTestFace([3]float64{-10, -10, -10}, [3]float64{10, -10, -10}, [3]float64{0, 10, -10}, 0, 0, -1)

	clampFloor := func(cage *Cage) {
		ox, oy, oz, odx, ody, odz, otx, oty, otz, lAABB := cage.BeginStaticClamp()
		cage.StaticFaceClamp(lAABB, floor)
		cage.StaticFaceFlushClamp(lAABB)
		cage.EndStaticClamp(ox, oy, oz, odx, ody, odz, otx, oty, otz)
	}

	queryStacked := func(cage *Cage, wake WakeRunningFn) {
		if cage == cBot {
			cage.AddFace(floor, nil)
			cage.Commit(nil)
			if !cage.HasSeen(cTop) {
				cage.Seen(cTop)
				if !sTop.IsSleeping() {
					cTop.Seen(cage)
				}
				cage.AddFace(topBotFace, botTopFace)
				if sTop.IsSleeping() {
					wake(sTop)
					cTop.Seen(cage)
				}
				cage.Commit(cTop)
			}
		} else if cage == cTop {
			if !cage.HasSeen(cBot) {
				cage.Seen(cBot)
				if !sBot.IsSleeping() {
					cBot.Seen(cage)
				}
				cage.AddFace(botTopFace, topBotFace)
				if sBot.IsSleeping() {
					wake(sBot)
					cBot.Seen(cage)
				}
				cage.Commit(cBot)
			}
		}
	}

	activeQuery := QueryContactsFn(queryStacked)
	coord := NewCoordinator(10, 16, func(cage *Cage, wake WakeRunningFn) {
		activeQuery(cage, wake)
	}, clampFloor)

	// Run calm steps until both stacked boxes settle and sleep as one connected island.
	for step := 0; step < 120; step++ {
		coord.Reset()
		coord.Stage(sBot)
		coord.Stage(sTop)
		coord.Step()
		if sBot.IsSleeping() && sTop.IsSleeping() {
			break
		}
	}
	if !sBot.IsSleeping() || !sTop.IsSleeping() {
		t.Fatalf("expected calm stacked island to enter sleep together, got bot=%v top=%v", sBot.IsSleeping(), sTop.IsSleeping())
	}

	// Active striker wakes only sTop; cascade wake must also wake connected sBot in the same tick.
	sStriker, cStriker := newDynamicCoordinatorSolver(0, 0, 60, 20, 20, 20, true)
	sStriker.GetEntity().SetV(0, 0, -200)

	wokeBotInSameTick := false
	activeQuery = func(cage *Cage, wake WakeRunningFn) {
		if cage == cStriker && sTop.IsSleeping() {
			wake(sTop)
		}
		if cage == cBot {
			wokeBotInSameTick = true
		}
		queryStacked(cage, wake)
	}
	coord.Reset()
	coord.Stage(sBot)
	coord.Stage(sTop)
	coord.Stage(sStriker)
	coord.Step()

	if sTop.IsSleeping() || sBot.IsSleeping() || !wokeBotInSameTick {
		t.Fatalf("expected wake cascade from top to bottom in same tick, topSleep=%v botSleep=%v botRun=%v",
			sTop.IsSleeping(), sBot.IsSleeping(), wokeBotInSameTick)
	}
}

func TestCoordinatorTracksRemovables(t *testing.T) {
	coord := NewCoordinator(4, 4, func(*Cage, WakeRunningFn) {}, func(*Cage) {})
	sActive, _ := newDynamicCoordinatorSolver(0, 0, 10, 20, 20, 20, false)
	sDead, _ := newDynamicCoordinatorSolver(50, 0, 10, 20, 20, 20, false)
	sDead.SetActive(false)

	coord.Reset()
	coord.Stage(sActive)
	coord.Stage(sDead)
	coord.Step()

	if coord.RemovablesLen() != 1 || coord.RemovablesAt(0) != sDead {
		t.Fatalf("expected inactive solver in Removables, got len=%d", coord.RemovablesLen())
	}
	coord.ClearRemovables()
	if coord.RemovablesLen() != 0 {
		t.Fatalf("expected empty Removables after ClearRemovables(), got len=%d", coord.RemovablesLen())
	}
}

func TestCoordinatorStepZeroAllocs(t *testing.T) {
	rigid_body.SetFps(60)
	defer rigid_body.SetFps(120)

	s1, _ := newDynamicCoordinatorSolver(0, 0, 10, 20, 20, 20, false)
	floorObj := newTestUprightObject(0, 0, -5, 400, 400, 10)
	floor := &solverParentFace{
		testFace: newTestFace([3]float64{-200, -200, 0}, [3]float64{200, -200, 0}, [3]float64{0, 200, 0}, 0, 0, 1),
		parent:   floorObj.solver,
	}
	queryFn := func(cage *Cage, _ WakeRunningFn) {
		cage.AddFace(floor, nil)
		cage.Commit(nil)
	}
	clampFn := func(*Cage) {}
	coord := NewCoordinator(6, 16, queryFn, clampFn)

	allocs := testing.AllocsPerRun(50, func() {
		coord.Reset()
		coord.Stage(s1)
		coord.Step()
	})
	if allocs != 0 {
		t.Fatalf("expected 0 allocations per Coordinator.Step, got %v", allocs)
	}
}

func TestCoordinatorStaticFaceCacheReusesBroadphaseAndZeroAllocs(t *testing.T) {
	rigid_body.SetFps(60)
	defer rigid_body.SetFps(120)

	sA, cA := newDynamicCoordinatorSolver(0, 0, 10, 20, 20, 20, false)
	sB, cB := newDynamicCoordinatorSolver(18, 0, 10, 20, 20, 20, false)
	floorObj := newTestUprightObject(0, 0, -5, 400, 400, 10)
	floor := &solverParentFace{
		testFace: newTestFace([3]float64{-200, -200, 0}, [3]float64{200, -200, 0}, [3]float64{0, 200, 0}, 0, 0, 1),
		parent:   floorObj.solver,
	}
	wall := &solverParentFace{
		testFace: newTestFace([3]float64{-40, -200, 0}, [3]float64{-40, 200, 0}, [3]float64{-40, 200, 100}, 1, 0, 0),
		parent:   floorObj.solver,
	}
	botTopFace := newTestFace([3]float64{10, -10, -10}, [3]float64{10, 10, -10}, [3]float64{10, 10, 10}, 1, 0, 0)
	topBotFace := newTestFace([3]float64{-10, -10, -10}, [3]float64{-10, 10, -10}, [3]float64{-10, 10, 10}, -1, 0, 0)

	broadphaseQueries := 0
	clampStaticQueriesA := 0
	clampStaticQueriesB := 0

	queryFn := func(cage *Cage, _ WakeRunningFn) {
		broadphaseQueries++
		cage.AddFace(floor, nil)
		cage.AddFace(wall, nil)
		cage.CachePendingStaticFaces(cage.GetAABB())
		cage.Commit(nil)
		if cage == cA && !cage.HasSeen(cB) {
			cage.Seen(cB)
			cB.Seen(cage)
			cage.AddFace(topBotFace, botTopFace)
			cage.Commit(cB)
		}
	}
	clampFn := func(cage *Cage) {
		for pass := 0; pass < 3; pass++ {
			moved := false
			ox, oy, oz, odx, ody, odz, otx, oty, otz, lAABB := cage.BeginStaticClamp()
			if !cage.HasStaticFaceCache(lAABB) {
				if cage == cA {
					clampStaticQueriesA++
				} else {
					clampStaticQueriesB++
				}
				oMinX, oMinY, oMinZ, oMaxX, oMaxY, oMaxZ := cage.BeginPaddedStaticFaceQuery(lAABB)
				cage.AddCachedStaticFace(floor)
				cage.AddCachedStaticFace(wall)
				cage.EndPaddedStaticFaceQuery(lAABB, oMinX, oMinY, oMinZ, oMaxX, oMaxY, oMaxZ)
			}
			if cage.ClampCachedStaticFaces(lAABB) {
				moved = true
			}
			if cage.StaticFaceFlushClamp(lAABB) {
				moved = true
			}
			cage.EndStaticClamp(ox, oy, oz, odx, ody, odz, otx, oty, otz)
			if !moved {
				break
			}
		}
		cage.CommitStaticClamp()
	}

	coord := NewCoordinator(8, 16, queryFn, clampFn)

	// Subcase 1: Body pushed incrementally beyond zero-displacement Rebuild bounds triggers at most 1 padded ClampStatic refresh per cage across 8 iterations.
	sB.GetEntity().SetV(-100, 0, 0)
	broadphaseQueries = 0
	clampStaticQueriesA = 0
	clampStaticQueriesB = 0
	coord.Reset()
	coord.Stage(sA)
	coord.Stage(sB)
	coord.Step()
	if broadphaseQueries != 2 {
		t.Fatalf("expected 1 initial broadphase query per running cage (total 2), got %d", broadphaseQueries)
	}
	if clampStaticQueriesA > 1 || clampStaticQueriesB > 1 || (clampStaticQueriesA+clampStaticQueriesB) == 0 {
		t.Fatalf("expected at most 1 padded ClampStatic refresh per cage across 8 iterations, got A=%d B=%d", clampStaticQueriesA, clampStaticQueriesB)
	}

	// Subcase 2: Single body sliding gently within its own Rebuild swept bounds reuses QueryContacts cache with 0 ClampStatic re-queries and 0 heap allocations.
	sSole, cSole := newDynamicCoordinatorSolver(0, 0, 10, 20, 20, 20, false)
	soleClampQueries := 0
	soleQueryFn := func(cage *Cage, _ WakeRunningFn) {
		cage.AddFace(floor, nil)
		cage.AddFace(wall, nil)
		cage.CachePendingStaticFaces(cage.GetAABB())
		cage.Commit(nil)
	}
	soleClampFn := func(cage *Cage) {
		for pass := 0; pass < 3; pass++ {
			moved := false
			ox, oy, oz, odx, ody, odz, otx, oty, otz, lAABB := cage.BeginStaticClamp()
			if !cage.HasStaticFaceCache(lAABB) {
				soleClampQueries++
				oMinX, oMinY, oMinZ, oMaxX, oMaxY, oMaxZ := cage.BeginPaddedStaticFaceQuery(lAABB)
				cage.AddCachedStaticFace(floor)
				cage.AddCachedStaticFace(wall)
				cage.EndPaddedStaticFaceQuery(lAABB, oMinX, oMinY, oMinZ, oMaxX, oMaxY, oMaxZ)
			}
			if cage.ClampCachedStaticFaces(lAABB) {
				moved = true
			}
			if cage.StaticFaceFlushClamp(lAABB) {
				moved = true
			}
			cage.EndStaticClamp(ox, oy, oz, odx, ody, odz, otx, oty, otz)
			if !moved {
				break
			}
		}
		cage.CommitStaticClamp()
	}
	soleCoord := NewCoordinator(8, 16, soleQueryFn, soleClampFn)
	sSole.GetEntity().SetV(-60, 0, 0)
	soleCoord.Reset()
	soleCoord.Stage(sSole)
	soleCoord.Step()
	if soleClampQueries != 0 {
		t.Fatalf("expected 0 ClampStatic re-queries within Rebuild swept bounds, got %d (pos=%v)", soleClampQueries, cSole.GetEntity())
	}

	sSole.GetEntity().SetV(0, 0, 0)
	allocs := testing.AllocsPerRun(50, func() {
		soleCoord.Reset()
		soleCoord.Stage(sSole)
		soleCoord.Step()
	})
	if allocs != 0 {
		t.Fatalf("expected 0 allocations per cached Coordinator.Step, got %v", allocs)
	}

	// Subcase 3: When pushed far outside Rebuild bounds during ClampStatic, cache misses at most once and reuses across remaining passes/iterations.
	clampStaticQueriesA = 0
	sA.GetEntity().AddTo(-80, 0, 0)
	for iter := 0; iter < 8; iter++ {
		if cA.NeedsStaticClamp() {
			clampFn(cA)
			cA.CommitStaticClamp()
		}
	}
	if clampStaticQueriesA != 1 {
		t.Fatalf("expected exactly 1 broadphase cache refresh after out-of-bounds push across 8 clamp loops, got %d", clampStaticQueriesA)
	}
	if ax, _, _ := sA.GetEntity().GetCenter(); ax < -30.5 {
		t.Fatalf("expected cached wall clamp to keep entity outside x=-40 wall (x >= -30), got x=%.3f", ax)
	}
}

func TestPrecariousBalanceSettling(t *testing.T) {
	rigid_body.SetFps(120)
	defer rigid_body.SetFps(120)

	floorObj := newTestUprightObject(0, 0, -25, 400, 400, 50)
	wrapFace := func(tf *testFace) IFace {
		return &solverParentFace{testFace: tf, parent: floorObj.solver}
	}

	// Platform ledge ending at x = 0, z = 0 (two triangles covering x in [-100, 0], y in [-100, 100]),
	// with a vertical riser at x = 0, z in [-50, 0] and a lower floor at z = -50.
	ledge1 := newTestFace([3]float64{-100, -100, 0}, [3]float64{0, -100, 0}, [3]float64{0, 100, 0}, 0, 0, 1)
	ledge1.edgeMask = EdgeABActive | EdgeBCActive
	ledge2 := newTestFace([3]float64{-100, -100, 0}, [3]float64{0, 100, 0}, [3]float64{-100, 100, 0}, 0, 0, 1)
	ledge2.edgeMask = EdgeBCActive | EdgeCAActive

	riser1 := newTestFace([3]float64{0, -100, -50}, [3]float64{0, 100, -50}, [3]float64{0, 100, 0}, 1, 0, 0)
	riser1.edgeMask = EdgeABActive | EdgeBCActive
	riser2 := newTestFace([3]float64{0, -100, -50}, [3]float64{0, 100, 0}, [3]float64{0, -100, 0}, 1, 0, 0)
	riser2.edgeMask = EdgeBCActive | EdgeCAActive

	lower1 := newTestFace([3]float64{-100, -100, -50}, [3]float64{200, -100, -50}, [3]float64{200, 100, -50}, 0, 0, 1)
	lower1.edgeMask = EdgeABActive | EdgeBCActive
	lower2 := newTestFace([3]float64{-100, -100, -50}, [3]float64{200, 100, -50}, [3]float64{-100, 100, -50}, 0, 0, 1)
	lower2.edgeMask = EdgeBCActive | EdgeCAActive

	faces := []IFace{wrapFace(ledge1), wrapFace(ledge2), wrapFace(riser1), wrapFace(riser2), wrapFace(lower1), wrapFace(lower2)}
	queryFn := func(cage *Cage, _ WakeRunningFn) {
		for _, f := range faces {
			cage.AddFace(f, nil)
		}
		cage.CachePendingStaticFaces(cage.GetAABB())
		cage.Commit(nil)
	}
	clampFn := func(cage *Cage) {
		for pass := 0; pass < 3; pass++ {
			moved := false
			ox, oy, oz, odx, ody, odz, otx, oty, otz, lAABB := cage.BeginStaticClamp()
			if cage.ClampCachedStaticFaces(lAABB) {
				moved = true
			}
			if cage.StaticFaceFlushClamp(lAABB) {
				moved = true
			}
			cage.EndStaticClamp(ox, oy, oz, odx, ody, odz, otx, oty, otz)
			if !moved {
				break
			}
		}
		cage.CommitStaticClamp()
	}

	assertSettled := func(name string, coord *Coordinator, steps, settleStart int, watch *Solver, participants ...*Solver) {
		t.Helper()
		minZ, maxZ := math.MaxFloat64, -math.MaxFloat64
		for step := 0; step < steps; step++ {
			coord.Reset()
			for _, p := range participants {
				coord.Stage(p)
			}
			coord.Step()
			_, _, cz := watch.GetEntity().GetCenter()
			if step >= settleStart {
				if cz < minZ {
					minZ = cz
				}
				if cz > maxZ {
					maxZ = cz
				}
			}
		}
		_, _, vz := watch.GetEntity().GetVelocity()
		if spanZ := maxZ - minZ; spanZ > 1e-4 {
			t.Fatalf("%s: expected settled vertical spanZ <= 1e-4, got %.6f (minZ=%.4f maxZ=%.4f vz=%.4f)",
				name, spanZ, minZ, maxZ, vz)
		}
		if math.Abs(vz) > 1e-4 {
			t.Fatalf("%s: expected settled vz ~= 0, got %.6f", name, vz)
		}
		for i, p := range participants {
			if !p.IsSleeping() {
				t.Fatalf("%s: participant %d failed to go to sleep (minZ=%.4f maxZ=%.4f)", name, i, minZ, maxZ)
			}
		}
	}

	makeSixDoFBox := func(initX, initZ, w, h, d, friction, tiltX, tiltY, comX, comZ float64) (*Solver, *rigid_body.Entity) {
		solver := NewSolver(true, 15.0, 0.0, friction, int(rigid_body.CombineAverage), int(rigid_body.CombineAverage), 490.0)
		ent := solver.GetEntity()
		ent.Rebuild(initX-w*0.5, -h*0.5, initZ-d*0.5, w, h, d)
		ent.SetKeepUpright(false)
		if comX != 0 || comZ != 0 {
			ent.SetCentroidLocal(comX, 0, comZ)
		}
		if tiltX != 0 || tiltY != 0 {
			ent.SetRotationEuler(tiltX, tiltY, 0)
		}
		_ = solver.Setup(&testObject{solver: solver, ent: ent, box: aabb.NewAABB()})
		solver.SetCanSleep(true)
		solver.SetElasticCollision(noopCollision)
		solver.SetInelasticCollision(noopCollision)
		return solver, ent
	}

	// KeepUpright ledge overhangs (maxStep=0 and maxStep=30): must rest calmly and sleep without vertical hopping.
	for _, xOff := range []float64{-2.0, 0.5, 1.0, 2.0, 3.0, 4.0, 8.0, 12.0} {
		solver, _ := newDynamicCoordinatorSolver(xOff, 0, 30.0, 40, 40, 60, true)
		coord := NewCoordinator(10, 8, queryFn, clampFn)
		assertSettled(fmt.Sprintf("KeepUpright_maxStep0_x=%.1f", xOff), coord, 600, 180, solver, solver)
	}
	for _, xOff := range []float64{4.0, 12.0, 16.0} {
		solver, _ := newDynamicCoordinatorSolver(xOff, 0, 30.0, 40, 40, 60, true)
		solver.SetMaxStep(30.0)
		coord := NewCoordinator(10, 8, queryFn, clampFn)
		assertSettled(fmt.Sprintf("KeepUpright_maxStep30_x=%.1f", xOff), coord, 480, 180, solver, solver)
	}

	// 6-DoF boxes overhanging or tilted on ledge edge (including offset CoM drops).
	for _, tc := range []struct {
		name  string
		initX float64
		initZ float64
		tiltX float64
		tiltY float64
		comX  float64
		comZ  float64
	}{
		{"tilted_0.04_at_-1", -1.0, 10.0, 0.0, 0.04, 0, 0},
		{"straddle_-2.0", -2.0, 10.0, 0.0, 0.03, 0, 0},
		{"straddle_-0.5", -0.5, 10.0, 0.0, 0.03, 0, 0},
		{"straddle_0.5_tip", 0.5, 10.0, 0.0, 0.03, 0, 0},
		{"overhang_5", 5.0, 10.0, 0.0, 0.0, 0, 0},
		{"overhang_9.5", 9.5, 10.0, 0.0, 0.0, 0, 0},
		{"tilted_0.3_at_0", 0.0, 10.0, 0.0, 0.3, 0, 0},
		{"tilted_0.5_at_-2", -2.0, 10.0, 0.2, 0.5, 0, 0},
		{"drop_overhang_4", 4.0, 12.0, 0.0, 0.0, 0, 0},
		{"drop_overhang_7", 7.0, 12.0, 0.0, 0.0, 0, 0},
		{"drop_overhang_9", 9.0, 12.0, 0.0, 0.0, 0, 0},
		{"drop_tilt_0.1_at_0", 0.0, 12.0, 0.0, 0.1, 0, 0},
		{"offset_com_edge", 0.0, 10.5, 0.0, 0.05, 3.0, -4.0},
		{"offset_com_drop", -1.0, 13.0, 0.0, 0.08, -2.0, -4.0},
	} {
		solver, _ := makeSixDoFBox(tc.initX, tc.initZ, 20, 20, 20, 0.2, tc.tiltX, tc.tiltY, tc.comX, tc.comZ)
		coord := NewCoordinator(10, 8, queryFn, clampFn)
		assertSettled("6DoF_ledge_"+tc.name, coord, 600, 360, solver, solver)
	}

	// Dynamic 6-DoF box balanced precariously atop another dynamic 6-DoF box rim.
	{
		sBot := NewSolver(true, 80.0, 0.0, 0.4, int(rigid_body.CombineAverage), int(rigid_body.CombineAverage), 490.0)
		eBot := sBot.GetEntity()
		eBot.Rebuild(-30, -10, 0, 20, 20, 20)
		eBot.SetKeepUpright(false)
		_ = sBot.Setup(&testObject{solver: sBot, ent: eBot, box: aabb.NewAABB()})
		sBot.SetCanSleep(true)
		sBot.SetElasticCollision(noopCollision)
		sBot.SetInelasticCollision(noopCollision)

		sTop := NewSolver(true, 15.0, 0.0, 0.4, int(rigid_body.CombineAverage), int(rigid_body.CombineAverage), 490.0)
		eTop := sTop.GetEntity()
		eTop.Rebuild(-21, -10, 20.1, 20, 20, 20)
		eTop.SetKeepUpright(false)
		eTop.SetRotationEuler(0, 0.15, 0)
		_ = sTop.Setup(&testObject{solver: sTop, ent: eTop, box: aabb.NewAABB()})
		sTop.SetCanSleep(true)
		sTop.SetElasticCollision(noopCollision)
		sTop.SetInelasticCollision(noopCollision)

		cBot := sBot.GetCage()
		cTop := sTop.GetCage()
		botFace := newTestFace([3]float64{-10, -10, 10}, [3]float64{10, -10, 10}, [3]float64{0, 10, 10}, 0, 0, 1)
		topFace := newTestFace([3]float64{-10, -10, -10}, [3]float64{10, -10, -10}, [3]float64{0, 10, -10}, 0, 0, -1)

		dynQuery := func(cage *Cage, wake WakeRunningFn) {
			queryFn(cage, wake)
			if cage == cBot && !cage.HasSeen(cTop) {
				cage.Seen(cTop)
				if !sTop.IsSleeping() {
					cTop.Seen(cage)
				}
				cage.AddFace(topFace, botFace)
				if sTop.IsSleeping() {
					wake(sTop)
					cTop.Seen(cage)
				}
				cage.Commit(cTop)
			} else if cage == cTop && !cage.HasSeen(cBot) {
				cage.Seen(cBot)
				if !sBot.IsSleeping() {
					cBot.Seen(cage)
				}
				cage.AddFace(botFace, topFace)
				if sBot.IsSleeping() {
					wake(sBot)
					cBot.Seen(cage)
				}
				cage.Commit(cBot)
			}
		}
		coord := NewCoordinator(10, 8, dynQuery, clampFn)
		assertSettled("6DoF_box_on_box_overhang", coord, 600, 360, sTop, sBot, sTop)
	}

	// 6-DoF box resting in bilico over a gentle convex ridge at x = 0.
	{
		s5 := math.Sin(0.08)
		c5 := math.Cos(0.08)
		ridgeL := wrapFace(newTestFace([3]float64{-100, -100, -100 * (s5 / c5)}, [3]float64{0, -100, 0}, [3]float64{0, 100, 0}, -s5, 0, c5))
		ridgeR := wrapFace(newTestFace([3]float64{0, -100, 0}, [3]float64{100, -100, -100 * (s5 / c5)}, [3]float64{100, 100, -100 * (s5 / c5)}, s5, 0, c5))
		convexQuery := func(cage *Cage, _ WakeRunningFn) {
			cage.AddFace(ridgeL, nil)
			cage.AddFace(ridgeR, nil)
			cage.CachePendingStaticFaces(cage.GetAABB())
			cage.Commit(nil)
		}
		convexClamp := func(cage *Cage) {
			ox, oy, oz, odx, ody, odz, otx, oty, otz, lAABB := cage.BeginStaticClamp()
			cage.ClampCachedStaticFaces(lAABB)
			cage.StaticFaceFlushClamp(lAABB)
			cage.EndStaticClamp(ox, oy, oz, odx, ody, odz, otx, oty, otz)
			cage.CommitStaticClamp()
		}
		for _, initX := range []float64{0.0, 0.2, 2.0} {
			solver, _ := makeSixDoFBox(initX, 10.0, 20, 20, 20, 0.5, 0, 0, 0, 0)
			coord := NewCoordinator(10, 8, convexQuery, convexClamp)
			assertSettled(fmt.Sprintf("6DoF_convex_ridge_x=%.1f", initX), coord, 600, 240, solver, solver)
		}
	}

	// Tall/thin 6-DoF plank leaning tilted against a step/wall corner near ledge edge.
	{
		stepWall1 := wrapFace(newTestFace([3]float64{20, -100, 0}, [3]float64{20, 100, 0}, [3]float64{20, 100, 40}, -1, 0, 0))
		stepWall2 := wrapFace(newTestFace([3]float64{20, -100, 0}, [3]float64{20, 100, 40}, [3]float64{20, -100, 40}, -1, 0, 0))
		cornerFaces := []IFace{wrapFace(ledge1), wrapFace(ledge2), wrapFace(lower1), wrapFace(lower2), stepWall1, stepWall2}
		qCorner := func(cage *Cage, _ WakeRunningFn) {
			for _, f := range cornerFaces {
				cage.AddFace(f, nil)
			}
			cage.CachePendingStaticFaces(cage.GetAABB())
			cage.Commit(nil)
		}
		cCorner := func(cage *Cage) {
			ox, oy, oz, odx, ody, odz, otx, oty, otz, lAABB := cage.BeginStaticClamp()
			cage.ClampCachedStaticFaces(lAABB)
			cage.StaticFaceFlushClamp(lAABB)
			cage.EndStaticClamp(ox, oy, oz, odx, ody, odz, otx, oty, otz)
			cage.CommitStaticClamp()
		}
		solver, _ := makeSixDoFBox(-2.0, 15.0, 6, 6, 30, 0.2, 0, 0.35, 0, 0)
		coord := NewCoordinator(10, 8, qCorner, cCorner)
		assertSettled("6DoF_leaning_plank", coord, 600, 300, solver, solver)
	}

	// KeepUpright object perched on the rim of a dynamic 6-DoF box.
	{
		sCrate := NewSolver(true, 100.0, 0.0, 0.4, int(rigid_body.CombineAverage), int(rigid_body.CombineAverage), 490.0)
		eCrate := sCrate.GetEntity()
		eCrate.Rebuild(-40, -40, 0, 40, 40, 20)
		eCrate.SetKeepUpright(false)
		_ = sCrate.Setup(&testObject{solver: sCrate, ent: eCrate, box: aabb.NewAABB()})
		sCrate.SetCanSleep(true)
		sCrate.SetElasticCollision(noopCollision)
		sCrate.SetInelasticCollision(noopCollision)

		sCyl, cCyl := newDynamicCoordinatorSolver(4.0, 0, 40.0, 20, 20, 40, true)
		cCrate := sCrate.GetCage()
		crateFace := newTestFace([3]float64{-20, -20, 10}, [3]float64{20, -20, 10}, [3]float64{0, 20, 10}, 0, 0, 1)
		cylFace := newTestFace([3]float64{-10, -10, -20}, [3]float64{10, -10, -20}, [3]float64{0, 10, -20}, 0, 0, -1)

		qDyn := func(cage *Cage, wake WakeRunningFn) {
			queryFn(cage, wake)
			if cage == cCrate && !cage.HasSeen(cCyl) {
				cage.Seen(cCyl)
				if !sCyl.IsSleeping() {
					cCyl.Seen(cage)
				}
				cage.AddFace(cylFace, crateFace)
				if sCyl.IsSleeping() {
					wake(sCyl)
					cCyl.Seen(cage)
				}
				cage.Commit(cCyl)
			} else if cage == cCyl && !cage.HasSeen(cCrate) {
				cage.Seen(cCrate)
				if !sCrate.IsSleeping() {
					cCrate.Seen(cage)
				}
				cage.AddFace(crateFace, cylFace)
				if sCrate.IsSleeping() {
					wake(sCrate)
					cCrate.Seen(cage)
				}
				cage.Commit(cCrate)
			}
		}
		coord := NewCoordinator(10, 8, qDyn, clampFn)
		assertSettled("KeepUpright_on_dyn_box_rim", coord, 600, 300, sCyl, sCrate, sCyl)
	}

	// Jointed 2-limb 6-DoF ragdolls straddling or perched on the ledge edge: must settle cleanly without vertical bouncing.
	{
		sTorso, _ := newFreeSixDoFSolver(40.0, -10, 0, 10, 20, 20, 20, 490.0, true)
		sLimb, _ := newFreeSixDoFSolver(15.0, 10, 0, 10, 20, 14, 14, 490.0, true)
		sTorso.GetEntity().WakeUp()
		sLimb.GetEntity().WakeUp()
		coord := NewCoordinator(10, 8, queryFn, clampFn)
		jc := NewJointConstraint(sTorso, sLimb, 10, 0, 0, -10, 0, 0, aabb.NewQuaternionIdentity(), 1.2, 6.0)
		coord.AddJoint(jc)
		assertSettled("6DoF_ragdoll_overhang_drop", coord, 900, 600, sLimb, sTorso, sLimb)
	}
	{
		sTorso, _ := newFreeSixDoFSolver(200.0, -14, 0, 10, 20, 20, 20, 490.0, true)
		sLimb, _ := newFreeSixDoFSolver(10.0, 6, 0, 10, 20, 14, 14, 490.0, true)
		sTorso.GetEntity().WakeUp()
		sLimb.GetEntity().WakeUp()
		coord := NewCoordinator(10, 8, queryFn, clampFn)
		jc := NewJointConstraint(sTorso, sLimb, 10, 0, 0, -10, 0, 0, aabb.NewQuaternionIdentity(), 1.2, 6.0)
		coord.AddJoint(jc)
		assertSettled("6DoF_ragdoll_perched_limb_overhang", coord, 900, 600, sLimb, sTorso, sLimb)
	}

	// Zero heap allocations for 6-DoF OBB static edge evaluation and clipped prism depth in evalStaticFace.
	{
		objDyn := newTestDynamicObject(0.0, 0.0, 10.0, 20, 20, 20)
		objDyn.ent.SetRotationEuler(0.05, 0.08, 0.2)
		cageDyn := objDyn.solver.GetCage()
		slopeFace := wrapFace(newTestFace([3]float64{-100, -100, -8}, [3]float64{0, -100, 0}, [3]float64{0, 100, 0}, -0.08, 0, 0.9968))
		sess := uint64(50)
		allocs := testing.AllocsPerRun(50, func() {
			sess++
			cageDyn.Rebuild(0, sess)
			cageDyn.AddFace(slopeFace, nil)
			cageDyn.Commit(nil)
		})
		if allocs != 0 {
			t.Fatalf("expected 0 heap allocs for 6-DoF static ridge Commit, got %v", allocs)
		}
		if cageDyn.GetSlotsLen() == 0 {
			t.Fatalf("expected static ridge Commit to produce contact slots")
		}
	}
}
