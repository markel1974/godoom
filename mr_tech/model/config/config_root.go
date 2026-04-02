package config

import (
	"github.com/markel1974/godoom/mr_tech/textures"
)

// ConfigRoot represents the root configuration for a level, including sectors, lights, player, scale, and loop settings.
type ConfigRoot struct {
	Sectors     []*ConfigSector `json:"sectors"`
	Things      []*ConfigThing  `json:"things"`
	Player      *ConfigPlayer   `json:"player"`
	ScaleFactor float64         `json:"scaleFactor"`
	DisableLoop bool            `json:"disableLoop"`
	Textures    textures.ITextures
}

// NewConfigRoot creates a new ConfigRoot instance with specified sectors, player, lights, scale factor, and loop status.
func NewConfigRoot(sectors []*ConfigSector, player *ConfigPlayer, things []*ConfigThing, scaleFactor float64, disableLoop bool, t textures.ITextures) *ConfigRoot {
	return &ConfigRoot{
		Sectors:     sectors,
		Player:      player,
		Things:      things,
		ScaleFactor: scaleFactor,
		DisableLoop: disableLoop,
		Textures:    t,
	}
}
