package open_gl

import (
	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/model"
	"github.com/markel1974/godoom/mr_tech/model/ithing"
	"github.com/markel1974/godoom/mr_tech/model/vertices"
	"github.com/markel1974/godoom/mr_tech/model/volumes"
	"github.com/markel1974/godoom/mr_tech/physics/aabb"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/gpu/buffers"
	"github.com/markel1974/godoom/mr_tech/textures"
)

func (w *BuilderVolume) cacheThingMeshPtr(rigidFaces *volumes.Faces, rm *buffers.ResidentMesh, gh, ver uint64, faceCount int) {
	if w.thingMeshesByPtr == nil {
		w.thingMeshesByPtr = make(map[*volumes.Faces]thingMeshPtrEntry, 512)
	}
	w.thingMeshesByPtr[rigidFaces] = thingMeshPtrEntry{
		rm:        rm,
		geomHash:  gh,
		version:   ver,
		faceCount: faceCount,
	}
}

// retrieveThingMesh returns a resident mesh for the given faces, reusing or rebuilding cached mesh data when needed.
// The returned bool reports whether the mesh was rebuilt during the current mesh frame.
func (w *BuilderVolume) retrieveThingMesh(rigidFaces *volumes.Faces) (*buffers.ResidentMesh, bool) {
	if rm := w.findCachedThingMesh(rigidFaces); rm != nil {
		if rm.LastSeenFrame != w.meshFrame {
			rm.LastSeenFrame = w.meshFrame
			if (rm.SrcFaces == rigidFaces && rm.NeedsRebuild(rigidFaces, true)) || rm.NeedsAnimRebuild() {
				rm.Rebuild(rigidFaces, w.texLookupFn)
				rm.RebuiltFrame = w.meshFrame
			}
		}
		return rm, rm.RebuiltFrame == w.meshFrame
	}
	gh := rigidFaces.GetGeomHash()
	ver := rigidFaces.GetVersion()
	faceCount := rigidFaces.Len()
	rm := buffers.NewResidentMesh(w.ctx)
	rm.LastSeenFrame = w.meshFrame
	rm.RebuiltFrame = w.meshFrame
	rm.Rebuild(rigidFaces, w.texLookupFn)
	w.thingMeshes[gh] = append(w.thingMeshes[gh], rm)
	w.cacheThingMeshPtr(rigidFaces, rm, gh, ver, faceCount)
	return rm, true
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
	if w.thingMeshesByPtr == nil {
		w.thingMeshesByPtr = make(map[*volumes.Faces]thingMeshPtrEntry, 512)
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
