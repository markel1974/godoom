package metrics

import "math"

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
	s.count = count
	if s.count == 0 {
		return
	}
	if s.count > ShadowLightsCount {
		s.count = ShadowLightsCount
	}
	for lx := uint32(0); lx < s.count; lx++ {
		light := shadowLights[lx]
		s.lights[lx] = light
		pX, pY, pZ := light.X, light.Y, light.Z
		dX, dY, dZ := light.DirX, light.DirY, light.DirZ
		// Il FOV dell'ombra deve abbracciare interamente il CutOff del faretto.
		// OuterCutOff e' cos(halfAngle): allarga il FOV solo oltre 45 gradi (cos < 0.7071).
		fovDeg := float32(90.0)
		if light.OuterCutOff < 0.70710678 {
			cosVal := float64(light.OuterCutOff)
			if cosVal < -0.9659258 {
				cosVal = -0.9659258
			}
			coneFov := float32(2.0*math.Acos(cosVal)*180.0/math.Pi) + 12.0
			if coneFov > 165.0 {
				coneFov = 165.0
			}
			if coneFov > fovDeg {
				fovDeg = coneFov
			}
		}
		near := float32(0.1)
		factor := light.Intensity
		if factor < 0.1 {
			factor = 0.1
		}
		falloff := light.Falloff
		if falloff < 1.0 {
			falloff = 1.0
		}
		effectiveRadius := float32(4.605) * falloff * factor
		if effectiveRadius < near+1.0 {
			effectiveRadius = near + 1.0
		}
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
