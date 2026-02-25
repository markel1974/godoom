package model

// ConfigLight represents a light source with its position, intensity, and associated Sector within a 3D space.
type ConfigLight struct {
	Where  XYZ    `json:"where"`
	Light  XYZ    `json:"light"`
	Sector string `json:"Sector"`
}
