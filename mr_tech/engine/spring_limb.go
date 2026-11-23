package engine

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/model/things"
)

// SpringLimb simulates an elastic chase boom that prevents the camera from clipping into walls
// while providing smooth cinematic lag behind players and vehicles.
type SpringLimb struct {
	cfg *SpringLimbConfig

	currentDist  float64
	currentPos   geometry.XYZ
	currentYaw   float64
	currentPitch float64
	currentRoll  float64
	initialized  bool
	hitWall      bool
}

// NewSpringLimb creates a new SpringLimb controller initialized with the provided configuration.
func NewSpringLimb(cfg *SpringLimbConfig) *SpringLimb {
	return &SpringLimb{
		cfg:         cfg,
		currentDist: cfg.TargetArmLength,
	}
}

// GetConfig returns a pointer to the current configuration for tuning.
func (sa *SpringLimb) GetConfig() *SpringLimbConfig {
	return sa.cfg
}

// SetConfig updates the configuration parameters.
func (sa *SpringLimb) SetConfig(cfg *SpringLimbConfig) {
	sa.cfg = cfg
}

// SetMode changes the active camera mode.
func (sa *SpringLimb) SetMode(mode CameraMode) {
	sa.cfg.Mode = mode
}

// Reset snaps the camera immediately to the target without applying lag smoothing.
func (sa *SpringLimb) Reset(x, y, z, yaw, pitch, roll float64) {
	sa.currentPos.X = x
	sa.currentPos.Y = y
	sa.currentPos.Z = z
	sa.currentYaw = yaw
	sa.currentPitch = pitch
	sa.currentRoll = roll
	sa.currentDist = sa.cfg.TargetArmLength
	sa.initialized = true
	sa.hitWall = false
}

// IsHitWall reports whether the camera is currently compressed by a solid obstacle.
func (sa *SpringLimb) IsHitWall() bool {
	return sa.hitWall
}

// GetCurrentDistance returns the active distance from target to camera.
func (sa *SpringLimb) GetCurrentDistance() float64 {
	return sa.currentDist
}

// Update calculates the next camera transform using dt, target pivot, orientation, speed, and triangle soup raycasting.
func (sa *SpringLimb) Update(dt float64, targetX, targetY, targetZ float64, targetYaw, targetPitch float64, speed float64, raycaster IRaycaster) (camX, camY, camZ, camYaw, camPitch float64) {
	if sa.cfg.Mode == CameraModeFirstPerson {
		sa.currentPos.X = targetX + sa.cfg.TargetOffset.X
		sa.currentPos.Y = targetY + sa.cfg.TargetOffset.Y
		sa.currentPos.Z = targetZ + sa.cfg.TargetOffset.Z
		sa.currentYaw = targetYaw
		sa.currentPitch = targetPitch
		sa.currentDist = 0.0
		sa.hitWall = false
		return sa.currentPos.X, sa.currentPos.Y, sa.currentPos.Z, sa.currentYaw, sa.currentPitch
	}

	targetPitch = geometry.ClampF(targetPitch, sa.cfg.PitchMin, sa.cfg.PitchMax)

	if !sa.initialized {
		sa.currentYaw = targetYaw
		sa.currentPitch = targetPitch
		sa.currentDist = sa.cfg.TargetArmLength
	}

	// Rotation Lag
	if sa.cfg.RotationLagSpeed > 0.0 && dt > 0.0 && sa.initialized {
		rAlpha := 1.0 - math.Exp(-sa.cfg.RotationLagSpeed*dt)
		if rAlpha > 1.0 {
			rAlpha = 1.0
		}
		sa.currentYaw += ShortestAngleDelta(sa.currentYaw, targetYaw) * rAlpha
		sa.currentPitch += (targetPitch - sa.currentPitch) * rAlpha
	} else {
		sa.currentYaw = targetYaw
		sa.currentPitch = targetPitch
	}

	// Compute View Vectors
	sinYaw := math.Sin(sa.currentYaw)
	cosYaw := math.Cos(sa.currentYaw)
	sinPitch := math.Sin(sa.currentPitch)
	cosPitch := math.Cos(sa.currentPitch)

	fwdX := cosPitch * cosYaw
	fwdY := cosPitch * sinYaw
	fwdZ := sinPitch

	rightX := sinYaw
	rightY := -cosYaw

	upX := -sinPitch * cosYaw
	upY := -sinPitch * sinYaw
	upZ := cosPitch

	// Compute Pivot Start Position
	pivotX := targetX + sa.cfg.TargetOffset.X
	pivotY := targetY + sa.cfg.TargetOffset.Y
	pivotZ := targetZ + sa.cfg.TargetOffset.Z

	// Apply socket offset (right / up)
	startX := pivotX + sa.cfg.SocketOffset.X*rightX + sa.cfg.SocketOffset.Z*upX
	startY := pivotY + sa.cfg.SocketOffset.X*rightY + sa.cfg.SocketOffset.Z*upY
	startZ := pivotZ + sa.cfg.SocketOffset.Z*upZ

	// 4. Desired Arm Length (with speed stretch for vehicles)
	desiredArm := sa.cfg.TargetArmLength
	if sa.cfg.SpeedStretchFactor > 0.0 && speed > 0.0 {
		desiredArm += speed * sa.cfg.SpeedStretchFactor
	}

	// Anti-Clip Raycast towards the Triangle Soup
	// Boom direction points backwards from the forward gaze
	backDirX := -fwdX
	backDirY := -fwdY
	backDirZ := -fwdZ

	blockedDist := desiredArm
	sa.hitWall = false

	if raycaster != nil && desiredArm > 0.0 {
		hitDist, _, _, _, hit := raycaster.RaycastSolid(startX, startY, startZ, backDirX, backDirY, backDirZ, desiredArm)
		if hit {
			safeDist := hitDist - sa.cfg.ProbeMargin
			if safeDist < sa.cfg.MinArmLength {
				safeDist = sa.cfg.MinArmLength
			}
			blockedDist = safeDist
			sa.hitWall = true
		}
	}

	// Distance Spring
	// If obstructed, pull camera in INSTANTLY to guarantee zero clipping
	// When unobstructed, smoothly spring back out to desired arm length
	if blockedDist < sa.currentDist {
		sa.currentDist = blockedDist
	} else if sa.cfg.PositionLagSpeed > 0.0 && dt > 0.0 && sa.initialized {
		dAlpha := 1.0 - math.Exp(-sa.cfg.PositionLagSpeed*dt)
		if dAlpha > 1.0 {
			dAlpha = 1.0
		}
		sa.currentDist += (blockedDist - sa.currentDist) * dAlpha
	} else {
		sa.currentDist = blockedDist
	}

	// Calculate Final Camera Position
	targetCamX := startX + backDirX*sa.currentDist
	targetCamY := startY + backDirY*sa.currentDist
	targetCamZ := startZ + backDirZ*sa.currentDist

	// Position Lag (for smooth car inertia / camera breathing)
	if sa.cfg.PositionLagSpeed > 0.0 && dt > 0.0 && sa.initialized && !sa.hitWall {
		pAlpha := 1.0 - math.Exp(-sa.cfg.PositionLagSpeed*dt)
		if pAlpha > 1.0 {
			pAlpha = 1.0
		}
		sa.currentPos.X += (targetCamX - sa.currentPos.X) * pAlpha
		sa.currentPos.Y += (targetCamY - sa.currentPos.Y) * pAlpha
		sa.currentPos.Z += (targetCamZ - sa.currentPos.Z) * pAlpha
	} else {
		sa.currentPos.X = targetCamX
		sa.currentPos.Y = targetCamY
		sa.currentPos.Z = targetCamZ
	}

	sa.initialized = true
	return sa.currentPos.X, sa.currentPos.Y, sa.currentPos.Z, sa.currentYaw, sa.currentPitch
}

// UpdateFromPlayer updates the spring arm using a Player instance's transform and speed.
func (sa *SpringLimb) UpdateFromPlayer(dt float64, player *things.Player, raycaster IRaycaster) (camX, camY, camZ, camYaw, camPitch float64) {
	if player == nil {
		return sa.currentPos.X, sa.currentPos.Y, sa.currentPos.Z, sa.currentYaw, sa.currentPitch
	}
	ent := player.GetEntity()
	pX, pY, pZ := ent.GetCenter()
	// Place target origin at player feet
	pZ -= ent.GetDepth() * 0.5
	yaw := player.GetAngle()
	pitch := player.GetPitch()
	vx, vy, vz := ent.GetVelocity()
	speed := math.Sqrt(vx*vx + vy*vy + vz*vz)

	return sa.Update(dt, pX, pY, pZ, yaw, pitch, speed, raycaster)
}

// UpdateFromVehicle updates the spring arm using a vehicle chassis transform, heading, and forward speed.
func (sa *SpringLimb) UpdateFromVehicle(dt float64, vehicleCenter geometry.XYZ, vehicleYaw, vehiclePitch, vehicleRoll float64, speed float64, freeLookYaw, freeLookPitch float64, useFreeLook bool, raycaster IRaycaster) (camX, camY, camZ, camYaw, camPitch float64) {
	yaw := vehicleYaw
	pitch := vehiclePitch
	if useFreeLook {
		yaw = freeLookYaw
		pitch = freeLookPitch
	}
	return sa.Update(dt, vehicleCenter.X, vehicleCenter.Y, vehicleCenter.Z, yaw, pitch, speed, raycaster)
}

// ApplyToViewMatrix copies the calculated camera coordinates and angles directly into the ViewMatrix.
func (sa *SpringLimb) ApplyToViewMatrix(vi *ViewMatrix) {
	if vi == nil {
		return
	}
	vi.view.X = sa.currentPos.X
	vi.view.Y = sa.currentPos.Y
	vi.view.Z = sa.currentPos.Z
	vi.angle = sa.currentYaw
	vi.pitch = sa.currentPitch
	vi.roll = sa.currentRoll
	vi.angleSin = math.Sin(sa.currentYaw)
	vi.angleCos = math.Cos(sa.currentYaw)
}
