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
	Where          XYZ
	AngleSin       float64
	AngleCos       float64
	Yaw            float64
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
	lx := x - vi.Where.X
	ly := y - vi.Where.Y
	// 2. Rotation in View Space
	tx := (lx * vi.AngleSin) - (ly * vi.AngleCos)
	tz := (lx * vi.AngleCos) + (ly * vi.AngleSin)
	return lx, ly, tx, tz
}
