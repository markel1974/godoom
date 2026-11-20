package engine

import "github.com/markel1974/godoom/mr_tech/geometry"

// CameraMode specifies the operational behavior of the camera controller.
type CameraMode int

const (
	// CameraModeFirstPerson positions the camera exactly at the entity's visual eye coordinate.
	CameraModeFirstPerson CameraMode = iota
	// CameraModeThirdPerson orbits behind the character (over-the-shoulder) with anti-clip raycasting.
	CameraModeThirdPerson
	// CameraModeVehicle follows behind a vehicle chassis with inertia, speed-stretch, and anti-clip.
	CameraModeVehicle
)

// SpringLimbConfig holds all tuning parameters for camera distance, offsets, lag, and clipping margins.
type SpringLimbConfig struct {
	Mode               CameraMode
	TargetArmLength    float64      // Rest distance behind the target
	MinArmLength       float64      // Minimum distance to shrink to when obstructed by walls
	ProbeMargin        float64      // Safety margin away from hit triangle surface
	TargetOffset       geometry.XYZ // World/local offset from the target origin (e.g. eye height)
	SocketOffset       geometry.XYZ // Camera offset relative to orientation (e.g. over right shoulder)
	PositionLagSpeed   float64      // Exponential decay rate for position smoothing (0 = instant)
	RotationLagSpeed   float64      // Exponential decay rate for yaw/pitch smoothing (0 = instant)
	SpeedStretchFactor float64      // Extra arm length added per unit of forward vehicle speed
	PitchMin           float64      // Minimum pitch angle in radians (looking down/up clamp)
	PitchMax           float64      // Maximum pitch angle in radians
}

// DefaultThirdPersonConfig returns standard over-the-shoulder third-person camera settings.
func DefaultThirdPersonConfig() SpringLimbConfig {
	return SpringLimbConfig{
		Mode:               CameraModeThirdPerson,
		TargetArmLength:    100.0,
		MinArmLength:       15.0,
		ProbeMargin:        5.0,
		TargetOffset:       geometry.XYZ{X: 0, Y: 0, Z: 40.0},
		SocketOffset:       geometry.XYZ{X: 12.0, Y: 0, Z: 0.0},
		PositionLagSpeed:   12.0,
		RotationLagSpeed:   10.0,
		SpeedStretchFactor: 0.0,
		PitchMin:           -1.2,
		PitchMax:           1.2,
	}
}

// DefaultVehicleConfig returns standard third-person vehicle chase camera settings.
func DefaultVehicleConfig() SpringLimbConfig {
	return SpringLimbConfig{
		Mode:               CameraModeVehicle,
		TargetArmLength:    220.0,
		MinArmLength:       30.0,
		ProbeMargin:        8.0,
		TargetOffset:       geometry.XYZ{X: 0, Y: 0, Z: 35.0},
		SocketOffset:       geometry.XYZ{X: 0, Y: 0, Z: 0.0},
		PositionLagSpeed:   8.0,
		RotationLagSpeed:   6.0,
		SpeedStretchFactor: 0.15,
		PitchMin:           -0.6,
		PitchMax:           0.8,
	}
}
