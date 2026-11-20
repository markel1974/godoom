package engine

import (
	"math"
	"testing"

	"github.com/markel1974/godoom/mr_tech/geometry"
)

type mockRaycaster struct {
	wallDist float64
	hit      bool
}

func (m *mockRaycaster) RaycastSolid(fromX, fromY, fromZ, dirX, dirY, dirZ, maxDist float64) (hitDist float64, nx, ny, nz float64, hit bool) {
	if m.hit && m.wallDist <= maxDist {
		return m.wallDist, 0, 0, 1, true
	}
	return 0, 0, 0, 1, false
}

func TestSpringArmOpenSpace(t *testing.T) {
	cfg := DefaultThirdPersonConfig()
	cfg.TargetArmLength = 100.0
	cfg.PositionLagSpeed = 0.0 // Instantaneous for testing
	cfg.RotationLagSpeed = 0.0
	cfg.SocketOffset = geometry.XYZ{X: 0, Y: 0, Z: 0}
	cfg.TargetOffset = geometry.XYZ{X: 0, Y: 0, Z: 0}

	arm := NewSpringLimb(cfg)
	mock := &mockRaycaster{hit: false}

	// Player at (0, 0, 0) facing +X (yaw = 0, pitch = 0)
	camX, camY, camZ, yaw, pitch := arm.Update(0.016, 0, 0, 0, 0, 0, 0, mock)

	// Since facing +X, backward boom goes along -X. Expected camX = -100.0
	if math.Abs(camX-(-100.0)) > 1e-4 {
		t.Fatalf("expected camX = -100.0, got %f", camX)
	}
	if math.Abs(camY) > 1e-4 || math.Abs(camZ) > 1e-4 {
		t.Fatalf("expected camY=0, camZ=0, got (%f, %f)", camY, camZ)
	}
	if yaw != 0.0 || pitch != 0.0 {
		t.Fatalf("expected yaw=0, pitch=0")
	}
	if arm.IsHitWall() {
		t.Fatalf("expected IsHitWall() to be false")
	}
}

func TestSpringArmAntiClipRetraction(t *testing.T) {
	cfg := DefaultThirdPersonConfig()
	cfg.TargetArmLength = 100.0
	cfg.MinArmLength = 15.0
	cfg.ProbeMargin = 5.0
	cfg.PositionLagSpeed = 10.0 // Lag is enabled, but retraction MUST be instant!
	cfg.SocketOffset = geometry.XYZ{X: 0, Y: 0, Z: 0}
	cfg.TargetOffset = geometry.XYZ{X: 0, Y: 0, Z: 0}

	arm := NewSpringLimb(cfg)

	// Wall behind player at distance 40.0 along -X
	mock := &mockRaycaster{
		hit:      true,
		wallDist: 40.0,
	}

	// First tick facing +X
	camX, _, _, _, _ := arm.Update(0.016, 0, 0, 0, 0, 0, 0, mock)

	expectedDist := 40.0 - 5.0 // wallDist - ProbeMargin = 35.0
	expectedCamX := -expectedDist

	if math.Abs(camX-expectedCamX) > 1e-4 {
		t.Fatalf("expected instantaneous anti-clip retraction to %f, got %f", expectedCamX, camX)
	}
	if !arm.IsHitWall() {
		t.Fatalf("expected IsHitWall() to be true when obstructed")
	}
	if math.Abs(arm.GetCurrentDistance()-expectedDist) > 1e-4 {
		t.Fatalf("expected current distance %f, got %f", expectedDist, arm.GetCurrentDistance())
	}

	// When wall clears, arm should smoothly expand back towards 100.0, not jump immediately
	mock.hit = false
	camX2, _, _, _, _ := arm.Update(0.016, 0, 0, 0, 0, 0, 0, mock)
	if camX2 >= -35.0 || camX2 <= -100.0 {
		t.Fatalf("expected smooth expansion between -35 and -100, got %f", camX2)
	}
}

func TestSpringArmVehicleModeSpeedStretch(t *testing.T) {
	cfg := DefaultVehicleConfig()
	cfg.TargetArmLength = 200.0
	cfg.SpeedStretchFactor = 0.5
	cfg.PositionLagSpeed = 0.0
	cfg.RotationLagSpeed = 0.0

	arm := NewSpringLimb(cfg)
	mock := &mockRaycaster{hit: false}

	// At rest (speed = 0)
	arm.Update(0.016, 0, 0, 0, 0, 0, 0, mock)
	if math.Abs(arm.GetCurrentDistance()-200.0) > 1e-4 {
		t.Fatalf("expected rest arm length 200, got %f", arm.GetCurrentDistance())
	}

	// At high speed 100.0 -> desired arm length = 200.0 + 100.0 * 0.5 = 250.0
	arm.Update(0.016, 0, 0, 0, 0, 0, 100.0, mock)
	if math.Abs(arm.GetCurrentDistance()-250.0) > 1e-4 {
		t.Fatalf("expected stretched arm length 250, got %f", arm.GetCurrentDistance())
	}
}

func TestSpringArmZeroAllocs(t *testing.T) {
	cfg := DefaultThirdPersonConfig()
	arm := NewSpringLimb(cfg)
	mock := &mockRaycaster{hit: true, wallDist: 50.0}

	// Warmup
	arm.Update(0.016, 0, 0, 0, 0.5, 0.2, 10.0, mock)

	allocs := testing.AllocsPerRun(100, func() {
		arm.Update(0.016, 0, 0, 0, 0.5, 0.2, 10.0, mock)
	})

	if allocs != 0 {
		t.Fatalf("expected 0 allocations in SpringArm.Update, got %v", allocs)
	}
}
