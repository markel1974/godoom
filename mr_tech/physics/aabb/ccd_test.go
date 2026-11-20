package aabb

import (
	"math"
	"testing"
)

func TestCCDThinPlateHighSpeedMinStepTunneling(t *testing.T) {
	// Thin plate A fixed at x in [-0.025, 0.025].
	a := NewBoundingBox()
	a.Rebuild(-0.025, -10, -10, 0.05, 20, 20)
	a.SnapshotRotation()

	// Thin plate B starts right ahead at gap = 0.005 (> ccdGapTol = 1e-3), moving fast through A.
	b := NewBoundingBox()
	b.Rebuild(0.030, -10, -10, 0.05, 20, 20)
	b.SnapshotRotation()

	om := NewOBBManifold()
	toi := om.ComputeSweptAtOffset(a, b, 0, 0, 0, -200, 0, 0)
	if !om.GetHit() {
		t.Fatalf("expected swept hit for thin plate closing gap=0.005 at v=-200, missed (toi=%v, gap=%v)", toi, om.GetGap())
	}
	if toi <= 0 || toi >= 0.01 {
		t.Fatalf("unexpected toi=%v for near-contact fast plate", toi)
	}
}

func TestCCDThinPlateMultiStepApproachWithRotation(t *testing.T) {
	// Stationary thin wall along X: x in [-0.05, 0.05].
	a := NewBoundingBox()
	a.Rebuild(-0.05, -20, -20, 0.1, 40, 40)
	a.SnapshotRotation()

	// Thin fast dart moving -X and spinning: starts at gap ~ 5.0 with rotBound > 0 so iter 0 shrinks gap
	// slightly above ccdGapTol, forcing subsequent step into minStep clamping if clamped to 1e-3.
	b := NewBoundingBox()
	b.Rebuild(5.05, -2, -2, 0.1, 4, 4)
	b.SnapshotRotation()
	b.Rotate(0, 0, 0.5, 1.0) // small rotation adds rotBound > 0 so dtStep stops just short of gap=0

	om := NewOBBManifold()
	toi := om.ComputeSweptAtOffset(a, b, 0, 0, 0, -500, 0, 0)
	if !om.GetHit() {
		t.Fatalf("expected swept hit for fast rotating thin plate approaching from x=5.05, missed (toi=%v, gap=%v)", toi, om.GetGap())
	}
}

func TestCCDLargeRotationThinBarSweep(t *testing.T) {
	// Thin stationary obstacle near the tip of a rotating bar.
	// Bar centered at (0,0,0), dimensions 40 x 0.2 x 2 (half-extents 20, 0.1, 1).
	bar := NewBoundingBox()
	bar.Rebuild(-20, -0.1, -1, 40, 0.2, 2)
	bar.SnapshotRotation()
	// Large angular sweep in one step: 170 degrees around Z so the bar sweeps almost a full semicircle!
	angle := 170.0 * math.Pi / 180.0
	bar.Rotate(0, 0, angle, 1.0)

	// Target pin placed at (0, 18, 0), size 0.4 x 0.4 x 4: swept through around t = 90/170 ~= 0.53!
	// Neither at t=0 (bar along X) nor at t=1 (bar at 170 deg) does it overlap the pin.
	pin := NewBoundingBox()
	pin.Rebuild(-0.2, 17.8, -2, 0.4, 0.4, 4)
	pin.SnapshotRotation()

	om := NewOBBManifold()
	toi := om.ComputeSweptAtOffset(bar, pin, 0, 0, 0, 0, 0, 0)
	if !om.GetHit() {
		t.Fatalf("expected swept hit when thin bar rotates 170 deg through thin pin at (0,18,0), missed (toi=%v, gap=%v)", toi, om.GetGap())
	}
	t.Logf("170-deg sweep hit pin at toi=%.4f (pen=%.4f)", toi, om.GetPenetration())
}

func TestCCDFastLinearGrazingEdge(t *testing.T) {
	// Thin projectile (0.1 x 0.1 x 0.1) flying diagonally at 1000 units/step past thin edge (0.1 x 10 x 10).
	wall := NewBoundingBox()
	wall.Rebuild(-0.05, -5, -5, 0.1, 10, 10)
	wall.SnapshotRotation()

	bullet := NewBoundingBox()
	bullet.Rebuild(10.05, -2.05, -0.05, 0.1, 0.1, 0.1)
	bullet.SnapshotRotation()

	om := NewOBBManifold()
	// Moves from (10.1, -2.0) to (-89.9, +2.0) -> crosses x=0 at t ~= 0.101 where y ~= -1.6 (inside [-5, 5]).
	toi := om.ComputeSweptAtOffset(wall, bullet, 0, 0, 0, -100, 4, 0)
	if !om.GetHit() {
		t.Fatalf("expected swept hit for thin bullet at high speed, missed (toi=%v, gap=%v)", toi, om.GetGap())
	}
	t.Logf("thin bullet hit at toi=%.5f", toi)
}

func TestCCDLargeRotationLongBarHitNearHub(t *testing.T) {
	// Long thin bar (40 x 0.05 x 2) rotating 170 deg hits a thin pin close to its pivot (r=3 vs radius=20).
	// Worst-case corner rotBound is ~6.7x higher than local contact velocity at r=3.
	bar := NewBoundingBox()
	bar.Rebuild(-20, -0.025, -1, 40, 0.05, 2)
	bar.SnapshotRotation()
	bar.Rotate(0, 0, 170.0*math.Pi/180.0, 1.0)

	pin := NewBoundingBox()
	pin.Rebuild(-0.025, 2.975, -1, 0.05, 0.05, 2)
	pin.SnapshotRotation()

	om := NewOBBManifold()
	toi := om.ComputeSweptAtOffset(bar, pin, 0, 0, 0, 0, 0, 0)
	if !om.GetHit() {
		t.Fatalf("expected swept hit for long rotating bar hitting pin near hub r=3, missed (toi=%v, gap=%v)", toi, om.GetGap())
	}
	t.Logf("hub hit toi=%.4f", toi)
}

func TestCCDLargeRotationUltraThinBarTip(t *testing.T) {
	// Ultra-thin bar (40 x 0.02 x 2) rotating 175 deg hits an ultra-thin pin at r=18.
	bar := NewBoundingBox()
	bar.Rebuild(-20, -0.01, -1, 40, 0.02, 2)
	bar.SnapshotRotation()
	bar.Rotate(0, 0, 175.0*math.Pi/180.0, 1.0)

	pin := NewBoundingBox()
	pin.Rebuild(-0.01, 17.99, -1, 0.02, 0.02, 2)
	pin.SnapshotRotation()

	om := NewOBBManifold()
	toi := om.ComputeSweptAtOffset(bar, pin, 0, 0, 0, 0, 0, 0)
	if !om.GetHit() {
		t.Fatalf("expected swept hit for ultra-thin bar rotating 175 deg through 0.02 pin, missed (toi=%v, gap=%v)", toi, om.GetGap())
	}
	t.Logf("ultra-thin tip hit toi=%.4f", toi)
}

func TestCCDMissAndZeroAllocs(t *testing.T) {
	// Grazing near-miss at high speed must NOT false-positive.
	wall := NewBoundingBox()
	wall.Rebuild(-0.025, -5, -5, 0.05, 10, 10)
	wall.SnapshotRotation()

	bullet := NewBoundingBox()
	bullet.Rebuild(5.0, 5.1, -0.05, 0.05, 0.05, 0.1)
	bullet.SnapshotRotation()
	bullet.Rotate(0, 0, 1.2, 1.0)

	om := NewOBBManifold()
	toi := om.ComputeSweptAtOffset(wall, bullet, 0, 0, 0, -200, 10, 0)
	if om.GetHit() {
		t.Fatalf("expected swept miss when passing outside wall bounds, got hit at toi=%v", toi)
	}

	// Swept hit must perform 0 heap allocations.
	hitBullet := NewBoundingBox()
	hitBullet.Rebuild(5.0, -0.025, -0.05, 0.05, 0.05, 0.1)
	hitBullet.SnapshotRotation()
	hitBullet.Rotate(0, 0, 2.5, 1.0)

	allocs := testing.AllocsPerRun(100, func() {
		om.ComputeSweptAtOffset(wall, hitBullet, 0, 0, 0, -200, 0, 0)
	})
	if allocs != 0 {
		t.Fatalf("expected 0 heap allocs during swept CCD, got %v", allocs)
	}
}
