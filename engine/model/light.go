package model

import "math"

// Light represents a light source with an intensity, type, and position in 3D space.
type Light struct {
	intensity float64
	kind      LightKind
	pos       XYZ
}

// NewLight creates and returns a new Light instance with default values for intensity and kind.
func NewLight() *Light {
	return &Light{
		intensity: 0.0,
		kind:      LightKindNone,
	}
}

// Setup configures the Light object by setting its intensity, kind, and position. Normalizes intensity between 0.0 and 1.0.
func (cl *Light) Setup(intensity float64, kind LightKind, pos XYZ) {
	cl.intensity = math.Max(0.0, math.Min(1.0, intensity))
	cl.kind = kind
	cl.pos = pos
}

// GetIntensity returns the current intensity of the light as a float64 value normalized between 0.0 and 1.0.
func (cl *Light) GetIntensity() float64 {
	return cl.intensity
}

// GetPos returns the position of the Light as an XYZ struct.
func (cl *Light) GetPos() XYZ {
	return cl.pos
}
