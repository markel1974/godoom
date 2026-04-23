package open_gl

import (
	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/model"
	"github.com/markel1974/godoom/mr_tech/textures"
)

//Omnidirezionali/CubeMaps

// ShadowLightNumber defines the maximum number of shadow-casting lights supported in the frame lighting system.
const ShadowLightNumber = 8

// Light represents the properties and attributes of a light source in 3D space, including position, direction, and intensity.
type Light struct {
	X, Y, Z                   float32
	Kind                      float32
	R, G, B, Intensity        float32
	DirX, DirY, DirZ, Falloff float32
	CutOff, OuterCutOff       float32
	Score                     float32
}

// FrameLights represents a container for managing lights and their properties in a frame-based rendering system.
type FrameLights struct {
	data              []float32
	index             int
	freezeIndex       int
	stride            int32
	shadowLights      [ShadowLightNumber]*Light
	shadowLightsIndex int32
	camX              float32
	camY              float32
	camZ              float32
}

// NewFrameLights initializes and returns a new FrameLights instance with a specified maximum number of lights.
func NewFrameLights(maxLights int) *FrameLights {
	const stride = 16
	fl := &FrameLights{
		data:              make([]float32, maxLights*stride),
		index:             0,
		freezeIndex:       0,
		shadowLightsIndex: 0,
		stride:            stride,
	}
	for idx := range fl.shadowLights {
		fl.shadowLights[idx] = &Light{}
	}
	return fl
}

// DeepReset resets the freezeIndex, shadowLightsIndex, and calls the Reset method to perform a full reset of the FrameLights.
func (f *FrameLights) DeepReset() {
	f.freezeIndex = 0
	f.shadowLightsIndex = 0
	f.Reset()
}

// Reset sets the current light index to the previously saved freeze index, effectively discarding added lights beyond that point.
func (f *FrameLights) Reset() {
	f.index = f.freezeIndex
}

// Freeze locks the current state of the FrameLights by saving the current index to the freezeIndex.
func (f *FrameLights) Freeze() {
	f.freezeIndex = f.index
}

// LightsStride returns the stride of the light data, scaled by 4, representing the total number of float32 per light.
func (f *FrameLights) LightsStride() int32 {
	return f.stride * 4
}

// GetLights retrieves the current light data as a slice of float32 and the count of lights as an int32.
func (f *FrameLights) GetLights() ([]float32, int32) {
	return f.data[:f.index], int32(f.index) / f.stride
}

// GetShadowLights retrieves the shadow-casting lights and their count from the current frame's lighting configuration.
func (f *FrameLights) GetShadowLights() ([ShadowLightNumber]*Light, int32) {
	return f.shadowLights, f.shadowLightsIndex
}

// Prepare resets the shadow lights index and updates the camera position values (camX, camY, camZ).
func (f *FrameLights) Prepare(camX, camY, camZ float32) {
	f.camX, f.camY, f.camZ = camX, camY, camZ
	f.shadowLightsIndex = 0
}

// Create adds a new light to the FrameLights based on its type, position, intensity, and other properties.
func (f *FrameLights) Create(light *model.Light) {
	r, g, b := float32(light.GetRed()), float32(light.GetGreen()), float32(light.GetBlue())
	dirGlX, dirGlY, dirGlZ := float32(light.GetDirX()), float32(light.GetDirY()), float32(light.GetDirZ())
	cutOff := float32(light.GetCutOff())
	outerCutOff := float32(light.GetOuterCutOff())
	intensity := float32(light.GetIntensityStyled(textures.CurrentTick()))

	falloff := float32(light.GetFalloff())
	lightType := float32(-1)
	posX, posY, posZ := light.GetPosXYZ()

	switch light.GetKind() {
	case config.LightKindOpenAir:
		// pos.Z = 100
		// lightType = 0
		return
	case config.LightKindAmbient:
		lightType = 0
	case config.LightKindSpot:
		//const baseCutoff = 30.0
		//const baseOuterCutOff = 40.0
		lightType = 1
		//dirGlX, dirGlY, dirGlZ = float32(0.0), float32(-1.0), float32(0.0)
		//cutOff = float32(math.Cos(35.0 * math.Pi / 180.0))
		//outerCutOff = float32(math.Cos(40 * math.Pi / 180.0))
		added := f.addShadowLight(
			float32(posX), float32(posZ), float32(-posY),
			lightType, r, g, b, intensity,
			dirGlX, dirGlY, dirGlZ, falloff,
			cutOff, outerCutOff,
		)
		if added {
			return
		}
	case config.LightKindNone:
		return
	default:
		lightType = 0
	}

	f.add(
		float32(posX), float32(posZ), float32(-posY), lightType,
		r, g, b, intensity,
		dirGlX, dirGlY, dirGlZ, falloff,
		cutOff, outerCutOff, 0.0, 0.0,
	)
}

// add adds a light's parameters to the data buffer and updates the index, expanding storage if necessary.
func (f *FrameLights) add(
	posX, posY, posZ, lightType float32,
	colR, colG, colB, intensity float32,
	dirX, dirY, dirZ, falloff float32,
	cutOff, outerCutOff, pad1, pad2 float32,
) {
	idx := f.index
	if idx+int(f.stride) > len(f.data) {
		f.grow()
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

// addShadowLight adds a shadow-casting light to the list of frame lights based on its attributes and distance to the camera.
// Returns true if the light was successfully added, or false if it didn't qualify or was replaced in the min-heap.
func (f *FrameLights) addShadowLight(
	posX, posY, posZ, lightType float32,
	colR, colG, colB, intensity float32,
	dirX, dirY, dirZ, falloff float32,
	cutOff, outerCutOff float32,
) bool {
	dx, dy, dz := posX-f.camX, posY-f.camY, posZ-f.camZ
	distSq := dx*dx + dy*dy + dz*dz
	if distSq > (falloff * falloff * 4.0) {
		return false
	}
	score := intensity / (distSq + 1.0)
	// Fase 1: Riempimento iniziale (0-7)
	if f.shadowLightsIndex < ShadowLightNumber {
		light := f.shadowLights[f.shadowLightsIndex]
		f.fillLightStruct(light, posX, posY, posZ, lightType, colR, colG, colB, intensity, dirX, dirY, dirZ, falloff, cutOff, outerCutOff, score)
		f.shadowLightsIndex++
		// Quando arriviamo a 8, costruiamo l'heap iniziale (una tantum per frame)
		if f.shadowLightsIndex == ShadowLightNumber {
			f.buildMinHeap()
		}
		return true
	}
	// La luce peggiore è SEMPRE all'indice 0. Ricerca O(1).
	if score <= f.shadowLights[0].Score {
		return false
	}
	// Sostituzione: declassiamo la radice (la peggiore) e inseriamo la nuova
	worst := f.shadowLights[0]
	f.add(
		worst.X, worst.Y, worst.Z, worst.Kind,
		worst.R, worst.G, worst.B, worst.Intensity,
		worst.DirX, worst.DirY, worst.DirZ, worst.Falloff,
		worst.CutOff, worst.OuterCutOff, 0.0, 0.0,
	)
	f.fillLightStruct(worst, posX, posY, posZ, lightType, colR, colG, colB, intensity, dirX, dirY, dirZ, falloff, cutOff, outerCutOff, score)
	// Ripristiniamo l'ordine dell'heap in O(log N)
	f.minHeapFixDown(0, ShadowLightNumber)
	return true
}

// fillLightStruct populates the given Light struct with the provided positional, directional, and light-related properties.
func (f *FrameLights) fillLightStruct(l *Light, posX, posY, posZ, lightType, colR, colG, colB, intensity, dirX, dirY, dirZ, falloff, cutOff, outerCutOff, score float32) {
	l.X, l.Y, l.Z = posX, posY, posZ
	l.Kind = lightType
	l.R, l.G, l.B = colR, colG, colB
	l.Intensity = intensity
	l.DirX, l.DirY, l.DirZ = dirX, dirY, dirZ
	l.Falloff = falloff
	l.CutOff, l.OuterCutOff = cutOff, outerCutOff
	l.Score = score
}

// grow dynamically expands the size of the underlying data slice when more capacity is needed.
func (f *FrameLights) grow() {
	newSize := len(f.data) * 2
	if newSize == 0 {
		newSize = 128 * int(f.stride)
	}
	newData := make([]float32, newSize)
	copy(newData, f.data)
	f.data = newData
}

// buildMinHeap constructs a min-heap for the shadowLights array based on their scores in O(n) time complexity.
func (f *FrameLights) buildMinHeap() {
	for i := (ShadowLightNumber / 2) - 1; i >= 0; i-- {
		f.minHeapFixDown(i, ShadowLightNumber)
	}
}

// minHeapFixDown restores the min-heap property for a subtree rooted at index i within a heap of size n.
func (f *FrameLights) minHeapFixDown(i, n int) {
	for {
		left := 2*i + 1
		if left >= n || left < 0 {
			break
		}
		smallest := left
		if right := left + 1; right < n && f.shadowLights[right].Score < f.shadowLights[left].Score {
			smallest = right
		}
		if f.shadowLights[smallest].Score >= f.shadowLights[i].Score {
			break
		}
		// Swap pointers
		f.shadowLights[i], f.shadowLights[smallest] = f.shadowLights[smallest], f.shadowLights[i]
		i = smallest
	}
}
