package config

import (
	"github.com/markel1974/godoom/mr_tech/model/geometry"
	"github.com/markel1974/godoom/mr_tech/textures"
	"github.com/markel1974/godoom/mr_tech/utils"
)

// Root represents the top-level configuration container, including sectors, things, player, and rendering properties.
type Root struct {
	Id          string           `json:"id"`
	Sectors     []*Sector        `json:"sectors"`
	Things      []*Thing         `json:"things"`
	Player      *Player          `json:"player"`
	ScaleFactor float64          `json:"scaleFactor"`
	Vertices    geometry.Polygon `json:"vertices"`
	Volumes     []*Volume        `json:"volumes"`
	Lights      []*Light         `json:"lights"`
	Full3d      bool             `json:"full3d"`
	textures    textures.ITextures
}

// NewConfigRoot creates and initializes a new Root object with the specified sectors, player, things, and configuration.
func NewConfigRoot(sectors []*Sector, player *Player, things []*Thing, scaleFactor float64, t textures.ITextures) *Root {
	return &Root{
		Id:          utils.NextUUId(),
		Sectors:     sectors,
		Player:      player,
		Things:      things,
		ScaleFactor: scaleFactor,
		textures:    t,
	}
}

// GetTextures retrieves the texture collection associated with the Root configuration.
func (cfg *Root) GetTextures() textures.ITextures {
	return cfg.textures
}

// Scale adjusts the dimensions of all entities in the Root object by the specified scale factor. If scale is 0, defaults to 1.
func (cfg *Root) Scale(scale float64) {
	if scale == 0 || scale == 1 {
		return
	}
	cfg.Player.Scale(scale)
	for _, sector := range cfg.Sectors {
		sector.Scale(scale)
	}
	for _, volume := range cfg.Volumes {
		volume.Scale(scale)
	}
	for _, thing := range cfg.Things {
		thing.Scale(scale)
	}
	for _, light := range cfg.Lights {
		light.Pos.Scale(scale)
	}
}
