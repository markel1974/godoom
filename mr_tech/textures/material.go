package textures

import (
	"fmt"
)

// _tickInterval defines the number of global ticks between frame updates in animations.
var _tickInterval = uint64(32)

// _globalTick is a monotonically increasing counter used to track global animation or system ticks within the application.
var _globalTick uint64

var _currentTick uint64

// SetTickInterval sets the tick interval duration in arbitrary units.
func SetTickInterval(interval uint64) {
	_tickInterval = interval
}

func TickInterval() uint64 {
	return _tickInterval
}

// Tick increments the global tick counter used for tracking application-wide progression or state updates.
func Tick() {
	_globalTick++
	_currentTick = _globalTick / _tickInterval
}

func GlobalTick() uint64 {
	return _globalTick
}

// TickGrouped calculates the tick grouped by the specified group size and returns the result as an integer.
func TickGrouped(tick uint64, groupSize int) float64 {
	frameFloat := float64(tick) / float64(groupSize)
	return frameFloat
}

// Material represents a collection of 2D texture frames used for rendering animations, along with its shader.
type Material struct {
	blendMode   int
	shader      string
	frame       *Texture
	frames      []*Texture
	totalFrames uint64
	kind        int
	scaleW      float64
	scaleH      float64
	u           float64
	v           float64
}

// NewMaterial creates a new Material instance from a provided slice of Texture pointers.
// If the slice contains only one Texture, it is set as the current frame.
func NewMaterial(shader string, frames []*Texture, kind int, scaleW, scaleH, u, v float64) *Material {
	if scaleW == 0 {
		scaleW = 1
	}
	if scaleH == 0 {
		scaleH = 1
	}
	a := &Material{
		shader:      shader,
		frames:      frames,
		frame:       nil,
		totalFrames: uint64(len(frames)),
		kind:        kind,
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
		frame.SetScaleFactor(scaleW, scaleH)
	}
	if a.totalFrames == 1 {
		a.frame = frames[0]
	}
	return a
}

// Kind returns the type of the animation as an integer value.
func (m *Material) Kind() int {
	return m.kind
}

// U returns the horizontal scroll offset of the material.
func (m *Material) U() float64 {
	return m.u
}

// V returns the vertical scroll offset of the material.
func (m *Material) V() float64 {
	return m.v
}

// Shader returns the name of the shader associated with this material.
func (m *Material) Shader() string {
	return m.shader
}

// CurrentFrame returns the currently active frame of the animation based on global tick and tick interval.
func (m *Material) CurrentFrame() *Texture {
	if m.totalFrames > 1 {
		frameIdx := _currentTick % m.totalFrames
		return m.frames[frameIdx]
	}
	return m.frame
}

func (m *Material) BlendMode() int {
	return m.blendMode
}

func (m *Material) SetBlendMode(b int) {
	m.blendMode = b
}
