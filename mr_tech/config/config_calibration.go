package config

// Calibration represents the configuration for map and rendering settings, including orthographic size and camera parameters.
type Calibration struct {
	Full3d             bool    `json:"full3d"`
	OrthoSize          float64 `json:"orthoSize"`
	MapCenterX         float64 `json:"mapCenterX"`
	MapCenterZ         float64 `json:"mapCenterZ"`
	LightCamY          float64 `json:"lightCamY"`
	ZNearRoom          float64 `json:"zNearRoom"`
	ZFarRoom           float64 `json:"zFarRoom"`
	ScaleFactor        float64 `json:"scaleFactor"`
	FovVerticalDegrees float64 `json:"fovVerticalDegrees"`
	FlashFovDeg        float64 `json:"flashFovDeg"`
	ZNearFlash         float64 `json:"zNearFlash"`
	ZFarFlash          float64 `json:"zFarFlash"`
	ShininessWall      float64 `json:"shininessWall"`
	ShininessFloor     float64 `json:"shininessFloor"`
	SpecBoostWall      float64 `json:"specBoostWall"`
	SpecBoostFloor     float64 `json:"specBoostFloor"`
	BeamRatio          float64 `json:"beamRatio"`
	VolSteps           float64 `json:"volSteps"`
	FlashFactor        float64 `json:"flashFactor"`
	Auto               bool    `json:"auto"`
}

// NewConfigCalibration creates and returns a new Calibration instance with the specified parameters.
// orthoSize defines the orthographic size of the calibration.
// mapCenterX and mapCenterZ set the center coordinates of the map.
// lightCamY specifies the Y-coordinate for the light camera.
// zNearRoom and zFarRoom determine near and far plane distances for the room.
// auto defines whether to enable automatic calibration.
func NewConfigCalibration(full3d bool, orthoSize, mapCenterX, mapCenterZ, lightCamY, zNearRoom, zFarRoom float64, auto bool) *Calibration {
	c := &Calibration{
		Full3d:     full3d,
		OrthoSize:  orthoSize,
		MapCenterX: mapCenterX,
		MapCenterZ: mapCenterZ,
		LightCamY:  lightCamY,
		ZNearRoom:  zNearRoom,
		ZFarRoom:   zFarRoom,
		Auto:       auto,
	}
	c.ScaleFactor = 0.4
	c.FovVerticalDegrees = 90
	c.FlashFovDeg = 85.0
	c.ZNearFlash = 0.1
	c.ZFarFlash = 2048.0

	c.ShininessWall = 128.0
	c.ShininessFloor = 64.0
	c.SpecBoostWall = 0.05
	c.SpecBoostFloor = 0.1
	c.BeamRatio = 0.05
	c.VolSteps = 16
	c.FlashFactor = 80
	return c
}
