package model

// ConfigRoot represents the root configuration for a level, including sectors, lights, player, scale, and loop settings.
type ConfigRoot struct {
	Sectors     []*ConfigSector `json:"sectors"`
	Lights      []*ConfigLight  `json:"lights"`
	Player      *ConfigPlayer   `json:"player"`
	ScaleFactor float64         `json:"scaleFactor"`
	DisableLoop bool            `json:"disableLoop"`
}

// NewConfigRoot creates a new ConfigRoot instance with specified sectors, player, lights, scale factor, and loop status.
func NewConfigRoot(sectors []*ConfigSector, player *ConfigPlayer, lights []*ConfigLight, scaleFactor float64, disableLoop bool) *ConfigRoot {
	return &ConfigRoot{
		Sectors:     sectors,
		Player:      player,
		Lights:      lights,
		ScaleFactor: scaleFactor,
		DisableLoop: disableLoop,
	}
}
