package model

import (
	"fmt"
	"math"

	"github.com/markel1974/godoom/mr_tech/config"
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
	ZLightFar          float64
	AspectRatio        float64
	FovVerticalDegrees float64
	BeamRatio          float64
	VolSteps           float64
	AoFactor           float64
	EnableShadowLights bool
	EnableShadows      bool
	EnableSunCSM       bool
	SunDirX            float64
	SunDirY            float64
	SunDirZ            float64
	SunColor           [3]float64
	SunIntensity       float64
	SunSkyColor        [3]float64
	SunGroundColor     [3]float64
	SunShadowDistance  float64
	SunCascadeLambda   float64
	EnableHorizonFog   bool
	FogColor           [3]float64
	FogSunColor        [3]float64
	FogStart           float64
	FogEnd             float64
	FogDensity         float64
	FogHeightBase      float64
	FogHeightFalloff   float64
	FogSunScatter      float64
	MSAASamples        int
	RenderScale        float64
	EnableFXAA         bool
	PrjScale           float64
	PrjScaleLights     float64
	volumes            *Volumes
}

// NewCalibration creates and initializes a new Calibration instance using the provided configuration and volume data.
func NewCalibration(cfg *config.Calibration, volumes *Volumes) *Calibration {
	c := &Calibration{
		OrthoSize:          cfg.OrthoSize,
		MapCenterX:         cfg.MapCenterX,
		MapCenterZ:         cfg.MapCenterZ,
		LightCamY:          cfg.LightCamY,
		ZFarMax:            cfg.ZFarMax,
		ZNear:              cfg.ZNear,
		ZFar:               cfg.ZFar,
		ZLightFar:          cfg.ZLightFar,
		AspectRatio:        cfg.AspectRatio,
		FovVerticalDegrees: cfg.FovVerticalDegrees,
		BeamRatio:          cfg.BeamRatio,
		VolSteps:           cfg.VolSteps,
		AoFactor:           cfg.AoFactor,
		EnableSunCSM:       cfg.EnableSunCSM,
		SunDirX:            cfg.SunDirX,
		SunDirY:            cfg.SunDirY,
		SunDirZ:            cfg.SunDirZ,
		SunColor:           cfg.SunColor,
		SunIntensity:       cfg.SunIntensity,
		SunSkyColor:        cfg.SunSkyColor,
		SunGroundColor:     cfg.SunGroundColor,
		SunShadowDistance:  cfg.SunShadowDistance,
		SunCascadeLambda:   cfg.SunCascadeLambda,
		EnableHorizonFog:   cfg.EnableHorizonFog,
		FogColor:           cfg.FogColor,
		FogSunColor:        cfg.FogSunColor,
		FogStart:           cfg.FogStart,
		FogEnd:             cfg.FogEnd,
		FogDensity:         cfg.FogDensity,
		FogHeightBase:      cfg.FogHeightBase,
		FogHeightFalloff:   cfg.FogHeightFalloff,
		FogSunScatter:      cfg.FogSunScatter,
		MSAASamples:        DefaultMSAASamples,
		RenderScale:        1.0,
		volumes:            volumes,
	}
	return c
}

// DefaultMSAASamples is the default number of MSAA samples used by the 3D scene framebuffer.
const DefaultMSAASamples = 0

// IsValidMSAASamples reports whether n is a supported MSAA sample count (0 disables MSAA).
func IsValidMSAASamples(n int) bool {
	switch n {
	case 0, 2, 4, 8:
		return true
	}
	return false
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
		c.ZFar = maxFar * 0.5
	}
	if c.ZLightFar <= 0 {
		c.ZLightFar = maxFar
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
