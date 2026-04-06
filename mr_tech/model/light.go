package model

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/model/config"
	"github.com/markel1974/godoom/mr_tech/model/geometry"
	"github.com/markel1974/godoom/mr_tech/physics"
)

// Light represents a light source with an intensity, type, and position in 3D space.
type Light struct {
	volume    *Volume
	intensity float64
	kind      config.LightKind
	pos       geometry.XYZ
	aabb      *physics.AABB
}

// NewLight creates and returns a new Light instance with default values for intensity and kind.
func NewLight() *Light {
	l := &Light{
		volume:    nil,
		intensity: 0.0,
		kind:      config.LightKindNone,
	}
	return l
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
	cl.Rebuild()
}

// Rebuild recalculates the AABB of the light based on its position and a fixed light radius.
func (cl *Light) Rebuild() {
	// Stima del raggio di influenza massimo per generare l'AABB della luce.
	// NOTA: Se model.Light espone il falloff (es. l.GetFalloff()), usa quello invece del valore fisso.
	const lightRadius = 250.0
	// Creiamo un Bounding Box cubico che circoscrive la sfera di illuminazione della luce
	cl.aabb = physics.NewAABB(
		cl.pos.X-lightRadius, cl.pos.Y-lightRadius, cl.pos.Z-lightRadius,
		cl.pos.X+lightRadius, cl.pos.Y+lightRadius, cl.pos.Z+lightRadius,
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

// GetPos returns the position of the Light as an XYZ struct.
func (cl *Light) GetPos() geometry.XYZ {
	return cl.pos
}
