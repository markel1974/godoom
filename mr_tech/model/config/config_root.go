package config

import (
	"github.com/markel1974/godoom/mr_tech/model/geometry"
	"github.com/markel1974/godoom/mr_tech/textures"
	"github.com/markel1974/godoom/mr_tech/utils"
)

// ConfigRoot represents the root configuration for a level, including sectors, lights, player, scale, and loop settings.
type ConfigRoot struct {
	Id          string           `json:"id"`
	Sectors     []*ConfigSector  `json:"sectors"`
	Things      []*ConfigThing   `json:"things"`
	Player      *ConfigPlayer    `json:"player"`
	ScaleFactor float64          `json:"scaleFactor"`
	DisableLoop bool             `json:"disableLoop"`
	Vertices    geometry.Polygon `json:"vertices"`
	textures    textures.ITextures
}

// NewConfigRoot creates a new ConfigRoot instance with specified sectors, player, lights, scale factor, and loop status.
func NewConfigRoot(sectors []*ConfigSector, player *ConfigPlayer, things []*ConfigThing, scaleFactor float64, disableLoop bool, t textures.ITextures) *ConfigRoot {
	return &ConfigRoot{
		Id:          utils.NextUUId(),
		Sectors:     sectors,
		Player:      player,
		Things:      things,
		ScaleFactor: scaleFactor,
		DisableLoop: disableLoop,
		textures:    t,
	}
}

// GetTextures retrieves the textures associated with the configuration root.
func (r *ConfigRoot) GetTextures() textures.ITextures {
	return r.textures
}
