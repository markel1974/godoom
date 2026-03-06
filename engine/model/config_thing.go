package model

// ConfigThing represents a non-player object (thing) in the game world.
type ConfigThing struct {
	Id       string  `json:"id"`
	Position XY      `json:"position"`
	Angle    float64 `json:"angle"`
	Type     int     `json:"type"`
	Sector   string  `json:"sector"`
}

// NewConfigThing creates and returns a new instance of ConfigThing.
func NewConfigThing(id string, position XY, angle float64, thingType int, sector string) *ConfigThing {
	return &ConfigThing{
		Id:       id,
		Position: position,
		Angle:    angle,
		Type:     thingType,
		Sector:   sector,
	}
}
