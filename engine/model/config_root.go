package model

import (
	"strings"

	"github.com/markel1974/godoom/engine/textures"
)

// ConfigRoot represents the root configuration for a level, including sectors, lights, player, scale, and loop settings.
type ConfigRoot struct {
	Sectors     []*ConfigSector `json:"sectors"`
	Things      []*ConfigThing  `json:"things"`
	Player      *ConfigPlayer   `json:"player"`
	ScaleFactor float64         `json:"scaleFactor"`
	DisableLoop bool            `json:"disableLoop"`
	Textures    textures.ITextures
	animations  map[string]*textures.Animation
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
		animations:  make(map[string]*textures.Animation),
	}
}

// GetAnimation retrieves an animation from the cache or creates a new one using the provided texture sources.
func (r *ConfigRoot) GetAnimation(ca *ConfigAnimation) *textures.Animation {
	if ca == nil {
		return textures.NewAnimation(nil, int(AnimationKindNone))
	}
	key := strings.Join(ca.Frames, ";")
	animation, ok := r.animations[key]
	if ok {
		return animation
	}
	tex := r.Textures.Get(ca.Frames)
	animation = textures.NewAnimation(tex, int(ca.Kind))
	r.animations[key] = animation
	return animation
}
