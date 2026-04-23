package model

import "github.com/markel1974/godoom/mr_tech/config"

// Flash represents a lighting system with configurable parameters including field of view, range, intensity, and offsets.
type Flash struct {
	fovDeg  float64
	zNear   float64
	zFar    float64
	factor  float64
	falloff float64
	offsetX float64
	offsetY float64
}

// NewFlash initializes a new Flash instance using the provided configuration values from the Flash configuration struct.
func NewFlash(c *config.Flash) *Flash {
	return &Flash{
		fovDeg:  c.FovDeg,
		zNear:   c.ZNear,
		zFar:    c.ZFar,
		factor:  c.Factor,
		falloff: c.Falloff,
		offsetX: c.OffsetX,
		offsetY: c.OffsetY,
	}
}

// GetFovDeg returns the field of view angle (in degrees) for the Flash instance.
func (p *Flash) GetFovDeg() float64 {
	return p.fovDeg
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
