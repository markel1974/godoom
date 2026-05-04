package model

import (
	"github.com/markel1974/godoom/mr_tech/config"
)

// ThingItem is a specialized wrapper around ThingBase, representing a specific type of Thing in the environment.
type ThingItem struct {
	*ThingBase
}

// NewThingItem creates a new ThingItem instance by initializing its base properties using the provided configuration.
func NewThingItem(things *Things, cfg *config.Thing, volume *Volume) *ThingItem {
	pos := cfg.Position
	thing := &ThingItem{}
	thing.ThingBase = NewThingBase(thing, things, cfg, pos, volume)
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

// StageThinking updates the state of the ThingItem instance based on the provided player coordinates (X, Y, Z).
func (t *ThingItem) StageThinking(playerX float64, playerY float64, playerZ float64) {
}
