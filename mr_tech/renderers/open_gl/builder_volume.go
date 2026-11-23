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
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/gpu"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/metrics"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/shaders"
	"github.com/markel1974/godoom/mr_tech/textures"
)

type RigidInstance struct {
	cx, cy, cz     float64
	qx, qy, qz, qw float64
}

type RigidInstanceGroup struct {
	mesh           *gpu.ResidentMesh
	instances      gpu.PtrPool[RigidInstance]
	baseByteOffset int32
}

type RigidTransItem struct {
	mesh           *gpu.ResidentMesh
	sub            *gpu.ResidentSubMesh
	baseByteOffset int32
	cx, cy, cz     float32
}

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

	cachedRawVols    []*volumes.Volume
	cachedRawVolsLen int
	cachedSky        *textures.Material
	staticGen        uint64
	staticVertBytes  int
	staticIdxBytes   int
	frameGen         uint64
	lastPreparedGen  uint64

	thingsRender *ThingsRender

	volMeshes           map[*volumes.Faces]*gpu.ResidentMesh
	staticAtlas         *gpu.ResidentMesh
	staticEntries       map[*volumes.Faces]*staticVolumeEntry
	staticAtlasBuilt    bool
	staticAtlasVolCount int
	thingMeshes         map[uint64][]*gpu.ResidentMesh
	texLookupFn         func(*textures.Texture) (float32, bool)

	staticOpaque   gpu.PtrPool[shaders.StaticDrawItem]
	staticLiquid   gpu.PtrPool[shaders.StaticDrawItem]
	staticAdditive gpu.PtrPool[shaders.StaticDrawItem]
	staticTransSrc gpu.PtrPool[shaders.TransparentDrawItem]

	instancePool     *gpu.InstancePool
	instGroups       gpu.PtrPool[RigidInstanceGroup]
	instFrame        uint64
	instOpaque       gpu.PtrPool[shaders.InstancedDrawItem]
	instLiquid       gpu.PtrPool[shaders.InstancedDrawItem]
	instAdditive     gpu.PtrPool[shaders.InstancedDrawItem]
	instTransSrc     gpu.PtrPool[RigidTransItem]
	lastInstVBO      uint32
	transparentItems *shaders.TransparentDrawItems

	lastCamGLX float32
	lastCamGLY float32
	lastCamGLZ float32
	meshFrame  uint64
}

type staticVolumeEntry struct {
	subStart    int
	subCount    int
	skyMaterial *textures.Material
	version     uint64
	faceCount   int
	detached    bool
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
		cachedRawVols:       make([]*volumes.Volume, 8192),
		thingsRender:        NewThingsRender(),
		volMeshes:           make(map[*volumes.Faces]*gpu.ResidentMesh, 1024),
		thingMeshes:         make(map[uint64][]*gpu.ResidentMesh, 256),
		instancePool:        gpu.NewInstancePool(ctx, 256),
		transparentItems:    shaders.NewTransparentDrawItems(64),
	}
	bv.instGroups.Init(32)
	bv.texLookupFn = bv.lookupTexLayer
	prjScale := bv.cal.PrjScale
	bv.frustumFront.Setup(float32(bv.cal.AspectRatio), float32(bv.cal.ZFar), float32(bv.cal.ZNear), prjScale, prjScale)

	bv.modes[bv.dcOpaque.GetBlendMode()] = bv.dcOpaque
	bv.modes[bv.dcAdditive.GetBlendMode()] = bv.dcAdditive
	bv.modes[bv.dcLiquid.GetBlendMode()] = bv.dcLiquid
	bv.modes[bv.dcTransparent.GetBlendMode()] = bv.dcTransparent

	return bv
}

// lookupTexLayer returns the texture layer identifier for tex, caching it on the texture when found.
// lookupTexLayer reports false when tex is nil or not registered in the builder texture map.
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
		w.dcRenderOpaque.SetResidentDraws(w.staticOpaque.Items, w.staticOpaque.Len, w.instOpaque.Items, w.instOpaque.Len)

		w.dcRenderAdditive.Prepare(w.dcAdditive.GetDrawCommands())
		w.dcRenderAdditive.SetResidentDraws(w.staticAdditive.Items, w.staticAdditive.Len, w.instAdditive.Items, w.instAdditive.Len)

		w.dcRenderLiquid.Prepare(w.dcLiquid.GetDrawCommands())
		w.dcRenderLiquid.SetResidentDraws(w.staticLiquid.Items, w.staticLiquid.Len, w.instLiquid.Items, w.instLiquid.Len)
		w.lastPreparedGen = w.frameGen
	}

	camGLX, camGLY, camGLZ := float32(px), float32(pz), float32(-py)
	transCmds := w.dcTransparent.GetDrawCommands()
	hasTrans := w.staticTransSrc.Len > 0 || w.instTransSrc.Len > 0 || len(transCmds) > 0
	camMoved := camGLX != w.lastCamGLX || camGLY != w.lastCamGLY || camGLZ != w.lastCamGLZ
	if hasTrans {
		if staticChanged || thingsChanged || camMoved || w.transparentItems.Len() == 0 {
			w.transparentItems.Begin()
			w.transparentItems.AddStatic(&w.staticTransSrc, camGLX, camGLY, camGLZ)
			for i := 0; i < w.instTransSrc.Len; i++ {
				rt := w.instTransSrc.Items[i]
				w.transparentItems.AddInstanced(rt.mesh, rt.sub, w.lastInstVBO, rt.baseByteOffset, rt.cx, rt.cy, rt.cz, camGLX, camGLY, camGLZ)
			}
			w.transparentItems.AddCommands(transCmds, camGLX, camGLY, camGLZ)
			w.transparentItems.End()
			w.dcRenderTransparent.SetTransparentDraws(w.transparentItems)
		}
	} else if w.transparentItems.Len() > 0 {
		w.transparentItems.Clear()
		w.dcRenderTransparent.SetTransparentDraws(nil)
	}
	w.lastCamGLX, w.lastCamGLY, w.lastCamGLZ = camGLX, camGLY, camGLZ
}

// pushQVolumes updates GPU-resident static volume meshes only when the visible volume set, geometry, or animated wall textures change.
func (w *BuilderVolume) pushQVolumes(vols *model.Volumes, frustumFront *aabb.Frustum, pX, pY, pZ float64) bool {
	if w.volMeshes == nil {
		w.volMeshes = make(map[*volumes.Faces]*gpu.ResidentMesh, 256)
	}
	if w.texLookupFn == nil {
		w.texLookupFn = w.lookupTexLayer
	}
	if !w.staticAtlasBuilt || (vols != nil && w.staticAtlasVolCount != vols.Len()) {
		w.buildStaticAtlas(vols)
	}
	w.visibleVol.Reset(vols.Len(), pX, pY, pZ)

	// Raccolta dal DBVH (Broad-Phase)
	vols.QueryFrustum(frustumFront, func(object aabb.IObject) bool {
		w.visibleVol.Add(object.(*volumes.Volume))
		return false
	})

	visLen := w.visibleVol.Len()
	canReuse := w.staticGen > 0 && visLen == w.cachedRawVolsLen
	if canReuse {
		for i := range visLen {
			vol := w.visibleVol.At(i)
			if vol != w.cachedRawVols[i] || vol.IsDirty() {
				canReuse = false
				break
			}
			faces := vol.GetFaces()
			if faces == nil {
				continue
			}
			if entry := w.staticEntries[faces]; entry != nil && !entry.detached {
				if faces.GetVersion() != entry.version || faces.Len() != entry.faceCount {
					canReuse = false
					break
				}
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

	if len(w.cachedRawVols) < visLen {
		w.cachedRawVols = make([]*volumes.Volume, max(len(w.cachedRawVols)*2, visLen))
	} else {
		for i := visLen; i < w.cachedRawVolsLen; i++ {
			w.cachedRawVols[i] = nil
		}
	}
	w.cachedRawVolsLen = visLen
	for i := range visLen {
		w.cachedRawVols[i] = w.visibleVol.At(i)
	}

	// Sort front-to-back for early-Z static submission order.
	w.visibleVol.Sort()

	w.staticOpaque.Reset()
	w.staticLiquid.Reset()
	w.staticAdditive.Reset()
	w.staticTransSrc.Reset()

	for vIdx := range visLen {
		vol := w.visibleVol.At(vIdx)
		faces := vol.GetFaces()
		if faces == nil {
			vol.ClearDirty()
			continue
		}
		if entry := w.staticEntries[faces]; entry != nil && !entry.detached {
			if vol.IsDirty() || faces.GetVersion() != entry.version || faces.Len() != entry.faceCount {
				entry.detached = true
			} else {
				if entry.skyMaterial != nil {
					w.cSky = entry.skyMaterial
				}
				for sIdx := entry.subStart; sIdx < entry.subStart+entry.subCount; sIdx++ {
					w.pushStaticSubMesh(w.staticAtlas, w.staticAtlas.SubMeshes[sIdx])
				}
				continue
			}
		}
		rm := w.volMeshes[faces]
		if rm == nil {
			rm = gpu.NewResidentMesh(w.ctx)
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
		for sIdx := 0; sIdx < rm.SubMeshesLen; sIdx++ {
			w.pushStaticSubMesh(rm, rm.SubMeshes[sIdx])
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

func (w *BuilderVolume) buildStaticAtlas(vols *model.Volumes) {
	w.staticAtlasBuilt = true
	w.staticAtlasVolCount = 0
	if vols == nil {
		return
	}
	w.staticAtlasVolCount = vols.Len()
	if vols.HasStreamer() || vols.GetVolumes() == nil || vols.Len() == 0 {
		return
	}
	rawVols := vols.GetVolumes().Slice()
	totalFaces := 0
	for _, vol := range rawVols {
		if vol == nil {
			continue
		}
		if faces := vol.GetFaces(); faces != nil {
			totalFaces += faces.Len()
		}
	}
	if totalFaces == 0 {
		return
	}
	if w.staticAtlas == nil {
		w.staticAtlas = gpu.NewResidentMesh(w.ctx)
	}
	w.staticEntries = make(map[*volumes.Faces]*staticVolumeEntry, len(rawVols))
	w.staticAtlas.ReserveStaticAtlas(totalFaces)
	w.staticAtlas.BeginStaticAtlas()
	for _, vol := range rawVols {
		if vol == nil {
			continue
		}
		faces := vol.GetFaces()
		if faces == nil || faces.Len() == 0 {
			continue
		}
		if w.staticEntries[faces] != nil {
			vol.ClearDirty()
			continue
		}
		subStart, subCount, skyMat, hasAnim, ok := w.staticAtlas.AppendStaticFaces(faces, w.texLookupFn)
		if !hasAnim && (ok || skyMat != nil) {
			w.staticEntries[faces] = &staticVolumeEntry{
				subStart:    subStart,
				subCount:    subCount,
				skyMaterial: skyMat,
				version:     faces.GetVersion(),
				faceCount:   faces.Len(),
			}
			vol.ClearDirty()
		}
	}
	if w.staticAtlas.SubMeshesLen > 0 {
		w.staticAtlas.EndStaticAtlas()
	}
}

func (w *BuilderVolume) pushStaticSubMesh(rm *gpu.ResidentMesh, sub *gpu.ResidentSubMesh) {
	if sub.IndexCount == 0 {
		return
	}
	switch sub.BlendMode {
	case config.BlendModeLiquid:
		item := w.staticLiquid.Acquire()
		item.Mesh = rm
		item.Sub = sub
	case config.BlendModeAdditive:
		item := w.staticAdditive.Acquire()
		item.Mesh = rm
		item.Sub = sub
	case config.BlendModeTransparent:
		item := w.staticTransSrc.Acquire()
		item.Kind = shaders.TransparentStatic
		item.DistSq = 0
		item.SortOrder = sub.SortOrder
		item.CullMode = sub.CullMode
		item.HasCutout = sub.HasCutout
		item.PolygonOffset = sub.PolygonOffset
		item.FirstIndex = 0
		item.IndexCount = 0
		item.Mesh = rm
		item.Sub = sub
		item.InstanceVBO = 0
		item.BaseByteOffset = 0
		item.InstanceCount = 0
	default:
		item := w.staticOpaque.Acquire()
		item.Mesh = rm
		item.Sub = sub
	}
}

// pushQThings processes visible things: rigid props/vehicles/buildings route into GPU instancing,
// while morphing 3D models, billboards, and UI elements route through the streaming vertex cache.
func (w *BuilderVolume) pushQThings(things *model.Things, frustumFront *aabb.Frustum) bool {
	if w.thingsRender == nil {
		w.thingsRender = NewThingsRender()
	}
	if w.thingMeshes == nil {
		w.thingMeshes = make(map[uint64][]*gpu.ResidentMesh, 256)
	}
	if w.meshFrame == 0 {
		w.meshFrame = 1
	}
	if w.instancePool == nil {
		w.instancePool = gpu.NewInstancePool(w.ctx, 256)
	}
	if w.texLookupFn == nil {
		w.texLookupFn = w.lookupTexLayer
	}
	w.thingsRender.Prepare()
	w.instFrame++
	w.instGroups.Reset()

	tick := textures.GlobalTick()

	things.StagePrepare()

	addRigidInstance := func(rm *gpu.ResidentMesh, cx, cy, cz, qx, qy, qz, qw float64) {
		var g *RigidInstanceGroup
		if rm.InstGroupFrame == w.instFrame {
			g = w.instGroups.Items[rm.InstGroupIdx]
		} else {
			rm.InstGroupFrame = w.instFrame
			rm.InstGroupIdx = w.instGroups.Len
			g = w.instGroups.Acquire()
			g.mesh = rm
			g.instances.Reset()
		}
		inst := g.instances.Acquire()
		inst.cx = cx
		inst.cy = cy
		inst.cz = cz
		inst.qx = qx
		inst.qy = qy
		inst.qz = qz
		inst.qw = qw
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
				if rm.SubMeshesLen > 0 {
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
		w.instOpaque.Reset()
		w.instLiquid.Reset()
		w.instAdditive.Reset()
		w.instTransSrc.Reset()

		w.instancePool.Reset()
		for gi := 0; gi < w.instGroups.Len; gi++ {
			g := w.instGroups.Items[gi]
			cnt := g.instances.Len
			if cnt == 0 {
				continue
			}
			baseByteOff, baseFloat := w.instancePool.Reserve(cnt)
			g.baseByteOffset = baseByteOff
			for k := range cnt {
				inst := g.instances.Items[k]
				w.instancePool.WriteTransformAt(baseFloat+k*gpu.InstanceMatrixFloats, inst.cx, inst.cy, inst.cz, inst.qx, inst.qy, inst.qz, inst.qw)
			}
		}
		instVBO := w.instancePool.Upload()
		w.lastInstVBO = instVBO

		for gi := 0; gi < w.instGroups.Len; gi++ {
			g := w.instGroups.Items[gi]
			cnt := int32(g.instances.Len)
			if cnt == 0 {
				continue
			}
			for sIdx := 0; sIdx < g.mesh.SubMeshesLen; sIdx++ {
				sub := g.mesh.SubMeshes[sIdx]
				if sub.IndexCount == 0 {
					continue
				}
				switch sub.BlendMode {
				case config.BlendModeLiquid:
					item := w.instLiquid.Acquire()
					item.Mesh = g.mesh
					item.Sub = sub
					item.InstanceVBO = instVBO
					item.BaseByteOffset = g.baseByteOffset
					item.InstanceCount = cnt
				case config.BlendModeAdditive:
					item := w.instAdditive.Acquire()
					item.Mesh = g.mesh
					item.Sub = sub
					item.InstanceVBO = instVBO
					item.BaseByteOffset = g.baseByteOffset
					item.InstanceCount = cnt
				case config.BlendModeTransparent:
					for k := 0; k < int(cnt); k++ {
						inst := g.instances.Items[k]
						rt := w.instTransSrc.Acquire()
						rt.mesh = g.mesh
						rt.sub = sub
						rt.baseByteOffset = g.baseByteOffset + int32(k*gpu.InstanceMatrixFloats*4)
						rt.cx = float32(inst.cx)
						rt.cy = float32(inst.cz)
						rt.cz = float32(-inst.cy)
					}
				default:
					item := w.instOpaque.Acquire()
					item.Mesh = g.mesh
					item.Sub = sub
					item.InstanceVBO = instVBO
					item.BaseByteOffset = g.baseByteOffset
					item.InstanceCount = cnt
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
func (w *BuilderVolume) GetOpaqueInstancedItems() []*shaders.InstancedDrawItem {
	return w.instOpaque.Items[:w.instOpaque.Len]
}

// GetResidentVolumeCount returns the number of active resident static volume submesh draws.
func (w *BuilderVolume) GetResidentVolumeCount() int {
	return w.staticOpaque.Len + w.staticLiquid.Len + w.staticAdditive.Len + w.staticTransSrc.Len
}

// GetInstancedDrawCount returns the number of active GPU-instanced submesh draw groups.
func (w *BuilderVolume) GetInstancedDrawCount() int {
	return w.instOpaque.Len + w.instLiquid.Len + w.instAdditive.Len + w.instTransSrc.Len
}

// GetStreamingVertexCount returns the current vertex count stored in the CPU streaming buffer.
func (w *BuilderVolume) GetStreamingVertexCount() int {
	_, _, vCount, _, _, _ := w.fv.GetVertices()
	return int(vCount) / gpu.StreamingVertexFloats
}

// GetHWOcclusion retrieves the hardware-based occlusion object associated with the BuilderVolume.
func (w *BuilderVolume) GetHWOcclusion() *OcclusionHW {
	return w.occlusion
}

// retrieveThingMesh returns a resident mesh for the given faces, reusing or rebuilding cached mesh data when needed.
// The returned bool reports whether the mesh was rebuilt during the current mesh frame.
func (w *BuilderVolume) retrieveThingMesh(rigidFaces *volumes.Faces) (*gpu.ResidentMesh, bool) {
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
	rm := gpu.NewResidentMesh(w.ctx)
	rm.LastSeenFrame = w.meshFrame
	rm.RebuiltFrame = w.meshFrame
	rm.Rebuild(rigidFaces, w.texLookupFn)
	w.thingMeshes[gh] = append(bucket, rm)
	return rm, true
}

// facesEquivalent reports whether two face collections contain the same ordered faces, materials, and points.
func facesEquivalent(a, b *volumes.Faces) bool {
	if a == b {
		return true
	}
	if a == nil || b == nil || a.Len() != b.Len() {
		return false
	}
	n := a.Len()
	for i := range n {
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
