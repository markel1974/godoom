package model

import (
	"math"

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
func (t *ThingHud) PostMessage(ec *ThingEvent) {
	t.inbox <- ec
}

// StartLoop begins a goroutine that processes incoming events.
func (t *ThingHud) StartLoop() {
	go func() {
		for {
			select {
			case evt := <-t.inbox:
				switch evt.GetKind() {
				case StageThinking:
					t.StageThinking(evt.GetCoords())

					// Stick it right in front of the player's camera!
					player := t.things.GetPlayer()
					if player != nil {
						px, py, pz := player.GetVisualPosition()
						angle, pitch := player.GetAngle(), player.GetPitch()

						// Distance from camera (e.g., 20 units)
						dist := 20.0

						// Compute direction vector from angle and pitch
						dirX := math.Cos(pitch) * math.Cos(angle)
						dirY := math.Cos(pitch) * math.Sin(angle)
						dirZ := math.Sin(pitch)

						// Position the HUD element
						nx := px + dirX*dist
						ny := py + dirY*dist
						nz := pz + 40.0 + dirZ*dist // Adjust +40.0 for player eye height

						t.GetEntity().MoveTo(nx, ny, nz)
					}

				case StageCompute:
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

func (t *ThingHud) StageThinking(playerX float64, playerY float64, playerZ float64) {
}
