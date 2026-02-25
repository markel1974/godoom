package model

// ConfigPlayer represents the player's position, orientation, and current sector in the game environment.
type ConfigPlayer struct {
	Position XY      `json:"position"`
	Angle    float64 `json:"angle"`
	Sector   string  `json:"sector"`
}
