package config

import (
	"github.com/markel1974/godoom/mr_tech/model/geometry"
	"github.com/markel1974/godoom/mr_tech/utils"
)

// LightKind represents the type of light source, defined as an integer-based enumeration.
type LightKind int

// LightKindNone represents the absence of a light source.
// LightKindSpot represents a focused spotlight effect.
// LightKindAmbient represents ambient light with no specific source or direction.
// LightKindOpenAir represents light evenly distributed in an outdoor environment.
// LightKindDiffuse represents light scattered and softened from a source.
// LightKindDirectional represents light coming from a specific direction, like sunlight.
// LightKindParticle represents light emitted from particles or small sources.
const (
	LightKindNone LightKind = iota
	LightKindSpot
	LightKindAmbient
	LightKindOpenAir
	LightKindDiffuse
	LightKindDirectional
	LightKindParticle
)

// ConfigLightSector represents the configuration of a light sector, including its unique ID, intensity, and type.
type ConfigLightSector struct {
	Id        string    `json:"id"`
	Intensity float64   `json:"Intensity"`
	Kind      LightKind `json:"kind"`
	Falloff   float64   `json:"falloff"`
}

// NewConfigLightSector creates a new ConfigLightSector instance with a unique ID, specified intensity, and LightKind.
func NewConfigLightSector(intensity float64, kind LightKind, falloff float64) *ConfigLightSector {
	return &ConfigLightSector{
		Id:        utils.NextUUId(),
		Intensity: intensity,
		Kind:      kind,
		Falloff:   falloff,
	}
}

// ConfigLightPos represents the configuration for a light source's position, intensity, and kind.
type ConfigLightPos struct {
	Id        string       `json:"id"`
	Pos       geometry.XYZ `json:"pos"`
	Intensity float64      `json:"Intensity"`
	Kind      LightKind    `json:"kind"`
	Falloff   float64      `json:"falloff"`
}

// NewConfigLightPos creates and returns a new ConfigLightPos with the specified position, intensity, and light kind.
func NewConfigLightPos(pos geometry.XYZ, intensity float64, kind LightKind, falloff float64) *ConfigLightPos {
	return &ConfigLightPos{
		Id:        utils.NextUUId(),
		Pos:       pos,
		Intensity: intensity,
		Kind:      kind,
		Falloff:   falloff,
	}
}
