package open_gl

import (
	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/model"
	"github.com/markel1974/godoom/mr_tech/model/ithing"
	"github.com/markel1974/godoom/mr_tech/model/vertices"
	"github.com/markel1974/godoom/mr_tech/model/volumes"
	"github.com/markel1974/godoom/mr_tech/physics/aabb"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/gpu/buffers"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/gpu/draws"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// RigidInstance stores world-space translation and quaternion orientation for one instanced rigid entity.
type RigidInstance struct {
	cx, cy, cz     float64
	qx, qy, qz, qw float64
}

// RigidInstanceGroup batches instances sharing one GPU ResidentMesh and upload offset.
type RigidInstanceGroup struct {
	mesh           *buffers.ResidentMesh
	instances      buffers.PtrPool[RigidInstance]
	baseByteOffset int32
}

// RigidTransItem tracks a single translucent instanced submesh draw for back-to-front sorting.
type RigidTransItem struct {
	mesh           *buffers.ResidentMesh
	sub            *buffers.ResidentSubMesh
	baseByteOffset int32
	cx, cy, cz     float32
}

type thingMeshPtrEntry struct {
	rm        *buffers.ResidentMesh
	geomHash  uint64
	version   uint64
	faceCount int
}

// BuilderThings manages entity render caches, GPU instancing for rigid objects,
// and CPU vertex staging for morphing 3D models, sprites, and UI elements.
type BuilderThings struct {
	bc               *BuilderContext
	thingMeshesByPtr map[*volumes.Faces]thingMeshPtrEntry
	thingMeshes      map[uint64][]*buffers.ResidentMesh
	thingsRender     *ThingsRender
	instancePool     *buffers.InstancePool
	instGroups       buffers.PtrPool[RigidInstanceGroup]
	instFrame        uint64
	playerId         uint64
}

// NewBuilderThings initializes a BuilderThings bound to bc.
func NewBuilderThings(bc *BuilderContext) *BuilderThings {
	var playerID uint64
	if bc.player != nil && bc.player.GetEntity() != nil {
		playerID = bc.player.GetEntity().GetId()
	}
	bt := &BuilderThings{
		bc:               bc,
		playerId:         playerID,
		thingMeshesByPtr: make(map[*volumes.Faces]thingMeshPtrEntry, 512),
		thingMeshes:      make(map[uint64][]*buffers.ResidentMesh, 256),
		thingsRender:     NewThingsRender(),
		instancePool:     buffers.NewInstancePool(bc.ctx, 256),
	}
	bt.instGroups.Init(32)
	return bt
}

func (w *BuilderThings) cacheThingMeshPtr(rigidFaces *volumes.Faces, rm *buffers.ResidentMesh, gh, ver uint64, faceCount int) {
	w.thingMeshesByPtr[rigidFaces] = thingMeshPtrEntry{
		rm:        rm,
		geomHash:  gh,
		version:   ver,
		faceCount: faceCount,
	}
}

// RetrieveMesh returns a resident mesh for rigidFaces, reusing or rebuilding cached mesh data when needed.
// The returned bool reports whether the mesh was rebuilt during the current mesh frame.
func (w *BuilderThings) RetrieveMesh(rigidFaces *volumes.Faces) (*buffers.ResidentMesh, bool) {
	if rm := w.FindCachedMesh(rigidFaces); rm != nil {
		needsAnim := rm.NeedsAnimRebuild()
		if rm.LastSeenFrame != w.bc.meshFrame || needsAnim {
			rm.LastSeenFrame = w.bc.meshFrame
			if (rm.SrcFaces == rigidFaces && rm.NeedsRebuild(rigidFaces, true)) || needsAnim {
				rm.Rebuild(rigidFaces, w.bc.LookupTexLayer)
				rm.RebuiltFrame = w.bc.meshFrame
			}
		}
		return rm, rm.RebuiltFrame == w.bc.meshFrame
	}
	gh := rigidFaces.GetGeomHash()
	ver := rigidFaces.GetVersion()
	faceCount := rigidFaces.Len()
	rm := buffers.NewResidentMesh(w.bc.ctx)
	rm.LastSeenFrame = w.bc.meshFrame
	rm.RebuiltFrame = w.bc.meshFrame
	rm.Rebuild(rigidFaces, w.bc.LookupTexLayer)
	w.thingMeshes[gh] = append(w.thingMeshes[gh], rm)
	w.cacheThingMeshPtr(rigidFaces, rm, gh, ver, faceCount)
	return rm, true
}

// Process stages visible things, routes rigid objects into GPU instancing,
// and streams non-rigid geometry into FrameVertices.
func (w *BuilderThings) Process(things *model.Things, frustumFront *aabb.Frustum) bool {
	if w.bc.meshFrame == 0 {
		w.bc.meshFrame = 1
	}
	if things == nil {
		if w.bc.frameGen == 0 {
			w.resetInstancedPools()
			w.instancePool.Reset()
			w.bc.lastInstVBO = 0
		}
		return false
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
		entry := w.thingsRender.Retrieve(ent)
		thing.Compile(tick)
		entry.compileFrame = w.bc.meshFrame

		if rs, ok := thing.(ithing.IRigidRenderSource); ok {
			if rigidFaces := rs.GetRigidFaces(); rigidFaces != nil && rigidFaces.Len() > 0 {
				rm, meshRebuilt := w.RetrieveMesh(rigidFaces)
				tPosX, tPosY, zBot := ent.GetCenter()
				qx, qy, qz, qw := ent.GetQRotation().Get()
				oX, oY, oZ := float32(tPosX), float32(zBot), float32(-tPosY)
				fqx, fqy, fqz, fqw := float32(qx), float32(qy), float32(qz), float32(qw)
				if meshRebuilt || !entry.EqualRigid(rigidFaces, rigidFaces.GetVersion(), rigidFaces.Len(), oX, oY, oZ, fqx, fqy, fqz, fqw) {
					entry.AssignRigid(rigidFaces, rigidFaces.GetVersion(), rigidFaces.Len(), oX, oY, oZ, fqx, fqy, fqz, fqw)
				}
				w.thingsRender.AddRef(entry)
				if rm.SubMeshesLen > 0 {
					addRigidInstance(rm, tPosX, tPosY, zBot, qx, qy, qz, qw)
				}
				return
			}
		}

		if !w.BuildNonRigidCache(thing, entry) {
			return
		}
		entry.committedBaseIdx = w.bc.fv.CommitWithOffset(entry.GetFrameVertices(), w.bc.modes)
		entry.committedFrame = w.bc.meshFrame
		w.thingsRender.AddRef(entry)
	}

	qOcclusion := func(object aabb.IObject) bool {
		thing := object.(ithing.IThing)
		if !thing.IsVisible() {
			return false
		}
		entId := thing.GetEntity().GetId()
		if w.bc.player == nil || w.playerId != entId {
			things.StageAdd(thing)
		}
		if w.bc.occlusion != nil && thing.GetKind() != config.ThingDoorDef {
			ab := thing.GetAABB()
			occState := w.bc.occlusion.GetState(entId)
			if occState != nil && !occState.IsVisible {
				w.bc.occlusion.Add(ab, occState)
				return false
			}
			w.bc.occlusion.Add(ab, occState)
		}
		pushPassThing(thing)
		return false
	}

	qStatic := func(object aabb.IObject) bool {
		pushPassThing(object.(ithing.IThing))
		return false
	}

	if w.bc.player != nil {
		things.StageAdd(w.bc.player)
	}
	things.QueryStatic(qStatic)
	if frustumFront != nil {
		things.QueryFrustum(frustumFront, qOcclusion)
	}
	things.StageCommit()

	thingsChanged := w.thingsRender.Commit()
	if thingsChanged || w.bc.frameGen == 0 {
		w.rebuildInstancedDraws()
	}

	return thingsChanged
}

func (w *BuilderThings) resetInstancedPools() {
	w.bc.instOpaque.Reset()
	w.bc.instLiquid.Reset()
	w.bc.instAdditive.Reset()
	w.bc.instTransSrc.Reset()
}

func (w *BuilderThings) rebuildInstancedDraws() {
	w.resetInstancedPools()
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
	w.bc.lastInstVBO = instVBO

	for gi := 0; gi < w.instGroups.Len; gi++ {
		g := w.instGroups.Items[gi]
		if g.instances.Len == 0 {
			continue
		}
		w.appendInstancedGroup(g, instVBO)
	}
}

func (w *BuilderThings) appendInstancedGroup(g *RigidInstanceGroup, instVBO uint32) {
	cnt := int32(g.instances.Len)
	for sIdx := 0; sIdx < g.mesh.SubMeshesLen; sIdx++ {
		sub := g.mesh.SubMeshes[sIdx]
		if sub.IndexCount == 0 {
			continue
		}
		if sub.BlendMode == config.BlendModeTransparent {
			for k := 0; k < int(cnt); k++ {
				inst := g.instances.Items[k]
				rt := w.bc.instTransSrc.Acquire()
				rt.mesh = g.mesh
				rt.sub = sub
				rt.baseByteOffset = g.baseByteOffset + int32(k*buffers.InstanceMatrixFloats*4)
				rt.cx = float32(inst.cx)
				rt.cy = float32(inst.cz)
				rt.cz = float32(-inst.cy)
			}
			continue
		}
		var dst *buffers.PtrPool[draws.DrawItemInstanced]
		switch sub.BlendMode {
		case config.BlendModeLiquid:
			dst = &w.bc.instLiquid
		case config.BlendModeAdditive:
			dst = &w.bc.instAdditive
		default:
			dst = &w.bc.instOpaque
		}
		item := dst.Acquire()
		item.Mesh = g.mesh
		item.Sub = sub
		item.InstanceVBO = instVBO
		item.BaseByteOffset = g.baseByteOffset
		item.InstanceCount = cnt
	}
}

// BuildNonRigidCache rebuilds or reuses CPU-staged vertex batches in entry2 for a streaming or morphed thing.
func (w *BuilderThings) BuildNonRigidCache(thing ithing.IThing, entry2 *ThingRenderCache) bool {
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
			if w.bc.tex != nil {
				tId, _ = w.bc.tex.Get(tex)
			}
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

// FindCachedMesh returns a cached resident mesh matching rigidFaces in O(1) after first verification, or nil if none exists.
func (w *BuilderThings) FindCachedMesh(rigidFaces *volumes.Faces) *buffers.ResidentMesh {
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
