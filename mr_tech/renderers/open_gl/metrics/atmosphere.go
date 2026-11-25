package metrics

import "github.com/markel1974/godoom/mr_tech/model"

// AtmosphereFog manages distance and height horizon fog uniforms for outdoor visibility and sky blending.
type AtmosphereFog struct {
	enabled       bool
	color         [3]float32
	sunColor      [3]float32
	start         float32
	end           float32
	autoStart     bool
	autoEnd       bool
	density       float32
	heightBase    float32
	heightFalloff float32
	sunScatter    float32
	zFar          float32
}

// NewAtmosphereFog initializes an AtmosphereFog state with daylight horizon haze defaults.
func NewAtmosphereFog() *AtmosphereFog {
	return &AtmosphereFog{
		color:      [3]float32{0.68, 0.78, 0.90},
		sunColor:   [3]float32{1.20, 1.02, 0.78},
		autoStart:  true,
		autoEnd:    true,
		density:    1.0,
		sunScatter: 0.45,
		zFar:       4096.0,
	}
}

// ConfigureFromCalibration applies fog parameters from model.Calibration.
func (f *AtmosphereFog) ConfigureFromCalibration(cal *model.Calibration) {
	if cal == nil {
		return
	}
	f.enabled = cal.EnableHorizonFog
	if cal.FogColor[0] > 0 || cal.FogColor[1] > 0 || cal.FogColor[2] > 0 {
		f.color = [3]float32{float32(cal.FogColor[0]), float32(cal.FogColor[1]), float32(cal.FogColor[2])}
	}
	if cal.FogSunColor[0] > 0 || cal.FogSunColor[1] > 0 || cal.FogSunColor[2] > 0 {
		f.sunColor = [3]float32{float32(cal.FogSunColor[0]), float32(cal.FogSunColor[1]), float32(cal.FogSunColor[2])}
	}
	f.autoStart = cal.FogStart <= 0
	f.autoEnd = cal.FogEnd <= 0
	if !f.autoStart {
		f.start = float32(cal.FogStart)
	}
	if !f.autoEnd {
		f.end = float32(cal.FogEnd)
	}
	if cal.FogDensity > 0 {
		f.density = float32(cal.FogDensity)
	} else {
		f.density = 1.0
	}
	f.heightBase = float32(cal.FogHeightBase)
	if cal.FogHeightFalloff > 0 {
		f.heightFalloff = float32(cal.FogHeightFalloff)
	} else {
		f.heightFalloff = 0
	}
	if cal.FogSunScatter < 0 {
		f.sunScatter = 0
	} else if cal.FogSunScatter > 1 {
		f.sunScatter = 1
	} else {
		f.sunScatter = float32(cal.FogSunScatter)
	}
	if cal.ZFar > 0 {
		f.UpdateZFar(float32(cal.ZFar))
	} else {
		f.UpdateZFar(4096.0)
	}
}

// SetEnabled toggles atmospheric horizon fog.
func (f *AtmosphereFog) SetEnabled(enabled bool) {
	f.enabled = enabled
}

// IsEnabled reports whether atmospheric horizon fog is active.
func (f *AtmosphereFog) IsEnabled() bool {
	return f.enabled
}

// SetColors updates linear RGB horizon fog color and forward sun in-scattering tint.
func (f *AtmosphereFog) SetColors(fogR, fogG, fogB, sunR, sunG, sunB float32) {
	f.color = [3]float32{fogR, fogG, fogB}
	f.sunColor = [3]float32{sunR, sunG, sunB}
}

// SetRange updates explicit distance bounds, density, and sun in-scattering strength.
// Passing start <= 0 or end <= 0 restores automatic derivation from zFar.
func (f *AtmosphereFog) SetRange(start, end, density, sunScatter float32) {
	f.autoStart = start <= 0
	f.autoEnd = end <= 0
	if !f.autoStart {
		f.start = start
	}
	if !f.autoEnd {
		f.end = end
	}
	if density > 0 {
		f.density = density
	}
	if sunScatter < 0 {
		sunScatter = 0
	} else if sunScatter > 1 {
		sunScatter = 1
	}
	f.sunScatter = sunScatter
	f.UpdateZFar(f.zFar)
}

// SetHeightParams configures base GL Y elevation and exponential vertical falloff.
func (f *AtmosphereFog) SetHeightParams(baseHeight, falloff float32) {
	f.heightBase = baseHeight
	if falloff < 0 {
		falloff = 0
	}
	f.heightFalloff = falloff
}

// UpdateZFar recalculates auto-derived fog distance bounds from the camera far plane.
func (f *AtmosphereFog) UpdateZFar(zFar float32) {
	if zFar <= 16.0 {
		zFar = 4096.0
	}
	f.zFar = zFar
	if f.autoEnd {
		f.end = zFar * 0.95
	}
	if f.autoStart {
		f.start = f.end * 0.25
	}
	if f.end <= f.start+1.0 {
		f.end = f.start + 1.0
	}
}

// GetShaderParams returns pre-packed fog colors and uniform parameters ready for GL upload.
func (f *AtmosphereFog) GetShaderParams() (enabled int32, color, sunColor [3]float32, fogParams [4]float32, heightParams [3]float32) {
	if f.enabled {
		enabled = 1
	}
	invRange := float32(1.0)
	if span := f.end - f.start; span > 1.0 {
		invRange = 1.0 / span
	}
	return enabled, f.color, f.sunColor, [4]float32{f.start, invRange, f.density, f.sunScatter}, [3]float32{f.heightBase, f.heightFalloff, f.zFar}
}
