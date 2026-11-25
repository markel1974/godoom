package open_gl

import (
	"math"
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
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/gpu/draws"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/shaders"
	"github.com/markel1974/godoom/mr_tech/textures"
)

type shadowInstRange struct {
	mesh           *buffers.ResidentMesh
	baseByteOffset int32
	instanceCount  int32
}

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

type shadowThingCache struct {
	slots         map[uint64]*ThingRenderCache
	free          []*ThingRenderCache
	lastSweepTick uint64
}

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

func newShadowTileBuilder(ctx api.IContext) *shadowTileBuilder {
	return &shadowTileBuilder{
		dcRender:   draws.NewDrawCommandsRender(ctx, "shadow_tile", false),
		streamDC:   draws.NewDrawCommands(config.BlendModeOpaque, 128),
		instRanges: make([]shadowInstRange, 0, 32),
	}
}

func glToEngineVP(m [16]float32) [16]float32 {
	// Engine AABBs use Z-up (ex, ey, ez); OpenGL world positions use Y-up (ex, ez, -ey).
	return [16]float32{
		m[0], m[1], m[2], m[3],
		-m[8], -m[9], -m[10], -m[11],
		m[4], m[5], m[6], m[7],
		m[12], m[13], m[14], m[15],
	}
}

func mixU64(h, x uint64) uint64 {
	z := h ^ x
	z ^= z >> 30
	z *= 0xbf58476d1ce4e5b9
	z ^= z >> 27
	z *= 0x94d049bb133111eb
	z ^= z >> 31
	return z
}

func f32BitsStable(v float32) uint64 {
	if v == 0 {
		return 0
	}
	return uint64(math.Float32bits(v))
}

func mixPtrHash(p unsafe.Pointer) uint64 {
	return mixU64(14695981039346656037, uint64(uintptr(p)))
}

// BuilderShadow performs per-light frustum culling, tile cache reuse validation,
// and shadow draw compilation across all atlas tiles.
type BuilderShadow struct {
	bc                 *BuilderContext
	bStatic            *BuilderStatic
	bThing             *BuilderThings
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
	shadowFrame        uint64
}

// NewBuilderShadow allocates and initializes a BuilderShadow connected to bc, bStatic, and bThing.
func NewBuilderShadow(bc *BuilderContext, bStatic *BuilderStatic, bThing *BuilderThings) *BuilderShadow {
	bs := &BuilderShadow{
		bc:             bc,
		bStatic:        bStatic,
		bThing:         bThing,
		shadowFrustum:  aabb.NewFrustum(),
		shadowInstPool: buffers.NewInstancePool(bc.ctx, 256),
	}
	bs.shadowInstGroups.Init(32)
	for t := range shaders.MaxShadowTiles {
		bs.shadowBuilders[t] = newShadowTileBuilder(bc.ctx)
	}
	return bs
}

// DisableAllTiles marks every shadow atlas tile inactive and non-reusable.
func (w *BuilderShadow) DisableAllTiles() {
	for t := range shaders.MaxShadowTiles {
		w.shadowTiles[t].Active = false
		w.shadowTiles[t].Reuse = false
	}
}

// GetShadowTiles returns the compiled per-light shadow tile array for Depth.RenderWithTiles.
func (w *BuilderShadow) GetShadowTiles() *[shaders.MaxShadowTiles]shaders.ShadowTileData {
	return &w.shadowTiles
}

// ComputeShadows updates shadow tile draw data for active depth tiles, reusing cached tiles when possible.
func (w *BuilderShadow) ComputeShadows(vols *model.Volumes, th *model.Things, d *shaders.Depth) {
	if d == nil {
		w.DisableAllTiles()
		return
	}
	if w.bc.meshFrame == 0 {
		w.bc.meshFrame = 1
	}
	w.shadowFrame++

	w.bStatic.TryBuildStaticAtlas(vols)

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
		// Advance shadowExtraGen separately so Main uploads off-camera vertices without invalidating BuilderContext.Commit.
		w.bc.shadowExtraGen++
		w.prevShadowExtraSig = shadowExtraSig
	}
}

func (w *BuilderShadow) collectShadowVolumes(vols *model.Volumes) (acc uint64, xorHash uint64, count int) {
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
		} else if entry := w.bStatic.GetEntry(faces); entry != nil && !entry.detached {
			if entry.chunk != nil {
				entry.chunk.lastSeenFrame = w.bc.meshFrame
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

func (w *BuilderShadow) updateShadowRigidEntry(rEntry *shadowRigidEntry, entID uint64, ent *rigid_body.Entity, rigidFaces *volumes.Faces) uint64 {
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
		rEntry.rm = w.bThing.FindCachedMesh(rigidFaces)
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

func (w *BuilderShadow) hashStreamingThing(entID uint64, ent *rigid_body.Entity, thing ithing.IThing, entry *ThingRenderCache) uint64 {
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

func (w *BuilderShadow) collectShadowThings(th *model.Things, tick uint64) (acc uint64, xorHash uint64, count int) {
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
			entry := w.bThing.thingsRender.PeekIfCurrent(ent)
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

func pushShadowStaticSubMeshes(tb *shadowTileBuilder, mesh *buffers.ResidentMesh, startIdx, endIdx int) {
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

func (w *BuilderShadow) buildShadowTileStatic(tb *shadowTileBuilder, vols *model.Volumes) {
	for _, vol := range w.shadowVolScratch {
		w.bStatic.ResolveMesh(vol, vols)
	}
	for _, vol := range w.shadowVolScratch {
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

func (w *BuilderShadow) buildShadowTileThings(tb *shadowTileBuilder, tick uint64, shadowExtraSig *uint64) bool {
	if len(w.shadowThingScratch) == 0 {
		return false
	}
	w.shadowGroupToken++
	w.shadowInstGroups.Reset()

	for _, ct := range w.shadowThingScratch {
		if rEntry := ct.rigid; rEntry != nil {
			rm := rEntry.rm
			if rm == nil || rm.LastSeenFrame != w.bc.meshFrame || rm.NeedsAnimRebuild() {
				rm, _ = w.bThing.RetrieveMesh(rEntry.faces)
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
		if entry := w.bThing.thingsRender.PeekIfCurrent(ent); entry != nil && entry.committedFrame == w.bc.meshFrame {
			w.bc.fv.ReplayOpaqueBatches(entry.committedBaseIdx, entry.GetFrameVertices(), tb.streamDC)
			continue
		}

		sEntry := w.shadowStreamCache.retrieve(ent.GetId(), w.shadowFrame)
		if sEntry.compileFrame != w.shadowFrame {
			thing.Compile(tick)
			sEntry.compileFrame = w.shadowFrame
		}
		if !w.bThing.BuildNonRigidCache(thing, sEntry) {
			continue
		}
		if sEntry.committedFrame != w.bc.meshFrame {
			sEntry.committedBaseIdx = w.bc.fv.CommitVerticesOnly(sEntry.GetFrameVertices())
			sEntry.committedFrame = w.bc.meshFrame
		}
		if sEntry.committedBaseIdx >= 0 {
			sig := mixU64(ent.GetId(), sEntry.gen)
			*shadowExtraSig += sig
			*shadowExtraSig ^= bits.RotateLeft64(sig, 13)
			w.bc.fv.ReplayOpaqueBatches(sEntry.committedBaseIdx, sEntry.GetFrameVertices(), tb.streamDC)
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
