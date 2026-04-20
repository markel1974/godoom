package model

import (
	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// ThingItem is a specialized wrapper around ThingBase, representing a specific type of Thing in the environment.
type ThingItem struct {
	*ThingBase
}

// NewThingItem creates a new ThingItem instance by initializing its base properties using the provided configuration.
func NewThingItem(things *Things, cfg *config.Thing, anim *textures.Animation, volume *Volume) *ThingItem {
	pos := cfg.Position
	thing := &ThingItem{
		ThingBase: NewThingBase(things, cfg, pos, anim, volume),
	}
	return thing
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
					t.Compute(evt.GetCoords())
				case StagePhysics:
					t.PhysicsApply()
				}
				evt.Done()
			case <-t.done:
				return
			}
		}
	}()
}

// Compute updates the state of the ThingItem instance based on the provided player coordinates (X, Y, Z).
func (t *ThingItem) Compute(playerX float64, playerY float64, playerZ float64) {
}

// PhysicsApply executes physics-related calculations and updates for the ThingItem instance by invoking the doPhysics method.
func (t *ThingItem) PhysicsApply() {
	t.doPhysics()
}
