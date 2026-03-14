package model

import "encoding/json"

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

// Clone creates a deep copy of the ConfigSector, optionally cloning its segments based on the cloneSegments parameter.
func (is *ConfigSector) Clone(cloneSegments bool) *ConfigSector {
	out := NewConfigSector(is.Id, is.Light.Intensity, is.Light.Kind)
	out.CeilY = is.CeilY
	out.FloorY = is.FloorY
	if is.Ceil != nil {
		out.Ceil = is.Ceil.Clone()
	}
	if is.Floor != nil {
		out.Floor = is.Floor.Clone()
	}
	out.Tag = is.Tag
	out.Segments = nil
	if cloneSegments {
		out.Segments = make([]*ConfigSegment, len(is.Segments))
		for idx, seg := range is.Segments {
			out.Segments[idx] = seg.Clone()
		}
	}
	return out
}

// GetCentroid calculates the centroid of the polygon formed by the sector's segments based on their vertex coordinates.
func (is *ConfigSector) GetCentroid() XY {
	var signedArea, cx, cy float64

	for i := range is.Segments {
		x0, y0 := is.Segments[i].Start.X, is.Segments[i].Start.Y
		x1, y1 := is.Segments[i].End.X, is.Segments[i].End.Y

		// Prodotto vettoriale 2D (determinante)
		a := (x0 * y1) - (x1 * y0)

		signedArea += a
		cx += (x0 + x1) * a
		cy += (y0 + y1) * a
	}

	signedArea *= 0.5

	if signedArea == 0 {
		// Fallback di sicurezza per topologia degenere (es. area nulla)
		return XY{X: is.Segments[0].Start.X, Y: is.Segments[0].Start.Y}
	}

	return XY{
		X: cx / (6.0 * signedArea),
		Y: cy / (6.0 * signedArea),
	}
}

// Print serializes the ConfigSector struct into JSON format; optionally indents the output if the indent parameter is true.
func (is *ConfigSector) Print(indent bool) []byte {
	if indent {
		d, _ := json.MarshalIndent(is, "", "  ")
		return d
	}
	d, _ := json.Marshal(is)
	return d
}
