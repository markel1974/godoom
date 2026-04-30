package model

import (
	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// Materials is a type that manages a collection of named materials and provides access to these materials.
type Materials struct {
	frames map[string]*textures.Material
	tex    textures.ITextures
	empty  *textures.Material
}

// NewMaterials creates a new Materials instance with the provided textures and initializes its material map.
func NewMaterials(tex textures.ITextures) *Materials {
	return &Materials{
		tex:    tex,
		frames: make(map[string]*textures.Material),
		empty:  textures.NewMaterial(nil, int(config.MaterialKindNone), 1, 1, 0, 0),
	}
}

// GetTextures returns the ITextures instance associated with the Materials object.
func (r *Materials) GetTextures() textures.ITextures {
	return r.tex
}

// GetMaterial retrieves or creates an material based on the given configuration and caches it for reuse.
func (r *Materials) GetMaterial(ca *config.Material) *textures.Material {
	if ca == nil {
		return r.empty
	}
	key := ca.HashKey()
	//key := fmt.Sprintf("%s|%d|%f|%f", strings.Join(ca.Frames, ";"), ca.Kind, ca.ScaleW, ca.ScaleH)
	material, ok := r.frames[key]
	if ok {
		return material
	}
	tex := r.tex.Get(ca.Frames)
	material = textures.NewMaterial(tex, int(ca.Kind), ca.ScaleW, ca.ScaleH, ca.U, ca.V)
	r.frames[key] = material
	return material
}
