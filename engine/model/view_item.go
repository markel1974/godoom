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
	Where          XYZ
	AngleSin       float64
	AngleCos       float64
	Yaw            float64
	Sector         *Sector
	Zoom           float64
	LightIntensity float64
}

// NewViewItem creates and returns a new instance of ViewItem with default values.
func NewViewItem() *ViewItem {
	return &ViewItem{}
}

func (vi *ViewItem) Translate(x float64, y float64, z float64) (float64, float64, float64, float64, float64) {
	// 1. Traslazione rispetto al ViewItem
	lx := x - vi.Where.X
	ly := y - vi.Where.Y

	// 2. Rotazione nel View Space
	tx := (lx * vi.AngleSin) - (ly * vi.AngleCos)
	ty := z - vi.Where.Z
	tz := (lx * vi.AngleCos) + (ly * vi.AngleSin)

	return lx, ly, tx, ty, tz
}
