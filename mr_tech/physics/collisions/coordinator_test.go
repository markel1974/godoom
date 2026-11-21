package collisions

import (
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
