package open_gl

import (
	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/engine"
	"github.com/markel1974/godoom/mr_tech/model"
	"github.com/markel1974/godoom/mr_tech/physics"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// BuilderVolume represents a structure that consolidates textures, frame vertices, draw commands, and other rendering resources.
type BuilderVolume struct {
	ctx              api.IContext
	tex              *Textures
	fv               *FrameVertices
	dcOpaque         *DrawCommands
	dcAdditive       *DrawCommands
	dcLiquid         *DrawCommands
	fl               *FrameLights
	dcRenderOpaque   *DrawCommandsRender
	dcRenderAdditive *DrawCommandsRender
	dcRenderLiquid   *DrawCommandsRender
	cSky             *textures.Texture
	cSkyU, cSkyV     float64
	cal              *model.Calibration
	visibleVol       *VisibleVolumes
	modes            []*DrawCommands
	//occlusion        *OcclusionHW
}

// NewBuilderVolume initializes and returns a new BuilderVolume instance with configured textures and calibration settings.
func NewBuilderVolume(ctx api.IContext, tex *Textures, calibration *model.Calibration) *BuilderVolume {
	bv := &BuilderVolume{
		ctx:              ctx,
		tex:              tex,
		fv:               NewFrameVertices(1048576),
		dcOpaque:         NewDrawCommands(config.BlendModeOpaque, 32768),
		dcAdditive:       NewDrawCommands(config.BlendModeAdditive, 4096),
		dcLiquid:         NewDrawCommands(config.BlendModeLiquid, 4096),
		fl:               NewFrameLights(1024),
		dcRenderOpaque:   NewDrawCommandsRender(ctx, false),
		dcRenderAdditive: NewDrawCommandsRender(ctx, true),
		dcRenderLiquid:   NewDrawCommandsRender(ctx, true),
		cSky:             nil,
		cSkyU:            0.0,
		cSkyV:            0.0,
		cal:              calibration,
		visibleVol:       NewVisibleVols(8192),
		modes:            make([]*DrawCommands, config.BlendModeLates),
		//occlusion:        NewOcclusionHW(ctx, 4096),
		//occBuffer:      NewOcclusionSW(640, 480),
	}
	bv.modes[bv.dcOpaque.GetBlendMode()] = bv.dcOpaque
	bv.modes[bv.dcAdditive.GetBlendMode()] = bv.dcAdditive
	bv.modes[bv.dcLiquid.GetBlendMode()] = bv.dcLiquid
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
func (w *BuilderVolume) GetDrawCommands() *DrawCommandsRender { return w.dcRenderOpaque }

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
	prjScale := 0.001

	fm, fr := CreateFrontRearFrustum(float32(w.cal.AspectRatio), float32(w.cal.ZFarRoom), float32(px), float32(py), float32(pz), angle, pitch, roll, prjScale, prjScale)
	frustumFront, frustumRear := vi.GetFrustum(fm, fr)

	// Ripristina VBO e Comandi allo stato congelato
	//w.fv.Reset()
	//w.dc.Reset()

	w.fv.DeepReset()
	w.dcOpaque.DeepReset()
	w.dcAdditive.DeepReset()
	w.dcLiquid.DeepReset()
	w.cSky = nil

	//if w.occlusion != nil {
	//	w.occlusion.Reset()
	//}

	//w.pushQVolumes(engine.GetVolumes(), frustumFront)
	w.pushQVolumes(engine.GetVolumes(), frustumFront, fm, px, py, pz)
	w.pushQLights(engine.GetLights(), frustumFront, frustumRear, fm, px, py, pz)
	w.pushQThings(engine.GetThings(), frustumFront, fm)

	w.dcRenderOpaque.Prepare(w.dcOpaque.GetDrawCommands())
	w.dcRenderAdditive.Prepare(w.dcAdditive.GetDrawCommands())
	w.dcRenderLiquid.Prepare(w.dcLiquid.GetDrawCommands())
}

// pushQVolumesHardware processes visible volumes, sorts them, and generates draw commands based on their material properties.
// It extracts geometry from the provided volumes within a frustum and applies texture and blending mode filters for rendering.
func (w *BuilderVolume) pushQVolumes(volumes *model.Volumes, frustumFront *physics.Frustum, mvp [16]float32, camX, camY, camZ float64) {
	//w.occBuffer.Clear()

	w.visibleVol.Reset(volumes.Len(), camX, camY, camZ)

	// Raccolta dal DBVH (Broad-Phase)
	volumes.QueryFrustum(frustumFront, func(object physics.IAABB) bool {
		w.visibleVol.Add(object.(*model.Volume))
		return false
	})

	w.visibleVol.Sort()

	counter := 0

	pushPass := func() {
		for vIdx := 0; vIdx < w.visibleVol.Len(); vIdx++ {
			vol := w.visibleVol.At(vIdx)
			faces, faceCount := vol.GetFaces()

			var added int
			for x := 0; x < faceCount; x++ {
				face := (*faces)[x]
				matObj := face.GetMaterialObj()
				mat, texKind := face.GetMaterialDetails()
				if mat == nil || matObj == nil {
					continue
				}
				tId := float32(0)
				if tId = mat.GetIdentifier(); tId < 0 {
					tId, _ = w.tex.Get(mat)
					mat.SetIdentifier(tId)
				}
				faceBlendT := float32(0.0)
				switch texKind {
				case int(config.MaterialKindSky):
					w.cSky, w.cSkyU, w.cSkyV = mat, matObj.U(), matObj.V()
					continue
				case int(config.MaterialKindLiquid):
					faceBlendT = 0.5
				}
				startIdx := w.fv.GetIndicesLen()
				p := face.GetPoints()
				u, v := face.GetUV()
				id0 := w.fv.AddVertex10(float32(p[0].X), float32(p[0].Z), float32(-p[0].Y), float32(u[0]), float32(-v[0]), tId, 0, 0, 0, faceBlendT)
				id1 := w.fv.AddVertex10(float32(p[1].X), float32(p[1].Z), float32(-p[1].Y), float32(u[1]), float32(-v[1]), tId, 0, 0, 0, faceBlendT)
				id2 := w.fv.AddVertex10(float32(p[2].X), float32(p[2].Z), float32(-p[2].Y), float32(u[2]), float32(-v[2]), tId, 0, 0, 0, faceBlendT)
				w.fv.AddTriangle(id0, id1, id2)
				//w.occBuffer.RasterizeTriangle(p[0], p[1], p[2], mvp)
				added++

				endIdx := w.fv.GetIndicesLen()
				if startIdx != endIdx {
					blendMode := matObj.BlendMode()
					targetDc := w.modes[blendMode]
					targetDc.Compute(startIdx, endIdx, matObj)
					startIdx = endIdx
				}
			}

			if added > 0 {
				counter++
			}
		}
	}

	pushPass()
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

		/*
			occState := w.occlusion.GetState(thing.GetEntity().GetId())
			if occState != nil && !occState.IsVisible {
				w.occlusion.Add(thing.GetAABB(), occState) // Schedulalo per controllarlo al prossimo frame
				return false                               // CULLATO! Non generiamo i vertici
			}

		*/

		pFaces, faceCount, pNextFaces, _, lp, renderMode := thing.GetVertices(textures.GlobalTick())
		/*
			if faceCount == 0 {
				w.occlusion.Add(thing.GetAABB(), occState) // Anche se non ha facce, teniamo vivo il test
				return false
			}
			w.occlusion.Add(thing.GetAABB(), occState) // Lo vediamo, aggiungiamolo ai test GPU


		*/
		if faceCount == 0 {
			return false
		}

		lerp := float32(lp)
		yaw := float32(thing.GetAngle())
		tPosX, tPosY, zBot := thing.GetDisplacement()
		oX, oY, oZ := float32(tPosX), float32(zBot), float32(-tPosY)

		pushPassThing := func() {
			faces := *pFaces
			nextFaces := *pNextFaces

			for fx := 0; fx < faceCount; fx++ {
				face := faces[fx]
				mat := face.GetMaterial()
				matObj := face.GetMaterialObj()
				if mat == nil || matObj == nil {
					continue
				}
				tId := float32(0)
				if tId = mat.GetIdentifier(); tId < 0 {
					tId, _ = w.tex.Get(mat)
					mat.SetIdentifier(tId)
				}
				p := face.GetPoints()
				u, v := face.GetUV()
				nf := nextFaces[fx]
				np := nf.GetPoints()
				startIndices := w.fv.GetIndicesLen()
				id0 := w.fv.AddVertex15(float32(p[0].X), float32(p[0].Z), float32(-p[0].Y), float32(u[0]), float32(-v[0]), tId, oX, oY, oZ, float32(renderMode), float32(np[0].X), float32(np[0].Z), float32(-np[0].Y), lerp, yaw)
				id1 := w.fv.AddVertex15(float32(p[1].X), float32(p[1].Z), float32(-p[1].Y), float32(u[1]), float32(-v[1]), tId, oX, oY, oZ, float32(renderMode), float32(np[1].X), float32(np[1].Z), float32(-np[1].Y), lerp, yaw)
				id2 := w.fv.AddVertex15(float32(p[2].X), float32(p[2].Z), float32(-p[2].Y), float32(u[2]), float32(-v[2]), tId, oX, oY, oZ, float32(renderMode), float32(np[2].X), float32(np[2].Z), float32(-np[2].Y), lerp, yaw)
				w.fv.AddTriangle(id0, id1, id2)

				currentIndices := w.fv.GetIndicesLen()
				if startIndices != currentIndices {
					blendMode := matObj.BlendMode()
					targetDc := w.modes[blendMode]
					targetDc.Compute(startIndices, currentIndices, matObj)
					startIndices = currentIndices
				}
			}
		}

		pushPassThing()
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

// GetDrawCommandsLiquid returns the liquid draw commands prepared for rendering.
func (w *BuilderVolume) GetDrawCommandsLiquid() *DrawCommandsRender {
	return w.dcRenderLiquid
}

// GetHWOcclusion retrieves the hardware-based occlusion object associated with the BuilderVolume.
func (w *BuilderVolume) GetHWOcclusion() *OcclusionHW {
	return nil
	//return w.occlusion
}

/*
// pushQVolumes processes and renders visible volumes intersecting the given frustum, applying material and texture filtering.

	func (w *BuilderVolume) pushQVolumes(volumes *model.Volumes, frustumFront *physics.Frustum) {
		counter := 0

		queryGeom := func(object physics.IAABB) bool {
			vol := object.(*model.Volume)
			faces, faceCount := vol.GetFaces()
			for x := 0; x < faceCount; x++ {
				face := (*faces)[x]
				mat, texKind := face.GetMaterialDetails()
				matObj := face.GetMaterialObj()
				if mat == nil || matObj == nil {
					continue
				}
				if texKind == int(config.MaterialKindSky) {
					w.cSky = mat
					w.cSkyU = matObj.U()
					w.cSkyV = matObj.V()
					continue
				}
				tId := float32(0)
				if tId = mat.GetIdentifier(); tId < 0 {
					tId, _ = w.tex.Get(mat)
					mat.SetIdentifier(tId)
				}

				startIdx := w.fv.GetIndicesLen()

				p := face.GetPoints()
				u, v := face.GetUV()
				renderMode := float32(0.0)
				if texKind == int(config.MaterialKindLiquid) {
					renderMode = 0.5
				}
				id0 := w.fv.AddVertex10(float32(p[0].X), float32(p[0].Z), float32(-p[0].Y), float32(u[0]), float32(-v[0]), tId, 0, 0, 0, renderMode)
				id1 := w.fv.AddVertex10(float32(p[1].X), float32(p[1].Z), float32(-p[1].Y), float32(u[1]), float32(-v[1]), tId, 0, 0, 0, renderMode)
				id2 := w.fv.AddVertex10(float32(p[2].X), float32(p[2].Z), float32(-p[2].Y), float32(u[2]), float32(-v[2]), tId, 0, 0, 0, renderMode)
				w.fv.AddTriangle(id0, id1, id2)

				endIdx := w.fv.GetIndicesLen()
				if startIdx != endIdx {
					blendMode := matObj.BlendMode()
					targetDc := w.modes[blendMode]
					targetDc.Compute(startIdx, endIdx, matObj)
					startIdx = endIdx
				}
			}
			counter++
			return false
		}

		volumes.QueryFrustum(frustumFront, queryGeom)

		//fmt.Println("VOLUMES", volumes.Len(), "DRAW", counter)
	}
*/
