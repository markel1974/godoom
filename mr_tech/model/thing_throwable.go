package model

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// ThingThrowable represents a throwable object in the system, extending the base functionality of ThingBase.
type ThingThrowable struct {
	*ThingBase
}

// NewThingThrowable creates and initializes a new throwable object with specific parameters and assigns its properties.
func NewThingThrowable(things *Things, cfg *config.Thing, anim *textures.Material, volume *Volume) *ThingThrowable {
	pos := cfg.Position
	thing := &ThingThrowable{}
	thing.ThingBase = NewThingBase(thing, things, cfg, pos, anim, volume)
	// Sovrascriviamo il maxStep della base: i proiettili non scavalcano i gradini
	thing.maxStep = 0.0
	// 1. Normalizzazione del Pitch (da [-5, 5] a radianti)
	// 2. Vettore Direzionale 3D normalizzato
	dirX := math.Cos(thing.angle) * math.Cos(cfg.Pitch)
	dirY := math.Sin(thing.angle) * math.Cos(cfg.Pitch)
	dirZ := math.Sin(cfg.Pitch)
	// 3. Muzzle Velocity (Iniezione istantanea di velocità)
	// Essendo il frame 0, impostiamo direttamente la velocità vettoriale.
	// Seleziona un moltiplicatore appropriato per la velocità dei tuoi proiettili (es. 50.0)
	muzzleVelocity := thing.speed * 5.0
	thing.entity.SetVx(dirX * muzzleVelocity)
	thing.entity.SetVy(dirY * muzzleVelocity)
	thing.entity.SetVz(dirZ * muzzleVelocity)
	return thing
}

// PostMessage sends a ThingEvent to the ThingThrowable's inbox channel for processing in the event loop.
func (t *ThingThrowable) PostMessage(ec *ThingEvent) {
	t.inbox <- ec
}

// StartLoop initializes a goroutine to process events from the inbox channel or terminate when signaled via the done channel.
func (t *ThingThrowable) StartLoop() {
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

// GetMaxZ retrieves the maximum Z-coordinate of the center for the associated entity.
func (t *ThingThrowable) GetMaxZ() float64 {
	_, _, z := t.entity.GetCenter()
	return z
}

// GetMinZ returns the minimum Z value of the ThingThrowable's entity center's position.
func (t *ThingThrowable) GetMinZ() float64 {
	_, _, z := t.entity.GetCenter()
	return z
}

// StageThinking calculates or updates the state of the `ThingThrowable` instance based on the player's coordinates.
func (t *ThingThrowable) StageThinking(playerX float64, playerY float64, playerZ float64) {
	// Logica eventuale di homing-missile o timeout qui
}
