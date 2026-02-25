package renderers

import "github.com/markel1974/godoom/engine/model"

// ViewItem represents a rendered view configuration within a Sector, including position, angles, Zoom, and related details.
type ViewItem struct {
	Where    model.XYZ
	AngleSin float64
	AngleCos float64
	Yaw      float64
	Sector   *model.Sector
	Zoom     float64
}
