package model

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/config"
)

// flashFactorIncrement defines the incremental value used to adjust the flashlight's intensity factor.
const flashFactorIncrement = 0.001

// degreesToRadians converts an angle in degrees to its equivalent in radians.
func degreesToRadians(degrees float64) float64 {
	return (degrees * math.Pi) / 180.0
}

// Flash represents a structure used for managing field-of-view, shading, and cone configurations in a rendering system.
type Flash struct {
	shadowFovScale float64
	shadowFovDeg   float64
	shadowFovRad   float64
	zNear          float64
	zFar           float64
	factor         float64
	falloff        float64
	offsetX        float64
	offsetY        float64
	fovScale       float64
	fovDeg         float64
	fovRad         float64
	coneStart      float64
	coneEnd        float64
	aspect         float64
}

// NewFlash creates and initializes a new Flash object based on the provided configuration structure.
func NewFlash(c *config.Flash) *Flash {
	shadowFovDeg := c.FovDeg * c.ShadowFovFactor
	f := &Flash{
		fovDeg:       c.FovDeg,
		shadowFovDeg: shadowFovDeg,
		shadowFovRad: degreesToRadians(shadowFovDeg),
		fovRad:       degreesToRadians(c.FovDeg),
		zNear:        c.ZNear,
		zFar:         c.ZFar,
		factor:       c.Factor,
		falloff:      c.Falloff,
		offsetX:      c.OffsetX,
		offsetY:      c.OffsetY,
	}
	f.Rebuild(2.0)
	return f
}

// Rebuild recalculates the field of view, shadow field of view, and cone boundaries using the given ndcRange value.
func (p *Flash) Rebuild(ndcRange float64) {
	p.fovScale = 1.0 / math.Tan(p.fovRad/ndcRange)
	p.shadowFovScale = 1.0 / math.Tan(p.shadowFovRad/ndcRange)
	p.coneStart = math.Cos(p.fovDeg/ndcRange*math.Pi/180.0) + 0.01
	p.coneEnd = math.Cos(p.fovDeg / ndcRange * 0.6 * math.Pi / 180.0)
}

// GetFovScale returns the field of view scale of the Flash instance as a float64 value.
func (p *Flash) GetFovScale() float64 {
	return p.fovScale
}

// GetFovDeg returns the field of view in degrees associated with the Flash instance.
func (p *Flash) GetFovDeg() float64 {
	return p.fovDeg
}

// GetFovRad returns the field of view (FOV) in radians for the Flash instance.
func (p *Flash) GetFovRad() float64 {
	return p.fovRad
}

// GetShadowFovScale returns the shadow field of view scale as a float64 value.
func (p *Flash) GetShadowFovScale() float64 {
	return p.shadowFovScale
}

// GetShadowFovDeg returns the shadow field of view in degrees.
func (p *Flash) GetShadowFovDeg() float64 {
	return p.shadowFovDeg
}

// GetShadowFovRad returns the shadow field of view in radians for the Flash instance.
func (p *Flash) GetShadowFovRad() float64 {
	return p.shadowFovRad
}

// GetZNear returns the near plane distance of the flashlight projection.
func (p *Flash) GetZNear() float64 {
	return p.zNear
}

// GetZFar returns the far clipping plane distance for the flashlight projection.
func (p *Flash) GetZFar() float64 {
	return p.zFar
}

// GetFactor returns the current intensity factor of the Flash.
func (p *Flash) GetFactor() float64 {
	return p.factor
}

// GetFalloff returns the falloff value of the Flash object, which influences the intensity decay over distance.
func (p *Flash) GetFalloff() float64 {
	return p.falloff
}

// GetConeStart returns the starting point of the cone for the flash in float64.
func (p *Flash) GetConeStart() float64 {
	return p.coneStart
}

// GetConeEnd returns the end value of the cone, typically used for spotlight calculations or rendering parameters.
func (p *Flash) GetConeEnd() float64 {
	return p.coneEnd
}

// GetOffsetX returns the horizontal offset value for the Flash object.
func (p *Flash) GetOffsetX() float64 {
	return p.offsetX
}

// GetOffsetY returns the vertical offset value of the Flash instance.
func (p *Flash) GetOffsetY() float64 {
	return p.offsetY
}

// IncreaseFlashFactor increments the flash factor by a predefined constant value, enhancing the intensity of the flash effect.
func (p *Flash) IncreaseFlashFactor() {
	p.factor += flashFactorIncrement
}

// DecreaseFlashFactor reduces the flash factor by a fixed increment if the current factor is greater than zero.
func (p *Flash) DecreaseFlashFactor() {
	if p.factor > 0 {
		p.factor -= flashFactorIncrement
	}
}
