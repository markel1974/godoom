package collisions

import (
	"math"
	"testing"
	"time"

	"github.com/markel1974/godoom/mr_tech/physics/aabb"
	"github.com/markel1974/godoom/mr_tech/physics/rigid_body"
)

type solverParentFace struct {
	*testFace
	parent *Solver
}

func (f *solverParentFace) GetParent() *Solver { return f.parent }

func noopCollision(IObject, float64) {}

func newConfiguredSolver(obj *testObject, location IObject, maxStep float64, canSleep bool, onElastic, onInelastic CollisionT) *Solver {
	s := obj.solver
	s.SetLocation(location)
	s.SetMaxStep(maxStep)
	s.SetCanSleep(canSleep)
	s.SetElasticCollision(onElastic)
	s.SetInelasticCollision(onInelastic)
	return s
}

func TestSolverResolvePartnerRespectsPartnerSolverIndex(t *testing.T) {
	rigid_body.SetFps(60)
	defer rigid_body.SetFps(120)

	// Partner with lower entity ID (< self ID) mimics sleeping/inactive body not participating in the solver tick.
	objA := newTestUprightObject(0, 0, 20, 20, 20, 40)
	objB := newTestUprightObject(0, 0, 50, 20, 20, 40)
	if objB.GetEntity().GetId() <= objA.GetEntity().GetId() {
		t.Fatalf("expected second entity id (%d) > first (%d)", objB.GetEntity().GetId(), objA.GetEntity().GetId())
	}

	solverA := newConfiguredSolver(objA, nil, 0, true, noopCollision, noopCollision)
	solverB := newConfiguredSolver(objB, nil, 0, true, noopCollision, noopCollision)
	cageA := solverA.GetCage()
	cageB := solverB.GetCage()

	solverA.StagePrepare(1)
	solverB.StagePrepare(1)

	// Record mutual contact on opposite buckets, with cageA not active in this tick (solverIndex == -1).
	eA := cageA.buckets[BucketCeiling].Add(BucketCeiling, cageA, cageB, nil, -0.1, 0.1, 0, 0, -1, 0, 0, 30, 60, ImpactElastic)
	eB := cageB.buckets[BucketFloor].Add(BucketFloor, cageB, cageA, nil, -0.1, 0.1, 0, 0, 1, 0, 0, 30, 40, ImpactElastic)
	if eA == nil || eB == nil {
		t.Fatalf("expected symmetric slots")
	}

	// Because cageA is inactive in this tick (solverIndex == -1), solverB must resolve the contact instead of skipping it.
	solverA.SetSolverIndex(-1)
	solverB.SetSolverIndex(0)
	if partner := solverB.resolveSlotPartner(eB); partner != objA.GetEntity() {
		t.Fatalf("expected inactive lower-ID partner to be resolved by solverB, got %v", partner)
	}

	// When both bodies are active in the tick, only canonical owner (lower ID: solverA) resolves the pair.
	solverA.SetSolverIndex(0)
	solverB.SetSolverIndex(1)
	if partner := solverB.resolveSlotPartner(eB); partner != nil {
		t.Fatalf("expected solverB to defer pair to active lower-ID owner solverA, got %v", partner)
	}
	if partner := solverA.resolveSlotPartner(eA); partner != objB.GetEntity() {
		t.Fatalf("expected lower-ID owner solverA to resolve pair, got %v", partner)
	}
}

func TestSolverLocationUpdateAndCallbacks(t *testing.T) {
	rigid_body.SetFps(60)
	defer rigid_body.SetFps(120)

	initLoc := newTestUprightObject(0, 0, 0, 10, 10, 10)
	newLoc := newTestUprightObject(0, 0, 0, 10, 10, 10)
	s := NewSolver(true, 80.0, 0.0, 0.2, int(rigid_body.CombineAverage), int(rigid_body.CombineAverage), 490.0)
	ent := s.GetEntity()
	ent.Rebuild(-10, -10, 0, 20, 20, 40)
	ent.SetKeepUpright(true)
	ent.LockUpright()
	ent.SetOnGround(false)
	ent.SetV(0, 0, -500)
	obj := &testObject{solver: s, ent: ent, box: aabb.NewAABB()}
	_ = s.Setup(obj)
	other := newTestUprightObject(0, 0, 50, 20, 20, 40)

	rCage := newTestCage(13, other)
	elasticCalls := 0
	inelasticCalls := 0
	var inelasticEnergy float64
	solver := newConfiguredSolver(obj, initLoc, 10.0, false,
		func(IObject, float64) { elasticCalls++ },
		func(_ IObject, energy float64) {
			inelasticCalls++
			inelasticEnergy = energy
		},
	)
	cage := solver.GetCage()
	if solver.GetLocation() != initLoc {
		t.Fatalf("expected initial location %v, got %v", initLoc, solver.GetLocation())
	}

	solver.StagePrepare(1)
	obj.GetEntity().SnapshotPreSolve()
	rCage.Rebuild(0, 1)

	floorFace := &solverParentFace{
		testFace: newTestFace([3]float64{-20, -20, 1}, [3]float64{20, -20, 1}, [3]float64{0, 20, 1}, 0, 0, 1),
		parent:   newLoc.solver,
	}
	cage.AddFace(floorFace, nil)
	cage.Commit(nil)
	if !cage.addToBucket(BucketCeiling, rCage, nil, -0.1, 0.1, 0, 0, -1, 0, 0, 30, 60, ImpactElastic) {
		t.Fatalf("expected elastic slot to be recorded")
	}

	vz := obj.GetEntity().GetVz()
	expectedEnergy := 0.5 * 80.0 * (vz / 50.0) * (vz / 50.0)
	solver.StageApply()
	if solver.GetLocation() != newLoc.solver.GetObject() {
		t.Fatalf("expected updated location %v after StageApply, got %v", newLoc.solver.GetObject(), solver.GetLocation())
	}
	if elasticCalls != 1 || inelasticCalls != 1 {
		t.Fatalf("expected 1 elastic and 1 inelastic callback, got elastic=%d inelastic=%d", elasticCalls, inelasticCalls)
	}
	if expectedEnergy <= 0 || math.Abs(inelasticEnergy-expectedEnergy) > 1e-6 {
		t.Fatalf("expected %v J inelastic floor impact energy, got %v", expectedEnergy, inelasticEnergy)
	}
}

func TestSolverInelasticMultiPointManifoldUsesPeakEnergy(t *testing.T) {
	rigid_body.SetFps(60)
	defer rigid_body.SetFps(120)

	floorVol := newTestUprightObject(0, 0, 0, 100, 100, 10)
	// 6DoF box centered at (0,0,19) resting/rotating onto floor at z=0.
	obj := newTestUprightObject(0, 0, 19, 20, 20, 40)
	ent := obj.GetEntity()
	ent.SetKeepUpright(false)
	ent.SetAngularVelocity(0, -10.0, 0) // Corner at x=-10 moves downward: vpz = -wy * rx = -(-10)*(-10) = -100.
	ent.SnapshotPreSolve()

	calls := 0
	var reportedEnergy float64
	solver := newConfiguredSolver(obj, floorVol, 0, true, noopCollision, func(_ IObject, energy float64) {
		calls++
		reportedEnergy = energy
	})
	cage := solver.GetCage()
	solver.StagePrepare(1)

	floorFace := &solverParentFace{
		testFace: newTestFace([3]float64{-50, -50, 0}, [3]float64{50, -50, 0}, [3]float64{0, 50, 0}, 0, 0, 1),
		parent:   floorVol.solver,
	}
	// Two consecutive manifold slots on the same static floor parent: weaker center point first, peak edge point second.
	if !cage.addToBucket(BucketFloor, nil, floorFace, -1, 1, 0, 0, 1, 0, 0, 0, 0, ImpactInelastic) {
		t.Fatalf("expected first inelastic manifold slot")
	}
	if !cage.addToBucket(BucketFloor, nil, floorFace, -1, 1, 0, 0, 1, -10, 0, 0, 0, ImpactInelastic) {
		t.Fatalf("expected second inelastic manifold slot")
	}

	solver.StageApply()
	expectedPeak := ent.ComputeStaticNormalKineticImpactAt(0, 0, 1, -10, 0, 0)
	if calls != 1 {
		t.Fatalf("expected 1 coalesced inelastic callback across contiguous manifold points, got %d", calls)
	}
	if expectedPeak <= 0 || math.Abs(reportedEnergy-expectedPeak) > 1e-6 {
		t.Fatalf("expected peak manifold impact energy %v, got %v", expectedPeak, reportedEnergy)
	}
}

func TestSolverPipelineZeroAllocs(t *testing.T) {
	rigid_body.SetFps(60)
	defer rigid_body.SetFps(120)

	loc := newTestUprightObject(0, 0, 0, 10, 10, 10)
	obj := newTestUprightObject(0, 0, 20, 20, 20, 40)
	solver := newConfiguredSolver(obj, loc, 10.0, true, noopCollision, noopCollision)
	cage := solver.GetCage()
	floorFace := &solverParentFace{
		testFace: newTestFace([3]float64{-20, -20, 0}, [3]float64{20, -20, 0}, [3]float64{0, 20, 0}, 0, 0, 1),
		parent:   loc.solver,
	}

	session := uint64(1)
	allocs := testing.AllocsPerRun(50, func() {
		session++
		solver.StagePrepare(session)
		cage.AddFace(floorFace, nil)
		cage.Commit(nil)
		solver.StageWarmStart()
		solver.StageResolve()
		solver.StageApply()
		solver.StageResolvePosition()
		_ = solver.UpdateSleepState()
	})
	if allocs != 0 {
		t.Fatalf("expected 0 allocations per solver tick, got %v", allocs)
	}
}

func TestPenetrableCycleAndCluster(t *testing.T) {
	// Set up 3 bodies in a cyclic contact configuration along BucketWallEast
	objA := newTestUprightObject(0, 0, 20, 20, 20, 40)
	objB := newTestUprightObject(30, 0, 20, 20, 20, 40)
	objC := newTestUprightObject(60, 0, 20, 20, 20, 40)

	solverA := newConfiguredSolver(objA, nil, 0, true, noopCollision, noopCollision)
	solverB := newConfiguredSolver(objB, nil, 0, true, noopCollision, noopCollision)
	solverC := newConfiguredSolver(objC, nil, 0, true, noopCollision, noopCollision)

	cageA := solverA.GetCage()
	cageB := solverB.GetCage()
	cageC := solverC.GetCage()

	solverA.StagePrepare(1)
	solverB.StagePrepare(1)
	solverC.StagePrepare(1)

	// A -> B (with 4 manifold points to test deduplication)
	for p := 0; p < 4; p++ {
		cageA.buckets[BucketWallEast].Add(BucketWallEast, cageA, cageB, nil, -0.1, 0.1, -1, 0, 0, 20, float64(p*5), 20, 40, ImpactElastic)
	}
	// B -> C (with 4 manifold points)
	for p := 0; p < 4; p++ {
		cageB.buckets[BucketWallEast].Add(BucketWallEast, cageB, cageC, nil, -0.1, 0.1, -1, 0, 0, 50, float64(p*5), 20, 40, ImpactElastic)
	}
	// C -> A (cycle back to A!)
	for p := 0; p < 4; p++ {
		cageC.buckets[BucketWallEast].Add(BucketWallEast, cageC, cageA, nil, -0.1, 0.1, -1, 0, 0, 80, float64(p*5), 20, 40, ImpactElastic)
	}

	entryA := cageA.buckets[BucketWallEast].currPool[0]

	// Cycle detection must immediately return without hanging or blowing up stack/CPU
	done := make(chan bool, 1)
	var res bool
	go func() {
		res = entryA.Penetrable(0)
		done <- true
	}()

	select {
	case <-done:
		// Successfully terminated in time. Since it's a closed loop mutual cycle, it cannot yield freely.
		if res != false {
			t.Fatalf("expected cyclic push chain to not be freely penetrable, got %v", res)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("Penetrable timed out / hung due to cycle in graph!")
	}
}
