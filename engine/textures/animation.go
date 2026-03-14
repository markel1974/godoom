package textures

// _tickInterval defines the number of global ticks between frame updates in animations.
var _tickInterval = uint64(32)

// _globalTick is a monotonically increasing counter used to track global animation or system ticks within the application.
var _globalTick uint64

// SetTickInterval sets the tick interval duration in arbitrary units.
func SetTickInterval(interval uint64) {
	_tickInterval = interval
}

// Tick increments the global tick counter used for tracking application-wide progression or state updates.
func Tick() {
	_globalTick++
}

// Animation represents a collection of 2D texture frames used for rendering animations.
type Animation struct {
	frame       *Texture
	frames      []*Texture
	totalFrames uint64
	kind        int
	scaleW      float64
	scaleH      float64
}

// NewAnimation creates a new Animation instance from a provided slice of Texture pointers.
// If the slice contains only one Texture, it is set as the current frame.
func NewAnimation(frames []*Texture, kind int, scaleW float64, scaleH float64) *Animation {
	a := &Animation{
		frames:      frames,
		frame:       nil,
		totalFrames: uint64(len(frames)),
		kind:        kind,
		scaleW:      scaleW,
		scaleH:      scaleH,
	}
	if a.totalFrames == 1 {
		a.frame = frames[0]
	}
	return a
}

// Kind returns the type of the animation as an integer value.
func (a *Animation) Kind() int {
	return a.kind
}

// CurrentFrame returns the currently active frame of the animation based on global tick and tick interval.
func (a *Animation) CurrentFrame() *Texture {
	if a.totalFrames > 1 {
		frameIdx := (_globalTick / _tickInterval) % a.totalFrames
		return a.frames[frameIdx]
	}
	return a.frame
}

// ScaleFactor returns the scaling factors for width and height of the animation as two float64 values.
func (a *Animation) ScaleFactor() (float64, float64) {
	return a.scaleW, a.scaleH
}
