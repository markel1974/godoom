package model

// ConfigSegmentAnimations represents a grouping of textures categorized as upper, middle, and lower segments for a configuration.
type ConfigSegmentAnimations struct {
	Upper  []string `json:"upper"`
	Middle []string `json:"middle"`
	Lower  []string `json:"lower"`
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
	out.Upper = cst.Upper
	out.Middle = cst.Middle
	out.Lower = cst.Lower
	return out
}
