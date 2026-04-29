package config

import (
	"github.com/markel1974/godoom/mr_tech/model/geometry"
)

// ThingType defines an enumeration type representing different kinds of objects or entities in the system.
type ThingType int

// ThingUnknownDef represents an unknown or default type of Thing.
// ThingPlayerDef represents a player Thing type.
// ThingEnemyDef represents an enemy Thing type.
// ThingWeaponDef represents a weapon Thing type.
// ThingBulletDef represents a bullet Thing type.
// ThingKeyDef represents a key Thing type.
// ThingItemDef represents an item Thing type.
const (
	ThingUnknownDef = ThingType(iota)
	ThingPlayerDef
	ThingEnemyDef
	ThingWeaponDef
	ThingBulletDef
	ThingThrowableDef
	ThingKeyDef
	ThingItemDef
)

// Thing represents a physical or logical entity with position, dimensions, motion properties, and optional animations or models.
type Thing struct {
	Id             string       `json:"id"`
	Position       geometry.XYZ `json:"position"`
	Kind           ThingType    `json:"kind"`
	Angle          float64      `json:"angle"`
	Mass           float64      `json:"mass"`
	Restitution    float64      `json:"restitution"`
	Friction       float64      `json:"friction"`
	Radius         float64      `json:"radius"`
	Height         float64      `json:"height"`
	Speed          float64      `json:"speed"`
	Acceleration   float64      `json:"acceleration"`
	JumpForce      float64      `json:"jumpForce"`
	Pitch          float64      `json:"pitch"`
	WakeUpDistance float64      `json:"wakeUpDistance"`
	Md2            *MD2         `json:"md2"`
	Material       *Material    `json:"material"`
}

// NewConfigThing creates and returns a pointer to a new Thing instance initialized with the provided parameters.
func NewConfigThing(id string, pos geometry.XYZ, angle float64, kind ThingType, mass, radius, height, speed float64, material *Material) *Thing {
	return &Thing{
		Id:             id,
		Position:       pos,
		Angle:          angle,
		Kind:           kind,
		Mass:           mass,
		Radius:         radius,
		Height:         height,
		Speed:          speed,
		Material:       material,
		Restitution:    0.0,
		WakeUpDistance: 25.0,
		JumpForce:      200,
		Friction:       0.2,
	}
}

// Scale adjusts the position of the Thing by scaling its X, Y, and Z components using the specified scale factor.
func (t *Thing) Scale(scale float64) {
	t.Position.Scale(scale)
}

// SetModel3d assigns the provided 3D model to the Thing's Md2 field.
func (t *Thing) SetModel3d(model *MD2) {
	t.Md2 = model
}
