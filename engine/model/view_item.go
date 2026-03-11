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
	Zoom           float64
	LightIntensity float64
	Sector         *Sector
}

// NewViewItem creates and returns a new instance of ViewItem with default values.
func NewViewItem() *ViewItem {
	return &ViewItem{}
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

// SetXY updates the X and Y coordinates of the ViewItem's position.
func (vi *ViewItem) SetXY(x float64, y float64) {
	vi.where.X, vi.where.Y = x, y
}

// SetXYZ updates the x, y, and z coordinates of the ViewItem's position in the 3D space.
func (vi *ViewItem) SetXYZ(x float64, y float64, z float64) {
	vi.where.X, vi.where.Y, vi.where.Z = x, y, z
}

// GetZ retrieves the Z-coordinate of the ViewItem's position.
func (vi *ViewItem) GetZ() float64 {
	return vi.where.Z
}

// SetZ updates the Z-coordinate of the ViewItem's position to the specified value.
func (vi *ViewItem) SetZ(z float64) {
	vi.where.Z = z
}

func (vi *ViewItem) GetAngle() (float64, float64) {
	return vi.angleSin, vi.angleCos
}

func (vi *ViewItem) SetAngle(sin float64, cos float64) {
	vi.angleSin = sin
	vi.angleCos = cos
}

func (vi *ViewItem) GetYaw() float64 {
	return vi.yaw
}

func (vi *ViewItem) SetYaw(yaw float64) {
	vi.yaw = yaw
}
