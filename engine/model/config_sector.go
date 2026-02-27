package model

import "encoding/json"

// ConfigSector represents a Sector configuration in a level, including geometric, texture, and tag information.
type ConfigSector struct {
	Id                 string           `json:"id"`
	Ceil               float64          `json:"ceil"`
	Floor              float64          `json:"floor"`
	Textures           bool             `json:"textures"`
	TextureFloor       string           `json:"textureFloor"`
	TextureCeil        string           `json:"textureCeil"`
	TextureUpper       string           `json:"textureUpper"`
	TextureLower       string           `json:"textureLower"`
	TextureWall        string           `json:"textureWall"`
	TextureScaleFactor float64          `json:"textureScaleFactor"`
	Segments           []*ConfigSegment `json:"segments"`
	Tag                string           `json:"tag"`
}

// NewConfigSector creates a new ConfigSector instance with the given id, initializing its fields with default values.
func NewConfigSector(id string) *ConfigSector {
	return &ConfigSector{Id: id}
}

// Clone creates a deep copy of the ConfigSector, optionally cloning its segments based on the cloneSegments parameter.
func (is *ConfigSector) Clone(cloneSegments bool) *ConfigSector {
	out := NewConfigSector(is.Id)
	out.Ceil = is.Ceil
	out.Floor = is.Floor
	out.Textures = is.Textures
	out.TextureFloor = is.TextureFloor
	out.TextureCeil = is.TextureCeil
	out.TextureUpper = is.TextureUpper
	out.TextureLower = is.TextureLower
	out.TextureWall = is.TextureWall
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

// Print serializes the ConfigSector struct into JSON format; optionally indents the output if the indent parameter is true.
func (is *ConfigSector) Print(indent bool) []byte {
	if indent {
		d, _ := json.MarshalIndent(is, "", "  ")
		return d
	}
	d, _ := json.Marshal(is)
	return d
}
