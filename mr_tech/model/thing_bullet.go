package model

import (
	"fmt"
	"math"

	"github.com/markel1974/godoom/mr_tech/model/config"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// ThingBullet represents a specialized type of Thing designed to simulate projectile-like behavior in the environment.
type ThingBullet struct {
	*ThingBase
}

// NewThingBullet creates and initializes a new ThingBullet instance.
func NewThingBullet(cfg *config.ConfigThing, anim *textures.Animation, volume *Volume, sectors *Volumes, entities *Entities, rawPitch float64) *ThingBullet {
	pos := cfg.Position
	pos.Z = volume.GetMinZ() + 4.0
	p := &ThingBullet{
		ThingBase: NewThingBase(cfg, pos, anim, volume, sectors, entities),
	}

	x, y, z := p.entity.GetCenter()
	fmt.Println("Current bullet position: ", x, y, z, p.volume.GetMinZ(), p.volume.GetMaxZ())

	//p.entity.SetFriction(1.0)
	p.entities.AddThing(p)
	// 1. Normalizzazione del Pitch (da [-5, 5] a radianti)
	// Supponiamo che 5.0 corrisponda a un'inclinazione massima desiderata di 60 gradi (1.047 radianti).
	// Formula: (rawPitch / maxValue) * maxRadian
	maxRadian := 1.047 // Regola questo limite in base al FOV del tuo motore
	pitchRad := (rawPitch / 5.0) * maxRadian

	// 2. Calcolo Sferico 3D VERO
	dirX := math.Cos(p.angle) * math.Cos(pitchRad) * p.speed
	dirY := math.Sin(p.angle) * math.Cos(pitchRad) * p.speed
	dirZ := math.Sin(pitchRad) * p.speed

	const acceleration = 0.15
	p.entity.SetVx(p.entity.GetVx()*(1-acceleration) + (dirX * acceleration))
	p.entity.SetVy(p.entity.GetVy()*(1-acceleration) + (dirY * acceleration))
	p.entity.SetVz(p.entity.GetVz()*(1-acceleration) + (dirZ * acceleration))
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

// PhysicsApply updates the bullet's position based on physics deltas (X, Y, Z)
// and synchronizes its state with the 3D spatial partitioning.
func (t *ThingBullet) PhysicsApply() {
	// 1. Recupero dal motore fisico (Baricentro Reale 3D)
	eX, eY, eZ := t.entity.GetCenter()
	// Calcolo quota base del proiettile
	baseZ := eZ - (t.entity.GetDepth() / 2.0)
	// 2. Calcolo dei delta completi
	tx := (eX - t.position.X) + t.entity.GetVx()
	ty := (eY - t.position.Y) + t.entity.GetVy()
	tz := (baseZ - t.position.Z) + t.entity.GetVz()
	if tx == 0 && ty == 0 && tz == 0 {
		return
	}
	// 3. Risoluzione dei vincoli ambientali 3D (Bounces e Portali)
	viewX := t.position.X
	viewY := t.position.Y
	viewZ := t.position.Z
	zBottom := viewZ
	zTop := viewZ + t.height
	zMinLimit := t.volume.GetMinZ()
	zMaxLimit := t.volume.GetMaxZ()
	// Rimbalzo sui muri (Broad & Narrow phase)
	velX, velY, velZ, _ := t.wallPhysics.AdjustVelocity(viewX, viewY, viewZ, tx, ty, tz, zTop, zBottom, zMinLimit, zMaxLimit, t.radius, true)
	if math.Abs(velX) > minMovement || math.Abs(velX) > minMovement || math.Abs(velX) > minMovement {
		t.entity.SetVx(velX)
		t.entity.SetVy(velY)
		t.entity.SetVz(velZ)
		// 4. Aggiornamento posizione logica
		t.position.X += velX
		t.position.Y += velY
		t.position.Z += velZ
		// 5. Aggiornamento AABB Tree (basato sul baricentro per prevenire cambi errati)
		bulletBaseZ := t.position.Z
		bulletTopZ := t.position.Z + t.height
		const bulletStep = 0.0
		if newVolume := t.volumes.SearchVolume3d(t.volume, t.position.X, t.position.Y, bulletBaseZ, bulletTopZ, bulletStep); newVolume != nil && newVolume != t.volume {
			t.volume = newVolume
		}
		t.entities.UpdateThing(t, t.position.X, t.position.Y, t.position.Z)
	} else {
		t.entity.Stop()
	}
}

// OnCollide handles the interaction when the bullet collides with another object.
func (t *ThingBullet) OnCollide(other IThing) {
	if enemy, ok := other.(*ThingEnemy); ok {
		_ = enemy
		// enemy.TakeDamage(...)
		// t.SetActive(false)
	}
}

// IsActive checks if the ThingBullet is currently active and operational.
func (t *ThingBullet) IsActive() bool {
	return t.isActive
}
