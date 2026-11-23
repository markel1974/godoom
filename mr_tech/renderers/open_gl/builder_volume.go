package open_gl

import (
	"unsafe"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/engine"
	"github.com/markel1974/godoom/mr_tech/model"
	"github.com/markel1974/godoom/mr_tech/model/ithing"
	"github.com/markel1974/godoom/mr_tech/model/lights"
	"github.com/markel1974/godoom/mr_tech/model/things"
	"github.com/markel1974/godoom/mr_tech/model/vertices"
	"github.com/markel1974/godoom/mr_tech/model/volumes"
	"github.com/markel1974/godoom/mr_tech/physics/aabb"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/metrics"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/shaders"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// BuilderVolume represents a structure that consolidates textures, frame vertices, draw commands, and other rendering resources.
type BuilderVolume struct {
	ctx                 api.IContext
	tex                 *Textures
	fv                  *FrameVertices
	dcOpaque            *DrawCommands
	dcAdditive          *DrawCommands
	dcLiquid            *DrawCommands
	dcTransparent       *DrawCommands
	fl                  *FrameLights
	dcRenderOpaque      *shaders.DrawCommandsRender
	dcRenderAdditive    *shaders.DrawCommandsRender
	dcRenderLiquid      *shaders.DrawCommandsRender
	dcRenderTransparent *shaders.DrawCommandsRender
	cSky                *textures.Material
	cal                 *model.Calibration
	visibleVol          *VisibleVolumes
	modes               []*DrawCommands
	occlusion           *OcclusionHW
	frustumFront        *metrics.Frustum
	lightBB             *aabb.BoundingBox
	playerId            uint64
	player              *things.Player

	cachedRawVols   []*volumes.Volume
	cachedSky       *textures.Material
	staticGen       uint64
	staticVertBytes int
	staticIdxBytes  int
	frameGen        uint64
	lastPreparedGen uint64

	thingsRender *ThingsRender

	volMeshes   map[*volumes.Faces]*shaders.ResidentMesh
	thingMeshes map[uint64][]*shaders.ResidentMesh
	texLookupFn func(*textures.Texture) (float32, bool)

	staticOpaque   []shaders.StaticDrawItem
	staticLiquid   []shaders.StaticDrawItem
	staticAdditive []shaders.StaticDrawItem
	staticTransSrc []shaders.TransparentDrawItem

	instancePool     *shaders.InstancePool
	instGroups       []rigidInstanceGroup
	numInstGroups    int
	instGroupMap     map[*shaders.ResidentMesh]int
	instOpaque       []shaders.InstancedDrawItem
	instLiquid       []shaders.InstancedDrawItem
	instAdditive     []shaders.InstancedDrawItem
	instTransSrc     []rigidTransItem
	lastInstVBO      uint32
	transparentItems []shaders.TransparentDrawItem
	lastCamGLX       float32
	lastCamGLY       float32
	lastCamGLZ       float32
	meshFrame        uint64
}

type rigidInstance struct {
	cx, cy, cz     float64
	qx, qy, qz, qw float64
}

type rigidInstanceGroup struct {
	mesh           *shaders.ResidentMesh
	instances      []rigidInstance
	baseByteOffset int32
}

type rigidTransItem struct {
	mesh           *shaders.ResidentMesh
	subIdx         int
	baseByteOffset int32
	cx, cy, cz     float32
}

// NewBuilderVolume initializes and returns a new BuilderVolume instance with configured textures and calibration settings.
func NewBuilderVolume(ctx api.IContext, tex *Textures, cal *model.Calibration, player *things.Player) *BuilderVolume {
	bv := &BuilderVolume{
		ctx:                 ctx,
		tex:                 tex,
		player:              player,
		playerId:            player.GetEntity().GetId(),
		fv:                  NewFrameVertices(ctx, 1048576),
		dcOpaque:            NewDrawCommands(config.BlendModeOpaque, 32768),
		dcAdditive:          NewDrawCommands(config.BlendModeAdditive, 4096),
		dcLiquid:            NewDrawCommands(config.BlendModeLiquid, 4096),
		dcTransparent:       NewDrawCommands(config.BlendModeTransparent, 4096),
		fl:                  NewFrameLights(ctx, 1024, cal.EnableShadowLights),
		dcRenderOpaque:      shaders.NewDrawCommandsRender(ctx, "opaque", false),
		dcRenderAdditive:    shaders.NewDrawCommandsRender(ctx, "additive", true),
		dcRenderLiquid:      shaders.NewDrawCommandsRender(ctx, "liquid", true),
		dcRenderTransparent: shaders.NewDrawCommandsRender(ctx, "transparent", false),
		cSky:                nil,
		cal:                 cal,
		visibleVol:          NewVisibleVols(8192),
		modes:               make([]*DrawCommands, config.BlendModeLatest),
		occlusion:           NewOcclusionHW(ctx, 4096),
		frustumFront:        metrics.NewFrustum(),
		lightBB:             aabb.NewBoundingBox(),
		cachedRawVols:       make([]*volumes.Volume, 0, 8192),
		thingsRender:        NewThingsRender(),
		volMeshes:           make(map[*volumes.Faces]*shaders.ResidentMesh, 1024),
		thingMeshes:         make(map[uint64][]*shaders.ResidentMesh, 256),
		instancePool:        shaders.NewInstancePool(ctx, 256),
		instGroups:          make([]rigidInstanceGroup, 32),
		instGroupMap:        make(map[*shaders.ResidentMesh]int, 64),
	}
	bv.texLookupFn = bv.lookupTexLayer
	prjScale := bv.cal.PrjScale
	bv.frustumFront.Setup(float32(bv.cal.AspectRatio), float32(bv.cal.ZFar), float32(bv.cal.ZNear), prjScale, prjScale)

	bv.modes[bv.dcOpaque.GetBlendMode()] = bv.dcOpaque
	bv.modes[bv.dcAdditive.GetBlendMode()] = bv.dcAdditive
	bv.modes[bv.dcLiquid.GetBlendMode()] = bv.dcLiquid
	bv.modes[bv.dcTransparent.GetBlendMode()] = bv.dcTransparent

	return bv
}

func (w *BuilderVolume) lookupTexLayer(tex *textures.Texture) (float32, bool) {
	if tex == nil {
		return 0, false
	}
	tId := tex.GetIdentifier()
	if tId < 0 {
		var ok bool
		tId, ok = w.tex.Get(tex)
		if !ok {
			return 0, false
		}
		tex.SetIdentifier(tId)
	}
	return tId, true
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

// GetUploadHints returns static prefix byte offsets and generation counters used for incremental GPU buffer uploads.
func (w *BuilderVolume) GetUploadHints() (staticVertBytes, staticIdxBytes int, staticGen, frameGen uint64) {
	return w.staticVertBytes, w.staticIdxBytes, w.staticGen, w.frameGen
}

// Compute processes volumes, lights, and things within the given frustum and prepares rendering draw commands.
func (w *BuilderVolume) Compute(fbw, fbh int32, vi *engine.ViewMatrix, engine *engine.Engine) {
	w.meshFrame++
	px, py, pz := vi.GetView()
	angle, pitch, roll := vi.GetAngle(), -vi.GetPitch(), vi.GetRoll()
	w.frustumFront.Update(float32(px), float32(py), float32(pz), angle, pitch, roll)

	frustumFront := vi.GetFrontFrustum(w.frustumFront.Get())
	size := w.cal.ZLightFar
	w.lightBB.RebuildCenter(px, py, pz, size, size, size)

	w.occlusion.Reset()

	staticChanged := w.pushQVolumes(engine.GetVolumes(), frustumFront, px, py, pz)
	w.pushQLights(engine.GetLights(), frustumFront, px, py, pz)
	thingsChanged := w.pushQThings(engine.GetThings(), frustumFront)

	if staticChanged || thingsChanged || w.frameGen == 0 {
		w.frameGen++
	}
	if w.lastPreparedGen != w.frameGen {
		w.dcRenderOpaque.Prepare(w.dcOpaque.GetDrawCommands())
		w.dcRenderOpaque.SetResidentDraws(w.staticOpaque, w.instOpaque)

		w.dcRenderAdditive.Prepare(w.dcAdditive.GetDrawCommands())
		w.dcRenderAdditive.SetResidentDraws(w.staticAdditive, w.instAdditive)

		w.dcRenderLiquid.Prepare(w.dcLiquid.GetDrawCommands())
		w.dcRenderLiquid.SetResidentDraws(w.staticLiquid, w.instLiquid)
		w.lastPreparedGen = w.frameGen
	}

	camGLX, camGLY, camGLZ := float32(px), float32(pz), float32(-py)
	transCmds := w.dcTransparent.GetDrawCommands()
	hasTrans := len(w.staticTransSrc) > 0 || len(w.instTransSrc) > 0 || len(transCmds) > 0
	camMoved := camGLX != w.lastCamGLX || camGLY != w.lastCamGLY || camGLZ != w.lastCamGLZ
	if hasTrans {
		if staticChanged || thingsChanged || camMoved || len(w.transparentItems) == 0 {
			w.transparentItems = w.transparentItems[:0]
			for i := range w.staticTransSrc {
				it := w.staticTransSrc[i]
				sub := &it.Mesh.SubMeshes[it.SubIdx]
				dx := sub.CentroidX - camGLX
				dy := sub.CentroidY - camGLY
				dz := sub.CentroidZ - camGLZ
				it.DistSq = dx*dx + dy*dy + dz*dz
				w.transparentItems = append(w.transparentItems, it)
			}
			for i := range w.instTransSrc {
				rt := &w.instTransSrc[i]
				sub := &rt.mesh.SubMeshes[rt.subIdx]
				dx := rt.cx - camGLX
				dy := rt.cy - camGLY
				dz := rt.cz - camGLZ
				w.transparentItems = append(w.transparentItems, shaders.TransparentDrawItem{
					Kind:           shaders.TransparentInstanced,
					DistSq:         dx*dx + dy*dy + dz*dz,
					SortOrder:      sub.SortOrder,
					CullMode:       sub.CullMode,
					HasCutout:      sub.HasCutout,
					PolygonOffset:  sub.PolygonOffset,
					Mesh:           rt.mesh,
					SubIdx:         rt.subIdx,
					InstanceVBO:    w.lastInstVBO,
					BaseByteOffset: rt.baseByteOffset,
					InstanceCount:  1,
				})
			}
			for _, cmd := range transCmds {
				if cmd.GetIndexCount() == 0 {
					continue
				}
				mat := cmd.GetMaterial()
				cx, cy, cz := cmd.GetCentroid()
				dx := cx - camGLX
				dy := cy - camGLY
				dz := cz - camGLZ
				sortOrder := 0
				cullMode := config.CullFront
				hasCutout := false
				polyOff := false
				if mat != nil {
					sortOrder = mat.GetSort()
					cullMode = mat.GetCullMode()
					hasCutout = mat.HasCutout()
					polyOff = mat.GetPolygonOffset()
				}
				w.transparentItems = append(w.transparentItems, shaders.TransparentDrawItem{
					Kind:          shaders.TransparentStreaming,
					DistSq:        dx*dx + dy*dy + dz*dz,
					SortOrder:     sortOrder,
					CullMode:      cullMode,
					HasCutout:     hasCutout,
					PolygonOffset: polyOff,
					FirstIndex:    cmd.GetFirstIndex(),
					IndexCount:    cmd.GetIndexCount(),
				})
			}
			shaders.SortTransparentDrawItems(w.transparentItems)
			w.dcRenderTransparent.SetTransparentDraws(w.transparentItems)
		}
	} else if len(w.transparentItems) > 0 {
		w.transparentItems = w.transparentItems[:0]
		w.dcRenderTransparent.SetTransparentDraws(nil)
	}
	w.lastCamGLX, w.lastCamGLY, w.lastCamGLZ = camGLX, camGLY, camGLZ
}

// pushQVolumes updates GPU-resident static volume meshes only when the visible volume set, geometry, or animated wall textures change.
func (w *BuilderVolume) pushQVolumes(vols *model.Volumes, frustumFront *aabb.Frustum, pX, pY, pZ float64) bool {
	if w.volMeshes == nil {
		w.volMeshes = make(map[*volumes.Faces]*shaders.ResidentMesh, 1024)
	}
	if w.texLookupFn == nil {
		w.texLookupFn = w.lookupTexLayer
	}
	w.visibleVol.Reset(vols.Len(), pX, pY, pZ)

	// Raccolta dal DBVH (Broad-Phase)
	vols.QueryFrustum(frustumFront, func(object aabb.IObject) bool {
		w.visibleVol.Add(object.(*volumes.Volume))
		return false
	})

	visLen := w.visibleVol.Len()
	canReuse := w.staticGen > 0 && visLen == len(w.cachedRawVols)
	if canReuse {
		for i := 0; i < visLen; i++ {
			vol := w.visibleVol.At(i)
			if vol != w.cachedRawVols[i] || vol.IsDirty() {
				canReuse = false
				break
			}
			faces := vol.GetFaces()
			if faces == nil {
				continue
			}
			rm := w.volMeshes[faces]
			if rm == nil || rm.NeedsRebuild(faces, true) {
				canReuse = false
				break
			}
		}
	}

	if canReuse {
		w.fv.Reset()
		w.dcOpaque.Reset()
		w.dcAdditive.Reset()
		w.dcLiquid.Reset()
		if w.dcTransparent != nil {
			w.dcTransparent.Reset()
		}
		w.cSky = w.cachedSky
		return false
	}

	w.meshFrame++
	w.fv.DeepReset()
	w.dcOpaque.DeepReset()
	w.dcAdditive.DeepReset()
	w.dcLiquid.DeepReset()
	if w.dcTransparent != nil {
		w.dcTransparent.DeepReset()
	}
	w.cSky = nil

	if cap(w.cachedRawVols) < visLen {
		w.cachedRawVols = make([]*volumes.Volume, visLen)
	} else {
		for i := visLen; i < len(w.cachedRawVols); i++ {
			w.cachedRawVols[i] = nil
		}
		w.cachedRawVols = w.cachedRawVols[:visLen]
	}
	copy(w.cachedRawVols, w.visibleVol.RawSlice())

	// Sort front-to-back for early-Z static submission order.
	w.visibleVol.Sort()

	w.staticOpaque = w.staticOpaque[:0]
	w.staticLiquid = w.staticLiquid[:0]
	w.staticAdditive = w.staticAdditive[:0]
	w.staticTransSrc = w.staticTransSrc[:0]

	for vIdx := 0; vIdx < visLen; vIdx++ {
		vol := w.visibleVol.At(vIdx)
		faces := vol.GetFaces()
		if faces == nil {
			vol.ClearDirty()
			continue
		}
		rm := w.volMeshes[faces]
		if rm == nil {
			rm = shaders.NewResidentMesh(w.ctx)
			w.volMeshes[faces] = rm
		}
		rm.LastSeenFrame = w.meshFrame
		if vol.IsDirty() || rm.NeedsRebuild(faces, true) {
			rm.Rebuild(faces, w.texLookupFn)
			vol.ClearDirty()
		}
		if rm.SkyMaterial != nil {
			w.cSky = rm.SkyMaterial
		}
		for sIdx := range rm.SubMeshes {
			sub := &rm.SubMeshes[sIdx]
			if sub.IndexCount == 0 {
				continue
			}
			switch sub.BlendMode {
			case config.BlendModeLiquid:
				w.staticLiquid = append(w.staticLiquid, shaders.StaticDrawItem{Mesh: rm, SubIdx: sIdx})
			case config.BlendModeAdditive:
				w.staticAdditive = append(w.staticAdditive, shaders.StaticDrawItem{Mesh: rm, SubIdx: sIdx})
			case config.BlendModeTransparent:
				w.staticTransSrc = append(w.staticTransSrc, shaders.TransparentDrawItem{
					Kind:          shaders.TransparentStatic,
					SortOrder:     sub.SortOrder,
					CullMode:      sub.CullMode,
					HasCutout:     sub.HasCutout,
					PolygonOffset: sub.PolygonOffset,
					Mesh:          rm,
					SubIdx:        sIdx,
				})
			default:
				w.staticOpaque = append(w.staticOpaque, shaders.StaticDrawItem{Mesh: rm, SubIdx: sIdx})
			}
		}
	}

	w.cachedSky = w.cSky
	w.fv.Freeze()
	w.dcOpaque.Freeze()
	w.dcAdditive.Freeze()
	w.dcLiquid.Freeze()
	if w.dcTransparent != nil {
		w.dcTransparent.Freeze()
	}
	w.staticVertBytes, w.staticIdxBytes = w.fv.FrozenBytes()
	w.staticGen++
	return true
}

// pushQThings processes visible things: rigid props/vehicles/buildings route into GPU instancing,
// while morphing 3D models, billboards, and UI elements route through the streaming vertex cache.
func (w *BuilderVolume) pushQThings(things *model.Things, frustumFront *aabb.Frustum) bool {
	if w.thingsRender == nil {
		w.thingsRender = NewThingsRender()
	}
	if w.thingMeshes == nil {
		w.thingMeshes = make(map[uint64][]*shaders.ResidentMesh, 256)
	}
	if w.meshFrame == 0 {
		w.meshFrame = 1
	}
	if w.instGroupMap == nil {
		w.instGroupMap = make(map[*shaders.ResidentMesh]int, 64)
	}
	if w.instancePool == nil {
		w.instancePool = shaders.NewInstancePool(w.ctx, 256)
	}
	if w.texLookupFn == nil {
		w.texLookupFn = w.lookupTexLayer
	}
	w.thingsRender.Prepare()
	w.numInstGroups = 0
	clear(w.instGroupMap)

	tick := textures.GlobalTick()

	things.StagePrepare()

	addRigidInstance := func(rm *shaders.ResidentMesh, cx, cy, cz, qx, qy, qz, qw float64) {
		gIdx, exists := w.instGroupMap[rm]
		if !exists {
			gIdx = w.numInstGroups
			w.numInstGroups++
			if gIdx >= len(w.instGroups) {
				w.instGroups = append(w.instGroups, rigidInstanceGroup{})
			}
			w.instGroupMap[rm] = gIdx
			w.instGroups[gIdx].mesh = rm
			w.instGroups[gIdx].instances = w.instGroups[gIdx].instances[:0]
		}
		w.instGroups[gIdx].instances = append(w.instGroups[gIdx].instances, rigidInstance{
			cx: cx, cy: cy, cz: cz,
			qx: qx, qy: qy, qz: qz, qw: qw,
		})
	}

	pushPassThing := func(thing ithing.IThing) {
		thing.Compile(tick)

		ent := thing.GetEntity()
		entry2 := w.thingsRender.Retrieve(ent)

		if rs, ok := thing.(ithing.IRigidRenderSource); ok {
			if rigidFaces := rs.GetRigidFaces(); rigidFaces != nil && rigidFaces.Len() > 0 {
				rm, meshRebuilt := w.retrieveThingMesh(rigidFaces)
				tPosX, tPosY, zBot := ent.GetCenter()
				qx, qy, qz, qw := ent.GetQRotation().Get()
				oX, oY, oZ := float32(tPosX), float32(zBot), float32(-tPosY)
				fqx, fqy, fqz, fqw := float32(qx), float32(qy), float32(qz), float32(qw)
				if meshRebuilt || !entry2.EqualRigid(rigidFaces, rigidFaces.GetVersion(), rigidFaces.Len(), oX, oY, oZ, fqx, fqy, fqz, fqw) {
					entry2.AssignRigid(rigidFaces, rigidFaces.GetVersion(), rigidFaces.Len(), oX, oY, oZ, fqx, fqy, fqz, fqw)
				}
				w.thingsRender.AddRef(entry2)
				if len(rm.SubMeshes) > 0 {
					addRigidInstance(rm, tPosX, tPosY, zBot, qx, qy, qz, qw)
				}
				return
			}
		}

		pFaces, pNextFaces, lp, renderMode := thing.GetUIFaces()
		if pFaces == nil || pFaces.Len() == 0 {
			return
		}

		lerp := float32(lp)
		yaw := float32(thing.GetAngle())
		if renderMode == vertices.RenderModeModel3D {
			yaw = 0.0
		}
		renderModeF := float32(renderMode)
		tPosX, tPosY, zBot := ent.GetCenter()
		oX, oY, oZ := float32(tPosX), float32(zBot), float32(-tPosY)

		facesVer := pFaces.GetVersion()
		facesLen := pFaces.Len()
		var nextVer uint64
		var nextLen int
		if pNextFaces != nil {
			nextVer = pNextFaces.GetVersion()
			nextLen = pNextFaces.Len()
		}

		fv := entry2.GetFrameVertices()
		cacheHit := entry2.Equal(pFaces, pNextFaces, facesVer, nextVer, facesLen, nextLen, lerp, yaw, renderModeF, oX, oY, oZ)
		if !cacheHit {
			fv.DeepReset()

			var currentMat *textures.Material
			batchStart := fv.GetIndicesLen()
			var centSumX, centSumY, centSumZ float64
			var centCount int

			flushBatch := func() {
				currentIndices := fv.GetIndicesLen()
				if currentIndices > batchStart && currentMat != nil {
					var bcx, bcy, bcz float32
					if centCount > 0 {
						inv := 1.0 / float64(centCount)
						bcx = float32(centSumX * inv)
						bcy = float32(centSumY * inv)
						bcz = float32(centSumZ * inv)
					} else {
						bcx, bcy, bcz = oX, oY, oZ
					}
					fv.AddBatchCentroid(currentMat.GetBlendMode(), currentIndices-batchStart, currentMat, bcx, bcy, bcz)
				}
				batchStart = currentIndices
				centSumX, centSumY, centSumZ = 0, 0, 0
				centCount = 0
			}

			limit := facesLen
			if pNextFaces != nil && nextLen < limit {
				limit = nextLen
			}

			for fx := 0; fx < limit; fx++ {
				face := pFaces.At(fx)
				if face == nil {
					continue
				}
				mat := face.GetMaterial()
				if mat == nil || mat.GetKind() == int(config.MaterialKindNone) {
					continue
				}
				tex := mat.GetCurrentFrame()
				if tex == nil {
					continue
				}
				if mat.IsAnimated() {
					fv.AddAnimRef(mat, tex)
				}
				tId := tex.GetIdentifier()
				if tId < 0 {
					tId, _ = w.tex.Get(tex)
					tex.SetIdentifier(tId)
				}

				if mat != currentMat {
					flushBatch()
					currentMat = mat
				}

				packed, spec, shininess, detailScale, detailFactor, envStr, envTint := mat.GetRenderParams()
				p0X, p0Y, p0Z, p1X, p1Y, p1Z, p2X, p2Y, p2Z := face.GetPointsFlat()
				u0, u1, u2, v0, v1, v2 := face.GetUVFlat()
				nx0, ny0, nz0, nx1, ny1, nz1, nx2, ny2, nz2 := face.GetVertexNormalsFlat()
				tx0, ty0, tz0, s0, tx1, ty1, tz1, s1, tx2, ty2, tz2, s2 := face.GetTangentsFlat()
				lmID, lmU, lmV := face.GetLightmap()
				lmIDFloat := float32(lmID)

				if renderMode == vertices.RenderModeOpaque || renderMode == vertices.RenderModeLiquid {
					centSumX += (p0X + p1X + p2X) / 3.0
					centSumY += (p0Z + p1Z + p2Z) / 3.0
					centSumZ += -(p0Y + p1Y + p2Y) / 3.0
				} else {
					centSumX += float64(oX)
					centSumY += float64(oY)
					centSumZ += float64(oZ)
				}
				centCount++

				var np0X, np0Y, np0Z, np1X, np1Y, np1Z, np2X, np2Y, np2Z float64
				if pNextFaces != nil {
					if nf := pNextFaces.At(fx); nf != nil {
						np0X, np0Y, np0Z, np1X, np1Y, np1Z, np2X, np2Y, np2Z = nf.GetPointsFlat()
					} else {
						np0X, np0Y, np0Z, np1X, np1Y, np1Z, np2X, np2Y, np2Z = p0X, p0Y, p0Z, p1X, p1Y, p1Z, p2X, p2Y, p2Z
					}
				} else {
					np0X, np0Y, np0Z, np1X, np1Y, np1Z, np2X, np2Y, np2Z = p0X, p0Y, p0Z, p1X, p1Y, p1Z, p2X, p2Y, p2Z
				}

				id0 := fv.AddVertexFull(float32(p0X), float32(p0Z), float32(-p0Y), float32(u0), float32(-v0), tId, float32(nx0), float32(nz0), float32(-ny0), float32(tx0), float32(tz0), float32(-ty0), float32(s0), float32(lmU[0]), float32(lmV[0]), lmIDFloat, oX, oY, oZ, renderModeF, float32(np0X), float32(np0Z), float32(-np0Y), lerp, yaw, packed, spec, shininess, detailScale, detailFactor, envStr, envTint)
				id1 := fv.AddVertexFull(float32(p1X), float32(p1Z), float32(-p1Y), float32(u1), float32(-v1), tId, float32(nx1), float32(nz1), float32(-ny1), float32(tx1), float32(tz1), float32(-ty1), float32(s1), float32(lmU[1]), float32(lmV[1]), lmIDFloat, oX, oY, oZ, renderModeF, float32(np1X), float32(np1Z), float32(-np1Y), lerp, yaw, packed, spec, shininess, detailScale, detailFactor, envStr, envTint)
				id2 := fv.AddVertexFull(float32(p2X), float32(p2Z), float32(-p2Y), float32(u2), float32(-v2), tId, float32(nx2), float32(nz2), float32(-ny2), float32(tx2), float32(tz2), float32(-ty2), float32(s2), float32(lmU[2]), float32(lmV[2]), lmIDFloat, oX, oY, oZ, renderModeF, float32(np2X), float32(np2Z), float32(-np2Y), lerp, yaw, packed, spec, shininess, detailScale, detailFactor, envStr, envTint)
				fv.AddTriangle(id0, id1, id2)
			}

			flushBatch()
			entry2.Assign(pFaces, pNextFaces, facesVer, nextVer, facesLen, nextLen, lerp, yaw, renderModeF, oX, oY, oZ)
		}

		w.fv.Commit(fv, w.modes)
		w.thingsRender.AddRef(entry2)
	}

	qOcclusion := func(object aabb.IObject) bool {
		thing := object.(ithing.IThing)
		if !thing.IsVisible() {
			return false
		}
		entId := thing.GetEntity().GetId()
		if w.playerId != entId {
			things.StageAdd(thing)
		}
		if thing.GetKind() != config.ThingDoorDef {
			ab := thing.GetAABB()
			occState := w.occlusion.GetState(entId)
			if occState != nil && !occState.IsVisible {
				w.occlusion.Add(ab, occState) // Schedulalo per controllarlo al prossimo frame
				return false                  // CULLATO! Non generiamo i vertici
			}
			w.occlusion.Add(ab, occState) // Lo vediamo, aggiungiamolo ai test GPU
		}
		pushPassThing(thing)
		return false
	}

	qStatic := func(object aabb.IObject) bool {
		pushPassThing(object.(ithing.IThing))
		return false
	}

	things.StageAdd(w.player)

	things.QueryStatic(qStatic) // UI Objects
	things.QueryFrustum(frustumFront, qOcclusion)

	things.StageCommit()

	thingsChanged := w.thingsRender.Commit()
	if thingsChanged || w.frameGen == 0 {
		w.instOpaque = w.instOpaque[:0]
		w.instLiquid = w.instLiquid[:0]
		w.instAdditive = w.instAdditive[:0]
		w.instTransSrc = w.instTransSrc[:0]

		w.instancePool.Reset()
		for gi := 0; gi < w.numInstGroups; gi++ {
			g := &w.instGroups[gi]
			cnt := len(g.instances)
			if cnt == 0 {
				continue
			}
			baseByteOff, baseFloat := w.instancePool.Reserve(cnt)
			g.baseByteOffset = baseByteOff
			for k, inst := range g.instances {
				w.instancePool.WriteTransformAt(baseFloat+k*shaders.InstanceMatrixFloats, inst.cx, inst.cy, inst.cz, inst.qx, inst.qy, inst.qz, inst.qw)
			}
		}
		instVBO := w.instancePool.Upload()
		w.lastInstVBO = instVBO

		for gi := 0; gi < w.numInstGroups; gi++ {
			g := &w.instGroups[gi]
			cnt := int32(len(g.instances))
			if cnt == 0 {
				continue
			}
			for sIdx := range g.mesh.SubMeshes {
				sub := &g.mesh.SubMeshes[sIdx]
				if sub.IndexCount == 0 {
					continue
				}
				switch sub.BlendMode {
				case config.BlendModeLiquid:
					w.instLiquid = append(w.instLiquid, shaders.InstancedDrawItem{
						Mesh:           g.mesh,
						SubIdx:         sIdx,
						InstanceVBO:    instVBO,
						BaseByteOffset: g.baseByteOffset,
						InstanceCount:  cnt,
					})
				case config.BlendModeAdditive:
					w.instAdditive = append(w.instAdditive, shaders.InstancedDrawItem{
						Mesh:           g.mesh,
						SubIdx:         sIdx,
						InstanceVBO:    instVBO,
						BaseByteOffset: g.baseByteOffset,
						InstanceCount:  cnt,
					})
				case config.BlendModeTransparent:
					for k, inst := range g.instances {
						w.instTransSrc = append(w.instTransSrc, rigidTransItem{
							mesh:           g.mesh,
							subIdx:         sIdx,
							baseByteOffset: g.baseByteOffset + int32(k*shaders.InstanceMatrixFloats*4),
							cx:             float32(inst.cx),
							cy:             float32(inst.cz),
							cz:             float32(-inst.cy),
						})
					}
				default:
					w.instOpaque = append(w.instOpaque, shaders.InstancedDrawItem{
						Mesh:           g.mesh,
						SubIdx:         sIdx,
						InstanceVBO:    instVBO,
						BaseByteOffset: g.baseByteOffset,
						InstanceCount:  cnt,
					})
				}
			}
		}
	}

	return thingsChanged
}

// pushQLights processes a collection of lights within the specified front and rear frustums and updates the frame lighting.
// It resets the frame lights state, prepares lighting at the given coordinates, and queries lights intersecting the frustums.
func (w *BuilderVolume) pushQLights(ls *model.Lights, frustumFront *aabb.Frustum, pX, pY, pZ float64) {
	w.fl.DeepReset()
	w.fl.Prepare()
	//counter := 0

	queryLights := func(object aabb.IObject) bool {
		light := object.(*lights.Light)
		if light.GetKind() == config.LightKindNone {
			return false
		}
		distSq := light.GetDistSq(pX, pY, pZ)
		isFront := light.GetAABB().IntersectFrustum(frustumFront)
		w.fl.Create(light, distSq, isFront)
		//counter++
		return false
	}

	ls.QueryOverlaps(w.lightBB, queryLights)
	//fmt.Println("Processed lights:", ls.Len(), counter)
}

// GetDrawCommandsAdditive returns the additive draw commands prepared for rendering.
func (w *BuilderVolume) GetDrawCommandsAdditive() *shaders.DrawCommandsRender {
	return w.dcRenderAdditive
}

// GetDrawCommandsLiquid returns the liquid draw commands prepared for rendering.
func (w *BuilderVolume) GetDrawCommandsLiquid() *shaders.DrawCommandsRender {
	return w.dcRenderLiquid
}

// GetDrawCommandsTransparent returns the sorted transparent draw commands prepared for rendering.
func (w *BuilderVolume) GetDrawCommandsTransparent() *shaders.DrawCommandsRender {
	return w.dcRenderTransparent
}

// GetOpaqueInstancedItems returns the active rigid instanced opaque draw items for inspection.
func (w *BuilderVolume) GetOpaqueInstancedItems() []shaders.InstancedDrawItem {
	return w.instOpaque
}

// GetResidentVolumeCount returns the number of active resident static volume submesh draws.
func (w *BuilderVolume) GetResidentVolumeCount() int {
	return len(w.staticOpaque) + len(w.staticLiquid) + len(w.staticAdditive) + len(w.staticTransSrc)
}

// GetInstancedDrawCount returns the number of active GPU-instanced submesh draw groups.
func (w *BuilderVolume) GetInstancedDrawCount() int {
	return len(w.instOpaque) + len(w.instLiquid) + len(w.instAdditive) + len(w.instTransSrc)
}

// GetStreamingVertexCount returns the current vertex count stored in the CPU streaming buffer.
func (w *BuilderVolume) GetStreamingVertexCount() int {
	_, _, vCount, _, _, _ := w.fv.GetVertices()
	return int(vCount) / shaders.StreamingVertexFloats
}

// GetHWOcclusion retrieves the hardware-based occlusion object associated with the BuilderVolume.
func (w *BuilderVolume) GetHWOcclusion() *OcclusionHW {
	return w.occlusion
}

func (w *BuilderVolume) retrieveThingMesh(rigidFaces *volumes.Faces) (*shaders.ResidentMesh, bool) {
	gh := rigidFaces.GetGeomHash()
	bucket := w.thingMeshes[gh]
	for _, rm := range bucket {
		if rm.SrcFaces == rigidFaces || (rm.FaceCount == rigidFaces.Len() && rm.SrcFaces.GetGeomHash() == gh && facesEquivalent(rigidFaces, rm.SrcFaces)) {
			if rm.LastSeenFrame != w.meshFrame {
				rm.LastSeenFrame = w.meshFrame
				if (rm.SrcFaces == rigidFaces && rm.NeedsRebuild(rigidFaces, true)) || rm.NeedsAnimRebuild() {
					rm.Rebuild(rigidFaces, w.texLookupFn)
					rm.RebuiltFrame = w.meshFrame
				}
			}
			return rm, rm.RebuiltFrame == w.meshFrame
		}
	}
	rm := shaders.NewResidentMesh(w.ctx)
	rm.LastSeenFrame = w.meshFrame
	rm.RebuiltFrame = w.meshFrame
	rm.Rebuild(rigidFaces, w.texLookupFn)
	w.thingMeshes[gh] = append(bucket, rm)
	return rm, true
}

func facesEquivalent(a, b *volumes.Faces) bool {
	if a == b {
		return true
	}
	if a == nil || b == nil || a.Len() != b.Len() {
		return false
	}
	n := a.Len()
	for i := 0; i < n; i++ {
		fa := a.At(i)
		fb := b.At(i)
		if fa == fb {
			continue
		}
		if fa == nil || fb == nil || fa.GetMaterial() != fb.GetMaterial() {
			return false
		}
		ax0, ay0, az0, ax1, ay1, az1, ax2, ay2, az2 := fa.GetPointsFlat()
		bx0, by0, bz0, bx1, by1, bz1, bx2, by2, bz2 := fb.GetPointsFlat()
		if ax0 != bx0 || ay0 != by0 || az0 != bz0 ||
			ax1 != bx1 || ay1 != by1 || az1 != bz1 ||
			ax2 != bx2 || ay2 != by2 || az2 != bz2 {
			return false
		}
	}
	return true
}

/*
// pushQLights processes a collection of lights within the specified front and rear frustums and updates the frame lighting.
// It resets the frame lights state, prepares lighting at the given coordinates, and queries lights intersecting the frustums.
func (w *BuilderVolume) pushQLights(ls *model.Lights, frustumFront, frustumRear *aabb.Frustum, pX, pY, pZ float64) {
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

// pushQLights processes a collection of lights within the specified front and rear frustums and updates the frame lighting.
// It resets the frame lights state, prepares lighting at the given coordinates, and queries lights intersecting the frustums.
func (w *BuilderVolume) pushLights(ls *model.Lights, frustumFront *aabb.Frustum, pX, pY, pZ float64) {
	//TODO IMPLEMENTAZIONE CON FRUSTUM
	w.fl.DeepReset()
	w.fl.Prepare()
	//counter := 0

	maxDistFrontSq := w.cal.ZLightFar * w.cal.ZLightFar
	maxDistBackSq := maxDistFrontSq

	for _, light := range ls.Get() {
		if light.GetKind() == config.LightKindNone {
			continue
		}
		distSq := light.GetDistSq(pX, pY, pZ)
		isFront := light.GetAABB().IntersectFrustum(frustumFront)
		if isFront {
			if distSq > maxDistFrontSq {
				continue
			}
		} else {
			if distSq > maxDistBackSq {
				continue
			}
		}
		w.fl.Create(light, distSq, isFront)
		//counter++
	}
	//_, _, dynamicCount := w.fl.GetLights()
	//_, shadowCount := w.fl.GetShadowLights()
	//fmt.Printf("LIGHTS %d (DRAW %d -> DYNAMIC %d, SHADOW %d)\n", ls.Len(), counter, dynamicCount, shadowCount)
}

*/
