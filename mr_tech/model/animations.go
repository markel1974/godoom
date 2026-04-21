package model

import (
	"fmt"
	"strings"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// Animations is a type that manages a collection of named animations and provides access to these animations.
type Animations struct {
	animations map[string]*textures.Animation
	tex        textures.ITextures
	gScale     float64
}

// NewAnimations creates a new Animations instance with the provided textures and initializes its animation map.
func NewAnimations(tex textures.ITextures, gScale float64) *Animations {
	if gScale == 0 {
		gScale = 1
	}
	return &Animations{
		tex:        tex,
		gScale:     gScale,
		animations: make(map[string]*textures.Animation),
	}
}

// GetTextures returns the ITextures instance associated with the Animations object.
func (r *Animations) GetTextures() textures.ITextures {
	return r.tex
}

// GetAnimation retrieves or creates an animation based on the given configuration and caches it for reuse.
func (r *Animations) GetAnimation(ca *config.Animation) *textures.Animation {
	if ca == nil {
		return textures.NewAnimation(nil, int(config.AnimationKindNone), r.gScale, 1, 1)
	}
	key := fmt.Sprintf("%s|%d|%f|%f", strings.Join(ca.Frames, ";"), ca.Kind, ca.ScaleW, ca.ScaleH)
	animation, ok := r.animations[key]
	if ok {
		return animation
	}
	tex := r.tex.Get(ca.Frames)
	animation = textures.NewAnimation(tex, int(ca.Kind), r.gScale, ca.ScaleW, ca.ScaleH)
	r.animations[key] = animation
	return animation
}
