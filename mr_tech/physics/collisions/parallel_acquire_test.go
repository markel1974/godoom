package collisions

import (
	"fmt"
	"math"
	"sync"
	"testing"

	"github.com/markel1974/godoom/mr_tech/physics/aabb"
	"github.com/markel1974/godoom/mr_tech/physics/rigid_body"
)

func buildDeterministicClusterFixtures(seedCount int) ([]*Solver, []IFace, []IFace, QueryContactsFn, ClampStaticFn) {
	floorObj := newTestUprightObject(0, 0, -25, 400, 400, 50)
	staticFaces := []IFace{
		&solverParentFace{testFace: newTestFace([3]float64{-200, -200, 0}, [3]float64{200, -200, 0}, [3]float64{200, 200, 0}, 0, 0, 1), parent: floorObj.solver},
		&solverParentFace{testFace: newTestFace([3]float64{-200, -200, 0}, [3]float64{200, 200, 0}, [3]float64{-200, 200, 0}, 0, 0, 1), parent: floorObj.solver},
	}
	solvers := make([]*Solver, seedCount)
	faces := make([]IFace, seedCount)
	cols := 4
	for i := range seedCount {
		layer := i / (cols * cols)
		slot := i % (cols * cols)
		row := slot / cols
		col := slot % cols
		x := -35.0 + float64(col)*22.0 + math.Sin(float64(i)*1.3)*1.5
		y := -35.0 + float64(row)*22.0 + math.Cos(float64(i)*1.9)*1.5
		z := 5.0 + float64(layer)*8.5
		s := NewSolver(true, 30.0, 0.0, 0.2, int(rigid_body.CombineAverage), int(rigid_body.CombineAverage), 490.0)
		ent := s.GetEntity()
		ent.Rebuild(x-8, y-6, z-4, 16, 12, 8)
		ent.SetKeepUpright(false)
		ent.SetRotationEuler(float64(i%4)*0.15, float64((i+1)%4)*0.12, float64(i)*0.4)
		_ = s.Setup(&testObject{solver: s, ent: ent, box: aabb.NewAABB()})
		s.SetCanSleep(true)
		s.SetElasticCollision(noopCollision)
		s.SetInelasticCollision(noopCollision)
		solvers[i] = s
		faces[i] = &solverParentFace{
			testFace: newTestFace([3]float64{-8, -6, -4}, [3]float64{8, -6, -4}, [3]float64{0, 6, -4}, 0, 0, 1),
			parent:   s,
		}
	}

	queryFn := func(cage *Cage, wake WakeRunningFn) {
		for _, sf := range staticFaces {
			cage.AddFace(sf, nil)
		}
		cage.CachePendingStaticFaces(cage.GetAABB())
		cage.Commit(nil)

		la := cage.GetAABB()
		lMinX, lMinY, lMinZ, lMaxX, lMaxY, lMaxZ := la.GetMinMax()
		for j := range seedCount {
			rCage := solvers[j].GetCage()
			wasSleep := solvers[j].IsSleeping()
			if !cage.CanAcquireDynamic(rCage) {
				continue
			}
			ra := rCage.GetAABB()
			rMinX, rMinY, rMinZ, rMaxX, rMaxY, rMaxZ := ra.GetMinMax()
			if lMaxX < rMinX || lMinX > rMaxX || lMaxY < rMinY || lMinY > rMaxY || lMaxZ < rMinZ || lMinZ > rMaxZ {
				continue
			}
			cage.AddFace(faces[j], faces[0])
			if wasSleep {
				if !cage.WillHitDynamic(rCage) {
					cage.DiscardPendingFaces()
					continue
				}
				if cage.ShouldWakeDynamic(rCage) {
					wake(solvers[j])
				}
			}
			cage.Commit(rCage)
		}
	}
	clampFn := func(cage *Cage) {
		solver := cage.GetSolver()
		ox, oy, oz, odx, ody, odz, otx, oty, otz, lAABB := cage.BeginStaticClamp()
		solver.ClampCachedStaticFaces(lAABB)
		solver.StaticFaceFlushClamp(lAABB)
		cage.EndStaticClamp(ox, oy, oz, odx, ody, odz, otx, oty, otz)
		cage.CommitStaticClamp()
	}
	return solvers, faces, staticFaces, queryFn, clampFn
}

func TestParallelAcquireDeterminismAcrossModesAndOrder(t *testing.T) {
	rigid_body.SetFps(120)
	defer rigid_body.SetFps(120)

	const nBodies = 24
	const steps = 40

	type bodySnapshot struct {
		cx, cy, cz     float64
		vx, vy, vz     float64
		qx, qy, qz, qw float64
		sleep          bool
	}
	runSimulation := func(parallel bool, reverseStage bool) []bodySnapshot {
		solvers, _, _, qFn, cFn := buildDeterministicClusterFixtures(nBodies)
		// Align cage IDs deterministically per run regardless of global _counter.
		for i, s := range solvers {
			s.GetCage().id = uint64(i + 1)
		}
		coord := NewCoordinator(10, nBodies+8, qFn, cFn)
		coord.SetParallelAcquire(parallel)
		coord.parallelMinBodies = 2

		for range steps {
			coord.Reset()
			if reverseStage {
				for i := nBodies - 1; i >= 0; i-- {
					coord.Stage(solvers[i])
				}
			} else {
				for i := 0; i < nBodies; i++ {
					coord.Stage(solvers[i])
				}
			}
			coord.Step()
		}
		out := make([]bodySnapshot, nBodies)
		for i, s := range solvers {
			ent := s.GetEntity()
			cx, cy, cz := ent.GetCenter()
			vx, vy, vz := ent.GetVelocity()
			qx, qy, qz, qw := ent.GetQRotation().Get()
			out[i] = bodySnapshot{
				cx: cx, cy: cy, cz: cz,
				vx: vx, vy: vy, vz: vz,
				qx: qx, qy: qy, qz: qz, qw: qw,
				sleep: s.IsSleeping(),
			}
		}
		return out
	}

	seqSnap := runSimulation(false, false)
	parSnap := runSimulation(true, false)

	for i := range nBodies {
		if seqSnap[i] != parSnap[i] {
			t.Fatalf("body %d parallel vs sequential diverged:\nseq=%+v\npar=%+v", i, seqSnap[i], parSnap[i])
		}
	}
}

func TestPile60HzSequentialVsParallel(t *testing.T) {
	t.Run("seq_100_60hz", func(t *testing.T) {
		runPileSimulationWithMode(t, 60, 100, 5, 60.0, false)
	})
	t.Run("par_100_60hz", func(t *testing.T) {
		runPileSimulationWithMode(t, 60, 100, 5, 60.0, true)
	})
	t.Run("seq_500_60hz", func(t *testing.T) {
		runPileSimulationWithMode(t, 60, 500, 10, 120.0, false)
	})
	t.Run("par_500_60hz", func(t *testing.T) {
		runPileSimulationWithMode(t, 60, 500, 10, 120.0, true)
	})
}

func runPileSimulationWithMode(t *testing.T, fps uint64, nBoxes int, cols int, wallHalf float64, parallel bool) {
	t.Helper()
	rigid_body.SetFps(fps)
	defer rigid_body.SetFps(120)

	floorSolver := NewSolver(false, 0.0, 0.0, 0.2, int(rigid_body.CombineAverage), int(rigid_body.CombineAverage), 0.0)
	floorObj := &testObject{solver: floorSolver, ent: floorSolver.GetEntity(), box: aabb.NewAABB()}
	floorObj.ent.Rebuild(-500, -500, -50, 1000, 1000, 50)
	floorObj.ent.SetKeepUpright(true)
	floorObj.ent.LockUpright()
	floorObj.ent.SetKinematic(true)
	_ = floorSolver.Setup(floorObj)
	staticFaces := []IFace{
		&solverParentFace{testFace: newTestFace([3]float64{-250, -250, 0}, [3]float64{250, -250, 0}, [3]float64{250, 250, 0}, 0, 0, 1), parent: floorObj.solver},
		&solverParentFace{testFace: newTestFace([3]float64{-250, -250, 0}, [3]float64{250, 250, 0}, [3]float64{-250, 250, 0}, 0, 0, 1), parent: floorObj.solver},
		&solverParentFace{testFace: newTestFace([3]float64{wallHalf, -250, 0}, [3]float64{wallHalf, -250, 300}, [3]float64{wallHalf, 250, 300}, -1, 0, 0), parent: floorObj.solver},
		&solverParentFace{testFace: newTestFace([3]float64{wallHalf, -250, 0}, [3]float64{wallHalf, 250, 300}, [3]float64{wallHalf, 250, 0}, -1, 0, 0), parent: floorObj.solver},
		&solverParentFace{testFace: newTestFace([3]float64{-wallHalf, -250, 0}, [3]float64{-wallHalf, 250, 300}, [3]float64{-wallHalf, -250, 300}, 1, 0, 0), parent: floorObj.solver},
		&solverParentFace{testFace: newTestFace([3]float64{-wallHalf, -250, 0}, [3]float64{-wallHalf, 250, 0}, [3]float64{-wallHalf, 250, 300}, 1, 0, 0), parent: floorObj.solver},
		&solverParentFace{testFace: newTestFace([3]float64{-250, wallHalf, 0}, [3]float64{250, wallHalf, 300}, [3]float64{-250, wallHalf, 300}, 0, -1, 0), parent: floorObj.solver},
		&solverParentFace{testFace: newTestFace([3]float64{-250, wallHalf, 0}, [3]float64{250, wallHalf, 0}, [3]float64{250, wallHalf, 300}, 0, -1, 0), parent: floorObj.solver},
		&solverParentFace{testFace: newTestFace([3]float64{-250, -wallHalf, 0}, [3]float64{-250, -wallHalf, 300}, [3]float64{250, -wallHalf, 300}, 0, 1, 0), parent: floorObj.solver},
		&solverParentFace{testFace: newTestFace([3]float64{-250, -wallHalf, 0}, [3]float64{250, -wallHalf, 300}, [3]float64{250, -wallHalf, 0}, 0, 1, 0), parent: floorObj.solver},
	}

	perLayer := cols * cols
	spanStart := -float64(cols-1) * 10.5
	solvers := make([]*Solver, nBoxes)
	faces := make([]IFace, nBoxes)
	for i := range nBoxes {
		layer := i / perLayer
		slot := i % perLayer
		row := slot / cols
		col := slot % cols
		x := spanStart + float64(col)*21.0 + math.Sin(float64(i)*1.7)*2.5
		y := spanStart + float64(row)*21.0 + math.Cos(float64(i)*2.3)*2.5
		z := 6.0 + float64(layer)*11.0
		s := NewSolver(true, 30.0, 0.0, 0.2, int(rigid_body.CombineAverage), int(rigid_body.CombineAverage), 490.0)
		ent := s.GetEntity()
		ent.Rebuild(x-8, y-6, z-4, 16, 12, 8)
		s.SetMaxStep(0.0)
		ent.SetKeepUpright(false)
		ent.SetRotationEuler(float64(i%5)*0.19, float64((i+2)%5)*0.17, float64(i)*0.55)
		_ = s.Setup(&testObject{solver: s, ent: ent, box: aabb.NewAABB()})
		s.SetCanSleep(true)
		s.SetElasticCollision(noopCollision)
		s.SetInelasticCollision(noopCollision)
		solvers[i] = s
		faces[i] = &solverParentFace{
			testFace: newTestFace([3]float64{-8, -6, -4}, [3]float64{8, -6, -4}, [3]float64{0, 6, -4}, 0, 0, 1),
			parent:   s,
		}
	}

	queryFn := func(cage *Cage, wake WakeRunningFn) {
		for _, sf := range staticFaces {
			cage.AddFace(sf, nil)
		}
		cage.CachePendingStaticFaces(cage.GetAABB())
		cage.Commit(nil)
		la := cage.GetAABB()
		lMinX, lMinY, lMinZ, lMaxX, lMaxY, lMaxZ := la.GetMinMax()
		for j := range nBoxes {
			rCage := solvers[j].GetCage()
			wasSleep := solvers[j].IsSleeping()
			if !cage.CanAcquireDynamic(rCage) {
				continue
			}
			ra := rCage.GetAABB()
			rMinX, rMinY, rMinZ, rMaxX, rMaxY, rMaxZ := ra.GetMinMax()
			if lMaxX < rMinX || lMinX > rMaxX || lMaxY < rMinY || lMinY > rMaxY || lMaxZ < rMinZ || lMinZ > rMaxZ {
				continue
			}
			cage.AddFace(faces[j], faces[0])
			if wasSleep {
				if !cage.WillHitDynamic(rCage) {
					cage.DiscardPendingFaces()
					continue
				}
				if cage.ShouldWakeDynamic(rCage) {
					wake(solvers[j])
				}
			}
			cage.Commit(rCage)
		}
	}
	clampFn := func(cage *Cage) {
		solver := cage.GetSolver()
		ox, oy, oz, odx, ody, odz, otx, oty, otz, lAABB := cage.BeginStaticClamp()
		solver.ClampCachedStaticFaces(lAABB)
		solver.StaticFaceFlushClamp(lAABB)
		cage.EndStaticClamp(ox, oy, oz, odx, ody, odz, otx, oty, otz)
		cage.CommitStaticClamp()
	}

	coord := NewCoordinator(10, nBoxes+16, queryFn, clampFn)
	coord.SetParallelAcquire(parallel)
	maxSteps := int(fps) * 8
	for range maxSteps {
		coord.Reset()
		for _, s := range solvers {
			coord.Stage(s)
		}
		coord.Step()
	}

	awake := 0
	for _, s := range solvers {
		if !s.IsSleeping() {
			awake++
		}
	}
	if awake != 0 {
		t.Fatalf("parallel=%v nBoxes=%d fps=%d: still awake=%d (running=%d)", parallel, nBoxes, fps, awake, coord.RunningLen())
	}
}

func TestSlotsRotateRangeToEnd(t *testing.T) {
	e := make([]*Entry, 6)
	for i := range e {
		e[i] = makeCradleEntry(float64(i), 0, 1, ImpactInelastic)
	}

	s := makeTestSlots([]*Entry{e[0], e[1], e[2], e[3], e[4], e[5]})
	s.RotateRangeToEnd(1, 3)
	want := []*Entry{e[0], e[3], e[4], e[5], e[1], e[2]}
	for i, w := range want {
		if s.At(i) != w {
			t.Fatalf("after RotateRangeToEnd(1, 3): idx %d got %p, want %p", i, s.At(i), w)
		}
	}

	s0 := makeTestSlots([]*Entry{e[0], e[1], e[2], e[3], e[4]})
	s0.RotateRangeToEnd(0, 2)
	want0 := []*Entry{e[2], e[3], e[4], e[0], e[1]}
	for i, w := range want0 {
		if s0.At(i) != w {
			t.Fatalf("after RotateRangeToEnd(0, 2): idx %d got %p, want %p", i, s0.At(i), w)
		}
	}

	// Boundary no-ops preserve element order.
	sNoop := makeTestSlots([]*Entry{e[0], e[1], e[2], e[3]})
	sNoop.RotateRangeToEnd(2, 2)
	sNoop.RotateRangeToEnd(3, 1)
	sNoop.RotateRangeToEnd(1, 4)
	for i := range 4 {
		if sNoop.At(i) != e[i] {
			t.Fatalf("no-op RotateRangeToEnd changed idx %d", i)
		}
	}

	allocs := testing.AllocsPerRun(50, func() {
		s0.RotateRangeToEnd(0, 3)
		s0.RotateRangeToEnd(0, 2)
	})
	if allocs != 0 {
		t.Fatalf("RotateRangeToEnd allocated %v times, want 0", allocs)
	}
}

func TestParallelAcquireMultiWaveWakeCascade(t *testing.T) {
	rigid_body.SetFps(60)
	defer rigid_body.SetFps(120)

	const nSleeping = 12
	solvers := make([]*Solver, nSleeping+1)
	faces := make([]IFace, nSleeping+1)
	floorObj := newTestUprightObject(0, 0, -25, 400, 400, 50)
	floorFace := &solverParentFace{
		testFace: newTestFace([3]float64{-200, -200, 0}, [3]float64{200, -200, 0}, [3]float64{0, 200, 0}, 0, 0, 1),
		parent:   floorObj.solver,
	}

	for i := range nSleeping {
		s := NewSolver(true, 30.0, 0.0, 0.2, int(rigid_body.CombineAverage), int(rigid_body.CombineAverage), 490.0)
		ent := s.GetEntity()
		z := 4.0 + float64(i)*8.0
		ent.Rebuild(-8, -6, z-4, 16, 12, 8)
		ent.SetKeepUpright(false)
		ent.SetOnGround(true)
		_ = s.Setup(&testObject{solver: s, ent: ent, box: aabb.NewAABB()})
		s.SetCanSleep(true)
		s.SetElasticCollision(noopCollision)
		s.SetInelasticCollision(noopCollision)
		s.EnterSleep()
		solvers[i] = s
		faces[i] = &solverParentFace{
			testFace: newTestFace([3]float64{-8, -6, -4}, [3]float64{8, -6, -4}, [3]float64{0, 6, -4}, 0, 0, 1),
			parent:   s,
		}
	}

	// Fast striker hitting top sleeping body and triggering multi-wave wake across stack.
	striker := NewSolver(true, 60.0, 0.0, 0.2, int(rigid_body.CombineAverage), int(rigid_body.CombineAverage), 490.0)
	stEnt := striker.GetEntity()
	topZ := 4.0 + float64(nSleeping-1)*8.0
	stEnt.Rebuild(-8, -6, topZ+6.0, 16, 12, 8)
	stEnt.SetKeepUpright(false)
	stEnt.SetV(0, 0, -300)
	_ = striker.Setup(&testObject{solver: striker, ent: stEnt, box: aabb.NewAABB()})
	striker.SetCanSleep(true)
	striker.SetElasticCollision(noopCollision)
	striker.SetInelasticCollision(noopCollision)
	solvers[nSleeping] = striker
	faces[nSleeping] = &solverParentFace{
		testFace: newTestFace([3]float64{-8, -6, -4}, [3]float64{8, -6, -4}, [3]float64{0, 6, -4}, 0, 0, 1),
		parent:   striker,
	}

	queryFn := func(cage *Cage, wake WakeRunningFn) {
		cage.AddFace(floorFace, nil)
		cage.CachePendingStaticFaces(cage.GetAABB())
		cage.Commit(nil)
		for j, rSolver := range solvers {
			rCage := rSolver.GetCage()
			wasSleep := rSolver.IsSleeping()
			if !cage.CanAcquireDynamic(rCage) {
				continue
			}
			cage.AddFace(faces[j], faces[0])
			if wasSleep {
				if !cage.WillHitDynamic(rCage) {
					cage.DiscardPendingFaces()
					continue
				}
				if cage.ShouldWakeDynamic(rCage) {
					wake(rSolver)
				}
			}
			cage.Commit(rCage)
		}
	}
	clampFn := func(cage *Cage) {
		solver := cage.GetSolver()
		ox, oy, oz, odx, ody, odz, otx, oty, otz, lAABB := cage.BeginStaticClamp()
		solver.ClampCachedStaticFaces(lAABB)
		solver.StaticFaceFlushClamp(lAABB)
		cage.EndStaticClamp(ox, oy, oz, odx, ody, odz, otx, oty, otz)
		cage.CommitStaticClamp()
	}

	coord := NewCoordinator(10, nSleeping+4, queryFn, clampFn)
	coord.SetParallelAcquire(true)
	coord.parallelMinBodies = 2

	for range 10 {
		coord.Reset()
		for _, s := range solvers {
			coord.Stage(s)
		}
		coord.Step()
	}

	woken := 0
	for i := range nSleeping {
		if !solvers[i].IsSleeping() {
			woken++
		}
	}
	if woken == 0 {
		t.Fatal("expected striker impact to wake sleeping boxes via parallel wavefront")
	}
}

func TestAABBTreeConcurrentReadAndZeroAllocs(t *testing.T) {
	tree := aabb.NewTree(256, 0.5)
	objs := make([]*testObject, 256)
	for i := range objs {
		x := float64(i%16) * 15.0
		y := float64(i/16) * 15.0
		obj := newTestUprightObject(x, y, 0, 10, 10, 10)
		objs[i] = obj
		tree.InsertObject(obj)
	}

	probe := newTestUprightObject(30, 30, 0, 40, 40, 20)
	allocs := testing.AllocsPerRun(50, func() {
		hits := 0
		tree.QueryOverlaps(probe, func(aabb.IObject) bool {
			hits++
			return false
		})
		if hits == 0 {
			t.Fatal("expected overlap hits")
		}
	})
	if allocs != 0 {
		t.Fatalf("Tree.QueryOverlaps allocated %v times, want 0", allocs)
	}

	rayAllocs := testing.AllocsPerRun(50, func() {
		hits := 0
		tree.QueryRay(-20, 35, 5, 1, 0, 0, 500, func(_ aabb.IObject, d float64) (float64, bool) {
			hits++
			return d, false
		})
		if hits == 0 {
			t.Fatal("expected ray hits")
		}
	})
	if rayAllocs != 0 {
		t.Fatalf("Tree.QueryRay allocated %v times, want 0", rayAllocs)
	}

	var wg sync.WaitGroup
	wg.Add(8)
	for g := range 8 {
		go func(id int) {
			defer wg.Done()
			localProbe := newTestUprightObject(float64(id*20), float64(id*10), 0, 45, 45, 20)
			cx, cy, cz := localProbe.ent.GetCenter()
			for range 100 {
				tree.QueryOverlaps(localProbe, func(aabb.IObject) bool {
					return false
				})
				tree.QueryPoint3d(cx, cy, cz, func(aabb.IObject) bool {
					return false
				})
				tree.QueryRay(-10, float64(id*15)+5, 5, 1, 0.01, 0, 500, func(_ aabb.IObject, _ float64) (float64, bool) {
					return 500, false
				})
			}
		}(g)
	}
	wg.Wait()
}

func BenchmarkDynamicPile500Acquire(b *testing.B) {
	rigid_body.SetFps(120)
	defer rigid_body.SetFps(120)

	for _, parallel := range []bool{false, true} {
		name := "Sequential"
		if parallel {
			name = "Parallel"
		}
		b.Run(fmt.Sprintf("%s/500Bodies", name), func(b *testing.B) {
			solvers, _, _, qFn, cFn := buildDeterministicClusterFixtures(500)
			coord := NewCoordinator(10, 512, qFn, cFn)
			coord.SetParallelAcquire(parallel)
			// Warmup 1 step so buckets and buffers pre-size cleanly.
			coord.Reset()
			for _, s := range solvers {
				coord.Stage(s)
			}
			coord.Step()

			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				coord.Reset()
				for _, s := range solvers {
					s.WakeUp()
					coord.Stage(s)
				}
				coord.Step()
			}
		})
	}
}

type nonSolidParentFace struct {
	*solverParentFace
}

func (*nonSolidParentFace) IsSolid() bool { return false }

func TestParallelAcquireAsymmetricSolidFacesAndCallback(t *testing.T) {
	rigid_body.SetFps(60)
	defer rigid_body.SetFps(120)

	for _, parallel := range []bool{false, true} {
		// Low-ID door (kinematic, upright, solid faces) created before high-ID player (upright, non-solid nil-material faces).
		doorObj := newTestUprightObject(100, 0, 32, 32, 32, 64)
		doorSolver := doorObj.solver
		doorSolver.GetEntity().SetKinematic(true)
		doorSolver.SetIgnoreStatic(true)
		doorSolver.SetCanSleep(true)

		doorTriggered := false
		doorSolver.SetElasticCollision(func(other aabb.IObject, _ float64) {
			if otherObj, ok := other.(*testObject); ok && otherObj.solver != doorSolver {
				doorTriggered = true
			}
		})

		playerObj := newTestUprightObject(60, 0, 28, 32, 32, 56)
		playerSolver := playerObj.solver
		playerSolver.SetCanSleep(false)

		if doorSolver.GetCage().id >= playerSolver.GetCage().id {
			t.Fatalf("expected door cage id (%d) < player cage id (%d)", doorSolver.GetCage().id, playerSolver.GetCage().id)
		}

		doorFace := &solverParentFace{
			testFace: newTestFace([3]float64{-16, -16, -32}, [3]float64{16, -16, -32}, [3]float64{16, 16, 32}, -1, 0, 0),
			parent:   doorSolver,
		}
		playerFace := &nonSolidParentFace{
			solverParentFace: &solverParentFace{
				testFace: newTestFace([3]float64{-16, -16, -28}, [3]float64{16, -16, -28}, [3]float64{16, 16, 28}, 1, 0, 0),
				parent:   playerSolver,
			},
		}

		pairObj := [2]*testObject{doorObj, playerObj}
		pairFace := [2]IFace{doorFace, playerFace}

		queryFn := func(cage *Cage, wake WakeRunningFn) {
			selfIdx := 0
			otherIdx := 1
			if cage == playerSolver.GetCage() {
				selfIdx = 1
				otherIdx = 0
			}
			rCage := pairObj[otherIdx].solver.GetCage()
			if cage.IsAcquiring() {
				if !cage.CanAcquireDynamic(rCage) {
					return
				}
			} else {
				if cage.HasSeen(rCage) {
					return
				}
				cage.Seen(rCage)
				if !pairObj[otherIdx].solver.IsSleeping() {
					rCage.Seen(cage)
				}
			}
			wasSleep := pairObj[otherIdx].solver.IsSleeping()
			la := cage.GetAABB()
			ra := rCage.GetAABB()
			lMinX, lMinY, lMinZ, lMaxX, lMaxY, lMaxZ := la.GetMinMax()
			rMinX, rMinY, rMinZ, rMaxX, rMaxY, rMaxZ := ra.GetMinMax()
			if lMaxX < rMinX || lMinX > rMaxX || lMaxY < rMinY || lMinY > rMaxY || lMaxZ < rMinZ || lMinZ > rMaxZ {
				return
			}
			cage.AddFace(pairFace[otherIdx], pairFace[selfIdx])
			if wasSleep {
				if !cage.WillHitDynamic(rCage) {
					cage.DiscardPendingFaces()
					return
				}
				if cage.ShouldWakeDynamic(rCage) && wake != nil {
					wake(pairObj[otherIdx].solver)
				}
			}
			cage.Commit(rCage)
		}

		coord := NewCoordinator(8, 16, queryFn, func(*Cage) {})
		coord.parallelMinBodies = 1
		coord.SetParallelAcquire(parallel)

		maxPlayerX := -math.MaxFloat64
		for step := 0; step < 30; step++ {
			playerObj.ent.SetV(250, 0, 0)
			coord.Reset()
			// Stage player before door just like builder/things.go does in real frames.
			coord.Stage(playerSolver)
			coord.Stage(doorSolver)
			coord.Step()
			px, _, _ := playerObj.ent.GetCenter()
			if px > maxPlayerX {
				maxPlayerX = px
			}
		}

		if !doorTriggered {
			t.Fatalf("parallel=%v: expected player collision to trigger door elastic callback", parallel)
		}
		// Door center is X=100 (half-extent 16 -> minX=84), player half-extent 16 -> blocked around X<=68+slop.
		if maxPlayerX > 72.0 {
			t.Fatalf("parallel=%v: player passed through door: maxPlayerX=%.3f > 72.0", parallel, maxPlayerX)
		}
	}
}
