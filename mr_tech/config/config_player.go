package config

import (
	"github.com/markel1974/godoom/mr_tech/model/geometry"
	"github.com/markel1974/godoom/mr_tech/utils"
)

type Bobbing struct {
	SwayScale     float64
	SwayOffsetX   float64
	SwayOffsetY   float64
	MaxAmplitudeX float64
	MaxAmplitudeY float64
	IdleDrift     float64
	StrideLength  float64
	SpeedLerp     float64
	AmpLerp       float64
	ImpactMax     float64
	ImpactScale   float64
	IdleAmp       float64
	SpringTension float64
	SpringDamping float64
}

// Player represents a specialized game entity with inherited attributes and behaviors from the Thing type.
type Player struct {
	*Thing
	Bobbing *Bobbing
}

// NewConfigPlayer creates and returns a new Player instance configured with the given position, angle, height, radius, and mass.
func NewConfigPlayer(position geometry.XYZ, angle float64, height float64, radius float64, mass float64) *Player {
	p := &Player{
		Thing: &Thing{
			Id:       utils.NextUUId(),
			Position: position,
			Angle:    angle,
			Height:   height,
			Radius:   radius,
			Mass:     mass,
		},
		Bobbing: &Bobbing{},
	}
	p.Bobbing.SwayScale = 4.0
	p.Bobbing.SwayOffsetX = 0.5
	p.Bobbing.SwayOffsetY = -0.1
	p.Bobbing.MaxAmplitudeX = 0.05
	p.Bobbing.MaxAmplitudeY = 0.80
	p.Bobbing.StrideLength = 0.015
	p.Bobbing.IdleAmp = 0.02
	p.Bobbing.IdleDrift = 0.03
	p.Bobbing.SpeedLerp = 0.15
	p.Bobbing.AmpLerp = 0.30

	p.Bobbing.ImpactMax = 20.0
	p.Bobbing.ImpactScale = 0.05
	p.Bobbing.SpringTension = 0.15
	p.Bobbing.SpringDamping = 0.75

	return p
}
