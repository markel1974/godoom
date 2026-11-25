package open_gl

import (
	"math"
	"unsafe"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/engine"
	"github.com/markel1974/godoom/mr_tech/model"
	"github.com/markel1974/godoom/mr_tech/model/things"
	"github.com/markel1974/godoom/mr_tech/model/volumes"
	"github.com/markel1974/godoom/mr_tech/physics/aabb"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/gpu/buffers"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/gpu/draws"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/gpu/queries"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/gpu/vramtex"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/metrics"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/shaders"
	"github.com/markel1974/godoom/mr_tech/textures"
)

type RigidInstance struct {
	cx, cy, cz     float64
	qx, qy, qz, qw float64
}

type RigidInstanceGroup struct {
	mesh           *buffers.ResidentMesh
	instances      buffers.PtrPool[RigidInstance]
	baseByteOffset int32
}

type RigidTransItem struct {
	mesh           *buffers.ResidentMesh
	sub            *buffers.ResidentSubMesh
	baseByteOffset int32
	cx, cy, cz     float32
}

type streamChunkEntry struct {
	atlas         *buffers.ResidentMesh
	faces         []*volumes.Faces
	lastSeenFrame uint64
}

type staticVolumeEntry struct {
	atlas       *buffers.ResidentMesh
	chunk       *streamChunkEntry
	subStart    int
	subCount    int
	skyMaterial *textures.Material
	version     uint64
	faceCount   int
	detached    bool
}

// BuilderVolume represents a structure that consolidates textures, frame vertices, draw commands, and other rendering resources.
type BuilderVolume struct {
	ctx           api.IContext
	tex           *vramtex.Textures
	fv            *draws.FrameVertices
	dcOpaque      *draws.DrawCommands
	dcAdditive    *draws.DrawCommands
	dcLiquid      *draws.DrawCommands
	dcTransparent *draws.DrawCommands

	dcRenderOpaque      *draws.DrawCommandsRender
	dcRenderAdditive    *draws.DrawCommandsRender
	dcRenderLiquid      *draws.DrawCommandsRender
	dcRenderTransparent *draws.DrawCommandsRender
	cSky                *textures.Material
	cal                 *model.Calibration
	visibleVol          *VisibleVolumes
	modes               []*draws.DrawCommands
	occlusion           *queries.OcclusionHW
	frustumFront        *metrics.Frustum

	playerId uint64
	player   *things.Player

	cachedRawVols    []*volumes.Volume
	cachedRawVolsLen int
	cachedSky        *textures.Material
	staticGen        uint64
	staticVertBytes  int
	staticIdxBytes   int
	frameGen         uint64
	lastPreparedGen  uint64

	thingsRender *ThingsRender

	volMeshes             map[*volumes.Faces]*buffers.ResidentMesh
	staticAtlas           *buffers.ResidentMesh
	staticEntries         map[*volumes.Faces]*staticVolumeEntry
	staticAtlasBuilt      bool
	staticAtlasStreamMode bool
	staticAtlasVolCount   int
	streamChunks          map[streamChunkKey]*streamChunkEntry
	chunkQueryBox         *aabb.BoundingBox
	chunkQueryVols        []*volumes.Volume
	thingMeshes           map[uint64][]*buffers.ResidentMesh
	thingMeshesByPtr      map[*volumes.Faces]thingMeshPtrEntry
	texLookupFn           func(*textures.Texture) (float32, bool)

	staticOpaque   buffers.PtrPool[draws.DrawItemStatic]
	staticLiquid   buffers.PtrPool[draws.DrawItemStatic]
	staticAdditive buffers.PtrPool[draws.DrawItemStatic]
	staticTransSrc buffers.PtrPool[draws.DrawItemTransparent]

	instancePool     *buffers.InstancePool
	instGroups       buffers.PtrPool[RigidInstanceGroup]
	instFrame        uint64
	instOpaque       buffers.PtrPool[draws.DrawItemInstanced]
	instLiquid       buffers.PtrPool[draws.DrawItemInstanced]
	instAdditive     buffers.PtrPool[draws.DrawItemInstanced]
	instTransSrc     buffers.PtrPool[RigidTransItem]
	lastInstVBO      uint32
	transparentItems *draws.DrawItemsTransparent

	shadowTiles        [shaders.MaxShadowTiles]shaders.ShadowTileData
	shadowBuilders     [shaders.MaxShadowTiles]*shadowTileBuilder
	shadowInstPool     *buffers.InstancePool
	shadowInstGroups   buffers.PtrPool[shadowRigidGroup]
	shadowGroupToken   uint64
	shadowFrustum      *aabb.Frustum
	shadowVolScratch   []*volumes.Volume
	shadowThingScratch []shadowCollectedThing
	shadowStreamCache  shadowThingCache
	shadowRigidCache   shadowRigidCache
	prevShadowExtraSig uint64

	lastCamGLX  float32
	lastCamGLY  float32
	lastCamGLZ  float32
	meshFrame   uint64
	shadowFrame uint64

	bl *BuilderLights
}

type thingMeshPtrEntry struct {
	rm        *buffers.ResidentMesh
	geomHash  uint64
	version   uint64
	faceCount int
}

const streamAtlasCellSize = 512.0

type streamChunkKey struct {
	x, y int
}

// NewBuilderVolume initializes and returns a new BuilderVolume instance with configured textures and calibration settings.
func NewBuilderVolume(ctx api.IContext, tex *vramtex.Textures, cal *model.Calibration, player *things.Player) *BuilderVolume {
	bv := &BuilderVolume{
		ctx:                 ctx,
		tex:                 tex,
		player:              player,
		playerId:            player.GetEntity().GetId(),
		fv:                  draws.NewFrameVertices(ctx, 1048576),
		dcOpaque:            draws.NewDrawCommands(config.BlendModeOpaque, 32768),
		dcAdditive:          draws.NewDrawCommands(config.BlendModeAdditive, 4096),
		dcLiquid:            draws.NewDrawCommands(config.BlendModeLiquid, 4096),
		dcTransparent:       draws.NewDrawCommands(config.BlendModeTransparent, 4096),
		dcRenderOpaque:      draws.NewDrawCommandsRender(ctx, "opaque", false),
		dcRenderAdditive:    draws.NewDrawCommandsRender(ctx, "additive", true),
		dcRenderLiquid:      draws.NewDrawCommandsRender(ctx, "liquid", true),
		dcRenderTransparent: draws.NewDrawCommandsRender(ctx, "transparent", false),
		cSky:                nil,
		cal:                 cal,
		visibleVol:          NewVisibleVols(8192),
		modes:               make([]*draws.DrawCommands, config.BlendModeLatest),
		occlusion:           queries.NewOcclusionHW(ctx, 4096),
		frustumFront:        metrics.NewFrustum(),
		cachedRawVols:       make([]*volumes.Volume, 8192),
		thingsRender:        NewThingsRender(),
		volMeshes:           make(map[*volumes.Faces]*buffers.ResidentMesh, 1024),
		thingMeshes:         make(map[uint64][]*buffers.ResidentMesh, 256),
		thingMeshesByPtr:    make(map[*volumes.Faces]thingMeshPtrEntry, 512),
		instancePool:        buffers.NewInstancePool(ctx, 256),
		transparentItems:    draws.NewDrawItemsTransparent(64),
		bl:                  NewBuilderLights(ctx, cal),
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
	return w.bl.GetFrameLights().GetShadowLights()
}

// GetVerticesStride returns the byte stride of the vertex data by delegating the computation to the underlying FrameVertices object.
func (w *BuilderVolume) GetVerticesStride() int32 { return w.fv.VerticesStride() }

// GetLightsStride returns the stride size of the lights buffer in bytes.
func (w *BuilderVolume) GetLightsStride() int32 { return w.bl.GetFrameLights().LightsStride() }

// GetDrawCommandsOpaque returns the prepared DrawCommandsRender object containing batched GPU drawing commands for rendering.
func (w *BuilderVolume) GetDrawCommandsOpaque() *draws.DrawCommandsRender {
	return w.dcRenderOpaque
}

// GetVertices retrieves the vertex buffer, vertex count, index buffer, and index count from the builder volume.
func (w *BuilderVolume) GetVertices() (unsafe.Pointer, int, int32, unsafe.Pointer, int, int32) {
	return w.fv.GetVertices()
}

// GetLights retrieves the light data and their count from the frame lights. It returns a slice of float32 and an int32 count.
func (w *BuilderVolume) GetLights() (unsafe.Pointer, int32, int32) {
	return w.bl.GetFrameLights().GetLights()
}

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
	sizeZLight := w.cal.ZLightFar

	w.occlusion.Reset()

	staticChanged := w.pushQVolumes(engine.GetVolumes(), frustumFront, px, py, pz)

	w.bl.Process(engine.GetLights(), frustumFront, px, py, pz, sizeZLight, sizeZLight, sizeZLight)

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
		w.volMeshes = make(map[*volumes.Faces]*buffers.ResidentMesh, 256)
	}
	if w.texLookupFn == nil {
		w.texLookupFn = w.lookupTexLayer
	}
	volsLen := vols.Len()
	hasStreamer := vols.HasStreamer()
	if !w.staticAtlasBuilt || w.staticAtlasStreamMode != hasStreamer || (!hasStreamer && w.staticAtlasVolCount != volsLen) {
		w.buildStaticAtlas(vols)
	}
	if hasStreamer && w.meshFrame > 0 && w.meshFrame&255 == 0 {
		w.evictStaleStreamChunks()
	}
	w.visibleVol.Reset(volsLen, pX, pY, pZ)

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
				if entry.chunk != nil {
					entry.chunk.lastSeenFrame = w.meshFrame
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
		w.resolveVolumeMesh(w.visibleVol.At(vIdx), vols)
	}
	for vIdx := range visLen {
		vol := w.visibleVol.At(vIdx)
		rm, entry, skyMat := w.lookupResolvedVolume(vol)
		if skyMat != nil {
			w.cSky = skyMat
		}
		if entry != nil && !entry.detached {
			for sIdx := entry.subStart; sIdx < entry.subStart+entry.subCount; sIdx++ {
				w.pushStaticSubMesh(entry.atlas, entry.atlas.SubMeshes[sIdx])
			}
			continue
		}
		if rm != nil {
			for sIdx := 0; sIdx < rm.SubMeshesLen; sIdx++ {
				w.pushStaticSubMesh(rm, rm.SubMeshes[sIdx])
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

func (w *BuilderVolume) lookupResolvedVolume(vol *volumes.Volume) (*buffers.ResidentMesh, *staticVolumeEntry, *textures.Material) {
	if vol == nil {
		return nil, nil, nil
	}
	faces := vol.GetFaces()
	if faces == nil {
		return nil, nil, nil
	}
	if entry := w.staticEntries[faces]; entry != nil && !entry.detached {
		return entry.atlas, entry, entry.skyMaterial
	}
	if rm := w.volMeshes[faces]; rm != nil {
		return rm, nil, rm.SkyMaterial
	}
	return nil, nil, nil
}

func (w *BuilderVolume) resolveVolumeMesh(vol *volumes.Volume, vols *model.Volumes) (*buffers.ResidentMesh, *staticVolumeEntry, *textures.Material) {
	if vol == nil {
		return nil, nil, nil
	}
	faces := vol.GetFaces()
	if faces == nil {
		vol.ClearDirty()
		return nil, nil, nil
	}
	if entry := w.staticEntries[faces]; entry != nil {
		if !entry.detached {
			if vol.IsDirty() || faces.GetVersion() != entry.version || faces.Len() != entry.faceCount {
				entry.detached = true
			} else {
				if entry.chunk != nil {
					entry.chunk.lastSeenFrame = w.meshFrame
				}
				return entry.atlas, entry, entry.skyMaterial
			}
		}
	} else if vols != nil && vols.HasStreamer() {
		if entry = w.buildStreamChunkForVolume(vol, vols); entry != nil && !entry.detached {
			if entry.chunk != nil {
				entry.chunk.lastSeenFrame = w.meshFrame
			}
			return entry.atlas, entry, entry.skyMaterial
		}
	}

	rm := w.volMeshes[faces]
	if rm == nil {
		rm = buffers.NewResidentMesh(w.ctx)
		w.volMeshes[faces] = rm
	}
	rm.LastSeenFrame = w.meshFrame
	if vol.IsDirty() || rm.NeedsRebuild(faces, true) {
		rm.Rebuild(faces, w.texLookupFn)
		vol.ClearDirty()
	}
	return rm, nil, rm.SkyMaterial
}

func (w *BuilderVolume) disposeStreamChunks() {
	for key, chunk := range w.streamChunks {
		if chunk.atlas != nil {
			chunk.atlas.Dispose()
			chunk.atlas = nil
		}
		chunk.faces = nil
		delete(w.streamChunks, key)
	}
}

func (w *BuilderVolume) evictStaleStreamChunks() {
	const chunkMaxAgeFrames = 600
	for key, chunk := range w.streamChunks {
		if w.meshFrame-chunk.lastSeenFrame <= chunkMaxAgeFrames {
			continue
		}
		for _, f := range chunk.faces {
			if e := w.staticEntries[f]; e != nil && e.chunk == chunk {
				delete(w.staticEntries, f)
			}
		}
		if chunk.atlas != nil {
			chunk.atlas.Dispose()
			chunk.atlas = nil
		}
		delete(w.streamChunks, key)
	}
}

func (w *BuilderVolume) packVolumesIntoAtlas(atlas *buffers.ResidentMesh, rawVols []*volumes.Volume, chunk *streamChunkEntry) {
	totalFaces := 0
	for _, vol := range rawVols {
		if vol == nil {
			continue
		}
		if faces := vol.GetFaces(); faces != nil {
			if prev := w.staticEntries[faces]; prev != nil && prev.detached {
				continue
			}
			totalFaces += faces.Len()
		}
	}
	if totalFaces == 0 {
		if chunk != nil && atlas != nil {
			atlas.Dispose()
		}
		return
	}
	atlas.ReserveStaticAtlas(totalFaces)
	atlas.BeginStaticAtlas()
	for _, vol := range rawVols {
		if vol == nil {
			continue
		}
		faces := vol.GetFaces()
		if faces == nil || faces.Len() == 0 {
			continue
		}
		if prev := w.staticEntries[faces]; prev != nil {
			if !prev.detached {
				vol.ClearDirty()
			}
			continue
		}
		subStart, subCount, skyMat, hasAnim, ok := atlas.AppendStaticFaces(faces, w.texLookupFn)
		if hasAnim {
			// Remember animated volumes so streamed chunk builds do not retry packing them on subsequent frames.
			if chunk != nil {
				w.staticEntries[faces] = &staticVolumeEntry{detached: true}
			}
			continue
		}
		if ok || skyMat != nil {
			w.staticEntries[faces] = &staticVolumeEntry{
				atlas:       atlas,
				chunk:       chunk,
				subStart:    subStart,
				subCount:    subCount,
				skyMaterial: skyMat,
				version:     faces.GetVersion(),
				faceCount:   faces.Len(),
			}
			if chunk != nil {
				chunk.faces = append(chunk.faces, faces)
			}
			vol.ClearDirty()
		}
	}
	if atlas.SubMeshesLen > 0 {
		atlas.EndStaticAtlas()
	} else if chunk != nil {
		atlas.Dispose()
	}
}

func (w *BuilderVolume) buildStreamChunkForVolume(vol *volumes.Volume, vols *model.Volumes) *staticVolumeEntry {
	if vol == nil || vols == nil {
		return nil
	}
	targetFaces := vol.GetFaces()
	box := vol.GetAABB()
	if box == nil || targetFaces == nil {
		return nil
	}
	if w.staticEntries == nil {
		w.staticEntries = make(map[*volumes.Faces]*staticVolumeEntry, 256)
	}
	if w.streamChunks == nil {
		w.streamChunks = make(map[streamChunkKey]*streamChunkEntry, 64)
	}
	if w.chunkQueryBox == nil {
		w.chunkQueryBox = aabb.NewBoundingBox()
	}

	midX, midY, _ := box.GetCentroid()
	cx := int(math.Floor(midX / streamAtlasCellSize))
	cy := int(math.Floor(midY / streamAtlasCellSize))
	key := streamChunkKey{x: cx, y: cy}

	minX := float64(cx) * streamAtlasCellSize
	minY := float64(cy) * streamAtlasCellSize
	w.chunkQueryBox.Rebuild(minX, minY, -1e12, minX+streamAtlasCellSize, minY+streamAtlasCellSize, 1e12)

	w.chunkQueryVols = w.chunkQueryVols[:0]
	foundTarget := false
	vols.QueryAABB(w.chunkQueryBox, func(v *volumes.Volume) {
		if v == nil {
			return
		}
		vb := v.GetAABB()
		if vb == nil {
			return
		}
		vx, vy, _ := vb.GetCentroid()
		if int(math.Floor(vx/streamAtlasCellSize)) != cx || int(math.Floor(vy/streamAtlasCellSize)) != cy {
			return
		}
		if v == vol {
			foundTarget = true
		}
		w.chunkQueryVols = append(w.chunkQueryVols, v)
	})
	if !foundTarget {
		w.chunkQueryVols = append(w.chunkQueryVols, vol)
	}

	chunk := w.streamChunks[key]
	if chunk == nil {
		chunk = &streamChunkEntry{
			atlas: buffers.NewResidentMesh(w.ctx),
		}
		w.streamChunks[key] = chunk
	} else {
		for _, oldFaces := range chunk.faces {
			if e := w.staticEntries[oldFaces]; e != nil && !e.detached && e.chunk == chunk {
				delete(w.staticEntries, oldFaces)
			}
		}
		chunk.faces = chunk.faces[:0]
		if chunk.atlas == nil {
			chunk.atlas = buffers.NewResidentMesh(w.ctx)
		}
	}
	chunk.lastSeenFrame = w.meshFrame

	w.packVolumesIntoAtlas(chunk.atlas, w.chunkQueryVols, chunk)
	if chunk.atlas.SubMeshesLen == 0 {
		chunk.atlas = nil
	}
	return w.staticEntries[targetFaces]
}

func (w *BuilderVolume) buildStaticAtlas(vols *model.Volumes) {
	w.staticAtlasBuilt = true
	w.staticAtlasVolCount = 0
	hasStreamer := vols.HasStreamer()
	volsLen := vols.Len()
	w.staticAtlasStreamMode = hasStreamer
	w.disposeStreamChunks()
	if hasStreamer {
		w.staticAtlas = nil
		w.staticAtlasVolCount = vols.Len()
		w.staticEntries = make(map[*volumes.Faces]*staticVolumeEntry, 256)
		w.streamChunks = make(map[streamChunkKey]*streamChunkEntry, 64)
		return
	}
	w.staticAtlasVolCount = volsLen
	if volsLen == 0 {
		return
	}
	rawVols := vols.GetVolumes().Slice()
	if w.staticAtlas == nil {
		w.staticAtlas = buffers.NewResidentMesh(w.ctx)
	}
	w.staticEntries = make(map[*volumes.Faces]*staticVolumeEntry, len(rawVols))
	w.packVolumesIntoAtlas(w.staticAtlas, rawVols, nil)
}

func (w *BuilderVolume) pushStaticSubMesh(rm *buffers.ResidentMesh, sub *buffers.ResidentSubMesh) {
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
		item.Kind = draws.TransparentStatic
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

// GetDrawCommandsAdditive returns the additive draw commands prepared for rendering.
func (w *BuilderVolume) GetDrawCommandsAdditive() *draws.DrawCommandsRender {
	return w.dcRenderAdditive
}

// GetDrawCommandsLiquid returns the liquid draw commands prepared for rendering.
func (w *BuilderVolume) GetDrawCommandsLiquid() *draws.DrawCommandsRender {
	return w.dcRenderLiquid
}

// GetDrawCommandsTransparent returns the sorted transparent draw commands prepared for rendering.
func (w *BuilderVolume) GetDrawCommandsTransparent() *draws.DrawCommandsRender {
	return w.dcRenderTransparent
}

// GetOpaqueInstancedItems returns the active rigid instanced opaque draw items for inspection.
func (w *BuilderVolume) GetOpaqueInstancedItems() []*draws.DrawItemInstanced {
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
	return int(vCount) / buffers.StreamingVertexFloats
}

// GetHWOcclusion retrieves the hardware-based occlusion object associated with the BuilderVolume.
func (w *BuilderVolume) GetHWOcclusion() *queries.OcclusionHW {
	return w.occlusion
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
