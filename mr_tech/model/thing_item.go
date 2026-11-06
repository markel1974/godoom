package model

import (
	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/geometry"
)

// ThingItem is a specialized wrapper around ThingBase, representing a specific type of Thing in the environment.
type ThingItem struct {
	*ThingBase
}

// NewThingItem creates a new ThingItem instance by initializing its base properties using the provided configuration.
func NewThingItem(things *Things, cfg *config.Thing, volume *Volume) *ThingItem {
	thing := &ThingItem{}
	thing.ThingBase = NewThingBase(thing, things, cfg, volume)
	cfg.OnBind(thing)
	return thing
}

// IsEqual reports whether other is the same ThingItem instance.
func (t *ThingItem) IsEqual(other IThing) bool {
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
func (t *ThingItem) PostMessage(ec *ThingEvent) {
	t.inbox <- ec
}

// StartLoop begins a goroutine that processes incoming events or signals termination via the 'done' channel.
func (t *ThingItem) StartLoop() {
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

// HasThinking determines if the ThingItem instance has a thinking function assigned. Returns true if it is set.
func (t *ThingItem) HasThinking() bool {
	return false
}

// StageThinking updates the state of the ThingItem instance based on the provided player coordinates (X, Y, Z).
func (t *ThingItem) StageThinking(playerX float64, playerY float64, playerZ float64) {
}
