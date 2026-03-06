package renderers

import (
	"github.com/markel1974/godoom/engine/model"
	"github.com/markel1974/godoom/engine/portal"
)

// IRender defines the contract for rendering systems, including setup, rendering logic, and debug controls.
type IRender interface {
	Setup(portal *portal.Portal, player *portal.Player) error

	Start()

	Render(vi *model.ViewItem, css []*model.CompiledSector, compiled int)

	RenderSector(sector *model.Sector)

	DebugMoveSector(forward bool)

	DebugMoveSectorToggle()
}
