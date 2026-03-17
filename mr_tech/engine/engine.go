package engine

import (
	"github.com/markel1974/godoom/mr_tech/model"
	"github.com/markel1974/godoom/mr_tech/portal"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// Engine provides the core functionality for rendering, entity management, and spatial computations in a 3D environment.
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

// NewEngine initializes and returns a new instance of Engine with the specified width, height, and maxQueue size.
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

// GetPlayer returns the player instance associated with the engine.
func (e *Engine) GetPlayer() *model.Player {
	return e.player
}

// GetTextures returns the textures instance associated with the Engine.
func (e *Engine) GetTextures() textures.ITextures {
	return e.textures
}

// GetWidth returns the width of the Engine in pixels.
func (e *Engine) GetWidth() int {
	return e.w
}

// GetHeight returns the height of the Engine instance.
func (e *Engine) GetHeight() int {
	return e.h
}

// SectorAt returns the Sector object located at the given index within the engine's portal.
func (e *Engine) SectorAt(idx int) *model.Sector {
	return e.portal.SectorAt(idx)
}

// GetSectorsMaxHeight returns the maximum height value among all sectors as a float64.
func (e *Engine) GetSectorsMaxHeight() float64 {
	return e.sectorsMaxHeight
}

// Len returns the number of sectors managed by the Engine.
func (e *Engine) Len() int {
	return e.portal.Len()
}

// Setup initializes the Engine by configuring its components using the provided configuration and setting up internal resources.
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

// Compute performs the main game logic including view matrix syncing, AI updates, physics simulation, and sector rendering.
func (e *Engine) Compute(player *model.Player, vi *model.ViewMatrix) ([]*model.CompiledSector, int, []model.IThing) {
	// 1. Pre-Sync ViewMatrix
	vi.Compute(player)

	// 2. AI & External Forces: Wake up entities BEFORE physics calculation
	pX, pY := player.GetXY()
	for _, t := range e.things {
		t.Compute(pX, pY)
	}

	// 3. Static Player Motion
	player.Compute(vi)

	// 4. Dynamic Solver
	entities := e.entities.Compute()

	// 5. Sync Up (Physics -> Model) - Player
	player.MoveEntityApply()

	// 6. Sync Up (Physics -> Model) - Things
	for _, ent := range entities {
		if t, ok := e.thingsDict[ent.GetId()]; ok {
			t.MoveEntityApply()
		}
	}

	// 7. Post-Sync ViewMatrix
	vi.Compute(player)

	// 8. Portal Compute
	cs, count := e.portal.Compute(vi)

	return cs, count, e.things
}
