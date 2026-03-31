package open_gl

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/model"
)

// FrameLights manages light data for rendering, including position, color, intensity, and other attributes.
type FrameLights struct {
	data   []float32
	count  int
	stride int32
}

// NewFrameLights creates and returns a new instance of FrameLights with a specified maximum number of lights.
func NewFrameLights(maxLights int) *FrameLights {
	const stride = 16
	return &FrameLights{
		data:   make([]float32, 0, maxLights*stride),
		stride: stride,
	}
}

// Reset clears all previously stored light data and resets the count to prepare for new light data.
func (f *FrameLights) Reset() {
	f.data = f.data[:0]
	f.count = 0
}

// LightsStride returns the total stride value for lights, calculated as the internal stride multiplied by 4.
func (f *FrameLights) LightsStride() int32 {
	return f.stride * 4
}

// GetLights retrieves the current list of light data and the count of lights in the frame as a slice and an integer.
func (f *FrameLights) GetLights() ([]float32, int32) {
	return f.data, int32(f.count)
}

// Create processes the given light and adds its data to the FrameLights if the light type is supported.
func (f *FrameLights) Create(light *model.Light) {
	r, g, b := float32(1.0), float32(1.0), float32(1.0)
	dirGlX, dirGlY, dirGlZ := float32(0.0), float32(0.0), float32(0.0)
	cutOff := float32(0)
	outerCutOff := float32(0)
	pos := light.GetPos()
	intensity := float32(light.GetIntensity())
	falloff := float32(0.0)
	lightType := float32(-1)

	switch light.GetKind() {
	case model.LightKindOpenAir:
		return
		pos.Z = 100
		r, g, b = float32(1.0), float32(1.0), float32(1.0)
		lightType = 0
		falloff = 500.0
	case model.LightKindAmbient:
		r, g, b = float32(1.0), float32(1.0), float32(1.0)
		lightType = 0
		falloff = 10.0
	case model.LightKindSpot:
		lightType = 1
		falloff = 100.0
		r, g, b = float32(1.0), float32(1.0), float32(1.0)
		dirGlX, dirGlY, dirGlZ = float32(0.0), float32(-1.0), float32(0.0)
		cutOff = float32(math.Cos(35.0 * math.Pi / 180.0))
		outerCutOff = float32(math.Cos(40 * math.Pi / 180.0))
	case model.LightKindNone:
		return
	default:
		lightType = 0
	}

	f.Add(
		float32(pos.X), float32(pos.Z), float32(-pos.Y), lightType,
		r, g, b, intensity,
		dirGlX, dirGlY, dirGlZ, falloff,
		cutOff, outerCutOff, 0.0, 0.0,
	)
}

// Add appends properties of a light source (position, color, direction, etc.) to the FrameLights storage.
func (f *FrameLights) Add(
	posX, posY, posZ, lightType float32,
	colR, colG, colB, intensity float32,
	dirX, dirY, dirZ, falloff float32,
	cutOff, outerCutOff, pad1, pad2 float32,
) {
	if f.count*int(f.stride+f.stride) > len(f.data) {
		f.Grow()
	}
	f.data = append(f.data,
		posX, posY, posZ, lightType,
		colR, colG, colB, intensity,
		dirX, dirY, dirZ, falloff,
		cutOff, outerCutOff, pad1, pad2,
	)
	f.count++
}

// Grow doubles the size of the internal data slice or initializes it if empty to accommodate additional elements.
func (f *FrameLights) Grow() {
	newSize := len(f.data) * 2
	if newSize == 0 {
		newSize = 128 * int(f.stride)
	}
	newData := make([]float32, newSize)
	copy(newData, f.data)
	f.data = newData
}
