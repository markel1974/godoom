package textures

import (
	"fmt"
)

// Material represents a collection of 2D texture frames used for rendering animations, along with its shader.
type Material struct {
	blendMode   int
	name        string
	frame       *Texture
	frames      []*Texture
	totalFrames uint64
	kind        int
	u           float64
	v           float64

	clampAnim bool
	startTick uint64

	shader          string
	cullMode        int
	depthWrite      bool
	alphaTest       float32
	hasFog          bool
	fogColor        []float32
	polygonOffset   bool
	sort            int
	noMipmaps       bool
	noPicMip        bool
	entityMergeable bool
	deformVertexes  [][]string
	surfaceParms    map[string]bool
	skyParms        []string
	cloudParms      []string
	fogGen          []string
	isLightning     bool
	isSky           bool
	isBacksided     bool

	specularBoost float64
	shininess     float64

	// Texture Slots
	diffuseMap  string
	normalMap   string
	specularMap string
	emissionMap string
	detailMap   string
	blendMap    string

	// Multi-layer / Terrain Blend
	diffuseMap2 string
	normalMap2  string

	// Detail Properties
	detailScale  float32
	detailFactor float32

	// Environment / Cubemap Reflections & Tint
	envMap     string
	envMapMask string
	envMapTint []float32
	colorTint  []float32

	// _tickInterval defines the number of ticks required to form one logical interval for animation or updates.
	tickInterval uint64
}

// NewMaterial creates a new Material instance from a provided slice of Texture pointers.
// If the slice contains only one Texture, it is set as the current frame.
func NewMaterial(name string, frames []*Texture, kind int) *Material {
	a := &Material{
		name:         name,
		frames:       frames,
		frame:        nil,
		totalFrames:  uint64(len(frames)),
		kind:         kind,
		detailScale:  1.0,
		detailFactor: 1.0,
		// TODO FROM CONFIG defines the number of ticks required to form one logical interval for animation or updates.
		tickInterval: 32,
	}
	for _, frame := range frames {
		if frame == nil {
			a.totalFrames = 0
			a.frames = nil
			fmt.Println("nil frame")
			break
		}
	}
	if a.totalFrames == 1 {
		a.frame = frames[0]
	}
	return a
}

// GetName returns the name of the Material instance.
func (m *Material) GetName() string { return m.name }

// GetKind returns the type of the animation as an integer value.
func (m *Material) GetKind() int {
	return m.kind
}

// GetU returns the horizontal scroll offset of the material.
func (m *Material) GetU() float64 {
	return m.u
}

// GetV returns the vertical scroll offset of the material.
func (m *Material) GetV() float64 {
	return m.v
}

// GetShader returns the shader associated with the Material instance as a string.
func (m *Material) GetShader() string { return m.shader }

// GetCullMode returns the culling mode (e.g. front, back, none).
func (m *Material) GetCullMode() int { return m.cullMode }

// GetDepthWrite returns whether the material writes to the depth buffer.
func (m *Material) GetDepthWrite() bool { return m.depthWrite }

// GetAlphaTest returns the alpha threshold for discarding pixels.
func (m *Material) GetAlphaTest() float32 { return m.alphaTest }

// HasFog returns whether this is a fog material.
func (m *Material) HasFog() bool { return m.hasFog }

// GetFogColor returns the RGB components of the fog color.
func (m *Material) GetFogColor() []float32 { return m.fogColor }

// GetPolygonOffset returns whether this material needs depth offset (e.g. decals).
func (m *Material) GetPolygonOffset() bool { return m.polygonOffset }

// GetSort returns the sorting order of the material.
func (m *Material) GetSort() int { return m.sort }

// IsEntityMergeable returns whether this material's entity can be merged.
func (m *Material) IsEntityMergeable() bool { return m.entityMergeable }

// GetDeformVertexes returns the vertex deformation stages.
func (m *Material) GetDeformVertexes() [][]string { return m.deformVertexes }

// GetSurfaceParms returns the surface parameters dictionary.
func (m *Material) GetSurfaceParms() map[string]bool { return m.surfaceParms }

// GetSkyParms returns the sky parameters array.
func (m *Material) GetSkyParms() []string { return m.skyParms }

// GetCloudParms returns the cloud parameters array.
func (m *Material) GetCloudParms() []string { return m.cloudParms }

// GetFogGen returns the fog generation parameters array.
func (m *Material) GetFogGen() []string { return m.fogGen }

// GetSpecularBoost retrieves the specular boost value, which intensifies the specular highlights of the material during rendering.
func (m *Material) GetSpecularBoost() float64 { return m.specularBoost }

// GetShininess returns the shininess property of the material, which determines the specular highlight intensity.
func (m *Material) GetShininess() float64 { return m.shininess }

// GetDiffuseMap returns the diffuse map path/name.
func (m *Material) GetDiffuseMap() string { return m.diffuseMap }

// GetNormalMap returns the normal map path/name.
func (m *Material) GetNormalMap() string { return m.normalMap }

// GetSpecularMap returns the specular map path/name.
func (m *Material) GetSpecularMap() string { return m.specularMap }

// GetEmissionMap returns the emission map path/name.
func (m *Material) GetEmissionMap() string { return m.emissionMap }

// GetDetailMap returns the detail map path/name.
func (m *Material) GetDetailMap() string { return m.detailMap }

// GetBlendMap returns the blend map path/name.
func (m *Material) GetBlendMap() string { return m.blendMap }

// GetDiffuseMap2 returns the secondary diffuse map for terrain blending.
func (m *Material) GetDiffuseMap2() string { return m.diffuseMap2 }

// GetNormalMap2 returns the secondary normal map for terrain blending.
func (m *Material) GetNormalMap2() string { return m.normalMap2 }

// GetDetailScale returns the tiling scale of the detail texture.
func (m *Material) GetDetailScale() float32 { return m.detailScale }

// GetDetailFactor returns the blend factor of the detail texture.
func (m *Material) GetDetailFactor() float32 { return m.detailFactor }

// GetEnvMap returns the cubemap environment reflection identifier.
func (m *Material) GetEnvMap() string { return m.envMap }

// GetEnvMapMask returns the environment reflection mask path/name.
func (m *Material) GetEnvMapMask() string { return m.envMapMask }

// GetEnvMapTint returns the environment reflection color tint.
func (m *Material) GetEnvMapTint() []float32 { return m.envMapTint }

// GetColorTint returns the stage/material color tint.
func (m *Material) GetColorTint() []float32 { return m.colorTint }

// GetBlendMode returns the current blend mode identifier as an integer.
func (m *Material) GetBlendMode() int {
	return m.blendMode
}

// IsLightning returns whether this is a lightning material.
func (m *Material) IsLightning() bool { return m.isLightning }

// IsSky returns whether this material represents the sky.
func (m *Material) IsSky() bool { return m.isSky }

// IsBacksided returns whether this material is backsided.
func (m *Material) IsBacksided() bool { return m.isBacksided }

// RestartAnim resets the animation by setting the start tick to the current global tick.
func (m *Material) RestartAnim() {
	m.startTick = GlobalTick()
}

// GetHeadFrame returns the first frame of the material's texture sequence, or nil if no frames are available.
func (m *Material) GetHeadFrame() *Texture {
	if len(m.frames) == 0 {
		return nil
	}
	return m.frames[0]
}

// GetCurrentFrame returns the currently active frame of the animation based on global tick and tick interval.
func (m *Material) GetCurrentFrame() *Texture {
	if m.totalFrames > 1 {
		if m.clampAnim {
			return m.GetClampedFrame()
		}
		currentTick := GlobalTick() / m.tickInterval
		return m.frames[currentTick%m.totalFrames]
	}
	return m.frame
}

// GetClampedFrame calculates and returns the appropriate texture frame based on clamped animation logic and elapsed time.
func (m *Material) GetClampedFrame() *Texture {
	elapsed := uint64(0)
	if _globalTick > m.startTick {
		elapsed = _globalTick - m.startTick
	}
	elapsedTick := elapsed / m.tickInterval
	if elapsedTick >= m.totalFrames-1 {
		elapsedTick = m.totalFrames - 1
	}
	return m.frames[elapsedTick]
}

// SetClampAnim sets whether the animation should clamp to the last frame when its duration exceeds the total frame count.
func (m *Material) SetClampAnim(clamp bool) {
	m.clampAnim = clamp
}

// SetBlendMode updates the blend mode of the material, determining how it combines with other textures during rendering.
func (m *Material) SetBlendMode(b int) {
	m.blendMode = b
}

// SetUV sets the horizontal scroll offset of the material.
func (m *Material) SetUV(u float64, v float64) {
	m.u = u
	m.v = v
}

// SetShader sets the name of the shader associated with this material.
func (m *Material) SetShader(shader string) { m.shader = shader }

// SetStartTick sets the tick at which the clamped animation starts.
func (m *Material) SetStartTick(tick uint64) { m.startTick = tick }

// SetTickInterval sets the number of ticks required to form one logical interval for animation or updates.
// A zero value is ignored to avoid division by zero.
func (m *Material) SetTickInterval(interval uint64) {
	if interval == 0 {
		return
	}
	m.tickInterval = interval
}

// SetCullMode sets the culling mode (e.g. front, back, none).
func (m *Material) SetCullMode(cullMode int) { m.cullMode = cullMode }

// SetDepthWrite sets whether the material writes to the depth buffer.
func (m *Material) SetDepthWrite(depthWrite bool) { m.depthWrite = depthWrite }

// SetAlphaTest sets the alpha threshold for discarding pixels.
func (m *Material) SetAlphaTest(alphaTest float32) { m.alphaTest = alphaTest }

// SetFog sets whether this is a fog material.
func (m *Material) SetFog(hasFog bool) { m.hasFog = hasFog }

// SetFogColor sets the RGB components of the fog color.
func (m *Material) SetFogColor(fogColor []float32) { m.fogColor = fogColor }

// SetPolygonOffset sets whether this material needs depth offset (e.g. decals).
func (m *Material) SetPolygonOffset(polygonOffset bool) { m.polygonOffset = polygonOffset }

// SetSort sets the sorting order of the material.
func (m *Material) SetSort(sort int) { m.sort = sort }

// SetNoMipmaps sets whether mipmaps are disabled for this material.
func (m *Material) SetNoMipmaps(noMipmaps bool) { m.noMipmaps = noMipmaps }

// SetNoPicMip sets whether picmip reduction is disabled for this material.
func (m *Material) SetNoPicMip(noPicMip bool) { m.noPicMip = noPicMip }

// SetEntityMergeable sets whether this material's entity can be merged.
func (m *Material) SetEntityMergeable(entityMergeable bool) { m.entityMergeable = entityMergeable }

// SetDeformVertexes sets the vertex deformation stages.
func (m *Material) SetDeformVertexes(deformVertexes [][]string) { m.deformVertexes = deformVertexes }

// SetSurfaceParms sets the surface parameters dictionary.
func (m *Material) SetSurfaceParms(surfaceParms map[string]bool) { m.surfaceParms = surfaceParms }

// SetSkyParms sets the sky parameters array.
func (m *Material) SetSkyParms(skyParms []string) { m.skyParms = skyParms }

// SetCloudParms sets the cloud parameters array.
func (m *Material) SetCloudParms(cloudParms []string) { m.cloudParms = cloudParms }

// SetFogGen sets the fog generation parameters array.
func (m *Material) SetFogGen(fogGen []string) { m.fogGen = fogGen }

// SetLightning sets whether this is a lightning material.
func (m *Material) SetLightning(isLightning bool) { m.isLightning = isLightning }

// SetSky sets whether this material represents the sky.
func (m *Material) SetSky(isSky bool) { m.isSky = isSky }

// SetBacksided sets whether this material is backsided.
func (m *Material) SetBacksided(isBacksided bool) { m.isBacksided = isBacksided }

// SetSpecularBoost sets the specular boost value.
func (m *Material) SetSpecularBoost(specularBoost float64) { m.specularBoost = specularBoost }

// SetShininess sets the shininess property of the material.
func (m *Material) SetShininess(shininess float64) { m.shininess = shininess }

// SetDiffuseMap sets the diffuse map path/name.
func (m *Material) SetDiffuseMap(diffuseMap string) { m.diffuseMap = diffuseMap }

// SetNormalMap sets the normal map path/name.
func (m *Material) SetNormalMap(normalMap string) { m.normalMap = normalMap }

// SetSpecularMap sets the specular map path/name.
func (m *Material) SetSpecularMap(specularMap string) { m.specularMap = specularMap }

// SetEmissionMap sets the emission map path/name.
func (m *Material) SetEmissionMap(emissionMap string) { m.emissionMap = emissionMap }

// SetDetailMap sets the detail map path/name.
func (m *Material) SetDetailMap(detailMap string) { m.detailMap = detailMap }

// SetBlendMap sets the blend map path/name.
func (m *Material) SetBlendMap(blendMap string) { m.blendMap = blendMap }

// SetDiffuseMap2 sets the secondary diffuse map for terrain blending.
func (m *Material) SetDiffuseMap2(diffuseMap2 string) { m.diffuseMap2 = diffuseMap2 }

// SetNormalMap2 sets the secondary normal map for terrain blending.
func (m *Material) SetNormalMap2(normalMap2 string) { m.normalMap2 = normalMap2 }

// SetDetailScale sets the tiling scale of the detail texture.
func (m *Material) SetDetailScale(detailScale float32) { m.detailScale = detailScale }

// SetDetailFactor sets the blend factor of the detail texture.
func (m *Material) SetDetailFactor(detailFactor float32) { m.detailFactor = detailFactor }

// SetEnvMap sets the cubemap environment reflection identifier.
func (m *Material) SetEnvMap(envMap string) { m.envMap = envMap }

// SetEnvMapMask sets the environment reflection mask path/name.
func (m *Material) SetEnvMapMask(envMapMask string) { m.envMapMask = envMapMask }

// SetEnvMapTint sets the environment reflection color tint.
func (m *Material) SetEnvMapTint(envMapTint []float32) { m.envMapTint = envMapTint }

// SetColorTint sets the stage/material color tint.
func (m *Material) SetColorTint(colorTint []float32) { m.colorTint = colorTint }
