package model

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/textures"
)

// ThingBullet represents a specialized type of Thing designed to simulate projectile-like behavior in the environment.
type ThingBullet struct {
	*ThingBase
}

// NewThingBullet creates and initializes a new ThingBullet instance with specific properties and links it to the game world.
// cfg specifies the configuration of the bullet, anim defines its animation, and sector represents its initial sector.
// sectors and entities provide references to all sectors and entities in the game world.
func NewThingBullet(cfg *ConfigThing, anim *textures.Animation, sector *Sector, sectors *Sectors, entities *Entities) *ThingBullet {
	p := &ThingBullet{
		ThingBase: NewThingBase(cfg, anim, sector, sectors, entities),
	}

	// Annulla il decadimento inerziale per mantenere una velocità lineare costante
	p.entity.Friction = 1.0
	p.entity.GForce = 1.0

	return p
}

// Compute updates the bullet's direction and handles its collision, potentially triggering its deallocation.
func (t *ThingBullet) Compute(playerX float64, playerY float64) {
	if t.speed == 0 {
		return
	}

	// Calculate the directional vector based on the original firing angle
	dirX := math.Cos(t.angle) * t.speed
	dirY := math.Sin(t.angle) * t.speed

	t.modifyDirection(dirX, dirY)

	// Trigger for impact handling and subsequent deallocation
	if t.entity.Collider != nil {
		// Hit/explosion logic, damage application and entity removal
		t.speed = 0
		t.entity.Invalidate()
	}
}
