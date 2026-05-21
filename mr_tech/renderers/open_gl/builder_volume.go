package open_gl

import (
	"fmt"
	"sort"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/engine"
	"github.com/markel1974/godoom/mr_tech/model"
	"github.com/markel1974/godoom/mr_tech/physics"
	"github.com/markel1974/godoom/mr_tech/textures"
)

type VisibleVols struct {
	volumes          []*model.Volume
	index            int
	camX, camY, camZ float64
}

func NewVisibleVols(initSize int) *VisibleVols {
	return &VisibleVols{
		volumes: make([]*model.Volume, initSize),
		index:   0,
		camX:    0,
		camY:    0,
		camZ:    0,
	}
}

func (vs *VisibleVols) Reset(maxLen int, camX, camY, camZ float64) {
	vs.index = 0
	vs.camX = camX
	vs.camY = camY
	vs.camZ = camZ
	if maxLen >= len(vs.volumes) {
		vs.volumes = make([]*model.Volume, maxLen*2)
	}
}

func (vs *VisibleVols) At(index int) *model.Volume {
	return vs.volumes[index]
}

func (vs *VisibleVols) Add(volume *model.Volume) {
	vs.volumes[vs.index] = volume
	vs.index++
}

func (vs *VisibleVols) Sort() {
	sort.Sort(vs)
}

func (vs *VisibleVols) Len() int      { return vs.index }
func (vs *VisibleVols) Swap(i, j int) { vs.volumes[i], vs.volumes[j] = vs.volumes[j], vs.volumes[i] }
func (vs *VisibleVols) Less(i, j int) bool {
	// Sorting Front-to-Back (Distanza Quadra Pura)
	a := vs.volumes[i]
	b := vs.volumes[j]
	aX, aY, aZ := a.GetAABB().GetCentroid()
	bX, bY, bZ := b.GetAABB().GetCentroid()
	distA := (aX-vs.camX)*(aX-vs.camX) + (aY-vs.camY)*(aY-vs.camY) + (aZ-vs.camZ)*(aZ-vs.camZ)
	distB := (bX-vs.camX)*(bX-vs.camX) + (bY-vs.camY)*(bY-vs.camY) + (bZ-vs.camZ)*(bZ-vs.camZ)
	return distA < distB
}

type BuilderVolume struct {
	tex        *Textures
	fv         *FrameVertices
	dc         *DrawCommands
	fl         *FrameLights
	dcRender   *DrawCommandsRender
	cSky       *textures.Texture
	cal        *model.Calibration
	occBuffer  *OcclusionBuffer
	visibleVol *VisibleVols
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

	//w.pushQVolumesOcclusion(engine.GetVolumes(), frustumFront, fm, px, py, pz)
	//w.pushQVolumes(engine.GetVolumes(), frustumFront)
	w.pushQVolumesHardware(engine.GetVolumes(), frustumFront, px, py, pz)
	w.pushQLights(engine.GetLights(), frustumFront, frustumRear, px, py, pz)
	w.pushQThings(engine.GetThings(), frustumFront)

	w.dcRender.Prepare(w.dc.GetDrawCommands())
}

// pushQVolumesHardware processes and sorts visible volumes within the frustum, preparing vertex and draw command buffers.
func (w *BuilderVolume) pushQVolumesHardware(volumes *model.Volumes, frustumFront *physics.Frustum, pX, pY, pZ float64) {
	w.fv.DeepReset()
	w.dc.DeepReset()
	w.cSky = nil

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

	// 3. Ingestione Hardware (Early-Z friendly)
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
	w.fv.DeepReset()
	w.dc.DeepReset()
	w.cSky = nil

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

	// 4. Test di occlusione e ingestione facce
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
	// Ripristina VBO e Comandi allo stato congelato
	//w.fv.Reset()
	//w.dc.Reset()

	w.fv.DeepReset()
	w.dc.DeepReset()
	w.cSky = nil

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
func (w *BuilderVolume) pushQLights(lights *model.Lights, frustumFront, frustumRear *physics.Frustum, pX, pY, pZ float64) {
	w.fl.DeepReset()
	w.fl.Prepare(pX, pY, pZ)
	queryLights := func(object physics.IAABB) bool {
		light := object.(*model.Light)
		w.fl.Create(light)
		return false
	}
	lights.QueryMultiFrustum(frustumFront, frustumRear, queryLights)
}

// pushQLights processes lights within the provided frustum, filtering them and adding valid lights to the FrameLights instance.
func (w *BuilderVolume) pushQThings(things *model.Things, frustumFront *physics.Frustum) {
	q := func(object physics.IAABB) bool {
		thing := object.(model.IThing)
		faces2, faceCount, nextFaces2, _, lp, billBoard := thing.GetVertices()
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
		return false
	}

	things.QueryFrustum(frustumFront, q)

	//fmt.Println("TOTAL THINGS", count)
}
