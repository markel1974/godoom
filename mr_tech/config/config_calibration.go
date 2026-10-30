package config

// Calibration represents the configuration for map and rendering settings, including orthographic size and camera parameters.
type Calibration struct {
	OrthoSize          float64 `json:"orthoSize"`
	MapCenterX         float64 `json:"mapCenterX"`
	MapCenterZ         float64 `json:"mapCenterZ"`
	LightCamY          float64 `json:"lightCamY"`
	ZNearRoom          float64 `json:"zNearRoom"`
	ZFarRoom           float64 `json:"zFarRoom"`
	AspectRatio        float64 `json:"aspectRatio"`
	FovVerticalDegrees float64 `json:"fovVerticalDegrees"`
	BeamRatio          float64 `json:"beamRatio"`
	VolSteps           float64 `json:"volSteps"`
	AoFactor           float64 `json:"aoFactor"`
	Auto               bool    `json:"auto"`
}

// NewConfigCalibration creates and returns a new Calibration instance with the specified parameters.
// orthoSize defines the orthographic size of the calibration.
// mapCenterX and mapCenterZ set the center coordinates of the map.
// lightCamY specifies the Y-coordinate for the light camera.
// zNearRoom and zFarRoom determine near and far plane distances for the room.
// auto defines whether to enable automatic calibration.
func NewConfigCalibration(orthoSize, mapCenterX, mapCenterZ, lightCamY, zNearRoom, zFarRoom float64, auto bool) *Calibration {
	c := &Calibration{
		OrthoSize:  orthoSize,
		MapCenterX: mapCenterX,
		MapCenterZ: mapCenterZ,
		LightCamY:  lightCamY,
		ZNearRoom:  zNearRoom,
		ZFarRoom:   zFarRoom,
		Auto:       auto,
	}
	c.AspectRatio = 0.4
	c.FovVerticalDegrees = 90
	c.BeamRatio = 0.05
	c.VolSteps = 8
	c.AoFactor = 0.8
	return c
}
