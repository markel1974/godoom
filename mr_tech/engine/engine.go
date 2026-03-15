package engine

import (
	"fmt"
	"os"

	"github.com/markel1974/godoom/mr_tech/model"
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
	}
}

// Setup initializes the Engine instance using the provided configuration, setting up textures, player, portal, and sectors.
func (e *Engine) Setup(cfg *model.ConfigRoot) error {
	e.textures = cfg.Textures
	compiler := model.NewCompiler()
	if err := compiler.Setup(cfg); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	playerSector, err := compiler.Get(cfg.Player.Sector)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	e.player = model.NewPlayer(cfg.Player, playerSector, false)
	e.portal = portal.NewPortal(e.w, e.h, e.maxQueue)
	if err = e.portal.Setup(compiler.GetSectors()); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	e.sectorsMaxHeight = compiler.GetMaxHeight()
	//TODO
	e.things = compiler.GetThings()
	return nil
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

// Compile generates a set of compiled sectors and associated things based on the player's view matrix and current state.
func (e *Engine) Compile(player *model.Player, vi *model.ViewMatrix) ([]*model.CompiledSector, int, []*model.Thing) {
	cs, count := e.portal.Compile(player, vi)
	return cs, count, e.things
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
