package things

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/model/volumes"
)

// Throwable represents a throwable object in the system, extending the base functionality of Base.
type Throwable struct {
	*Base
}

// NewThingThrowable creates and initializes a new throwable object with specific parameters and assigns its properties.
func NewThingThrowable(things IThingHandler, cfg *config.Thing, volume *volumes.Volume) *Throwable {
	throwable := &Throwable{}
	throwable.Base = NewBase(throwable, things, cfg, volume)
	cfg.OnBind(throwable)
	// Sovrascriviamo il maxStep della base: i proiettili non scavalcano i gradini
	throwable.maxStep = 0.0
	// Normalizzazione del Pitch (da [-5, 5] a radianti)
	// Vettore Direzionale 3D normalizzato
	dirX := math.Cos(throwable.angle) * math.Cos(cfg.Pitch)
	dirY := math.Sin(throwable.angle) * math.Cos(cfg.Pitch)
	dirZ := math.Sin(cfg.Pitch)
	// 3. Muzzle Velocity (Iniezione istantanea di velocità)
	// Essendo il frame 0, impostiamo direttamente la velocità vettoriale.
	// Seleziona un moltiplicatore appropriato per la velocità dei tuoi proiettili (es. 50.0)
	muzzleVelocity := throwable.speed * 5.0
	entity := throwable.VerticesGetEntity()
	//entity.SetRotationEuler(0, cfg.Pitch, throwable.angle)
	entity.SetRotationEuler(0, 0, 0)
	entity.SetVx(dirX * muzzleVelocity)
	entity.SetVy(dirY * muzzleVelocity)
	entity.SetVz(dirZ * muzzleVelocity)
	return throwable
}

// IsEqual reports whether other is the same Throwable instance.
func (t *Throwable) IsEqual(other volumes.IThing) bool {
	if v, ok := other.(*Throwable); ok && t == v {
		return true
	}
	return false
}

// Jump attempts to make the enemy leap with the given horizontal movement and vertical scaling, reporting success.
func (t *Throwable) Jump(leapX float64, leapY float64, zFactor float64) bool {
	return false
}

// FireHitscan performs a raycast to detect the first intersecting object within a specified direction and range.
func (t *Throwable) FireHitscan(id string, pos geometry.XYZ, force float64, dirX, dirY, dirZ float64) {
}

// LaunchObject spawns a bullet at the specified position, angle, and pitch using predefined physical parameters.
func (t *Throwable) LaunchObject(throwableIndex int, onCollision config.CollisionFunc, onImpact config.ImpactFunc, pos geometry.XYZ, angle, pitch, speed float64) {
}

// PostMessage sends a ThingEvent to the Throwable's inbox channel for processing in the event loop.
func (t *Throwable) PostMessage(ec *volumes.ThingEvent) {
	t.inbox <- ec
}

// StartLoop initializes a goroutine to process events from the inbox channel or terminate when signaled via the done channel.
func (t *Throwable) StartLoop() {
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

// HasThinking checks if the Throwable instance has a "thinking" state behavior enabled and returns false by default.
func (t *Throwable) HasThinking() bool {
	return false
}

// StageThinking calculates or updates the state of the `ThingThrowable` instance based on the player's coordinates.
func (t *Throwable) StageThinking(playerX float64, playerY float64, playerZ float64) {
	// Logica eventuale di homing-missile o timeout qui
}
