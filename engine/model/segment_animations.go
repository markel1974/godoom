package model

import "github.com/markel1974/godoom/engine/textures"

// SegmentAnimations groups animations for the upper, middle, and lower sections of a 2D segment rendering system.
type SegmentAnimations struct {
	upper  *textures.Animation
	middle *textures.Animation
	lower  *textures.Animation
}

// NewSegmentAnimation creates a new SegmentAnimations instance with specified upper, middle, and lower animations.
func NewSegmentAnimation(upper, middle, lower *textures.Animation) *SegmentAnimations {
	return &SegmentAnimations{
		upper:  upper,
		middle: middle,
		lower:  lower,
	}
}

// Clone creates a new SegmentAnimations instance with copied references to the upper, middle, and lower animations.
func (cst *SegmentAnimations) Clone() *SegmentAnimations {
	out := &SegmentAnimations{}
	out.upper = cst.upper
	out.middle = cst.middle
	out.lower = cst.lower
	return out
}

// Upper retrieves the upper segment's animation from the SegmentAnimations instance.
func (cst *SegmentAnimations) Upper() *textures.Animation {
	return cst.upper
}

// Middle returns the `middle` animation of the `SegmentAnimations` instance.
func (cst *SegmentAnimations) Middle() *textures.Animation {
	return cst.middle
}

// Lower returns the lower segment's animation associated with the SegmentAnimations instance.
func (cst *SegmentAnimations) Lower() *textures.Animation {
	return cst.lower
}
