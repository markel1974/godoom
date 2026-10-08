package model

import (
	"github.com/markel1974/godoom/mr_tech/config"
)

// ThingHud is a specialized wrapper around ThingBase, representing a HUD entity that follows the camera.
type ThingHud struct {
	*ThingBase
}

// NewThingHud creates a new ThingHud instance.
func NewThingHud(things *Things, cfg *config.Thing, volume *Volume) *ThingHud {
	thing := &ThingHud{}
	thing.ThingBase = NewThingBase(thing, things, cfg, volume)
	return thing
}

// PostMessage sends an ThingEvent instance to the ThingItem's inbox channel for processing.
func (t *ThingHud) PostMessage(_ *ThingEvent) {
}

// StartLoop begins a goroutine that processes incoming events.
func (t *ThingHud) StartLoop() {
}

func (t *ThingHud) StageThinking(playerX float64, playerY float64, playerZ float64) {
}

// GetVertices overrides the default GetVertices to use the HUD render mode.
// The vertex shader will bypass the view matrix completely and place it in screen space.
func (t *ThingHud) GetVertices(tick uint64) (*[]*Face, int, *[]*Face, int, float64, float64) {
	f1, c1, f2, c2, lp, _ := t.ThingBase.GetVertices(tick)
	// RenderMode 3.0 triggers the 2D HUD screen-space bypass in main.vert!
	// It doesn't move in the 3D world at all.
	return f1, c1, f2, c2, lp, RenderModeInterface
}
