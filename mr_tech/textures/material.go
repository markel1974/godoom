package textures

import (
	"fmt"

	"github.com/markel1974/godoom/mr_tech/model/geometry"
)

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

func CurrentTick() uint64 {
	return _globalTick
}

// TickGrouped calculates the tick grouped by the specified group size and returns the result as an integer.
func TickGrouped(tick uint64, groupSize int) float64 {
	frameFloat := float64(tick) / float64(groupSize)
	return frameFloat
}

// Material represents a collection of 2D texture frames used for rendering animations.
type Material struct {
	frame       *Texture
	frames      []*Texture
	totalFrames uint64
	kind        int
	gScale      geometry.XYZ
	scaleW      float64
	scaleH      float64
	u           float64
	v           float64
}

// NewMaterial creates a new Material instance from a provided slice of Texture pointers.
// If the slice contains only one Texture, it is set as the current frame.
func NewMaterial(frames []*Texture, kind int, gScale geometry.XYZ, scaleW, scaleH, u, v float64) *Material {
	if scaleW == 0 {
		scaleW = 1
	}
	if scaleH == 0 {
		scaleH = 1
	}
	a := &Material{
		frames:      frames,
		frame:       nil,
		totalFrames: uint64(len(frames)),
		kind:        kind,
		gScale:      gScale,
		scaleW:      scaleW,
		scaleH:      scaleH,
		u:           u,
		v:           v,
	}
	for _, frame := range frames {
		if frame == nil {
			a.totalFrames = 0
			a.frames = nil
			fmt.Println("nil frame")
			break
		}
		frame.SetScaleFactor(gScale, scaleW, scaleH)
	}
	if a.totalFrames == 1 {
		a.frame = frames[0]
	}
	return a
}

// Kind returns the type of the animation as an integer value.
func (a *Material) Kind() int {
	return a.kind
}

// CurrentFrame returns the currently active frame of the animation based on global tick and tick interval.
func (a *Material) CurrentFrame() *Texture {
	if a.totalFrames > 1 {
		frameIdx := (_globalTick / _tickInterval) % a.totalFrames
		return a.frames[frameIdx]
	}
	return a.frame
}
