package engine

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/model"
	"github.com/markel1974/godoom/mr_tech/physics"
	"github.com/markel1974/godoom/mr_tech/portal"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// TODO REMOVE
var _enemies = map[int]bool{
	3004: true,
	9:    true,
	65:   true,
	3001: true,
	3002: true,
	58:   true,
	3003: true,
	69:   true,
	3005: true,
	3006: true,
	68:   true,
	71:   true,
	66:   true,
	67:   true,
	64:   true,
	16:   true,
	7:    true,
}

// Engine represents the core game engine handling rendering, player interactions, and environment configuration.
type Engine struct {
	portal           *portal.Portal
	textures         textures.ITextures
	w                int
	h                int
	maxQueue         int
	things           []*model.Thing
	thingsDict       map[string]*model.Thing
	sectorsMaxHeight float64
	manager          *EntityManager
	player           *model.Player
	playerEnt        *physics.Entity
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
		manager:          nil,
		sectorTree:       nil,
		player:           nil,
		thingsDict:       make(map[string]*model.Thing),
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

	e.textures = cfg.Textures

	e.portal = portal.NewPortal(e.w, e.h, e.maxQueue)
	if err := e.portal.Setup(e.sectorTree.GetSectors()); err != nil {
		return err
	}

	e.manager = NewEntityManager(uint(1 + len(e.things)))
	pX, pY := e.player.GetXY()
	e.playerEnt = e.manager.Spawn("PLAYER", pX, pY, e.player.GetRadius(), e.player.GetMass())
	for _, thing := range e.things {
		tP := thing.Position
		e.thingsDict[thing.Id] = thing
		e.manager.Spawn(thing.Id, tP.X, tP.Y, thing.Radius, thing.Mass)
	}
	return nil
}

func (e *Engine) Compute(player *model.Player, vi *model.ViewMatrix) ([]*model.CompiledSector, int, []*model.Thing) {
	// 1. Pre-Sync ViewMatrix
	vi.Compute(player)

	// 2. AI & Forze Esterne: Svegliamo le entità PRIMA del calcolo fisico
	e.moveEnemies()

	// 3. Moto Statico Player
	player.Compute(vi)

	// 4. Sync Down Player (Model -> Physics)
	pX, pY := player.GetXY()
	pRadius := e.playerEnt.GetWidth() / 2.0
	e.playerEnt.MoveTo(pX-pRadius, pY-pRadius)
	e.playerEnt.Vx, e.playerEnt.Vy = player.GetVelocity()
	e.manager.UpdateObject(e.playerEnt)

	// 5. Solver Dinamico
	entities := e.manager.Compute()

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
	for _, physEnt := range entities {
		if t, ok := e.thingsDict[physEnt.Id]; ok {
			tPx := physEnt.GetCenterX()
			tPy := physEnt.GetCenterY()

			// 1. Delta Passivo (rimbalzi calcolati da SetupCollision)
			tDx := tPx - t.Position.X
			tDy := tPy - t.Position.Y

			// 2. Delta Attivo (Kinematic Drive) aggiunto solo se c'è intenzionalità
			if physEnt.G > 0 {
				tDx += physEnt.Vx
				tDy += physEnt.Vy
			}

			if math.Abs(tDx) > 0.001 || math.Abs(tDy) > 0.001 {
				// 3. Traslazione del modello logico
				t.MoveApply(tDx, tDy)

				// 4. Retro-Correzione (Sync-Back) AABB fisico
				r := physEnt.GetWidth() / 2.0
				physEnt.MoveTo(t.Position.X-r, t.Position.Y-r)
				e.manager.UpdateObject(physEnt)
			}
		}
	}

	// 6. Post-Sync ViewMatrix
	vi.Compute(player)

	cs, count := e.portal.Compute(vi)
	return cs, count, e.things
}

func (e *Engine) moveEnemies() {
	pX, pY := e.player.GetXY()
	acceleration := 0.15

	for _, t := range e.things {
		physEnt, ok := e.manager.entities[t.Id]
		if !ok {
			continue
		}
		if t.Speed == 0 {
			continue
		}

		dx := pX - t.Position.X
		dy := pY - t.Position.Y
		dist := math.Sqrt(dx*dx + dy*dy)

		if dist < 25.0 {
			targetSpeed := t.Speed
			invDist := 1.0 / dist
			dirX := dx * invDist * targetSpeed
			dirY := dy * invDist * targetSpeed
			physEnt.Vx = physEnt.Vx*(1-acceleration) + (dirX * acceleration)
			physEnt.Vy = physEnt.Vy*(1-acceleration) + (dirY * acceleration)
			if physEnt.GForce == 0 {
				physEnt.GForce = 1.0
			}
			if physEnt.Friction < 0.2 {
				physEnt.Friction = 0.99
			}
		}
	}
}
