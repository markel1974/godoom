package config

import (
	"github.com/markel1974/godoom/mr_tech/model/geometry"
	"github.com/markel1974/godoom/mr_tech/utils"
)

// LightKind represents the type or classification of a light source, typically defined by an integer value.
type LightKind int

// LightKindNone represents an undefined or absent type of light.
// LightKindSpot represents a spotlight type of light.
// LightKindAmbient represents ambient light, providing overall illumination.
// LightKindOpenAir represents open-air light, simulating outdoor lighting conditions.
// LightKindDiffuse represents diffused light, typically scattered and soft.
// LightKindDirectional represents directional light, coming from a specific direction.
// LightKindParticle represents particle-based light, often used for effects.
const (
	LightKindNone LightKind = iota
	LightKindSpot
	LightKindAmbient
	LightKindOpenAir
	LightKindDiffuse
	LightKindDirectional
	LightKindParticle
)

// LightSector represents a lighting configuration within a sector, including properties like intensity and falloff.
type LightSector struct {
	Id        string    `json:"id"`
	Intensity float64   `json:"Intensity"`
	Kind      LightKind `json:"kind"`
	Falloff   float64   `json:"falloff"`
}

// NewConfigLightSector initializes and returns a new LightSector instance with specified intensity, kind, and falloff values.
func NewConfigLightSector(intensity float64, kind LightKind, falloff float64) *LightSector {
	return &LightSector{
		Id:        utils.NextUUId(),
		Intensity: intensity,
		Kind:      kind,
		Falloff:   falloff,
	}
}

// Light represents a light source with position, intensity, type, and falloff properties for rendering or simulation.
type Light struct {
	Id        string       `json:"id"`
	Pos       geometry.XYZ `json:"pos"`
	Intensity float64      `json:"Intensity"`
	Kind      LightKind    `json:"kind"`
	Falloff   float64      `json:"falloff"`
}

// NewConfigLight creates and returns a new Light configured with the specified position, intensity, type, and falloff values.
func NewConfigLight(pos geometry.XYZ, intensity float64, kind LightKind, falloff float64) *Light {
	return &Light{
		Id:        utils.NextUUId(),
		Pos:       pos,
		Intensity: intensity,
		Kind:      kind,
		Falloff:   falloff,
	}
}

// Scale adjusts the position of the light by scaling its X, Y, and Z coordinates with the provided scale factor.
func (t *Light) Scale(scale float64) {
	t.Pos.Scale(scale)
}
