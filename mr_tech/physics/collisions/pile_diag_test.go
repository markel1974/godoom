package collisions

import (
	"math"
	"testing"
	"time"

	"github.com/markel1974/godoom/mr_tech/physics/aabb"
	"github.com/markel1974/godoom/mr_tech/physics/rigid_body"
)

func runPileSimulation(t *testing.T, nBoxes int, wakeOnBroadphase bool) {
	t.Helper()
	rigid_body.SetFps(120)
	defer rigid_body.SetFps(120)

	floorObj := newTestUprightObject(0, 0, -25, 1000, 1000, 50)
	staticFaces := []IFace{
		&solverParentFace{testFace: newTestFace([3]float64{-200, -200, 0}, [3]float64{200, -200, 0}, [3]float64{200, 200, 0}, 0, 0, 1), parent: floorObj.solver},
		&solverParentFace{testFace: newTestFace([3]float64{-200, -200, 0}, [3]float64{200, 200, 0}, [3]float64{-200, 200, 0}, 0, 0, 1), parent: floorObj.solver},
		// Confining walls at x = ±60, y = ±60 to force a realistic multi-layer heap/mound.
		&solverParentFace{testFace: newTestFace([3]float64{60, -200, 0}, [3]float64{60, -200, 300}, [3]float64{60, 200, 300}, -1, 0, 0), parent: floorObj.solver},
		&solverParentFace{testFace: newTestFace([3]float64{60, -200, 0}, [3]float64{60, 200, 300}, [3]float64{60, 200, 0}, -1, 0, 0), parent: floorObj.solver},
		&solverParentFace{testFace: newTestFace([3]float64{-60, -200, 0}, [3]float64{-60, 200, 300}, [3]float64{-60, -200, 300}, 1, 0, 0), parent: floorObj.solver},
		&solverParentFace{testFace: newTestFace([3]float64{-60, -200, 0}, [3]float64{-60, 200, 0}, [3]float64{-60, 200, 300}, 1, 0, 0), parent: floorObj.solver},
		&solverParentFace{testFace: newTestFace([3]float64{-200, 60, 0}, [3]float64{200, 60, 300}, [3]float64{-200, 60, 300}, 0, -1, 0), parent: floorObj.solver},
		&solverParentFace{testFace: newTestFace([3]float64{-200, 60, 0}, [3]float64{200, 60, 0}, [3]float64{200, 60, 300}, 0, -1, 0), parent: floorObj.solver},
		&solverParentFace{testFace: newTestFace([3]float64{-200, -60, 0}, [3]float64{-200, -60, 300}, [3]float64{200, -60, 300}, 0, 1, 0), parent: floorObj.solver},
		&solverParentFace{testFace: newTestFace([3]float64{-200, -60, 0}, [3]float64{200, -60, 300}, [3]float64{200, -60, 0}, 0, 1, 0), parent: floorObj.solver},
	}

	solvers := make([]*Solver, nBoxes)
	faces := make([]IFace, nBoxes)
	for i := range nBoxes {
		layer := i / 25
		slot := i % 25
		row := slot / 5
		col := slot % 5
		x := -42.0 + float64(col)*21.0 + math.Sin(float64(i)*1.7)*2.5
		y := -42.0 + float64(row)*21.0 + math.Cos(float64(i)*2.3)*2.5
		z := 6.0 + float64(layer)*11.0
		// Doom 3 medikit parameters: mass=30, restitution=0, friction=0.2, gForce=490, maxStep=0
		s := NewSolver(true, 15.0, 0.0, 0.2, int(rigid_body.CombineAverage), int(rigid_body.CombineAverage), 490.0)
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
			if wakeOnBroadphase {
				// Reproduce mr_tech/model/things.go: TranslateCage + Translate AABB overlap wakes before Commit.
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
				continue
			}
			preSlots := cage.GetSlotsLen()
			preRFloor := rCage.HasFloor()
			cage.AddFace(faces[j], faces[0])
			cage.Commit(rCage)
			if wasSleep && (cage.GetSlotsLen() > preSlots || (!preRFloor && rCage.HasFloor())) {
				wake(solvers[j])
				rCage.Seen(cage)
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
	var stepTimeTotal time.Duration
	var stepCountWindow int
	for step := 0; step < 960; step++ {
		t0 := time.Now()
		coord.Reset()
		for _, s := range solvers {
			coord.Stage(s)
		}
		coord.Step()
		stepTimeTotal += time.Since(t0)
		stepCountWindow++
		if step >= 954 && step <= 959 {
			s := solvers[65]
			ent := s.GetEntity()
			cx, cy, cz := ent.GetCenter()
			vx, vy, vz := ent.GetVelocity()
			wx, wy, wz := ent.GetAngularVelocity()
			nx, ny, nz := s.GetCage().GetNetDisplacement()
			px, py, pz := s.GetCage().GetPushOffset()
			t.Logf("trace[65] step=%d pos=(%.3f,%.3f,%.3f) v=(%.3f,%.3f,%.3f) w=(%.3f,%.3f,%.3f) net=(%.5f,%.5f,%.5f) push=(%.5f,%.5f,%.5f) floor=%v slots=%d",
				step, cx, cy, cz, vx, vy, vz, wx, wy, wz, nx, ny, nz, px, py, pz, s.GetCage().HasFloor(), s.GetCage().GetSlotsLen())
			for si := 0; si < s.GetCage().GetSlotsLen(); si++ {
				sl := s.GetCage().GetSlot(si)
				snx, sny, snz := sl.GetNormal()
				rId := uint64(0)
				if rc := sl.GetRCage(); rc != nil {
					rId = rc.GetEntity().GetId()
				}
				t.Logf("    slot %d: bucket=%s mode=%d rId=%d n=(%.3f,%.3f,%.3f) pen=%.4f warmN=%.3f",
					si, sl.GetBucket(), sl.GetImpactMode(), rId, snx, sny, snz, sl.GetPenetration(), sl.GetSolverState().GetNormal())
			}
		}
		if step == 120 || step == 240 || step == 360 || step == 480 || step == 600 || step == 720 || step == 840 || step == 959 {
			awake := 0
			calmV := 0
			noGround := 0
			hiNet := 0
			hiW := 0
			maxSpdSq := 0.0
			maxIdx := -1
			for idx, s := range solvers {
				if s.IsSleeping() {
					continue
				}
				awake++
				ent := s.GetEntity()
				vx, vy, vz := ent.GetVelocity()
				wx, wy, wz := ent.GetAngularVelocity()
				spdSq := vx*vx + vy*vy + vz*vz
				if spdSq > maxSpdSq {
					maxSpdSq = spdSq
					maxIdx = idx
				}
				if spdSq <= 16.0 {
					calmV++
				}
				if !ent.IsOnGround() {
					noGround++
				}
				nx, ny, nz := s.GetCage().GetNetDisplacement()
				if nx*nx+ny*ny+nz*nz > 0.0016 {
					hiNet++
				}
				if wx*wx+wy*wy+wz*wz > 0.16*0.16 {
					hiW++
				}
			}
			avgMs := float64(stepTimeTotal.Microseconds()) / float64(stepCountWindow) / 1000.0
			stepTimeTotal = 0
			stepCountWindow = 0
			t.Logf("step %3d: running=%3d awake=%3d calmV=%3d noGnd=%3d hiNet=%3d hiW=%3d maxSpd=%.2f(#%d) avgStep=%.3fms",
				step, coord.RunningLen(), awake, calmV, noGround, hiNet, hiW, math.Sqrt(maxSpdSq), maxIdx, avgMs)
			if step == 959 && maxIdx >= 0 {
				for _, idx := range []int{3, 8, maxIdx} {
					s := solvers[idx]
					ent := s.GetEntity()
					cx, cy, cz := ent.GetCenter()
					vx, vy, vz := ent.GetVelocity()
					wx, wy, wz := ent.GetAngularVelocity()
					nx, ny, nz := s.GetCage().GetNetDisplacement()
					px, py, pz := s.GetCage().GetPushOffset()
					t.Logf("  box[%3d]: pos=(%.2f,%.2f,%.2f) v=(%.4f,%.4f,%.4f) w=(%.4f,%.4f,%.4f) net=(%.5f,%.5f,%.5f) push=(%.5f,%.5f,%.5f) floor=%v onGnd=%v tilted=%v slots=%d",
						idx, cx, cy, cz, vx, vy, vz, wx, wy, wz, nx, ny, nz, px, py, pz, s.GetCage().HasFloor(), ent.IsOnGround(), ent.IsTilted(), s.GetCage().GetSlotsLen())
					for si := 0; si < s.GetCage().GetSlotsLen(); si++ {
						sl := s.GetCage().GetSlot(si)
						snx, sny, snz := sl.GetNormal()
						acc := sl.GetSolverState()
						rId := uint64(0)
						if rc := sl.GetRCage(); rc != nil {
							rId = rc.GetEntity().GetId()
						}
						t.Logf("    slot %d: bucket=%s mode=%d rId=%d n=(%.3f,%.3f,%.3f) pen=%.4f warmN=%.3f",
							si, sl.GetBucket(), sl.GetImpactMode(), rId, snx, sny, snz, sl.GetPenetration(), acc.GetNormal())
					}
				}
			}
		}
	}
}

func TestDiagDynamicPileSleep(t *testing.T) {
	t.Run("100_obb_hit_wake", func(t *testing.T) {
		runPileSimulation(t, 100, false)
	})
}
