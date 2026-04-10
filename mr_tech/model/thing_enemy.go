package model

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/model/config"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// ThingEnemy represents a physical or logical entity in the environment with attributes like position, mass, and associated data.
type ThingEnemy struct {
	*ThingBase
	active bool
}

// NewThingEnemy creates and initializes a new ThingEnemy instance with the specified configuration, animation, sector, and entities.
func NewThingEnemy(cfg *config.ConfigThing, anim *textures.Animation, volume *Volume, sectors *Volumes, entities *Entities) *ThingEnemy {
	pos := cfg.Position
	pos.Z = volume.GetMinZ()
	e := &ThingEnemy{
		ThingBase: NewThingBase(cfg, pos, anim, volume, sectors, entities),
		active:    false,
	}
	e.entities.AddThing(e)
	return e
}

// Compute updates the Thing's direction and position based on the player's coordinates and its current speed.
func (t *ThingEnemy) Compute(playerX float64, playerY float64, playerZ float64) {
	dx := playerX - t.pos.X
	dy := playerY - t.pos.Y
	dz := playerZ - t.pos.Z
	// 1. Attivazione (Aggro): Utilizza la distanza sferica 3D
	dist3D := math.Sqrt(dx*dx + dy*dy + dz*dz)
	if !t.active {
		if dist3D < 25.0 {
			t.active = true
		}
		return
	}
	// 2. Inseguimento Terrestre: Utilizza la proiezione cilindrica 2D
	dist2D := math.Sqrt(dx*dx + dy*dy)
	if dist2D > 0.001 {
		invDist := 1.0 / dist2D
		// Vettore direzionale puro (Normalizzato -1.0 / 1.0)
		nx := dx * invDist
		ny := dy * invDist
		const forceScale = 100.0
		fx := nx * forceScale * t.speed
		fy := ny * forceScale * t.speed
		t.entity.AddForce(fx, fy, 0.0)
	}
}
