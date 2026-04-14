package config

import (
	"github.com/markel1974/godoom/mr_tech/model/geometry"
	"github.com/markel1974/godoom/mr_tech/textures"
	"github.com/markel1974/godoom/mr_tech/utils"
)

// Root represents the root configuration for a level, including sectors, lights, player, scale, and loop settings.
type Root struct {
	Id          string           `json:"id"`
	Sectors     []*Sector        `json:"sectors"`
	Things      []*Thing         `json:"things"`
	Player      *Player          `json:"player"`
	ScaleFactor float64          `json:"scaleFactor"`
	DisableLoop bool             `json:"disableLoop"`
	Vertices    geometry.Polygon `json:"vertices"`
	Volumes     []*Volume        `json:"volumes"`
	Lights      []*ConfigLight   `json:"lights"`
	Full3d      bool             `json:"full3d"`
	textures    textures.ITextures
}

// NewConfigRoot creates a new Root instance with specified sectors, player, lights, scale factor, and loop status.
func NewConfigRoot(sectors []*Sector, player *Player, things []*Thing, scaleFactor float64, disableLoop bool, t textures.ITextures) *Root {
	return &Root{
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
func (r *Root) GetTextures() textures.ITextures {
	return r.textures
}
