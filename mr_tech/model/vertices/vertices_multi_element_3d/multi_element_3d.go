package vertices_multi_element_3d

import (
	"fmt"
	"math"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/model/ithing"
	"github.com/markel1974/godoom/mr_tech/model/materials"
	"github.com/markel1974/godoom/mr_tech/model/vertices"
	"github.com/markel1974/godoom/mr_tech/model/volumes"
	"github.com/markel1974/godoom/mr_tech/physics/aabb"
	"github.com/markel1974/godoom/mr_tech/physics/articulated_dynamics"
	"github.com/markel1974/godoom/mr_tech/physics/collisions"
	"github.com/markel1974/godoom/mr_tech/physics/rigid_body"
)

// PartSpawnFunc creates a thing for a separated part using its vertices and collision group.
type PartSpawnFunc func(partVerts ithing.IVertices, collisionGroup uint64) ithing.IThing

// PartReleaseFunc is called when a part is released.
type PartReleaseFunc func(part ithing.IThing)

// RagdollJointAddFunc handles registration of a ragdoll joint for a child thing, including detachment rules.
type RagdollJointAddFunc func(joint *collisions.JointConstraint, childThing ithing.IThing, canDetach bool, breakThreshold float64)

// IPartedVertices supports spawning, releasing, and jointing detachable vertex parts for composite bodies.
type IPartedVertices interface {
	SetupParts(spawnPart PartSpawnFunc, onRelease PartReleaseFunc, addJoint RagdollJointAddFunc)
}

// MultiElement3D represents a 3D vertex-based physical entity with articulated parts, collision solving, and bounds tracking.
type MultiElement3D struct {
	id            string
	mass          float64
	owner         ithing.IThing
	solver        *collisions.Solver
	ent           *rigid_body.Entity
	actions       *Actions
	currentAction int
	links         *articulated_dynamics.Links
	elements      *Elements
	joints        *articulated_dynamics.Joints
	partViews     []*Part
	partThings    []ithing.IThing
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

// NewMultiElement3D creates and initializes a 3D vertices model from the provided thing configuration and materials.
// It sets up elements, links, joints, actions, part views, compiled bounds, and base offset.
func NewMultiElement3D(cfg *config.Thing, materials *materials.Materials) *MultiElement3D {
	v := &MultiElement3D{
		id:            cfg.Id,
		mass:          cfg.Mass,
		links:         articulated_dynamics.NewLinks(),
		joints:        articulated_dynamics.NewJoints(),
		elements:      NewElements(cfg, materials),
		actions:       NewActions(),
		currentAction: -1,
		partScratchQ:  aabb.NewQuaternionIdentity(),
	}
	configLinks, cErr := v.createConfigLinks(cfg.MultiElement3D)
	if cErr != nil {
		panic(cErr)
	}
	if err := v.links.Setup(configLinks); err != nil {
		panic(err)
	}
	configJoints, cErr := v.createConfigJoints(cfg.MultiElement3D, v.elements)
	if cErr != nil {
		panic(cErr)
	}
	if err := v.joints.Setup(configJoints, v.links); err != nil {
		panic(err)
	}
	if err := v.actions.Setup(cfg.MultiElement3D, v.elements); err != nil {
		panic(err)
	}

	numParts := v.elements.Len()
	v.partViews = make([]*Part, numParts)
	for i := 0; i < numParts; i++ {
		v.partViews[i] = NewPart(v, v.elements.GetEntry(i), i, 0)
	}

	v.ent = v.elements.GetSolver().GetEntity()
	v.Compile(0)
	if v.minZ <= v.maxZ {
		baseX := -(v.minX + v.maxX) * 0.5
		baseY := -(v.minY + v.maxY) * 0.5
		baseZ := -(v.minZ + v.maxZ) * 0.5
		if math.Abs(baseX) > 1e-6 || math.Abs(baseY) > 1e-6 || math.Abs(baseZ) > 1e-6 {
			v.joints.SetBaseOffset(baseX, baseY, baseZ)
			v.compiled = false
			v.Compile(0)
		}
	}

	v.elements.UpdateEntityBounds(v.minX, v.minY, v.minZ, v.maxX, v.maxY, v.maxZ)
	return v
}

// Setup initializes the vertices for thing, caches solver state, and assigns the entity collision group.
func (v *MultiElement3D) Setup(thing ithing.IThing) error {
	if err := v.elements.Setup(thing); err != nil {
		return err
	}
	v.owner = thing
	v.solver = v.elements.GetSolver()
	v.ent = v.solver.GetEntity()
	v.solver.SetCollisionGroup(v.ent.GetId())
	return nil
}

// SetupParts registers part lifecycle callbacks, spawns part things, assigns their collision group, and recompiles.
func (v *MultiElement3D) SetupParts(spawnPart PartSpawnFunc, onRelease PartReleaseFunc, addJoint RagdollJointAddFunc) {
	v.onPartRelease = onRelease
	v.onAddJoint = addJoint
	numParts := len(v.partViews)
	v.partThings = make([]ithing.IThing, numParts)
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

// GetPartLen returns the number of part things associated with the vertices.
func (v *MultiElement3D) GetPartLen() int {
	return len(v.partThings)
}

// GetPartAt returns the part thing at idx.
// idx must be within the range of configured parts.
func (v *MultiElement3D) GetPartAt(idx int) ithing.IThing {
	return v.partThings[idx]
}

// GetSolver returns the collision solver associated with the MultiElement3D instance.
func (v *MultiElement3D) GetSolver() *collisions.Solver {
	return v.solver
}

// QueryOverlaps finds element objects overlapping object and calls callback for each until it returns false.
func (v *MultiElement3D) QueryOverlaps(object aabb.IObject, callback func(object aabb.IObject) bool) {
	v.elements.QueryOverlaps(object, callback)
}

// GetAABB returns the entity's axis-aligned bounding box.
func (v *MultiElement3D) GetAABB() *aabb.AABB {
	return v.ent.GetAABB()
}

// Translate shifts the multi-element 3D AABB by the specified displacement.
func (v *MultiElement3D) Translate(dx, dy, dz float64) {
}

// HandlePartImpact applies an impact to a part, records the hit, adds knockback, and may detach stressed joints.
func (v *MultiElement3D) HandlePartImpact(partIdx int, force, dirX, dirY, dirZ float64) {
	if force <= 1e-6 {
		return
	}
	dLen := math.Sqrt(dirX*dirX + dirY*dirY + dirZ*dirZ)
	if dLen > 1e-6 {
		dirX /= dLen
		dirY /= dLen
		dirZ /= dLen
	} else {
		dirX, dirY, dirZ = 1.0, 0.0, 0.0
	}

	numParts := len(v.partViews)
	if partIdx < 0 || partIdx >= numParts {
		v.joints.RecordLastHit(dirX, dirY, dirZ, force, -1)
		return
	}

	v.joints.RecordLastHit(dirX, dirY, dirZ, force, partIdx)
	if v.joints.IsRagdoll() || v.partViews[partIdx].IsPhysicsFree() {
		return
	}

	// Deliver physical knockback at the impacted segment's world position while attached to the living owner.
	px, py, pz := v.partViews[partIdx].ent.GetCenter()
	v.ent.AddForceAtPoint(dirX*force, dirY*force, dirZ*force, px, py, pz)

	if numParts <= 1 || !v.joints.CanDetachMore() {
		return
	}

	normImpulse := math.Min(force/500000.0, 1.6)
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
				v.releaseSubtree(curr, dirX, dirY, dirZ, force)
				break
			}
		}
		curr = v.links.GetParent(curr)
		attenuation *= 0.60
	}
}

// releaseSubtree detaches partIdx and eligible descendants, releases their physics, and recreates joints within the group.
func (v *MultiElement3D) releaseSubtree(partIdx int, dirX, dirY, dirZ, fMag float64) {
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

// SetRagdollMode switches the articulated model into ragdoll physics.
// It releases attached parts, applies inherited and hit impulse motion, and creates ragdoll joint constraints.
func (v *MultiElement3D) SetRagdollMode() {
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

// buildJointConstraint creates a ragdoll joint constraint between parent and child part solvers.
// It derives anchors from frame centers and vertex tags, then applies the joint rest rotation and limits.
func (v *MultiElement3D) buildJointConstraint(pIdx, i int) *collisions.JointConstraint {
	pEntry := v.elements.GetEntry(pIdx)
	cEntry := v.elements.GetEntry(i)
	mcP := pEntry.FrameCenterA()
	mcC := cEntry.FrameCenterA()

	var ax, ay, az float64
	restQ := aabb.NewQuaternionIdentity()
	parentVol := v.elements.volsA[pIdx]
	if parentVol == nil && pEntry.GetVolumesLen() > 0 {
		parentVol = pEntry.GetVolume(0)
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

// SetAction changes the active action and applies mapped actions to attached parts.
// SetAction may enter ragdoll mode if a mapped part action is clamped.
func (v *MultiElement3D) SetAction(idx int) {
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

// SyncAttachedParts updates non-free part entities to match the parent entity transform.
// It recompiles when the parent rotation changes and skips synchronization in ragdoll mode.
func (v *MultiElement3D) SyncAttachedParts() {
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
		pv.UpdateAABB()
	}
}

// Compile updates part transforms and bounds for the given tick unless ragdoll or cached state is current.
func (v *MultiElement3D) Compile(tick uint64) bool {
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
		pv.UpdateAABB()
		entryMinX, entryMinY, entryMinZ, entryMaxX, entryMaxY, entryMaxZ := entry.GetMinMax()

		if pMinX := relX + entryMinX; pMinX < minX {
			minX = pMinX
		}
		if pMinY := relY + entryMinY; pMinY < minY {
			minY = pMinY
		}
		if pMinZ := relZ + entryMinZ; pMinZ < minZ {
			minZ = pMinZ
		}
		if pMaxX := relX + entryMaxX; pMaxX > maxX {
			maxX = pMaxX
		}
		if pMaxY := relY + entryMaxY; pMaxY > maxY {
			maxY = pMaxY
		}
		if pMaxZ := relZ + entryMaxZ; pMaxZ > maxZ {
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

// GetUIFaces returns empty UI face sets with zero depth and the 3D model render mode.
func (v *MultiElement3D) GetUIFaces() (*volumes.Faces, *volumes.Faces, float64, float64) {
	return v.elements.emptyFaces, v.elements.emptyFaces, 0.0, vertices.RenderModeModel3D
}

// Deform applies an impulse-based deformation to all part views within radius of impact.
// It returns true if any part was deformed.
func (v *MultiElement3D) Deform(impact geometry.XYZ, impulse geometry.XYZ, radius float64) bool {
	deformed := false
	for _, part := range v.partViews {
		if part != nil && part.Deform(impact, impulse, radius) {
			deformed = true
		}
	}
	return deformed
}

// DeformDent dents vertices around impact by pushing them opposite the normal by depth within radius.
// It returns true if any part was deformed.
func (v *MultiElement3D) DeformDent(impact geometry.XYZ, normal geometry.XYZ, depth, radius float64) bool {
	impulse := geometry.XYZ{
		X: -normal.X * depth,
		Y: -normal.Y * depth,
		Z: -normal.Z * depth,
	}
	return v.Deform(impact, impulse, radius)
}

// createConfigJoints builds articulated joints for each configured model part.
// It initializes local centers from element geometry and applies root or configured joint limits.
func (v *MultiElement3D) createConfigJoints(cfg *config.Model3D, elements *Elements) ([]*articulated_dynamics.Joint, error) {
	numParts := len(cfg.Parts)
	joints := make([]*articulated_dynamics.Joint, numParts)
	for i := 0; i < numParts; i++ {
		element := elements.GetEntry(i)
		if element == nil {
			return nil, fmt.Errorf("entry %d is nil", i)
		}
		j := articulated_dynamics.NewJoint2(8.0)
		joints[i] = j

		if vol := element.GetVolume(0); vol != nil {
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

// createConfigLinks builds articulated links from the model configuration.
// It resolves parent tag indices when available and warns when parent or tag data is missing.
func (v *MultiElement3D) createConfigLinks(cfg *config.Model3D) ([]*articulated_dynamics.Link, error) {
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
		for frameIdx := 0; frameIdx < element.GetVolumesLen(); frameIdx++ {
			vol := element.GetVolume(frameIdx)
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
