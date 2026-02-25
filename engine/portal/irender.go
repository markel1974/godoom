package portal

import "github.com/markel1974/godoom/pixels"

type IRender interface {
	Render(surface *pixels.PictureRGBA, vi *viewItem, css []*CompiledSector, compiled int)

	DebugMoveSector(forward bool)

	DebugMoveSectorToggle()
}
