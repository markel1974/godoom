package collisions

import (
	"math"
	"testing"

	"github.com/markel1974/godoom/mr_tech/physics/aabb"
	"github.com/markel1974/godoom/mr_tech/physics/rigid_body"
)

func newFreeSixDoFSolver(mass, cx, cy, cz, w, h, d, gForce float64, canSleep bool) (*Solver, *testObject) {
	solver := NewSolver(true, mass, 0.0, 0.5, int(rigid_body.CombineAverage), int(rigid_body.CombineAverage), gForce)
	ent := solver.GetEntity()
	ent.Rebuild(cx-w*0.5, cy-h*0.5, cz-d*0.5, w, h, d)
	ent.SetKeepUpright(false)
	ent.SetOnGround(false)
	obj := &testObject{solver: solver, ent: ent, box: aabb.NewAABB()}
	_ = solver.Setup(obj)
	solver.SetCanSleep(canSleep)
	return solver, obj
}

func TestJointConstraintAnchorInvarianceConeLimitAndSleep(t *testing.T) {
	rigid_body.SetFps(60)
	defer rigid_body.SetFps(120)

	// Anchor A is a heavy floor-supported body at (0,0,10); B is a limb at (0,0,30) joined at z=20.
	sA, _ := newFreeSixDoFSolver(200.0, 0, 0, 10, 20, 20, 20, 490.0, true)
	sB, _ := newFreeSixDoFSolver(20.0, 0, 0, 30, 16, 16, 20, 490.0, true)
	sA.GetEntity().WakeUp()
	sB.GetEntity().WakeUp()

	// Kick B laterally to test ball-socket cohesion and cone limit clamping.
	sB.GetEntity().SetV(180.0, 0.0, 40.0)
	sB.GetEntity().SetAngularVelocity(0.0, 12.0, 0.0)

	// Static 2-triangle floor quad at z=0.
	floorParent := NewSolver(false, 0, 0, 0.5, int(rigid_body.CombineAverage), int(rigid_body.CombineAverage), 0)
	_ = floorParent.Setup(&testObject{solver: floorParent, ent: floorParent.GetEntity(), box: aabb.NewAABB()})
	floor0 := &jointFloorFace{
		testFace: newTestFace([3]float64{-500, -500, 0}, [3]float64{500, -500, 0}, [3]float64{500, 500, 0}, 0, 0, 1),
		parent:   floorParent,
	}
	floor1 := &jointFloorFace{
		testFace: newTestFace([3]float64{-500, -500, 0}, [3]float64{500, 500, 0}, [3]float64{-500, 500, 0}, 0, 0, 1),
		parent:   floorParent,
	}

	queryContacts := func(cage *Cage, _ WakeRunningFn) {
		for _, cand := range []*Solver{sA, sB} {
			ent := cand.GetEntity()
			_, _, z := ent.GetCenter()
			if z <= 18.0 && cand.GetCage() == cage {
				cage.AddFace(floor0, nil)
				cage.AddFace(floor1, nil)
				cage.Commit(nil)
			}
		}
	}
	clampStatic := func(cage *Cage) {
		ent := cage.GetEntity()
		_, _, z := ent.GetCenter()
		_, _, d := ent.GetSize()
		if minZ := z - d*0.5; minZ < 0.0 {
			ent.AddTo(0, 0, -minZ)
			if _, _, vz := ent.GetVelocity(); vz < 0 {
				ent.SetVz(0)
			}
		}
	}

	coord := NewCoordinator(10, 8, queryContacts, clampStatic)

	const maxCone = 0.90
	jc := NewJointConstraint(sA, sB, 0, 0, 10, 0, 0, -10, aabb.NewQuaternionIdentity(), maxCone, 6.0)
	coord.AddJoint(jc)

	var maxAnchorGap float64
	var maxConeSeen float64
	for step := 0; step < 600; step++ {
		coord.Reset()
		coord.Stage(sA)
		coord.Stage(sB)
		coord.Step()

		ax, ay, az := sA.GetEntity().GetCenter()
		ax0, ay0, az0, _, _, _, a2x, a2y, a2z := sA.GetEntity().GetAxes()
		_ = ax0
		_ = ay0
		_ = az0
		pAx := ax + 10.0*a2x
		pAy := ay + 10.0*a2y
		pAz := az + 10.0*a2z

		bx, by, bz := sB.GetEntity().GetCenter()
		_, _, _, _, _, _, b2x, b2y, b2z := sB.GetEntity().GetAxes()
		pBx := bx - 10.0*b2x
		pBy := by - 10.0*b2y
		pBz := bz - 10.0*b2z

		gap := math.Sqrt((pBx-pAx)*(pBx-pAx) + (pBy-pAy)*(pBy-pAy) + (pBz-pAz)*(pBz-pAz))
		if gap > maxAnchorGap {
			maxAnchorGap = gap
		}

		// Measure relative cone angle between A and B.
		qA := sA.GetEntity().GetQRotation()
		qB := sB.GetEntity().GetQRotation()
		var qConj, qRel aabb.Quaternion
		qConj.Conjugate(qA)
		qConj.Multiplies(qB, &qRel)
		ex, ey, ez, ew := qRel.Get()
		if ew < 0 {
			ex, ey, ez, ew = -ex, -ey, -ez, -ew
		}
		ang := 2.0 * math.Atan2(math.Sqrt(ex*ex+ey*ey+ez*ez), ew)
		if ang > maxConeSeen {
			maxConeSeen = ang
		}
	}

	if maxAnchorGap > 1.5 {
		t.Fatalf("ball-socket anchor separated too far: maxGap=%.4f > 1.5", maxAnchorGap)
	}
	if maxConeSeen > maxCone+0.15 {
		t.Fatalf("cone limit exceeded tolerance: maxConeSeen=%.4f > %.4f", maxConeSeen, maxCone+0.15)
	}
	if !sA.IsSleeping() || !sB.IsSleeping() {
		avx, avy, avz := sA.GetEntity().GetVelocity()
		awx, awy, awz := sA.GetEntity().GetAngularVelocity()
		apx, apy, apz := sA.GetCage().GetNetDisplacement()
		bvx, bvy, bvz := sB.GetEntity().GetVelocity()
		bwx, bwy, bwz := sB.GetEntity().GetAngularVelocity()
		bpx, bpy, bpz := sB.GetCage().GetNetDisplacement()
		t.Fatalf("expected grounded joint island to sleep: sA=%v (onGround=%v v=(%.4f,%.4f,%.4f) w=(%.4f,%.4f,%.4f) net=(%.4f,%.4f,%.4f)) sB=%v (onGround=%v v=(%.4f,%.4f,%.4f) w=(%.4f,%.4f,%.4f) net=(%.4f,%.4f,%.4f)) gap=%.4f",
			sA.IsSleeping(), sA.GetEntity().IsOnGround(), avx, avy, avz, awx, awy, awz, apx, apy, apz,
			sB.IsSleeping(), sB.GetEntity().IsOnGround(), bvx, bvy, bvz, bwx, bwy, bwz, bpx, bpy, bpz, maxAnchorGap)
	}

	// Waking sB must cascade-wake sA through the joint edge.
	sB.GetEntity().SetV(0, 0, 50)
	coord.Reset()
	coord.Stage(sA)
	coord.Stage(sB)
	coord.Step()
	if sA.IsSleeping() {
		t.Fatalf("expected waking child limb sB to cascade-wake sA via JointConstraint")
	}

	// Zero heap allocations in ResolveVelocity / ResolvePosition.
	sA.SetSolverIndex(0)
	sB.SetSolverIndex(1)
	allocs := testing.AllocsPerRun(100, func() {
		jc.ResolveVelocity()
		jc.ResolvePosition()
	})
	if allocs != 0 {
		t.Fatalf("expected 0 heap allocs in joint PGS step, got %v", allocs)
	}
}

type jointFloorFace struct {
	*testFace
	parent *Solver
}

func (f *jointFloorFace) GetParent() *Solver { return f.parent }
