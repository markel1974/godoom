package engine

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/model"
	"github.com/markel1974/godoom/mr_tech/physics"
	"github.com/markel1974/godoom/mr_tech/portal"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// Engine represents the core game engine handling rendering, player interactions, and environment configuration.
type Engine struct {
	player           *model.Player
	portal           *portal.Portal
	textures         textures.ITextures
	w                int
	h                int
	maxQueue         int
	things           []*model.Thing
	sectorsMaxHeight float64
	tree             *EntityManager
	playerEnt        *physics.Entity
}

// NewEngine initializes and returns a new Engine instance with specified width, height, and maximum render queue size.
func NewEngine(w int, h int, maxQueue int) *Engine {
	return &Engine{
		player:           nil,
		portal:           nil,
		w:                w,
		h:                h,
		maxQueue:         maxQueue,
		things:           nil,
		sectorsMaxHeight: 0,
		tree:             nil,
	}
}

// Setup initializes the Engine instance using the provided configuration, setting up textures, player, portal, and sectors.
func (e *Engine) Setup(cfg *model.ConfigRoot) error {
	e.textures = cfg.Textures
	compiler := model.NewCompiler()
	if err := compiler.Setup(cfg); err != nil {
		return err
	}
	playerSector, err := compiler.Get(cfg.Player.Sector)
	if err != nil {
		return err
	}
	e.player = model.NewPlayer(cfg.Player, playerSector, false)
	e.portal = portal.NewPortal(e.w, e.h, e.maxQueue)
	if err = e.portal.Setup(compiler.GetSectors()); err != nil {
		return err
	}
	e.sectorsMaxHeight = compiler.GetMaxHeight()

	e.tree = NewEntityManager(4096)
	pX, pY := e.player.GetXY()
	e.playerEnt = e.tree.Spawn("PLAYER", pX, pY, 20.0, 100.0)
	e.things = compiler.GetThings()
	for _, thing := range compiler.GetThings() {
		tP := thing.Position
		e.tree.Spawn(thing.Id, tP.X, tP.Y, thing.Radius, thing.Mass)
	}
	return nil
}

// ComputeOLD performs calculations for rendering, updates player and tree states, and returns visible sectors, count, and entities.
func (e *Engine) ComputeOLD(player *model.Player, vi *model.ViewMatrix) ([]*model.CompiledSector, int, []*model.Thing) {
	vi.Compute(player)
	cs, count := e.portal.Compute(vi)
	player.Compute(vi)
	e.tree.Compute()
	return cs, count, e.things
}

// Compute esegue l'integrazione del frame unendo la logica dei portali al solver fisico.
// Compute esegue l'integrazione del frame unendo la logica dei portali al solver fisico.
func (e *Engine) Compute(player *model.Player, vi *model.ViewMatrix) ([]*model.CompiledSector, int, []*model.Thing) {
	// 1. Pre-Sync ViewMatrix
	vi.Compute(player)

	// 2. Moto Statico
	player.Compute(vi)

	// 3. Sync Down (Model -> Physics)
	pX, pY := player.GetXY()
	pRadius := e.playerEnt.GetWidth() / 2.0
	e.playerEnt.MoveTo(pX-pRadius, pY-pRadius)
	e.playerEnt.Vx, e.playerEnt.Vy = player.GetVelocity()
	e.tree.tree.UpdateObject(e.playerEnt)

	// 4. Solver Dinamico
	e.tree.Compute()

	// 5. Sync Up (Physics -> Model) - Player
	newPx := e.playerEnt.GetCenterX()
	newPy := e.playerEnt.GetCenterY()
	pX, pY = player.GetXY()
	dx := newPx - pX
	dy := newPy - pY

	if math.Abs(dx) > 0.001 || math.Abs(dy) > 0.001 {
		player.MoveApply(dx, dy)
	}

	// 5b. Sync Up (Physics -> Model) - Things
	for _, t := range e.things {
		if physEnt, ok := e.tree.entities[t.Id]; ok {
			tPx := physEnt.GetCenterX()
			tPy := physEnt.GetCenterY()
			tDx := tPx - t.Position.X
			tDy := tPy - t.Position.Y

			// EPSILON FIX: Ignora i micro-spostamenti da virgola mobile
			if math.Abs(tDx) > 0.001 || math.Abs(tDy) > 0.001 {
				// 1. Taglia il vettore fisico contro i muri logici
				cDx, cDy := t.ClipMovement(tDx, tDy)
				// 2. Applica il movimento spaziale (e aggiorna i portali)
				t.MoveApply(cDx, cDy)
				// 3. RETRO-CORREZIONE (Sync-Back)
				// Se il muro ci ha deviato o bloccato, l'AABB fisico è rimasto dentro il muro.
				// Dobbiamo risincronizzarlo istantaneamente alle coordinate logiche esatte.
				if cDx != tDx || cDy != tDy {
					r := physEnt.GetWidth() / 2.0
					physEnt.MoveTo(t.Position.X-r, t.Position.Y-r)
					e.tree.tree.UpdateObject(physEnt)
				}
			}
		}
	}

	// 6. Post-Sync ViewMatrix
	vi.Compute(player)

	cs, count := e.portal.Compute(vi)
	return cs, count, e.things
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
