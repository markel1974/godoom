package model

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/model/config"
	"github.com/markel1974/godoom/mr_tech/model/geometry"
)

// Light represents a light source with an intensity, type, and position in 3D space.
type Light struct {
	volume    *Volume
	intensity float64
	kind      config.LightKind
	pos       geometry.XYZ
}

// NewLight creates and returns a new Light instance with default values for intensity and kind.
func NewLight() *Light {
	return &Light{
		volume:    nil,
		intensity: 0.0,
		kind:      config.LightKindNone,
	}
}

// Setup configures the Light object by setting its intensity, kind, and position. Normalizes intensity between 0.0 and 1.0.
func (cl *Light) Setup(volume *Volume, intensity float64, kind config.LightKind, coords geometry.XYZ) {
	cl.volume = volume
	lightZ := coords.Z * 1.0
	//TODO TERMINARE CON TUTTI I TIPI DI LUCE
	if kind == config.LightKindOpenAir {
		lightZ = math.Abs(lightZ) * 1000 //50
	} else {
		//TODO TEST
		//n := rand.Intn(2) + 1
		//kind = LightKind(n)
	}
	pos := geometry.XYZ{X: coords.X, Y: coords.Y, Z: lightZ}
	cl.intensity = math.Max(0.0, math.Min(1.0, intensity))
	cl.kind = kind
	cl.pos = pos
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

// GetPos returns the position of the Light as an XYZ struct.
func (cl *Light) GetPos() geometry.XYZ {
	return cl.pos
}
