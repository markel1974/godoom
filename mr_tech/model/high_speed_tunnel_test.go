package model

import (
	"testing"

	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/model/volumes"
	"github.com/markel1974/godoom/mr_tech/physics/aabb"
	"github.com/markel1974/godoom/mr_tech/physics/collisions"
	"github.com/markel1974/godoom/mr_tech/physics/rigid_body"
	"github.com/markel1974/godoom/mr_tech/textures"
)

func runHighSpeedStep(t *testing.T, vols *Volumes, cx, cy, cz, w, h, d, vx, vy, vz float64, keepUpright bool, maxOverlaps int, steps int) (float64, float64, float64) {
	t.Helper()
	solver := collisions.NewSolver(true, 80.0, 0.0, 0.2, int(rigid_body.CombineAverage), int(rigid_body.CombineAverage), 490.0)
	ent := solver.GetEntity()
	ent.Rebuild(cx-w*0.5, cy-h*0.5, cz-d*0.5, w, h, d)
	ent.SetKeepUpright(keepUpright)
	if keepUpright {
		ent.LockUpright()
	}
	ent.SetOnGround(false)
	ent.SetV(vx, vy, vz)
	obj := &stepTestObject{ent: ent, box: aabb.NewAABB()}
	if err := solver.Setup(obj); err != nil {
		t.Fatal(err)
	}
	solver.SetCanSleep(false)
	solver.SetMaxStep(18.0)

	coord := collisions.NewCoordinator(10, 16)
	queryContacts := func(cage *collisions.Cage, _ func(*collisions.Solver)) {
		vols.QueryCollisionCage(cage, maxOverlaps)
	}
	for s := 0; s < steps; s++ {
		coord.Reset()
		coord.Stage(solver)
		coord.Step(queryContacts, vols.ClampStaticCage)
	}
	return ent.GetCenter()
}

func TestHighSpeedWallTunneling(t *testing.T) {
	rigid_body.SetFps(60)
	defer rigid_body.SetFps(120)

	volsFlat := newSolidVolume([][3]geometry.XYZ{
		{{X: 100, Y: 200, Z: 0}, {X: 100, Y: -200, Z: 0}, {X: 100, Y: -200, Z: 200}},
		{{X: 100, Y: 200, Z: 0}, {X: 100, Y: -200, Z: 200}, {X: 100, Y: 200, Z: 200}},
	})
	if x, _, _ := runHighSpeedStep(t, volsFlat, 50, 0, 50, 40, 40, 60, 9000, 0, 0, true, 512, 5); x > 80.5 {
		t.Fatalf("upright flat wall tunneled: x=%.3f > 80.5", x)
	}
	if x, _, _ := runHighSpeedStep(t, volsFlat, 50, 0, 50, 40, 40, 60, 9000, 0, 0, false, 512, 5); x > 80.5 {
		t.Fatalf("6DoF flat wall tunneled: x=%.3f > 80.5", x)
	}
	// Pressed slightly into wall (0 <= distStart < rayEff - satFilterEpsilon) at high speed must still block
	if x, _, _ := runHighSpeedStep(t, volsFlat, 97.6, 0, 50, 5, 5, 66, 3600, 0, 0, true, 512, 2); x > 98.0 {
		t.Fatalf("small-radius player pressed into wall tunneled: x=%.3f > 98.0", x)
	}

	volsTilted := newSolidVolume([][3]geometry.XYZ{
		{{X: 100, Y: 200, Z: 0}, {X: 100, Y: -200, Z: 0}, {X: 90, Y: -200, Z: 200}},
		{{X: 100, Y: 200, Z: 0}, {X: 90, Y: -200, Z: 200}, {X: 90, Y: 200, Z: 200}},
	})
	if x, _, _ := runHighSpeedStep(t, volsTilted, 20, 0, 50, 40, 40, 60, 18000, 0, 0, true, 512, 2); x > 80.5 {
		t.Fatalf("upright tilted wall tunneled at dx=300: x=%.3f > 80.5", x)
	}

	volsLintel := newSolidVolume([][3]geometry.XYZ{
		{{X: 100, Y: 200, Z: 55}, {X: 100, Y: -200, Z: 55}, {X: 100, Y: -200, Z: 200}},
		{{X: 100, Y: 200, Z: 55}, {X: 100, Y: -200, Z: 200}, {X: 100, Y: 200, Z: 200}},
	})
	if x, _, _ := runHighSpeedStep(t, volsLintel, 20, 0, 50, 40, 40, 60, 18000, 0, 0, true, 512, 2); x > 80.5 {
		t.Fatalf("upright lintel tunneled at dx=300: x=%.3f > 80.5", x)
	}
	if x, _, _ := runHighSpeedStep(t, volsLintel, 90, 0, 50, 5, 5, 66, 1800, 0, 0, true, 512, 3); x > 98.0 {
		t.Fatalf("small-radius player (r=2.5, vx=1800) tunneled lintel: x=%.3f > 98.0", x)
	}

	// Busy multi-volume scene >512 triangles before the far wall volume (relies on ClampStaticCage)
	mat := textures.NewMaterial("solid", nil, 0)
	volFloor := volumes.NewVolumeConcrete(0, "floor_busy", "floor")
	for k := 0; k < 600; k++ {
		fx := float64(k%30) * 10.0
		fy := float64(k/30)*10.0 - 100.0
		volFloor.AddFace(volumes.NewFace([3]geometry.XYZ{
			{X: fx, Y: fy, Z: 0}, {X: fx + 8, Y: fy, Z: 0}, {X: fx, Y: fy + 8, Z: 0},
		}, "f", mat))
	}
	volWall := volumes.NewVolumeConcrete(1, "back_wall", "wall")
	volWall.AddFace(volumes.NewFace([3]geometry.XYZ{
		{X: 250, Y: 200, Z: 0}, {X: 250, Y: -200, Z: 0}, {X: 250, Y: -200, Z: 200},
	}, "w", mat))
	volWall.AddFace(volumes.NewFace([3]geometry.XYZ{
		{X: 250, Y: 200, Z: 0}, {X: 250, Y: -200, Z: 200}, {X: 250, Y: 200, Z: 200},
	}, "w", mat))
	volsBusy := NewVolumes([]*volumes.Volume{volFloor, volWall})
	if err := volsBusy.Setup(); err != nil {
		t.Fatal(err)
	}
	if x, _, _ := runHighSpeedStep(t, volsBusy, 20, 0, 30, 40, 40, 60, 15000, 0, 0, true, 512, 5); x > 230.5 {
		t.Fatalf("busy sector (>512 faces) wall tunneled: x=%.3f > 230.5", x)
	}

	// Finite-Z walls crossed mid-sweep while falling or launching rapidly
	volsBand := newSolidVolume([][3]geometry.XYZ{
		{{X: 100, Y: 200, Z: 20}, {X: 100, Y: -200, Z: 20}, {X: 100, Y: -200, Z: 100}},
		{{X: 100, Y: 200, Z: 20}, {X: 100, Y: -200, Z: 100}, {X: 100, Y: 200, Z: 100}},
	})
	if x, _, _ := runHighSpeedStep(t, volsBand, 20, 0, 100, 40, 40, 60, 12000, 0, -12000, true, 512, 1); x > 80.5 {
		t.Fatalf("upright falling across finite-Z wall tunneled: x=%.3f > 80.5", x)
	}
	if x, _, _ := runHighSpeedStep(t, volsBand, 20, 0, 100, 40, 40, 40, 12000, 0, -12000, false, 512, 1); x > 80.5 {
		t.Fatalf("6DoF falling across finite-Z wall tunneled: x=%.3f > 80.5", x)
	}

	volsLowBand := newSolidVolume([][3]geometry.XYZ{
		{{X: 100, Y: 200, Z: 0}, {X: 100, Y: -200, Z: 0}, {X: 100, Y: -200, Z: 100}},
		{{X: 100, Y: 200, Z: 0}, {X: 100, Y: -200, Z: 100}, {X: 100, Y: 200, Z: 100}},
	})
	if x, _, _ := runHighSpeedStep(t, volsLowBand, 20, 0, 50, 40, 40, 60, 12000, 0, 9600, true, 512, 1); x > 80.5 {
		t.Fatalf("upward+forward launch across wall tunneled: x=%.3f > 80.5", x)
	}

	// Clean high jump above low wall must not be falsely blocked
	volsJump := newSolidVolume([][3]geometry.XYZ{
		{{X: 100, Y: 200, Z: 0}, {X: 100, Y: -200, Z: 0}, {X: 100, Y: -200, Z: 60}},
		{{X: 100, Y: 200, Z: 0}, {X: 100, Y: -200, Z: 60}, {X: 100, Y: 200, Z: 60}},
	})
	if x, _, _ := runHighSpeedStep(t, volsJump, 20, 0, 120, 40, 40, 60, 12000, 0, 6000, true, 512, 1); x < 200.0 {
		t.Fatalf("clean jump over low wall was falsely blocked: x=%.3f < 200.0", x)
	}

	// Convex corner slide graze along Wall 1 past Wall 2 must not snag in +Y
	volsCorner := newSolidVolume([][3]geometry.XYZ{
		{{X: 100, Y: 100, Z: 0}, {X: 100, Y: 0, Z: 0}, {X: 100, Y: 0, Z: 200}},
		{{X: 100, Y: 100, Z: 0}, {X: 100, Y: 0, Z: 200}, {X: 100, Y: 100, Z: 200}},
		{{X: 100, Y: 100, Z: 0}, {X: 200, Y: 100, Z: 0}, {X: 200, Y: 100, Z: 200}},
		{{X: 100, Y: 100, Z: 0}, {X: 200, Y: 100, Z: 200}, {X: 100, Y: 100, Z: 200}},
	})
	if _, y, _ := runHighSpeedStep(t, volsCorner, 80.001, 50, 50, 40, 40, 60, 0, 9000, 0, true, 512, 1); y < 195.0 {
		t.Fatalf("convex corner +Y graze snagged: y=%.3f < 195.0", y)
	}

	// Doorway post shoulder hit must block before tunneling through x=100
	volsDoorPost := newSolidVolume([][3]geometry.XYZ{
		{{X: 100, Y: 200, Z: 0}, {X: 100, Y: 50, Z: 0}, {X: 100, Y: 50, Z: 200}},
		{{X: 100, Y: 200, Z: 0}, {X: 100, Y: 50, Z: 200}, {X: 100, Y: 200, Z: 200}},
	})
	if x, _, _ := runHighSpeedStep(t, volsDoorPost, 20, 40, 50, 40, 40, 60, 12000, 0, 0, true, 512, 1); x > 85.0 {
		t.Fatalf("doorway post shoulder hit tunneled: x=%.3f > 85.0", x)
	}

	// Diagonal phantom infinite-plane extension miss must not falsely block
	volsPhantomDiag := newSolidVolume([][3]geometry.XYZ{
		{{X: 100, Y: 100, Z: 0}, {X: 200, Y: 200, Z: 0}, {X: 200, Y: 200, Z: 200}},
		{{X: 100, Y: 100, Z: 0}, {X: 200, Y: 200, Z: 200}, {X: 100, Y: 100, Z: 200}},
	})
	if x, _, _ := runHighSpeedStep(t, volsPhantomDiag, 50, 80, 50, 40, 40, 60, 4200, -2400, 0, true, 512, 1); x < 119.0 {
		t.Fatalf("phantom diagonal extension falsely blocked: x=%.3f < 119.0", x)
	}

	// High-speed dive across thin floating platform (both 0<=distTarget<r and tunneled)
	volsPlat := newSolidVolume([][3]geometry.XYZ{
		{{X: 100, Y: -100, Z: 100}, {X: 140, Y: -100, Z: 100}, {X: 140, Y: 100, Z: 100}},
		{{X: 100, Y: -100, Z: 100}, {X: 140, Y: 100, Z: 100}, {X: 100, Y: 100, Z: 100}},
	})
	if _, _, z := runHighSpeedStep(t, volsPlat, 50, 0, 140, 40, 40, 60, 9000, 0, -2100, true, 512, 1); z < 129.0 {
		t.Fatalf("floating platform tunneled (0<=distTarget<r): z=%.3f < 129.0", z)
	}
	if _, _, z := runHighSpeedStep(t, volsPlat, 50, 0, 140, 40, 40, 60, 9000, 0, -6000, true, 512, 1); z < 129.0 {
		t.Fatalf("floating platform tunneled (tunneled): z=%.3f < 129.0", z)
	}

	// Angled 45-degree wall x+y=200 at extreme speed for upright and 6DoF
	volsDiag := newSolidVolume([][3]geometry.XYZ{
		{{X: 0, Y: 200, Z: 0}, {X: 200, Y: 0, Z: 0}, {X: 200, Y: 0, Z: 200}},
		{{X: 0, Y: 200, Z: 0}, {X: 200, Y: 0, Z: 200}, {X: 0, Y: 200, Z: 200}},
	})
	if x, y, _ := runHighSpeedStep(t, volsDiag, 0, 0, 50, 40, 40, 60, 12000, 12000, 0, true, 512, 2); x+y > 175.0 {
		t.Fatalf("upright 45deg wall tunneled: x+y=%.3f > 175.0", x+y)
	}
	if x, y, _ := runHighSpeedStep(t, volsDiag, 0, 0, 50, 40, 40, 40, 12000, 12000, 0, false, 512, 2); x+y > 175.0 {
		t.Fatalf("6DoF 45deg wall tunneled: x+y=%.3f > 175.0", x+y)
	}

	// Short angled 45-degree wall segment with pen0 > 0 must not be rejected by single-axis overlap heuristic.
	volsShortDiag := newSolidVolume([][3]geometry.XYZ{
		{{X: 100, Y: 100, Z: 0}, {X: 110, Y: 90, Z: 0}, {X: 110, Y: 90, Z: 200}},
		{{X: 100, Y: 100, Z: 0}, {X: 110, Y: 90, Z: 200}, {X: 100, Y: 100, Z: 200}},
	})
	if x, y, _ := runHighSpeedStep(t, volsShortDiag, 89, 89, 50, 40, 40, 60, 6000, 6000, 0, true, 512, 1); x+y > 175.0 {
		t.Fatalf("short 45deg wall with pen0>0 tunneled: x+y=%.3f > 175.0", x+y)
	}
	if x, y, _ := runHighSpeedStep(t, volsShortDiag, 89, 89, 50, 40, 40, 40, 6000, 6000, 0, false, 512, 3); x+y > 175.0 {
		t.Fatalf("6DoF short 45deg wall with pen0>0 tunneled: x+y=%.3f > 175.0", x+y)
	}

	// 6-DoF elongated/spinning object hit by repeated off-center hitscan impulses against a wall.
	{
		solver := collisions.NewSolver(true, 10.0, 0.2, 0.5, int(rigid_body.CombineAverage), int(rigid_body.CombineAverage), 490.0)
		ent := solver.GetEntity()
		ent.Rebuild(70-4, -15, 40, 8, 30, 8)
		ent.SetCentroid(70+3, 10, 44)
		ent.SetKeepUpright(false)
		ent.SetOnGround(false)
		obj := &stepTestObject{ent: ent, box: aabb.NewAABB()}
		if err := solver.Setup(obj); err != nil {
			t.Fatal(err)
		}
		solver.SetCanSleep(false)
		coord := collisions.NewCoordinator(10, 16)
		queryContacts := func(cage *collisions.Cage, _ func(*collisions.Solver)) {
			volsFlat.QueryCollisionCage(cage, 512)
		}
		for step := 0; step < 10; step++ {
			cx, cy, cz := ent.GetCenter()
			ent.AddForceAtPoint(500000.0, 120000.0, 0.0, cx, cy+12.0, cz+3.0)
			coord.Reset()
			coord.Stage(solver)
			coord.Step(queryContacts, volsFlat.ClampStaticCage)
			if nx, _, _ := ent.GetCenter(); nx > 100.0 {
				t.Fatalf("spinning shot 6DoF object tunneled through flat wall at step %d: x=%.3f > 100.0", step, nx)
			}
		}
	}

	// Narrow concave wedge + opposite-facing adjacent sub-volume face (84_3d pattern) across multi-iteration PGS.
	volsWedge := newSolidVolume([][3]geometry.XYZ{
		// Bottom wedge wall facing NNW into wedge (-0.197, +0.980, 0)
		{{X: 3556.4, Y: 3527.6, Z: 209}, {X: 3407.7, Y: 3497.8, Z: 209}, {X: 3407.7, Y: 3497.8, Z: 594}},
		{{X: 3556.4, Y: 3527.6, Z: 209}, {X: 3407.7, Y: 3497.8, Z: 594}, {X: 3556.4, Y: 3527.6, Z: 594}},
		// Top wedge wall facing SE into wedge (+0.610, -0.792, 0)
		{{X: 3407.7, Y: 3497.8, Z: 209}, {X: 3608.0, Y: 3652.0, Z: 209}, {X: 3608.0, Y: 3652.0, Z: 594}},
		{{X: 3407.7, Y: 3497.8, Z: 209}, {X: 3608.0, Y: 3652.0, Z: 594}, {X: 3407.7, Y: 3497.8, Z: 594}},
		// Back-to-back opposite face (-0.610, +0.792, 0) on same top chord
		{{X: 3608.0, Y: 3652.0, Z: 209}, {X: 3407.7, Y: 3497.8, Z: 209}, {X: 3407.7, Y: 3497.8, Z: 594}},
		{{X: 3608.0, Y: 3652.0, Z: 209}, {X: 3407.7, Y: 3497.8, Z: 594}, {X: 3608.0, Y: 3652.0, Z: 594}},
	})
	if x, y, _ := runHighSpeedStep(t, volsWedge, 3417.70, 3502.34, 253, 5, 5, 66, -3500, -1000, 0, true, 512, 3); (x-3407.7)*(-0.197)+(y-3497.8)*0.980 < 0.0 {
		t.Fatalf("concave wedge pierced bottom wall: (%.3f, %.3f)", x, y)
	}

	// Q1/Q2 two-sided 2-triangle quad approached from -n by a tumbling 6DoF box must use consistent pass displacement
	// across both quad triangles instead of letting tri1 push-back reverse tri2 into tunneling (-448 Q2 pattern).
	volsQ2Quad := newSolidVolume([][3]geometry.XYZ{
		{{X: -128, Y: -448, Z: 128}, {X: 0, Y: -448, Z: 128}, {X: 0, Y: -448, Z: 0}},
		{{X: 0, Y: -448, Z: 128}, {X: -64, Y: -448, Z: 0}, {X: 0, Y: -448, Z: 0}},
	})
	if _, y, _ := runHighSpeedStep(t, volsQ2Quad, -40.3, -432.7, 40.0, 36.4, 30.4, 45.2, 600, -2400, 0, false, 512, 3); y < -448.0 {
		t.Fatalf("two-sided Q2 quad reversed between triangles and tunneled: y=%.3f < -448.0", y)
	}
}
