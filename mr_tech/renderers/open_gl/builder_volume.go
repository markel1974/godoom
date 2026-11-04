package open_gl

import (
	"math"
	"unsafe"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/engine"
	"github.com/markel1974/godoom/mr_tech/model"
	"github.com/markel1974/godoom/mr_tech/model/lights"
	"github.com/markel1974/godoom/mr_tech/physics/aabb"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/metrics"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/shaders"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// BuilderVolume represents a structure that consolidates textures, frame vertices, draw commands, and other rendering resources.
type BuilderVolume struct {
	ctx               api.IContext
	tex               *Textures
	fv                *FrameVertices
	dcOpaque          *DrawCommands
	dcAdditive        *DrawCommands
	dcLiquid          *DrawCommands
	fl                *FrameLights
	dcRenderOpaque    *shaders.DrawCommandsRender
	dcRenderAdditive  *shaders.DrawCommandsRender
	dcRenderLiquid    *shaders.DrawCommandsRender
	cSky              *textures.Material
	cal               *model.Calibration
	visibleVol        *VisibleVolumes
	modes             []*DrawCommands
	occlusion         *OcclusionHW
	frustumFront      *metrics.Frustum
	frustumLightFront *metrics.Frustum
	frustumLightRear  *metrics.Frustum
	lightFrontFrustum *aabb.Frustum
	lightRearFrustum  *aabb.Frustum
}

// NewBuilderVolume initializes and returns a new BuilderVolume instance with configured textures and calibration settings.
func NewBuilderVolume(ctx api.IContext, tex *Textures, cal *model.Calibration) *BuilderVolume {
	bv := &BuilderVolume{
		ctx:               ctx,
		tex:               tex,
		fv:                NewFrameVertices(ctx, 1048576),
		dcOpaque:          NewDrawCommands(config.BlendModeOpaque, 32768),
		dcAdditive:        NewDrawCommands(config.BlendModeAdditive, 4096),
		dcLiquid:          NewDrawCommands(config.BlendModeLiquid, 4096),
		fl:                NewFrameLights(ctx, 1024, cal.EnableShadowLights),
		dcRenderOpaque:    shaders.NewDrawCommandsRender(ctx, "opaque", false),
		dcRenderAdditive:  shaders.NewDrawCommandsRender(ctx, "additive", true),
		dcRenderLiquid:    shaders.NewDrawCommandsRender(ctx, "liquid", true),
		cSky:              nil,
		cal:               cal,
		visibleVol:        NewVisibleVols(8192),
		modes:             make([]*DrawCommands, config.BlendModeLatest),
		occlusion:         NewOcclusionHW(ctx, 4096),
		frustumFront:      metrics.NewFrustum(),
		frustumLightFront: metrics.NewFrustum(),
		frustumLightRear:  metrics.NewFrustum(),
		lightFrontFrustum: aabb.NewFrustum(),
		lightRearFrustum:  aabb.NewFrustum(),
	}
	prjScale := bv.cal.PrjScale
	bv.frustumFront.Setup(float32(bv.cal.AspectRatio), float32(bv.cal.ZFar), float32(bv.cal.ZNear), prjScale, prjScale)

	prjScaleLights := bv.cal.PrjScaleLights
	bv.frustumLightFront.Setup(1.0, float32(bv.cal.ZLightFarFront), float32(bv.cal.ZLightNear), prjScaleLights, prjScaleLights)
	bv.frustumLightRear.Setup(1.0, float32(bv.cal.ZLightFarRear), float32(bv.cal.ZLightNear), prjScaleLights, prjScaleLights)

	bv.modes[bv.dcOpaque.GetBlendMode()] = bv.dcOpaque
	bv.modes[bv.dcAdditive.GetBlendMode()] = bv.dcAdditive
	bv.modes[bv.dcLiquid.GetBlendMode()] = bv.dcLiquid
	return bv
}

// GetShadowLights retrieves up to 8 shadow-casting lights and their count from the current frame lighting data.
func (w *BuilderVolume) GetShadowLights() ([8]*metrics.Light, int32) {
	return w.fl.GetShadowLights()
}

// GetVerticesStride returns the byte stride of the vertex data by delegating the computation to the underlying FrameVertices object.
func (w *BuilderVolume) GetVerticesStride() int32 { return w.fv.VerticesStride() }

// GetLightsStride returns the stride size of the lights buffer in bytes.
func (w *BuilderVolume) GetLightsStride() int32 { return w.fl.LightsStride() }

// GetDrawCommandsOpaque returns the prepared DrawCommandsRender object containing batched GPU drawing commands for rendering.
func (w *BuilderVolume) GetDrawCommandsOpaque() *shaders.DrawCommandsRender { return w.dcRenderOpaque }

// GetVertices retrieves the vertex buffer, vertex count, index buffer, and index count from the builder volume.
func (w *BuilderVolume) GetVertices() (unsafe.Pointer, int, int32, unsafe.Pointer, int, int32) {
	return w.fv.GetVertices()
}

// GetLights retrieves the light data and their count from the frame lights. It returns a slice of float32 and an int32 count.
func (w *BuilderVolume) GetLights() (unsafe.Pointer, int32, int32) { return w.fl.GetLights() }

// GetSkyMaterial returns the sky material associated with the BuilderVolume.
func (w *BuilderVolume) GetSkyMaterial() *textures.Material { return w.cSky }

// Compute processes volumes, lights, and things within the given frustum and prepares rendering draw commands.
func (w *BuilderVolume) Compute(fbw, fbh int32, vi *model.ViewMatrix, engine *engine.Engine) {
	px, py, pz := vi.GetView()
	angle, pitch, roll := vi.GetAngle(), -vi.GetPitch(), vi.GetRoll()
	rAngle, rPitch, rRoll := angle+math.Pi, -pitch, -roll
	w.frustumFront.Update(float32(px), float32(py), float32(pz), angle, pitch, roll)

	w.frustumLightFront.Update(float32(px), float32(py), float32(pz), angle, pitch, roll)
	w.frustumLightRear.Update(float32(px), float32(py), float32(pz), rAngle, rPitch, rRoll)

	frustumFront := vi.GetFrontFrustum(w.frustumFront.Get())
	frustumLightFront := w.lightFrontFrustum.Rebuild(w.frustumLightFront.Get())
	//frustumLightRear := w.lightRearFrustum.Rebuild(w.frustumLightRear.Get())

	// Ripristina VBO e Comandi allo stato congelato
	//w.fv.Reset()
	//w.dc.Reset()

	w.fv.DeepReset()
	w.dcOpaque.DeepReset()
	w.dcAdditive.DeepReset()
	w.dcLiquid.DeepReset()
	w.cSky = nil

	w.occlusion.Reset()

	//w.pushQVolumes(engine.GetVolumes(), frustumFront)
	w.pushQVolumes(engine.GetVolumes(), frustumFront, px, py, pz)
	w.pushLights(engine.GetLights(), frustumLightFront, px, py, pz)
	//w.pushQLights(engine.GetLights(), frustumLightFront, frustumLightRear, px, py, pz)
	w.pushQThings(engine.GetThings(), frustumFront)

	w.dcRenderOpaque.Prepare(w.dcOpaque.GetDrawCommands())
	w.dcRenderAdditive.Prepare(w.dcAdditive.GetDrawCommands())
	w.dcRenderLiquid.Prepare(w.dcLiquid.GetDrawCommands())
}

// pushQVolumesHardware processes visible volumes, sorts them, and generates draw commands based on their material properties.
// It extracts geometry from the provided volumes within a frustum and applies texture and blending mode filters for rendering.
func (w *BuilderVolume) pushQVolumes(volumes *model.Volumes, frustumFront *aabb.Frustum, camX, camY, camZ float64) {
	//w.occBuffer.Clear()

	w.visibleVol.Reset(volumes.Len(), camX, camY, camZ)

	// Raccolta dal DBVH (Broad-Phase)
	volumes.QueryFrustum(frustumFront, func(object aabb.IObject) bool {
		w.visibleVol.Add(object.(*model.Volume))
		return false
	})

	//GPU CULLING HELPER
	w.visibleVol.Sort()

	//counter := 0

	pushPass := func() {
		for vIdx := 0; vIdx < w.visibleVol.Len(); vIdx++ {
			vol := w.visibleVol.At(vIdx)
			marker := vol.GetMarker()
			if marker <= 0 {
				marker = w.fv.CreateMarker()
				vol.SetMarker(marker)
			}
			fv := w.fv.Stage(marker)
			if !vol.IsDirty() {
				w.fv.Commit(fv, w.modes)
				if skyMat := fv.GetMaterial(0); skyMat != nil {
					w.cSky = skyMat
				}
				continue
			}

			fv.DeepReset()
			hasAnim := false
			faces, faceCount := vol.GetFaces()
			startIdx := fv.GetIndicesLen()
			//added := false
			//var added int
			for x := 0; x < faceCount; x++ {
				face := (*faces)[x]
				mat := face.GetMaterial()
				if mat == nil || mat.GetKind() == int(config.MaterialKindNone) {
					continue
				}
				if mat.IsAnimated() {
					hasAnim = true
				}
				tex := mat.GetCurrentFrame()
				if tex == nil {
					continue
				}
				tId := tex.GetIdentifier()
				if tId < 0 {
					tId, _ = w.tex.Get(tex)
					tex.SetIdentifier(tId)
				}
				if mat.GetKind() == int(config.MaterialKindSky) {
					w.cSky = mat // Nebbia volumetrica atmosferica
					fv.SetMaterial(0, mat)
					continue
				}
				faceBlendT := float32(0.0)
				if config.MaterialKind(mat.GetKind()).ToMaterialFamily() == config.MaterialFamilyLiquid {
					faceBlendT = 0.5
				}
				material := mat.GetPacked()
				spec := float32(mat.GetSpecularBoost())
				shininess := float32(mat.GetShininess())
				detailScale := mat.GetDetailScale()
				detailFactor := mat.GetDetailFactor()
				p0X, p0Y, p0Z := face.GetPoint0()
				p1X, p1Y, p1Z := face.GetPoint1()
				p2X, p2Y, p2Z := face.GetPoint2()
				u0, v0 := face.GetUV0()
				u1, v1 := face.GetUV1()
				u2, v2 := face.GetUV2()
				id0 := fv.AddVertex14(float32(p0X), float32(p0Z), float32(-p0Y), float32(u0), float32(-v0), tId, 0, 0, 0, faceBlendT, material, spec, shininess, detailScale, detailFactor)
				id1 := fv.AddVertex14(float32(p1X), float32(p1Z), float32(-p1Y), float32(u1), float32(-v1), tId, 0, 0, 0, faceBlendT, material, spec, shininess, detailScale, detailFactor)
				id2 := fv.AddVertex14(float32(p2X), float32(p2Z), float32(-p2Y), float32(u2), float32(-v2), tId, 0, 0, 0, faceBlendT, material, spec, shininess, detailScale, detailFactor)
				fv.AddTriangle(id0, id1, id2)
				//w.occBuffer.RasterizeTriangle(p[0], p[1], p[2], mvp)
				endIdx := fv.GetIndicesLen()
				if startIdx != endIdx {
					blendMode := mat.GetBlendMode()
					fv.AddBatch(blendMode, endIdx-startIdx, mat)
					startIdx = endIdx
					//added = true
				}
			}

			w.fv.Commit(fv, w.modes)
			if !hasAnim {
				vol.ClearDirty()
			}
			//if added {
			//	counter++
			//}
		}
	}

	pushPass()

	//fmt.Println("Volumes:", volumes.Len(), counter, volumes.Len()-counter)
}

// pushQThings processes and prepares "things" objects for rendering by querying them against the frustum and applying transformations.
func (w *BuilderVolume) pushQThings(things *model.Things, frustumFront *aabb.Frustum) {
	//counter := 0
	pushPassThing := func(thing model.IThing, pFaces *[]*model.Face, faceCount int, pNextFaces *[]*model.Face, _ int, lp float64, renderMode float64) {
		if !thing.IsVisible() {
			return
		}

		lerp := float32(lp)
		yaw := float32(thing.GetAngle())
		tPosX, tPosY, zBot := thing.GetVerticesCenter()
		oX, oY, oZ := float32(tPosX), float32(zBot), float32(-tPosY)
		faces := *pFaces
		nextFaces := *pNextFaces

		var currentMat *textures.Material
		var currentTargetDc *DrawCommands
		batchStart := w.fv.GetIndicesLen()

		for fx := 0; fx < faceCount; fx++ {
			face := faces[fx]
			mat := face.GetMaterial()
			if mat == nil || mat.GetKind() == int(config.MaterialKindNone) {
				continue
			}
			tex := mat.GetCurrentFrame()
			if tex == nil {
				continue
			}
			tId := tex.GetIdentifier()
			if tId < 0 {
				tId, _ = w.tex.Get(tex)
				tex.SetIdentifier(tId)
			}

			if mat != currentMat {
				currentIndices := w.fv.GetIndicesLen()
				if currentIndices > batchStart && currentTargetDc != nil {
					currentTargetDc.Compute(batchStart, currentIndices, currentMat)
				}
				batchStart = currentIndices
				currentMat = mat
				currentTargetDc = w.modes[mat.GetBlendMode()]
			}

			material := mat.GetPacked()
			spec := float32(mat.GetSpecularBoost())
			shininess := float32(mat.GetShininess())
			detailScale := mat.GetDetailScale()
			detailFactor := mat.GetDetailFactor()
			p0X, p0Y, p0Z := face.GetPoint0()
			p1X, p1Y, p1Z := face.GetPoint1()
			p2X, p2Y, p2Z := face.GetPoint2()
			u0, v0 := face.GetUV0()
			u1, v1 := face.GetUV1()
			u2, v2 := face.GetUV2()
			nf := nextFaces[fx]
			np0X, np0Y, np0Z := nf.GetPoint0()
			np1X, np1Y, np1Z := nf.GetPoint1()
			np2X, np2Y, np2Z := nf.GetPoint2()

			id0 := w.fv.AddVertex20(float32(p0X), float32(p0Z), float32(-p0Y), float32(u0), float32(-v0), tId, oX, oY, oZ, float32(renderMode), float32(np0X), float32(np0Z), float32(-np0Y), lerp, yaw, material, spec, shininess, detailScale, detailFactor)
			id1 := w.fv.AddVertex20(float32(p1X), float32(p1Z), float32(-p1Y), float32(u1), float32(-v1), tId, oX, oY, oZ, float32(renderMode), float32(np1X), float32(np1Z), float32(-np1Y), lerp, yaw, material, spec, shininess, detailScale, detailFactor)
			id2 := w.fv.AddVertex20(float32(p2X), float32(p2Z), float32(-p2Y), float32(u2), float32(-v2), tId, oX, oY, oZ, float32(renderMode), float32(np2X), float32(np2Z), float32(-np2Y), lerp, yaw, material, spec, shininess, detailScale, detailFactor)
			w.fv.AddTriangle(id0, id1, id2)
		}

		currentIndices := w.fv.GetIndicesLen()
		if currentIndices > batchStart && currentTargetDc != nil {
			currentTargetDc.Compute(batchStart, currentIndices, currentMat)
		}
	}

	qOcclusion := func(object aabb.IObject) bool {
		thing := object.(model.IThing)
		occState := w.occlusion.GetState(thing.GetEntity().GetId())
		if occState != nil && !occState.IsVisible {
			w.occlusion.Add(thing.GetAABB(), occState) // Schedulalo per controllarlo al prossimo frame
			return false                               // CULLATO! Non generiamo i vertici
		}
		w.occlusion.Add(thing.GetAABB(), occState) // Lo vediamo, aggiungiamolo ai test GPU
		pFaces, faceCount, pNextFaces, nFaceCount, lp, renderMode := thing.GetVertices(textures.GlobalTick())
		if faceCount == 0 {
			// Anche se non ha facce, teniamo vivo occlusion test
			return false
		}

		pushPassThing(thing, pFaces, faceCount, pNextFaces, nFaceCount, lp, renderMode)
		//counter++
		return false
	}

	qStatic := func(object aabb.IObject) bool {
		thing := object.(model.IThing)
		pFaces, faceCount, pNextFaces, nFaceCount, lp, renderMode := thing.GetVertices(textures.GlobalTick())
		if faceCount > 0 {
			pushPassThing(thing, pFaces, faceCount, pNextFaces, nFaceCount, lp, renderMode)
		}
		return false
	}

	things.QueryStatic(qStatic) //UI Objects
	things.QueryFrustum(frustumFront, qOcclusion)

	//fmt.Println("THINGS", things.Len(), "DRAW", counter)
}

// pushQLights processes a collection of lights within the specified front and rear frustums and updates the frame lighting.
// It resets the frame lights state, prepares lighting at the given coordinates, and queries lights intersecting the frustums.
func (w *BuilderVolume) pushLights(ls *lights.Lights, frustumFront *aabb.Frustum, pX, pY, pZ float64) {
	//TODO IMPLEMENTAZIONE CON FRUSTUM
	w.fl.DeepReset()
	w.fl.Prepare()
	counter := 0

	for _, light := range ls.Get() {
		if light.GetKind() == config.LightKindNone {
			continue
		}
		posX, posY, posZ := light.GetPosXYZ()
		dx := posX - pX
		dy := posY - pY
		dz := posZ - pZ
		distSq := dx*dx + dy*dy + dz*dz
		b := light.GetAABB()
		isFront := b.IntersectFrustum(frustumFront)
		w.fl.Create(light, distSq, isFront)
		counter++
	}
	//_, _, dynamicCount := w.fl.GetLights()
	//_, shadowCount := w.fl.GetShadowLights()
	//fmt.Printf("LIGHTS %d (DRAW %d -> DYNAMIC %d, SHADOW %d)\n", ls.Len(), counter, dynamicCount, shadowCount)
}

// pushQLights processes a collection of lights within the specified front and rear frustums and updates the frame lighting.
// It resets the frame lights state, prepares lighting at the given coordinates, and queries lights intersecting the frustums.
func (w *BuilderVolume) pushQLights(ls *lights.Lights, frustumFront, frustumRear *aabb.Frustum, pX, pY, pZ float64) {
	w.fl.DeepReset()
	w.fl.Prepare()
	counter := 0

	queryLights := func(object aabb.IObject, isFront bool, isBack bool) bool {
		light := object.(*lights.Light)
		if light.GetKind() == config.LightKindNone {
			return false
		}
		posX, posY, posZ := light.GetPosXYZ()
		dx := posX - pX
		dy := posY - pY
		dz := posZ - pZ
		distSq := dx*dx + dy*dy + dz*dz
		w.fl.Create(light, distSq, isFront)
		counter++
		return false
	}
	ls.QueryMultiFrustum(frustumFront, frustumRear, queryLights)
}

// GetDrawCommandsAdditive returns the additive draw commands prepared for rendering.
func (w *BuilderVolume) GetDrawCommandsAdditive() *shaders.DrawCommandsRender {
	return w.dcRenderAdditive
}

// GetDrawCommandsLiquid returns the liquid draw commands prepared for rendering.
func (w *BuilderVolume) GetDrawCommandsLiquid() *shaders.DrawCommandsRender {
	return w.dcRenderLiquid
}

// GetHWOcclusion retrieves the hardware-based occlusion object associated with the BuilderVolume.
func (w *BuilderVolume) GetHWOcclusion() *OcclusionHW {
	return w.occlusion
}
