package model

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/config"
)

// Calibration contains parameters for configuring the camera and map projection in a 3D engine.
type Calibration struct {
	OrthoSize          float64
	MapCenterX         float64
	MapCenterZ         float64
	LightCamY          float64
	ZNearRoom          float64
	ZFarRoom           float64
	AspectRatio        float64
	FovVerticalDegrees float64
	BeamRatio          float64
	VolSteps           float64
	AoFactor           float64
	EnableShadowLights bool
	EnableShadows      bool
	PrjScale           float64
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
		AspectRatio:        cfg.AspectRatio,
		FovVerticalDegrees: cfg.FovVerticalDegrees,
		BeamRatio:          cfg.BeamRatio,
		VolSteps:           cfg.VolSteps,
		AoFactor:           cfg.AoFactor,
		volumes:            volumes,
	}
	return c
}

// Setup initializes the calibration's parameters based on the dimensions and position of the root volume if Auto is enabled.
func (c *Calibration) Setup() error {
	//if !c.Auto {
	//	return nil
	//}
	root, ok := c.volumes.tree.GetRoot()
	if !ok {
		c.ZNearRoom = 0.1
		if c.ZFarRoom <= c.ZNearRoom || math.IsNaN(c.ZFarRoom) || math.IsInf(c.ZFarRoom, 0) {
			c.ZFarRoom = 8192.0
		}
		if c.PrjScale <= 0 || math.IsNaN(c.PrjScale) || math.IsInf(c.PrjScale, 0) {
			c.PrjScale = 0.85
		}
		return nil
	}
	width := root.GetWidth()
	h := root.GetHeight()
	depth := root.GetDepth()
	if width > depth {
		c.OrthoSize = width / 2.0
	} else {
		c.OrthoSize = depth / 2.0
	}
	if math.IsNaN(c.OrthoSize) || math.IsInf(c.OrthoSize, 0) || c.OrthoSize <= 0 {
		c.OrthoSize = 1024.0
	}
	c.MapCenterX = root.GetMinX() + (width / 2.0)
	c.MapCenterZ = root.GetMinZ() + (depth / 2.0)
	c.LightCamY = root.GetMaxY()
	c.ZNearRoom = 0.1
	// Diagonale 3D della mappa: massima distanza euclidea teorica visibile
	diag := math.Sqrt(width*width + h*h + depth*depth)
	if math.IsNaN(diag) || math.IsInf(diag, 0) || diag <= c.ZNearRoom {
		diag = 8192.0
	}
	c.ZFarRoom = diag

	fovY := c.FovVerticalDegrees
	if fovY <= 0 || fovY >= 180 || math.IsNaN(fovY) || math.IsInf(fovY, 0) {
		fovY = 90.0
	}
	fovYRad := (fovY * math.Pi) / 180.0
	focal := 1.0 / math.Tan(fovYRad/2.0)
	const guardBand = 0.85
	prjScale := focal * guardBand
	if math.IsNaN(prjScale) || math.IsInf(prjScale, 0) || prjScale <= 0 {
		prjScale = 1.0 * guardBand
	}
	c.PrjScale = prjScale

	return nil
}
