package engine

import (
	"github.com/markel1974/godoom/mr_tech/model"
	"github.com/markel1974/godoom/mr_tech/portal"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// Engine represents the core game engine handling rendering, player interactions, and environment configuration.
type Engine struct {
	portal           *portal.Portal
	textures         textures.ITextures
	w                int
	h                int
	maxQueue         int
	things           []model.IThing
	thingsDict       map[string]model.IThing
	sectorsMaxHeight float64
	entities         *model.Entities
	player           *model.Player
	sectorTree       *model.Sectors
}

// NewEngine initializes and returns a new Engine instance with specified width, height, and maximum render queue size.
func NewEngine(w int, h int, maxQueue int) *Engine {
	return &Engine{
		portal:           nil,
		w:                w,
		h:                h,
		maxQueue:         maxQueue,
		things:           nil,
		sectorsMaxHeight: 0,
		entities:         nil,
		sectorTree:       nil,
		player:           nil,
		thingsDict:       make(map[string]model.IThing),
	}
}

// GetPlayer returns the current player instance associated with the engine.
func (e *Engine) GetPlayer() *model.Player {
	return e.player
}

// GetTextures retrieves the ITextures implementation associated with the engine.
func (e *Engine) GetTextures() textures.ITextures {
	return e.textures
}

// GetWidth returns the width of the engine as an integer.
func (e *Engine) GetWidth() int {
	return e.w
}

// GetHeight returns the height of the Engine.
func (e *Engine) GetHeight() int {
	return e.h
}

// SectorAt retrieves the Sector at the specified index within the portal's sector list.
func (e *Engine) SectorAt(idx int) *model.Sector {
	return e.portal.SectorAt(idx)
}

// GetSectorsMaxHeight returns the maximum height value among all sectors in the engine.
func (e *Engine) GetSectorsMaxHeight() float64 {
	return e.sectorsMaxHeight
}

// Len returns the number of sectors currently managed by the Engine.
func (e *Engine) Len() int {
	return e.portal.Len()
}

// Setup initializes the Engine instance using the provided configuration, setting up textures, player, portal, and sectors.
func (e *Engine) Setup(cfg *model.ConfigRoot) error {
	compiler := model.NewCompiler()
	if err := compiler.Setup(cfg); err != nil {
		return err
	}
	e.sectorTree = compiler.GetSectors()
	e.player = compiler.GetPlayer()
	e.sectorsMaxHeight = compiler.GetMaxHeight()
	e.things = compiler.GetThings()
	e.entities = compiler.GetEntities()

	e.textures = cfg.Textures

	e.portal = portal.NewPortal(e.w, e.h, e.maxQueue)
	if err := e.portal.Setup(e.sectorTree.GetSectors()); err != nil {
		return err
	}

	for _, thing := range e.things {
		e.thingsDict[thing.GetId()] = thing
	}
	return nil
}

func (e *Engine) Compute(player *model.Player, vi *model.ViewMatrix) ([]*model.CompiledSector, int, []model.IThing) {
	// 1. Pre-Sync ViewMatrix
	vi.Compute(player)

	// 2. AI & Forze Esterne: Svegliamo le entità PRIMA del calcolo fisico
	pX, pY := e.player.GetXY()
	for _, t := range e.things {
		t.Compute(pX, pY)
	}

	// 3. Moto Statico Player
	player.Compute(vi)

	// 5. Solver Dinamico
	entities := e.entities.Compute()

	// 5. Sync Up (Physics -> Model) - Player
	e.player.MoveEntityApply()

	// 5b. Sync Up (Physics -> Model) - Things
	for _, ent := range entities {
		if t, ok := e.thingsDict[ent.GetId()]; ok {
			t.MoveEntityApply()
		}
	}

	// 6. Post-Sync ViewMatrix
	vi.Compute(player)

	cs, count := e.portal.Compute(vi)

	return cs, count, e.things
}
