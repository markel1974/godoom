package open_gl

import (
	"math"
	"math/bits"
	"unsafe"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/engine"
	"github.com/markel1974/godoom/mr_tech/model"
	"github.com/markel1974/godoom/mr_tech/model/ithing"
	"github.com/markel1974/godoom/mr_tech/model/vertices"
	"github.com/markel1974/godoom/mr_tech/model/volumes"
	"github.com/markel1974/godoom/mr_tech/physics/aabb"
	"github.com/markel1974/godoom/mr_tech/physics/rigid_body"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/gpu/buffers"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/gpu/draws"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/shaders"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// shadowInstRange describes a contiguous range of shadow mesh instances and its base buffer offset.
type shadowInstRange struct {
	mesh           *buffers.ResidentMesh
	baseByteOffset int32
	instanceCount  int32
}

// shadowTileBuilder assembles draw commands and pooled draw items for rendering a shadow tile.
type shadowRigidGroup struct {
	mesh    *buffers.ResidentMesh
	entries []*shadowRigidEntry
}

type shadowCollectedThing struct {
	thing ithing.IThing
	rigid *shadowRigidEntry
}

type shadowRigidEntry struct {
	entID          uint64
	inUse          bool
	seenFrame      uint64
	lastSeenFrame  uint64
	matValid       bool
	cx, cy, cz     float64
	qx, qy, qz, qw float64
	mat            [buffers.InstanceMatrixFloats]float32
	faces          *volumes.Faces
	facesVer       uint64
	facesLen       int
	rm             *buffers.ResidentMesh
	hash           uint64
}

type shadowRigidCache struct {
	slots         map[uint64]*shadowRigidEntry
	free          []*shadowRigidEntry
	lastSweepTick uint64
}

func (c *shadowRigidCache) retrieve(entID uint64, frame uint64) *shadowRigidEntry {
	if c.slots == nil {
		c.slots = make(map[uint64]*shadowRigidEntry, 128)
	}
	if entry, ok := c.slots[entID]; ok {
		entry.lastSeenFrame = frame
		return entry
	}
	var entry *shadowRigidEntry
	if n := len(c.free); n > 0 {
		entry = c.free[n-1]
		c.free = c.free[:n-1]
	} else {
		entry = &shadowRigidEntry{}
	}
	entry.entID = entID
	entry.inUse = true
	entry.seenFrame = 0
	entry.lastSeenFrame = frame
	entry.matValid = false
	entry.faces = nil
	entry.facesVer = 0
	entry.facesLen = 0
	entry.rm = nil
	entry.hash = 0
	c.slots[entID] = entry
	return entry
}

func (c *shadowRigidCache) sweep(frame uint64) {
	if len(c.slots) == 0 || frame-c.lastSweepTick < 256 {
		return
	}
	c.lastSweepTick = frame
	for id, entry := range c.slots {
		if frame-entry.lastSeenFrame > 256 {
			entry.inUse = false
			entry.faces = nil
			entry.rm = nil
			delete(c.slots, id)
			c.free = append(c.free, entry)
		}
	}
}

type shadowTileBuilder struct {
	dcRender    *draws.DrawCommandsRender
	streamDC    *draws.DrawCommands
	staticItems buffers.PtrPool[draws.DrawItemStatic]
	instItems   buffers.PtrPool[draws.DrawItemInstanced]
	instRanges  []shadowInstRange
}

// shadowThingCache caches per-thing shadow render state and recycles stale entries across frames.
type shadowThingCache struct {
	slots         map[uint64]*ThingRenderCache
	free          []*ThingRenderCache
	lastSweepTick uint64
}

// retrieve returns the cache entry for entID, creating or reusing one and marking it seen for frame.
func (c *shadowThingCache) retrieve(entID uint64, frame uint64) *ThingRenderCache {
	if c.slots == nil {
		c.slots = make(map[uint64]*ThingRenderCache, 128)
	}
	if entry, ok := c.slots[entID]; ok {
		entry.lastSeenFrame = frame
		return entry
	}
	var entry *ThingRenderCache
	if n := len(c.free); n > 0 {
		entry = c.free[n-1]
		c.free = c.free[:n-1]
	} else {
		entry = NewThingRenderCache()
	}
	entry.entID = entID
	entry.inUse = true
	entry.valid = false
	entry.pFaces = nil
	entry.pNextFaces = nil
	entry.lastSeenFrame = frame
	entry.compileFrame = 0
	entry.committedFrame = 0
	entry.committedBaseIdx = -1
	entry.fv.DeepReset()
	c.slots[entID] = entry
	return entry
}

// sweep removes stale cache entries not seen for over 256 frames and recycles them for reuse.
// sweep runs at most once every 256 frames.
func (c *shadowThingCache) sweep(frame uint64) {
	if len(c.slots) == 0 || frame-c.lastSweepTick < 256 {
		return
	}
	c.lastSweepTick = frame
	for id, entry := range c.slots {
		if frame-entry.lastSeenFrame > 256 {
			entry.inUse = false
			entry.valid = false
			entry.pFaces = nil
			entry.pNextFaces = nil
			entry.fv.DeepReset()
			delete(c.slots, id)
			c.free = append(c.free, entry)
		}
	}
}

// newShadowTileBuilder creates and initializes a shadow tile builder for the given builder volume.
func newShadowTileBuilder(w *BuilderVolume) *shadowTileBuilder {
	return &shadowTileBuilder{
		dcRender:   draws.NewDrawCommandsRender(w.ctx, "shadow_tile", false),
		streamDC:   draws.NewDrawCommands(config.BlendModeOpaque, 128),
		instRanges: make([]shadowInstRange, 0, 32),
	}
}

// glToEngineVP converts an OpenGL Y-up view-projection matrix to the engine's Z-up coordinate basis.
// It remaps axes for frustum tests against engine AABBs.
func glToEngineVP(m [16]float32) [16]float32 {
	// Engine AABBs use Z-up (ex, ey, ez); OpenGL world positions use Y-up (ex, ez, -ey).
	return [16]float32{
		m[0], m[1], m[2], m[3],
		-m[8], -m[9], -m[10], -m[11],
		m[4], m[5], m[6], m[7],
		m[12], m[13], m[14], m[15],
	}
}

// mixU64 combines a hash state and value into a well-distributed 64-bit hash.
func mixU64(h, x uint64) uint64 {
	z := h ^ x
	z ^= z >> 30
	z *= 0xbf58476d1ce4e5b9
	z ^= z >> 27
	z *= 0x94d049bb133111eb
	z ^= z >> 31
	return z
}

// f32BitsStable returns the IEEE 754 bits of v as uint64, normalizing both +0 and -0 to 0.
func f32BitsStable(v float32) uint64 {
	if v == 0 {
		return 0
	}
	return uint64(math.Float32bits(v))
}

// mixPtrHash returns a mixed 64-bit hash of p's address.
func mixPtrHash(p unsafe.Pointer) uint64 {
	return mixU64(14695981039346656037, uint64(uintptr(p)))
}

// ensureShadowState initializes any missing shadow rendering state, caches, pools, and tile builders.
func (w *BuilderVolume) ensureShadowState() {
	if w.texLookupFn == nil {
		w.texLookupFn = w.lookupTexLayer
	}
	if w.volMeshes == nil {
		w.volMeshes = make(map[*volumes.Faces]*buffers.ResidentMesh, 256)
	}
	if w.thingMeshes == nil {
		w.thingMeshes = make(map[uint64][]*buffers.ResidentMesh, 256)
	}
	if w.thingMeshesByPtr == nil {
		w.thingMeshesByPtr = make(map[*volumes.Faces]thingMeshPtrEntry, 512)
	}
	if w.thingsRender == nil {
		w.thingsRender = NewThingsRender()
	}
	if w.shadowFrustum == nil {
		w.shadowFrustum = aabb.NewFrustum()
	}
	if w.shadowInstPool == nil {
		w.shadowInstPool = buffers.NewInstancePool(w.ctx, 256)
		w.shadowInstGroups.Init(32)
	}
	for t := range shaders.MaxShadowTiles {
		if w.shadowBuilders[t] == nil {
			w.shadowBuilders[t] = newShadowTileBuilder(w)
		}
	}
}

// GetShadowTiles returns the shadow tile data array used by the builder volume.
func (w *BuilderVolume) GetShadowTiles() *[shaders.MaxShadowTiles]shaders.ShadowTileData {
	return &w.shadowTiles
}

// ComputeShadows updates shadow tile draw data for active depth tiles, reusing cached tiles when possible.
// It disables all shadow tiles when the engine or depth shader state is nil.
func (w *BuilderVolume) ComputeShadows(en *engine.Engine, d *shaders.Depth) {
	if d == nil || en == nil {
		for t := range shaders.MaxShadowTiles {
			w.shadowTiles[t].Active = false
			w.shadowTiles[t].Reuse = false
		}
		return
	}

	w.ensureShadowState()
	if w.meshFrame == 0 {
		w.meshFrame = 1
	}
	w.shadowFrame++

	vols := en.GetVolumes()
	th := en.GetThings()
	hasStreamer := vols != nil && vols.HasStreamer()
	if vols != nil && (!w.staticAtlasBuilt || w.staticAtlasStreamMode != hasStreamer || (!hasStreamer && w.staticAtlasVolCount != vols.Len())) {
		w.buildStaticAtlas(vols)
	}

	w.shadowInstPool.Reset()
	tick := textures.GlobalTick()
	shadowExtraSig := uint64(0)
	anyShadowInstances := false
	var tileDirty [shaders.MaxShadowTiles]bool

	w.shadowStreamCache.sweep(w.shadowFrame)
	w.shadowRigidCache.sweep(w.shadowFrame)

	for t := range shaders.MaxShadowTiles {
		active, orthoDepthClamp, matPtr := d.GetTileConfig(t)
		if !active || matPtr == nil {
			w.shadowTiles[t].Active = false
			w.shadowTiles[t].Reuse = false
			continue
		}

		mat := *(*[16]float32)(unsafe.Pointer(matPtr))
		w.shadowFrustum.Rebuild(glToEngineVP(mat))
		if orthoDepthClamp {
			// Room ortho enables DEPTH_CLAMP so near overhangs above lightCamY still cast shadows.
			pNear := w.shadowFrustum.Planes[4]
			pNear.NormalX, pNear.NormalY, pNear.NormalZ, pNear.D = 0, 0, 0, 1
		}

		volAcc, volXor, volCount := w.collectShadowVolumes(vols)
		thingAcc, thingXor, thingCount := w.collectShadowThings(th, tick)

		sceneHash := mixU64(volAcc^volXor, uint64(volCount))
		sceneHash = mixU64(sceneHash, thingAcc^thingXor)
		sceneHash = mixU64(sceneHash, uint64(thingCount))

		if d.CanReuseTile(t, &mat, sceneHash) {
			ptr := &w.shadowTiles[t]
			ptr.Active = true
			ptr.Matrix = mat
			ptr.SceneHash = sceneHash
			ptr.Reuse = true
			continue
		}

		tileDirty[t] = true
		tb := w.shadowBuilders[t]
		tb.staticItems.Reset()
		tb.instItems.Reset()
		tb.instRanges = tb.instRanges[:0]
		tb.streamDC.DeepReset()

		w.buildShadowTileStatic(tb, vols)
		if w.buildShadowTileThings(tb, tick, &shadowExtraSig) {
			anyShadowInstances = true
		}
		ptr := &w.shadowTiles[t]
		ptr.Active = true
		ptr.RenderDC = tb.dcRender
		ptr.Matrix = mat
		ptr.SceneHash = sceneHash
		ptr.Reuse = false
	}

	var shadowInstVBO uint32
	if anyShadowInstances {
		shadowInstVBO = w.shadowInstPool.Upload()
	}

	for t := range shaders.MaxShadowTiles {
		if !tileDirty[t] {
			continue
		}
		tb := w.shadowBuilders[t]
		for rIdx := range tb.instRanges {
			r := &tb.instRanges[rIdx]
			for sIdx := 0; sIdx < r.mesh.SubMeshesLen; sIdx++ {
				sub := r.mesh.SubMeshes[sIdx]
				if sub.IndexCount == 0 || sub.BlendMode != config.BlendModeOpaque {
					continue
				}
				item := tb.instItems.Acquire()
				item.Mesh = r.mesh
				item.Sub = sub
				item.InstanceVBO = shadowInstVBO
				item.BaseByteOffset = r.baseByteOffset
				item.InstanceCount = r.instanceCount
			}
		}
		tb.dcRender.Prepare(tb.streamDC.GetDrawCommands())
		tb.dcRender.SetShadowResidentDraws(tb.staticItems.Items, tb.staticItems.Len, tb.instItems.Items, tb.instItems.Len)
	}

	if shadowExtraSig != w.prevShadowExtraSig {
		w.frameGen++
		w.prevShadowExtraSig = shadowExtraSig
	}
}

// collectShadowVolumes gathers visible shadow volumes and returns aggregate hash data and the collected count.
func (w *BuilderVolume) collectShadowVolumes(vols *model.Volumes) (acc uint64, xorHash uint64, count int) {
	w.shadowVolScratch = w.shadowVolScratch[:0]
	if vols == nil {
		return 0, 0, 0
	}
	vols.QueryFrustum(w.shadowFrustum, func(object aabb.IObject) bool {
		vol, ok := object.(*volumes.Volume)
		if !ok || vol == nil {
			return false
		}
		faces := vol.GetFaces()
		if faces == nil || faces.Len() == 0 {
			return false
		}
		w.shadowVolScratch = append(w.shadowVolScratch, vol)

		h := mixPtrHash(unsafe.Pointer(faces))
		h = mixU64(h, faces.GetVersion())
		h = mixU64(h, uint64(faces.Len()))
		if vol.IsDirty() {
			h = mixU64(h, 0x9e3779b97f4a7c15)
		} else if entry := w.staticEntries[faces]; entry != nil && !entry.detached {
			if entry.chunk != nil {
				entry.chunk.lastSeenFrame = w.meshFrame
			}
		} else if rm := w.volMeshes[faces]; rm != nil {
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

// findCachedThingMesh returns a cached resident mesh matching the given faces, or nil if none exists.
func (w *BuilderVolume) findCachedThingMesh(rigidFaces *volumes.Faces) *buffers.ResidentMesh {
	gh := rigidFaces.GetGeomHash()
	ver := rigidFaces.GetVersion()
	faceCount := rigidFaces.Len()
	if pe, ok := w.thingMeshesByPtr[rigidFaces]; ok && pe.rm != nil && pe.geomHash == gh && pe.version == ver && pe.faceCount == faceCount {
		return pe.rm
	}
	for _, rm := range w.thingMeshes[gh] {
		if rm.SrcFaces == rigidFaces || (rm.FaceCount == faceCount && rm.SrcFaces.GetGeomHash() == gh && facesEquivalent(rigidFaces, rm.SrcFaces)) {
			w.cacheThingMeshPtr(rigidFaces, rm, gh, ver, faceCount)
			return rm
		}
	}
	return nil
}

// updateShadowRigidEntry refreshes a rigid shadow caster's transform matrix, cached mesh ref, and state hash at most once per shadow frame.
func (w *BuilderVolume) updateShadowRigidEntry(rEntry *shadowRigidEntry, entID uint64, ent *rigid_body.Entity, rigidFaces *volumes.Faces) uint64 {
	tPosX, tPosY, zBot := ent.GetCenter()
	qx, qy, qz, qw := ent.GetQRotation().Get()
	if !rEntry.matValid || rEntry.cx != tPosX || rEntry.cy != tPosY || rEntry.cz != zBot || rEntry.qx != qx || rEntry.qy != qy || rEntry.qz != qz || rEntry.qw != qw {
		rEntry.cx, rEntry.cy, rEntry.cz = tPosX, tPosY, zBot
		rEntry.qx, rEntry.qy, rEntry.qz, rEntry.qw = qx, qy, qz, qw
		buffers.BuildTransformMatrix(rEntry.mat[:], tPosX, tPosY, zBot, qx, qy, qz, qw)
		rEntry.matValid = true
	}

	facesVer := rigidFaces.GetVersion()
	facesLen := rigidFaces.Len()
	if rEntry.rm == nil || rEntry.faces != rigidFaces || rEntry.facesVer != facesVer || rEntry.facesLen != facesLen {
		rEntry.rm = w.findCachedThingMesh(rigidFaces)
		rEntry.faces = rigidFaces
		rEntry.facesVer = facesVer
		rEntry.facesLen = facesLen
	}

	h := mixU64(entID, mixPtrHash(unsafe.Pointer(rigidFaces)))
	h = mixU64(h, facesVer)
	h = mixU64(h, uint64(facesLen))
	h = mixU64(h, f32BitsStable(float32(tPosX)))
	h = mixU64(h, f32BitsStable(float32(tPosY)))
	h = mixU64(h, f32BitsStable(float32(zBot)))
	h = mixU64(h, f32BitsStable(float32(qx)))
	h = mixU64(h, f32BitsStable(float32(qy)))
	h = mixU64(h, f32BitsStable(float32(qz)))
	h = mixU64(h, f32BitsStable(float32(qw)))
	if rm := rEntry.rm; rm != nil && rm.AnimLen > 0 {
		for i := 0; i < rm.AnimLen; i++ {
			h = mixU64(h, mixPtrHash(unsafe.Pointer(rm.AnimMats[i].GetCurrentFrame())))
		}
	}
	rEntry.hash = h
	rEntry.seenFrame = w.shadowFrame
	return h
}

// hashStreamingThing returns a hash of a thing's streaming render state, or 0 when it has no render faces.
func (w *BuilderVolume) hashStreamingThing(entID uint64, ent *rigid_body.Entity, thing ithing.IThing, entry *ThingRenderCache) uint64 {
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

// collectShadowThings gathers visible shadow-casting things in the shadow frustum and returns aggregate hashes.
func (w *BuilderVolume) collectShadowThings(th *model.Things, tick uint64) (acc uint64, xorHash uint64, count int) {
	w.shadowThingScratch = w.shadowThingScratch[:0]
	if th == nil {
		return 0, 0, 0
	}
	th.QueryFrustum(w.shadowFrustum, func(object aabb.IObject) bool {
		thing, ok := object.(ithing.IThing)
		if !ok || thing == nil || !thing.IsVisible() {
			return false
		}
		ent := thing.GetEntity()
		entID := ent.GetId()

		var h uint64
		rEntry, okR := w.shadowRigidCache.slots[entID]
		if okR && rEntry.seenFrame == w.shadowFrame {
			h = rEntry.hash
		} else {
			entry := w.thingsRender.PeekIfCurrent(ent)
			if entry == nil {
				if sEntry, okCache := w.shadowStreamCache.slots[entID]; okCache {
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
						rEntry = w.shadowRigidCache.retrieve(entID, w.shadowFrame)
					} else {
						rEntry.lastSeenFrame = w.shadowFrame
					}
					h = w.updateShadowRigidEntry(rEntry, entID, ent, rigidFaces)
				}
			}
			if h == 0 {
				if okR {
					rEntry.seenFrame = 0
					rEntry = nil
				}
				streamEntry := entry
				if streamEntry == nil {
					sEntry := w.shadowStreamCache.retrieve(entID, w.shadowFrame)
					sEntry.compileFrame = w.shadowFrame
					streamEntry = sEntry
				}
				h = w.hashStreamingThing(entID, ent, thing, streamEntry)
			}
		}

		if h == 0 {
			return false
		}
		w.shadowThingScratch = append(w.shadowThingScratch, shadowCollectedThing{thing: thing, rigid: rEntry})
		acc += h
		xorHash ^= bits.RotateLeft64(h, 23)
		count++
		return false
	})
	return acc, xorHash, count
}

// buildShadowTileStatic populates the tile builder with opaque static draw items from resolved shadow volumes.
func (w *BuilderVolume) buildShadowTileStatic(tb *shadowTileBuilder, vols *model.Volumes) {
	for _, vol := range w.shadowVolScratch {
		w.resolveVolumeMesh(vol, vols)
	}
	for _, vol := range w.shadowVolScratch {
		rm, entry, _ := w.lookupResolvedVolume(vol)
		if entry != nil && !entry.detached {
			for sIdx := entry.subStart; sIdx < entry.subStart+entry.subCount; sIdx++ {
				sub := entry.atlas.SubMeshes[sIdx]
				if sub.IndexCount == 0 || sub.BlendMode != config.BlendModeOpaque {
					continue
				}
				item := tb.staticItems.Acquire()
				item.Mesh = entry.atlas
				item.Sub = sub
			}
			continue
		}
		if rm != nil {
			for sIdx := 0; sIdx < rm.SubMeshesLen; sIdx++ {
				sub := rm.SubMeshes[sIdx]
				if sub.IndexCount == 0 || sub.BlendMode != config.BlendModeOpaque {
					continue
				}
				item := tb.staticItems.Acquire()
				item.Mesh = rm
				item.Sub = sub
			}
		}
	}
}

// buildShadowTileThings builds shadow draw data for queued shadow things into tb.
// It reuses cached non-rigid geometry when possible and batches rigid meshes into instance ranges.
// It updates shadowExtraSig for streamed non-rigid entries and reports whether rigid instances were added.
func (w *BuilderVolume) buildShadowTileThings(tb *shadowTileBuilder, tick uint64, shadowExtraSig *uint64) bool {
	if len(w.shadowThingScratch) == 0 {
		return false
	}
	w.shadowGroupToken++
	w.shadowInstGroups.Reset()

	for _, ct := range w.shadowThingScratch {
		if rEntry := ct.rigid; rEntry != nil {
			rm := rEntry.rm
			if rm == nil || rm.LastSeenFrame != w.meshFrame {
				rm, _ = w.retrieveThingMesh(rEntry.faces)
				rEntry.rm = rm
			}
			if rm != nil && rm.SubMeshesLen > 0 {
				var g *shadowRigidGroup
				if rm.ShadowInstGroupFrame == w.shadowGroupToken {
					g = w.shadowInstGroups.Items[rm.ShadowInstGroupIdx]
				} else {
					rm.ShadowInstGroupFrame = w.shadowGroupToken
					rm.ShadowInstGroupIdx = w.shadowInstGroups.Len
					g = w.shadowInstGroups.Acquire()
					g.mesh = rm
					g.entries = g.entries[:0]
				}
				g.entries = append(g.entries, rEntry)
			}
			continue
		}

		thing := ct.thing
		ent := thing.GetEntity()
		if entry := w.thingsRender.PeekIfCurrent(ent); entry != nil && entry.committedFrame == w.meshFrame {
			w.fv.ReplayOpaqueBatches(entry.committedBaseIdx, entry.GetFrameVertices(), tb.streamDC)
			continue
		}

		sEntry := w.shadowStreamCache.retrieve(ent.GetId(), w.shadowFrame)
		if sEntry.compileFrame != w.shadowFrame {
			thing.Compile(tick)
			sEntry.compileFrame = w.shadowFrame
		}
		if !w.buildNonRigidThingCache(thing, sEntry) {
			continue
		}
		if sEntry.committedFrame != w.meshFrame {
			sEntry.committedBaseIdx = w.fv.CommitVerticesOnly(sEntry.GetFrameVertices())
			sEntry.committedFrame = w.meshFrame
		}
		if sEntry.committedBaseIdx >= 0 {
			sig := mixU64(ent.GetId(), sEntry.gen)
			*shadowExtraSig += sig
			*shadowExtraSig ^= bits.RotateLeft64(sig, 13)
			w.fv.ReplayOpaqueBatches(sEntry.committedBaseIdx, sEntry.GetFrameVertices(), tb.streamDC)
		}
	}

	hasInstances := false
	for gi := 0; gi < w.shadowInstGroups.Len; gi++ {
		g := w.shadowInstGroups.Items[gi]
		cnt := len(g.entries)
		if cnt == 0 {
			continue
		}
		hasInstances = true
		baseByteOff, baseFloat := w.shadowInstPool.Reserve(cnt)
		for k := range cnt {
			w.shadowInstPool.WriteMatrixAt(baseFloat+k*buffers.InstanceMatrixFloats, &g.entries[k].mat)
		}
		tb.instRanges = append(tb.instRanges, shadowInstRange{
			mesh:           g.mesh,
			baseByteOffset: baseByteOff,
			instanceCount:  int32(cnt),
		})
	}
	return hasInstances
}
