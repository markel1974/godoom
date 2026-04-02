package model

import (
	"strings"

	"github.com/markel1974/godoom/mr_tech/model/config"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// Animations is a type that manages a collection of named animations and provides access to these animations.
type Animations struct {
	animations map[string]*textures.Animation
	tex        textures.ITextures
}

// NewAnimations creates a new Animations instance with the provided textures and initializes its animation map.
func NewAnimations(tex textures.ITextures) *Animations {
	return &Animations{
		tex:        tex,
		animations: make(map[string]*textures.Animation),
	}
}

// GetTextures returns the ITextures instance associated with the Animations object.
func (r *Animations) GetTextures() textures.ITextures {
	return r.tex
}

// GetAnimation retrieves or creates an animation based on the given configuration and caches it for reuse.
func (r *Animations) GetAnimation(ca *config.ConfigAnimation) *textures.Animation {
	if ca == nil {
		return textures.NewAnimation(nil, int(config.AnimationKindNone), 1, 1)
	}
	key := strings.Join(ca.Frames, ";")
	animation, ok := r.animations[key]
	if ok {
		return animation
	}
	tex := r.tex.Get(ca.Frames)
	animation = textures.NewAnimation(tex, int(ca.Kind), ca.ScaleW, ca.ScaleH)
	r.animations[key] = animation
	return animation
}
