package model

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/config"
)

// Flash represents a lighting system with configurable parameters including field of view, range, intensity, and offsets.
type Flash struct {
	fovDeg    float64
	zNear     float64
	zFar      float64
	factor    float64
	falloff   float64
	offsetX   float64
	offsetY   float64
	fov       float64
	fovRad    float64
	coneStart float64
	coneEnd   float64
	aspect    float64
}

// NewFlash initializes a new Flash instance using the provided configuration values from the Flash configuration struct.
func NewFlash(c *config.Flash) *Flash {
	f := &Flash{
		fovDeg:  c.FovDeg,
		fovRad:  (c.FovDeg * math.Pi) / 180.0,
		zNear:   c.ZNear,
		zFar:    c.ZFar,
		factor:  c.Factor,
		falloff: c.Falloff,
		offsetX: c.OffsetX,
		offsetY: c.OffsetY,
	}
	f.Rebuild(2.0)
	return f
}

func (p *Flash) Rebuild(ndcRange float64) {
	p.fov = 1.0 / math.Tan(p.fovRad/ndcRange)
	p.coneStart = math.Cos(p.fovDeg/ndcRange*math.Pi/180.0) + 0.01
	p.coneEnd = math.Cos(p.fovDeg / ndcRange * 0.6 * math.Pi / 180.0)
}

// GetFov returns the field of view value (in radians) for the Flash instance.
func (p *Flash) GetFov() float64 {
	return p.fov
}

// GetFovDeg retrieves the field of view value of the Flash object in degrees.
func (p *Flash) GetFovDeg() float64 {
	return p.fovDeg
}

// GetFovRad returns the field of view in radians for the Flash instance.
func (p *Flash) GetFovRad() float64 {
	return p.fovRad
}

// GetZNear returns the near clipping plane distance for the flash.
func (p *Flash) GetZNear() float64 {
	return p.zNear
}

// GetZFar returns the far clipping distance of the Flash object.
func (p *Flash) GetZFar() float64 {
	return p.zFar
}

// GetFactor retrieves the flashFactor value, which represents the scaling factor associated with the Flash instance.
func (p *Flash) GetFactor() float64 {
	return p.factor
}

// GetFalloff returns the falloff value of the Flash instance.
func (p *Flash) GetFalloff() float64 {
	return p.falloff
}

// GetConeStart retrieves the starting value of the cone region for the Flash instance.
func (p *Flash) GetConeStart() float64 {
	return p.coneStart
}

// GetConeEnd retrieves the end value of the cone for the Flash instance, defining the outer boundary of the light cone.
func (p *Flash) GetConeEnd() float64 {
	return p.coneEnd
}

// GetOffsetX returns the horizontal offset value of the Flash object.
func (p *Flash) GetOffsetX() float64 {
	return p.offsetX
}

// GetOffsetY returns the vertical offset (Y-axis) value of the Flash object.
func (p *Flash) GetOffsetY() float64 {
	return p.offsetY
}

// IncreaseFlashFactor increments the flashlight's intensity factor by increasing the `factor` field by 1.
func (p *Flash) IncreaseFlashFactor() {
	p.factor++
}

// DecreaseFlashFactor reduces the flashlight's intensity factor by 1, ensuring it does not drop below 0.
func (p *Flash) DecreaseFlashFactor() {
	if p.factor > 0 {
		p.factor--
	}
}
