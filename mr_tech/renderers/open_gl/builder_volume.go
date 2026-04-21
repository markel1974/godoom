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
	tex       *Textures
	fv        *FrameVertices
	dc        *DrawCommands
	fl        *FrameLights
	dcRender  *DrawCommandsRender
	mapBuilt  bool
	cSky      *textures.Texture
	volRanges map[*model.Volume]VolumeRange // CACHE DI CULLING
	cal       *model.Calibration
}

func NewBuilderVolume(tex *Textures, calibration *model.Calibration) *BuilderVolume {
	bv := &BuilderVolume{
		tex:       tex,
		dcRender:  NewDrawCommandsRender(),
		fv:        NewFrameVertices(startBatchVertices),
		dc:        NewDrawCommands(startFrameCommands),
		fl:        NewFrameLights(256),
		volRanges: make(map[*model.Volume]VolumeRange), // Inizializzazione
		mapBuilt:  false,
		cSky:      nil,
		cal:       calibration,
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
				id0 := w.fv.AddVertex6(float32(p[0].X), float32(p[0].Z), float32(-p[0].Y), float32(u[0]), float32(-v[0]), layer)
				id1 := w.fv.AddVertex6(float32(p[1].X), float32(p[1].Z), float32(-p[1].Y), float32(u[1]), float32(-v[1]), layer)
				id2 := w.fv.AddVertex6(float32(p[2].X), float32(p[2].Z), float32(-p[2].Y), float32(u[2]), float32(-v[2]), layer)
				w.fv.AddTriangle(id0, id1, id2)
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

	//const usFrustum = true

	//if usFrustum {
	px, py, pz := vi.GetXYZ()
	angle, pitch, roll := vi.GetAngle(), vi.GetPitch(), vi.GetRoll()
	fm, fr := CreateFrontRearFrustum(float32(fbw), float32(fbh), float32(w.cal.ZFarRoom), float32(px), float32(py), float32(pz), angle, pitch, roll)
	frustumFront, frustumRear := vi.GetFrustum(fm, fr)
	w.pushQVolumes(engine.GetVolumes(), frustumFront, frustumRear)
	w.pushQLights(engine.GetLights(), frustumFront, frustumRear)
	w.pushQThings(engine.GetThings(), frustumFront, frustumRear)
	//} else {
	//	for _, vr := range w.volRanges {
	//		w.dc.Compute(vr.start, vr.end)
	//	}
	//	for _, vl := range engine.GetLights().Get() {
	//		w.fl.Create(vl)
	//	}
	//	tA, tC := engine.GetThings().GetActive()
	//	for idx := 0; idx < tC; idx++ {
	//		w.pushThing(tA[idx], w.fv, w.dc)
	//	}
	//}
	w.dcRender.Prepare(w.dc.GetDrawCommands())
}

// pushQVolumes queries geometry volumes within the specified frustums and processes them using the associated draw commands.
func (w *BuilderVolume) pushQVolumes(volumes *model.Volumes, frustumFront, frustumRear *physics.Frustum) {
	queryGeom := func(object physics.IAABB) bool {
		if vol, ok := object.(*model.Volume); ok {
			if vr, exists := w.volRanges[vol]; exists {
				w.dc.Compute(vr.start, vr.end)
			}
		}
		return false
	}
	volumes.QueryFrustum(frustumFront, queryGeom)
}

// pushQLights processes lights within the provided frustum, filtering them and adding valid lights to the FrameLights instance.
func (w *BuilderVolume) pushQLights(lights *model.Lights, frustumFront, frustumRear *physics.Frustum) {
	queryLights := func(object physics.IAABB) bool {
		if l, ok := object.(*model.Light); ok {
			w.fl.Create(l)
		}
		return false
	}
	lights.QueryMultiFrustum(frustumFront, frustumRear, queryLights)
}

// pushQLights processes lights within the provided frustum, filtering them and adding valid lights to the FrameLights instance.
func (w *BuilderVolume) pushQThings(things *model.Things, frustumFront, frustumRear *physics.Frustum) {
	q := func(object physics.IAABB) bool {
		if l, ok := object.(model.IThing); ok {
			w.pushThing(l, w.fv, w.dc)
		}
		return false
	}
	things.QueryFrustum(frustumFront, q)
}

// pushThing processes a single IThing instance, extracts its vertex and material data, and appends it to frame vertices and draw commands.
func (w *BuilderVolume) pushThing(thing model.IThing, fv *FrameVertices, dc *DrawCommands) {
	faces, nextFaces, lp, billBoard := thing.GetVertices()
	if faces == nil {
		return
	}
	lerp := float32(lp)
	yaw := float32(thing.GetAngle())
	tPosX, tPosY, zBot := thing.GetPosition()
	oX, oY, oZ := float32(tPosX), float32(zBot), float32(-tPosY)
	b := float32(billBoard)
	startIndices := fv.GetIndicesLen()
	for fx, f := range faces {
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
		np := nextFaces[fx].GetPoints()
		id0 := w.fv.AddVertex15(float32(p[0].X), float32(p[0].Z), float32(-p[0].Y), float32(u[0]), float32(-v[0]), l, oX, oY, oZ, b, float32(np[0].X), float32(np[0].Z), float32(-np[0].Y), lerp, yaw)
		id1 := w.fv.AddVertex15(float32(p[1].X), float32(p[1].Z), float32(-p[1].Y), float32(u[1]), float32(-v[1]), l, oX, oY, oZ, b, float32(np[1].X), float32(np[1].Z), float32(-np[1].Y), lerp, yaw)
		id2 := w.fv.AddVertex15(float32(p[2].X), float32(p[2].Z), float32(-p[2].Y), float32(u[2]), float32(-v[2]), l, oX, oY, oZ, b, float32(np[2].X), float32(np[2].Z), float32(-np[2].Y), lerp, yaw)
		fv.AddTriangle(id0, id1, id2)
	}
	currentIndices := fv.GetIndicesLen()
	dc.Compute(startIndices, currentIndices)
}
