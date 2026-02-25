package model

// ConfigRoot defines the structure used to represent game data including sectors, lights, player information, and configuration.
type ConfigRoot struct {
	Sectors     []*ConfigSector `json:"sectors"`
	Lights      []*ConfigLight  `json:"lights"`
	Player      *ConfigPlayer   `json:"player"`
	ScaleFactor float64         `json:"scaleFactor"`
	DisableLoop bool            `json:"disableLoop"`
}
