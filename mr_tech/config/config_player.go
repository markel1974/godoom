package config

import (
	"github.com/markel1974/godoom/mr_tech/model/geometry"
	"github.com/markel1974/godoom/mr_tech/utils"
)

// Player represents a specialized game entity with inherited attributes and behaviors from the Thing type.
type Player struct {
	*Thing
}

// NewConfigPlayer2d creates and returns a new Player instance with the specified position, angle, height, radius, and mass.
func NewConfigPlayer2d(position geometry.XY, angle float64, height float64, radius float64, mass float64) *Player {
	return &Player{
		&Thing{
			Id:       utils.NextUUId(),
			Position: geometry.XYZ{X: position.X, Y: position.Y, Z: 0},
			Angle:    angle,
			Height:   height,
			Radius:   radius,
			Mass:     mass,
			HasZPos:  false,
		}}
}

// NewConfigPlayer3d creates and returns a new Player instance configured with the given position, angle, height, radius, and mass.
func NewConfigPlayer3d(position geometry.XYZ, angle float64, height float64, radius float64, mass float64) *Player {
	return &Player{
		&Thing{
			Id:       utils.NextUUId(),
			Position: position,
			Angle:    angle,
			Height:   height,
			Radius:   radius,
			Mass:     mass,
			HasZPos:  true,
		}}
}
