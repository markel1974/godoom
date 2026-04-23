package model

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/model/geometry"
	"github.com/markel1974/godoom/mr_tech/physics"
	"github.com/markel1974/godoom/mr_tech/textures"
)

const defaultFalloff = 10.0

// Light represents a light source with an intensity, falloff, type, and position in 3D space.
type Light struct {
	volume      *Volume
	intensity   float64
	falloff     float64
	kind        config.LightKind
	pos         geometry.XYZ
	aabb        *physics.AABB
	r           float64
	g           float64
	b           float64
	dirX        float64
	dirY        float64
	dirZ        float64
	cutOff      float64
	outerCutOff float64
	style       []float64
}

// NewLight creates and returns a new Light instance with default values for intensity, falloff, and stage.
func NewLight() *Light {
	l := &Light{
		volume:    nil,
		intensity: 0.0,
		falloff:   defaultFalloff,
		kind:      config.LightKindNone,
		aabb:      physics.NewAABB(),
		r:         1.0,
		g:         1.0,
		b:         1.0,
	}
	return l
}

// Setup configures the Light object by setting its intensity, falloff, stage, and position. Normalizes intensity between 0.0 and 1.0.
func (cl *Light) Setup(volume *Volume, c *config.Light, coords geometry.XYZ) {
	cl.volume = volume
	cl.r = c.R
	cl.g = c.G
	cl.b = c.B
	cl.dirX = c.DirX
	cl.dirY = c.DirY
	cl.dirZ = c.DirZ
	lightZ := coords.Z * 1.0
	//TODO TERMINARE CON TUTTI I TIPI DI LUCE
	if c.Kind == config.LightKindOpenAir {
		lightZ = math.Abs(lightZ) * 1000 //50
	} else if c.Kind == config.LightKindSpot {
		//TODO DINAMIC
		cl.dirX, cl.dirY, cl.dirZ = 0.0, -1.0, 0.0
		cl.cutOff = math.Cos(35.0 * math.Pi / 180.0)
		cl.outerCutOff = math.Cos(40 * math.Pi / 180.0)
	}
	pos := geometry.XYZ{X: coords.X, Y: coords.Y, Z: lightZ}
	cl.intensity = c.Intensity //math.Max(0.0, math.Min(1.0, intensity))
	cl.falloff = c.Falloff
	if cl.falloff <= 0 {
		cl.falloff = defaultFalloff
	}
	cl.kind = c.Kind
	cl.style = c.Style
	if len(cl.style) == 0 {
		cl.style = []float64{1.0}
	}
	cl.pos = pos
	cl.Rebuild()
}

// Rebuild recalculates the AABB of the light based on its position and real falloff radius.
func (cl *Light) Rebuild() {
	// Usiamo il falloff reale per il Culling. L'AABB rappresenterà
	// esattamente il raggio di influenza massimo della luce nel mondo.
	const influence = 10
	r := cl.falloff * influence
	cl.aabb.Rebuild(cl.pos.X-r, cl.pos.Y-r, cl.pos.Z-r, cl.pos.X+r, cl.pos.Y+r, cl.pos.Z+r)
}

// GetAABB retrieves the axis-aligned bounding box (AABB) associated with the Light object. Returns a pointer to AABB.
func (cl *Light) GetAABB() *physics.AABB {
	return cl.aabb
}

// GetKind retrieves the type of the light as a LightKind value.
func (cl *Light) GetKind() config.LightKind {
	return cl.kind
}

// GetVolume retrieves the location associated with the Light instance. Returns a pointer to a Sector object.
func (cl *Light) GetVolume() *Volume {
	return cl.volume
}

// GetIntensity returns the current intensity of the light as a float64 value normalized between 0.0 and 1.0.
func (cl *Light) GetIntensity() float64 {
	return cl.intensity
}

// GetIntensityStyled calculates the styled intensity of the light at
func (cl *Light) GetIntensityStyled(tick uint64) float64 {
	const groupSize = 6.0
	frameFloat := textures.TickGrouped(tick, int(groupSize))
	idx := int(frameFloat) % len(cl.style)
	return cl.intensity * cl.style[idx]
}

// GetFalloff returns the attenuation distance (radius of influence) of the light.
func (cl *Light) GetFalloff() float64 {
	return cl.falloff
}

// GetPos retrieves the position of the light in 3D space as a geometry.XYZ value.
func (cl *Light) GetPos() geometry.XYZ {
	return cl.pos
}

// GetPosXYZ retrieves the X, Y, and Z coordinates of the light's position as separate float64 values.
func (cl *Light) GetPosXYZ() (float64, float64, float64) {
	return cl.pos.X, cl.pos.Y, cl.pos.Z
}

// GetRed returns the red color component of the light.
func (cl *Light) GetRed() float64 {
	return cl.r
}

// GetGreen returns the green color component of the light.
func (cl *Light) GetGreen() float64 {
	return cl.g
}

// GetBlue returns the blue color component of the light.
func (cl *Light) GetBlue() float64 {
	return cl.b
}

// GetDirX returns the X component of the light's direction vector.
func (cl *Light) GetDirX() float64 {
	return cl.dirX
}

// GetDirY returns the Y component of the light's direction vector.
func (cl *Light) GetDirY() float64 {
	return cl.dirY
}

// GetDirZ returns the Z component of the light's direction vector.
func (cl *Light) GetDirZ() float64 {
	return cl.dirZ
}

// GetCutOff returns the inner cutoff angle (in cosine) for spotlight calculations.
func (cl *Light) GetCutOff() float64 {
	return cl.cutOff
}

// GetOuterCutOff returns the outer cutoff angle (in cosine) for spotlight calculations.
func (cl *Light) GetOuterCutOff() float64 {
	return cl.outerCutOff
}
