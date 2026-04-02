package config

type AnimationKind int

const (
	AnimationKindNone AnimationKind = iota
	AnimationKindLoop
	AnimationKindSky
)

// ConfigAnimation represents animation properties including a sequence of frames and the type of animation.
type ConfigAnimation struct {
	Frames []string      `json:"frames"`
	Kind   AnimationKind `json:"kind"`
	ScaleW float64       `json:"scaleW"`
	ScaleH float64       `json:"scaleH"`
}

// NewConfigAnimation creates and initializes a new ConfigAnimation instance with the provided animation and kind values.
func NewConfigAnimation(animation []string, kind AnimationKind, scaleW float64, scaleH float64) *ConfigAnimation {
	return &ConfigAnimation{
		Frames: animation,
		Kind:   kind,
		ScaleW: scaleW,
		ScaleH: scaleH,
	}
}

// Clone creates a deep copy of the ConfigAnimation object, duplicating its fields into a new instance.
func (cst *ConfigAnimation) Clone() *ConfigAnimation {
	out := NewConfigAnimation(cst.Frames, cst.Kind, cst.ScaleW, cst.ScaleH)
	return out
}
