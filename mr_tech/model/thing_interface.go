package model

import (
	"github.com/markel1974/godoom/mr_tech/config"
)

// ThingInterface is a specialized wrapper around ThingBase, representing a HUD entity that follows the camera.
type ThingInterface struct {
	*ThingBase
}

// NewThingInterface creates a new ThingInterface instance.
func NewThingInterface(things *Things, cfg *config.Thing, volume *Volume) *ThingInterface {
	thing := &ThingInterface{}
	thing.ThingBase = NewThingBase(thing, things, cfg, volume)
	return thing
}

// PostMessage sends an ThingEvent instance to the ThingItem's inbox channel for processing.
func (t *ThingInterface) PostMessage(_ *ThingEvent) {
}

// StartLoop begins a goroutine that processes incoming events.
func (t *ThingInterface) StartLoop() {
}

func (t *ThingInterface) StageThinking(playerX float64, playerY float64, playerZ float64) {
}

// GetVertices overrides the default GetVertices to use the HUD render mode.
// The vertex shader will bypass the view matrix completely and place it in screen space.
func (t *ThingInterface) GetVertices(tick uint64) (*[]*Face, int, *[]*Face, int, float64, float64) {
	//TODO posizione assoluta e' un pezzo dell'interfaccia
	t.GetEntity().MoveTo(0, 0, 0)
	f1, c1, f2, c2, lp, _ := t.ThingBase.GetVertices(tick)
	// Usiamo 1.1 (Spherical Billboard) in modo che lo shader originale lo processi correttamente
	return f1, c1, f2, c2, lp, RenderModeInterface
}
