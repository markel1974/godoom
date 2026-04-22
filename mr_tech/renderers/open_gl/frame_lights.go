package open_gl

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/model"
)

// Light represents a light source with position, color, intensity, direction, and attenuation properties.
type Light struct {
	X, Y, Z                   float32
	Kind                      float32
	R, G, B, Intensity        float32
	DirX, DirY, DirZ, Falloff float32
	CutOff, OuterCutOff       float32
}

// FrameLights represents a structure used for managing and storing light data for rendering frameworks.
type FrameLights struct {
	data              []float32
	index             int
	freezeIndex       int
	stride            int32
	shadowLights      [8]*Light
	shadowLightsIndex int32
	camX              float32
	camY              float32
	camZ              float32
}

// NewFrameLights initializes and returns a new FrameLights instance with preallocated memory for light data and shadow lights.
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

// DeepReset fully resets the frame lights, including dynamic and shadow light indices, to prepare for new calculations.
func (f *FrameLights) DeepReset() {
	f.freezeIndex = 0
	f.shadowLightsIndex = 0
	f.Reset()
}

// Reset sets the current index of FrameLights to the freeze index, effectively resetting the state to the last saved point.
func (f *FrameLights) Reset() {
	f.index = f.freezeIndex
}

// Freeze captures the current state by storing the current index into the freezeIndex field.
func (f *FrameLights) Freeze() {
	f.freezeIndex = f.index
}

// LightsStride calculates and returns the stride of light data, scaled by 4 to represent the size in bytes per light entry.
func (f *FrameLights) LightsStride() int32 {
	return f.stride * 4
}

// GetLights retrieves the current light data as a slice of float32 and the total number of lights as an int32 value.
func (f *FrameLights) GetLights() ([]float32, int32) {
	return f.data[:f.index], int32(f.index) / f.stride
}

// GetLights retrieves the current light data as a slice of float32 and the total number of lights as an int32 value.
func (f *FrameLights) GetShadowLights() ([8]*Light, int32) {
	return f.shadowLights, f.shadowLightsIndex
}

// Prepare initializes the camera position and resets the shadow lights index for the FrameLights instance.
func (f *FrameLights) Prepare(camX, camY, camZ float32) {
	f.camX, f.camY, f.camZ = camX, camY, camZ
	f.shadowLightsIndex = 0
}

// Create processes a Light object and adds it to the FrameLights based on its type, position, and properties.
func (f *FrameLights) Create(light *model.Light) {
	r, g, b := float32(1.0), float32(1.0), float32(1.0)
	dirGlX, dirGlY, dirGlZ := float32(0.0), float32(0.0), float32(0.0)
	cutOff := float32(0)
	outerCutOff := float32(0)
	pos := light.GetPos()
	intensity := float32(light.GetIntensity())
	falloff := float32(light.GetFalloff())
	lightType := float32(-1)

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
		dirGlX, dirGlY, dirGlZ = float32(0.0), float32(-1.0), float32(0.0)
		spotDirX, SpotDirY, spotDirZ := float32(0.0), float32(-1.0), float32(0.0)
		r, g, b = float32(1.0), float32(1.0), float32(1.0)
		cutOff = float32(math.Cos(35.0 * math.Pi / 180.0))
		outerCutOff = float32(math.Cos(40 * math.Pi / 180.0))
		// FIX CUTOFF: Usiamo i valori reali del faretto
		// Convertiamo in radianti e poi in Coseno (come richiesto dallo shader)
		//const toRad = math.Pi / 180.0
		//cutOff2 := float32(math.Cos(baseCutoff * toRad))
		//outerCutOff2 := float32(math.Cos(baseOuterCutOff * toRad))
		added := f.addShadowLight(
			float32(pos.X), float32(pos.Z), float32(-pos.Y),
			lightType, r, g, b, intensity,
			spotDirX, SpotDirY, spotDirZ, falloff,
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
		float32(pos.X), float32(pos.Z), float32(-pos.Y), lightType,
		r, g, b, intensity,
		dirGlX, dirGlY, dirGlZ, falloff,
		cutOff, outerCutOff, 0.0, 0.0,
	)
}

// add updates the light data buffer with position, color, type, intensity, direction, and attenuation properties.
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

// addShadowLight adds a shadow-casting light to the shadowLights array with specified properties. Returns false if limit is reached.
func (f *FrameLights) addShadowLight(
	posX, posY, posZ, lightType float32,
	colR, colG, colB, intensity float32,
	dirX, dirY, dirZ, falloff float32,
	cutOff, outerCutOff float32,
) bool {
	//return false
	if f.shadowLightsIndex >= int32(len(f.shadowLights)) {
		//fmt.Println("Shadow light limit reached")
		return false
	}
	light := f.shadowLights[f.shadowLightsIndex]
	f.shadowLightsIndex++
	light.Intensity = intensity
	light.Kind = lightType
	light.X, light.Y, light.Z = posX, posY, posZ
	light.R, light.G, light.B = colR, colG, colB
	light.DirX, light.DirY, light.DirZ = dirX, dirY, dirZ
	light.Falloff = falloff
	light.CutOff, light.OuterCutOff = cutOff, outerCutOff
	return true
}

// grow dynamically increases the size of the internal data slice to accommodate new elements when capacity is exceeded.
func (f *FrameLights) grow() {
	newSize := len(f.data) * 2
	if newSize == 0 {
		newSize = 128 * int(f.stride)
	}
	newData := make([]float32, newSize)
	copy(newData, f.data)
	f.data = newData
}

/*
type ScoredLight struct {
	Light Light
	Score float32
}

func (w *FrameLights) UpdateDynamicShadows(camX, camY, camZ float32) [][16]float32 {
	var candidates []ScoredLight
	for _, l := range w.lights {
		// Filtriamo solo i faretti (Spot Lights)
		if l.Kind != float32(config.LightKindSpot) {
			continue
		}
		// Calcolo distanza al quadrato (risparmiamo la radice)
		dx, dy, dz := l.X-camX, l.Y-camY, l.Z-camZ
		distSq := dx*dx + dy*dy + dz*dz
		// Heuristic: Scarta a priori le luci la cui distanza supera di molto il loro falloff
		if distSq > (l.Falloff * l.Falloff * 4.0) {
			continue
		}
		// Calcolo dello Score: più è vicina e luminosa, più alto è il punteggio
		intensity := l.Intensity
		score := intensity / (distSq + 1.0) // +1.0 evita divisioni per zero
		candidates = append(candidates, ScoredLight{Light: l, Score: score})
	}

	// Sorting decrescente per Score
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Score > candidates[j].Score
	})

	// Prendiamo solo i primi 8 (o meno)
	limit := 8
	if len(candidates) < limit {
		limit = len(candidates)
	}
	return candidates
}

/*

		var dynaMatrices [][16]float32
		for i := 0; i < limit; i++ {
			l := candidates[i].Light
			pos := l.GetPos()
			dir := l.GetDir() // Assumendo che ritorni il vettore direzionale

			// Mappatura coordinate OpenGL (X, Z, -Y)
			pX, pY, pZ := float32(pos.X), float32(pos.Z), float32(-pos.Y)
			dX, dY, dZ := float32(dir.X), float32(dir.Z), float32(-dir.Y)

			// FOV a 90° per coprire interamente un tipico outer-cutoff di 40-45°
			//mat := w.metrics.CreateSpotLightSpaceMatrix(pX, pY, pZ, dX, dY, dZ, 90.0, 1.0, float32(l.GetFalloff()))
			dynaMatrices = append(dynaMatrices, mat)
		}


	return dynaMatrices
}


*/
