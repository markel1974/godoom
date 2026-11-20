package aabb

import (
	"math"
	"testing"
)

func TestContactPointsFinalizePreservesDistinctPoints(t *testing.T) {
	cp := NewContactPoints()
	cp.SetIndex(0, 1, 2, 3, 0.1)
	cp.SetIndex(1, 4, 5, 6, 0.2)
	cp.SetIndex(2, 7, 8, 9, 0.9) // deepest at index > 0
	cp.SetIndex(3, 10, 11, 12, 0.4)

	cp.Finalize(4)
	if cp.GetCount() != 4 {
		t.Fatalf("expected count 4, got %d", cp.GetCount())
	}

	x0, y0, z0, p0 := cp.GetIndex(0)
	if p0 != 0.9 || x0 != 7 || y0 != 8 || z0 != 9 {
		t.Fatalf("expected deepest point (7,8,9,0.9) at index 0, got (%v,%v,%v,%v)", x0, y0, z0, p0)
	}

	want := [4][4]float64{
		{7, 8, 9, 0.9},
		{1, 2, 3, 0.1},
		{4, 5, 6, 0.2},
		{10, 11, 12, 0.4},
	}
	for i := 0; i < 4; i++ {
		x, y, z, p := cp.GetIndex(i)
		if x != want[i][0] || y != want[i][1] || z != want[i][2] || p != want[i][3] {
			t.Fatalf("point %d = (%v,%v,%v,%v), want (%v,%v,%v,%v)", i, x, y, z, p, want[i][0], want[i][1], want[i][2], want[i][3])
		}
	}
}

func TestOBBManifoldPopulatesPointsFromFaceAndEightCandidates(t *testing.T) {
	// Face-face overlap with <= 4 contact points must populate om.Points (not stale data).
	a := NewBoundingBox()
	a.Rebuild(0, 0, 0, 20, 20, 20)
	b := NewBoundingBox()
	b.Rebuild(0, 0, 19, 20, 20, 20)

	om := NewOBBManifold()
	// Pre-seed Points[0] with sentinel coordinates to ensure stale data is overwritten.
	om.points.SetIndex(0, -999, -999, -999, 99)
	om.ComputeAtOffset(a, b, 0, 0, 0)

	if !om.hit {
		t.Fatalf("expected OBB hit")
	}
	if om.points.GetCount() != 4 {
		t.Fatalf("expected 4 contact points, got %d", om.points.GetCount())
	}
	for i := 0; i < om.points.GetCount(); i++ {
		x, y, z, pen := om.points.GetIndex(i)
		if x == -999 || y == -999 || z == -999 {
			t.Fatalf("point %d retained stale sentinel data (%v,%v,%v)", i, x, y, z)
		}
		if math.Abs(pen-1.0) > 1e-6 || math.Abs(z-19.5) > 1e-6 {
			t.Fatalf("unexpected point %d (%v,%v,%v pen=%v)", i, x, y, z, pen)
		}
	}

	// Rotated square vs axis-aligned square produces 8 intersection vertices (> MaxManifoldPoints);
	// candidates must hold all 8 before Reduce trims down to 4 distinct spread points.
	rotBox := NewBoundingBox()
	rotBox.Rebuild(0, 0, 19, 20, 20, 20)
	rotBox.Rotate(0, 0, math.Pi/4.0, 1.0)
	om.ComputeAtOffset(a, rotBox, 0, 0, 0)
	if !om.hit || om.points.GetCount() != MaxManifoldPoints {
		t.Fatalf("expected hit with %d reduced points, got hit=%v count=%d", MaxManifoldPoints, om.hit, om.points.GetCount())
	}
	for i := 0; i < MaxManifoldPoints; i++ {
		xi, yi, zi, _ := om.points.GetIndex(i)
		for j := i + 1; j < MaxManifoldPoints; j++ {
			xj, yj, zj, _ := om.points.GetIndex(j)
			if PointDistSq(xi, yi, zi, xj, yj, zj) < 1e-4 {
				t.Fatalf("duplicate reduced contact point %d and %d at (%v,%v,%v)", i, j, xi, yi, zi)
			}
		}
	}
}

func TestOBBManifoldContactPointsZeroAllocs(t *testing.T) {
	a := NewBoundingBox()
	a.Rebuild(0, 0, 0, 20, 20, 20)
	b := NewBoundingBox()
	b.Rebuild(0, 0, 19, 20, 20, 20)
	b.Rotate(0, 0, math.Pi/4.0, 1.0)
	om := NewOBBManifold()

	allocs := testing.AllocsPerRun(100, func() {
		om.ComputeAtOffset(a, b, 0, 0, 0)
	})
	if allocs != 0 {
		t.Fatalf("expected 0 heap allocs per OBBManifold compute, got %v", allocs)
	}
}
