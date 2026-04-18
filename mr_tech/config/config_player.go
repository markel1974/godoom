package config

import (
	"github.com/markel1974/godoom/mr_tech/model/geometry"
	"github.com/markel1974/godoom/mr_tech/utils"
)

// Player represents a specialized game entity with inherited attributes and behaviors from the Thing type.
type Player struct {
	*Thing
	BobbingMaxAmplitude float64
	BobbingIdleDrift    float64
	BobbingStrideLength float64
	BobbingSpeedLerp    float64
	BobbingAmpLerp      float64
}

// NewConfigPlayer creates and returns a new Player instance configured with the given position, angle, height, radius, and mass.
func NewConfigPlayer(position geometry.XYZ, angle float64, height float64, radius float64, mass float64) *Player {
	return &Player{
		Thing: &Thing{
			Id:       utils.NextUUId(),
			Position: position,
			Angle:    angle,
			Height:   height,
			Radius:   radius,
			Mass:     mass,
		},
		BobbingMaxAmplitude: 0.9,
		BobbingIdleDrift:    0.03,
		BobbingStrideLength: 0.015,
		BobbingSpeedLerp:    0.15,
		BobbingAmpLerp:      0.10,
	}
}
