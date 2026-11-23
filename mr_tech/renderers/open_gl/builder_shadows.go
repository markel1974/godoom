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

type shadowInstRange struct {
	mesh           *buffers.ResidentMesh
	baseByteOffset int32
	instanceCount  int32
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

func newShadowTileBuilder(w *BuilderVolume) *shadowTileBuilder {
	return &shadowTileBuilder{
		dcRender:   draws.NewDrawCommandsRender(w.ctx, "shadow_tile", false),
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

// GetShadowTiles returns the per-light shadow tile descriptors prepared by ComputeShadows.
func (w *BuilderVolume) GetShadowTiles() *[shaders.MaxShadowTiles]shaders.ShadowTileData {
	return &w.shadowTiles
}

// ComputeShadows performs per-light frustum culling against world volumes and things,
// checks per-buffer tile cache validity on Depth, and compiles draw commands only for dirty shadow atlas tiles.
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

	w.shadowStreamCache.sweep(w.meshFrame)

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
			w.shadowTiles[t] = shaders.ShadowTileData{
				Active:    true,
				Matrix:    mat,
				SceneHash: sceneHash,
				Reuse:     true,
			}
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

		w.shadowTiles[t] = shaders.ShadowTileData{
			Active:    true,
			RenderDC:  tb.dcRender,
			Matrix:    mat,
			SceneHash: sceneHash,
			Reuse:     false,
		}
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

func (w *BuilderVolume) findCachedThingMesh(rigidFaces *volumes.Faces) *buffers.ResidentMesh {
	gh := rigidFaces.GetGeomHash()
	for _, rm := range w.thingMeshes[gh] {
		if rm.SrcFaces == rigidFaces || (rm.FaceCount == rigidFaces.Len() && rm.SrcFaces.GetGeomHash() == gh && facesEquivalent(rigidFaces, rm.SrcFaces)) {
			return rm
		}
	}
	return nil
}

func (w *BuilderVolume) hashRigidThing(entID uint64, ent *rigid_body.Entity, rigidFaces *volumes.Faces) uint64 {
	tPosX, tPosY, zBot := ent.GetCenter()
	qx, qy, qz, qw := ent.GetQRotation().Get()
	h := mixU64(entID, mixPtrHash(unsafe.Pointer(rigidFaces)))
	h = mixU64(h, rigidFaces.GetVersion())
	h = mixU64(h, uint64(rigidFaces.Len()))
	h = mixU64(h, f32BitsStable(float32(tPosX)))
	h = mixU64(h, f32BitsStable(float32(tPosY)))
	h = mixU64(h, f32BitsStable(float32(zBot)))
	h = mixU64(h, f32BitsStable(float32(qx)))
	h = mixU64(h, f32BitsStable(float32(qy)))
	h = mixU64(h, f32BitsStable(float32(qz)))
	h = mixU64(h, f32BitsStable(float32(qw)))
	if rm := w.findCachedThingMesh(rigidFaces); rm != nil && rm.AnimLen > 0 {
		for i := 0; i < rm.AnimLen; i++ {
			h = mixU64(h, mixPtrHash(unsafe.Pointer(rm.AnimMats[i].GetCurrentFrame())))
		}
	}
	return h
}

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
		if entry := w.thingsRender.PeekIfCurrent(ent); entry != nil {
			if rs, okR := thing.(ithing.IRigidRenderSource); okR {
				if rigidFaces := rs.GetRigidFaces(); rigidFaces != nil && rigidFaces.Len() > 0 {
					h = w.hashRigidThing(entID, ent, rigidFaces)
				}
			}
			if h == 0 {
				h = w.hashStreamingThing(entID, ent, thing, entry)
			}
		} else {
			if sEntry, okCache := w.shadowStreamCache.slots[entID]; okCache {
				if sEntry.compileFrame != w.meshFrame {
					thing.Compile(tick)
					sEntry.compileFrame = w.meshFrame
				}
			} else {
				thing.Compile(tick)
			}
			if rs, okR := thing.(ithing.IRigidRenderSource); okR {
				if rigidFaces := rs.GetRigidFaces(); rigidFaces != nil && rigidFaces.Len() > 0 {
					h = w.hashRigidThing(entID, ent, rigidFaces)
				}
			}
			if h == 0 {
				sEntry := w.shadowStreamCache.retrieve(entID, w.meshFrame)
				sEntry.compileFrame = w.meshFrame
				h = w.hashStreamingThing(entID, ent, thing, sEntry)
			}
		}

		if h == 0 {
			return false
		}
		w.shadowThingScratch = append(w.shadowThingScratch, thing)
		acc += h
		xorHash ^= bits.RotateLeft64(h, 23)
		count++
		return false
	})
	return acc, xorHash, count
}

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

func (w *BuilderVolume) buildShadowTileThings(tb *shadowTileBuilder, tick uint64, shadowExtraSig *uint64) bool {
	if len(w.shadowThingScratch) == 0 {
		return false
	}
	w.shadowGroupToken++
	w.shadowInstGroups.Reset()

	for _, thing := range w.shadowThingScratch {
		ent := thing.GetEntity()
		if rs, ok := thing.(ithing.IRigidRenderSource); ok {
			if rigidFaces := rs.GetRigidFaces(); rigidFaces != nil && rigidFaces.Len() > 0 {
				rm, _ := w.retrieveThingMesh(rigidFaces)
				if rm.SubMeshesLen > 0 {
					tPosX, tPosY, zBot := ent.GetCenter()
					qx, qy, qz, qw := ent.GetQRotation().Get()
					var g *RigidInstanceGroup
					if rm.ShadowInstGroupFrame == w.shadowGroupToken {
						g = w.shadowInstGroups.Items[rm.ShadowInstGroupIdx]
					} else {
						rm.ShadowInstGroupFrame = w.shadowGroupToken
						rm.ShadowInstGroupIdx = w.shadowInstGroups.Len
						g = w.shadowInstGroups.Acquire()
						g.mesh = rm
						g.instances.Reset()
					}
					inst := g.instances.Acquire()
					inst.cx = tPosX
					inst.cy = tPosY
					inst.cz = zBot
					inst.qx = qx
					inst.qy = qy
					inst.qz = qz
					inst.qw = qw
				}
				continue
			}
		}

		if entry := w.thingsRender.PeekIfCurrent(ent); entry != nil && entry.committedFrame == w.meshFrame {
			w.fv.ReplayOpaqueBatches(entry.committedBaseIdx, entry.GetFrameVertices(), tb.streamDC)
			continue
		}

		sEntry := w.shadowStreamCache.retrieve(ent.GetId(), w.meshFrame)
		if sEntry.compileFrame != w.meshFrame {
			thing.Compile(tick)
			sEntry.compileFrame = w.meshFrame
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
		cnt := g.instances.Len
		if cnt == 0 {
			continue
		}
		hasInstances = true
		baseByteOff, baseFloat := w.shadowInstPool.Reserve(cnt)
		for k := range cnt {
			inst := g.instances.Items[k]
			w.shadowInstPool.WriteTransformAt(baseFloat+k*buffers.InstanceMatrixFloats, inst.cx, inst.cy, inst.cz, inst.qx, inst.qy, inst.qz, inst.qw)
		}
		tb.instRanges = append(tb.instRanges, shadowInstRange{
			mesh:           g.mesh,
			baseByteOffset: baseByteOff,
			instanceCount:  int32(cnt),
		})
	}
	return hasInstances
}
