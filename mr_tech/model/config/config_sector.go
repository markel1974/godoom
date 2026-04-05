package config

// ConfigSector represents a Sector configuration in a level, including geometric, texture, and tag information.
type ConfigSector struct {
	Id       string           `json:"id"`
	CeilY    float64          `json:"ceilY"`
	FloorY   float64          `json:"floorY"`
	Ceil     *ConfigAnimation `json:"ceil"`
	Floor    *ConfigAnimation `json:"floor"`
	Light    *ConfigLight     `json:"light"`
	Segments []*ConfigSegment `json:"segments"`
	Tag      string           `json:"tag"`
}

// NewConfigSector creates a new ConfigSector instance with the given id, initializing its fields with default values.
func NewConfigSector(id string, lightIntensity float64, kind LightKind) *ConfigSector {
	return &ConfigSector{
		Id:    id,
		Ceil:  nil,
		Floor: nil,
		Light: NewConfigLight(lightIntensity, kind),
	}
}
