package collisions

import (
	"math"
	"testing"

	"github.com/markel1974/godoom/mr_tech/physics/aabb"
	"github.com/markel1974/godoom/mr_tech/physics/rigid_body"
)

func runPileSimulation(t *testing.T, nBoxes int, cols int, wallHalf float64, wakeOnBroadphase bool) {
	t.Helper()
	rigid_body.SetFps(120)
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
		// Confining walls to force a realistic multi-layer heap/mound.
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
		// Doom 3 medikit parameters: mass=30, restitution=0, friction=0.2, gForce=490, maxStep=0
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
			if rCage == cage || cage.HasSeen(rCage) {
				continue
			}
			ra := rCage.GetAABB()
			rMinX, rMinY, rMinZ, rMaxX, rMaxY, rMaxZ := ra.GetMinMax()
			if lMaxX < rMinX || lMinX > rMaxX || lMaxY < rMinY || lMinY > rMaxY || lMaxZ < rMinZ || lMinZ > rMaxZ {
				continue
			}
			wasSleep := solvers[j].IsSleeping()
			cage.Seen(rCage)
			if !wasSleep {
				rCage.Seen(cage)
			}
			if !wakeOnBroadphase {
				cage.AddFace(faces[j], faces[0])
				if wasSleep {
					if !cage.WillHitDynamic(rCage) {
						cage.DiscardPendingFaces()
						continue
					}
					if cage.ShouldWakeDynamic(rCage) {
						wake(solvers[j])
						rCage.Seen(cage)
					}
				}
				cage.Commit(rCage)
				continue
			}
			// Reproduce broadphase-wake behavior for comparison.
			_, dx, dy, dz := cage.TranslateCage(0, rCage)
			zEnt := cage.Translate(1, faces[j], dx, dy, dz)
			za := zEnt.GetAABB()
			zMinX, zMinY, zMinZ, zMaxX, zMaxY, zMaxZ := za.GetMinMax()
			if !(zMaxX < -8 || zMinX > 8 || zMaxY < -6 || zMinY > 6 || zMaxZ < -4 || zMinZ > 4) {
				if wasSleep {
					wake(solvers[j])
					rCage.Seen(cage)
				}
				cage.AddFace(faces[j], faces[0])
				cage.Commit(rCage)
			}
		}
	}
	clampFn := func(cage *Cage) {
		ox, oy, oz, odx, ody, odz, otx, oty, otz, lAABB := cage.BeginStaticClamp()
		cage.ClampCachedStaticFaces(lAABB)
		cage.StaticFaceFlushClamp(lAABB)
		cage.EndStaticClamp(ox, oy, oz, odx, ody, odz, otx, oty, otz)
		cage.CommitStaticClamp()
	}

	coord := NewCoordinator(10, nBoxes+16, queryFn, clampFn)
	const maxSteps = 720
	for step := 0; step < maxSteps; step++ {
		coord.Reset()
		for _, s := range solvers {
			coord.Stage(s)
		}
		coord.Step()
	}

	awake := 0
	for idx, s := range solvers {
		ent := s.GetEntity()
		cx, cy, cz := ent.GetCenter()
		if math.IsNaN(cx) || math.IsNaN(cy) || math.IsNaN(cz) {
			t.Fatalf("box %d produced NaN center: (%v,%v,%v)", idx, cx, cy, cz)
		}
		if cz < 3.5 {
			t.Fatalf("box %d sunk below floor: cz=%.4f", idx, cz)
		}
		if !s.IsSleeping() {
			awake++
		}
	}
	if awake != 0 {
		t.Fatalf("expected all %d 6-DoF boxes to settle and sleep within %d steps, still awake=%d (running=%d)",
			nBoxes, maxSteps, awake, coord.RunningLen())
	}

	// Dropping a fast striker into the settled mound must wake the struck local cluster cleanly.
	topIdx := nBoxes - 1
	tx, ty, tz := solvers[topIdx].GetEntity().GetCenter()
	striker := NewSolver(true, 30.0, 0.0, 0.2, int(rigid_body.CombineAverage), int(rigid_body.CombineAverage), 490.0)
	sEnt := striker.GetEntity()
	sEnt.Rebuild(tx-8, ty-6, tz+14.0, 16, 12, 8)
	sEnt.SetKeepUpright(false)
	sEnt.SetV(0, 0, -150)
	_ = striker.Setup(&testObject{solver: striker, ent: sEnt, box: aabb.NewAABB()})
	striker.SetCanSleep(true)
	striker.SetElasticCollision(noopCollision)
	striker.SetInelasticCollision(noopCollision)
	sFace := &solverParentFace{
		testFace: newTestFace([3]float64{-8, -6, -4}, [3]float64{8, -6, -4}, [3]float64{0, 6, -4}, 0, 0, 1),
		parent:   striker,
	}
	solvers = append(solvers, striker)
	faces = append(faces, sFace)
	nBoxes++
	for step := 0; step < 30; step++ {
		coord.Reset()
		for _, s := range solvers {
			coord.Stage(s)
		}
		coord.Step()
	}
	wokenCount := 0
	for i := 0; i < nBoxes-1; i++ {
		if !solvers[i].IsSleeping() {
			wokenCount++
		}
	}
	if wokenCount == 0 {
		t.Fatalf("expected striker impact to wake at least the struck pile box")
	}
}

func TestDynamicSixDoFPileSleep(t *testing.T) {
	t.Run("100_tight_pit", func(t *testing.T) {
		runPileSimulation(t, 100, 5, 60.0, false)
	})
	t.Run("500_room_pile", func(t *testing.T) {
		runPileSimulation(t, 500, 10, 120.0, false)
	})
}

func TestWillHitDynamicNonWakingZeroAlloc(t *testing.T) {
	rigid_body.SetFps(120)
	defer rigid_body.SetFps(120)

	// Two diagonal OBBs whose inflated world AABBs overlap without any true narrow-phase OBB contact.
	sA := NewSolver(true, 30.0, 0.0, 0.2, int(rigid_body.CombineAverage), int(rigid_body.CombineAverage), 490.0)
	eA := sA.GetEntity()
	eA.Rebuild(-8, -6, 0, 16, 12, 8)
	eA.SetKeepUpright(false)
	eA.SetRotationEuler(0, 0, 0.785)
	_ = sA.Setup(&testObject{solver: sA, ent: eA, box: aabb.NewAABB()})

	sB := NewSolver(true, 30.0, 0.0, 0.2, int(rigid_body.CombineAverage), int(rigid_body.CombineAverage), 490.0)
	eB := sB.GetEntity()
	eB.Rebuild(10.5, 10.5, 0, 16, 12, 8)
	eB.SetKeepUpright(false)
	eB.SetRotationEuler(0, 0, 0.785)
	eB.SetOnGround(true)
	_ = sB.Setup(&testObject{solver: sB, ent: eB, box: aabb.NewAABB()})
	sB.EnterSleep()

	cA := sA.GetCage()
	cB := sB.GetCage()
	fB := newTestFace([3]float64{-8, -6, -4}, [3]float64{8, -6, -4}, [3]float64{0, 6, -4}, 0, 0, 1)
	fA := newTestFace([3]float64{-8, -6, -4}, [3]float64{8, -6, -4}, [3]float64{0, 6, -4}, 0, 0, 1)

	cA.Rebuild(0, 1)
	cA.AddFace(fB, fA)
	if cA.WillHitDynamic(cB) {
		t.Fatalf("expected diagonal near-miss OBBs not to report WillHitDynamic")
	}
	cA.DiscardPendingFaces()
	if !sB.IsSleeping() {
		t.Fatalf("expected non-contacting sleeping neighbor to stay asleep")
	}

	allocs := testing.AllocsPerRun(200, func() {
		cA.AddFace(fB, fA)
		if cA.WillHitDynamic(cB) {
			_ = cA.ShouldWakeDynamic(cB)
		}
		cA.DiscardPendingFaces()
	})
	if allocs != 0 {
		t.Fatalf("expected 0 heap allocs in WillHitDynamic/DiscardPendingFaces, got %v", allocs)
	}
}
