package open_gl

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/model"
	"github.com/markel1974/godoom/mr_tech/model/volumes"
	"github.com/markel1974/godoom/mr_tech/physics/aabb"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/gpu/buffers"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/gpu/draws"
	"github.com/markel1974/godoom/mr_tech/textures"
)

const streamAtlasCellSize = 512.0

type streamChunkKey struct {
	x, y int
}

type streamChunkEntry struct {
	atlas         *buffers.ResidentMesh
	faces         []*volumes.Faces
	lastSeenFrame uint64
}

// StaticVolumeEntry tracks a volume packed into either the world static atlas or a streamed chunk atlas.
type StaticVolumeEntry struct {
	atlas       *buffers.ResidentMesh
	chunk       *streamChunkEntry
	subStart    int
	subCount    int
	skyMaterial *textures.Material
	version     uint64
	faceCount   int
	detached    bool
}

// BuilderStatic manages visible world volumes, coalesced static atlases, streamed chunk atlases,
// and detached per-volume fallback meshes.
type BuilderStatic struct {
	bc *BuilderContext

	visibleVol *VisibleVolumes
	volMeshes  map[*volumes.Faces]*buffers.ResidentMesh
	cSky       *textures.Material

	cachedRawVols    []*volumes.Volume
	cachedVolMeshes  []*buffers.ResidentMesh
	cachedRawVolsLen int
	cachedSky        *textures.Material

	staticGen           uint64
	staticVertBytes     int
	staticIdxBytes      int
	lastChunkEvictFrame uint64

	streamChunks map[streamChunkKey]*streamChunkEntry

	chunkQueryBox  *aabb.BoundingBox
	chunkQueryVols []*volumes.Volume

	staticAtlas           *buffers.ResidentMesh
	staticEntries         map[*volumes.Faces]*StaticVolumeEntry
	staticAtlasBuilt      bool
	staticAtlasStreamMode bool
	staticAtlasVolCount   int
}

// NewBuilderStatic allocates a BuilderStatic bound to bc.
func NewBuilderStatic(bc *BuilderContext) *BuilderStatic {
	return &BuilderStatic{
		bc:              bc,
		visibleVol:      NewVisibleVols(8192),
		cachedRawVols:   make([]*volumes.Volume, 8192),
		cachedVolMeshes: make([]*buffers.ResidentMesh, 8192),
		volMeshes:       make(map[*volumes.Faces]*buffers.ResidentMesh, 1024),
	}
}

// GetUploadHints returns static vertex/index prefix byte sizes and the monotonic static generation counter.
func (w *BuilderStatic) GetUploadHints() (staticVertBytes, staticIdxBytes int, staticGen uint64) {
	return w.staticVertBytes, w.staticIdxBytes, w.staticGen
}

// GetSkyMaterial returns the sky material detected among active static volumes.
func (w *BuilderStatic) GetSkyMaterial() *textures.Material {
	return w.cSky
}

// GetEntry returns the static atlas descriptor for faces when mapped, or nil.
func (w *BuilderStatic) GetEntry(faces *volumes.Faces) *StaticVolumeEntry {
	return w.staticEntries[faces]
}

// GetResidentMesh returns the standalone detached resident mesh for faces, or nil.
func (w *BuilderStatic) GetResidentMesh(faces *volumes.Faces) *buffers.ResidentMesh {
	return w.volMeshes[faces]
}

// TryBuildStaticAtlas initializes or rebuilds the static/streaming atlas layout when volume storage mode or count changes.
func (w *BuilderStatic) TryBuildStaticAtlas(vols *model.Volumes) {
	if vols == nil {
		return
	}
	hasStreamer := vols.HasStreamer()
	volsLen := vols.Len()
	if !w.staticAtlasBuilt || w.staticAtlasStreamMode != hasStreamer || (!hasStreamer && w.staticAtlasVolCount != volsLen) {
		w.buildStaticAtlas(vols)
	}
}

// Process updates GPU-resident static volume meshes only when the visible volume set, mesh detachment, geometry, or animated textures change.
func (w *BuilderStatic) Process(vols *model.Volumes, frustumFront *aabb.Frustum, pX, pY, pZ float64) bool {
	if w.bc.meshFrame == 0 {
		w.bc.meshFrame = 1
	}
	if vols == nil {
		if w.cachedRawVolsLen == 0 && w.staticGen > 0 {
			w.bc.ResetDynamicDraws()
			return false
		}
		for i := range w.cachedRawVolsLen {
			w.cachedRawVols[i] = nil
			w.cachedVolMeshes[i] = nil
		}
		w.cachedRawVolsLen = 0
		w.bc.DeepResetDynamicDraws()
		w.bc.ResetStaticPools()
		w.cSky = nil
		w.cachedSky = nil
		w.bc.FreezeDynamicDraws()
		w.staticVertBytes, w.staticIdxBytes = w.bc.fv.FrozenBytes()
		w.staticGen++
		return true
	}

	w.TryBuildStaticAtlas(vols)
	hasStreamer := vols.HasStreamer()
	if hasStreamer && w.bc.meshFrame-w.lastChunkEvictFrame >= 256 {
		w.lastChunkEvictFrame = w.bc.meshFrame
		w.evictStaleStreamChunks()
	}

	volsLen := vols.Len()
	w.visibleVol.Reset(volsLen, pX, pY, pZ)
	if frustumFront != nil {
		vols.QueryFrustum(frustumFront, func(object aabb.IObject) bool {
			w.visibleVol.Add(object.(*volumes.Volume))
			return false
		})
	}

	visLen := w.visibleVol.Len()
	if w.canReuseVisible(visLen) {
		w.bc.ResetDynamicDraws()
		w.cSky = w.cachedSky
		return false
	}

	w.bc.DeepResetDynamicDraws()
	w.bc.ResetStaticPools()
	w.cSky = nil

	w.ensureCachedCapacity(visLen)
	w.cachedRawVolsLen = visLen
	for i := range visLen {
		w.cachedRawVols[i] = w.visibleVol.At(i)
	}

	// Sort front-to-back for early-Z static submission order.
	w.visibleVol.Sort()

	for vIdx := range visLen {
		w.ResolveMesh(w.visibleVol.At(vIdx), vols)
	}
	for vIdx := range visLen {
		vol := w.visibleVol.At(vIdx)
		rm, entry, skyMat := w.LookupResolved(vol)
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
	// Record resolved primary mesh per cachedRawVols slot so external detaches (e.g. shadow ResolveMesh) invalidate canReuse.
	for i := range visLen {
		w.cachedVolMeshes[i] = w.primaryMeshFor(w.cachedRawVols[i])
	}

	w.cachedSky = w.cSky
	w.bc.FreezeDynamicDraws()
	w.staticVertBytes, w.staticIdxBytes = w.bc.fv.FrozenBytes()
	w.staticGen++
	return true
}

func (w *BuilderStatic) canReuseVisible(visLen int) bool {
	if w.staticGen == 0 || visLen != w.cachedRawVolsLen {
		return false
	}
	for i := range visLen {
		vol := w.visibleVol.At(i)
		if vol != w.cachedRawVols[i] || vol.IsDirty() {
			return false
		}
		faces := vol.GetFaces()
		if faces == nil {
			if w.cachedVolMeshes[i] != nil {
				return false
			}
			continue
		}
		if entry := w.staticEntries[faces]; entry != nil && !entry.detached {
			if w.cachedVolMeshes[i] != entry.atlas || faces.GetVersion() != entry.version || faces.Len() != entry.faceCount {
				return false
			}
			if entry.chunk != nil {
				entry.chunk.lastSeenFrame = w.bc.meshFrame
			}
			continue
		}
		rm := w.volMeshes[faces]
		if rm == nil || w.cachedVolMeshes[i] != rm || rm.NeedsRebuild(faces, true) {
			return false
		}
	}
	return true
}

func (w *BuilderStatic) primaryMeshFor(vol *volumes.Volume) *buffers.ResidentMesh {
	rm, entry, _ := w.LookupResolved(vol)
	if entry != nil && !entry.detached {
		return entry.atlas
	}
	return rm
}

func (w *BuilderStatic) ensureCachedCapacity(visLen int) {
	if len(w.cachedRawVols) < visLen {
		newCap := max(len(w.cachedRawVols)*2, visLen)
		w.cachedRawVols = make([]*volumes.Volume, newCap)
		w.cachedVolMeshes = make([]*buffers.ResidentMesh, newCap)
		return
	}
	for i := visLen; i < w.cachedRawVolsLen; i++ {
		w.cachedRawVols[i] = nil
		w.cachedVolMeshes[i] = nil
	}
}

func (w *BuilderStatic) buildStreamChunkForVolume(vol *volumes.Volume, vols *model.Volumes) *StaticVolumeEntry {
	if vol == nil || vols == nil {
		return nil
	}
	targetFaces := vol.GetFaces()
	box := vol.GetAABB()
	if box == nil || targetFaces == nil {
		return nil
	}
	if w.staticEntries == nil {
		w.staticEntries = make(map[*volumes.Faces]*StaticVolumeEntry, 256)
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
			atlas: buffers.NewResidentMesh(w.bc.ctx),
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
			chunk.atlas = buffers.NewResidentMesh(w.bc.ctx)
		}
	}
	chunk.lastSeenFrame = w.bc.meshFrame

	w.packVolumesIntoAtlas(chunk.atlas, w.chunkQueryVols, chunk)
	if chunk.atlas.SubMeshesLen == 0 {
		chunk.atlas = nil
	}
	return w.staticEntries[targetFaces]
}

func (w *BuilderStatic) evictStaleStreamChunks() {
	const chunkMaxAgeFrames = 600
	for key, chunk := range w.streamChunks {
		if w.bc.meshFrame-chunk.lastSeenFrame <= chunkMaxAgeFrames {
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

func (w *BuilderStatic) packVolumesIntoAtlas(atlas *buffers.ResidentMesh, rawVols []*volumes.Volume, chunk *streamChunkEntry) {
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
		subStart, subCount, skyMat, hasAnim, ok := atlas.AppendStaticFaces(faces, w.bc.LookupTexLayer)
		if hasAnim {
			// Remember animated volumes so streamed chunk builds do not retry packing them on subsequent frames.
			if chunk != nil {
				w.staticEntries[faces] = &StaticVolumeEntry{detached: true}
			}
			continue
		}
		if ok || skyMat != nil {
			w.staticEntries[faces] = &StaticVolumeEntry{
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

func (w *BuilderStatic) pushStaticSubMesh(rm *buffers.ResidentMesh, sub *buffers.ResidentSubMesh) {
	if sub.IndexCount == 0 {
		return
	}
	switch sub.BlendMode {
	case config.BlendModeLiquid:
		item := w.bc.staticLiquid.Acquire()
		item.Mesh = rm
		item.Sub = sub
	case config.BlendModeAdditive:
		item := w.bc.staticAdditive.Acquire()
		item.Mesh = rm
		item.Sub = sub
	case config.BlendModeTransparent:
		item := w.bc.staticTransSrc.Acquire()
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
		item := w.bc.staticOpaque.Acquire()
		item.Mesh = rm
		item.Sub = sub
	}
}

// LookupResolved returns the resident mesh, active atlas entry (when attached), and optional sky material for vol.
func (w *BuilderStatic) LookupResolved(vol *volumes.Volume) (*buffers.ResidentMesh, *StaticVolumeEntry, *textures.Material) {
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

// ResolveMesh ensures vol's GPU mesh representation (atlas-packed or detached fallback) is built and up to date.
func (w *BuilderStatic) ResolveMesh(vol *volumes.Volume, vols *model.Volumes) (*buffers.ResidentMesh, *StaticVolumeEntry, *textures.Material) {
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
					entry.chunk.lastSeenFrame = w.bc.meshFrame
				}
				return entry.atlas, entry, entry.skyMaterial
			}
		}
	} else if vols != nil && vols.HasStreamer() {
		if entry = w.buildStreamChunkForVolume(vol, vols); entry != nil && !entry.detached {
			if entry.chunk != nil {
				entry.chunk.lastSeenFrame = w.bc.meshFrame
			}
			return entry.atlas, entry, entry.skyMaterial
		}
	}

	rm := w.volMeshes[faces]
	if rm == nil {
		rm = buffers.NewResidentMesh(w.bc.ctx)
		w.volMeshes[faces] = rm
	}
	rm.LastSeenFrame = w.bc.meshFrame
	if vol.IsDirty() || rm.NeedsRebuild(faces, true) {
		rm.Rebuild(faces, w.bc.LookupTexLayer)
		vol.ClearDirty()
	}
	return rm, nil, rm.SkyMaterial
}

func (w *BuilderStatic) buildStaticAtlas(vols *model.Volumes) {
	w.staticAtlasBuilt = true
	w.staticAtlasVolCount = 0
	for i := range w.cachedRawVolsLen {
		w.cachedRawVols[i] = nil
		w.cachedVolMeshes[i] = nil
	}
	w.cachedRawVolsLen = 0
	hasStreamer := vols.HasStreamer()
	volsLen := vols.Len()
	w.staticAtlasStreamMode = hasStreamer
	w.disposeStreamChunks()
	if hasStreamer {
		w.staticAtlas = nil
		w.staticAtlasVolCount = volsLen
		w.staticEntries = make(map[*volumes.Faces]*StaticVolumeEntry, 256)
		w.streamChunks = make(map[streamChunkKey]*streamChunkEntry, 64)
		return
	}
	w.staticAtlasVolCount = volsLen
	if volsLen == 0 {
		w.staticAtlas = nil
		w.staticEntries = make(map[*volumes.Faces]*StaticVolumeEntry)
		return
	}
	rawVols := vols.GetVolumes().Slice()
	if w.staticAtlas == nil {
		w.staticAtlas = buffers.NewResidentMesh(w.bc.ctx)
	}
	w.staticEntries = make(map[*volumes.Faces]*StaticVolumeEntry, len(rawVols))
	w.packVolumesIntoAtlas(w.staticAtlas, rawVols, nil)
}

func (w *BuilderStatic) disposeStreamChunks() {
	for key, chunk := range w.streamChunks {
		if chunk.atlas != nil {
			chunk.atlas.Dispose()
			chunk.atlas = nil
		}
		chunk.faces = nil
		delete(w.streamChunks, key)
	}
}
