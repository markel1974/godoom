package model

// ConfigSectorAnimations defines texture configuration for a sector, including floor/ceiling textures and scaling factor.
type ConfigSectorAnimations struct {
	Floors      []string `json:"floor"`
	Ceils       []string `json:"ceil"`
	ScaleFactor float64  `json:"scaleFactor"`
}

// NewConfigSectorAnimations creates a new instance of ConfigSectorAnimations with default values.
func NewConfigSectorAnimations() *ConfigSectorAnimations {
	return &ConfigSectorAnimations{
		Floors:      nil,
		Ceils:       nil,
		ScaleFactor: 1.0,
	}
}

// Clone creates a deep copy of the ConfigSectorAnimations instance, duplicating its fields into a new object.
func (cst *ConfigSectorAnimations) Clone() *ConfigSectorAnimations {
	out := NewConfigSectorAnimations()
	out.Floors = cst.Floors
	out.Ceils = cst.Ceils
	out.ScaleFactor = cst.ScaleFactor
	return out
}
