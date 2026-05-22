package config

import (
	"github.com/markel1974/godoom/mr_tech/model/geometry"
)

// ThingType defines an enumeration for the various types of objects or entities within the system.
type ThingType int

// ThingUnknownDef represents the default undefined type for a thing.
// ThingPlayerDef represents a player entity type.
// ThingEnemyDef represents an enemy entity type.
// ThingWeaponDef represents a weapon entity type.
// ThingBulletDef represents a bullet entity type.
// ThingThrowableDef represents a throwable object entity type.
// ThingKeyDef represents a key entity type.
// ThingItemDef represents a general item entity type.
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

// Thing represents a game entity with physical, visual, and behavior attributes in a simulation environment.
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
	GForce         float64      `json:"gForce"`
	MD1            *MD1         `json:"md1"`
	MultiSprite    *MultiSprite `json:"multiSprite"`
	Sprite         *Sprite      `json:"sprite"`

	OnThinking  ThinkingFunc
	OnCollision CollisionFunc
	OnImpact    ImpactFunc
}

// NewConfigThing creates and returns a new Thing instance with the specified ID, position, angle, type, and physical attributes.
func NewConfigThing(id string, pos geometry.XYZ, angle float64, kind ThingType, mass, radius, height, speed float64) *Thing {
	return &Thing{
		Id:             id,
		Position:       pos,
		Angle:          angle,
		Kind:           kind,
		Mass:           mass,
		Radius:         radius,
		Height:         height,
		Speed:          speed,
		Restitution:    0.0,
		WakeUpDistance: 100.0,
		JumpForce:      600,
		Friction:       0.2,
		GForce:         9.8,
	}
}

// Scale applies the given scaling factors to the Position of the Thing by modifying its X, Y, and Z coordinates.
func (t *Thing) Scale(scale geometry.XYZ) {
	t.Position.Scale(scale)
}

// Clone creates and returns a deep copy of the Thing instance, replicating all its fields and values.
func (t *Thing) Clone() *Thing {
	return &Thing{
		Id:             t.Id,
		Position:       t.Position,
		Kind:           t.Kind,
		Angle:          t.Angle,
		Mass:           t.Mass,
		Restitution:    t.Restitution,
		Friction:       t.Friction,
		Radius:         t.Radius,
		Height:         t.Height,
		Speed:          t.Speed,
		Acceleration:   t.Acceleration,
		JumpForce:      t.JumpForce,
		Pitch:          t.Pitch,
		WakeUpDistance: t.WakeUpDistance,
		GForce:         t.GForce,
		MD1:            t.MD1,
		MultiSprite:    t.MultiSprite,
		Sprite:         t.Sprite,
		OnThinking:     t.OnThinking,
		OnCollision:    t.OnCollision,
		OnImpact:       t.OnImpact,
	}
}
