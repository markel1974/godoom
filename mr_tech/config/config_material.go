package config

import "github.com/markel1974/godoom/mr_tech/utils"

type AnimationKind int

const (
	AnimationKindNone AnimationKind = iota
	AnimationKindLoop
	AnimationKindSky
)

// Material represents animation properties including a sequence of frames and the type of animation.
type Material struct {
	Id     string        `json:"id"`
	Frames []string      `json:"frames"`
	Kind   AnimationKind `json:"kind"`
	ScaleW float64       `json:"scaleW"`
	ScaleH float64       `json:"scaleH"`
	U      float64       `json:"u"`
	V      float64       `json:"v"`
}

// NewConfigMaterial creates and initializes a new Material instance with the provided animation and kind values.
func NewConfigMaterial(frames []string, kind AnimationKind, scaleW, scaleH, u, v float64) *Material {
	return &Material{
		Id:     utils.NextUUId(),
		Frames: frames,
		Kind:   kind,
		ScaleW: scaleW,
		ScaleH: scaleH,
		U:      u,
		V:      v,
	}
}
