package model

const (
	HFov = 0.73
	VFov = 0.2

	NearZ    = 1e-4
	NearSide = 1e-5
	FarZ     = 5.0
	FarSide  = 20.0
)

// ViewItem represents the state of the current viewing perspective, including position, angles, and rendering parameters.
type ViewItem struct {
	Where         XYZ
	AngleSin      float64
	AngleCos      float64
	Yaw           float64
	Sector        *Sector
	Zoom          float64
	LightDistance float64
}

// NewViewItem creates and returns a new instance of ViewItem with default values.
func NewViewItem() *ViewItem {
	return &ViewItem{}
}
