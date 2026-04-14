package config

import "github.com/markel1974/godoom/mr_tech/utils"

type AnimationKind int

const (
	AnimationKindNone AnimationKind = iota
	AnimationKindLoop
	AnimationKindSky
)

// Animation represents animation properties including a sequence of frames and the type of animation.
type Animation struct {
	Id     string        `json:"id"`
	Frames []string      `json:"frames"`
	Kind   AnimationKind `json:"kind"`
	ScaleW float64       `json:"scaleW"`
	ScaleH float64       `json:"scaleH"`
}

// NewConfigAnimation creates and initializes a new Animation instance with the provided animation and kind values.
func NewConfigAnimation(animation []string, kind AnimationKind, scaleW float64, scaleH float64) *Animation {
	return &Animation{
		Id:     utils.NextUUId(),
		Frames: animation,
		Kind:   kind,
		ScaleW: scaleW,
		ScaleH: scaleH,
	}
}
