package vehicle

import (
	"math"
	"testing"

	"github.com/markel1974/godoom/mr_tech/physics/rigid_body"
)

// flatFloorRaycast simulates an infinite flat ground plane at Z = 0.
func flatFloorRaycast(fromX, fromY, fromZ, dirX, dirY, dirZ, maxDist float64) (float64, float64, float64, float64, bool) {
	if dirZ >= -1e-6 {
		return 0, 0, 0, 1, false
	}
	t := fromZ / (-dirZ)
	if t >= 0.0 && t <= maxDist {
		return t, 0, 0, 1, true
	}
	return 0, 0, 0, 1, false
}

func newTestVehicleChassis(mass, cx, cy, cz, w, h, d float64) *rigid_body.Entity {
	ent := rigid_body.NewEntity(mass, 0.0, 0.5, int(rigid_body.CombineAverage), int(rigid_body.CombineAverage), 9.81)
	ent.Rebuild(cx-w*0.5, cy-h*0.5, cz-d*0.5, w, h, d)
	ent.SetKeepUpright(false)
	ent.SetOnGround(false)
	ent.WakeUp()
	return ent
}

func stepTestChassis(ent *rigid_body.Entity) {
	ent.Update()
	dx, dy, dz := ent.GetFrameDisplacement()
	ent.AddTo(dx, dy, dz)
}

func TestVehicleSuspensionSettling(t *testing.T) {
	rigid_body.SetFps(60)
	defer rigid_body.SetFps(120)

	// Sedan chassis with COM at Z = 0.8m above ground
	chassis := newTestVehicleChassis(1500.0, 0, 0, 0.8, 1.8, 4.2, 1.4)
	veh := NewRaycastVehicle(chassis, flatFloorRaycast)
	veh.SetupDefault(0.9, 1.4, 1.4, -0.2)

	dt := 1.0 / 60.0
	chassis.SetDt(dt)

	// Simulate 120 frames (2 seconds) of resting on suspension
	for step := 0; step < 120; step++ {
		veh.Update(dt)
		stepTestChassis(chassis)
	}

	// All 4 wheels must be contacting ground with positive suspension support force
	for i := 0; i < veh.WheelsLen(); i++ {
		w := veh.GetWheel(i)
		if !w.InContact {
			t.Fatalf("expected wheel %d to contact ground", i)
		}
		if w.SuspensionForce < 1000.0 {
			t.Fatalf("wheel %d suspension force too low: %.1f N", i, w.SuspensionForce)
		}
	}

	// Vertical velocity must be calm and close to rest
	_, _, vz := chassis.GetVelocity()
	if math.Abs(vz) > 0.5 {
		t.Fatalf("chassis vertical velocity not settled: vz = %.4f", vz)
	}
}

func TestVehicleDriveAndBraking(t *testing.T) {
	rigid_body.SetFps(60)
	defer rigid_body.SetFps(120)

	chassis := newTestVehicleChassis(1500.0, 0, 0, 0.6, 1.8, 4.2, 1.4)
	veh := NewRaycastVehicle(chassis, flatFloorRaycast)
	veh.SetupDefault(0.9, 1.4, 1.4, -0.2)

	dt := 1.0 / 60.0
	chassis.SetDt(dt)

	// 1. Settle on ground for 30 ticks
	for step := 0; step < 30; step++ {
		veh.Update(dt)
		stepTestChassis(chassis)
	}

	// 2. Full throttle forward along Y axis for 60 ticks (1 second)
	veh.SetThrottle(1.0)
	for step := 0; step < 60; step++ {
		veh.Update(dt)
		stepTestChassis(chassis)
	}

	_, vyAccel, _ := chassis.GetVelocity()
	if vyAccel < 5.0 {
		t.Fatalf("vehicle failed to accelerate forward: vy = %.2f m/s", vyAccel)
	}

	// 3. Full braking for 60 ticks
	veh.SetThrottle(0.0)
	veh.SetBrake(1.0)
	for step := 0; step < 60; step++ {
		veh.Update(dt)
		stepTestChassis(chassis)
	}

	_, vyBrake, _ := chassis.GetVelocity()
	if vyBrake >= vyAccel*0.4 {
		t.Fatalf("vehicle failed to brake effectively: vy before=%.2f, after=%.2f", vyAccel, vyBrake)
	}
}

func TestVehicleSteeringAndCornering(t *testing.T) {
	rigid_body.SetFps(60)
	defer rigid_body.SetFps(120)

	chassis := newTestVehicleChassis(1500.0, 0, 0, 0.6, 1.8, 4.2, 1.4)
	veh := NewRaycastVehicle(chassis, flatFloorRaycast)
	veh.SetupDefault(0.9, 1.4, 1.4, -0.2)

	dt := 1.0 / 60.0
	chassis.SetDt(dt)

	// Settle and accelerate forward
	veh.SetThrottle(1.0)
	for step := 0; step < 60; step++ {
		veh.Update(dt)
		stepTestChassis(chassis)
	}

	// Apply left steer while driving
	veh.SetSteer(1.0)
	for step := 0; step < 40; step++ {
		veh.Update(dt)
		stepTestChassis(chassis)
	}

	// Vehicle must develop yaw rate (turning around Z axis)
	_, _, wz := chassis.GetAngularVelocity()
	if math.Abs(wz) < 0.15 {
		t.Fatalf("vehicle failed to generate cornering yaw response: wz = %.4f", wz)
	}
}

func TestVehicleHandbrakeDrift(t *testing.T) {
	rigid_body.SetFps(60)
	defer rigid_body.SetFps(120)

	chassis := newTestVehicleChassis(1500.0, 0, 0, 0.6, 1.8, 4.2, 1.4)
	veh := NewRaycastVehicle(chassis, flatFloorRaycast)
	veh.SetupDefault(0.9, 1.4, 1.4, -0.2)

	dt := 1.0 / 60.0
	chassis.SetDt(dt)

	// Accelerate to high speed
	veh.SetThrottle(1.0)
	for step := 0; step < 80; step++ {
		veh.Update(dt)
		stepTestChassis(chassis)
	}

	// Turn and rip handbrake!
	veh.SetSteer(1.0)
	veh.SetHandbrake(true)
	for step := 0; step < 20; step++ {
		veh.Update(dt)
		stepTestChassis(chassis)
	}

	// Rear wheels must have high skid factor indicating drift
	rearLeft := veh.GetWheel(2)
	rearRight := veh.GetWheel(3)
	if rearLeft.SkidFactor < 0.5 && rearRight.SkidFactor < 0.5 {
		t.Fatalf("expected rear wheels to break traction during handbrake drift: left=%.2f, right=%.2f",
			rearLeft.SkidFactor, rearRight.SkidFactor)
	}
}

func TestVehicleZeroHeapAllocations(t *testing.T) {
	chassis := newTestVehicleChassis(1500.0, 0, 0, 0.6, 1.8, 4.2, 1.4)
	veh := NewRaycastVehicle(chassis, flatFloorRaycast)
	veh.SetupDefault(0.9, 1.4, 1.4, -0.2)

	dt := 1.0 / 60.0
	veh.SetThrottle(0.8)
	veh.SetSteer(0.5)

	allocs := testing.AllocsPerRun(100, func() {
		veh.Update(dt)
	})
	if allocs != 0 {
		t.Fatalf("expected 0 heap allocs in vehicle Update, got %v", allocs)
	}
}
