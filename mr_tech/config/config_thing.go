package config

import (
	"github.com/markel1974/godoom/mr_tech/model/geometry"
)

// ThingType represents an enumeration for categorizing the type or kind of a Thing in the system.
type ThingType int

// ThingUnknownDef represents an unknown thing type.
// ThingPlayerDef represents a player thing type.
// ThingEnemyDef represents an enemy thing type.
// ThingWeaponDef represents a weapon thing type.
// ThingBulletDef represents a bullet thing type.
// ThingKeyDef represents a key thing type.
// ThingItemDef represents an item thing type.
const (
	ThingUnknownDef = ThingType(iota)
	ThingPlayerDef
	ThingEnemyDef
	ThingWeaponDef
	ThingBulletDef
	ThingKeyDef
	ThingItemDef
)

// Thing represents a game object with physical and visual properties for simulation and rendering.
type Thing struct {
	Id           string       `json:"id"`
	Position     geometry.XYZ `json:"position"`
	Angle        float64      `json:"angle"`
	Mass         float64      `json:"mass"`
	Restitution  float64      `json:"restitution"`
	Radius       float64      `json:"radius"`
	Height       float64      `json:"height"`
	Kind         ThingType    `json:"kind"`
	Speed        float64      `json:"speed"`
	Acceleration float64      `json:"acceleration"`
	Animation    *Animation   `json:"animation"`
	HasZPos      bool         `json:"hasZPos"`
}

// NewConfigThing2d creates a 2D Thing object with the specified properties such as position, angle, kind, and animation.
func NewConfigThing2d(id string, pos geometry.XY, angle float64, kind ThingType, mass, radius, height, speed float64, anim *Animation) *Thing {
	return &Thing{
		Id:          id,
		Position:    geometry.XYZ{X: pos.X, Y: pos.Y, Z: 0},
		Angle:       angle,
		Kind:        kind,
		Mass:        mass,
		Radius:      radius,
		Height:      height,
		Speed:       speed,
		Animation:   anim,
		Restitution: 0.0,
		HasZPos:     false,
	}
}

// NewConfigThing3d initializes and returns a pointer to a Thing with the provided parameters.
func NewConfigThing3d(id string, pos geometry.XYZ, angle float64, kind ThingType, mass, radius, height, speed float64, anim *Animation) *Thing {
	return &Thing{
		Id:          id,
		Position:    pos,
		Angle:       angle,
		Kind:        kind,
		Mass:        mass,
		Radius:      radius,
		Height:      height,
		Speed:       speed,
		Animation:   anim,
		Restitution: 0.0,
		HasZPos:     true,
	}
}
