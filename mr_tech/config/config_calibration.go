package config

// Calibration represents the configuration for map and rendering settings, including orthographic size and camera parameters.
type Calibration struct {
	OrthoSize          float64    `json:"orthoSize"`
	MapCenterX         float64    `json:"mapCenterX"`
	MapCenterZ         float64    `json:"mapCenterZ"`
	LightCamY          float64    `json:"lightCamY"`
	ZFarMax            float64    `json:"zFarMax"`
	ZNear              float64    `json:"zNearFront"`
	ZFar               float64    `json:"zFarFront"`
	ZLightFar          float64    `json:"zLightFarFront"`
	AspectRatio        float64    `json:"aspectRatio"`
	FovVerticalDegrees float64    `json:"fovVerticalDegrees"`
	BeamRatio          float64    `json:"beamRatio"`
	VolSteps           float64    `json:"volSteps"`
	AoFactor           float64    `json:"aoFactor"`
	SolverIterations   int        `json:"solverIterations"`
	MaxOverlaps        int        `json:"maxOverlaps"`
	EnableSunCSM       bool       `json:"enableSunCSM,omitempty"`
	SunDirX            float64    `json:"sunDirX,omitempty"`
	SunDirY            float64    `json:"sunDirY,omitempty"`
	SunDirZ            float64    `json:"sunDirZ,omitempty"`
	SunColor           [3]float64 `json:"sunColor,omitempty"`
	SunIntensity       float64    `json:"sunIntensity,omitempty"`
	SunSkyColor        [3]float64 `json:"sunSkyColor,omitempty"`
	SunGroundColor     [3]float64 `json:"sunGroundColor,omitempty"`
	SunShadowDistance  float64    `json:"sunShadowDistance,omitempty"`
	SunCascadeLambda   float64    `json:"sunCascadeLambda,omitempty"`
	EnableHorizonFog   bool       `json:"enableHorizonFog,omitempty"`
	FogColor           [3]float64 `json:"fogColor,omitempty"`
	FogSunColor        [3]float64 `json:"fogSunColor,omitempty"`
	FogStart           float64    `json:"fogStart,omitempty"`
	FogEnd             float64    `json:"fogEnd,omitempty"`
	FogDensity         float64    `json:"fogDensity,omitempty"`
	FogHeightBase      float64    `json:"fogHeightBase,omitempty"`
	FogHeightFalloff   float64    `json:"fogHeightFalloff,omitempty"`
	FogSunScatter      float64    `json:"fogSunScatter,omitempty"`
}

// NewConfigCalibration initializes and returns a Calibration object with default configuration values.
func NewConfigCalibration() *Calibration {
	c := &Calibration{}
	c.AspectRatio = 0.4
	c.FovVerticalDegrees = 90
	c.BeamRatio = 0.05
	c.VolSteps = 8
	c.AoFactor = 0.8
	c.ZFarMax = 4096.0
	c.SolverIterations = 10
	c.MaxOverlaps = 512
	c.SunDirX = 0.35
	c.SunDirY = -0.45
	c.SunDirZ = -0.82
	c.SunColor = [3]float64{1.0, 0.96, 0.88}
	c.SunIntensity = 2.5
	c.SunSkyColor = [3]float64{0.42, 0.62, 0.92}
	c.SunGroundColor = [3]float64{0.22, 0.21, 0.19}
	c.SunCascadeLambda = 0.78
	c.FogColor = [3]float64{0.68, 0.78, 0.90}
	c.FogSunColor = [3]float64{1.20, 1.02, 0.78}
	c.FogDensity = 1.0
	c.FogSunScatter = 0.45
	return c
}
