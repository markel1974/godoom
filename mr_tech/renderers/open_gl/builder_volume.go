package open_gl

import (
	"fmt"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/engine"
	"github.com/markel1974/godoom/mr_tech/model"
	"github.com/markel1974/godoom/mr_tech/physics"
	"github.com/markel1974/godoom/mr_tech/textures"
)

type BuilderVolume struct {
	tex        *Textures
	fv         *FrameVertices
	dc         *DrawCommands
	fl         *FrameLights
	dcRender   *DrawCommandsRender
	cSky       *textures.Texture
	cal        *model.Calibration
	occBuffer  *OcclusionBuffer
	visibleVol *VisibleVolumes
}

func NewBuilderVolume(tex *Textures, calibration *model.Calibration) *BuilderVolume {
	bv := &BuilderVolume{
		tex:        tex,
		dcRender:   NewDrawCommandsRender(),
		fv:         NewFrameVertices(startBatchVertices),
		dc:         NewDrawCommands(startFrameCommands),
		fl:         NewFrameLights(256),
		occBuffer:  NewOcclusionBuffer(256, 144),
		cSky:       nil,
		cal:        calibration,
		visibleVol: NewVisibleVols(256),
	}
	return bv
}

func (w *BuilderVolume) GetShadowLights() ([8]*Light, int32) {
	return w.fl.GetShadowLights()
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
	px, py, pz := vi.GetView()
	angle, pitch, roll := vi.GetAngle(), vi.GetPitch(), vi.GetRoll()
	fm, fr := CreateFrontRearFrustum(float32(w.cal.AspectRatio), float32(w.cal.ZFarRoom), float32(px), float32(py), float32(pz), angle, pitch, roll)
	frustumFront, frustumRear := vi.GetFrustum(fm, fr)

	// Ripristina VBO e Comandi allo stato congelato
	//w.fv.Reset()
	//w.dc.Reset()

	w.fv.DeepReset()
	w.dc.DeepReset()
	w.cSky = nil

	//w.pushQVolumesOcclusion(engine.GetVolumes(), frustumFront, fm, px, py, pz)
	//w.pushQVolumes(engine.GetVolumes(), frustumFront)
	w.pushQVolumesHardware(engine.GetVolumes(), frustumFront, px, py, pz)
	w.pushQLights(engine.GetLights(), frustumFront, frustumRear, fm, px, py, pz)
	w.pushQThings(engine.GetThings(), frustumFront, fm)

	w.dcRender.Prepare(w.dc.GetDrawCommands())
}

// pushQVolumesHardware processes and sorts visible volumes within the frustum, preparing vertex and draw command buffers.
func (w *BuilderVolume) pushQVolumesHardware(volumes *model.Volumes, frustumFront *physics.Frustum, pX, pY, pZ float64) {
	//camX, camY, camZ := pX, pZ, -pY
	camX, camY, camZ := pX, pY, pZ

	w.visibleVol.Reset(volumes.Len(), camX, camY, camZ)

	// Raccolta dal DBVH (Broad-Phase)
	volumes.QueryFrustum(frustumFront, func(object physics.IAABB) bool {
		w.visibleVol.Add(object.(*model.Volume))
		return false
	})

	w.visibleVol.Sort()

	counter := 0

	// Ingestione Hardware (Early-Z friendly)
	for vIdx := 0; vIdx < w.visibleVol.Len(); vIdx++ {
		vol := w.visibleVol.At(vIdx)
		startIdx := w.fv.GetIndicesLen()
		faces, faceCount := vol.GetFaces()

		for x := 0; x < faceCount; x++ {
			face := (*faces)[x]
			tex, texKind := face.GetMaterialDetails()
			if tex == nil {
				continue
			}
			if texKind == int(config.MaterialKindSky) {
				w.cSky = tex
				continue
			}
			layer, hasLayer := w.tex.Get(tex)
			if !hasLayer {
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
		if startIdx != endIdx {
			w.dc.Compute(startIdx, endIdx)
			counter++
		}
	}
}

// pushQVolumes queries geometry volumes within the specified frustums, sorts them front-to-back,
// and culls occluded geometry using a CPU software occlusion buffer.
func (w *BuilderVolume) pushQVolumesOcclusion(volumes *model.Volumes, frustumFront *physics.Frustum, mvp [16]float32, pX, pY, pZ float64) {
	w.occBuffer.Clear()

	//camX, camY, camZ := pX, pZ, -pY
	camX, camY, camZ := pX, pY, pZ
	w.visibleVol.Reset(volumes.Len(), camX, camY, camZ)

	// Raccolta dal DBVH (Broad-Phase)
	volumes.QueryFrustum(frustumFront, func(object physics.IAABB) bool {
		w.visibleVol.Add(object.(*model.Volume))
		return false
	})

	w.visibleVol.Sort()

	counter := 0

	// Test di occlusione e ingestione facce
	for vIdx := 0; vIdx < w.visibleVol.Len(); vIdx++ {
		vol := w.visibleVol.At(vIdx)
		aabb := vol.GetAABB()

		// READ: Testiamo l'AABB dell'intero chunk contro il buffer
		if w.occBuffer.IsAABBOccluded(aabb, mvp) {
			continue
		}

		startIdx := w.fv.GetIndicesLen()
		faces, faceCount := vol.GetFaces()

		for fIdx := 0; fIdx < faceCount; fIdx++ {
			face := (*faces)[fIdx]
			tex, texKind := face.GetMaterialDetails()
			if tex == nil {
				continue
			}
			if texKind == int(config.MaterialKindSky) {
				w.cSky = tex
				continue
			}
			layer, hasLayer := w.tex.Get(tex)
			if !hasLayer {
				continue
			}
			p := face.GetPoints()
			u, v := face.GetUV()
			id0 := w.fv.AddVertex6(float32(p[0].X), float32(p[0].Z), float32(-p[0].Y), float32(u[0]), float32(-v[0]), layer)
			id1 := w.fv.AddVertex6(float32(p[1].X), float32(p[1].Z), float32(-p[1].Y), float32(u[1]), float32(-v[1]), layer)
			id2 := w.fv.AddVertex6(float32(p[2].X), float32(p[2].Z), float32(-p[2].Y), float32(u[2]), float32(-v[2]), layer)
			w.fv.AddTriangle(id0, id1, id2)
			w.occBuffer.RasterizeTriangle(p[0], p[1], p[2], mvp)
		}

		endIdx := w.fv.GetIndicesLen()
		if startIdx != endIdx {
			w.dc.Compute(startIdx, endIdx)
			counter++
		}
	}

	//fmt.Printf("FRUSTUM VOLUMES: %d, CULLED: %d, DRAW: %d\n", w.visibleVolsIndex, w.visibleVolsIndex-counter, counter)
}

// pushQVolumes queries geometry volumes within the specified frustums and processes them using the associated draw commands.
func (w *BuilderVolume) pushQVolumes(volumes *model.Volumes, frustumFront *physics.Frustum) {
	counter := 0

	queryGeom := func(object physics.IAABB) bool {
		vol := object.(*model.Volume)
		startIdx := w.fv.GetIndicesLen()
		faces, faceCount := vol.GetFaces()
		for x := 0; x < faceCount; x++ {
			face := (*faces)[x]
			tex, texKind := face.GetMaterialDetails()
			if tex == nil {
				continue
			}
			if texKind == int(config.MaterialKindSky) {
				w.cSky = tex
				continue
			}
			layer, hasLayer := w.tex.Get(tex)
			if !hasLayer {
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
		w.dc.Compute(startIdx, endIdx)
		counter++
		return false
	}

	volumes.QueryFrustum(frustumFront, queryGeom)

	fmt.Println("VOLUMES", volumes.Len(), "DRAW", counter)
}

// pushQLights processes lights within the provided frustum, filtering them and adding valid lights to the FrameLights instance.
func (w *BuilderVolume) pushQLights(lights *model.Lights, frustumFront, frustumRear *physics.Frustum, mvp [16]float32, pX, pY, pZ float64) {
	w.fl.DeepReset()
	w.fl.Prepare(pX, pY, pZ)
	counter := 0
	queryLights := func(object physics.IAABB) bool {
		light := object.(*model.Light)
		//if w.occBuffer.IsAABBOccluded(light.GetAABB(), mvp) {
		//	return false
		//}
		w.fl.Create(light)
		counter++
		return false
	}
	lights.QueryMultiFrustum(frustumFront, frustumRear, queryLights)

	//fmt.Println("LIGHTS", lights.Len(), "DRAW", counter)
}

// pushQLights processes lights within the provided frustum, filtering them and adding valid lights to the FrameLights instance.
func (w *BuilderVolume) pushQThings(things *model.Things, frustumFront *physics.Frustum, mvp [16]float32) {
	counter := 0
	q := func(object physics.IAABB) bool {
		thing := object.(model.IThing)
		//if w.occBuffer.IsAABBOccluded(thing.GetAABB(), mvp) {
		//	return false
		//}
		faces2, faceCount, nextFaces2, _, lp, billBoard := thing.GetVertices(textures.GlobalTick())
		if faceCount == 0 {
			return false
		}
		lerp := float32(lp)
		yaw := float32(thing.GetAngle())
		tPosX, tPosY, zBot := thing.GetDisplacement()
		oX, oY, oZ := float32(tPosX), float32(zBot), float32(-tPosY)
		b := float32(billBoard)
		startIndices := w.fv.GetIndicesLen()
		for fx := 0; fx < faceCount; fx++ {
			f := (*faces2)[fx]
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
			np := (*nextFaces2)[fx].GetPoints()
			id0 := w.fv.AddVertex15(float32(p[0].X), float32(p[0].Z), float32(-p[0].Y), float32(u[0]), float32(-v[0]), l, oX, oY, oZ, b, float32(np[0].X), float32(np[0].Z), float32(-np[0].Y), lerp, yaw)
			id1 := w.fv.AddVertex15(float32(p[1].X), float32(p[1].Z), float32(-p[1].Y), float32(u[1]), float32(-v[1]), l, oX, oY, oZ, b, float32(np[1].X), float32(np[1].Z), float32(-np[1].Y), lerp, yaw)
			id2 := w.fv.AddVertex15(float32(p[2].X), float32(p[2].Z), float32(-p[2].Y), float32(u[2]), float32(-v[2]), l, oX, oY, oZ, b, float32(np[2].X), float32(np[2].Z), float32(-np[2].Y), lerp, yaw)
			w.fv.AddTriangle(id0, id1, id2)
		}
		currentIndices := w.fv.GetIndicesLen()
		w.dc.Compute(startIndices, currentIndices)
		counter++
		return false
	}

	things.QueryFrustum(frustumFront, q)

	//fmt.Println("THINGS", things.Len(), "DRAW", counter)
}
