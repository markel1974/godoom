package config

import "github.com/markel1974/godoom/mr_tech/model/textures"

// Materials is a type that manages a collection of named materials and provides access to these materials.
type Materials struct {
	container map[string]*textures.Material
	handler   textures.ITextures
	empty     *textures.Material
}

// NewMaterials creates a new Materials instance with the provided textures and initializes its material map.
func NewMaterials(handler textures.ITextures) *Materials {
	return &Materials{
		handler:   handler,
		container: make(map[string]*textures.Material),
		empty:     textures.NewMaterial("", nil, int(MaterialKindNone), 0, 0),
	}
}

// Setup initializes the Materials collection and prepares it for use, returning an error if setup fails.
func (r *Materials) Setup() error {
	return nil
}

// GetTextures returns the ITextures instance associated with the Materials object.
func (r *Materials) GetTextures() textures.ITextures {
	return r.handler
}

// GetEmptyMaterial retrieves the default empty material instance stored in the Materials object.
func (r *Materials) GetEmptyMaterial() *textures.Material {
	return r.empty
}

// GetMaterial retrieves or creates an material based on the given configuration and caches it for reuse.
func (r *Materials) GetMaterial(ca *Material) *textures.Material {
	key := ca.HashKey()
	material, ok := r.container[key]
	if ok {
		return material
	}
	tex := r.handler.GetMulti(ca.Frames)
	material = textures.NewMaterial(ca.Shader, tex, int(ca.Kind), ca.U, ca.V)
	material.SetBlendMode(ca.BlendMode)
	material.SetProperties(
		ca.CullMode, ca.DepthWrite, ca.AlphaTest,
		ca.IsFog, ca.FogColor, ca.PolygonOffset,
		ca.Sort, ca.NoMipmaps, ca.NoPicMip,
		ca.EntityMergeable, ca.DeformVertexes,
		ca.SurfaceParms, ca.SkyParms, ca.CloudParms,
		ca.FogGen, ca.IsLightning, ca.IsSky, ca.IsBacksided,
	)
	r.container[key] = material
	return material
}
