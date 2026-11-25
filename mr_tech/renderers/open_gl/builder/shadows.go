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
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/gpu/buffers"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/shaders"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// ShadowsRigidGroup represents a collection of rigid shadows, including a mesh and associated shadow entries.
type ShadowsRigidGroup struct {
	mesh    *buffers.ResidentMesh
	entries []*ShadowRigidEntry
}

// ShadowsCollectedThing represents an entity and its associated rigid entry used during shadow processing.
type ShadowsCollectedThing struct {
	thing ithing.IThing
	rigid *ShadowRigidEntry
}

// Shadows represents a system responsible for managing shadow generation, caching, and rendering state.
type Shadows struct {
	bContext           *Context
	bStatic            *Static
	bThings            *Things
	tiles              [shaders.MaxShadowTiles]shaders.ShadowTileData
	builders           [shaders.MaxShadowTiles]*ShadowTileBuilder
	instPool           *buffers.InstancePool
	instGroups         buffers.PtrPool[ShadowsRigidGroup]
	groupToken         uint64
	frustum            *aabb.Frustum
	volScratch         []*volumes.Volume
	thingScratch       []ShadowsCollectedThing
	streamCache        shadowThingCache
	rigidCache         ShadowRigidCache
	prevShadowExtraSig uint64
	shadowFrame        uint64
}

// NewShadows initializes and returns a new Shadows instance with configured context, static resources, and things.
func NewShadows(bContext *Context, bStatic *Static, bThing *Things) *Shadows {
	bs := &Shadows{
		bContext: bContext,
		bStatic:  bStatic,
		bThings:  bThing,
		frustum:  aabb.NewFrustum(),
		instPool: buffers.NewInstancePool(bContext.ctx, 256),
	}
	bs.instGroups.Init(32)
	for t := range shaders.MaxShadowTiles {
		bs.builders[t] = NewShadowTileBuilder(bContext.ctx)
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

// GetTiles returns a pointer to the array of shadow tiles, which store per-light shadow data for the shadow atlas.
func (w *Shadows) GetTiles() *[shaders.MaxShadowTiles]shaders.ShadowTileData {
	return &w.tiles
}

// Process performs shadow computation for the given volumes, things, and depth settings, updating shadow tiles and buffers.
func (w *Shadows) Process(vols *model.Volumes, th *model.Things, d *shaders.Depth) {
	if d == nil {
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
	var tileDirty [shaders.MaxShadowTiles]bool

	w.streamCache.sweep(w.shadowFrame)
	w.rigidCache.Sweep(w.shadowFrame)

	for t := range shaders.MaxShadowTiles {
		active, orthoDepthClamp, matPtr := d.GetTileConfig(t)
		if !active || matPtr == nil {
			w.tiles[t].Active = false
			w.tiles[t].Reuse = false
			continue
		}

		mat := *(*[16]float32)(unsafe.Pointer(matPtr))
		w.frustum.Rebuild(glToEngineVP(mat))
		if orthoDepthClamp {
			// Room ortho enables DEPTH_CLAMP so near overhangs above lightCamY still cast shadows.
			pNear := w.frustum.Planes[4]
			pNear.NormalX, pNear.NormalY, pNear.NormalZ, pNear.D = 0, 0, 0, 1
		}

		volAcc, volXor, volCount := w.collectShadowVolumes(vols)
		thingAcc, thingXor, thingCount := w.collectShadowThings(th, tick)

		sceneHash := mixU64(volAcc^volXor, uint64(volCount))
		sceneHash = mixU64(sceneHash, thingAcc^thingXor)
		sceneHash = mixU64(sceneHash, uint64(thingCount))

		if d.CanReuseTile(t, &mat, sceneHash) {
			ptr := &w.tiles[t]
			ptr.Active = true
			ptr.Matrix = mat
			ptr.SceneHash = sceneHash
			ptr.Reuse = true
			continue
		}

		tileDirty[t] = true
		tb := w.builders[t]
		tb.staticItems.Reset()
		tb.instItems.Reset()
		tb.instRanges = tb.instRanges[:0]
		tb.streamDC.DeepReset()

		w.buildTileStatic(tb, vols)
		if w.buildTileThings(tb, tick, &shadowExtraSig) {
			anyShadowInstances = true
		}
		ptr := &w.tiles[t]
		ptr.Active = true
		ptr.RenderDC = tb.dcRender
		ptr.Matrix = mat
		ptr.SceneHash = sceneHash
		ptr.Reuse = false
	}

	var shadowInstVBO uint32
	if anyShadowInstances {
		shadowInstVBO = w.instPool.Upload()
	}

	for t := range shaders.MaxShadowTiles {
		if !tileDirty[t] {
			continue
		}
		tb := w.builders[t]
		tb.Prepare(shadowInstVBO)
	}

	if shadowExtraSig != w.prevShadowExtraSig {
		// Advance shadowExtraGen separately so Main uploads off-camera vertices without invalidating Context.Commit.
		w.bContext.shadowExtraGen++
		w.prevShadowExtraSig = shadowExtraSig
	}
}

// collectShadowVolumes processes shadow-casting volumes intersecting the shadow frustum and computes hashes for caching.
func (w *Shadows) collectShadowVolumes(vols *model.Volumes) (acc uint64, xorHash uint64, count int) {
	w.volScratch = w.volScratch[:0]
	if vols == nil {
		return 0, 0, 0
	}
	vols.QueryFrustum(w.frustum, func(object aabb.IObject) bool {
		vol, ok := object.(*volumes.Volume)
		if !ok || vol == nil {
			return false
		}
		faces := vol.GetFaces()
		if faces == nil || faces.Len() == 0 {
			return false
		}
		w.volScratch = append(w.volScratch, vol)

		h := mixPtrHash(unsafe.Pointer(faces))
		h = mixU64(h, faces.GetVersion())
		h = mixU64(h, uint64(faces.Len()))
		if vol.IsDirty() {
			h = mixU64(h, 0x9e3779b97f4a7c15)
		} else if entry := w.bStatic.GetEntry(faces); entry != nil && !entry.detached {
			if entry.chunk != nil {
				entry.chunk.lastSeenFrame = w.bContext.meshFrame
			}
		} else if rm := w.bStatic.GetResidentMesh(faces); rm != nil {
			for i := 0; i < rm.AnimLen; i++ {
				h = mixU64(h, mixPtrHash(unsafe.Pointer(rm.AnimMats[i].GetCurrentFrame())))
			}
		}

		acc += h
		xorHash ^= bits.RotateLeft64(h, 17)
		count++
		return false
	})
	return acc, xorHash, count
}

// hashStreamingThing computes a stable hash for a streaming thing based on its rendering and state-related properties.
func (w *Shadows) hashStreamingThing(entID uint64, ent *rigid_body.Entity, thing ithing.IThing, entry *ThingRenderCache) uint64 {
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
	if entry == nil || !entry.Equal(pFaces, pNextFaces, facesVer, nextVer, facesLen, nextLen, lerp, yaw, renderModeF, oX, oY, oZ) {
		gen++
	}
	h = mixU64(h, gen)
	return h
}

// collectShadowThings gathers shadowable things within the frustum, computes their hashes, and updates internal caches.
// It returns the accumulated hash, a combined XOR hash, and the total count of collected things.
func (w *Shadows) collectShadowThings(th *model.Things, tick uint64) (acc uint64, xorHash uint64, count int) {
	w.thingScratch = w.thingScratch[:0]
	if th == nil {
		return 0, 0, 0
	}
	th.QueryFrustum(w.frustum, func(object aabb.IObject) bool {
		thing, ok := object.(ithing.IThing)
		if !ok || thing == nil || !thing.IsVisible() {
			return false
		}
		ent := thing.GetEntity()
		entID := ent.GetId()

		var h uint64
		rEntry, okR := w.rigidCache.slots[entID]
		if okR && rEntry.seenFrame == w.shadowFrame {
			h = rEntry.hash
		} else {
			entry := w.bThings.thingsRender.PeekIfCurrent(ent)
			if entry == nil {
				if sEntry, okCache := w.streamCache.slots[entID]; okCache {
					if sEntry.compileFrame != w.shadowFrame {
						thing.Compile(tick)
						sEntry.compileFrame = w.shadowFrame
					}
				} else {
					thing.Compile(tick)
				}
			}
			if rs, okSrc := thing.(ithing.IRigidRenderSource); okSrc {
				if rigidFaces := rs.GetRigidFaces(); rigidFaces != nil && rigidFaces.Len() > 0 {
					if !okR {
						rEntry = w.rigidCache.Retrieve(entID, w.shadowFrame)
					} else {
						rEntry.lastSeenFrame = w.shadowFrame
					}
					h = rEntry.Update(entID, ent, rigidFaces, w.bThings, w.shadowFrame)
				}
			}
			if h == 0 {
				if okR {
					rEntry.seenFrame = 0
					rEntry = nil
				}
				streamEntry := entry
				if streamEntry == nil {
					sEntry := w.streamCache.retrieve(entID, w.shadowFrame)
					sEntry.compileFrame = w.shadowFrame
					streamEntry = sEntry
				}
				h = w.hashStreamingThing(entID, ent, thing, streamEntry)
			}
		}

		if h == 0 {
			return false
		}
		w.thingScratch = append(w.thingScratch, ShadowsCollectedThing{thing: thing, rigid: rEntry})
		acc += h
		xorHash ^= bits.RotateLeft64(h, 23)
		count++
		return false
	})
	return acc, xorHash, count
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

// buildTileStatic resolves and processes static shadow volumes for rendering within a shadow tile.
func (w *Shadows) buildTileStatic(tb *ShadowTileBuilder, vols *model.Volumes) {
	for _, vol := range w.volScratch {
		w.bStatic.ResolveMesh(vol, vols)
	}
	for _, vol := range w.volScratch {
		rm, entry, _ := w.bStatic.LookupResolved(vol)
		if entry != nil && !entry.detached {
			pushShadowStaticSubMeshes(tb, entry.atlas, entry.subStart, entry.subStart+entry.subCount)
			continue
		}
		if rm != nil {
			pushShadowStaticSubMeshes(tb, rm, 0, rm.SubMeshesLen)
		}
	}
}

// buildTileThings processes dynamic shadow-casting entities and groups instanced rendering data for shadow tiles.
// It updates shadow instance groups, rigid and non-rigid mesh rendering entries, and computes extra signature data.
// Returns true if any instanced shadow data is added to the tile builder, otherwise returns false.
func (w *Shadows) buildTileThings(tb *ShadowTileBuilder, tick uint64, shadowExtraSig *uint64) bool {
	if len(w.thingScratch) == 0 {
		return false
	}
	w.groupToken++
	w.instGroups.Reset()

	for _, ct := range w.thingScratch {
		if rEntry := ct.rigid; rEntry != nil {
			rm := rEntry.rm
			if rm == nil || rm.LastSeenFrame != w.bContext.meshFrame || rm.NeedsAnimRebuild() {
				rm, _ = w.bThings.RetrieveMesh(rEntry.faces)
				rEntry.rm = rm
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

		sEntry := w.streamCache.retrieve(ent.GetId(), w.shadowFrame)
		if sEntry.compileFrame != w.shadowFrame {
			thing.Compile(tick)
			sEntry.compileFrame = w.shadowFrame
		}
		if !w.bThings.BuildNonRigidCache(thing, sEntry) {
			continue
		}
		if sEntry.committedFrame != w.bContext.meshFrame {
			sEntry.committedBaseIdx = w.bContext.fv.CommitVerticesOnly(sEntry.GetFrameVertices())
			sEntry.committedFrame = w.bContext.meshFrame
		}
		if sEntry.committedBaseIdx >= 0 {
			sig := mixU64(ent.GetId(), sEntry.gen)
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
