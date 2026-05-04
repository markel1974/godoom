package config

import (
	"github.com/markel1974/godoom/mr_tech/model/geometry"
)

type Bobbing struct {
	SwayScale       float64 `json:"swayScale"`
	SwaySensitivity float64 `json:"swaySensitivity"`
	SwayMultiplierX float64 `json:"swayMultiplierX"`
	SwayMultiplierY float64 `json:"swayMultiplierY"`
	SwayOffsetX     float64 `json:"swayOffsetX"`
	SwayOffsetY     float64 `json:"swayOffsetY"`
	MaxAmplitudeX   float64 `json:"maxAmplitudeX"`
	MaxAmplitudeY   float64 `json:"maxAmplitudeY"`
	IdleDrift       float64 `json:"idleDrift"`
	StrideLength    float64 `json:"strideLength"`
	SpeedLerp       float64 `json:"speedLerp"`
	AmpLerp         float64 `json:"ampLerp"`
	ImpactMax       float64 `json:"impactMax"`
	ImpactScale     float64 `json:"impactScale"`
	IdleAmpX        float64 `json:"idleAmpX"`
	IdleAmpY        float64 `json:"idleAmpY"`
	SpringTension   float64 `json:"springTension"`
	SpringDamping   float64 `json:"springDamping"`
	TiltAmp         float64 `json:"tiltAmp"`
}

type Flash struct {
	FovDeg  float64 `json:"flashFovDeg"`
	ZNear   float64 `json:"flashZNear"`
	ZFar    float64 `json:"flashZFar"`
	Factor  float64 `json:"flashFactor"`
	Falloff float64 `json:"flashFalloff"`
	OffsetX float64 `json:"flashOffsetX"`
	OffsetY float64 `json:"flashOffsetY"`
}

// Player represents a specialized game entity with inherited attributes and behaviors from the Thing type.
type Player struct {
	*Thing
	Bobbing *Bobbing `json:"bobbing"`
	Flash   *Flash   `json:"flash"`
}

// NewConfigPlayer creates and returns a new Player instance configured with the given position, angle, height, radius, and mass.
func NewConfigPlayer(position geometry.XYZ, angle float64, mass, speed, radius, height float64) *Player {
	thing := NewConfigThing("PLAYER", position, angle, -1, mass, radius, height, speed)
	p := &Player{
		Thing:   thing,
		Bobbing: &Bobbing{},
		Flash:   &Flash{},
	}
	p.Flash.FovDeg = 80.0
	p.Flash.ZNear = 0.1
	p.Flash.ZFar = 2048.0
	p.Flash.Falloff = 200
	p.Flash.Factor = 0.4
	p.Flash.OffsetX = 0.2
	p.Flash.OffsetY = -0.4
	p.Bobbing.SwayOffsetX = 3.0
	p.Bobbing.SwayOffsetY = 0.2
	p.Bobbing.SwayScale = 2.0
	p.Bobbing.SwaySensitivity = 0.01
	p.Bobbing.SwayMultiplierX = 1.1
	p.Bobbing.SwayMultiplierY = 1.2
	p.Bobbing.MaxAmplitudeX = 0.05
	p.Bobbing.MaxAmplitudeY = 0.80
	p.Bobbing.StrideLength = 0.015
	p.Bobbing.IdleAmpX = 0.02
	p.Bobbing.IdleAmpY = 0.02
	p.Bobbing.IdleDrift = 0.03
	p.Bobbing.SpeedLerp = 0.15
	p.Bobbing.AmpLerp = 0.30
	p.Bobbing.ImpactMax = 20.0
	p.Bobbing.ImpactScale = 0.05
	p.Bobbing.SpringTension = 0.15
	p.Bobbing.SpringDamping = 0.75
	p.Bobbing.TiltAmp = 0.03

	return p
}
