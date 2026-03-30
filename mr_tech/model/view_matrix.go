package model

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
	where          XYZ
	angleSin       float64
	angleCos       float64
	yaw            float64
	lightIntensity float64
	sector         *Sector
	bobPhase       float64
}

// NewViewMatrix creates and returns a new instance of ViewMatrix with default values.
func NewViewMatrix() *ViewMatrix {
	return &ViewMatrix{}
}

// Update updates the ViewMatrix's position, orientation, sector, and lighting based on the given ThingPlayer's state.
func (vi *ViewMatrix) Update(player *ThingPlayer) {
	vi.angleSin, vi.angleCos = player.GetAngle()
	vi.sector = player.GetSector()
	vi.where.X, vi.where.Y, vi.where.Z = player.GetXYZ()
	vi.yaw = player.GetYaw()
	vi.lightIntensity = 0.0 //player.GetLightIntensity()
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
func (vi *ViewMatrix) GetLightIntensityFactor(factor float64) float64 {
	l := 1 - (factor * vi.lightIntensity)
	if l < 0 {
		return 0
	}
	return l
}

// GetSector retrieves the Sector instance associated with the ViewMatrix.
func (vi *ViewMatrix) GetSector() *Sector {
	return vi.sector
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
