package config

import (
	"github.com/markel1974/godoom/mr_tech/model/geometry"
)

type ThingType int

const (
	ThingUnknownDef = ThingType(iota)
	ThingPlayerDef
	ThingEnemyDef
	ThingWeaponDef
	ThingBulletDef
	ThingKeyDef
	ThingItemDef
)

// ConfigThing represents a game entity with physical properties, animation, and positional data within a specific sector.
type ConfigThing struct {
	Id        string           `json:"id"`
	Position  geometry.XYZ     `json:"position"`
	Angle     float64          `json:"angle"`
	Mass      float64          `json:"mass"`
	Radius    float64          `json:"radius"`
	Height    float64          `json:"height"`
	Kind      ThingType        `json:"kind"`
	Speed     float64          `json:"speed"`
	Animation *ConfigAnimation `json:"animation"`
}

// NewConfigThing creates and initializes a new ConfigThing with the given properties representing an object in the game world.
// id is the unique identifier for the thing.
// pos specifies the position of the thing in 2D space.
// angle represents the orientation of the thing in degrees.
// kind is an integer representing the type or category of the thing.
// sector assigns the thing to a specific sector in the level layout.
// mass defines the weight of the thing, used in physics calculations.
// radius and height define the dimensions of the thing for collision and spatial representation.
// anim is the animation configuration associated with the thing, such as frames, kind, and scale.
func NewConfigThing(id string, pos geometry.XYZ, angle float64, kind ThingType, mass, radius, height, speed float64, anim *ConfigAnimation) *ConfigThing {
	return &ConfigThing{
		Id:        id,
		Position:  pos,
		Angle:     angle,
		Kind:      kind,
		Mass:      mass,
		Radius:    radius,
		Height:    height,
		Speed:     speed,
		Animation: anim,
	}
}
