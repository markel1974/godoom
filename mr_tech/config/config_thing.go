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
}

// NewConfigThing creates Thing object with the specified properties such as position, angle, kind, and animation.
func NewConfigThing(id string, pos geometry.XYZ, angle float64, kind ThingType, mass, radius, height, speed float64, anim *Animation) *Thing {
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
	}
}

// Scale scales the position of the Thing by dividing each coordinate of its Position by the given scale factor.
func (t *Thing) Scale(scale float64) {
	t.Position.Scale(scale)
}
