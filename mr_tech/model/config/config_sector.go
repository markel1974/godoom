package config

const (

	// DefinitionUnknown represents an unspecified or undefined configuration type in the context of segment definition.
	DefinitionUnknown = iota

	// DefinitionWall represents a configuration segment identified as a wall within the level's geometry or structure.
	DefinitionWall

	// DefinitionJoin represents a constant used to define a join or connection segment within a configuration structure.
	DefinitionJoin
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
