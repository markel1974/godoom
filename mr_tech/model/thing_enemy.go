package model

import (
	"github.com/markel1974/godoom/mr_tech/config"
)

// ThingEnemy represents an enemy entity that extends ThingBase and defines behavior through a custom thinking function.
type ThingEnemy struct {
	*ThingBase
	onThinking func(self config.IThingConfig, playerX, playerY, playerZ float64)
}

// NewThingEnemy initializes and returns a new instance of ThingEnemy with the specified configuration and parameters.
// It ensures that default values are set for speed and acceleration if not provided.
// The function panics if the OnThinking callback in the config is nil.
func NewThingEnemy(things *Things, cfg *config.Thing, volume *Volume) *ThingEnemy {
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
		onThinking: cfg.OnThinking,
	}
	thing.ThingBase = NewThingBase(thing, things, cfg, volume)
	return thing
}

// PostMessage sends a ThingEvent instance to the inbox channel of ThingEnemy for processing.
func (t *ThingEnemy) PostMessage(ec *ThingEvent) {
	t.inbox <- ec
}

// StartLoop initializes and starts a goroutine to handle ThingEvent messages and process them based on their compute stage.
func (t *ThingEnemy) StartLoop() {
	go func() {
		for {
			select {
			case evt := <-t.inbox:
				switch evt.GetKind() {
				case StageThinking:
					t.StageThinking(evt.GetCoords())
				case StageCompute:
					//t.StageCompute()
				case StageResolve:
					t.StageResolve(evt.GetSolverIndex(), evt.GetSolverJitter())
				case StageApply:
					t.StageApply(evt.GetSolverJitter())
				}
				evt.Done()
			case <-t.done:
				return
			}
		}
	}()
}

// StageThinking processes the enemy's thinking phase using the player's coordinates (X, Y, Z) as input.
func (t *ThingEnemy) StageThinking(playerX float64, playerY float64, playerZ float64) {
	t.onThinking(t, playerX, playerY, playerZ)
}
