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

	coord := NewCoordinator(10, 16)
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
		cage.ClampStaticFace(lAABB, floor)
		cage.StaticFaceFlushClamp(lAABB)
		cage.EndStaticClamp(ox, oy, oz, odx, ody, odz, otx, oty, otz)
	}

	queryStacked := func(cage *Cage, wake func(*Solver)) {
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

	// Run calm steps until both stacked boxes settle and sleep as one connected island.
	for step := 0; step < 120; step++ {
		coord.Reset()
		coord.Stage(sBot)
		coord.Stage(sTop)
		coord.Step(queryStacked, clampFloor)
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
	coord.Reset()
	coord.Stage(sBot)
	coord.Stage(sTop)
	coord.Stage(sStriker)
	coord.Step(func(cage *Cage, wake func(*Solver)) {
		if cage == cStriker && sTop.IsSleeping() {
			wake(sTop)
		}
		if cage == cBot {
			wokeBotInSameTick = true
		}
		queryStacked(cage, wake)
	}, clampFloor)

	if sTop.IsSleeping() || sBot.IsSleeping() || !wokeBotInSameTick {
		t.Fatalf("expected wake cascade from top to bottom in same tick, topSleep=%v botSleep=%v botRun=%v",
			sTop.IsSleeping(), sBot.IsSleeping(), wokeBotInSameTick)
	}
}

func TestCoordinatorTracksRemovables(t *testing.T) {
	coord := NewCoordinator(4, 4)
	sActive, _ := newDynamicCoordinatorSolver(0, 0, 10, 20, 20, 20, false)
	sDead, _ := newDynamicCoordinatorSolver(50, 0, 10, 20, 20, 20, false)
	sDead.SetActive(false)

	coord.Reset()
	coord.Stage(sActive)
	coord.Stage(sDead)
	coord.Step(func(*Cage, func(*Solver)) {}, func(*Cage) {})

	rem := coord.Removables()
	if len(rem) != 1 || rem[0] != sDead {
		t.Fatalf("expected inactive solver in Removables(), got %v", rem)
	}
	coord.ClearRemovables()
	if len(coord.Removables()) != 0 {
		t.Fatalf("expected empty Removables() after ClearRemovables()")
	}
}

func TestCoordinatorStepZeroAllocs(t *testing.T) {
	rigid_body.SetFps(60)
	defer rigid_body.SetFps(120)

	coord := NewCoordinator(6, 16)
	s1, _ := newDynamicCoordinatorSolver(0, 0, 10, 20, 20, 20, false)
	floorObj := newTestUprightObject(0, 0, -5, 400, 400, 10)
	floor := &solverParentFace{
		testFace: newTestFace([3]float64{-200, -200, 0}, [3]float64{200, -200, 0}, [3]float64{0, 200, 0}, 0, 0, 1),
		parent:   floorObj.solver,
	}
	queryFn := func(cage *Cage, _ func(*Solver)) {
		cage.AddFace(floor, nil)
		cage.Commit(nil)
	}
	clampFn := func(*Cage) {}

	allocs := testing.AllocsPerRun(50, func() {
		coord.Reset()
		coord.Stage(s1)
		coord.Step(queryFn, clampFn)
	})
	if allocs != 0 {
		t.Fatalf("expected 0 allocations per Coordinator.Step, got %v", allocs)
	}
}
