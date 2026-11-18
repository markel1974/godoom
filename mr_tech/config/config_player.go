package config

import (
	"github.com/markel1974/godoom/mr_tech/geometry"
)

// Player represents a specialized game entity with inherited attributes and behaviors from the Thing type.
type Player struct {
	*Thing
	Bobbing         *Bobbing `json:"bobbing"`
	Flash           *Flash   `json:"flash"`
	DefaultWeaponId string   `json:"defaultWeaponId"`
	Impulse         float64  `json:"impulse"`
	IsVisible       bool     `json:"isVisible"`
}

// NewConfigPlayer creates and returns a new Player instance configured with the given position, angle, height, radius, and mass.
func NewConfigPlayer(position geometry.XYZ, angle float64, mass, speed, radius, height float64) *Player {
	thing := NewConfigThing("PLAYER", position, angle, ThingPlayerDef, mass, radius, height, speed)
	p := &Player{
		Thing:     thing,
		Bobbing:   NewBobbing(),
		Flash:     NewFlash(),
		IsVisible: false,
	}
	p.DefaultWeaponId = weaponTest
	p.Impulse = 0.06
	p.Flash.FovDeg = 80.0
	p.Flash.ShadowFovFactor = 1.75
	p.Flash.ZNear = 0.1
	p.Flash.ZFar = 2048.0
	p.Flash.Falloff = 200
	p.Flash.Intensity = 0.4
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
	p.Bobbing.StrideLength = 0.09 //0.015
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
