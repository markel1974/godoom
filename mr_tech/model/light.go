package model

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/model/config"
	"github.com/markel1974/godoom/mr_tech/model/geometry"
	"github.com/markel1974/godoom/mr_tech/physics"
)

const defaultFalloff = 10.0

// Light represents a light source with an intensity, falloff, type, and position in 3D space.
type Light struct {
	volume    *Volume
	intensity float64
	falloff   float64
	kind      config.LightKind
	pos       geometry.XYZ
	aabb      *physics.AABB
}

// NewLight creates and returns a new Light instance with default values for intensity, falloff, and kind.
func NewLight() *Light {
	l := &Light{
		volume:    nil,
		intensity: 0.0,
		falloff:   defaultFalloff,
		kind:      config.LightKindNone,
	}
	return l
}

// Setup configures the Light object by setting its intensity, falloff, kind, and position. Normalizes intensity between 0.0 and 1.0.
func (cl *Light) Setup(volume *Volume, intensity float64, falloff float64, kind config.LightKind, coords geometry.XYZ) {
	cl.volume = volume
	lightZ := coords.Z * 1.0

	//TODO TERMINARE CON TUTTI I TIPI DI LUCE
	if kind == config.LightKindOpenAir {
		lightZ = math.Abs(lightZ) * 1000 //50
	}

	pos := geometry.XYZ{X: coords.X, Y: coords.Y, Z: lightZ}
	cl.intensity = math.Max(0.0, math.Min(1.0, intensity))

	cl.falloff = falloff
	if cl.falloff <= 0 {
		cl.falloff = defaultFalloff
	}

	cl.kind = kind
	cl.pos = pos
	cl.Rebuild()
}

// Rebuild recalculates the AABB of the light based on its position and real falloff radius.
func (cl *Light) Rebuild() {
	// Usiamo il falloff reale per il Culling. L'AABB rappresenterà
	// esattamente il raggio di influenza massimo della luce nel mondo.
	const influence = 25.0
	radius := cl.falloff * influence
	cl.aabb = physics.NewAABB(
		cl.pos.X-radius, cl.pos.Y-radius, cl.pos.Z-radius,
		cl.pos.X+radius, cl.pos.Y+radius, cl.pos.Z+radius,
	)
}

// GetAABB retrieves the axis-aligned bounding box (AABB) associated with the Light object. Returns a pointer to AABB.
func (cl *Light) GetAABB() *physics.AABB {
	return cl.aabb
}

// GetKind retrieves the type of the light as a LightKind value.
func (cl *Light) GetKind() config.LightKind {
	return cl.kind
}

// GetVolume retrieves the volume associated with the Light instance. Returns a pointer to a Sector object.
func (cl *Light) GetVolume() *Volume {
	return cl.volume
}

// GetIntensity returns the current intensity of the light as a float64 value normalized between 0.0 and 1.0.
func (cl *Light) GetIntensity() float64 {
	return cl.intensity
}

// GetFalloff returns the attenuation distance (radius of influence) of the light.
func (cl *Light) GetFalloff() float64 {
	return cl.falloff
}

// GetPos returns the position of the Light as an XYZ struct.
func (cl *Light) GetPos() geometry.XYZ {
	return cl.pos
}
