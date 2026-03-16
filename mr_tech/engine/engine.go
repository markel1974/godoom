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
	player           *model.Player
	portal           *portal.Portal
	textures         textures.ITextures
	w                int
	h                int
	maxQueue         int
	things           []*model.Thing
	thingsDict       map[string]*model.Thing
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
	e.playerEnt = e.tree.Spawn("PLAYER", pX, pY, e.player.GetRadius(), e.player.GetMass())
	e.things = compiler.GetThings()
	for _, thing := range compiler.GetThings() {
		tP := thing.Position
		e.thingsDict[thing.Id] = thing
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
	e.tree.tree.UpdateObject(e.playerEnt)

	// 5. Solver Dinamico: Ora 'e.tree.counter' includerà i nemici con Vx/Vy > 0
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
	for idx := 0; idx < e.tree.counter; idx++ {
		physEnt := e.tree.moving[idx]
		if t, ok := e.thingsDict[physEnt.Id]; ok {
			//if physEnt, ok := e.tree.entities[t.Id]; ok {
			tPx := physEnt.GetCenterX()
			tPy := physEnt.GetCenterY()
			tDx := tPx - t.Position.X
			tDy := tPy - t.Position.Y

			// EPSILON FIX: Ignora i micro-spostamenti da virgola mobile
			if math.Abs(tDx) > 0.001 || math.Abs(tDy) > 0.001 {
				// 1. Taglia il vettore fisico contro i muri logici
				cDx, cDy := t.ClipMovement(tDx, tDy)
				// 2. Applica il movimento spaziale (e aggiorna i portali)
				//fmt.Printf("MOVEMENT APPLY %s %f:%f\n", t.Id, cDx, cDy)
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

func (e *Engine) moveEnemiesOld() {
	pX, pY := e.player.GetXY()

	for _, t := range e.things {
		if t.Animation == nil {
			continue
		}
		//TODO REMOVE
		//if !_enemies[t.Type] {
		//	continue
		//}

		physEnt, ok := e.tree.entities[t.Id]
		if !ok {
			continue
		}

		// 1. Calcolo direzione verso il player
		dx := pX - t.Position.X
		dy := pY - t.Position.Y
		dist := math.Sqrt(dx*dx + dy*dy)

		// 2. Se il nemico è abbastanza vicino ma non troppo (range di attacco)
		if dist > 4.0 && dist < 200.0 {
			speed := 0.2 // Tuning della velocità del nemico
			invDist := 1.0 / dist

			//t.MoveApply(dx*invDist*speed, dy*invDist*speed)
			physEnt.Vx = dx * invDist * speed
			physEnt.Vy = dy * invDist * speed

			// Iniezione della forza nel motore fisico
			//physEnt.ApplyImpulse(dx*invDist*speed, dy*invDist*speed, 1.0)
		}
	}
}

func (e *Engine) moveEnemies() {
	pX, pY := e.player.GetXY()
	acceleration := 0.15 // Fattore di accelerazione (0.0 a 1.0)

	for _, t := range e.things {
		if t.Animation == nil {
			continue
		}

		physEnt, ok := e.tree.entities[t.Id]
		if !ok {
			continue
		}

		// 1. Calcolo direzione verso il player
		dx := pX - t.Position.X
		dy := pY - t.Position.Y
		dist := math.Sqrt(dx*dx + dy*dy)

		// 2. Comportamento: Blending Vettoriale
		if dist > 32.0 && dist < 1000.0 {
			targetSpeed := 3.0 // Velocità di crociera
			invDist := 1.0 / dist

			// Vettore intenzionale dell'IA
			dirX := dx * invDist * targetSpeed
			dirY := dy * invDist * targetSpeed

			// VELOCITY BLEND: Conserva il knockback fisico e aggiunge la spinta dell'IA
			physEnt.Vx = physEnt.Vx*(1-acceleration) + (dirX * acceleration)
			physEnt.Vy = physEnt.Vy*(1-acceleration) + (dirY * acceleration)

		} else {
			// Decelera morbidamente. Se il nemico viene spinto, scivolerà per inerzia.
			physEnt.Vx *= (1 - acceleration)
			physEnt.Vy *= (1 - acceleration)
		}
	}
}

/*
func (e *Engine) moveEnemies() {
	pX, pY := e.player.GetXY()

	for _, t := range e.things {
		if t.Animation == nil {
			continue
		}

		physEnt, ok := e.tree.entities[t.Id]
		if !ok {
			continue
		}

		// 1. Calcolo direzione verso il player
		dx := pX - t.Position.X
		dy := pY - t.Position.Y
		dist := math.Sqrt(dx*dx + dy*dy)

		// 2. Comportamento: Inseguimento Cinematico
		if dist > 32.0 && dist < 1000.0 {
			speed := 4.0 // Velocità costante desiderata (non è più una "forza")
			invDist := 1.0 / dist

			// Impostiamo direttamente la velocità cinematica bersaglio.
			// NOTA: Se il nemico sta subendo un forte knockback (es. Vx > 10 per un'esplosione),
			// potresti voler evitare di sovrascriverla, ma per il movimento base questo è il pattern corretto.
			physEnt.Vx = dx * invDist * speed
			physEnt.Vy = dy * invDist * speed
		} else {
			// Se è fuori range o è arrivato addosso al player, si ferma.
			// Senza questo, il solver fisico lo farebbe scivolare lentamente per l'inerzia
			// (a meno che l'attrito non sia a 1.0).
			physEnt.Vx = 0
			physEnt.Vy = 0
		}
	}
}

*/
