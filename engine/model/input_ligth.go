package model

// InputLight represents a light source with its position, intensity, and associated sector within a 3D space.
type InputLight struct {
	Where  XYZ    `json:"where"`
	Light  XYZ    `json:"light"`
	Sector string `json:"sector"`
}
