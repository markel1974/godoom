package model

import "github.com/markel1974/godoom/mr_tech/config"

// Calibration contains parameters for configuring the camera and map projection in a 3D engine.
type Calibration struct {
	OrthoSize          float64
	MapCenterX         float64
	MapCenterZ         float64
	LightCamY          float64
	ZNearRoom          float64
	ZFarRoom           float64
	Auto               bool
	Full3d             bool
	ScaleFactor        float64
	FovVerticalDegrees float64
	ShininessWall      float64
	ShininessFloor     float64
	SpecBoostWall      float64
	SpecBoostFloor     float64
	BeamRatio          float64
	VolSteps           float64
	volumes            *Volumes
}

// NewCalibration creates and initializes a new Calibration instance using the provided configuration and volume data.
func NewCalibration(cfg *config.Calibration, volumes *Volumes) *Calibration {
	c := &Calibration{
		OrthoSize:          cfg.OrthoSize,
		MapCenterX:         cfg.MapCenterX,
		MapCenterZ:         cfg.MapCenterZ,
		LightCamY:          cfg.LightCamY,
		ZNearRoom:          cfg.ZNearRoom,
		ZFarRoom:           cfg.ZFarRoom,
		Auto:               cfg.Auto,
		Full3d:             cfg.Full3d,
		ScaleFactor:        cfg.ScaleFactor,
		FovVerticalDegrees: cfg.FovVerticalDegrees,
		ShininessWall:      cfg.ShininessWall,
		ShininessFloor:     cfg.ShininessFloor,
		SpecBoostWall:      cfg.SpecBoostWall,
		SpecBoostFloor:     cfg.SpecBoostFloor,
		BeamRatio:          cfg.BeamRatio,
		VolSteps:           cfg.VolSteps,
		volumes:            volumes,
	}
	c.init()
	return c
}

// init initializes the calibration's parameters based on the dimensions and position of the root volume if Auto is enabled.
func (c *Calibration) init() {
	if !c.Auto {
		return
	}
	root, ok := c.volumes.tree.GetRoot()
	if !ok {
		return
	}
	width := root.GetWidth()
	depth := root.GetDepth()
	// 2. OrthoSize è esattamente la metà dell'asse maggiore
	if width > depth {
		c.OrthoSize = width / 2.0
	} else {
		c.OrthoSize = depth / 2.0
	}
	c.MapCenterX = root.GetMinX() + (width / 2.0)
	c.MapCenterZ = root.GetMinZ() + (depth / 2.0)
	// La telecamera si posiziona appena sopra il punto più alto della mappa
	c.LightCamY = root.GetMaxY() //+ 2.0
	// Distanze di proiezione relative dalla telecamera
	c.ZNearRoom = 1.0
	c.ZFarRoom = root.GetMaxY() - root.GetMinY()
}
