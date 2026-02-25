package model

// InputConfig defines the structure used to represent game data including sectors, lights, player information, and configuration.
type InputConfig struct {
	Sectors     []*InputSector `json:"sectors"`
	Lights      []*InputLight  `json:"lights"`
	Player      *InputPlayer   `json:"player"`
	ScaleFactor float64        `json:"scaleFactor"`
	DisableLoop bool           `json:"disableLoop"`
}
