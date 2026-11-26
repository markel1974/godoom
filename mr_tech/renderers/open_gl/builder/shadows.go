package builder

import (
	"math/bits"
	"unsafe"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/model"
	"github.com/markel1974/godoom/mr_tech/model/ithing"
	"github.com/markel1974/godoom/mr_tech/model/vertices"
	"github.com/markel1974/godoom/mr_tech/model/volumes"
	"github.com/markel1974/godoom/mr_tech/physics/aabb"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/gpu/buffers"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/shaders"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// ShadowsCollectedVol records a volume that has contributed shadow data along with its collection mask.
type ShadowsCollectedVol struct {
	vol  *volumes.Volume
	mask uint32
}

func (v *ShadowsCollectedVol) Assign(vol *volumes.Volume, mask uint32) {
	v.vol = vol
	v.mask = mask
}

// ShadowsCollectedThing records a shadow-casting thing collected for rendering and its tile coverage.
type ShadowsCollectedThing struct {
	thing   ithing.IThing
	entry   *ShadowsRigidEntry
	isRigid bool
	mask    uint32
}

func (t *ShadowsCollectedThing) Assign(thing ithing.IThing, entry *ShadowsRigidEntry, isRigid bool, mask uint32) {
	t.thing = thing
	t.entry = entry
	t.isRigid = isRigid
	t.mask = mask
}

// ShadowTileDeps records dependency stamps used to validate cached shadow tile reuse.
type ShadowTileDeps struct {
	valid      bool
	atlasGen   uint64
	sceneHash  uint64
	worldToken uint64
	tick       uint64
}

func (std *ShadowTileDeps) Assign(valid bool, atlasGen uint64, sceneHash uint64, worldToken uint64, tick uint64) {
	std.valid = valid
	std.atlasGen = atlasGen
	std.sceneHash = sceneHash
	std.worldToken = worldToken
	std.tick = tick
}

// Shadows manages shadow tile data, builders, and cached render state for generating shadow maps.
type Shadows struct {
	bContext           *Context
	bStatic            *Static
	bThings            *Things
	tiles              [shaders.MaxShadowTiles]shaders.ShadowsTileData
	tileDeps           [2][shaders.MaxShadowTiles]ShadowTileDeps
	builders           [shaders.MaxShadowTiles]*ShadowTileBuilder
	tileFrustums       [shaders.MaxShadowTiles]aabb.Frustum
	instPool           *buffers.InstancePool
	instGroups         buffers.PtrPool[ShadowsRigidGroup]
	groupToken         uint64
	volScratch         []ShadowsCollectedVol
	volScratchLen      int
	thingScratch       []ShadowsCollectedThing
	thingScratchLen    int
	entityCache        ShadowRigidCache
	prevShadowExtraSig uint64
	shadowFrame        uint64
}

// NewShadows creates a Shadows builder using the provided context, static geometry, and thing sources.
func NewShadows(bContext *Context, bStatic *Static, bThing *Things) *Shadows {
	var ctx api.IContext
	if bContext != nil {
		ctx = bContext.ctx
	}
	bs := &Shadows{
		bContext:        bContext,
		bStatic:         bStatic,
		bThings:         bThing,
		instPool:        buffers.NewInstancePool(ctx, 256),
		volScratch:      make([]ShadowsCollectedVol, 256),
		volScratchLen:   0,
		thingScratch:    make([]ShadowsCollectedThing, 256),
		thingScratchLen: 0,
	}
	bs.instGroups.Init(32)
	for t := range shaders.MaxShadowTiles {
		bs.builders[t] = NewShadowTileBuilder(ctx)
	}
	return bs
}

// DisableAllTiles marks every shadow tile inactive and clears its reuse state.
func (w *Shadows) DisableAllTiles() {
	for t := range shaders.MaxShadowTiles {
		w.tiles[t].SetActive(false)
		w.tiles[t].SetReuse(false)
	}
}

// GetTiles returns the shadow tile data array used by the shadow atlas.
func (w *Shadows) GetTiles() *[shaders.MaxShadowTiles]shaders.ShadowsTileData {
	return &w.tiles
}

// Process updates shadow atlas tiles for the current volumes, things, and depth tile configuration.
// It reuses cached tiles when possible and rebuilds dirty tiles, including static and instanced shadow data.
func (w *Shadows) Process(vols *model.Volumes, th *model.Things, depth *shaders.Depth) {
	if w.bContext.meshFrame == 0 {
		w.bContext.meshFrame = 1
	}
	w.shadowFrame++

	if w.bStatic != nil {
		w.bStatic.TryBuildStaticAtlas(vols)
	}

	w.instPool.Reset()
	tick := textures.GlobalTick()
	shadowExtraSig := uint64(0)
	anyShadowInstances := false
	hasStreamer := vols.HasStreamer()
	worldToken := w.createToken(vols, th)
	targetBufIdx, targetGen, targetOK := depth.TargetInfo()
	streamKeepAlive := hasStreamer && (w.shadowFrame&127 == 0)

	var tileMatrices [shaders.MaxShadowTiles][16]float32
	var activeMask uint32
	var queryMask uint32
	for t := range shaders.MaxShadowTiles {
		active, orthoDepthClamp, matPtr := depth.GetTileConfig(t)
		if !active || matPtr == nil {
			w.tiles[t].SetActive(false)
			w.tiles[t].SetReuse(false)
			continue
		}
		mat := *(*[16]float32)(unsafe.Pointer(matPtr))
		tileMatrices[t] = mat
		tileBit := uint32(1) << t
		activeMask |= tileBit

		if !streamKeepAlive {
			if bufIdx, gen, cachedHash, ok := depth.PeekReusableTile(t, &mat); ok && bufIdx >= 0 && bufIdx < len(w.tileDeps) {
				dep := &w.tileDeps[bufIdx][t]
				if dep.valid && dep.atlasGen == gen && dep.sceneHash == cachedHash && dep.worldToken == worldToken && dep.tick == tick {
					ptr := &w.tiles[t]
					ptr.Assign(true, nil, mat, cachedHash, true)
					continue
				}
			}
		}

		w.tileFrustums[t].Rebuild(glToEngineVP(mat))
		if orthoDepthClamp {
			// Room ortho enables DEPTH_CLAMP so near overhangs above lightCamY still cast shadows.
			w.tileFrustums[t].DisableNearPlane()
		}
		queryMask |= tileBit
	}
	if activeMask == 0 {
		return
	}
	if queryMask == 0 {
		if w.prevShadowExtraSig != 0 {
			w.bContext.shadowExtraGen++
			w.prevShadowExtraSig = 0
		}
		return
	}

	if queryMask == activeMask {
		w.entityCache.Sweep(w.shadowFrame)
	}

	var volAcc, volXor [shaders.MaxShadowTiles]uint64
	var volCount [shaders.MaxShadowTiles]int
	var thingAcc, thingXor [shaders.MaxShadowTiles]uint64
	var thingCount [shaders.MaxShadowTiles]int

	w.collectShadowVolumes(vols, queryMask, &volAcc, &volXor, &volCount)
	w.collectShadowThings(th, queryMask, &thingAcc, &thingXor, &thingCount)

	var tileHashes [shaders.MaxShadowTiles]uint64
	var dirtyMask uint32
	for rem := queryMask; rem != 0; {
		t := bits.TrailingZeros32(rem)
		rem &^= 1 << t

		sceneHash := mixU64(volAcc[t]^volXor[t], uint64(volCount[t]))
		sceneHash = mixU64(sceneHash, thingAcc[t]^thingXor[t])
		sceneHash = mixU64(sceneHash, uint64(thingCount[t]))

		if depth.CanReuseTile(t, &tileMatrices[t], sceneHash) {
			ptr := &w.tiles[t]
			ptr.Assign(true, nil, tileMatrices[t], sceneHash, true)
			if targetOK && targetBufIdx >= 0 && targetBufIdx < len(w.tileDeps) {
				ptrTD := &w.tileDeps[targetBufIdx][t]
				ptrTD.Assign(true, targetGen, sceneHash, worldToken, tick)
			}
			continue
		}

		tileHashes[t] = sceneHash
		dirtyMask |= uint32(1) << t
	}

	if dirtyMask != 0 {
		w.resolveDirtyVolumes(dirtyMask, vols)

		for rem := dirtyMask; rem != 0; {
			t := bits.TrailingZeros32(rem)
			rem &^= 1 << t

			tileBit := uint32(1) << t
			tb := w.builders[t]
			tb.Reset()

			w.buildTileStaticForBit(tb, tileBit)
			if w.buildTileThingsForBit(tb, tileBit, &shadowExtraSig) {
				anyShadowInstances = true
			}
			ptr := &w.tiles[t]
			ptr.Assign(true, tb.dcRender, tileMatrices[t], tileHashes[t], false)
			if targetOK && targetBufIdx >= 0 && targetBufIdx < len(w.tileDeps) {
				tdPtr := &w.tileDeps[targetBufIdx][t]
				tdPtr.Assign(true, targetGen, tileHashes[t], worldToken, tick)
			}
		}
	}

	var shadowInstVBO uint32
	if anyShadowInstances {
		shadowInstVBO = w.instPool.Upload()
	}

	for rem := dirtyMask; rem != 0; {
		t := bits.TrailingZeros32(rem)
		rem &^= 1 << t
		w.builders[t].Prepare(shadowInstVBO)
	}

	if shadowExtraSig != w.prevShadowExtraSig {
		// Advance shadowExtraGen separately so Main uploads off-camera vertices without invalidating Context.Commit.
		w.bContext.shadowExtraGen++
		w.prevShadowExtraSig = shadowExtraSig
	}
}

// createToken builds a mixed world-state token used to detect shadow cache invalidation.
func (w *Shadows) createToken(vols *model.Volumes, th *model.Things) uint64 {
	tk := geometry.HashOffset64
	tk = geometry.MixU64(tk, geometry.MixUnsafePtr(unsafe.Pointer(vols)))
	tk = geometry.MixU64(tk, uint64(vols.Len()))
	tk = geometry.MixU64(tk, vols.TreeVersion())
	tk = geometry.MixU64(tk, volumes.GlobalDirtyEpoch())
	if w.bStatic != nil {
		tk = geometry.MixU64(tk, w.bStatic.StateVersion())
		tk = geometry.MixU64(tk, w.bStatic.AnimTickSig())
	}
	tk = geometry.MixU64(tk, geometry.MixUnsafePtr(unsafe.Pointer(th)))
	tk = geometry.MixU64(tk, th.TreeVersion())
	tk = geometry.MixU64(tk, aabb.GlobalBoxPoseEpoch())
	tk = geometry.MixU64(tk, vertices.StateEpoch())
	return tk
}

// collectShadowVolumes gathers hashed shadow volumes intersecting active tile frustums into scratch storage.
// It updates per-tile accumulation, xor, and count signatures for shadow tile reuse checks.
func (w *Shadows) collectShadowVolumes(vols *model.Volumes, activeMask uint32, volAcc, volXor *[shaders.MaxShadowTiles]uint64, volCount *[shaders.MaxShadowTiles]int) {
	w.volScratchLen = 0
	if activeMask == 0 {
		return
	}
	vols.QueryFrustumsMask(w.tileFrustums[:], activeMask, func(object aabb.IObject, leafMask uint32) bool {
		vol, ok := object.(*volumes.Volume)
		if !ok || vol == nil {
			return false
		}
		h := vol.GetHash()
		if h == 0 {
			return false
		}
		if w.volScratchLen >= len(w.volScratch) {
			newLen := len(w.volScratch) * 2
			volS := make([]ShadowsCollectedVol, newLen)
			copy(volS, w.volScratch)
			w.volScratch = volS
		}

		ptr := &w.volScratch[w.volScratchLen]
		ptr.Assign(vol, leafMask)
		w.volScratchLen++

		if entry := w.bStatic.GetEntry(vol.GetFaces()); entry != nil && !entry.detached && entry.chunk != nil {
			entry.chunk.lastSeenFrame = w.bContext.meshFrame
		}

		xorH := bits.RotateLeft64(h, 17)
		for rem := leafMask; rem != 0; {
			t := bits.TrailingZeros32(rem)
			rem &^= 1 << t
			volAcc[t] += h
			volXor[t] ^= xorH
			volCount[t]++
		}
		return false
	})
}

// collectShadowThings collects visible shadow things for active tiles and updates per-tile hash summaries.
func (w *Shadows) collectShadowThings(th *model.Things, activeMask uint32, thingAcc, thingXor *[shaders.MaxShadowTiles]uint64, thingCount *[shaders.MaxShadowTiles]int) {
	w.thingScratchLen = 0
	if activeMask == 0 {
		return
	}
	th.QueryFrustumsMask(w.tileFrustums[:], activeMask, func(object aabb.IObject, leafMask uint32) bool {
		thing, ok := object.(ithing.IThing)
		if !ok || thing == nil {
			return false
		}
		hash := thing.GetHash()
		if hash == 0 {
			return false
		}
		ent := thing.GetEntity()
		if ent == nil {
			return false
		}

		sEntry := w.entityCache.Retrieve(ent, w.shadowFrame)
		sEntry.Update(w.shadowFrame, thing, ent, hash, w.bThings)

		if w.thingScratchLen >= len(w.thingScratch) {
			newLen := len(w.thingScratch) * 2
			volS := make([]ShadowsCollectedThing, newLen)
			copy(volS, w.thingScratch)
			w.thingScratch = volS
		}

		thingScratch := &w.thingScratch[w.thingScratchLen]
		thingScratch.Assign(thing, sEntry, sEntry.isRigidActive, leafMask)

		w.thingScratchLen++

		xorH := bits.RotateLeft64(hash, 23)
		for rem := leafMask; rem != 0; {
			t := bits.TrailingZeros32(rem)
			rem &^= 1 << t
			thingAcc[t] += hash
			thingXor[t] ^= xorH
			thingCount[t]++
		}
		return false
	})
}

// pushShadowStaticSubMeshes adds opaque static submeshes in the given index range to the shadow tile builder.
func pushShadowStaticSubMeshes(tb *ShadowTileBuilder, mesh *buffers.ResidentMesh, startIdx, endIdx int) {
	for sIdx := startIdx; sIdx < endIdx; sIdx++ {
		sub := mesh.SubMeshes[sIdx]
		if sub.IndexCount == 0 || sub.BlendMode != config.BlendModeOpaque {
			continue
		}
		item := tb.staticItems.Acquire()
		item.Mesh = mesh
		item.Sub = sub
	}
}

// resolveDirtyVolumes resolves meshes for collected volumes that intersect any dirty shadow tile.
func (w *Shadows) resolveDirtyVolumes(dirtyMask uint32, vols *model.Volumes) {
	for i := range w.volScratchLen {
		cv := &w.volScratch[i]
		if (cv.mask & dirtyMask) != 0 {
			w.bStatic.ResolveMesh(cv.vol, vols)
		}
	}
}

// buildTileStaticForBit adds static shadow submeshes for collected volumes whose mask includes tileBit.
func (w *Shadows) buildTileStaticForBit(tb *ShadowTileBuilder, tileBit uint32) {
	for i := range w.volScratchLen {
		cv := &w.volScratch[i]
		if (cv.mask & tileBit) == 0 {
			continue
		}
		rm, entry, _ := w.bStatic.LookupResolved(cv.vol)
		if entry != nil && !entry.detached {
			pushShadowStaticSubMeshes(tb, entry.atlas, entry.subStart, entry.subStart+entry.subCount)
			continue
		}
		if rm != nil {
			pushShadowStaticSubMeshes(tb, rm, 0, rm.SubMeshesLen)
		}
	}
}

// buildTileThingsForBit builds shadow draw data for collected things intersecting tileBit.
// It streams non-rigid opaque batches and groups rigid meshes into instance ranges.
// It updates shadowExtraSig for streamed caches and reports whether any instances were added.
func (w *Shadows) buildTileThingsForBit(tileBuilder *ShadowTileBuilder, tileBit uint32, shadowExtraSig *uint64) bool {
	if w.thingScratchLen == 0 {
		return false
	}
	w.groupToken++
	w.instGroups.Reset()

	for i := range w.thingScratchLen {
		ct := &w.thingScratch[i]
		if (ct.mask & tileBit) == 0 {
			continue
		}
		if ct.isRigid {
			rEntry := ct.entry
			rm := rEntry.rm
			if rm == nil || rm.LastSeenFrame != w.bContext.meshFrame || rm.NeedsAnimRebuild() {
				rm, _ = w.bThings.RetrieveMesh(rEntry.faces)
				rEntry.rm = rm
			}
			if rm != nil && rm.SubMeshesLen > 0 {
				var srg *ShadowsRigidGroup
				if rm.ShadowInstGroupFrame == w.groupToken {
					srg = w.instGroups.Items[rm.ShadowInstGroupIdx]
				} else {
					rm.ShadowInstGroupFrame = w.groupToken
					rm.ShadowInstGroupIdx = w.instGroups.Len
					srg = w.instGroups.Acquire()
					srg.Assign(rm)
				}
				srg.Add(rEntry)
			}
			continue
		}

		thing := ct.thing
		ent := thing.GetEntity()
		if entry := w.bThings.thingsRender.PeekIfCurrent(ent); entry != nil && entry.committedFrame == w.bContext.meshFrame {
			w.bContext.fv.ReplayOpaqueBatches(entry.committedBaseIdx, entry.GetFrameVertices(), tileBuilder.streamDC)
			continue
		}

		sEntry := ct.entry.EnsureStreamCache()
		if !w.bThings.BuildNonRigidCache(thing, sEntry) {
			continue
		}
		committedFrame := sEntry.GetCommittedFrame()
		meshFrame := w.bContext.GetMeshFrame()
		if committedFrame != meshFrame {
			sEntry.SetCommittedFrame(w.bContext.fv.CommitVerticesOnly(sEntry.GetFrameVertices()), meshFrame)
		}
		committedBaseIdx := sEntry.GetCommittedBaseIdx()
		if committedBaseIdx >= 0 {
			sig := mixU64(ct.entry.entID, sEntry.GetGen())
			*shadowExtraSig += sig
			*shadowExtraSig ^= bits.RotateLeft64(sig, 13)
			w.bContext.fv.ReplayOpaqueBatches(committedBaseIdx, sEntry.GetFrameVertices(), tileBuilder.GetStreamDC())
		}
	}

	hasInstances := false
	for gi := 0; gi < w.instGroups.Len; gi++ {
		g := w.instGroups.Items[gi]
		cnt := g.Len()
		if cnt == 0 {
			continue
		}
		hasInstances = true
		baseByteOff, baseFloat := w.instPool.Reserve(cnt)
		for k := range cnt {
			w.instPool.WriteMatrixAt(baseFloat+k*buffers.InstanceMatrixFloats, g.At(k))
		}
		tileBuilder.Add(g.mesh, baseByteOff, int32(cnt))
	}
	return hasInstances
}
