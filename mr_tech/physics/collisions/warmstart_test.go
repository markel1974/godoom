package collisions

import (
	"math"
	"testing"
)

func TestBucketWarmStartSkipsInitialTransientAndRebound(t *testing.T) {
	cage := newTestCage(1, newTestUprightObject(0, 0, 20, 20, 20, 40))
	cage.Rebuild(0, 1)
	b := cage.buckets[BucketFloor]

	// Frame 0: initial crash contact accumulates large normal impulse without prior warm match.
	b.Rebuild()
	e0 := b.Add(BucketFloor, cage, nil, nil, 0.0, 0.1, 0, 0, 1, 0, 0, 0, 0, ImpactElastic)
	if e0 == nil {
		t.Fatalf("expected frame 0 entry")
	}
	accN, accTx, accTy, accTz, restT, hasRest := e0.GetSolverState()
	*accN = 15000.0
	*accTx, *accTy, *accTz = 100.0, 0.0, 0.0
	*restT = 0.0
	*hasRest = true

	// Frame 1: persistent contact becomes warm-eligible for frame 2+, but does not seed the frame 0 crash impulse.
	b.Rebuild()
	e1 := b.Add(BucketFloor, cage, nil, nil, 0.0, 0.01, 0, 0, 1, 0, 0, 0, 0, ImpactElastic)
	if e1 == nil {
		t.Fatalf("expected frame 1 entry")
	}
	warmN1, _, _, _ := e1.GetWarmImpulse()
	if warmN1 != 0.0 {
		t.Fatalf("expected 0 warm-start impulse immediately after initial crash frame, got %v", warmN1)
	}
	accN, accTx, accTy, accTz, restT, hasRest = e1.GetSolverState()
	*accN = 122.0
	*accTx, *accTy, *accTz = 10.0, 0.0, 0.0
	*restT = 0.0
	*hasRest = true

	// Frame 2: resting contact across consecutive frames seeds 0.80 * steady load.
	b.Rebuild()
	e2 := b.Add(BucketFloor, cage, nil, nil, 0.0, 0.01, 0, 0, 1, 0, 0, 0, 0, ImpactElastic)
	if e2 == nil {
		t.Fatalf("expected frame 2 entry")
	}
	warmN2, warmTx2, _, _ := e2.GetWarmImpulse()
	if math.Abs(warmN2-122.0*warmStartFactor) > 1e-6 || math.Abs(warmTx2-10.0*warmStartFactor) > 1e-6 {
		t.Fatalf("expected resting warm-start (%.4f, %.4f), got (%.4f, %.4f)", 122.0*warmStartFactor, 10.0*warmStartFactor, warmN2, warmTx2)
	}

	// Simulate an elastic bounce on frame 2: frame 3 must reset warm start.
	_, _, _, _, restT, _ = e2.GetSolverState()
	*restT = 50.0

	b.Rebuild()
	e3 := b.Add(BucketFloor, cage, nil, nil, 0.0, 0.01, 0, 0, 1, 0, 0, 0, 0, ImpactElastic)
	if e3 == nil {
		t.Fatalf("expected frame 3 entry")
	}
	warmN3, _, _, _ := e3.GetWarmImpulse()
	if warmN3 != 0.0 {
		t.Fatalf("expected 0 warm-start impulse after rebounding frame, got %v", warmN3)
	}
}
