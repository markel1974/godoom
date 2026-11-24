package open_gl

import (
	"math"
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

// BuilderVolume represents a structure that consolidates textures, frame vertices, draw commands, and other rendering resources.
type BuilderVolume struct {
	ctx                 api.IContext
	tex                 *vramtex.Textures
	fv                  *draws.FrameVertices
	dcOpaque            *draws.DrawCommands
	dcAdditive          *draws.DrawCommands
	dcLiquid            *draws.DrawCommands
	dcTransparent       *draws.DrawCommands
	fl                  *FrameLights
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
	shadowInstGroups   buffers.PtrPool[RigidInstanceGroup]
	shadowGroupToken   uint64
	shadowFrustum      *aabb.Frustum
	shadowVolScratch   []*volumes.Volume
	shadowThingScratch []ithing.IThing
	shadowStreamCache  shadowThingCache
	prevShadowExtraSig uint64

	lastCamGLX float32
	lastCamGLY float32
	lastCamGLZ float32
	meshFrame  uint64
}

const streamAtlasCellSize = 512.0

type streamChunkKey struct {
	x, y int
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
		fl:                  NewFrameLights(ctx, 1024, cal.EnableShadowLights),
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
		lightBB:             aabb.NewBoundingBox(),
		cachedRawVols:       make([]*volumes.Volume, 8192),
		thingsRender:        NewThingsRender(),
		volMeshes:           make(map[*volumes.Faces]*buffers.ResidentMesh, 1024),
		thingMeshes:         make(map[uint64][]*buffers.ResidentMesh, 256),
		instancePool:        buffers.NewInstancePool(ctx, 256),
		transparentItems:    draws.NewDrawItemsTransparent(64),
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
func (w *BuilderVolume) GetDrawCommandsOpaque() *draws.DrawCommandsRender {
	return w.dcRenderOpaque
}

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
		w.volMeshes = make(map[*volumes.Faces]*buffers.ResidentMesh, 256)
	}
	if w.texLookupFn == nil {
		w.texLookupFn = w.lookupTexLayer
	}
	hasStreamer := vols != nil && vols.HasStreamer()
	if !w.staticAtlasBuilt || w.staticAtlasStreamMode != hasStreamer || (!hasStreamer && vols != nil && w.staticAtlasVolCount != vols.Len()) {
		w.buildStaticAtlas(vols)
	}
	if hasStreamer && w.meshFrame > 0 && w.meshFrame&255 == 0 {
		w.evictStaleStreamChunks()
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
	hasStreamer := vols != nil && vols.HasStreamer()
	w.staticAtlasStreamMode = hasStreamer
	w.disposeStreamChunks()
	if hasStreamer {
		w.staticAtlas = nil
		if vols != nil {
			w.staticAtlasVolCount = vols.Len()
		}
		w.staticEntries = make(map[*volumes.Faces]*staticVolumeEntry, 256)
		w.streamChunks = make(map[streamChunkKey]*streamChunkEntry, 64)
		return
	}
	if vols == nil {
		return
	}
	w.staticAtlasVolCount = vols.Len()
	if vols.GetVolumes() == nil || vols.Len() == 0 {
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

// pushQThings processes visible things: rigid props/vehicles/buildings route into GPU instancing,
// while morphing 3D models, billboards, and UI elements route through the streaming vertex cache.
func (w *BuilderVolume) pushQThings(things *model.Things, frustumFront *aabb.Frustum) bool {
	if w.thingsRender == nil {
		w.thingsRender = NewThingsRender()
	}
	if w.thingMeshes == nil {
		w.thingMeshes = make(map[uint64][]*buffers.ResidentMesh, 256)
	}
	if w.meshFrame == 0 {
		w.meshFrame = 1
	}
	if w.instancePool == nil {
		w.instancePool = buffers.NewInstancePool(w.ctx, 256)
	}
	if w.texLookupFn == nil {
		w.texLookupFn = w.lookupTexLayer
	}
	w.thingsRender.Prepare()
	w.instFrame++
	w.instGroups.Reset()

	tick := textures.GlobalTick()

	things.StagePrepare()

	addRigidInstance := func(rm *buffers.ResidentMesh, cx, cy, cz, qx, qy, qz, qw float64) {
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
		ent := thing.GetEntity()
		entry2 := w.thingsRender.Retrieve(ent)
		thing.Compile(tick)
		entry2.compileFrame = w.meshFrame

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

		if !w.buildNonRigidThingCache(thing, entry2) {
			return
		}
		entry2.committedBaseIdx = w.fv.CommitWithOffset(entry2.GetFrameVertices(), w.modes)
		entry2.committedFrame = w.meshFrame
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
				w.instancePool.WriteTransformAt(baseFloat+k*buffers.InstanceMatrixFloats, inst.cx, inst.cy, inst.cz, inst.qx, inst.qy, inst.qz, inst.qw)
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
						rt.baseByteOffset = g.baseByteOffset + int32(k*buffers.InstanceMatrixFloats*4)
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

func (w *BuilderVolume) buildNonRigidThingCache(thing ithing.IThing, entry2 *ThingRenderCache) bool {
	pFaces, pNextFaces, lp, renderMode := thing.GetUIFaces()
	if pFaces == nil || pFaces.Len() == 0 {
		return false
	}

	lerp := float32(lp)
	yaw := float32(thing.GetAngle())
	if renderMode == vertices.RenderModeModel3D {
		yaw = 0.0
	}
	renderModeF := float32(renderMode)
	ent := thing.GetEntity()
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
	if entry2.Equal(pFaces, pNextFaces, facesVer, nextVer, facesLen, nextLen, lerp, yaw, renderModeF, oX, oY, oZ) {
		return true
	}

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
			centSumX += float64(p0X+p1X+p2X) / 3.0
			centSumY += float64(p0Z+p1Z+p2Z) / 3.0
			centSumZ += -float64(p0Y+p1Y+p2Y) / 3.0
		} else {
			centSumX += float64(oX)
			centSumY += float64(oY)
			centSumZ += float64(oZ)
		}
		centCount++

		var np0X, np0Y, np0Z, np1X, np1Y, np1Z, np2X, np2Y, np2Z float32
		if pNextFaces != nil {
			if nf := pNextFaces.At(fx); nf != nil {
				np0X, np0Y, np0Z, np1X, np1Y, np1Z, np2X, np2Y, np2Z = nf.GetPointsFlat()
			} else {
				np0X, np0Y, np0Z, np1X, np1Y, np1Z, np2X, np2Y, np2Z = p0X, p0Y, p0Z, p1X, p1Y, p1Z, p2X, p2Y, p2Z
			}
		} else {
			np0X, np0Y, np0Z, np1X, np1Y, np1Z, np2X, np2Y, np2Z = p0X, p0Y, p0Z, p1X, p1Y, p1Z, p2X, p2Y, p2Z
		}

		id0 := fv.AddVertexFull(p0X, p0Z, -p0Y, u0, -v0, tId, nx0, nz0, -ny0, tx0, tz0, -ty0, s0, lmU[0], lmV[0], lmIDFloat, oX, oY, oZ, renderModeF, np0X, np0Z, -np0Y, lerp, yaw, packed, spec, shininess, detailScale, detailFactor, envStr, envTint)
		id1 := fv.AddVertexFull(p1X, p1Z, -p1Y, u1, -v1, tId, nx1, nz1, -ny1, tx1, tz1, -ty1, s1, lmU[1], lmV[1], lmIDFloat, oX, oY, oZ, renderModeF, np1X, np1Z, -np1Y, lerp, yaw, packed, spec, shininess, detailScale, detailFactor, envStr, envTint)
		id2 := fv.AddVertexFull(p2X, p2Z, -p2Y, u2, -v2, tId, nx2, nz2, -ny2, tx2, tz2, -ty2, s2, lmU[2], lmV[2], lmIDFloat, oX, oY, oZ, renderModeF, np2X, np2Z, -np2Y, lerp, yaw, packed, spec, shininess, detailScale, detailFactor, envStr, envTint)
		fv.AddTriangle(id0, id1, id2)
	}

	flushBatch()
	entry2.Assign(pFaces, pNextFaces, facesVer, nextVer, facesLen, nextLen, lerp, yaw, renderModeF, oX, oY, oZ)
	return true
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

// retrieveThingMesh returns a resident mesh for the given faces, reusing or rebuilding cached mesh data when needed.
// The returned bool reports whether the mesh was rebuilt during the current mesh frame.
func (w *BuilderVolume) retrieveThingMesh(rigidFaces *volumes.Faces) (*buffers.ResidentMesh, bool) {
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
	rm := buffers.NewResidentMesh(w.ctx)
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
