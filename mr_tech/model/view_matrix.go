package model

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/model/geometry"
	"github.com/markel1974/godoom/mr_tech/physics"
)

// HFov represents the horizontal field of view in radians.
// VFov represents the vertical field of view in radians.
// NearZ defines the near clipping plane distance in a 3D view.
const (
	HFov  = 0.73
	VFov  = 0.2
	NearZ = 1e-4
)

// ViewMatrix represents a view configuration with position, orientation, zoom level, and lighting intensity for rendering.
type ViewMatrix struct {
	view            geometry.XYZ
	angleSin        float64
	angleCos        float64
	angle           float64
	roll            float64
	pitch           float64
	lightIntensity  float64
	volume          *Volume
	swayX           float64
	swayY           float64
	swaySensitivity float64
	front           *physics.Frustum
	rear            *physics.Frustum
}

// NewViewMatrix creates and returns a new instance of ViewMatrix with default values.
func NewViewMatrix() *ViewMatrix {
	return &ViewMatrix{
		front: physics.NewFrustum(),
		rear:  physics.NewFrustum(),
	}
}

// Update updates the ViewMatrix's position, orientation, sector, and lighting based on the given ThingPlayer's state.
func (vi *ViewMatrix) Update(player *ThingPlayer) {
	vi.angleSin, vi.angleCos = player.GetAngleFull()
	vi.volume = player.GetLocation()
	vi.view.X, vi.view.Y, vi.view.Z = player.GetVisualPosition()
	vi.pitch = player.GetPitch()
	vi.angle = player.GetAngle()
	vi.roll = player.GetTilt()
	vi.lightIntensity = player.GetLightIntensity()
	vi.swayX, vi.swayY, vi.swaySensitivity = player.GetSway()
}

// TranslateXY applies a translation and rotation to a given (x, y) point relative to the ViewMatrix's position and orientation.
// It returns the translated local coordinates (lx, ly) and the transformed coordinates (tx, tz) in the view space.
func (vi *ViewMatrix) TranslateXY(x float64, y float64) (float64, float64, float64, float64) {
	// 1. Translation relative to the ViewMatrix
	lx := x - vi.view.X
	ly := y - vi.view.Y
	// 2. Rotation in View Space
	tx := (lx * vi.angleSin) - (ly * vi.angleCos)
	tz := (lx * vi.angleCos) + (ly * vi.angleSin)
	return lx, ly, tx, tz
}

// GetView retrieves the X, Y, and Z coordinates of the ViewMatrix's position as a tuple of three float64 values.
func (vi *ViewMatrix) GetView() (float64, float64, float64) {
	return vi.view.X, vi.view.Y, vi.view.Z
}

// GetAngleFull returns the sine and cosine of the ViewMatrix's angle as two float64 values.
func (vi *ViewMatrix) GetAngleFull() (float64, float64) {
	return vi.angleSin, vi.angleCos
}

// GetAngle retrieves the current yaw angle (orientation) of the ViewMatrix in radians.
func (vi *ViewMatrix) GetAngle() float64 {
	return vi.angle
}

// GetRoll returns the roll angle of the ViewMatrix, representing its sideways tilt in radians.
func (vi *ViewMatrix) GetRoll() float64 {
	return vi.roll
}

// GetPitch retrieves the pitch value of the ViewMatrix, representing its vertical orientation angle in radians.
func (vi *ViewMatrix) GetPitch() float64 {
	return vi.pitch
}

// GetLightIntensity retrieves the light intensity property of the ViewMatrix as a float64 value.
func (vi *ViewMatrix) GetLightIntensity() float64 {
	return vi.lightIntensity
}

// GetVolume retrieves the Volume instance associated with the ViewMatrix.
func (vi *ViewMatrix) GetVolume() *Volume {
	return vi.volume
}

// ZDistance calculates the distance between the given value and the Z-coordinate of the ViewMatrix's position.
func (vi *ViewMatrix) ZDistance(v float64) float64 {
	return v - vi.view.Z
}

// GetSway returns the horizontal and vertical sway values of the ViewMatrix as two float64 values.
func (vi *ViewMatrix) GetSway() (float64, float64, float64) {
	return vi.swayX, vi.swayY, vi.swaySensitivity
}

func (vi *ViewMatrix) GetForwardVector() (float32, float32, float32) {
	// Calcoliamo il seno e coseno del pitch (inclinazione verticale)
	pitchCos := math.Cos(vi.pitch)
	pitchSin := math.Sin(vi.pitch)

	// Conversione da coordinate sferiche a cartesiane 3D.
	// Assumendo che Y sia l'asse verticale (Up) nel tuo shader:
	dirX := pitchCos * vi.angleCos
	dirY := pitchSin
	dirZ := pitchCos * vi.angleSin

	// Nota: a seconda se il tuo engine è Right-Handed o Left-Handed,
	// potresti dover invertire il segno di uno degli assi (es. -dirZ o -dirX)
	// per allinearlo perfettamente con il Frustum.
	return float32(dirX), float32(dirY), float32(dirZ)
}

// GetFrustum computes and returns the front and rear frustums of the ViewMatrix based on the provided matrices.
func (vi *ViewMatrix) GetFrustum(f [16]float32, r [16]float32) (*physics.Frustum, *physics.Frustum) {
	return vi.GetFrontFrustum(f), vi.GetRearFrustum(r)
}

// GetFrontFrustum computes and returns the front frustum based on the given 4x4 view-projection matrix.
func (vi *ViewMatrix) GetFrontFrustum(f [16]float32) *physics.Frustum {
	vi.front.Rebuild(f)
	return vi.front
}

// GetRearFrustum rebuilds and returns the rear frustum using the provided 4x4 column-major matrix.
func (vi *ViewMatrix) GetRearFrustum(f [16]float32) *physics.Frustum {
	vi.rear.Rebuild(f)
	return vi.rear
}
