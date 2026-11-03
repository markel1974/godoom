package config

// Calibration represents the configuration for map and rendering settings, including orthographic size and camera parameters.
type Calibration struct {
	OrthoSize          float64 `json:"orthoSize"`
	MapCenterX         float64 `json:"mapCenterX"`
	MapCenterZ         float64 `json:"mapCenterZ"`
	LightCamY          float64 `json:"lightCamY"`
	ZNearFront         float64 `json:"zNearFront"`
	ZNearRear          float64 `json:"zNearRear"`
	ZFarFront          float64 `json:"zFarFront"`
	ZFarRear           float64 `json:"zFarRear"`
	AspectRatio        float64 `json:"aspectRatio"`
	FovVerticalDegrees float64 `json:"fovVerticalDegrees"`
	BeamRatio          float64 `json:"beamRatio"`
	VolSteps           float64 `json:"volSteps"`
	AoFactor           float64 `json:"aoFactor"`
}

// NewConfigCalibration initializes and returns a Calibration object with default configuration values.
func NewConfigCalibration() *Calibration {
	c := &Calibration{}
	c.AspectRatio = 0.4
	c.FovVerticalDegrees = 90
	c.BeamRatio = 0.05
	c.VolSteps = 8
	c.AoFactor = 0.8
	return c
}
