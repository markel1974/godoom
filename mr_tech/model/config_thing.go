package model

// ConfigThing defines the properties of an in-game object, including its position, orientation, and physical attributes.
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

// NewConfigThing creates a new ConfigThing with the specified properties such as position, angle, kind, and animation.
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
