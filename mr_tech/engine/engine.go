package engine

import (
	"github.com/markel1974/godoom/mr_tech/model"
	"github.com/markel1974/godoom/mr_tech/portal"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// Engine represents a core game simulation system, managing entities, sectors, player, and rendering configurations.
type Engine struct {
	portal   *portal.Portal
	w        int
	h        int
	maxQueue int
	things   *model.Things
	entities *model.Entities
	player   *model.ThingPlayer
	sectors  *model.Sectors
}

// NewEngine creates and initializes a new Engine instance with the specified width, height, and maximum queue size.
func NewEngine(w int, h int, maxQueue int) *Engine {
	return &Engine{
		portal:   nil,
		w:        w,
		h:        h,
		maxQueue: maxQueue,
		things:   nil,
		entities: nil,
		sectors:  nil,
		player:   nil,
	}
}

// GetPlayer returns the current player instance managed by the engine.
func (e *Engine) GetPlayer() *model.ThingPlayer {
	return e.player
}

// GetWidth returns the width of the Engine's configured screen or viewport as an integer value.
func (e *Engine) GetWidth() int {
	return e.w
}

// GetHeight returns the height of the engine in pixels.
func (e *Engine) GetHeight() int {
	return e.h
}

// GetTextures retrieves the ITextures instance, providing access to texture names and indexed textures.
func (e *Engine) GetTextures() textures.ITextures {
	return e.things.GetTextures()
}

// SectorAt returns the sector at the specified index from the portal within the engine.
func (e *Engine) SectorAt(idx int) *model.Sector {
	return e.portal.SectorAt(idx)
}

// Len returns the number of sectors currently managed by the Engine.
func (e *Engine) Len() int {
	return e.portal.Len()
}

// Setup initializes the Engine using the provided configuration, creating sectors, player, things, entities, and the portal.
func (e *Engine) Setup(cfg *model.ConfigRoot) error {
	compiler := model.NewCompiler()
	if err := compiler.Compile(cfg); err != nil {
		return err
	}
	e.sectors = compiler.GetSectors()
	e.player = compiler.GetPlayer()
	e.things = compiler.GetThings()
	e.entities = compiler.GetEntities()

	e.portal = portal.NewPortal(e.w, e.h, e.maxQueue)
	if err := e.portal.Setup(e.sectors.GetSectors()); err != nil {
		return err
	}
	return nil
}

// Compute handles the main game logic loop, updating the player, entities, physics, portals, and view matrix synchronously.
func (e *Engine) Compute(player *model.ThingPlayer, vi *model.ViewMatrix) ([]*model.CompiledSector, int, []model.IThing) {
	// 1. Pre-Sync ViewMatrix
	vi.Update(player)

	// 2. AI & External Forces: Wake up entities BEFORE physics calculation
	pX, pY := player.GetPosition()

	e.things.Compute(pX, pY)

	// 3. Static ThingPlayer Motion
	player.Update(vi)

	// 4. Dynamic Solver
	entities := e.entities.Compute()

	// 5. Sync Up (Physics -> Model) - Things
	for _, ent := range entities {
		ent.PhysicsApply()
	}

	// 6. Post-Sync ViewMatrix
	vi.Update(player)

	// 7. Portal Compute
	cs, count := e.portal.Compute(vi)

	return cs, count, e.things.GetThings()
}

// Fire spawns a bullet in the specified sector at the given coordinates and angle.
func (e *Engine) Fire(sector *model.Sector, x float64, y float64, angle float64) {
	e.things.CreateBullet(sector, x, y, angle)
}
