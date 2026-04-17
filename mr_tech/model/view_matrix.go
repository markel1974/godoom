package model

import (
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
	where          geometry.XYZ
	angleSin       float64
	angleCos       float64
	yaw            float64
	lightIntensity float64
	volume         *Volume
	bobPhase       float64
	front          *physics.Frustum
	rear           *physics.Frustum
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
	vi.angleSin, vi.angleCos = player.GetAngle()
	vi.volume = player.GetLocation()
	vi.where.X, vi.where.Y, vi.where.Z = player.GetPosition()
	vi.yaw = player.GetYaw()
	vi.lightIntensity = player.GetLightIntensity()
	bob, bobPhase := player.GetBobPhase()
	vi.bobPhase = bobPhase
	vi.where.Z += bob
}

// TranslateXY applies a translation and rotation to a given (x, y) point relative to the ViewMatrix's position and orientation.
// It returns the translated local coordinates (lx, ly) and the transformed coordinates (tx, tz) in the view space.
func (vi *ViewMatrix) TranslateXY(x float64, y float64) (float64, float64, float64, float64) {
	// 1. Translation relative to the ViewMatrix
	lx := x - vi.where.X
	ly := y - vi.where.Y
	// 2. Rotation in View Space
	tx := (lx * vi.angleSin) - (ly * vi.angleCos)
	tz := (lx * vi.angleCos) + (ly * vi.angleSin)
	return lx, ly, tx, tz
}

// GetXY retrieves the X and Y coordinates from the ViewMatrix's position.
func (vi *ViewMatrix) GetXY() (float64, float64) {
	return vi.where.X, vi.where.Y
}

// GetXYZ retrieves the X, Y, and Z coordinates of the ViewMatrix's position as a tuple of three float64 values.
func (vi *ViewMatrix) GetXYZ() (float64, float64, float64) {
	return vi.where.X, vi.where.Y, vi.where.Z
}

// GetZ retrieves the Z-coordinate of the ViewMatrix's position.
func (vi *ViewMatrix) GetZ() float64 {
	return vi.where.Z
}

// GetAngle returns the sine and cosine of the ViewMatrix's rotation angle for transformations and calculations.
func (vi *ViewMatrix) GetAngle() (float64, float64) {
	return vi.angleSin, vi.angleCos
}

// GetYaw returns the yaw angle of the ViewMatrix as a float64 value.
func (vi *ViewMatrix) GetYaw() float64 {
	return vi.yaw
}

// GetLightIntensity retrieves the light intensity property of the ViewMatrix as a float64 value.
func (vi *ViewMatrix) GetLightIntensity() float64 {
	return vi.lightIntensity
}

// GetLightIntensityFactor computes the adjusted light intensity factor by reducing 1 with the product of input factor and light intensity.
//func (vi *ViewMatrix) GetLightIntensityFactor(factor float64) float64 {
//	l := 1 - (factor * vi.lightIntensity)
//	if l < 0 {
//		return 0
//	}
//	return l
//}

// GetVolume retrieves the Volume instance associated with the ViewMatrix.
func (vi *ViewMatrix) GetVolume() *Volume {
	return vi.volume
}

// ComputeYaw calculates a new yaw value based on the input parameters and the ViewMatrix's current yaw attribute.
func (vi *ViewMatrix) ComputeYaw(y float64, z float64) float64 {
	return y + (z * vi.yaw)
}

// ZDistance calculates the distance between the given value and the Z-coordinate of the ViewMatrix's position.
func (vi *ViewMatrix) ZDistance(v float64) float64 {
	return v - vi.where.Z
}

// GetBobPhase returns the current bob phase value of the ViewMatrix as a float64 for use in animation or rendering calculations.
func (vi *ViewMatrix) GetBobPhase() float64 {
	return vi.bobPhase
}

// GetFrustum computes and returns the front and rear frustums using the given framebuffer dimensions and far plane distance.
func (vi *ViewMatrix) GetFrustum(fbw, fbh int32, zFarRoom float32) (*physics.Frustum, *physics.Frustum) {
	return vi.GetFrontFrustum(fbw, fbh, zFarRoom), vi.GetRearFrustum(fbw, fbh, zFarRoom)
}

// GetFrontFrustum calculates and returns the camera's visual frustum based on the current ViewMatrix properties.
// It reconstructs the View and Projection matrices to accurately extract the 6 clipping planes for AABB culling.
func (vi *ViewMatrix) GetFrontFrustum(fbw, fbh int32, zFarRoom float32) *physics.Frustum {
	// Calcolo Aspect Ratio e Scale per la Proiezione
	pitchShear := float32(-vi.yaw)
	aspect := float32(fbw) / float32(fbh)
	scaleX := (2.0 / aspect) * float32(HFov)
	scaleY := 2.0 * float32(VFov)
	zNear := float32(0.1)
	zFar := zFarRoom
	if zFar <= zNear {
		zFar = 10000.0 // Fallback di sicurezza se zFarRoom non è valido
	}
	// Costruzione Matrice di Proiezione (Column-Major)
	proj := [16]float32{
		-scaleX, 0, 0, 0,
		0, scaleY, 0, 0,
		0, pitchShear, (zFar + zNear) / (zNear - zFar), -1,
		0, 0, (2 * zFar * zNear) / (zNear - zFar), 0,
	}
	// Costruzione Matrice di View (Column-Major)
	fX, fZ := float32(vi.angleCos), float32(-vi.angleSin)
	rX, rZ := float32(vi.angleSin), float32(vi.angleCos)
	ex, ey, ez := float32(vi.where.X), float32(vi.where.Z), float32(-vi.where.Y)
	tx := -(rX*ex + rZ*ez)
	ty := -ey
	tz := fX*ex + fZ*ez
	view := [16]float32{
		rX, 0, -fX, 0,
		0, 1, 0, 0,
		rZ, 0, -fZ, 0,
		tx, ty, tz, 1,
	}
	vp := matrixMultiply(proj, view)
	vi.front.Rebuild(vp)
	return vi.front
}

// GetRearFrustum calcola il frustum posteriore invertendo i vettori direzionali.
func (vi *ViewMatrix) GetRearFrustum(fbw, fbh int32, zFarRoom float32) *physics.Frustum {
	aspect := float32(fbw) / float32(fbh)
	scaleX := (2.0 / aspect) * float32(HFov)
	scaleY := 2.0 * float32(VFov)
	// Inversione del pitch (se guardi in alto, dietro guardi in basso)
	pitchShear := float32(vi.yaw)
	zNear := float32(0.1)
	zFar := zFarRoom
	if zFar <= zNear {
		zFar = 10000.0
	}
	proj := [16]float32{
		-scaleX, 0, 0, 0,
		0, scaleY, 0, 0,
		0, pitchShear, (zFar + zNear) / (zNear - zFar), -1,
		0, 0, (2 * zFar * zNear) / (zNear - zFar), 0,
	}
	// Inversione vettori Forward e Right
	fX, fZ := float32(-vi.angleCos), float32(vi.angleSin)
	rX, rZ := float32(-vi.angleSin), float32(-vi.angleCos)
	ex, ey, ez := float32(vi.where.X), float32(vi.where.Z), float32(-vi.where.Y)
	tx := -(rX*ex + rZ*ez)
	ty := -ey
	tz := fX*ex + fZ*ez
	view := [16]float32{
		rX, 0, -fX, 0,
		0, 1, 0, 0,
		rZ, 0, -fZ, 0,
		tx, ty, tz, 1,
	}
	vp := matrixMultiply(proj, view)
	vi.rear.Rebuild(vp)
	return vi.rear
}

// matrixMultiply computes the product of two 4x4 matrices stored in column-major order and returns the resulting matrix.
func matrixMultiply(proj, view [16]float32) [16]float32 {
	var vp [16]float32
	for col := 0; col < 4; col++ {
		for row := 0; row < 4; row++ {
			sum := float32(0.0)
			for i := 0; i < 4; i++ {
				sum += proj[i*4+row] * view[col*4+i]
			}
			vp[col*4+row] = sum
		}
	}
	return vp
}
