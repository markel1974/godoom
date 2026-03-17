package model

import (
	"github.com/markel1974/godoom/mr_tech/physics"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// IThing defines an interface for game objects with properties such as ID, position, animation, and lighting,
// and methods for computation and movement handling.
type IThing interface {
	GetId() string

	SetIdentifier(identifier int)

	GetIdentifier() int

	GetKind() ThingType

	GetAABB() *physics.AABB

	GetAnimation() *textures.Animation

	GetPosition() (float64, float64)

	GetLight() *Light

	GetFloorY() float64

	GetCeilY() float64

	GetEntity() *physics.Entity

	Compute(playerX float64, playerY float64)

	GetSector() *Sector

	PhysicsApply()
}
