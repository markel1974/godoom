package model

// ConfigPlayer represents the configuration of a player in the game, including position, angle, and current sector.
type ConfigPlayer struct {
	Position XY      `json:"position"`
	Angle    float64 `json:"angle"`
	Sector   string  `json:"Sector"`
}

// NewConfigPlayer creates and returns a new instance of ConfigPlayer with the specified position, angle, and sector.
func NewConfigPlayer(position XY, angle float64, sector string) *ConfigPlayer {
	return &ConfigPlayer{
		Position: position,
		Angle:    angle,
		Sector:   sector,
	}
}
