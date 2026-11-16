package volumes

import (
	"fmt"
	"math"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/model/materials"
	"github.com/markel1974/godoom/mr_tech/physics/aabb"
	"github.com/markel1974/godoom/mr_tech/physics/articulated_dynamics"
	"github.com/markel1974/godoom/mr_tech/physics/collisions"
	"github.com/markel1974/godoom/mr_tech/physics/rigid_body"
)

// PartSpawnFunc constructs an IThing wrapping one segment's IVertices so all parts live in Things.tree from creation.
type PartSpawnFunc func(partVerts IVertices, collisionGroup uint64) IThing

// PartReleaseFunc notifies the world Thing manager when an attached segment transitions to autonomous physics mid-tick.
type PartReleaseFunc func(part IThing)

// RagdollJointAddFunc registers a bilateral PGS joint constraint with optional breakability on the child limb.
type RagdollJointAddFunc func(joint *collisions.JointConstraint, childThing IThing, canDetach bool, breakThreshold float64)

// IPartedVertices is optionally implemented by composite models whose segments live as autonomous IThings in Things.tree.
type IPartedVertices interface {
	SetupParts(spawnPart PartSpawnFunc, onRelease PartReleaseFunc, addJoint RagdollJointAddFunc)
}

// Vertices3D represents a composite animated 3D model whose linked segments each have a Volume, Solver, and node in Things.tree.
type Vertices3D struct {
	id            string
	mass          float64
	owner         IThing
	solver        *collisions.Solver
	ent           *rigid_body.Entity
	actions       *Vertices3DActions
	currentAction int
	links         *articulated_dynamics.Links
	elements      *Vertices3DElements
	joints        *articulated_dynamics.Joints
	partViews     []*Vertices3DPart
	partThings    []IThing
	partScratchQ  *aabb.Quaternion
	onPartRelease PartReleaseFunc
	onAddJoint    RagdollJointAddFunc
	minX          float64
	minY          float64
	minZ          float64
	maxX          float64
	maxY          float64
	maxZ          float64
	compiled      bool
	compiledTick  uint64
	compiledQX    float64
	compiledQY    float64
	compiledQZ    float64
	compiledQW    float64
	compiledCX    float64
	compiledCY    float64
	compiledCZ    float64
}

// NewVertices3D creates a Vertices3D model from cfg and materials, centering each segment frame on its geometric center.
func NewVertices3D(cfg *config.Thing, materials *materials.Materials) *Vertices3D {
	v := &Vertices3D{
		id:            cfg.Id,
		mass:          cfg.Mass,
		links:         articulated_dynamics.NewLinks(),
		joints:        articulated_dynamics.NewJoints(),
		elements:      NewVertices3DElements(cfg, materials),
		actions:       NewVertices3DActions(),
		currentAction: -1,
		partScratchQ:  aabb.NewQuaternionIdentity(),
	}
	configLinks, cErr := v.createConfigLinks(cfg.Model3D)
	if cErr != nil {
		panic(cErr)
	}
	if err := v.links.Setup(configLinks); err != nil {
		panic(err)
	}
	configJoints, cErr := v.createConfigJoints(cfg.Model3D, v.elements)
	if cErr != nil {
		panic(cErr)
	}
	if err := v.joints.Setup(configJoints, v.links); err != nil {
		panic(err)
	}
	if err := v.actions.Setup(cfg.Model3D, v.elements); err != nil {
		panic(err)
	}

	numParts := v.elements.Len()
	v.partViews = make([]*Vertices3DPart, numParts)
	for i := 0; i < numParts; i++ {
		v.partViews[i] = NewVertices3DPart(v, v.elements.GetEntry(i), i, 0)
	}

	v.ent = v.elements.GetSolver().GetEntity()
	v.Compile(0)
	if v.minZ <= v.maxZ {
		if baseZ := -(v.minZ + v.maxZ) * 0.5; math.Abs(baseZ) > 1e-6 {
			v.joints.SetBaseOffsetZ(baseZ)
			v.compiled = false
			v.Compile(0)
		}
	}

	v.elements.UpdateEntityBounds(v.minX, v.minY, v.minZ, v.maxX, v.maxY, v.maxZ)
	return v
}

// Setup binds the owner locomotion virtual volume to thing.
func (v *Vertices3D) Setup(thing IThing) error {
	if err := v.elements.Setup(thing); err != nil {
		return err
	}
	v.owner = thing
	v.solver = v.elements.GetSolver()
	v.ent = v.solver.GetEntity()
	v.solver.SetCollisionGroup(v.ent.GetId())
	return nil
}

// SetupParts instantiates a standalone IThing for each segment so all segments reside in Things.tree from creation.
func (v *Vertices3D) SetupParts(spawnPart PartSpawnFunc, onRelease PartReleaseFunc, addJoint RagdollJointAddFunc) {
	v.onPartRelease = onRelease
	v.onAddJoint = addJoint
	numParts := len(v.partViews)
	v.partThings = make([]IThing, numParts)
	groupID := v.ent.GetId()
	for i, pv := range v.partViews {
		pv.groupID = groupID
		if spawnPart != nil {
			v.partThings[i] = spawnPart(pv, groupID)
		}
	}
	v.compiled = false
	v.Compile(v.compiledTick)
}

// GetParts returns the slice of per-segment IThings belonging to this model.
func (v *Vertices3D) GetParts() []IThing {
	return v.partThings
}

// GetSolver returns the owner locomotion collision solver.
func (v *Vertices3D) GetSolver() *collisions.Solver {
	return v.solver
}

// QueryOverlaps queries face overlaps across all attached segment volumes.
func (v *Vertices3D) QueryOverlaps(object aabb.IObject, callback func(object aabb.IObject) bool) {
	v.elements.vVolume.QueryOverlaps(object, callback)
}

// GetAABB returns the enclosing world-space AABB for the owner's locomotion entity.
func (v *Vertices3D) GetAABB() *aabb.AABB {
	ent := v.ent
	ab := ent.GetAABB()
	if v.minX <= v.maxX {
		cx, cy, cz := ent.GetCenter()
		ab.Rebuild(cx+v.minX, cy+v.minY, cz+v.minZ, cx+v.maxX, cy+v.maxY, cz+v.maxZ)
	}
	return ab
}

// ApplyImpact forwards an owner-level impact directly to the primary segment handler.
func (v *Vertices3D) ApplyImpact(fx, fy, fz, px, py, pz float64) {
	if len(v.partViews) == 0 {
		return
	}
	v.ApplyPartImpact(0, fx, fy, fz, px, py, pz)
}

// ApplyPartImpact handles a localized hit directly on segment partIdx identified by QueryRay in Things.tree,
// applying physical impulse to the living owner and detaching any stressed joint into autonomous 6-DoF motion.
func (v *Vertices3D) ApplyPartImpact(partIdx int, fx, fy, fz, px, py, pz float64) {
	fMag := math.Sqrt(fx*fx + fy*fy + fz*fz)
	if fMag < 1e-6 {
		return
	}
	dirX, dirY, dirZ := fx/fMag, fy/fMag, fz/fMag

	numParts := len(v.partViews)
	if partIdx < 0 || partIdx >= numParts {
		v.joints.RecordLastHit(dirX, dirY, dirZ, fMag, -1)
		return
	}

	v.joints.RecordLastHit(dirX, dirY, dirZ, fMag, partIdx)
	if v.joints.IsRagdoll() || v.partViews[partIdx].IsPhysicsFree() {
		return
	}

	// Deliver physical knockback to the living upright owner while attached.
	v.ent.AddForceAtPoint(fx, fy, fz, px, py, pz)

	if numParts <= 1 || !v.joints.CanDetachMore() {
		return
	}

	normImpulse := math.Min(fMag/500000.0, 1.6)
	if normImpulse < 0.45 {
		normImpulse = 0.45
	}

	curr := partIdx
	attenuation := 1.0
	for depth := 0; depth < 3 && curr > 0 && curr < numParts; depth++ {
		if v.joints.IsDetached(curr) || v.partViews[curr].IsPhysicsFree() {
			break
		}
		j := v.joints.GetJoint(curr)
		if j.CanDetach() {
			stress := 0.75 * normImpulse * attenuation
			if j.AccumulateStress(stress) {
				v.releaseSubtree(curr, dirX, dirY, dirZ, fMag)
				break
			}
		}
		curr = v.links.GetParent(curr)
		attenuation *= 0.60
	}
}

// releaseSubtree detaches partIdx and its attached descendants directly in-place inside Things.tree,
// adding internal PGS JointConstraints if multiple segments detach together.
func (v *Vertices3D) releaseSubtree(partIdx int, dirX, dirY, dirZ, fMag float64) {
	v.compiled = false
	v.Compile(v.compiledTick)

	numParts := len(v.partViews)
	inSubtree := make([]bool, numParts)
	eoLen := v.links.ExecutionOrderLen()
	for eoIdx := 0; eoIdx < eoLen; eoIdx++ {
		i := v.links.ExecutionOrderAt(eoIdx)
		if i == partIdx {
			inSubtree[i] = !v.partViews[i].IsPhysicsFree()
		} else if pIdx := v.links.GetParent(i); pIdx >= 0 && inSubtree[pIdx] && !v.joints.IsDetached(i) && !v.partViews[i].IsPhysicsFree() {
			inSubtree[i] = true
		}
	}

	pvx, pvy, pvz := v.ent.GetVelocity()
	j := v.joints.GetJoint(partIdx)
	sideSign := 1.0
	if partIdx&1 == 1 {
		sideSign = -1.0
	}
	v.joints.DetachPart(partIdx, dirX, dirY, dirZ, fMag, pvx, pvy, pvz)
	vx, vy, vz, wx, wy, wz := j.Detach(dirX, dirY, dirZ, fMag, sideSign)
	vx += pvx
	vy += pvy
	vz += pvz

	subtreeCount := 0
	for i := 0; i < numParts; i++ {
		if inSubtree[i] {
			subtreeCount++
		}
	}
	var subGroupID uint64
	if subtreeCount > 1 && v.partThings[partIdx] != nil {
		subGroupID = v.partThings[partIdx].GetEntity().GetId()
	}

	for i := 0; i < numParts; i++ {
		if !inSubtree[i] {
			continue
		}
		v.partViews[i].ReleasePhysics(subGroupID, vx, vy, vz, wx, wy, wz)
		if v.onPartRelease != nil && v.partThings[i] != nil {
			v.onPartRelease(v.partThings[i])
		}
	}

	if subtreeCount > 1 && v.onAddJoint != nil {
		for i := 0; i < numParts; i++ {
			if !inSubtree[i] || i == partIdx {
				continue
			}
			pIdx := v.links.GetParent(i)
			if pIdx < 0 || !inSubtree[pIdx] {
				continue
			}
			jc := v.buildJointConstraint(pIdx, i)
			jCfg := v.joints.GetJoint(i)
			v.onAddJoint(jc, v.partThings[i], jCfg.CanDetach(), jCfg.GetDetachThreshold())
		}
	}
}

// SetRagdollMode transitions all remaining attached segments in-place to 6-DoF rigid-body ragdoll physics
// connected by bilateral PGS JointConstraints and disables the owner's single locomotion collider.
func (v *Vertices3D) SetRagdollMode() {
	if v.joints.IsRagdoll() {
		return
	}
	v.compiled = false
	v.Compile(v.compiledTick)

	numParts := len(v.partViews)
	if numParts == 0 {
		return
	}

	ent := v.ent
	pvx, pvy, pvz := ent.GetVelocity()
	hitDx, hitDy, hitDz, hitMag, hitPart := v.joints.LastHitImpulse()
	if hitDx*hitDx+hitDy*hitDy+hitDz*hitDz < 1e-6 {
		ax0, ay0, _ := ent.GetAxes0()
		hitDx, hitDy, hitDz = ax0, ay0, 0
	}

	// If the killing blow left a detachable extremity critically stressed, sever it before linking ragdoll joints.
	if v.joints.CanDetachMore() && hitPart > 0 && hitPart < numParts && !v.partViews[hitPart].IsPhysicsFree() {
		if hj := v.joints.GetJoint(hitPart); hj.IsCriticallyDamaged() {
			v.releaseSubtree(hitPart, hitDx, hitDy, hitDz, hitMag)
		}
	}

	v.joints.ActivateRagdoll(pvx, pvy, pvz)

	hitKick := 35.0
	fallSpin := 3.5
	if hitMag > 0 {
		scale := math.Min(hitMag/500000.0, 2.0)
		hitKick = math.Max(hitKick, scale*65.0)
		fallSpin = math.Max(fallSpin, scale*4.5)
	}
	groupID := ent.GetId()

	inRagdoll := make([]bool, numParts)
	for i := 0; i < numParts; i++ {
		if v.partViews[i].IsPhysicsFree() {
			continue
		}
		inRagdoll[i] = true
		vx, vy, vz := pvx, pvy, pvz
		var wx, wy, wz float64
		if i == 0 || i == hitPart {
			vx += hitDx * hitKick
			vy += hitDy * hitKick
			vz += hitDz*hitKick + 25.0
			wx = -hitDy * fallSpin
			wy = hitDx * fallSpin
		}
		v.partViews[i].ReleasePhysics(groupID, vx, vy, vz, wx, wy, wz)
		if v.onPartRelease != nil && v.partThings[i] != nil {
			v.onPartRelease(v.partThings[i])
		}
	}

	if v.onAddJoint != nil {
		for i := 1; i < numParts; i++ {
			if !inRagdoll[i] {
				continue
			}
			pIdx := v.links.GetParent(i)
			if pIdx < 0 || !inRagdoll[pIdx] {
				continue
			}
			jc := v.buildJointConstraint(pIdx, i)
			jCfg := v.joints.GetJoint(i)
			v.onAddJoint(jc, v.partThings[i], jCfg.CanDetach(), jCfg.GetDetachThreshold())
		}
	}

	if v.solver != nil {
		v.solver.SetActive(false)
	}
}

func (v *Vertices3D) buildJointConstraint(pIdx, i int) *collisions.JointConstraint {
	pEntry := v.elements.GetEntry(pIdx)
	cEntry := v.elements.GetEntry(i)
	mcP := pEntry.FrameCenterA()
	mcC := cEntry.FrameCenterA()

	var ax, ay, az float64
	restQ := aabb.NewQuaternionIdentity()
	parentVol := v.elements.volsA[pIdx]
	if parentVol == nil && len(pEntry.volumes) > 0 {
		parentVol = pEntry.volumes[0]
	}
	tagIdx := v.links.GetTagIndex(i)
	if tagIdx >= 0 && parentVol != nil {
		if tagAxis := parentVol.GetVertexTag(tagIdx); tagAxis != nil {
			tx, ty, tz := tagAxis.GetOrigin()
			ax = tx - mcP.X
			ay = ty - mcP.Y
			az = tz - mcP.Z
			tagAxis.ToQuaternion(restQ)
		}
	}
	bx := -mcC.X
	by := -mcC.Y
	bz := -mcC.Z

	jCfg := v.joints.GetJoint(i)
	return collisions.NewJointConstraint(
		v.partViews[pIdx].GetSolver(), v.partViews[i].GetSolver(),
		ax, ay, az, bx, by, bz,
		restQ,
		jCfg.GetMaxAngleRagdoll(),
		jCfg.GetDamping(),
	)
}

// SetAction selects action idx and propagates mapped sub-actions to still-attached parts in execution order.
func (v *Vertices3D) SetAction(idx int) {
	if idx < 0 || v.currentAction == idx || v.joints.IsRagdoll() {
		return
	}
	v.currentAction = idx
	v.compiled = false
	enteringRagdoll := false
	eoLen := v.links.ExecutionOrderLen()
	for eoIdx := 0; eoIdx < eoLen; eoIdx++ {
		i := v.links.ExecutionOrderAt(eoIdx)
		el := v.elements.GetEntry(i)
		if el == nil || v.partViews[i].IsPhysicsFree() {
			continue
		}
		if targetAction, ok := v.actions.Get(i, idx); ok {
			el.SetAction(targetAction)
			if el.IsActionClamped(targetAction) {
				enteringRagdoll = true
			}
		} else {
			fmt.Printf("[warning] Vertices3D: part %d has no mapping for action %d, falling back to action 0\n", i, idx)
			el.SetAction(0)
		}
	}
	if enteringRagdoll {
		v.SetRagdollMode()
	}
}

// SyncAttachedParts refreshes positions and AABBs of all attached segments after owner locomotion.
func (v *Vertices3D) SyncAttachedParts() {
	if v.joints.IsRagdoll() || v.ent == nil {
		return
	}
	cx, cy, cz := v.ent.GetCenter()
	qx, qy, qz, qw := v.ent.GetQRotation().Get()
	if qx != v.compiledQX || qy != v.compiledQY || qz != v.compiledQZ || qw != v.compiledQW {
		v.compiled = false
		v.Compile(v.compiledTick)
		return
	}
	v.compiledCX, v.compiledCY, v.compiledCZ = cx, cy, cz
	numParts := len(v.partViews)
	for i := 0; i < numParts; i++ {
		pv := v.partViews[i]
		if pv.IsPhysicsFree() {
			continue
		}
		entry := v.elements.GetEntry(i)
		mcA := entry.FrameCenterA()
		rotA := v.joints.GetRotationA(i)
		rcAx, rcAy, rcAz := rotA.RotatePoint(mcA.X, mcA.Y, mcA.Z)
		origAX, origAY, origAZ := v.joints.GetOriginsA(i)
		pv.ent.MoveCenterTo(cx+origAX+rcAx, cy+origAY+rcAy, cz+origAZ+rcAz)
		pv.GetAABB()
	}
}

// Compile updates hierarchical animation frames and syncs every still-attached segment's render faces and tree AABB.
func (v *Vertices3D) Compile(tick uint64) bool {
	if v.joints.IsRagdoll() {
		return false
	}
	ent := v.ent
	cx, cy, cz := ent.GetCenter()
	qx, qy, qz, qw := ent.GetQRotation().Get()

	if v.compiled && v.compiledTick == tick && qx == v.compiledQX && qy == v.compiledQY && qz == v.compiledQZ && qw == v.compiledQW {
		if cx != v.compiledCX || cy != v.compiledCY || cz != v.compiledCZ {
			v.SyncAttachedParts()
		}
		if v.minX <= v.maxX {
			ent.GetAABB().Rebuild(cx+v.minX, cy+v.minY, cz+v.minZ, cx+v.maxX, cy+v.maxY, cz+v.maxZ)
		}
		return false
	}
	v.compiled = true
	v.compiledTick = tick
	v.compiledQX, v.compiledQY, v.compiledQZ, v.compiledQW = qx, qy, qz, qw
	v.compiledCX, v.compiledCY, v.compiledCZ = cx, cy, cz

	minX, minY, minZ := math.MaxFloat64, math.MaxFloat64, math.MaxFloat64
	maxX, maxY, maxZ := -math.MaxFloat64, -math.MaxFloat64, -math.MaxFloat64

	eoLen := v.links.ExecutionOrderLen()
	for eoIdx := 0; eoIdx < eoLen; eoIdx++ {
		i := v.links.ExecutionOrderAt(eoIdx)
		pv := v.partViews[i]
		if pv.IsPhysicsFree() {
			continue
		}

		v.elements.Bind(i, tick)

		pIdx := v.links.GetParent(i)
		if pIdx < 0 || pIdx == i {
			v.joints.Restore(i, ent.GetQRotation())
		} else {
			volA := v.elements.volsA[pIdx]
			volB := v.elements.volsB[pIdx]
			if volA == nil || volB == nil {
				v.joints.Restore(i, ent.GetQRotation())
			} else {
				tagIdx := v.links.GetTagIndex(i)
				if tagIdx < 0 {
					fmt.Printf("[warning] Vertices3D: missing tag %q on parent %d for part %d, inheriting parent origin and rotation without tag offset\n", v.links.GetTag(i), pIdx, i)
					v.joints.Reset(i, pIdx)
				} else {
					tagAxisA := volA.GetVertexTag(tagIdx)
					tagAxisB := volB.GetVertexTag(tagIdx)
					v.joints.CompileNormal(i, pIdx, tagAxisA, tagAxisB)
				}
			}
		}

		entry := v.elements.GetEntry(i)
		origAX, origAY, origAZ := v.joints.GetOriginsA(i)
		origBX, origBY, origBZ := v.joints.GetOriginsB(i)
		rotA := v.joints.GetRotationA(i)
		rotB := v.joints.GetRotationB(i)

		mcA := entry.FrameCenterA()
		mcB := entry.FrameCenterB()
		rcAx, rcAy, rcAz := rotA.RotatePoint(mcA.X, mcA.Y, mcA.Z)
		rcBx, rcBy, rcBz := rotB.RotatePoint(mcB.X, mcB.Y, mcB.Z)
		dOffX := (origBX - origAX) + (rcBx - rcAx)
		dOffY := (origBY - origAY) + (rcBy - rcAy)
		dOffZ := (origBZ - origAZ) + (rcBz - rcAz)

		entry.CompileAttached(rotA, rotB, dOffX, dOffY, dOffZ)

		relX := origAX + rcAx
		relY := origAY + rcAy
		relZ := origAZ + rcAz
		rotA.ToQuaternion(v.partScratchQ)
		pqx, pqy, pqz, pqw := v.partScratchQ.Get()
		pv.ent.MoveCenterTo(cx+relX, cy+relY, cz+relZ)
		pv.ent.SetRotationQuaternion(pqx, pqy, pqz, pqw)
		pv.GetAABB()

		if pMinX := relX + entry.minX; pMinX < minX {
			minX = pMinX
		}
		if pMinY := relY + entry.minY; pMinY < minY {
			minY = pMinY
		}
		if pMinZ := relZ + entry.minZ; pMinZ < minZ {
			minZ = pMinZ
		}
		if pMaxX := relX + entry.maxX; pMaxX > maxX {
			maxX = pMaxX
		}
		if pMaxY := relY + entry.maxY; pMaxY > maxY {
			maxY = pMaxY
		}
		if pMaxZ := relZ + entry.maxZ; pMaxZ > maxZ {
			maxZ = pMaxZ
		}
	}

	if minX <= maxX {
		v.minX, v.minY, v.minZ = minX, minY, minZ
		v.maxX, v.maxY, v.maxZ = maxX, maxY, maxZ
		ent.GetAABB().Rebuild(cx+v.minX, cy+v.minY, cz+v.minZ, cx+v.maxX, cy+v.maxY, cz+v.maxZ)
	}

	return false
}

// GetUIFaces returns empty faces on the owner because individual segments render through their own tree nodes.
func (v *Vertices3D) GetUIFaces() (*Faces, *Faces, float64, float64) {
	return v.elements.emptyFaces, v.elements.emptyFaces, 0.0, RenderModeModel3D
}

func (v *Vertices3D) createConfigJoints(cfg *config.Model3D, elements *Vertices3DElements) ([]*articulated_dynamics.Joint, error) {
	numParts := len(cfg.Parts)
	joints := make([]*articulated_dynamics.Joint, numParts)
	for i := 0; i < numParts; i++ {
		element := elements.GetEntry(i)
		if element == nil {
			return nil, fmt.Errorf("entry %d is nil", i)
		}
		j := articulated_dynamics.NewJoint2(8.0)
		joints[i] = j

		if vol := element.volumes[0]; vol != nil {
			if fc := vol.GetFaceCount(); fc > 0 {
				minX, minY, minZ := math.MaxFloat64, math.MaxFloat64, math.MaxFloat64
				maxX, maxY, maxZ := -math.MaxFloat64, -math.MaxFloat64, -math.MaxFloat64
				for fIdx := 0; fIdx < fc; fIdx++ {
					f := vol.GetFace(fIdx)
					minX, minY, minZ, maxX, maxY, maxZ = f.GetMinMax(minX, minY, minZ, maxX, maxY, maxZ)
				}
				j.SetLocalCenter(minX, minY, minZ, maxX, maxY, maxZ)
			}
		}
		if i == 0 || v.links.GetParent(i) < 0 {
			j.SetMaxAngleRagdoll(0)
			j.SetDamping(10.0)
			j.SetCanDetach(false)
		} else {
			tag := v.links.GetTag(i)
			z := cfg.Joints.Get(tag)
			j.SetMaxAngleRagdoll(z.MaxAngleRagdoll)
			j.SetDamping(z.Damping)
			j.SetDetachThreshold(z.DetachThreshold)
			j.SetCanDetach(z.CanDetach)
		}
	}
	return joints, nil
}

func (v *Vertices3D) createConfigLinks(cfg *config.Model3D) ([]*articulated_dynamics.Link, error) {
	numParts := len(cfg.Parts)
	links := make([]*articulated_dynamics.Link, numParts)
	for i := 0; i < numParts; i++ {
		parent, tag := articulated_dynamics.LinkInvalidIndex, ""
		if i < len(cfg.Links) {
			parent, tag = cfg.Links[i].Parent, cfg.Links[i].Tag
		}
		link := articulated_dynamics.NewLink(parent, tag)
		links[i] = link
		lParent := link.GetParent()
		if lParent < 0 {
			continue
		}
		element := v.elements.GetEntry(lParent)
		if element == nil {
			fmt.Printf("[warning] Vertices3D: parent %d not found for part %d, inheriting parent origin and rotation without tag offset\n", lParent, i)
			continue
		}
		lTag := link.GetTag()
		tagIdx := element.GetTagIndex(lTag)
		if tagIdx < 0 {
			fmt.Printf("[warning] Vertices3D: tag %q not found on parent %d for part %d, inheriting parent origin and rotation without tag offset\n", lTag, lParent, i)
			continue
		}
		link.SetTagIndex(tagIdx)
		for frameIdx, vol := range element.volumes {
			if vol == nil {
				fmt.Printf("[warning] Vertices3D: volume not found in frame %d of parent %d for part %d, inheriting parent origin and rotation without tag offset\n", frameIdx, lParent, i)
				continue
			}
			if !vol.HasVertexTag(tagIdx) {
				fmt.Printf("[warning] Vertices3D: tag %q (idx %d) missing in frame %d of parent %d for part %d, inheriting parent origin and rotation without tag offset\n", lTag, tagIdx, frameIdx, lParent, i)
				continue
			}
		}
	}
	return links, nil
}
