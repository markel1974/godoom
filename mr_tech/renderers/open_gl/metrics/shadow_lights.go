package metrics

const ShadowLightsCount = 8

// ShadowLights is a structure for managing shadow mapping metrics and transformation matrices for multiple lights.
type ShadowLights struct {
	lights   [ShadowLightsCount]*Light
	metrics  [ShadowLightsCount]*ShadowLight
	matrices [ShadowLightsCount]*float32
	count    uint32
}

// NewShadowLights creates and initializes a new ShadowLights instance with default values.
func NewShadowLights() *ShadowLights {
	sl := &ShadowLights{}
	for lx := uint32(0); lx < ShadowLightsCount; lx++ {
		sl.lights[lx] = nil
		sl.matrices[lx] = nil
		sl.metrics[lx] = NewShadowLight()
	}
	return sl
}

// Update updates the shadow light metrics and transformation matrices based on the provided lights and their properties.
func (s *ShadowLights) Update(shadowLights [ShadowLightsCount]*Light, count uint32) {
	if count >= ShadowLightsCount {
		count = ShadowLightsCount - 1
	}
	s.count = count

	for lx := uint32(0); lx < count; lx++ {
		light := shadowLights[lx]
		s.lights[lx] = light
		pX, pY, pZ := light.X, light.Y, light.Z
		dX, dY, dZ := light.DirX, light.DirY, light.DirZ
		// Il FOV dell'ombra deve abbracciare interamente il CutOff del faretto
		// Se il faretto ha un outer cutoff di 40°, il FOV deve essere circa 80-90°
		//TODO FROM CONFIG
		fovDeg := float32(90.0)
		near := float32(0.1)
		factor := light.Intensity
		falloff := light.Falloff
		//TODO FROM CONFIG
		effectiveRadius := float32(4.605) * falloff * factor
		shadowLightMetric := s.metrics[lx]
		shadowLightMetric.CreateSpace(pX, pY, pZ, dX, dY, dZ, fovDeg, near, effectiveRadius)
		s.matrices[lx] = shadowLightMetric.GetSpacePtr()
	}
}

// GetLightCount returns the number of active shadow lights in the ShadowLights structure.
func (s *ShadowLights) GetLightCount() uint32 {
	return s.count
}

// GetLight retrieves the light and its associated shadow light space pointer at the specified index.
func (s *ShadowLights) GetLight(idx int) (*Light, *float32) {
	return s.lights[idx], s.metrics[idx].GetSpacePtr()
}
