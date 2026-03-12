package model

// HFov represents the horizontal field of view in radians.
// VFov represents the vertical field of view in radians.
// NearZ defines the near clipping plane distance in a 3D view.
// NearSide defines the minimum side clipping distance in a 3D view.
// FarZ defines the far clipping plane distance in a 3D view.
// FarSide defines the maximum side clipping distance in a 3D view.
const (
	HFov = 0.73
	VFov = 0.2

	NearZ = 1e-4
	//NearSide = 1e-5
	//FarZ     = 5.0
	//FarSide  = 20.0
)

// ViewItem represents a view configuration with position, orientation, zoom level, and lighting intensity for rendering.
type ViewItem struct {
	where          XYZ
	angleSin       float64
	angleCos       float64
	yaw            float64
	lightIntensity float64
	sector         *Sector
}

// NewViewItem creates and returns a new instance of ViewItem with default values.
func NewViewItem() *ViewItem {
	return &ViewItem{}
}

// Compute updates the ViewItem's position, orientation, sector, and lighting based on the given Player's state.
func (vi *ViewItem) Compute(player *Player) {
	vi.angleSin, vi.angleCos = player.GetAngle()
	vi.sector = player.GetSector()
	vi.where.X, vi.where.Y, vi.where.Z = player.GetXYZ()
	vi.yaw = player.GetYaw()
	vi.lightIntensity = player.GetLightIntensity()
}

// TranslateXY applies a translation and rotation to a given (x, y) point relative to the ViewItem's position and orientation.
// It returns the translated local coordinates (lx, ly) and the transformed coordinates (tx, tz) in the view space.
func (vi *ViewItem) TranslateXY(x float64, y float64) (float64, float64, float64, float64) {
	// 1. Translation relative to the ViewItem
	lx := x - vi.where.X
	ly := y - vi.where.Y
	// 2. Rotation in View Space
	tx := (lx * vi.angleSin) - (ly * vi.angleCos)
	tz := (lx * vi.angleCos) + (ly * vi.angleSin)
	return lx, ly, tx, tz
}

// GetXY retrieves the X and Y coordinates from the ViewItem's position.
func (vi *ViewItem) GetXY() (float64, float64) {
	return vi.where.X, vi.where.Y
}

// GetXYZ retrieves the X, Y, and Z coordinates of the ViewItem's position as a tuple of three float64 values.
func (vi *ViewItem) GetXYZ() (float64, float64, float64) {
	return vi.where.X, vi.where.Y, vi.where.Z
}

// GetZ retrieves the Z-coordinate of the ViewItem's position.
func (vi *ViewItem) GetZ() float64 {
	return vi.where.Z
}

// GetAngle returns the sine and cosine of the ViewItem's rotation angle for transformations and calculations.
func (vi *ViewItem) GetAngle() (float64, float64) {
	return vi.angleSin, vi.angleCos
}

// GetYaw returns the yaw angle of the ViewItem as a float64 value.
func (vi *ViewItem) GetYaw() float64 {
	return vi.yaw
}

// GetLightIntensity retrieves the light intensity property of the ViewItem as a float64 value.
func (vi *ViewItem) GetLightIntensity() float64 {
	return vi.lightIntensity
}

// GetSector retrieves the Sector instance associated with the ViewItem.
func (vi *ViewItem) GetSector() *Sector {
	return vi.sector
}

// ComputeYaw calculates a new yaw value based on the input parameters and the ViewItem's current yaw attribute.
func (vi *ViewItem) ComputeYaw(y float64, z float64) float64 {
	return y + (z * vi.yaw)
}

// ZDistance calculates the distance between the given value and the Z-coordinate of the ViewItem's position.
func (vi *ViewItem) ZDistance(v float64) float64 {
	return v - vi.where.Z
}
