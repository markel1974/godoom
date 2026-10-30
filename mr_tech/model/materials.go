package model

import (
	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/textures"
)

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
		empty:     textures.NewMaterial("empty", nil, int(config.MaterialKindNone)),
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
func (r *Materials) GetMaterial(ca *config.Material) *textures.Material {
	key := ca.HashKey()
	material, ok := r.container[key]
	if ok {
		return material
	}
	material = textures.NewMaterial(ca.Name, r.handler.GetFrames(ca.Frames), int(ca.Kind))
	material.SetUV(ca.U, ca.V)
	material.SetBlendMode(ca.BlendMode)
	material.SetCullMode(ca.CullMode)
	material.SetDepthWrite(ca.DepthWrite)
	material.SetAlphaTest(ca.AlphaTest)
	material.SetFog(ca.IsFog)
	material.SetFogColor(ca.FogColor)
	material.SetPolygonOffset(ca.PolygonOffset)
	material.SetSort(ca.Sort)
	material.SetNoMipmaps(ca.NoMipmaps)
	material.SetNoPicMip(ca.NoPicMip)
	material.SetEntityMergeable(ca.EntityMergeable)
	material.SetDeformVertexes(ca.DeformVertexes)
	material.SetSurfaceParms(ca.SurfaceParms)
	material.SetSkyParms(ca.SkyParms)
	material.SetCloudParms(ca.CloudParms)
	material.SetFogGen(ca.FogGen)
	material.SetLightning(ca.IsLightning)
	material.SetSky(ca.IsSky)
	material.SetBacksided(ca.IsBacksided)
	material.SetSpecularBoost(ca.SpecularBoost)
	material.SetShininess(ca.Shininess)
	material.SetDiffuseMap(ca.DiffuseMap)
	material.SetNormalMap(ca.NormalMap)
	material.SetSpecularMap(ca.SpecularMap)
	material.SetEmissionMap(ca.EmissionMap)
	material.SetDetailMap(ca.DetailMap)
	material.SetBlendMap(ca.BlendMap)
	material.SetDiffuseMap2(ca.DiffuseMap2)
	material.SetNormalMap2(ca.NormalMap2)
	material.SetDetailScale(ca.DetailScale)
	material.SetDetailFactor(ca.DetailFactor)
	material.SetEnvMap(ca.EnvMap)
	material.SetEnvMapMask(ca.EnvMapMask)
	material.SetEnvMapTint(ca.EnvMapTint)
	material.SetColorTint(ca.ColorTint)

	r.container[key] = material
	return material
}
