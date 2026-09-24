package open_gl

import (
	"fmt"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/engine"
	"github.com/markel1974/godoom/mr_tech/model"
	"github.com/markel1974/godoom/mr_tech/physics"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// BuilderVolume represents a structure that consolidates textures, frame vertices, draw commands, and other rendering resources.
type BuilderVolume struct {
	tex              *Textures
	fv               *FrameVertices
	dc               *DrawCommands
	dcAdditive       *DrawCommands
	fl               *FrameLights
	dcRender         *DrawCommandsRender
	dcRenderAdditive *DrawCommandsRender
	cSky             *textures.Texture
	cSkyU, cSkyV     float64
	cal              *model.Calibration
	occBuffer        *OcclusionBuffer
	visibleVol       *VisibleVolumes
}

// NewBuilderVolume initializes and returns a new BuilderVolume instance with configured textures and calibration settings.
func NewBuilderVolume(tex *Textures, calibration *model.Calibration) *BuilderVolume {
	bv := &BuilderVolume{
		tex:              tex,
		fv:               NewFrameVertices(1048576),
		dc:               NewDrawCommands(32768),
		dcAdditive:       NewDrawCommands(4096),
		fl:               NewFrameLights(1024),
		dcRender:         NewDrawCommandsRender(),
		dcRenderAdditive: NewDrawCommandsRender(),
		cSky:             nil,
		cSkyU:            0.0,
		cSkyV:            0.0,
		cal:              calibration,
		occBuffer:        NewOcclusionBuffer(640, 480),
		visibleVol:       NewVisibleVols(8192),
	}
	return bv
}

// GetShadowLights retrieves up to 8 shadow-casting lights and their count from the current frame lighting data.
func (w *BuilderVolume) GetShadowLights() ([8]*Light, int32) {
	return w.fl.GetShadowLights()
}

// GetVerticesStride returns the byte stride of the vertex data by delegating the computation to the underlying FrameVertices object.
func (w *BuilderVolume) GetVerticesStride() int32 { return w.fv.VerticesStride() }

// GetLightsStride returns the stride size of the lights buffer in bytes.
func (w *BuilderVolume) GetLightsStride() int32 { return w.fl.LightsStride() }

// GetDrawCommands returns the prepared DrawCommandsRender object containing batched GPU drawing commands for rendering.
func (w *BuilderVolume) GetDrawCommands() *DrawCommandsRender { return w.dcRender }

// GetVertices retrieves the vertex buffer, vertex count, index buffer, and index count from the builder volume.
func (w *BuilderVolume) GetVertices() ([]float32, int32, []uint32, int32) { return w.fv.GetVertices() }

// GetLights retrieves the light data and their count from the frame lights. It returns a slice of float32 and an int32 count.
func (w *BuilderVolume) GetLights() ([]float32, int32) { return w.fl.GetLights() }

// GetSkyTexture retrieves the current sky texture associated with the BuilderVolume. Returns nil if no texture is set.
func (w *BuilderVolume) GetSkyTexture() *textures.Texture { return w.cSky }

// GetSkyUV retrieves the horizontal and vertical scroll offsets for the sky texture.
func (w *BuilderVolume) GetSkyUV() (float64, float64) { return w.cSkyU, w.cSkyV }

// Compute processes volumes, lights, and things within the given frustum and prepares rendering draw commands.
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
	w.dcAdditive.DeepReset()
	w.cSky = nil

	//w.pushQVolumesOcclusion(engine.GetVolumes(), frustumFront, fm, px, py, pz)
	//w.pushQVolumes(engine.GetVolumes(), frustumFront)
	w.pushQVolumesHardware(engine.GetVolumes(), frustumFront, px, py, pz)
	w.pushQLights(engine.GetLights(), frustumFront, frustumRear, fm, px, py, pz)
	w.pushQThings(engine.GetThings(), frustumFront, fm)

	w.dcRender.Prepare(w.dc.GetDrawCommands())
	w.dcRenderAdditive.Prepare(w.dcAdditive.GetDrawCommands())
}

// pushQVolumesHardware processes visible volumes, sorts them, and generates draw commands based on their material properties.
// It extracts geometry from the provided volumes within a frustum and applies texture and blending mode filters for rendering.
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

	pushPass := func(targetBlendMode int, targetDc *DrawCommands) {
		for vIdx := 0; vIdx < w.visibleVol.Len(); vIdx++ {
			vol := w.visibleVol.At(vIdx)
			startIdx := w.fv.GetIndicesLen()
			faces, faceCount := vol.GetFaces()

			var added int
			for x := 0; x < faceCount; x++ {
				face := (*faces)[x]

				matObj := face.GetMaterialObj()
				if matObj != nil && matObj.BlendMode() != targetBlendMode {
					continue
				}

				tex, texKind := face.GetMaterialDetails()
				if tex == nil {
					continue
				}
				if texKind == int(config.MaterialKindSky) {
					w.cSky = tex
					w.cSkyU = matObj.U()
					w.cSkyV = matObj.V()
					//fmt.Println("sky ", w.cSkyU, w.cSkyV)
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
				added++
			}

			endIdx := w.fv.GetIndicesLen()
			if added > 0 && startIdx != endIdx {
				targetDc.Compute(startIdx, endIdx)
				counter++
			}
		}
	}

	pushPass(int(config.BlendModeOpaque), w.dc)
	pushPass(int(config.BlendModeAdditive), w.dcAdditive)
}

// pushQVolumesOcclusion applies frustum culling and occlusion testing on volumes and populates draw buffers with visible geometry.
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
				if matObj := face.GetMaterialObj(); matObj != nil {
					w.cSkyU = matObj.U()
					w.cSkyV = matObj.V()
				}
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
			w.dcAdditive.Compute(startIdx, endIdx)
			counter++
		}
	}

	//fmt.Printf("FRUSTUM VOLUMES: %d, CULLED: %d, DRAW: %d\n", w.visibleVolsIndex, w.visibleVolsIndex-counter, counter)
}

// pushQVolumes processes and renders visible volumes intersecting the given frustum, applying material and texture filtering.
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
				if matObj := face.GetMaterialObj(); matObj != nil {
					w.cSkyU = matObj.U()
					w.cSkyV = matObj.V()
				}
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
		w.dcAdditive.Compute(startIdx, endIdx)
		counter++
		return false
	}

	volumes.QueryFrustum(frustumFront, queryGeom)

	fmt.Println("VOLUMES", volumes.Len(), "DRAW", counter)
}

// pushQLights processes a collection of lights within the specified front and rear frustums and updates the frame lighting.
// It resets the frame lights state, prepares lighting at the given coordinates, and queries lights intersecting the frustums.
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

// pushQThings processes and prepares "things" objects for rendering by querying them against the frustum and applying transformations.
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
		pushPassThing := func(targetBlendMode int, targetDc *DrawCommands) {
			startIndices := w.fv.GetIndicesLen()
			var added int
			for fx := 0; fx < faceCount; fx++ {
				f := (*faces2)[fx]
				mat := f.GetMaterial()
				if mat == nil {
					continue
				}
				blendMode := config.BlendModeOpaque
				if mat.IsEmissive() {
					blendMode = config.BlendModeAdditive
				}
				if blendMode != targetBlendMode {
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
				added++
			}
			currentIndices := w.fv.GetIndicesLen()
			if added > 0 && startIndices != currentIndices {
				targetDc.Compute(startIndices, currentIndices)
			}
		}

		pushPassThing(int(config.BlendModeOpaque), w.dc)
		pushPassThing(int(config.BlendModeAdditive), w.dcAdditive)
		counter++
		return false
	}

	things.QueryFrustum(frustumFront, q)

	//fmt.Println("THINGS", things.Len(), "DRAW", counter)
}

// GetDrawCommandsAdditive returns the additive draw commands prepared for rendering.
func (w *BuilderVolume) GetDrawCommandsAdditive() *DrawCommandsRender {
	return w.dcRenderAdditive
}
