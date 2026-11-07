package things

import (
	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/model/volumes"
)

// ThingItem is a specialized wrapper around ThingBase, representing a specific type of Thing in the environment.
type ThingItem struct {
	*ThingBase
}

// NewThingItem creates a new ThingItem instance by initializing its base properties using the provided configuration.
func NewThingItem(things IThingHandler, cfg *config.Thing, volume *volumes.Volume) *ThingItem {
	item := &ThingItem{}
	item.ThingBase = NewThingBase(item, things, cfg, volume)
	cfg.OnBind(item)
	return item
}

// IsEqual reports whether other is the same ThingItem instance.
func (t *ThingItem) IsEqual(other volumes.IThing) bool {
	if v, ok := other.(*ThingItem); ok && t == v {
		return true
	}
	return false
}

// Jump attempts to make the enemy leap with the given horizontal movement and vertical scaling, reporting success.
func (t *ThingItem) Jump(leapX float64, leapY float64, zFactor float64) bool {
	return false
}

// FireHitscan performs a raycast to detect the first intersecting object within a specified direction and range.
func (t *ThingItem) FireHitscan(id string, pos geometry.XYZ, force float64, dirX, dirY, dirZ float64) {
}

// LaunchObject spawns a bullet at the specified position, angle, and pitch using predefined physical parameters.
func (t *ThingItem) LaunchObject(throwableIndex int, onCollision config.CollisionFunc, onImpact config.ImpactFunc, pos geometry.XYZ, angle, pitch, speed float64) {
}

// PostMessage sends an ThingEvent instance to the ThingItem's inbox channel for processing.
func (t *ThingItem) PostMessage(ec *volumes.ThingEvent) {
	t.inbox <- ec
}

// StartLoop begins a goroutine that processes incoming events or signals termination via the 'done' channel.
func (t *ThingItem) StartLoop() {
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

// HasThinking determines if the ThingItem instance has a thinking function assigned. Returns true if it is set.
func (t *ThingItem) HasThinking() bool {
	return false
}

// StageThinking updates the state of the ThingItem instance based on the provided player coordinates (X, Y, Z).
func (t *ThingItem) StageThinking(playerX float64, playerY float64, playerZ float64) {
}
