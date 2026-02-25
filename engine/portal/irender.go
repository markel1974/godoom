package portal

import (
	"github.com/markel1974/godoom/engine/model"
	"github.com/markel1974/godoom/engine/renderers"
	"github.com/markel1974/godoom/pixels"
)

// IRender defines methods for rendering and debugging sectors within a graphical environment.
type IRender interface {
	Render(surface *pixels.PictureRGBA, vi *renderers.ViewItem, css []*model.CompiledSector, compiled int)

	DebugMoveSector(forward bool)

	DebugMoveSectorToggle()
}
