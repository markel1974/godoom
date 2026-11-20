package collisions

import (
	"math"
	"testing"

	"github.com/markel1974/godoom/mr_tech/physics/aabb"
	"github.com/markel1974/godoom/mr_tech/physics/rigid_body"
)

func TestHingeConstraintRevoluteLimitsAndZeroAllocs(t *testing.T) {
	rigid_body.SetFps(60)
	defer rigid_body.SetFps(120)

	// Solver A (base/parent) at origin (0, 0, 10), Solver B (lever/child) at (0, 20, 10).
	// Joined at (0, 10, 10) with hinge axis along Z (0, 0, 1).
	sA, _ := newFreeSixDoFSolver(100.0, 0, 0, 10, 20, 20, 20, 0, false)
	sB, _ := newFreeSixDoFSolver(10.0, 0, 20, 10, 20, 20, 20, 0, false)
	sA.SetSolverIndex(0)
	sB.SetSolverIndex(1)
	sA.GetEntity().WakeUp()
	sB.GetEntity().WakeUp()

	minAngle := -math.Pi * 0.25 // -45 deg
	maxAngle := math.Pi * 0.25  // +45 deg
	hinge := NewHingeJoint(sA, sB, 0, 10, 0, 0, -10, 0, 0, 0, 1, minAngle, maxAngle, 2.0)

	// Zero heap allocations in ResolveVelocity / ResolvePosition
	allocs := testing.AllocsPerRun(100, func() {
		hinge.ResolveVelocity()
		hinge.ResolvePosition()
	})
	if allocs != 0 {
		t.Fatalf("expected 0 heap allocs in hinge PGS solve, got %v", allocs)
	}

	// Apply off-axis angular velocity to B (pitch around X and roll around Y)
	sB.GetEntity().SetAngularVelocity(10.0, 10.0, 5.0)

	for iter := 0; iter < 10; iter++ {
		hinge.ResolveVelocity()
	}

	bwx, bwy, bwz := sB.GetEntity().GetAngularVelocity()

	if math.Abs(bwx) > 0.5 || math.Abs(bwy) > 0.5 {
		t.Fatalf("hinge failed to lock orthogonal angular axes: wx=%.4f, wy=%.4f", bwx, bwy)
	}
	if math.Abs(bwz) < 0.1 {
		t.Fatalf("hinge erroneously blocked allowed rotation around hinge axis Z: wz=%.4f", bwz)
	}
}

func TestSliderConstraintPrismaticLimitsAndZeroAllocs(t *testing.T) {
	rigid_body.SetFps(60)
	defer rigid_body.SetFps(120)

	// Solver A at (0, 0, 10), Solver B at (30, 0, 10).
	// Slider along X axis (1, 0, 0) with limits [-50, 50].
	sA, _ := newFreeSixDoFSolver(100.0, 0, 0, 10, 20, 20, 20, 0, false)
	sB, _ := newFreeSixDoFSolver(10.0, 30, 0, 10, 20, 20, 20, 0, false)
	sA.SetSolverIndex(0)
	sB.SetSolverIndex(1)
	sA.GetEntity().WakeUp()
	sB.GetEntity().WakeUp()

	slider := NewSliderJoint(sA, sB, 0, 0, 0, 0, 0, 0, 1, 0, 0, -50.0, 50.0, 1.0)

	// Zero heap allocations in ResolveVelocity / ResolvePosition
	allocs := testing.AllocsPerRun(100, func() {
		slider.ResolveVelocity()
		slider.ResolvePosition()
	})
	if allocs != 0 {
		t.Fatalf("expected 0 heap allocs in slider PGS solve, got %v", allocs)
	}

	// Try to push B laterally (Y and Z) and twist it
	sB.GetEntity().SetV(20.0, 40.0, 40.0)
	sB.GetEntity().SetAngularVelocity(5.0, 5.0, 5.0)

	for iter := 0; iter < 10; iter++ {
		slider.ResolveVelocity()
	}

	bvx, bvy, bvz := sB.GetEntity().GetVelocity()
	bwx, bwy, bwz := sB.GetEntity().GetAngularVelocity()

	if math.Abs(bvy) > 0.5 || math.Abs(bvz) > 0.5 {
		t.Fatalf("slider failed to lock lateral velocities: vy=%.4f, vz=%.4f", bvy, bvz)
	}
	if math.Abs(bvx) < 1.0 {
		t.Fatalf("slider erroneously blocked motion along slide axis X: vx=%.4f", bvx)
	}
	if math.Abs(bwx) > 0.5 || math.Abs(bwy) > 0.5 || math.Abs(bwz) > 0.5 {
		t.Fatalf("slider failed to lock angular orientation: w=(%.4f, %.4f, %.4f)", bwx, bwy, bwz)
	}
}

func TestFixedConstraintWeldAndZeroAllocs(t *testing.T) {
	rigid_body.SetFps(60)
	defer rigid_body.SetFps(120)

	// Solver A at (0, 0, 10), Solver B at (20, 0, 10) welded at midpoint (10, 0, 10).
	sA, _ := newFreeSixDoFSolver(50.0, 0, 0, 10, 20, 20, 20, 0, false)
	sB, _ := newFreeSixDoFSolver(50.0, 20, 0, 10, 20, 20, 20, 0, false)
	sA.SetSolverIndex(0)
	sB.SetSolverIndex(1)
	sA.GetEntity().WakeUp()
	sB.GetEntity().WakeUp()

	fixed := NewFixedJoint(sA, sB, 10, 0, 0, -10, 0, 0, aabb.NewQuaternionIdentity())

	// Zero heap allocations in ResolveVelocity / ResolvePosition
	allocs := testing.AllocsPerRun(100, func() {
		fixed.ResolveVelocity()
		fixed.ResolvePosition()
	})
	if allocs != 0 {
		t.Fatalf("expected 0 heap allocs in fixed weld PGS solve, got %v", allocs)
	}

	// Apply opposing forces and twists to A and B
	sA.GetEntity().SetV(30.0, -20.0, 10.0)
	sB.GetEntity().SetV(-30.0, 20.0, -10.0)
	sA.GetEntity().SetAngularVelocity(4.0, -2.0, 3.0)
	sB.GetEntity().SetAngularVelocity(-4.0, 2.0, -3.0)

	for iter := 0; iter < 15; iter++ {
		fixed.ResolveVelocity()
	}

	avx, avy, avz := sA.GetEntity().GetVelocity()
	bvx, bvy, bvz := sB.GetEntity().GetVelocity()
	awx, awy, awz := sA.GetEntity().GetAngularVelocity()
	bwx, bwy, bwz := sB.GetEntity().GetAngularVelocity()

	// Relative velocity and relative angular velocity must converge towards zero
	relV := math.Sqrt((bvx-avx)*(bvx-avx) + (bvy-avy)*(bvy-avy) + (bvz-avz)*(bvz-avz))
	relW := math.Sqrt((bwx-awx)*(bwx-awx) + (bwy-awy)*(bwy-awy) + (bwz-awz)*(bwz-awz))

	if relV > 1.0 {
		t.Fatalf("fixed joint relative velocity did not lock: relV=%.4f", relV)
	}
	if relW > 0.5 {
		t.Fatalf("fixed joint relative angular velocity did not lock: relW=%.4f", relW)
	}
}

func TestCylindricalAndUniversalJointZeroAllocs(t *testing.T) {
	sA, _ := newFreeSixDoFSolver(50.0, 0, 0, 10, 20, 20, 20, 0, false)
	sB, _ := newFreeSixDoFSolver(50.0, 0, 0, 30, 20, 20, 20, 0, false)
	sA.SetSolverIndex(0)
	sB.SetSolverIndex(1)

	cyl := NewCylindricalJoint(sA, sB, 0, 0, 10, 0, 0, -10, 0, 0, 1, -20, 20, -1, 1, 0.5)
	allocsCyl := testing.AllocsPerRun(100, func() {
		cyl.ResolveVelocity()
		cyl.ResolvePosition()
	})
	if allocsCyl != 0 {
		t.Fatalf("expected 0 heap allocs in cylindrical joint solve, got %v", allocsCyl)
	}

	univ := NewUniversalJoint(sA, sB, 0, 0, 10, 0, 0, -10, 1, 0, 0, 0, 1, 0, 0.5)
	allocsUniv := testing.AllocsPerRun(100, func() {
		univ.ResolveVelocity()
		univ.ResolvePosition()
	})
	if allocsUniv != 0 {
		t.Fatalf("expected 0 heap allocs in universal joint solve, got %v", allocsUniv)
	}
}
