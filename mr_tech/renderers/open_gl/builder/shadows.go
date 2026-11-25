package builder

import (
	"math/bits"
	"unsafe"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/model"
	"github.com/markel1974/godoom/mr_tech/model/ithing"
	"github.com/markel1974/godoom/mr_tech/model/vertices"
	"github.com/markel1974/godoom/mr_tech/model/volumes"
	"github.com/markel1974/godoom/mr_tech/physics/aabb"
	"github.com/markel1974/godoom/mr_tech/physics/rigid_body"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/gpu/buffers"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/shaders"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// ShadowsRigidGroup represents a collection of rigid shadows sharing a resident mesh within one shadow tile.
type ShadowsRigidGroup struct {
	mesh    *buffers.ResidentMesh
	entries []*ShadowRigidEntry
}

// ShadowsCollectedVol tracks a volume intersected during batched shadow culling and its matching tile bitmask.
type ShadowsCollectedVol struct {
	vol  *volumes.Volume
	mask uint32
}

// ShadowsCollectedThing tracks an entity intersected during batched shadow culling and its matching tile bitmask.
type ShadowsCollectedThing struct {
	thing   ithing.IThing
	entry   *ShadowRigidEntry
	isRigid bool
	mask    uint32
}

type shadowTileDeps struct {
	valid         bool
	atlasGen      uint64
	sceneHash     uint64
	worldToken    uint64
	tick          uint64
	hasTickDepend bool
}

// Shadows manages per-light shadow tile frustum culling, O(1) entity caching, and shadow instance uploads.
type Shadows struct {
	bContext           *Context
	bStatic            *Static
	bThings            *Things
	tiles              [shaders.MaxShadowTiles]shaders.ShadowsTileData
	tileDeps           [2][shaders.MaxShadowTiles]shadowTileDeps
	builders           [shaders.MaxShadowTiles]*ShadowTileBuilder
	tileFrustums       [shaders.MaxShadowTiles]aabb.Frustum
	instPool           *buffers.InstancePool
	instGroups         buffers.PtrPool[ShadowsRigidGroup]
	groupToken         uint64
	volScratch         []ShadowsCollectedVol
	thingScratch       []ShadowsCollectedThing
	entityCache        ShadowRigidCache
	prevShadowExtraSig uint64
	shadowFrame        uint64
}

// NewShadows initializes and returns a new Shadows instance with configured context, static resources, and things.
func NewShadows(bContext *Context, bStatic *Static, bThing *Things) *Shadows {
	var ctx api.IContext
	if bContext != nil {
		ctx = bContext.ctx
	}
	bs := &Shadows{
		bContext: bContext,
		bStatic:  bStatic,
		bThings:  bThing,
		instPool: buffers.NewInstancePool(ctx, 256),
	}
	bs.instGroups.Init(32)
	for t := range shaders.MaxShadowTiles {
		bs.builders[t] = NewShadowTileBuilder(ctx)
	}
	return bs
}

// DisableAllTiles deactivates all shadow tiles in the shadow atlas and resets their reuse state to false.
func (w *Shadows) DisableAllTiles() {
	for t := range shaders.MaxShadowTiles {
		w.tiles[t].Active = false
		w.tiles[t].Reuse = false
	}
}

// GetTiles returns a pointer to the array of shadow tiles storing per-light shadow data for the shadow atlas.
func (w *Shadows) GetTiles() *[shaders.MaxShadowTiles]shaders.ShadowsTileData {
	return &w.tiles
}

// Process performs batched per-light frustum culling and shadow tile updates for all active shadow lights.
func (w *Shadows) Process(vols *model.Volumes, th *model.Things, d2 *shaders.Depth) {
	if d2 == nil {
		w.DisableAllTiles()
		return
	}
	if w.bContext.meshFrame == 0 {
		w.bContext.meshFrame = 1
	}
	w.shadowFrame++

	w.bStatic.TryBuildStaticAtlas(vols)

	w.instPool.Reset()
	tick := textures.GlobalTick()
	shadowExtraSig := uint64(0)
	anyShadowInstances := false

	worldToken := uint64(14695981039346656037)
	worldToken = mixU64(worldToken, mixPtrHash(unsafe.Pointer(vols)))
	hasStreamer := false
	if vols != nil {
		worldToken = mixU64(worldToken, uint64(vols.Len()))
		worldToken = mixU64(worldToken, vols.TreeVersion())
		hasStreamer = vols.HasStreamer()
	}
	worldToken = mixU64(worldToken, volumes.GlobalDirtyEpoch())
	worldToken = mixU64(worldToken, w.bStatic.StateVersion())
	worldToken = mixU64(worldToken, w.bStatic.AnimTickSig())
	worldToken = mixU64(worldToken, mixPtrHash(unsafe.Pointer(th)))
	if th != nil {
		worldToken = mixU64(worldToken, th.TreeVersion())
	}
	worldToken = mixU64(worldToken, aabb.GlobalBoxPoseEpoch())
	worldToken = mixU64(worldToken, vertices.StateEpoch())

	targetBufIdx, targetGen, targetOK := d2.TargetInfo()
	streamKeepAlive := hasStreamer && (w.shadowFrame&127 == 0)

	var tileMatrices [shaders.MaxShadowTiles][16]float32
	var activeMask uint32
	var queryMask uint32
	for t := range shaders.MaxShadowTiles {
		active, orthoDepthClamp, matPtr := d2.GetTileConfig(t)
		if !active || matPtr == nil {
			w.tiles[t].Active = false
			w.tiles[t].Reuse = false
			continue
		}
		mat := *(*[16]float32)(unsafe.Pointer(matPtr))
		tileMatrices[t] = mat
		tileBit := uint32(1) << t
		activeMask |= tileBit

		if !streamKeepAlive {
			if bufIdx, gen, cachedHash, ok := d2.PeekReusableTile(t, &mat); ok && bufIdx >= 0 && bufIdx < len(w.tileDeps) {
				dep := &w.tileDeps[bufIdx][t]
				if dep.valid && dep.atlasGen == gen && dep.sceneHash == cachedHash && dep.worldToken == worldToken && (!dep.hasTickDepend || dep.tick == tick) {
					ptr := &w.tiles[t]
					ptr.Active = true
					ptr.Matrix = mat
					ptr.SceneHash = cachedHash
					ptr.Reuse = true
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
	var tileHasTickDep [shaders.MaxShadowTiles]bool

	w.collectShadowVolumes(vols, queryMask, &volAcc, &volXor, &volCount, &tileHasTickDep)
	w.collectShadowThings(th, queryMask, &thingAcc, &thingXor, &thingCount, &tileHasTickDep)

	var tileHashes [shaders.MaxShadowTiles]uint64
	var dirtyMask uint32
	for rem := queryMask; rem != 0; {
		t := bits.TrailingZeros32(rem)
		rem &^= 1 << t

		sceneHash := mixU64(volAcc[t]^volXor[t], uint64(volCount[t]))
		sceneHash = mixU64(sceneHash, thingAcc[t]^thingXor[t])
		sceneHash = mixU64(sceneHash, uint64(thingCount[t]))

		if d2.CanReuseTile(t, &tileMatrices[t], sceneHash) {
			ptr := &w.tiles[t]
			ptr.Active = true
			ptr.Matrix = tileMatrices[t]
			ptr.SceneHash = sceneHash
			ptr.Reuse = true
			if targetOK && targetBufIdx >= 0 && targetBufIdx < len(w.tileDeps) {
				w.tileDeps[targetBufIdx][t] = shadowTileDeps{
					valid:         true,
					atlasGen:      targetGen,
					sceneHash:     sceneHash,
					worldToken:    worldToken,
					tick:          tick,
					hasTickDepend: tileHasTickDep[t],
				}
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
			tb.staticItems.Reset()
			tb.instItems.Reset()
			tb.instRanges = tb.instRanges[:0]
			tb.streamDC.DeepReset()

			w.buildTileStaticForBit(tb, tileBit)
			if w.buildTileThingsForBit(tb, tileBit, &shadowExtraSig) {
				anyShadowInstances = true
			}
			ptr := &w.tiles[t]
			ptr.Active = true
			ptr.RenderDC = tb.dcRender
			ptr.Matrix = tileMatrices[t]
			ptr.SceneHash = tileHashes[t]
			ptr.Reuse = false
			if targetOK && targetBufIdx >= 0 && targetBufIdx < len(w.tileDeps) {
				w.tileDeps[targetBufIdx][t] = shadowTileDeps{
					valid:         true,
					atlasGen:      targetGen,
					sceneHash:     tileHashes[t],
					worldToken:    worldToken,
					tick:          tick,
					hasTickDepend: tileHasTickDep[t],
				}
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

func (w *Shadows) collectShadowVolumes(vols *model.Volumes, activeMask uint32, volAcc, volXor *[shaders.MaxShadowTiles]uint64, volCount *[shaders.MaxShadowTiles]int, tileHasTickDep *[shaders.MaxShadowTiles]bool) {
	w.volScratch = w.volScratch[:0]
	if vols == nil || activeMask == 0 {
		return
	}
	vols.QueryFrustumsMask(w.tileFrustums[:], activeMask, func(object aabb.IObject, leafMask uint32) bool {
		vol, ok := object.(*volumes.Volume)
		if !ok || vol == nil {
			return false
		}
		faces := vol.GetFaces()
		if faces == nil || faces.Len() == 0 {
			return false
		}
		w.volScratch = append(w.volScratch, ShadowsCollectedVol{vol: vol, mask: leafMask})

		h := mixPtrHash(unsafe.Pointer(faces))
		h = mixU64(h, faces.GetVersion())
		h = mixU64(h, uint64(faces.Len()))
		volHasAnim := false
		if vol.IsDirty() {
			h = mixU64(h, 0x9e3779b97f4a7c15)
		} else if entry := w.bStatic.GetEntry(faces); entry != nil && !entry.detached {
			if entry.chunk != nil {
				entry.chunk.lastSeenFrame = w.bContext.meshFrame
			}
		} else if rm := w.bStatic.GetResidentMesh(faces); rm != nil && rm.AnimLen > 0 {
			volHasAnim = true
			for i := 0; i < rm.AnimLen; i++ {
				h = mixU64(h, mixPtrHash(unsafe.Pointer(rm.AnimMats[i].GetCurrentFrame())))
			}
		}

		xorH := bits.RotateLeft64(h, 17)
		for rem := leafMask; rem != 0; {
			t := bits.TrailingZeros32(rem)
			rem &^= 1 << t
			volAcc[t] += h
			volXor[t] ^= xorH
			volCount[t]++
			if volHasAnim {
				tileHasTickDep[t] = true
			}
		}
		return false
	})
}

func (w *Shadows) hashStreamingThing(sEntry *ShadowRigidEntry, entID uint64, ent *rigid_body.Entity, thing ithing.IThing, entry *ThingRenderCache) uint64 {
	pFaces, pNextFaces, lp, renderMode := thing.GetUIFaces()
	if pFaces == nil || pFaces.Len() == 0 {
		return 0
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

	matched := entry != nil && entry.Equal(pFaces, pNextFaces, facesVer, nextVer, facesLen, nextLen, lerp, yaw, renderModeF, oX, oY, oZ)
	if matched && sEntry.streamHashValid && sEntry.streamHashSrc == entry && sEntry.streamHashGen == entry.gen {
		return sEntry.streamHash
	}

	h := mixU64(entID, mixPtrHash(unsafe.Pointer(pFaces)))
	h = mixU64(h, facesVer)
	h = mixU64(h, uint64(facesLen))
	h = mixU64(h, mixPtrHash(unsafe.Pointer(pNextFaces)))
	h = mixU64(h, nextVer)
	h = mixU64(h, uint64(nextLen))
	h = mixU64(h, f32BitsStable(lerp))
	h = mixU64(h, f32BitsStable(yaw))
	h = mixU64(h, f32BitsStable(renderModeF))
	h = mixU64(h, f32BitsStable(oX))
	h = mixU64(h, f32BitsStable(oY))
	h = mixU64(h, f32BitsStable(oZ))
	gen := uint64(0)
	if entry != nil {
		gen = entry.gen
	}
	if !matched {
		gen++
	}
	h = mixU64(h, gen)

	if matched {
		sEntry.streamHash = h
		sEntry.streamHashGen = entry.gen
		sEntry.streamHashSrc = entry
		sEntry.streamHashValid = true
	} else {
		sEntry.streamHashValid = false
	}
	return h
}

func (w *Shadows) evalThingShadowHash(thing ithing.IThing, ent *rigid_body.Entity, sEntry *ShadowRigidEntry) (uint64, bool) {
	if !sEntry.rigidSrcChecked {
		sEntry.rigidSrc, _ = thing.(ithing.IRigidRenderSource)
		sEntry.rigidSrcChecked = true
	}
	if sEntry.rigidSrc != nil {
		if rigidFaces := sEntry.rigidSrc.GetRigidFaces(); rigidFaces != nil && rigidFaces.Len() > 0 {
			h := sEntry.Update(sEntry.entID, ent, rigidFaces, w.bThings, w.shadowFrame)
			if !sEntry.staticChecked {
				sEntry.staticCap, _ = thing.(staticRigidPoseChecker)
				sEntry.staticChecked = true
			}
			sEntry.staticFastOK = !sEntry.hasAnimMat && h != 0 && sEntry.staticCap != nil && sEntry.staticCap.IsStaticRigidPose()
			return h, true
		}
	}

	streamEntry := w.bThings.thingsRender.PeekIfCurrent(ent)
	if streamEntry == nil {
		streamEntry = sEntry.EnsureStreamCache()
	}
	h := w.hashStreamingThing(sEntry, sEntry.entID, ent, thing, streamEntry)
	sEntry.staticFastOK = false
	sEntry.poseGen = ent.GetPoseGen()
	sEntry.hash = h
	sEntry.seenFrame = w.shadowFrame
	return h, false
}

func (w *Shadows) collectShadowThings(th *model.Things, activeMask uint32, thingAcc, thingXor *[shaders.MaxShadowTiles]uint64, thingCount *[shaders.MaxShadowTiles]int, tileHasTickDep *[shaders.MaxShadowTiles]bool) {
	w.thingScratch = w.thingScratch[:0]
	if th == nil || activeMask == 0 {
		return
	}
	th.QueryFrustumsMask(w.tileFrustums[:], activeMask, func(object aabb.IObject, leafMask uint32) bool {
		thing, ok := object.(ithing.IThing)
		if !ok || thing == nil || !thing.IsVisible() {
			return false
		}
		ent := thing.GetEntity()
		if ent == nil {
			return false
		}

		sEntry := w.entityCache.Retrieve(ent, w.shadowFrame)
		var (
			h       uint64
			isRigid bool
		)
		if sEntry.seenFrame == w.shadowFrame {
			h = sEntry.hash
			isRigid = sEntry.isRigidActive
		} else if sEntry.staticFastOK && sEntry.poseGen == ent.GetPoseGen() && sEntry.faces != nil && sEntry.faces.GetVersion() == sEntry.facesVer {
			h = sEntry.hash
			isRigid = true
			sEntry.seenFrame = w.shadowFrame
		} else {
			h, isRigid = w.evalThingShadowHash(thing, ent, sEntry)
			sEntry.isRigidActive = isRigid
			sEntry.hash = h
			sEntry.seenFrame = w.shadowFrame
		}

		if h == 0 {
			return false
		}
		w.thingScratch = append(w.thingScratch, ShadowsCollectedThing{
			thing:   thing,
			entry:   sEntry,
			isRigid: isRigid,
			mask:    leafMask,
		})

		isTickDep := !sEntry.staticFastOK
		xorH := bits.RotateLeft64(h, 23)
		for rem := leafMask; rem != 0; {
			t := bits.TrailingZeros32(rem)
			rem &^= 1 << t
			thingAcc[t] += h
			thingXor[t] ^= xorH
			thingCount[t]++
			if isTickDep {
				tileHasTickDep[t] = true
			}
		}
		return false
	})
}

// pushShadowStaticSubMeshes adds static sub-meshes from the specified range to the shadow tile builder's static item pool.
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

func (w *Shadows) resolveDirtyVolumes(dirtyMask uint32, vols *model.Volumes) {
	for i := range w.volScratch {
		cv := &w.volScratch[i]
		if (cv.mask & dirtyMask) != 0 {
			w.bStatic.ResolveMesh(cv.vol, vols)
		}
	}
}

func (w *Shadows) buildTileStaticForBit(tb *ShadowTileBuilder, tileBit uint32) {
	for i := range w.volScratch {
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

func (w *Shadows) buildTileThingsForBit(tb *ShadowTileBuilder, tileBit uint32, shadowExtraSig *uint64) bool {
	if len(w.thingScratch) == 0 {
		return false
	}
	w.groupToken++
	w.instGroups.Reset()

	for i := range w.thingScratch {
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
				if rm != nil && rm.AnimLen > 0 {
					rEntry.hasAnimMat = true
					rEntry.staticFastOK = false
				}
			}
			if rm != nil && rm.SubMeshesLen > 0 {
				var g *ShadowsRigidGroup
				if rm.ShadowInstGroupFrame == w.groupToken {
					g = w.instGroups.Items[rm.ShadowInstGroupIdx]
				} else {
					rm.ShadowInstGroupFrame = w.groupToken
					rm.ShadowInstGroupIdx = w.instGroups.Len
					g = w.instGroups.Acquire()
					g.mesh = rm
					g.entries = g.entries[:0]
				}
				g.entries = append(g.entries, rEntry)
			}
			continue
		}

		thing := ct.thing
		ent := thing.GetEntity()
		if entry := w.bThings.thingsRender.PeekIfCurrent(ent); entry != nil && entry.committedFrame == w.bContext.meshFrame {
			w.bContext.fv.ReplayOpaqueBatches(entry.committedBaseIdx, entry.GetFrameVertices(), tb.streamDC)
			continue
		}

		sEntry := ct.entry.EnsureStreamCache()
		if !w.bThings.BuildNonRigidCache(thing, sEntry) {
			continue
		}
		if sEntry.committedFrame != w.bContext.meshFrame {
			sEntry.committedBaseIdx = w.bContext.fv.CommitVerticesOnly(sEntry.GetFrameVertices())
			sEntry.committedFrame = w.bContext.meshFrame
		}
		if sEntry.committedBaseIdx >= 0 {
			sig := mixU64(ct.entry.entID, sEntry.gen)
			*shadowExtraSig += sig
			*shadowExtraSig ^= bits.RotateLeft64(sig, 13)
			w.bContext.fv.ReplayOpaqueBatches(sEntry.committedBaseIdx, sEntry.GetFrameVertices(), tb.streamDC)
		}
	}

	hasInstances := false
	for gi := 0; gi < w.instGroups.Len; gi++ {
		g := w.instGroups.Items[gi]
		cnt := len(g.entries)
		if cnt == 0 {
			continue
		}
		hasInstances = true
		baseByteOff, baseFloat := w.instPool.Reserve(cnt)
		for k := range cnt {
			w.instPool.WriteMatrixAt(baseFloat+k*buffers.InstanceMatrixFloats, &g.entries[k].mat)
		}
		tb.instRanges = append(tb.instRanges, ShadowInstRange{
			mesh:           g.mesh,
			baseByteOffset: baseByteOff,
			instanceCount:  int32(cnt),
		})
	}
	return hasInstances
}
