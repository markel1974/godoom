package model

// InputPlayer represents the player's position, orientation, and current sector in the game environment.
type InputPlayer struct {
	Position XY      `json:"position"`
	Angle    float64 `json:"angle"`
	Sector   string  `json:"sector"`
}
