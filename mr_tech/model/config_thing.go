package model

// ConfigThing represents a game entity with physical properties, animation, and positional data within a specific sector.
type ConfigThing struct {
	Id        string
	Position  XY
	Angle     float64
	Mass      float64
	Radius    float64
	Height    float64
	Kind      int
	Sector    string
	Animation *ConfigAnimation
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
func NewConfigThing(id string, pos XY, angle float64, kind int, sector string, mass, radius, height float64, anim *ConfigAnimation) *ConfigThing {
	return &ConfigThing{
		Id:        id,
		Position:  pos,
		Angle:     angle,
		Kind:      kind,
		Sector:    sector,
		Mass:      mass,
		Radius:    radius,
		Height:    height,
		Animation: anim,
	}
}
