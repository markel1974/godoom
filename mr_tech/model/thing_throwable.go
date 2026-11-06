package model

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/geometry"
)

// ThingThrowable represents a throwable object in the system, extending the base functionality of ThingBase.
type ThingThrowable struct {
	*ThingBase
}

// NewThingThrowable creates and initializes a new throwable object with specific parameters and assigns its properties.
func NewThingThrowable(things *Things, cfg *config.Thing, volume *Volume) *ThingThrowable {
	thing := &ThingThrowable{}
	thing.ThingBase = NewThingBase(thing, things, cfg, volume)
	cfg.OnBind(thing)
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
	entity := thing.VerticesGetEntity()
	entity.SetVx(dirX * muzzleVelocity)
	entity.SetVy(dirY * muzzleVelocity)
	entity.SetVz(dirZ * muzzleVelocity)
	return thing
}

// IsEqual reports whether other is the same ThingThrowable instance.
func (t *ThingThrowable) IsEqual(other IThing) bool {
	if v, ok := other.(*ThingThrowable); ok && t == v {
		return true
	}
	return false
}

// Jump attempts to make the enemy leap with the given horizontal movement and vertical scaling, reporting success.
func (t *ThingThrowable) Jump(leapX float64, leapY float64, zFactor float64) bool {
	return false
}

// FireHitscan performs a raycast to detect the first intersecting object within a specified direction and range.
func (t *ThingThrowable) FireHitscan(id string, pos geometry.XYZ, force float64, dirX, dirY, dirZ float64) {
}

// LaunchObject spawns a bullet at the specified position, angle, and pitch using predefined physical parameters.
func (t *ThingThrowable) LaunchObject(throwableIndex int, onCollision config.CollisionFunc, onImpact config.ImpactFunc, pos geometry.XYZ, angle, pitch, speed float64) {
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

// HasThinking checks if the ThingThrowable instance has a "thinking" state behavior enabled and returns false by default.
func (t *ThingThrowable) HasThinking() bool {
	return false
}

// StageThinking calculates or updates the state of the `ThingThrowable` instance based on the player's coordinates.
func (t *ThingThrowable) StageThinking(playerX float64, playerY float64, playerZ float64) {
	// Logica eventuale di homing-missile o timeout qui
}
