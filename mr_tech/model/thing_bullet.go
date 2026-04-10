package model

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/model/config"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// ThingBullet represents a specialized type of Thing designed to simulate projectile-like behavior in the environment.
type ThingBullet struct {
	*ThingBase
}

// NewThingBullet creates and initializes a new ThingBullet instance.
func NewThingBullet(cfg *config.ConfigThing, anim *textures.Animation, volume *Volume, sectors *Volumes, things *Things, pitchRad float64) *ThingBullet {
	pos := cfg.Position
	p := &ThingBullet{
		ThingBase: NewThingBase(cfg, pos, anim, volume, sectors, things),
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

// OnCollide handles the interaction when the bullet collides with another object.
func (t *ThingBullet) OnCollide(other IThing) {
	if enemy, ok := other.(*ThingEnemy); ok {
		_ = enemy
		// enemy.TakeDamage(...)
		// t.SetActive(false)
	}
}
