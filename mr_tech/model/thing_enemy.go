package model

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/textures"
)

// ThingEnemy represents a physical or logical entity in the environment with attributes like position, mass, and associated data.
type ThingEnemy struct {
	*ThingBase
}

// NewThingEnemy creates and initializes a new ThingEnemy instance with the specified configuration, animation, sector, and entities.
func NewThingEnemy(cfg *ConfigThing, anim *textures.Animation, sector *Sector, sectors *Sectors, entities *Entities) *ThingEnemy {
	e := &ThingEnemy{
		ThingBase: NewThingBase(cfg, anim, sector, sectors, entities),
	}
	return e
}

// Compute updates the Thing's direction and position based on the player's coordinates and its current speed.
func (t *ThingEnemy) Compute(playerX float64, playerY float64) {
	if t.speed == 0 {
		return
	}
	dx := playerX - t.position.X
	dy := playerY - t.position.Y
	dist := math.Sqrt(dx*dx + dy*dy)
	if dist < 25.0 {
		invDist := 1.0 / dist
		dirX := dx * invDist * t.speed
		dirY := dy * invDist * t.speed
		t.modifyDirection(dirX, dirY)
	}
}
