package engine

import (
	"github.com/markel1974/godoom/mr_tech/model"
	"github.com/markel1974/godoom/mr_tech/portal"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// Engine provides the core functionality for rendering, entity management, and spatial computations in a 3D environment.
type Engine struct {
	portal   *portal.Portal
	w        int
	h        int
	maxQueue int
	things   *model.Things
	entities *model.Entities
	player   *model.ThingPlayer
	sectors  *model.Sectors
	config   *model.ConfigRoot
}

// NewEngine initializes and returns a new instance of Engine with the specified width, height, and maxQueue size.
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
		config:   nil,
	}
}

// GetPlayer returns the player instance associated with the engine.
func (e *Engine) GetPlayer() *model.ThingPlayer {
	return e.player
}

// GetWidth returns the width of the Engine in pixels.
func (e *Engine) GetWidth() int {
	return e.w
}

// GetHeight returns the height of the Engine instance.
func (e *Engine) GetHeight() int {
	return e.h
}

func (e *Engine) GetTextures() textures.ITextures {
	return e.things.GetTextures()
}

// SectorAt returns the Sector object located at the given index within the engine's portal.
func (e *Engine) SectorAt(idx int) *model.Sector {
	return e.portal.SectorAt(idx)
}

// Len returns the number of sectors managed by the Engine.
func (e *Engine) Len() int {
	return e.portal.Len()
}

// Setup initializes the Engine by configuring its components using the provided configuration and setting up internal resources.
func (e *Engine) Setup(cfg *model.ConfigRoot) error {
	e.config = cfg
	compiler := model.NewCompiler()
	if err := compiler.Setup(cfg); err != nil {
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

// Fire triggers the creation of a bullet in the specified sector at the given position and angle.
func (e *Engine) Fire(sector *model.Sector, x float64, y float64, angle float64) {
	e.things.CreateBullet(sector, x, y, angle)
}

// Compute performs the main game logic including view matrix syncing, AI updates, physics simulation, and sector rendering.
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

/*
// ComputeNew performs the main game logic including view matrix syncing, AI updates, physics simulation, and sector rendering.
func (e *Engine) Compute(player *model.ThingPlayer, vi *model.ViewMatrix) ([]*model.CompiledSector, int, []model.IThing) {
	// 1. Pre-Sync ViewMatrix
	vi.Compute(player)

	var things []model.IThing

	// 8. Portal Compute
	cs, count := e.portal.Compute(vi)

	csDict := make(map[*model.Sector]bool)
	for _, compiled := range cs {
		csDict[compiled.Sector] = true
	}

	// 2. AI & External Forces: Wake up entities BEFORE physics calculation
	pX, pY := player.GetXY()
	for _, thing := range e.things {
		thing.Compute(pX, pY)
		if _, ok := csDict[thing.GetSector()]; ok {
			things = append(things, thing)
		}
	}

	// 3. Static ThingPlayer Motion
	player.Compute(vi)

	// 4. Dynamic Solver
	entities := e.entities.Compute()

	// 5. Sync Up (Physics -> Model) - ThingPlayer
	player.MoveEntityApply()

	// 6. Sync Up (Physics -> Model) - Things
	for _, ent := range entities {
		if t, ok := e.thingsDict[ent.GetId()]; ok {
			t.MoveEntityApply()
		}
	}

	// 7. Post-Sync ViewMatrix
	vi.Compute(player)

	return cs, count, things
}
*/
