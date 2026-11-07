package model

import (
	"fmt"
	"math"

	"github.com/markel1974/godoom/mr_tech/config"
	volumes2 "github.com/markel1974/godoom/mr_tech/model/volumes"
)

// Calibration contains parameters for configuring the camera and map projection in a 3D engine.
type Calibration struct {
	OrthoSize          float64
	MapCenterX         float64
	MapCenterZ         float64
	LightCamY          float64
	ZFarMax            float64
	ZNear              float64
	ZFar               float64
	ZLightFarFront     float64
	ZLightFarRear      float64
	AspectRatio        float64
	FovVerticalDegrees float64
	BeamRatio          float64
	VolSteps           float64
	AoFactor           float64
	EnableShadowLights bool
	EnableShadows      bool
	PrjScale           float64
	PrjScaleLights     float64
	volumes            *volumes2.Volumes
}

// NewCalibration creates and initializes a new Calibration instance using the provided configuration and volume data.
func NewCalibration(cfg *config.Calibration, volumes *volumes2.Volumes) *Calibration {
	c := &Calibration{
		OrthoSize:          cfg.OrthoSize,
		MapCenterX:         cfg.MapCenterX,
		MapCenterZ:         cfg.MapCenterZ,
		LightCamY:          cfg.LightCamY,
		ZFarMax:            cfg.ZFarMax,
		ZNear:              cfg.ZNear,
		ZFar:               cfg.ZFar,
		ZLightFarFront:     cfg.ZLightFarFront,
		ZLightFarRear:      cfg.ZLightFarRear,
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
	root, ok := c.volumes.GetRoot()
	if !ok {
		return fmt.Errorf("no root volume found")
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
		return fmt.Errorf("invalid orthographic size")
	}
	c.MapCenterX = root.GetMinX() + (width / 2.0)
	c.MapCenterZ = root.GetMinZ() + (depth / 2.0)
	c.LightCamY = root.GetMaxY()
	if c.ZNear <= 0 {
		c.ZNear = 0.1
	}
	if c.ZNear <= 0 {
		c.ZNear = 0.1
	}
	// Diagonale 3D della mappa: massima distanza euclidea teorica visibile
	maxFar := math.Sqrt(width*width + h*h + depth*depth)
	if c.ZFarMax > 0 {
		if maxFar > c.ZFarMax {
			maxFar = c.ZFarMax
		}
	}
	if math.IsNaN(maxFar) || math.IsInf(maxFar, 0) || maxFar <= c.ZNear {
		return fmt.Errorf("invalid far plane")
	}
	if c.ZFar <= 0 {
		c.ZFar = maxFar / 2
	}
	if c.ZFar <= 0 {
		c.ZFar = maxFar / 4
	}
	if c.ZLightFarFront <= 0 {
		c.ZLightFarFront = maxFar
	}
	if c.ZLightFarRear <= 0 {
		c.ZLightFarRear = maxFar / 4
	}
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

	// Wide-angle projection scale for lights frustum to cover 360 degrees
	const fovLights = 170.0
	fovLightsRad := (fovLights * math.Pi) / 180.0
	c.PrjScaleLights = 1.0 / math.Tan(fovLightsRad/2.0)

	return nil
}
