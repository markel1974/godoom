package model

// ConfigSegmentAnimations represents a grouping of textures categorized as upper, middle, and lower segments for a configuration.
type ConfigSegmentAnimations struct {
	Upper  *ConfigAnimation `json:"upper"`
	Middle *ConfigAnimation `json:"middle"`
	Lower  *ConfigAnimation `json:"lower"`
}

// NewConfigSegmentAnimations creates and returns a new instance of ConfigSegmentAnimations with nil textures for upper, middle, and lower segments.
func NewConfigSegmentAnimations() *ConfigSegmentAnimations {
	return &ConfigSegmentAnimations{
		Upper:  nil,
		Middle: nil,
		Lower:  nil,
	}
}

// Clone creates and returns a deep copy of the current ConfigSegmentAnimations instance.
func (cst *ConfigSegmentAnimations) Clone() *ConfigSegmentAnimations {
	out := NewConfigSegmentAnimations()
	out.Upper = cst.Upper.Clone()
	out.Middle = cst.Middle.Clone()
	out.Lower = cst.Lower.Clone()
	return out
}
