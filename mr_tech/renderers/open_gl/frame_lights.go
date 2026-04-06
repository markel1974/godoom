package open_gl

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/model"
	"github.com/markel1974/godoom/mr_tech/model/config"
)

// FrameLights manages light data for rendering, including position, color, intensity, and other attributes.
type FrameLights struct {
	data        []float32
	index       int
	freezeIndex int
	stride      int32
}

// NewFrameLights creates and returns a new instance of FrameLights with a specified maximum number of lights.
func NewFrameLights(maxLights int) *FrameLights {
	const stride = 16
	return &FrameLights{
		data:        make([]float32, maxLights*stride),
		index:       0,
		freezeIndex: 0,
		stride:      stride,
	}
}

// DeepReset resets the freezeIndex and calls Reset to clear the stored light data and prepare for new inputs.
func (f *FrameLights) DeepReset() {
	f.freezeIndex = 0
	f.Reset()
}

// Reset clears all previously stored light data and resets the count to prepare for new light data.
func (f *FrameLights) Reset() {
	f.index = f.freezeIndex
}

// Freeze sets the freezeIndex to the current index, marking the current state for later resets.
func (f *FrameLights) Freeze() {
	f.freezeIndex = f.index
}

// LightsStride returns the total stride value for lights, calculated as the internal stride multiplied by 4.
func (f *FrameLights) LightsStride() int32 {
	return f.stride * 4
}

// GetLights retrieves the current list of light data and the count of lights in the frame as a slice and an integer.
func (f *FrameLights) GetLights() ([]float32, int32) {
	return f.data[:f.index], int32(f.index) / f.stride
}

// Create processes the given light and adds its data to the FrameLights if the light type is supported.
func (f *FrameLights) Create(light *model.Light) {
	r, g, b := float32(1.0), float32(1.0), float32(1.0)
	dirGlX, dirGlY, dirGlZ := float32(0.0), float32(0.0), float32(0.0)
	cutOff := float32(0)
	outerCutOff := float32(0)

	pos := light.GetPos()
	intensity := float32(light.GetIntensity())

	// Estrazione diretta del falloff configurato nel modello
	falloff := float32(light.GetFalloff())

	lightType := float32(-1)

	switch light.GetKind() {
	case config.LightKindOpenAir:
		return
		// pos.Z = 100
		// r, g, b = float32(1.0), float32(1.0), float32(1.0)
		// lightType = 0
	case config.LightKindAmbient:
		r, g, b = float32(1.0), float32(1.0), float32(1.0)
		lightType = 0
	case config.LightKindSpot:
		lightType = 1
		r, g, b = float32(1.0), float32(1.0), float32(1.0)
		dirGlX, dirGlY, dirGlZ = float32(0.0), float32(-1.0), float32(0.0)
		cutOff = float32(math.Cos(35.0 * math.Pi / 180.0))
		outerCutOff = float32(math.Cos(40 * math.Pi / 180.0))
	case config.LightKindNone:
		return
	default:
		lightType = 0
	}

	f.add(
		float32(pos.X), float32(pos.Z), float32(-pos.Y), lightType,
		r, g, b, intensity,
		dirGlX, dirGlY, dirGlZ, falloff,
		cutOff, outerCutOff, 0.0, 0.0,
	)
}

// Add appends properties of a light source (position, color, direction, etc.) to the FrameLights storage.
func (f *FrameLights) add(
	posX, posY, posZ, lightType float32,
	colR, colG, colB, intensity float32,
	dirX, dirY, dirZ, falloff float32,
	cutOff, outerCutOff, pad1, pad2 float32,
) {
	idx := f.index
	if idx+int(f.stride) > len(f.data) {
		f.Grow()
	}
	f.index += int(f.stride)

	f.data[idx] = posX
	f.data[idx+1] = posY
	f.data[idx+2] = posZ
	f.data[idx+3] = lightType
	f.data[idx+4] = colR
	f.data[idx+5] = colG
	f.data[idx+6] = colB
	f.data[idx+7] = intensity
	f.data[idx+8] = dirX
	f.data[idx+9] = dirY
	f.data[idx+10] = dirZ
	f.data[idx+11] = falloff
	f.data[idx+12] = cutOff
	f.data[idx+13] = outerCutOff
	f.data[idx+14] = pad1
	f.data[idx+15] = pad2
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
