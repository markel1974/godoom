package model

import (
	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/geometry"
)

// ThingEnemy represents an enemy entity that extends ThingBase and defines behavior through a custom thinking function.
type ThingEnemy struct {
	*ThingBase
	onThinking func(playerX, playerY, playerZ float64)
}

// NewThingEnemy initializes and returns a new instance of ThingEnemy with the specified configuration and parameters.
// It ensures that default values are set for speed and acceleration if not provided.
// The function panics if the OnThinking callback in the config is nil.
func NewThingEnemy(things *Things, cfg *config.Thing, volume *Volume) *ThingEnemy {
	if cfg.OnThinking == nil {
		panic("onThinking is nil for enemy:" + cfg.Id)
	}
	enemy := &ThingEnemy{
		onThinking: cfg.OnThinking,
	}
	enemy.ThingBase = NewThingBase(enemy, things, cfg, volume)
	cfg.OnBind(enemy)
	return enemy
}

// IsEqual reports whether other is the same ThingEnemy instance.
func (t *ThingEnemy) IsEqual(other IThing) bool {
	if v, ok := other.(*ThingEnemy); ok && t == v {
		return true
	}
	return false
}

// HasThinking determines if the ThingEnemy instance has a thinking function assigned. Returns true if it is set.
func (t *ThingEnemy) HasThinking() bool {
	return t.onThinking != nil
}

// Jump attempts to make the enemy leap with the given horizontal movement and vertical scaling, reporting success.
func (t *ThingEnemy) Jump(leapX float64, leapY float64, zFactor float64) bool {
	return t.ThingBase.Jump(t, leapX, leapY, zFactor)
}

// FireHitscan performs a raycast to detect the first intersecting object within a specified direction and range.
func (t *ThingEnemy) FireHitscan(id string, pos geometry.XYZ, force float64, dirX, dirY, dirZ float64) {
	t.ThingBase.FireHitscan(t, id, pos, force, dirX, dirY, dirZ)
}

// LaunchObject spawns a bullet at the specified position, angle, and pitch using predefined physical parameters.
func (t *ThingEnemy) LaunchObject(throwableIndex int, onCollision config.CollisionFunc, onImpact config.ImpactFunc, pos geometry.XYZ, angle, pitch, speed float64) {
	t.things.CreateThrowable(t, throwableIndex, onCollision, onImpact, t.location, pos, angle, pitch, speed)
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
					t.StageResolve()
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

// StageThinking processes the enemy's thinking phase using the player's coordinates (X, Y, Z) as input.
func (t *ThingEnemy) StageThinking(playerX float64, playerY float64, playerZ float64) {
	t.onThinking(playerX, playerY, playerZ)
}
