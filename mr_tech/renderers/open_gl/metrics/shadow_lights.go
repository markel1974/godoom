package metrics

// ShadowLights is a structure for managing shadow mapping metrics and transformation matrices for multiple lights.
type ShadowLights struct {
	shadowLights        [8]*Light
	shadowLightMetrics  []*ShadowLight
	shadowLightMatrices []*float32
	shadowLightsNum     uint32
}

// NewShadowLights creates and initializes a new ShadowLights instance with default values.
func NewShadowLights() *ShadowLights {
	return &ShadowLights{}
}

// Update updates the shadow light metrics and transformation matrices based on the provided lights and their properties.
func (s *ShadowLights) Update(shadowLights [8]*Light, shadowLightsNum uint32) {
	s.shadowLightsNum = shadowLightsNum
	s.shadowLights = shadowLights
	if int(shadowLightsNum) >= len(s.shadowLightMatrices) {
		slLen := shadowLightsNum * 2
		s.shadowLightMatrices = make([]*float32, slLen)
		s.shadowLightMetrics = make([]*ShadowLight, slLen)
		for lx := uint32(0); lx < slLen; lx++ {
			s.shadowLightMetrics[lx] = NewShadowLight()
		}
	}

	for lx := uint32(0); lx < shadowLightsNum; lx++ {
		light := shadowLights[lx]
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
		shadowLightMetric := s.shadowLightMetrics[lx]
		shadowLightMetric.CreateSpace(pX, pY, pZ, dX, dY, dZ, fovDeg, near, effectiveRadius)
		s.shadowLightMatrices[lx] = shadowLightMetric.GetSpacePtr()
	}
}

// GetShadowLightMatrices returns the array of pointers to shadow light matrices and the count of active shadow lights.
func (s *ShadowLights) GetShadowLightMatrices() ([]*float32, uint32) {
	return s.shadowLightMatrices, s.shadowLightsNum
}

// GetLightCount returns the number of active shadow lights in the ShadowLights structure.
func (s *ShadowLights) GetLightCount() uint32 {
	return s.shadowLightsNum
}

// GetLight retrieves the light and its associated shadow light space pointer at the specified index.
func (s *ShadowLights) GetLight(idx int) (*Light, *float32) {
	return s.shadowLights[idx], s.shadowLightMetrics[idx].GetSpacePtr()
}

// GetSpacePtr retrieves a pointer to the light's shadow space matrix at the specified index. Returns nil if out of range.
func (s *ShadowLights) GetSpacePtr(idx int) *float32 {
	if uint32(idx) >= s.shadowLightsNum {
		return nil
	}
	return s.shadowLightMetrics[idx].GetSpacePtr()
}
