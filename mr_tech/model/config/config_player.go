package config

import (
	"github.com/markel1974/godoom/mr_tech/model/geometry"
	"github.com/markel1974/godoom/mr_tech/utils"
)

// ConfigPlayer represents the configuration of a player in the game, including position, angle, and current sector.
type ConfigPlayer struct {
	Id       string      `json:"id"`
	Position geometry.XY `json:"position"`
	Angle    float64     `json:"angle"`
	Radius   float64     `json:"radius"`
	Mass     float64     `json:"mass"`
}

// NewConfigPlayer creates and returns a new instance of ConfigPlayer with the specified position, angle, and sector.
func NewConfigPlayer(position geometry.XY, angle float64, radius float64, mass float64) *ConfigPlayer {
	return &ConfigPlayer{
		Id:       utils.NextUUId(),
		Position: position,
		Angle:    angle,
		Radius:   radius,
		Mass:     mass,
	}
}
