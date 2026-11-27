package builder

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

type spotLightCandidate struct {
	id          uint64
	prevSlot    int
	distSq      float32
	stabScore   float32
	rawScore    float32
	posX        float32
	posY        float32
	posZ        float32
	lightType   float32
	colR        float32
	colG        float32
	colB        float32
	intensity   float32
	dirX        float32
	dirY        float32
	dirZ        float32
	falloff     float32
	cutOff      float32
	outerCutOff float32
}

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
	prevShadowIDs     [ShadowLightNumber]uint64
	prevShadowCount   int
	spotCandidates    []spotLightCandidate
	shadowFinalized   bool
	//glCamX            float32
	//glCamY            float32
	//glCamZ            float32
	useShadowLight    bool
	suppressDirForCSM bool
	hasSceneDir       bool
	sceneDirGL        [3]float32
	sceneColorInt     [4]float32
	debug             bool
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
		spotCandidates:    make([]spotLightCandidate, 0, 64),
		stride:            stride,
		useShadowLight:    useShadowLight,
	}
	if ctx != nil {
		fl.dataPtr = ctx.Ptr(fl.data2)
	}
	for idx := range fl.shadowLights {
		fl.shadowLights[idx] = metrics.NewLight()
	}
	return fl
}

// DeepReset resets the freezeIndex, shadowLightsIndex, and calls the Reset method to perform a full reset of the FrameLights.
func (f *FrameLights) DeepReset() {
	f.freezeIndex = 0
	f.shadowLightsIndex = 0
	f.shadowFinalized = false
	f.spotCandidates = f.spotCandidates[:0]
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
	if !f.shadowFinalized {
		f.Finalize()
	}
	size := int32(f.index * 4)
	count := int32(f.index) / f.stride
	return f.dataPtr, size, count
}

// GetShadowLights retrieves the shadow-casting lights and their count from the current frame's lighting configuration.
func (f *FrameLights) GetShadowLights() ([ShadowLightNumber]*metrics.Light, int32) {
	if !f.shadowFinalized {
		f.Finalize()
	}
	return f.shadowLights, f.shadowLightsIndex
}

// SetSuppressDirectionalForCSM controls whether LightKindDirectional entries are omitted from the
// unshadowed clustered UBO because the outdoor SunCSM pass already evaluates shadowed directional sun lighting.
func (f *FrameLights) SetSuppressDirectionalForCSM(suppress bool) {
	f.suppressDirForCSM = suppress
}

// GetSceneDirectional returns the latest global directional light collected in the frame, if any.
func (f *FrameLights) GetSceneDirectional() (dirGL [3]float32, colorInt [4]float32, ok bool) {
	return f.sceneDirGL, f.sceneColorInt, f.hasSceneDir
}

// HasSceneDir reports whether a scene directional light has been collected for the current frame.
func (f *FrameLights) HasSceneDir() bool {
	return f.hasSceneDir
}

// GetSceneDir returns the current scene directional light direction as x, y, and z components.
func (f *FrameLights) GetSceneDir() (float32, float32, float32) {
	ptr := &f.sceneDirGL
	return ptr[0], ptr[1], ptr[2]
}

// GetSceneColorInt returns the scene directional light color components and intensity.
func (f *FrameLights) GetSceneColorInt() (float32, float32, float32, float32) {
	ptr := &f.sceneColorInt
	return ptr[0], ptr[1], ptr[2], ptr[3]
}

// Prepare initializes camera position for lighting calculations and resets the shadow lights index to zero.
func (f *FrameLights) Prepare() {
	//TODO VERIFICA
	//f.glCamX, f.glCamY, f.glCamZ = float32(pX), float32(pY), float32(pZ)
	//f.glCamX, f.glCamY, f.glCamZ = float32(pX), float32(pZ), float32(-pY)
	f.shadowLightsIndex = 0
	f.shadowFinalized = false
	f.spotCandidates = f.spotCandidates[:0]
	f.hasSceneDir = false
}

// Create adds a new light to the FrameLights based on its type, position, intensity, and other properties.
func (f *FrameLights) Create(light *lights.Light, distSq float64, isFront bool) {
	intensity := float32(light.GetIntensityStyled(textures.GlobalTick()))
	if intensity <= 0.001 {
		return
	}
	r, g, b := light.GetRGB()
	posX, posY, posZ := light.GetPosXYZ()
	dirX, dirY, dirZ := light.GetDirXYZ()
	glPosX, glPosY, glPosZ := float32(posX), float32(posZ), float32(-posY)
	glDirX, glDirY, glDirZ := float32(dirX), float32(dirZ), float32(-dirY)
	//glDirX, glDirY, glDirZ := float32(light.GetDirX()), float32(light.GetDirY()), float32(light.GetDirZ())
	cutOff := float32(light.GetCutOff())
	outerCutOff := float32(light.GetOuterCutOff())
	//intensity := float32(light.GetIntensityStyled(textures.GlobalTick()))
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
		if !f.hasSceneDir {
			f.hasSceneDir = true
			f.sceneDirGL = [3]float32{glDirX, glDirY, glDirZ}
			f.sceneColorInt = [4]float32{float32(r), float32(g), float32(b), intensity}
		}
		if f.suppressDirForCSM {
			return
		}
		lType = 2
	case config.LightKindSpot:
		lType = 1
		if f.useShadowLight {
			if f.queueShadowLight(
				light.GetId(),
				float32(distSq),
				isFront,
				glPosX, glPosY, glPosZ,
				lType,
				float32(r), float32(g), float32(b), intensity,
				glDirX, glDirY, glDirZ, falloff,
				cutOff, outerCutOff,
				float32(light.GetRadius()),
				float32(light.GetIntensity()),
			) {
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
		float32(r), float32(g), float32(b), intensity,
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

func (f *FrameLights) prevSlotOf(id uint64) int {
	for i := 0; i < f.prevShadowCount; i++ {
		if f.prevShadowIDs[i] == id {
			return i
		}
	}
	return -1
}

// queueShadowLight evaluates eligibility with near-camera invariance and range hysteresis,
// queuing qualifying spotlights for top-N selection at the end of the frame.
func (f *FrameLights) queueShadowLight(
	id uint64,
	distSq float32,
	isFront bool,
	posX, posY, posZ float32,
	lightType float32,
	colR, colG, colB, intensity float32,
	dirX, dirY, dirZ, falloff float32,
	cutOff, outerCutOff float32,
	radius float32,
	unStyledIntensity float32,
) bool {
	prevSlot := f.prevSlotOf(id)

	attInt := unStyledIntensity
	if intensity > attInt {
		attInt = intensity
	}
	if attInt < 0.1 {
		attInt = 0.1
	}
	attFall := falloff
	if attFall < 1.0 {
		attFall = 1.0
	}
	effRadius := float32(4.605) * attFall * attInt

	enterRange := effRadius * 3.0
	if radius*3.0 > enterRange {
		enterRange = radius * 3.0
	}
	if enterRange < 2048.0 {
		enterRange = 2048.0
	}
	if prevSlot >= 0 {
		enterRange *= 1.25
	}
	maxDistSq := enterRange * enterRange

	nearRadius := effRadius + 300.0
	if prevSlot >= 0 {
		nearRadius = effRadius + 420.0
	}
	nearDistSq := nearRadius * nearRadius

	if distSq > maxDistSq || (!isFront && distSq > nearDistSq) {
		return false
	}

	kneeSq := effRadius*effRadius*0.25 + 1.0
	rawScore := (attInt * effRadius) / (distSq + kneeSq)
	stabScore := rawScore
	if prevSlot >= 0 {
		// Strong incumbent hysteresis (~4x score / ~2x distance) keeps active shadowed spotlights
		// anchored in their atlas slots as the player moves and turns.
		stabScore *= 4.0
	} else if !isFront {
		stabScore *= 0.5
	}

	f.spotCandidates = append(f.spotCandidates, spotLightCandidate{
		id:          id,
		prevSlot:    prevSlot,
		distSq:      distSq,
		stabScore:   stabScore,
		rawScore:    rawScore,
		posX:        posX,
		posY:        posY,
		posZ:        posZ,
		lightType:   lightType,
		colR:        colR,
		colG:        colG,
		colB:        colB,
		intensity:   intensity,
		dirX:        dirX,
		dirY:        dirY,
		dirZ:        dirZ,
		falloff:     falloff,
		cutOff:      cutOff,
		outerCutOff: outerCutOff,
	})
	return true
}

// Finalize resolves top-N spotlight shadow candidates with hysteresis, preserves previous
// shadow slot indices where possible, and routes non-winning spotlights to the unshadowed buffer.
func (f *FrameLights) Finalize() {
	if f.shadowFinalized {
		return
	}
	f.shadowFinalized = true

	n := len(f.spotCandidates)
	if n == 0 {
		f.shadowLightsIndex = 0
		f.prevShadowCount = 0
		clear(f.prevShadowIDs[:])
		return
	}

	winnerCount := n
	if winnerCount > ShadowLightNumber {
		winnerCount = ShadowLightNumber
	}

	for i := 0; i < winnerCount; i++ {
		best := i
		for j := i + 1; j < n; j++ {
			if candidateBetter(f.spotCandidates[j], f.spotCandidates[best]) {
				best = j
			}
		}
		if best != i {
			f.spotCandidates[i], f.spotCandidates[best] = f.spotCandidates[best], f.spotCandidates[i]
		}
	}

	for i := winnerCount; i < n; i++ {
		c := f.spotCandidates[i]
		f.add(
			c.posX, c.posY, c.posZ, c.lightType,
			c.colR, c.colG, c.colB, c.intensity,
			c.dirX, c.dirY, c.dirZ, c.falloff,
			c.cutOff, c.outerCutOff, 0.0, 0.0,
		)
	}

	var slotUsed [ShadowLightNumber]bool
	var assigned [ShadowLightNumber]int
	for i := 0; i < winnerCount; i++ {
		assigned[i] = -1
		slot := f.spotCandidates[i].prevSlot
		if slot >= 0 && slot < winnerCount && !slotUsed[slot] {
			slotUsed[slot] = true
			assigned[i] = slot
		}
	}

	nextFree := 0
	for i := 0; i < winnerCount; i++ {
		if assigned[i] >= 0 {
			continue
		}
		for slotUsed[nextFree] {
			nextFree++
		}
		slotUsed[nextFree] = true
		assigned[i] = nextFree
	}

	f.shadowLightsIndex = int32(winnerCount)
	f.prevShadowCount = winnerCount
	clear(f.prevShadowIDs[:])
	for i := 0; i < winnerCount; i++ {
		c := f.spotCandidates[i]
		slot := assigned[i]
		f.shadowLights[slot].Assign(
			c.posX, c.posY, c.posZ, c.lightType,
			c.colR, c.colG, c.colB, c.intensity,
			c.dirX, c.dirY, c.dirZ, c.falloff,
			c.cutOff, c.outerCutOff, c.rawScore,
		)
		f.prevShadowIDs[slot] = c.id
	}
}

func candidateBetter(a, b spotLightCandidate) bool {
	if a.stabScore != b.stabScore {
		return a.stabScore > b.stabScore
	}
	aPrev := a.prevSlot >= 0
	bPrev := b.prevSlot >= 0
	if aPrev != bPrev {
		return aPrev
	}
	if aPrev && bPrev && a.prevSlot != b.prevSlot {
		return a.prevSlot < b.prevSlot
	}
	return a.id < b.id
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
	if f.ctx != nil {
		f.dataPtr = f.ctx.Ptr(f.data2)
	}
}
