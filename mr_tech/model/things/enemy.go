package things

import (
	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/model/volumes"
)

// Enemy represents an enemy entity that extends Base and defines behavior through a custom thinking function.
type Enemy struct {
	*Base
	onThinking func(playerX, playerY, playerZ float64)
}

// NewEnemy initializes and returns a new instance of Enemy with the specified configuration and parameters.
// It ensures that default values are set for speed and acceleration if not provided.
// The function panics if the OnThinking callback in the config is nil.
func NewEnemy(things IThingHandler, cfg *config.Thing, volume *volumes.Volume) *Enemy {
	if cfg.OnThinking == nil {
		panic("onThinking is nil for enemy:" + cfg.Id)
	}
	enemy := &Enemy{
		onThinking: cfg.OnThinking,
	}
	enemy.Base = NewBase(enemy, things, cfg, volume)
	entity := enemy.GetEntity()
	entity.SetKeepUpright(true)
	cfg.OnBind(enemy)
	return enemy
}

// IsEqual reports whether other is the same Enemy instance.
func (t *Enemy) IsEqual(other volumes.IThing) bool {
	if v, ok := other.(*Enemy); ok && t == v {
		return true
	}
	return false
}

// HasThinking determines if the Enemy instance has a thinking function assigned. Returns true if it is set.
func (t *Enemy) HasThinking() bool {
	return t.onThinking != nil
}

// Jump attempts to make the enemy leap with the given horizontal movement and vertical scaling, reporting success.
func (t *Enemy) Jump(leapX float64, leapY float64, zFactor float64) bool {
	return t.Base.Jump(t, leapX, leapY, zFactor)
}

// FireHitscan performs a raycast to detect the first intersecting object within a specified direction and range.
func (t *Enemy) FireHitscan(id string, pos geometry.XYZ, force float64, dirX, dirY, dirZ float64) {
	t.Base.FireHitscan(t, id, pos, force, dirX, dirY, dirZ)
}

// LaunchObject spawns a bullet at the specified position, angle, and pitch using predefined physical parameters.
func (t *Enemy) LaunchObject(throwableIndex int, onCollision config.CollisionFunc, onImpact config.ImpactFunc, pos geometry.XYZ, angle, pitch, speed float64) {
	t.things.CreateThrowable(t, throwableIndex, onCollision, onImpact, t.location, pos, angle, pitch, speed)
}

// PostMessage sends a ThingEvent instance to the inbox channel of Enemy for processing.
func (t *Enemy) PostMessage(ec *volumes.ThingEvent) {
	t.inbox <- ec
}

// StartLoop initializes and starts a goroutine to handle ThingEvent messages and process them based on their compute stage.
func (t *Enemy) StartLoop() {
	go func() {
		for {
			select {
			case evt := <-t.inbox:
				switch evt.GetKind() {
				case volumes.StageThinking:
					t.StageThinking(evt.GetCoords())
				case volumes.StageCompute:
					//t.StageCompute()
				case volumes.StageResolve:
					t.StageResolve()
				case volumes.StageApply:
					t.StageApply()
				}
				evt.Done()
			case <-t.done:
				return
			}
		}
	}()
}

// StageThinking processes the enemy's thinking phase using the player's coordinates (X, Y, Z) as input.
func (t *Enemy) StageThinking(playerX float64, playerY float64, playerZ float64) {
	t.onThinking(playerX, playerY, playerZ)
}
