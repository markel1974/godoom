package textures

// Animation represents a sequence of textures that can be animated over time.
// frame stores the current single texture when no animation sequence is used.
// frames holds a list of textures to be used as frames in the animation sequence.
// tick tracks the animation progress and determines the current frame based on time.
// singleFrame indicates whether the animation consists of only a single static frame.
type Animation struct {
	frame       *Texture
	frames      []*Texture
	tick        uint
	singleFrame bool
}

// NewAnimation creates a new Animation instance from a slice of Texture pointers, handling cases with zero or one frame.
func NewAnimation(frames []*Texture) *Animation {
	a := &Animation{
		frames:      frames,
		tick:        0,
		frame:       nil,
		singleFrame: false,
	}
	if len(frames) == 0 {
		a.singleFrame = true
	} else if len(frames) == 1 {
		a.singleFrame = true
		a.frame = frames[0]
	}
	return a
}

// Advance progresses the animation by one tick and returns the current frame's texture based on the animation state.
func (a *Animation) Advance() *Texture {
	if a.singleFrame {
		return a.frame
	}
	a.tick++
	const tickInterval = 32
	frame := a.tick / tickInterval
	return a.frames[int(frame)%len(a.frames)]
}
