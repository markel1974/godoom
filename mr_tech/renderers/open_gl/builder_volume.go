package open_gl

import (
	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/engine"
	"github.com/markel1974/godoom/mr_tech/model"
	"github.com/markel1974/godoom/mr_tech/physics"
	"github.com/markel1974/godoom/mr_tech/textures"
)

type VolumeRange struct {
	start int32
	end   int32
}

type BuilderVolume struct {
	tex         *Textures
	fv          *FrameVertices
	dc          *DrawCommands
	fl          *FrameLights
	dcRender    *DrawCommandsRender
	mapBuilt    bool
	cSky        *textures.Texture
	volRanges   map[*model.Volume]VolumeRange // CACHE DI CULLING
	calibration *model.Calibration
}

func NewBuilderVolume(tex *Textures, calibration *model.Calibration) *BuilderVolume {
	bv := &BuilderVolume{
		tex:         tex,
		dcRender:    NewDrawCommandsRender(),
		fv:          NewFrameVertices(startBatchVertices),
		dc:          NewDrawCommands(startFrameCommands),
		fl:          NewFrameLights(256),
		volRanges:   make(map[*model.Volume]VolumeRange), // Inizializzazione
		mapBuilt:    false,
		cSky:        nil,
		calibration: calibration,
	}
	return bv
}

// GetVerticesStride returns the byte stride of the vertex data as an int32 by delegating to the underlying FrameVertices.
func (w *BuilderVolume) GetVerticesStride() int32 { return w.fv.VerticesStride() }

// GetLightsStride returns the stride value for light data, representing the size in bytes of a single light entry.
func (w *BuilderVolume) GetLightsStride() int32 { return w.fl.LightsStride() }

// GetDrawCommands returns the prepared draw commands for rendering stored in the DrawCommandsRender instance.
func (w *BuilderVolume) GetDrawCommands() *DrawCommandsRender { return w.dcRender }

// GetVertices retrieves the vertex buffer, vertex count, index buffer, and index count from the BuilderVolume instance.
func (w *BuilderVolume) GetVertices() ([]float32, int32, []uint32, int32) { return w.fv.GetVertices() }

// GetLights retrieves the current set of light data and the number of lights in the frame as a slice and an integer.
func (w *BuilderVolume) GetLights() ([]float32, int32) { return w.fl.GetLights() }

// GetSkyTexture retrieves the current sky texture associated with the BuilderVolume. Returns nil if no texture is set.
func (w *BuilderVolume) GetSkyTexture() *textures.Texture { return w.cSky }

func (w *BuilderVolume) Compute(fbw, fbh int32, vi *model.ViewMatrix, engine *engine.Engine) {
	// Ripristina VBO e Comandi allo stato congelato
	w.fv.Reset()
	w.dc.Reset()
	// Reset TOTALE del buffer luci ogni frame (le calcoliamo dinamicamente)
	w.fl.DeepReset()
	if !w.mapBuilt {
		w.fv.DeepReset()
		w.dc.DeepReset()
		w.cSky = nil
		volumes := engine.GetVolumes()
		for _, vol := range volumes.GetVolumes() {
			startIdx := w.fv.GetIndicesLen()
			for _, face := range vol.GetFaces() {
				tex, texKind := face.GetMaterialDetails()
				if tex == nil {
					continue
				}
				if texKind == int(config.AnimationKindSky) {
					w.cSky = tex
					continue
				}
				layer, ok := w.tex.Get(tex)
				if !ok {
					continue
				}
				p := face.GetPoints()
				u, v := face.GetUV()
				id1 := w.fv.AddVertex(float32(p[0].X), float32(p[0].Z), float32(-p[0].Y), float32(u[0]), float32(v[0]), layer, 0, 0, 0, 0)
				id2 := w.fv.AddVertex(float32(p[1].X), float32(p[1].Z), float32(-p[1].Y), float32(u[1]), float32(v[1]), layer, 0, 0, 0, 0)
				id3 := w.fv.AddVertex(float32(p[2].X), float32(p[2].Z), float32(-p[2].Y), float32(u[2]), float32(v[2]), layer, 0, 0, 0, 0)
				w.fv.AddTriangle(id1, id2, id3)
			}
			endIdx := w.fv.GetIndicesLen()
			if endIdx > startIdx {
				w.volRanges[vol] = VolumeRange{start: startIdx, end: endIdx}
			}
		}
		// Congeliamo SOLO Geometria e DrawCommands
		w.fv.Freeze()
		w.dc.Freeze()
		w.mapBuilt = true
	}

	const usFrustum = false

	if usFrustum {
		queryGeom := func(object physics.IAABB) bool {
			if vol, ok := object.(*model.Volume); ok {
				if vr, exists := w.volRanges[vol]; exists {
					w.dc.Compute(vr.start, vr.end)
				}
			}
			return false
		}
		frustumFront, frustumRear := vi.GetFrustum(fbw, fbh, w.calibration.ZFarRoom)
		engine.QueryMultiFrustum(frustumFront, frustumRear, queryGeom)
		w.pushLights(w.fl, engine.GetLights(), frustumFront, frustumRear)
	} else {
		for _, vr := range w.volRanges {
			w.dc.Compute(vr.start, vr.end)
		}
		for _, vl := range engine.GetLights().Get() {
			w.fl.Create(vl)
		}
	}
	// 4. Entità Dinamiche
	tA, tC := engine.GetThings().GetActive()
	w.pushThings(w.fv, w.dc, vi, tA, tC)
	w.dcRender.Prepare(w.dc.GetDrawCommands())
}

// pushThings processes and adds the given list of things to the frame vertices and draw commands for rendering.
func (w *BuilderVolume) pushThings(fv *FrameVertices, dc *DrawCommands, vi *model.ViewMatrix, things []model.IThing, thingsCount int) {
	if len(things) == 0 {
		return
	}
	for idx := 0; idx < thingsCount; idx++ {
		thing := things[idx]
		faces, billboard := thing.GetVertices()
		if faces == nil {
			continue
		}

		tPosX, tPosY, zBot := thing.GetPosition()
		oX, oY, oZ := float32(tPosX), float32(zBot), float32(-tPosY)
		b := float32(billboard)

		startIndices := fv.GetIndicesLen()
		for _, f := range faces {
			mat := f.GetMaterial()
			if mat == nil {
				continue
			}
			l, ok := w.tex.Get(mat)
			if !ok {
				continue
			}
			p := f.GetPoints()
			u, v := f.GetUV()

			id0 := w.fv.AddVertex(float32(p[0].X), float32(p[0].Z), float32(-p[0].Y), float32(u[0]), float32(v[0]), l, oX, oY, oZ, b)
			id1 := w.fv.AddVertex(float32(p[1].X), float32(p[1].Z), float32(-p[1].Y), float32(u[1]), float32(v[1]), l, oX, oY, oZ, b)
			id2 := w.fv.AddVertex(float32(p[2].X), float32(p[2].Z), float32(-p[2].Y), float32(u[2]), float32(v[2]), l, oX, oY, oZ, b)
			fv.AddTriangle(id0, id1, id2)
		}
		currentIndices := fv.GetIndicesLen()
		dc.Compute(startIndices, currentIndices)
	}
}

// pushLights processes lights within the provided frustum, filtering them and adding valid lights to the FrameLights instance.
func (w *BuilderVolume) pushLights(fl *FrameLights, lights *model.Lights, frustumFront, frustumRear *physics.Frustum) {
	queryLights := func(object physics.IAABB) bool {
		if l, ok := object.(*model.Light); ok {
			fl.Create(l)
		}
		return false
	}
	lights.QueryMultiFrustum(frustumFront, frustumRear, queryLights)
}
