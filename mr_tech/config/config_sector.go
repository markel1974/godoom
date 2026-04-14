package config

// Sector represents a Sector configuration in a level, including geometric, texture, and tag information.
type Sector struct {
	Id       string       `json:"id"`
	CeilY    float64      `json:"ceilY"`
	FloorY   float64      `json:"floorY"`
	Ceil     *Animation   `json:"ceil"`
	Floor    *Animation   `json:"floor"`
	Light    *LightSector `json:"light"`
	Segments []*Segment   `json:"segments"`
	Tag      string       `json:"tag"`
}

// NewConfigSector creates a new Sector instance with the given id, initializing its fields with default values.
func NewConfigSector(id string, lightIntensity float64, kind LightKind, falloff float64) *Sector {
	return &Sector{
		Id:    id,
		Ceil:  nil,
		Floor: nil,
		Light: NewConfigLightSector(lightIntensity, kind, falloff),
	}
}
