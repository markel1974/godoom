package config

import (
	"github.com/markel1974/godoom/mr_tech/model/geometry"
	"github.com/markel1974/godoom/mr_tech/utils"
)

type LightKind int

const (
	LightKindNone LightKind = iota
	LightKindSpot
	LightKindAmbient
	LightKindOpenAir
	LightKindDiffuse
	LightKindDirectional
	LightKindParticle
)

// ConfigLight defines a light configuration with intensity and type attributes.
type ConfigLight struct {
	Id        string       `json:"id"`
	Pos       geometry.XYZ `json:"pos"`
	Intensity float64      `json:"Intensity"`
	Kind      LightKind    `json:"kind"`
}

// NewConfigLight creates a new instance of ConfigLight with default values for Intensity and Kind.
func NewConfigLight(pos geometry.XYZ, intensity float64, kind LightKind) *ConfigLight {
	return &ConfigLight{
		Id:        utils.NextUUId(),
		Pos:       pos,
		Intensity: intensity,
		Kind:      kind,
	}
}
