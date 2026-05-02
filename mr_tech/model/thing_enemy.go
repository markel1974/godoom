package model

import (
	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// ThingEnemy represents a physical or logical entity in the environment with attributes like position, mass, and associated data.
type ThingEnemy struct {
	*ThingBase
	onThinking func(self config.IThingConfig, playerX, playerY, playerZ float64)
}

// NewThingEnemy creates and initializes a new ThingEnemy instance.
func NewThingEnemy(things *Things, cfg *config.Thing, anim *textures.Material, volume *Volume) *ThingEnemy {
	pos := cfg.Position
	if cfg.Speed <= 0 {
		cfg.Speed = 6
	}
	if cfg.Acceleration <= 0 {
		cfg.Acceleration = 3
	}
	if cfg.OnThinking == nil {
		panic("onThinking is nil for enemy:" + cfg.Id)
	}
	thing := &ThingEnemy{
		ThingBase:  NewThingBase(things, cfg, pos, anim, volume),
		onThinking: cfg.OnThinking,
	}
	thing.volume.SetThing(thing)
	return thing
}

func (t *ThingEnemy) PostMessage(ec *ThingEvent) {
	t.inbox <- ec
}

func (t *ThingEnemy) StartLoop() {
	go func() {
		for {
			select {
			case evt := <-t.inbox:
				switch evt.GetKind() {
				case StageThinking:
					t.StageThinking(evt.GetCoords())
				case StageCompute:
					t.StageCompute()
				case StageResolve:
					t.StageResolve(evt.GetSolverJitter())
				case StageApply:
					t.StageApply()
				}
				evt.Done()
			case <-t.done:
				return
			}
		}
	}()
}

// StageThinking updates the Thing's direction, position, and attack logic based on the player's coordinates.
func (t *ThingEnemy) StageThinking(playerX float64, playerY float64, playerZ float64) {
	t.onThinking(t, playerX, playerY, playerZ)
}
