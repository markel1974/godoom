package config

import "github.com/markel1974/godoom/mr_tech/model/geometry"

// Sector represents a Sector configuration in a level, including geometric, texture, and tag information.
type Sector struct {
	Id       string     `json:"id"`
	CeilY    float64    `json:"ceilY"`
	FloorY   float64    `json:"floorY"`
	Ceil     *Material  `json:"ceil"`
	Floor    *Material  `json:"floor"`
	Light    *Light     `json:"light"`
	Segments []*Segment `json:"segments"`
	Tag      string     `json:"tag"`
}

// NewConfigSector creates a new Sector instance with the given id, initializing its fields with default values.
func NewConfigSector(id string, lightIntensity float64, kind LightKind, falloff float64) *Sector {
	return &Sector{
		Id:    id,
		Ceil:  nil,
		Floor: nil,
		Light: NewConfigLight(geometry.XYZ{}, lightIntensity, kind, falloff),
	}
}

// Scale scales all the segments of the Sector by the given scale factor by applying it to their start and end points.
func (s *Sector) Scale(scale float64) {
	for _, seg := range s.Segments {
		seg.Start.Scale(scale)
		seg.End.Scale(scale)
	}
}
