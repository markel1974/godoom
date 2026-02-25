package model

// ConfigPlayer represents the player's position, orientation, and current Sector in the game environment.
type ConfigPlayer struct {
	Position XY      `json:"position"`
	Angle    float64 `json:"angle"`
	Sector   string  `json:"Sector"`
}
