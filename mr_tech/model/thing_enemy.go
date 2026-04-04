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
	e := &ThingEnemy{
		ThingBase: NewThingBase(cfg, anim, volume, sectors, entities),
		active:    false,
	}
	e.entities.AddThing(e)
	return e
}

// Compute updates the Thing's direction and position based on the player's coordinates and its current speed.
func (t *ThingEnemy) Compute(playerX float64, playerY float64) {
	dx := playerX - t.position.X
	dy := playerY - t.position.Y
	dist := math.Sqrt(dx*dx + dy*dy)

	if !t.active {
		if dist < 25.0 {
			t.active = true
		}
		return
	}

	invDist := 1.0 / dist
	dirX := dx * invDist * t.speed
	dirY := dy * invDist * t.speed
	t.modifyDirection(dirX, dirY)
}

// modifyDirection adjusts the entity's velocity based on the provided direction vector and applies acceleration and friction.
func (t *ThingEnemy) modifyDirection(dirX, dirY float64) {
	const acceleration = 0.15
	t.entity.SetVx(t.entity.GetVx()*(1-acceleration) + (dirX * acceleration))
	t.entity.SetVy(t.entity.GetVy()*(1-acceleration) + (dirY * acceleration))
	//if t.entity.GForce == 0 {
	//	t.entity.GForce = 1.0
	//}
	//if t.entity.Friction < 0.2 {
	//	t.entity.Friction = 0.99
	//}
}
