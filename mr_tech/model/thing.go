package model

import (
	"github.com/markel1974/godoom/mr_tech/textures"
)

// IThing defines an interface for game objects with properties such as ID, position, animation, and lighting,
// and methods for computation and movement handling.
type IThing interface {
	GetId() string

	GetAnimation() *textures.Animation

	GetPosition() XY

	GetLight() *Light

	GetFloorY() float64

	GetCeilY() float64

	Compute(playerX float64, playerY float64)

	MoveApply(tx float64, ty float64)

	MoveEntityApply()
}
