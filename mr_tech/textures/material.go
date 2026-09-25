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

	clampAnim bool
	startTick uint64

	cullMode        int
	depthWrite      bool
	alphaTest       float32
	isFog           bool
	fogColor        []float32
	polygonOffset   bool
	sort            int
	noMipmaps       bool
	noPicMip        bool
	entityMergeable bool
	deformVertexes  [][]string
	surfaceParms    map[string]bool
	skyParms        []string
	cloudParms      []string
	fogGen          []string
	isLightning     bool
	isSky           bool
	isBacksided     bool
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

func (m *Material) SetClampAnim(clamp bool) {
	m.clampAnim = clamp
}

func (m *Material) RestartAnim() {
	m.startTick = GlobalTick()
}

// CurrentFrame returns the currently active frame of the animation based on global tick and tick interval.
func (m *Material) CurrentFrame() *Texture {
	if m.totalFrames > 1 {
		if m.clampAnim {
			return m.clampedFrame()
		}
		return m.frames[_currentTick%m.totalFrames]
	}
	return m.frame
}

func (m *Material) clampedFrame() *Texture {
	elapsed := uint64(0)
	if _globalTick > m.startTick {
		elapsed = _globalTick - m.startTick
	}
	elapsedTick := elapsed / _tickInterval
	if elapsedTick >= m.totalFrames-1 {
		elapsedTick = m.totalFrames - 1
	}
	return m.frames[elapsedTick]
}

func (m *Material) BlendMode() int {
	return m.blendMode
}

func (m *Material) SetBlendMode(b int) {
	m.blendMode = b
}

// SetProperties assigns all the advanced rendering properties extracted from the shader configuration.
func (m *Material) SetProperties(cullMode int, depthWrite bool, alphaTest float32, isFog bool, fogColor []float32, polygonOffset bool, sort int, noMipmaps bool, noPicMip bool, entityMergeable bool, deformVertexes [][]string, surfaceParms map[string]bool, skyParms []string, cloudParms []string, fogGen []string, isLightning bool, isSky bool, isBacksided bool) {
	m.cullMode = cullMode
	m.depthWrite = depthWrite
	m.alphaTest = alphaTest
	m.isFog = isFog
	m.fogColor = fogColor
	m.polygonOffset = polygonOffset
	m.sort = sort
	m.noMipmaps = noMipmaps
	m.noPicMip = noPicMip
	m.entityMergeable = entityMergeable
	m.deformVertexes = deformVertexes
	m.surfaceParms = surfaceParms
	m.skyParms = skyParms
	m.cloudParms = cloudParms
	m.fogGen = fogGen
	m.isLightning = isLightning
	m.isSky = isSky
	m.isBacksided = isBacksided
}

// GetCullMode returns the culling mode (e.g. front, back, none).
func (m *Material) GetCullMode() int { return m.cullMode }

// GetDepthWrite returns whether the material writes to the depth buffer.
func (m *Material) GetDepthWrite() bool { return m.depthWrite }

// GetAlphaTest returns the alpha threshold for discarding pixels.
func (m *Material) GetAlphaTest() float32 { return m.alphaTest }

// IsFog returns whether this is a fog material.
func (m *Material) IsFog() bool { return m.isFog }

// GetFogColor returns the RGB components of the fog color.
func (m *Material) GetFogColor() []float32 { return m.fogColor }

// GetPolygonOffset returns whether this material needs depth offset (e.g. decals).
func (m *Material) GetPolygonOffset() bool { return m.polygonOffset }

// GetSort returns the sorting order of the material.
func (m *Material) GetSort() int { return m.sort }

// IsEntityMergeable returns whether this material's entity can be merged.
func (m *Material) IsEntityMergeable() bool { return m.entityMergeable }

// GetDeformVertexes returns the vertex deformation stages.
func (m *Material) GetDeformVertexes() [][]string { return m.deformVertexes }

// GetSurfaceParms returns the surface parameters dictionary.
func (m *Material) GetSurfaceParms() map[string]bool { return m.surfaceParms }

// GetSkyParms returns the sky parameters array.
func (m *Material) GetSkyParms() []string { return m.skyParms }

// GetCloudParms returns the cloud parameters array.
func (m *Material) GetCloudParms() []string { return m.cloudParms }

// GetFogGen returns the fog generation parameters array.
func (m *Material) GetFogGen() []string { return m.fogGen }

// IsLightning returns whether this is a lightning material.
func (m *Material) IsLightning() bool { return m.isLightning }

// IsSky returns whether this material represents the sky.
func (m *Material) IsSky() bool { return m.isSky }

// IsBacksided returns whether this material is backsided.
func (m *Material) IsBacksided() bool { return m.isBacksided }
