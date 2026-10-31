package open_gl

import (
	"unsafe"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/model/lights"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/metrics"
	"github.com/markel1974/godoom/mr_tech/textures"
)

//Omnidirezionali/CubeMaps

// ShadowLightNumber defines the maximum number of shadow-casting lights supported in the frame lighting system.
const ShadowLightNumber = 8

// FrameLights represents a container for managing lights and their properties in a frame-based rendering system.
type FrameLights struct {
	ctx               api.IContext
	data2             []float32
	dataPtr           unsafe.Pointer
	index             int
	freezeIndex       int
	stride            int32
	shadowLights      [ShadowLightNumber]*metrics.Light
	shadowLightsIndex int32
	//glCamX            float32
	//glCamY            float32
	//glCamZ            float32
	useShadowLight bool
	debug          bool
}

// NewFrameLights initializes and returns a new FrameLights instance with a specified maximum number of lights.
func NewFrameLights(ctx api.IContext, maxLights int, useShadowLight bool) *FrameLights {
	const stride = 16
	fl := &FrameLights{
		ctx:               ctx,
		data2:             make([]float32, maxLights*stride),
		index:             0,
		freezeIndex:       0,
		shadowLightsIndex: 0,
		stride:            stride,
		useShadowLight:    useShadowLight,
	}
	fl.dataPtr = ctx.Ptr(fl.data2)
	for idx := range fl.shadowLights {
		fl.shadowLights[idx] = metrics.NewLight()
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
func (f *FrameLights) GetLights() (unsafe.Pointer, int32, int32) {
	size := int32(f.index * 4)
	count := int32(f.index) / f.stride
	return f.dataPtr, size, count
}

// GetShadowLights retrieves the shadow-casting lights and their count from the current frame's lighting configuration.
func (f *FrameLights) GetShadowLights() ([ShadowLightNumber]*metrics.Light, int32) {
	return f.shadowLights, f.shadowLightsIndex
}

// Prepare initializes camera position for lighting calculations and resets the shadow lights index to zero.
func (f *FrameLights) Prepare() {
	//TODO VERIFICA
	//f.glCamX, f.glCamY, f.glCamZ = float32(pX), float32(pY), float32(pZ)
	//f.glCamX, f.glCamY, f.glCamZ = float32(pX), float32(pZ), float32(-pY)
	f.shadowLightsIndex = 0
}

// Create adds a new light to the FrameLights based on its type, position, intensity, and other properties.
func (f *FrameLights) Create(light *lights.Light, distSq float64) {
	r, g, b := float32(light.GetRed()), float32(light.GetGreen()), float32(light.GetBlue())
	posX, posY, posZ := light.GetPosXYZ()
	glPosX, glPosY, glPosZ := float32(posX), float32(posZ), float32(-posY)
	glDirX, glDirY, glDirZ := float32(light.GetDirX()), float32(light.GetDirZ()), float32(-light.GetDirY())
	//glDirX, glDirY, glDirZ := float32(light.GetDirX()), float32(light.GetDirY()), float32(light.GetDirZ())
	cutOff := float32(light.GetCutOff())
	outerCutOff := float32(light.GetOuterCutOff())
	intensity := float32(light.GetIntensityStyled(textures.GlobalTick()))
	falloff := float32(light.GetFalloff())
	lType := float32(-1)

	switch light.GetKind() {
	case config.LightKindOpenAir:
		// pos.Z = 100
		// lightType = 0
		return
	case config.LightKindAmbient:
		lType = 3
	case config.LightKindDirectional:
		lType = 2
	case config.LightKindSpot:
		lType = 1
		if f.useShadowLight {
			added := f.addShadowLight(
				float32(distSq),
				glPosX, glPosY, glPosZ,
				lType,
				r, g, b, intensity,
				glDirX, glDirY, glDirZ, falloff,
				cutOff, outerCutOff,
				float32(light.GetRadius()),
			)
			if added {
				return
			}
		}
	case config.LightKindNone:
		return
	default:
		lType = 0
	}

	f.add(
		glPosX, glPosY, glPosZ, lType,
		r, g, b, intensity,
		glDirX, glDirY, glDirZ, falloff,
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
	if idx+int(f.stride) > len(f.data2) {
		f.grow()
	}
	f.index += int(f.stride)

	f.data2[idx] = posX
	f.data2[idx+1] = posY
	f.data2[idx+2] = posZ
	f.data2[idx+3] = lightType
	f.data2[idx+4] = colR
	f.data2[idx+5] = colG
	f.data2[idx+6] = colB
	f.data2[idx+7] = intensity
	f.data2[idx+8] = dirX
	f.data2[idx+9] = dirY
	f.data2[idx+10] = dirZ
	f.data2[idx+11] = falloff
	f.data2[idx+12] = cutOff
	f.data2[idx+13] = outerCutOff
	f.data2[idx+14] = pad1
	f.data2[idx+15] = pad2
}

// addShadowLight adds a shadow-casting light to the list of frame lights based on its attributes and distance to the camera.
// Returns true if the light was successfully added, or false if it didn't qualify or was replaced in the min-heap.
func (f *FrameLights) addShadowLight(
	distSq float32,
	posX, posY, posZ float32,
	lightType float32,
	colR, colG, colB, intensity float32,
	dirX, dirY, dirZ, falloff float32,
	cutOff, outerCutOff float32,
	radius float32,
) bool {
	//TODO VERIFICA LE ombre generati non sono coerenti
	//return false

	//dx, dy, dz := posX-camX, posY-camY, posZ-camZ
	//distSq := dx*dx + dy*dy + dz*dz

	//target := radius
	target := radius * radius * 4
	if distSq > target {
		return false
	}

	score := intensity / (distSq + 1.0)

	// Fase 1: Riempimento iniziale (0-7)
	if f.shadowLightsIndex < ShadowLightNumber {
		light := f.shadowLights[f.shadowLightsIndex]
		//fmt.Println("VERIFY HOTSPOT", distSq, target)
		light.Assign(posX, posY, posZ, lightType, colR, colG, colB, intensity, dirX, dirY, dirZ, falloff, cutOff, outerCutOff, score)
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
	worst.Assign(posX, posY, posZ, lightType, colR, colG, colB, intensity, dirX, dirY, dirZ, falloff, cutOff, outerCutOff, score)
	// Ripristiniamo l'ordine dell'heap in O(log N)
	f.minHeapFixDown(0, ShadowLightNumber)
	return true
}

// grow dynamically expands the size of the underlying data slice when more capacity is needed.
func (f *FrameLights) grow() {
	newSize := len(f.data2) * 2
	if newSize == 0 {
		newSize = 128 * int(f.stride)
	}
	newData := make([]float32, newSize)
	copy(newData, f.data2)
	f.data2 = newData
	f.dataPtr = f.ctx.Ptr(f.data2)
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
