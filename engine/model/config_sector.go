package model

import "encoding/json"

// ConfigSector represents a Sector configuration in a level, including geometric, texture, and tag information.
type ConfigSector struct {
	Id                 string           `json:"id"`
	Ceil               float64          `json:"ceil"`
	Floor              float64          `json:"floor"`
	TextureFloor       string           `json:"textureFloor"`
	TextureCeil        string           `json:"textureCeil"`
	TextureScaleFactor float64          `json:"textureScaleFactor"`
	LightDistance      float64          `json:"lightDistance"`
	Segments           []*ConfigSegment `json:"segments"`
	Tag                string           `json:"tag"`
}

// NewConfigSector creates a new ConfigSector instance with the given id, initializing its fields with default values.
func NewConfigSector(id string) *ConfigSector {
	return &ConfigSector{
		Id:                 id,
		TextureScaleFactor: 1.0,
		LightDistance:      -1.0,
	}
}

// Clone creates a deep copy of the ConfigSector, optionally cloning its segments based on the cloneSegments parameter.
func (is *ConfigSector) Clone(cloneSegments bool) *ConfigSector {
	out := NewConfigSector(is.Id)
	out.Ceil = is.Ceil
	out.Floor = is.Floor
	out.TextureFloor = is.TextureFloor
	out.TextureCeil = is.TextureCeil
	out.TextureScaleFactor = is.TextureScaleFactor
	out.LightDistance = is.LightDistance
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
