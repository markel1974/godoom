package config

import (
	"github.com/markel1974/godoom/mr_tech/model/geometry"
	"github.com/markel1974/godoom/mr_tech/utils"
)

// ConfigPlayer represents the configuration of a player in the game, including position, angle, and current sector.
type ConfigPlayer struct {
	*ConfigThing
}

// NewConfigPlayer creates and returns a new instance of ConfigPlayer with the specified position, angle, and sector.
func NewConfigPlayer(position geometry.XYZ, angle float64, height float64, radius float64, mass float64) *ConfigPlayer {
	return &ConfigPlayer{
		&ConfigThing{
			Id:       utils.NextUUId(),
			Position: position,
			Angle:    angle,
			Height:   height,
			Radius:   radius,
			Mass:     mass,
		}}
}
