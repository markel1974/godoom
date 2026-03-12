package model

// ConfigSectorAnimations defines texture configuration for a sector, including floor/ceiling textures and scaling factor.
type ConfigSectorAnimations struct {
	Ceil        *ConfigAnimation `json:"ceil"`
	Floor       *ConfigAnimation `json:"floor"`
	ScaleFactor float64          `json:"scaleFactor"`
}

// NewConfigSectorAnimations creates a new instance of ConfigSectorAnimations with default values.
func NewConfigSectorAnimations() *ConfigSectorAnimations {
	return &ConfigSectorAnimations{
		Ceil:        nil,
		Floor:       nil,
		ScaleFactor: 1.0,
	}
}

// Clone creates a deep copy of the ConfigSectorAnimations instance, duplicating its fields into a new object.
func (cst *ConfigSectorAnimations) Clone() *ConfigSectorAnimations {
	out := NewConfigSectorAnimations()
	out.Floor = cst.Floor.Clone()
	out.Ceil = cst.Ceil.Clone()
	out.ScaleFactor = cst.ScaleFactor
	return out
}
