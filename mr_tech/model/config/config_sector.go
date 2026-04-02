package config

import "encoding/json"

// DefinitionJoin represents a join action in a system with a numeric value of 3.
// DefinitionVoid represents a void action in a system with a numeric value of 1.
// DefinitionWall represents a wall action in a system with a numeric value of 2.
// DefinitionUnknown represents an undefined state in a system with a numeric value of 0.
const (
	DefinitionJoin    = 3
	DefinitionVoid    = 1
	DefinitionWall    = 2
	DefinitionUnknown = 0
)

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

// Print serializes the ConfigSector struct into JSON format; optionally indents the output if the indent parameter is true.
func (is *ConfigSector) Print(indent bool) []byte {
	if indent {
		d, _ := json.MarshalIndent(is, "", "  ")
		return d
	}
	d, _ := json.Marshal(is)
	return d
}
