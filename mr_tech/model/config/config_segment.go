package config

import (
	"github.com/markel1974/godoom/mr_tech/model/geometry"
	"github.com/markel1974/godoom/mr_tech/utils"
)

// ConfigSegment represents a segment of input data with spatial coordinates, type, and associated metadata.
type ConfigSegment struct {
	Parent string           `json:"parent"`
	Id     string           `json:"id"`
	Start  geometry.XY      `json:"start"`
	End    geometry.XY      `json:"end"`
	Kind   int              `json:"Kind"`
	Tag    string           `json:"tag"`
	Upper  *ConfigAnimation `json:"upper"`
	Middle *ConfigAnimation `json:"middle"`
	Lower  *ConfigAnimation `json:"lower"`
}

// NewConfigSegment creates a new ConfigSegment instance with the specified parent, Kind, start, and end coordinates.
func NewConfigSegment(parent string, kind int, s geometry.XY, e geometry.XY) *ConfigSegment {
	is := &ConfigSegment{
		Parent: parent,
		Id:     utils.NextUUId(),
		Start:  s,
		End:    e,
		Kind:   kind,
		Tag:    "",
		Upper:  nil,
		Lower:  nil,
		Middle: nil,
	}
	return is
}
