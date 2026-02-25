package model

import "encoding/json"

// InputSector represents a sector configuration in a level, including geometric, texture, and tag information.
type InputSector struct {
	Id           string          `json:"id"`
	Ceil         float64         `json:"ceil"`
	Floor        float64         `json:"floor"`
	Textures     bool            `json:"textures"`
	FloorTexture string          `json:"floorTexture"`
	CeilTexture  string          `json:"ceilTexture"`
	UpperTexture string          `json:"upperTexture"`
	LowerTexture string          `json:"lowerTexture"`
	WallTexture  string          `json:"wallTexture"`
	Segments     []*InputSegment `json:"segments"`
	Tag          string          `json:"tag"`
}

// NewInputSector creates a new InputSector instance with the given id, initializing its fields with default values.
func NewInputSector(id string) *InputSector {
	return &InputSector{Id: id}
}

// Clone creates a deep copy of the InputSector, optionally cloning its segments based on the cloneSegments parameter.
func (is *InputSector) Clone(cloneSegments bool) *InputSector {
	out := NewInputSector(is.Id)
	out.Ceil = is.Ceil
	out.Floor = is.Floor
	out.Textures = is.Textures
	out.FloorTexture = is.FloorTexture
	out.CeilTexture = is.CeilTexture
	out.UpperTexture = is.UpperTexture
	out.LowerTexture = is.LowerTexture
	out.WallTexture = is.WallTexture
	out.Tag = is.Tag
	out.Segments = nil
	if cloneSegments {
		out.Segments = make([]*InputSegment, len(is.Segments))
		for idx, seg := range is.Segments {
			out.Segments[idx] = seg.Clone()
		}
	}
	return out
}

// Print serializes the InputSector struct into JSON format; optionally indents the output if the indent parameter is true.
func (is *InputSector) Print(indent bool) []byte {
	if indent {
		d, _ := json.MarshalIndent(is, "", "  ")
		return d
	}
	d, _ := json.Marshal(is)
	return d
}
