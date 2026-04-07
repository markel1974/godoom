package model

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/model/config"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// ThingBullet represents a specialized type of Thing designed to simulate projectile-like behavior in the environment.
type ThingBullet struct {
	*ThingBase
	floorStartY float64
}

// NewThingBullet creates and initializes a new ThingBullet instance.
func NewThingBullet(cfg *config.ConfigThing, anim *textures.Animation, volume *Volume, sectors *Volumes, entities *Entities) *ThingBullet {
	p := &ThingBullet{
		ThingBase:   NewThingBase(cfg, anim, volume, sectors, entities),
		floorStartY: volume.GetMinZ(),
	}
	// Annulla il decadimento inerziale per mantenere una velocità lineare costante
	p.entity.SetFriction(1.0) // 1.0 = nessuna perdita di velocità su X/Y
	p.entity.SetGForce(1.0)
	p.entities.AddThing(p)

	// Calculate the directional vector based on the original firing angle
	dirX := math.Cos(p.angle) * p.speed
	dirY := math.Sin(p.angle) * p.speed

	const acceleration = 0.15
	p.entity.SetVx(p.entity.GetVx()*(1-acceleration) + (dirX * acceleration))
	p.entity.SetVy(p.entity.GetVy()*(1-acceleration) + (dirY * acceleration))
	return p
}

func (t *ThingBullet) GetFloorY() float64 {
	velSq := (t.entity.GetVx() * t.entity.GetVx()) + (t.entity.GetVy() * t.entity.GetVy())
	if velSq <= 0.01 || t.speed <= 0 {
		return t.volume.GetMinZ()
	}
	ratio := math.Sqrt(velSq) / t.speed
	if ratio <= 0 {
		return t.volume.GetMinZ()
	}
	if ratio > 1.0 {
		ratio = 1.0
	}
	return t.floorStartY * ratio
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

	if math.Abs(tx) > minMovement || math.Abs(ty) > minMovement || math.Abs(tz) > minMovement {
		// 3. Risoluzione dei vincoli ambientali 3D (Bounces e Portali)
		vx, vy, vz := t.adjustPassage(tx, ty, tz)

		// 4. Aggiornamento posizione logica
		t.position.X += vx
		t.position.Y += vy
		t.position.Z += vz

		// 5. Aggiornamento AABB Tree (basato sul baricentro per prevenire cambi errati)
		bulletBaseZ := t.position.Z
		bulletTopZ := t.position.Z + t.height
		const bulletStep = 0.0
		if newVolume := t.volumes.SearchVolume3d(t.volume, t.position.X, t.position.Y, bulletBaseZ, bulletTopZ, bulletStep); newVolume != nil && newVolume != t.volume {
			t.volume = newVolume
		}

		t.entities.UpdateThing(t, t.position.X, t.position.Y, t.position.Z)
	}
}

// adjustPassage resolves the 3D trajectory of the bullet, handling bounces via the spatial tree.
func (t *ThingBullet) adjustPassage(velX, velY, velZ float64) (float64, float64, float64) {
	viewZ := t.position.Z
	bottom := viewZ
	top := viewZ + t.height
	viewX, viewY := t.position.X, t.position.Y
	pX := viewX + velX
	pY := viewY + velY
	pZ := viewZ + velZ

	// Rimbalzo sui muri (Broad & Narrow phase)
	velX, velY, velZ, _ = t.slider.EffectBounce(viewX, viewY, viewZ, pX, pY, pZ, velX, velY, velZ, top, bottom, t.radius)
	//if !changed {
	//	return velX, velY, velZ
	//}

	t.entity.SetVx(velX)
	t.entity.SetVy(velY)
	t.entity.SetVz(velZ)

	// Clipping e Rimbalzo Pavimento/Soffitto
	nextZ := viewZ + velZ
	minZ := t.volume.GetMinZ()
	maxZ := t.volume.GetMaxZ()

	if nextZ < minZ {
		// Rimbalzo sul pavimento
		velZ = math.Abs(velZ) * 0.8 // Perde un 20% di energia
		t.entity.SetVz(velZ)
	} else if nextZ+t.height > maxZ {
		// Rimbalzo sul soffitto
		velZ = -math.Abs(velZ) * 0.8
		t.entity.SetVz(velZ)
	}

	return velX, velY, velZ
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
