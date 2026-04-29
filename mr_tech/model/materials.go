package model

import (
	"fmt"
	"strings"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// Materials is a type that manages a collection of named animations and provides access to these animations.
type Materials struct {
	frames map[string]*textures.Material
	tex    textures.ITextures
	gScale float64
}

// NewMaterials creates a new Materials instance with the provided textures and initializes its sprite map.
func NewMaterials(tex textures.ITextures, gScale float64) *Materials {
	if gScale == 0 {
		gScale = 1
	}
	return &Materials{
		tex:    tex,
		gScale: gScale,
		frames: make(map[string]*textures.Material),
	}
}

// GetTextures returns the ITextures instance associated with the Materials object.
func (r *Materials) GetTextures() textures.ITextures {
	return r.tex
}

// GetMaterial retrieves or creates an sprite based on the given configuration and caches it for reuse.
func (r *Materials) GetMaterial(ca *config.Material) *textures.Material {
	if ca == nil {
		return textures.NewMaterial(nil, int(config.AnimationKindNone), r.gScale, 1, 1)
	}
	key := fmt.Sprintf("%s|%d|%f|%f", strings.Join(ca.Frames, ";"), ca.Kind, ca.ScaleW, ca.ScaleH)
	animation, ok := r.frames[key]
	if ok {
		return animation
	}
	tex := r.tex.Get(ca.Frames)
	animation = textures.NewMaterial(tex, int(ca.Kind), r.gScale, ca.ScaleW, ca.ScaleH)
	r.frames[key] = animation
	return animation
}
