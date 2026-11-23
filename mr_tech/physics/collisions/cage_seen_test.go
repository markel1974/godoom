package collisions

import "testing"

func TestCageSeenAvoidsQuadraticPreallocAndGrowsOnDemand(t *testing.T) {
	objLow := newTestUprightObject(0, 0, 20, 20, 20, 40)
	objHigh := newTestUprightObject(50, 0, 20, 20, 20, 40)

	cLow := objLow.solver.GetCage()
	cHigh := NewCage(25000)
	if err := cHigh.Setup(objHigh.solver); err != nil {
		t.Fatalf("cHigh.Setup: %v", err)
	}

	if len(cHigh.seen) > 128 {
		t.Fatalf("expected bounded initial seen slice for high cage id, got len=%d", len(cHigh.seen))
	}

	// Simulate an active physics session; HasSeen on an unseen high id must not grow seen.
	cLow.Rebuild(0, 1)
	cHigh.Rebuild(0, 1)

	prevLen := len(cLow.seen)
	if cLow.HasSeen(cHigh) {
		t.Fatalf("expected unseen high cage to return false in session 1")
	}
	if len(cLow.seen) != prevLen {
		t.Fatalf("expected HasSeen not to reallocate seen slice, grew %d -> %d", prevLen, len(cLow.seen))
	}

	// Seen must grow on demand and accurately track the current session only.
	cLow.Seen(cHigh)
	if !cLow.HasSeen(cHigh) {
		t.Fatalf("expected HasSeen to return true after Seen in session 1")
	}

	cLow.Rebuild(0, 2)
	if cLow.HasSeen(cHigh) {
		t.Fatalf("expected HasSeen to return false in next session 2")
	}
}
