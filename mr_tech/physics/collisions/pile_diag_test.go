package collisions

import (
	"math"
	"testing"

	"github.com/markel1974/godoom/mr_tech/physics/aabb"
	"github.com/markel1974/godoom/mr_tech/physics/rigid_body"
)

func runPileSimulation(t *testing.T, fps uint64, nBoxes int, cols int, wallHalf float64, wakeOnBroadphase bool) {
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
	maxSteps := int(fps) * 10
	if fps == 120 {
		maxSteps = int(fps) * 6
	}
	for step := 0; step < maxSteps; step++ {
		coord.Reset()
		for _, s := range solvers {
			coord.Stage(s)
		}
		coord.Step()
		if fps == 60 && nBoxes == 100 && step >= 596 && step <= 599 {
			for _, idxT := range []int{71, 97} {
				s := solvers[idxT]
				ent := s.GetEntity()
				px, py, pz := s.GetCage().GetPushOffset()
				nx, ny, nz := s.GetCage().GetNetDisplacement()
				vx, vy, vz := ent.GetVelocity()
				_, _, cz := ent.GetCenter()
				t.Logf("  step %d #%d(id=%d) sleep=%v gnd=%v tilt=%v z=%.3f v=(%.2f,%.2f,%.2f) push=(%.3f,%.3f,%.3f) net=(%.3f,%.3f,%.3f) slots=%d",
					step, idxT, ent.GetId(), s.IsSleeping(), ent.IsOnGround(), ent.IsTilted(), cz, vx, vy, vz, px, py, pz, nx, ny, nz, s.GetCage().GetSlotsLen())
			}
		}
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
		shown := 0
		for idx, s := range solvers {
			if s.IsSleeping() {
				continue
			}
			ent := s.GetEntity()
			vx, vy, vz := ent.GetVelocity()
			wx, wy, wz := ent.GetAngularVelocity()
			nx, ny, nz := s.GetCage().GetNetDisplacement()
			netSq := nx*nx + ny*ny + nz*nz
			wSq := wx*wx + wy*wy + wz*wz
			spdSq := vx*vx + vy*vy + vz*vz
			if !ent.IsOnGround() || spdSq > 16.0 || wSq > 0.28*0.28 || netSq > 0.0016 {
				px, py, pz := s.GetCage().GetPushOffset()
				_, _, dz := ent.GetFrameDisplacement()
				t.Logf("  notReady[%3d]: id=%d onGnd=%v v=(%.2f,%.2f,%.2f) w=(%.3f,%.3f,%.3f) dZ=%.4f net=(%.4f,%.4f,%.4f) push=(%.4f,%.4f,%.4f) slots=%d",
					idx, ent.GetId(), ent.IsOnGround(), vx, vy, vz, wx, wy, wz, dz, nx, ny, nz, px, py, pz, s.GetCage().GetSlotsLen())
				for si := 0; si < s.GetCage().GetSlotsLen(); si++ {
					sl := s.GetCage().GetSlot(si)
					snx, sny, snz := sl.GetNormal()
					rId := uint64(0)
					if rc := sl.GetRCage(); rc != nil {
						rId = rc.GetEntity().GetId()
					}
					t.Logf("    slot %d rId=%d n=(%.3f,%.3f,%.3f) pen=%.4f warmN=%.2f", si, rId, snx, sny, snz, sl.GetPenetration(), sl.GetSolverState().GetNormal())
				}
				shown++
			}
			if shown >= 10 {
				break
			}
		}
		if shown == 0 {
			for idx, s := range solvers {
				if s.IsSleeping() {
					continue
				}
				ent := s.GetEntity()
				nx, ny, nz := s.GetCage().GetNetDisplacement()
				maxPen, maxNZ := 0.0, 0.0
				for si := 0; si < s.GetCage().GetSlotsLen(); si++ {
					sl := s.GetCage().GetSlot(si)
					if sl.GetPenetration() > maxPen {
						maxPen = sl.GetPenetration()
						_, _, maxNZ = sl.GetNormal()
					}
				}
				t.Logf("  calmAwake[%3d]: dt=%.5f g=%.1f net=(%.4f,%.4f,%.4f) sm=%v maxPen=%.4f(nZ=%.2f)",
					idx, ent.GetDt(), ent.GetGForce(), nx, ny, nz, ent.HasSupportMotion(nx*nx+ny*ny+nz*nz, false), maxPen, maxNZ)
				shown++
				if shown >= 8 {
					break
				}
			}
		}
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
	t.Run("100_tight_pit_120hz", func(t *testing.T) {
		runPileSimulation(t, 120, 100, 5, 60.0, false)
	})
	t.Run("500_room_pile_120hz", func(t *testing.T) {
		runPileSimulation(t, 120, 500, 10, 120.0, false)
	})
	t.Run("100_tight_pit_60hz", func(t *testing.T) {
		runPileSimulation(t, 60, 100, 5, 60.0, false)
	})
	t.Run("500_room_pile_60hz", func(t *testing.T) {
		runPileSimulation(t, 60, 500, 10, 120.0, false)
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
