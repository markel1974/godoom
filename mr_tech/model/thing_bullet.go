package model

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// ThingBullet represents a specialized type of Thing designed to simulate projectile-like behavior in the environment.
type ThingBullet struct {
	*ThingBase
}

// NewThingBullet creates and initializes a new ThingBullet instance.
func NewThingBullet(things *Things, cfg *config.Thing, anim *textures.Animation, volume *Volume, pitchRad float64) *ThingBullet {
	pos := cfg.Position
	p := &ThingBullet{
		ThingBase: NewThingBase(things, cfg, pos, anim, volume),
	}
	// Sovrascriviamo il maxStep della base: i proiettili non scavalcano i gradini
	p.maxStep = 0.0
	p.things.AddThing(p)
	// 1. Normalizzazione del Pitch (da [-5, 5] a radianti)
	// 2. Vettore Direzionale 3D normalizzato
	dirX := math.Cos(p.angle) * math.Cos(pitchRad)
	dirY := math.Sin(p.angle) * math.Cos(pitchRad)
	dirZ := math.Sin(pitchRad)
	// 3. Muzzle Velocity (Iniezione istantanea di velocità)
	// Essendo il frame 0, impostiamo direttamente la velocità vettoriale.
	// Seleziona un moltiplicatore appropriato per la velocità dei tuoi proiettili (es. 50.0)
	muzzleVelocity := p.speed * 5.0
	p.entity.SetVx(dirX * muzzleVelocity)
	p.entity.SetVy(dirY * muzzleVelocity)
	p.entity.SetVz(dirZ * muzzleVelocity)

	return p
}

func (t *ThingBullet) PostMessage(ec *ThingEvent) {
	t.inbox <- ec
}

func (t *ThingBullet) StartLoop() {
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

func (t *ThingBullet) GetMaxZ() float64 {
	_, _, z := t.entity.GetCenter()
	return z
}

func (t *ThingBullet) GetMinZ() float64 {
	_, _, z := t.entity.GetCenter()
	return z
}

func (t *ThingBullet) Compute(playerX float64, playerY float64, playerZ float64) {
	// Logica eventuale di homing-missile o timeout qui
}
