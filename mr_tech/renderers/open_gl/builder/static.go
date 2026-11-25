package builder

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

// streamAtlasCellSize defines the size of each cell in the spatial atlas used for partitioning streaming chunks.
const streamAtlasCellSize = 512.0

// streamChunkKey represents a unique identifier for a stream-chunk based on its grid coordinates x and y.
type streamChunkKey struct {
	x, y int
}

// streamChunkEntry represents an entry in the streaming system for chunk data with mesh and volume face references.
type streamChunkEntry struct {
	atlas         *buffers.ResidentMesh
	faces         []*volumes.Faces
	lastSeenFrame uint64
}

// StaticVolumeEntry represents a static volume in the graphics pipeline containing mesh, material, and rendering state.
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

// Static is a structure designed for managing and caching 3D static volume data, meshes, and rendering state.
type Static struct {
	bc *Context

	visibleVol *VisibleVolumes
	volMeshes  map[*volumes.Faces]*buffers.ResidentMesh
	cSky       *textures.Material

	cachedRawVols    []*volumes.Volume
	cachedVolMeshes  []*buffers.ResidentMesh
	cachedRawVolsLen int
	cachedSky        *textures.Material

	gen                 uint64
	vertBytes           int
	idxBytes            int
	lastChunkEvictFrame uint64

	streamChunks map[streamChunkKey]*streamChunkEntry

	chunkQueryBox  *aabb.BoundingBox
	chunkQueryVols []*volumes.Volume

	atlas           *buffers.ResidentMesh
	entries         map[*volumes.Faces]*StaticVolumeEntry
	atlasBuilt      bool
	atlasStreamMode bool
	atlasVolCount   int
}

// NewBuilderStatic creates and returns a new Static builder using the provided Context for resource management.
func NewBuilderStatic(bc *Context) *Static {
	return &Static{
		bc:              bc,
		visibleVol:      NewVisibleVols(8192),
		cachedRawVols:   make([]*volumes.Volume, 8192),
		cachedVolMeshes: make([]*buffers.ResidentMesh, 8192),
		volMeshes:       make(map[*volumes.Faces]*buffers.ResidentMesh, 1024),
	}
}

// GetUploadHints returns the current static vertex and index byte counts along with the generation identifier.
func (w *Static) GetUploadHints() (staticVertBytes, staticIdxBytes int, staticGen uint64) {
	return w.vertBytes, w.idxBytes, w.gen
}

// GetSkyMaterial returns the material used for sky rendering in the Static object.
func (w *Static) GetSkyMaterial() *textures.Material {
	return w.cSky
}

// GetEntry retrieves a StaticVolumeEntry associated with the given Faces instance from the staticEntries map.
func (w *Static) GetEntry(faces *volumes.Faces) *StaticVolumeEntry {
	return w.entries[faces]
}

// GetResidentMesh returns the resident mesh associated with the given set of faces from the volume mesh mapping.
func (w *Static) GetResidentMesh(faces *volumes.Faces) *buffers.ResidentMesh {
	return w.volMeshes[faces]
}

// TryBuildStaticAtlas attempts to rebuild the static atlas if conditions like stream mode or volume count have changed.
func (w *Static) TryBuildStaticAtlas(vols *model.Volumes) {
	if vols == nil {
		return
	}
	hasStreamer := vols.HasStreamer()
	volsLen := vols.Len()
	if !w.atlasBuilt || w.atlasStreamMode != hasStreamer || (!hasStreamer && w.atlasVolCount != volsLen) {
		w.buildAtlas(vols)
	}
}

// Process updates the static mesh state, processes visible volumes, and manages dynamic and static draw operations.
func (w *Static) Process(vols *model.Volumes, frustumFront *aabb.Frustum, pX, pY, pZ float64) bool {
	if w.bc.meshFrame == 0 {
		w.bc.meshFrame = 1
	}
	if vols == nil {
		if w.cachedRawVolsLen == 0 && w.gen > 0 {
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
		w.vertBytes, w.idxBytes = w.bc.fv.FrozenBytes()
		w.gen++
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
				w.pushSubMesh(entry.atlas, entry.atlas.SubMeshes[sIdx])
			}
			continue
		}
		if rm != nil {
			for sIdx := 0; sIdx < rm.SubMeshesLen; sIdx++ {
				w.pushSubMesh(rm, rm.SubMeshes[sIdx])
			}
		}
	}
	// Record resolved primary mesh per cachedRawVols slot so external detaches (e.g. shadow ResolveMesh) invalidate canReuse.
	for i := range visLen {
		w.cachedVolMeshes[i] = w.primaryMeshFor(w.cachedRawVols[i])
	}

	w.cachedSky = w.cSky
	w.bc.FreezeDynamicDraws()
	w.vertBytes, w.idxBytes = w.bc.fv.FrozenBytes()
	w.gen++
	return true
}

// canReuseVisible determines if cached visible volumes and meshes can be reused based on version, state, and configuration.
func (w *Static) canReuseVisible(visLen int) bool {
	if w.gen == 0 || visLen != w.cachedRawVolsLen {
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
		if entry := w.entries[faces]; entry != nil && !entry.detached {
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

// primaryMeshFor retrieves the primary mesh for a given volume, prioritizing atlas entries over standalone meshes.
func (w *Static) primaryMeshFor(vol *volumes.Volume) *buffers.ResidentMesh {
	rm, entry, _ := w.LookupResolved(vol)
	if entry != nil && !entry.detached {
		return entry.atlas
	}
	return rm
}

// ensureCachedCapacity adjusts the capacity of cachedRawVols and cachedVolMeshes to accommodate at least visLen elements.
func (w *Static) ensureCachedCapacity(visLen int) {
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

// buildStreamChunkForVolume creates or updates a stream chunk entry for the given volume, organizing its mesh within an atlas.
func (w *Static) buildStreamChunkForVolume(vol *volumes.Volume, vols *model.Volumes) *StaticVolumeEntry {
	if vol == nil || vols == nil {
		return nil
	}
	targetFaces := vol.GetFaces()
	box := vol.GetAABB()
	if box == nil || targetFaces == nil {
		return nil
	}
	if w.entries == nil {
		w.entries = make(map[*volumes.Faces]*StaticVolumeEntry, 256)
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
			if e := w.entries[oldFaces]; e != nil && !e.detached && e.chunk == chunk {
				delete(w.entries, oldFaces)
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
	return w.entries[targetFaces]
}

// evictStaleStreamChunks removes outdated stream chunks from memory based on their age and clears associated resources.
func (w *Static) evictStaleStreamChunks() {
	const chunkMaxAgeFrames = 600
	for key, chunk := range w.streamChunks {
		if w.bc.meshFrame-chunk.lastSeenFrame <= chunkMaxAgeFrames {
			continue
		}
		for _, f := range chunk.faces {
			if e := w.entries[f]; e != nil && e.chunk == chunk {
				delete(w.entries, f)
			}
		}
		if chunk.atlas != nil {
			chunk.atlas.Dispose()
			chunk.atlas = nil
		}
		delete(w.streamChunks, key)
	}
}

// packVolumesIntoAtlas packs a set of volumes into a static mesh atlas for efficient rendering and memory usage.
// atlas represents the target ResidentMesh for packing the volumes into a static atlas.
// rawVols is the list of volumes to be packed into the atlas.
// chunk refers to the current streamChunkEntry associated with the packing operation.
func (w *Static) packVolumesIntoAtlas(atlas *buffers.ResidentMesh, rawVols []*volumes.Volume, chunk *streamChunkEntry) {
	totalFaces := 0
	for _, vol := range rawVols {
		if vol == nil {
			continue
		}
		if faces := vol.GetFaces(); faces != nil {
			if prev := w.entries[faces]; prev != nil && prev.detached {
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
		if prev := w.entries[faces]; prev != nil {
			if !prev.detached {
				vol.ClearDirty()
			}
			continue
		}
		subStart, subCount, skyMat, hasAnim, ok := atlas.AppendStaticFaces(faces, w.bc.LookupTexLayer)
		if hasAnim {
			// Remember animated volumes so streamed chunk builds do not retry packing them on subsequent frames.
			if chunk != nil {
				w.entries[faces] = &StaticVolumeEntry{detached: true}
			}
			continue
		}
		if ok || skyMat != nil {
			w.entries[faces] = &StaticVolumeEntry{
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

// pushSubMesh processes and organizes a given static sub-mesh according to its blend mode and other properties.
func (w *Static) pushSubMesh(rm *buffers.ResidentMesh, sub *buffers.ResidentSubMesh) {
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

// LookupResolved retrieves the mesh, volume entry, and sky material associated with the given volume, if available.
func (w *Static) LookupResolved(vol *volumes.Volume) (*buffers.ResidentMesh, *StaticVolumeEntry, *textures.Material) {
	if vol == nil {
		return nil, nil, nil
	}
	faces := vol.GetFaces()
	if faces == nil {
		return nil, nil, nil
	}
	if entry := w.entries[faces]; entry != nil && !entry.detached {
		return entry.atlas, entry, entry.skyMaterial
	}
	if rm := w.volMeshes[faces]; rm != nil {
		return rm, nil, rm.SkyMaterial
	}
	return nil, nil, nil
}

// ResolveMesh generates or retrieves a mesh representation, associating it with a volume and its material properties.
func (w *Static) ResolveMesh(vol *volumes.Volume, vols *model.Volumes) (*buffers.ResidentMesh, *StaticVolumeEntry, *textures.Material) {
	if vol == nil {
		return nil, nil, nil
	}
	faces := vol.GetFaces()
	if faces == nil {
		vol.ClearDirty()
		return nil, nil, nil
	}
	if entry := w.entries[faces]; entry != nil {
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

// buildAtlas initializes and constructs a static atlas by packing volumes into a mesh for rendering optimization.
func (w *Static) buildAtlas(vols *model.Volumes) {
	w.atlasBuilt = true
	w.atlasVolCount = 0
	for i := range w.cachedRawVolsLen {
		w.cachedRawVols[i] = nil
		w.cachedVolMeshes[i] = nil
	}
	w.cachedRawVolsLen = 0
	hasStreamer := vols.HasStreamer()
	volsLen := vols.Len()
	w.atlasStreamMode = hasStreamer
	w.disposeStreamChunks()
	if hasStreamer {
		w.atlas = nil
		w.atlasVolCount = volsLen
		w.entries = make(map[*volumes.Faces]*StaticVolumeEntry, 256)
		w.streamChunks = make(map[streamChunkKey]*streamChunkEntry, 64)
		return
	}
	w.atlasVolCount = volsLen
	if volsLen == 0 {
		w.atlas = nil
		w.entries = make(map[*volumes.Faces]*StaticVolumeEntry)
		return
	}
	rawVols := vols.GetVolumes().Slice()
	if w.atlas == nil {
		w.atlas = buffers.NewResidentMesh(w.bc.ctx)
	}
	w.entries = make(map[*volumes.Faces]*StaticVolumeEntry, len(rawVols))
	w.packVolumesIntoAtlas(w.atlas, rawVols, nil)
}

// disposeStreamChunks releases resources held by stream chunk entries and removes them from the streamChunks map.
func (w *Static) disposeStreamChunks() {
	for key, chunk := range w.streamChunks {
		if chunk.atlas != nil {
			chunk.atlas.Dispose()
			chunk.atlas = nil
		}
		chunk.faces = nil
		delete(w.streamChunks, key)
	}
}
