package rigid_body

import (
	"math"
	"testing"
)

func TestKeepUprightWallContactNoVelocityRebound(t *testing.T) {
	player := NewEntity(80.0, 0.0, 0.2, int(CombineAverage), int(CombineAverage), 0.0)
	player.Rebuild(0.0, 0.0, 0.0, 40.0, 40.0, 60.0)
	player.SetKeepUpright(true)
	player.LockUpright()
	player.SetV(300.0, 150.0, 0.0)

	wall := NewEntity(0.0, 0.0, 0.2, int(CombineAverage), int(CombineAverage), 0.0)
	wall.Rebuild(40.0, -100.0, -100.0, 20.0, 200.0, 200.0)

	var accNormal, accTx, accTy, accTz, restTarget float64
	var hasRestTarget bool

	// Off-center impact point in both Y and Z relative to player center (20, 20, 30).
	const nx, ny, nz = -1.0, 0.0, 0.0
	const px, py, pz = 40.0, 32.0, 48.0

	for tick := 0; tick < 8; tick++ {
		if tick > 0 {
			player.WarmStartImpact(wall, nx, ny, nz, px, py, pz, &accNormal, &accTx, &accTy, &accTz)
		}
		for iter := 0; iter < 10; iter++ {
			player.ResolveImpactAccumulated(wall, nx, ny, nz, px, py, pz, &accNormal, &accTx, &accTy, &accTz, &restTarget, &hasRestTarget)
		}

		wx, wy, wz := player.GetAngularVelocity()
		if wx != 0.0 || wy != 0.0 || wz != 0.0 {
			t.Fatalf("tick %d: keepUpright entity acquired angular velocity (%v, %v, %v)", tick, wx, wy, wz)
		}
		player.StopAngular()

		vx := player.GetVx()
		if vx < -1e-6 {
			t.Fatalf("tick %d: upright entity rebounded away from zero-restitution wall: vx=%v", tick, vx)
		}
	}

	if vx := player.GetVx(); math.Abs(vx) > 1e-4 {
		t.Fatalf("expected normal velocity to settle at 0 after wall contact, got vx=%v", vx)
	}
	if vy := player.GetVy(); vy <= 0.0 {
		t.Fatalf("expected tangential slide velocity along wall to remain positive, got vy=%v", vy)
	}
}

func TestDynamic6DoFRetainsRotationalImpact(t *testing.T) {
	box := NewEntity(80.0, 0.0, 0.2, int(CombineAverage), int(CombineAverage), 0.0)
	box.Rebuild(0.0, 0.0, 0.0, 40.0, 40.0, 60.0)
	box.SetV(300.0, 0.0, 0.0)

	wall := NewEntity(0.0, 0.0, 0.2, int(CombineAverage), int(CombineAverage), 0.0)
	wall.Rebuild(40.0, -100.0, -100.0, 20.0, 200.0, 200.0)

	var accNormal, accTx, accTy, accTz, restTarget float64
	var hasRestTarget bool
	box.ResolveImpactAccumulated(wall, -1.0, 0.0, 0.0, 40.0, 32.0, 48.0, &accNormal, &accTx, &accTy, &accTz, &restTarget, &hasRestTarget)

	_, wy, wz := box.GetAngularVelocity()
	if wy == 0.0 && wz == 0.0 {
		t.Fatalf("expected free 6-DoF rigid body to gain rotational velocity from off-center wall impact")
	}
}

func TestKeepUprightImpactAllocs(t *testing.T) {
	player := NewEntity(80.0, 0.0, 0.2, int(CombineAverage), int(CombineAverage), 0.0)
	player.Rebuild(0.0, 0.0, 0.0, 40.0, 40.0, 60.0)
	player.SetKeepUpright(true)
	player.LockUpright()

	wall := NewEntity(0.0, 0.0, 0.2, int(CombineAverage), int(CombineAverage), 0.0)
	wall.Rebuild(40.0, -100.0, -100.0, 20.0, 200.0, 200.0)

	var accNormal, accTx, accTy, accTz, restTarget float64
	var hasRestTarget bool

	allocs := testing.AllocsPerRun(100, func() {
		player.SetV(300.0, 150.0, 0.0)
		player.WarmStartImpact(wall, -1.0, 0.0, 0.0, 40.0, 32.0, 48.0, &accNormal, &accTx, &accTy, &accTz)
		player.ResolveImpactAccumulated(wall, -1.0, 0.0, 0.0, 40.0, 32.0, 48.0, &accNormal, &accTx, &accTy, &accTz, &restTarget, &hasRestTarget)
	})
	if allocs != 0 {
		t.Fatalf("expected 0 heap allocs per impact solve, got %v", allocs)
	}
}
