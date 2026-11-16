package volumes

import (
	"fmt"
	"math"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/model/materials"
	"github.com/markel1974/godoom/mr_tech/physics/aabb"
	"github.com/markel1974/godoom/mr_tech/physics/articulated_dynamics"
	"github.com/markel1974/godoom/mr_tech/physics/collisions"
	"github.com/markel1974/godoom/mr_tech/physics/rigid_body"
)

// DetachVolumeFunc is invoked when a skeletal subtree breaks off as an autonomous centered Volume.
type DetachVolumeFunc func(vol *Volume, x, y, z, qx, qy, qz, qw, vx, vy, vz, wx, wy, wz float64)

// RagdollPartSpawnFunc spawns one articulated rigid-body ragdoll part in Things and returns it.
type RagdollPartSpawnFunc func(vol *Volume, x, y, z, qx, qy, qz, qw, vx, vy, vz, wx, wy, wz float64, collisionGroup uint64) IThing

// RagdollJointAddFunc registers a bilateral PGS joint constraint with optional breakability on the child limb.
type RagdollJointAddFunc func(joint *collisions.JointConstraint, childThing IThing, canDetach bool, breakThreshold float64)

// IDetachableVertices is optionally implemented by IVertices types that spawn detached rigid Volumes.
type IDetachableVertices interface {
	SetOnDetach(fn DetachVolumeFunc)
}

// IRagdollVertices is optionally implemented by multi-part IVertices types that transition into
// multi-body PGS ragdolls backed by one Volume and Solver per limb.
type IRagdollVertices interface {
	SetOnRagdoll(spawnPart RagdollPartSpawnFunc, addJoint RagdollJointAddFunc)
}

// Vertices3D represents a composite animated 3D model with linked parts, volumes, faces, and articulated physics state.
type Vertices3D struct {
	id              string
	mass            float64
	restitution     float64
	friction        float64
	restCombine     int
	frictionCombine int
	gForce          float64
	solver          *collisions.Solver
	ent             *rigid_body.Entity
	actions         *Vertices3DActions
	currentAction   int
	links           *articulated_dynamics.Links
	elements        *Vertices3DElements
	joints          *articulated_dynamics.Joints
	onDetach        DetachVolumeFunc
	onSpawnRagdoll  RagdollPartSpawnFunc
	onAddJoint      RagdollJointAddFunc
	ragdollRootVol  *Volume
	ragdollRootRot  *articulated_dynamics.TagAxis
	minX            float64
	minY            float64
	minZ            float64
	maxX            float64
	maxY            float64
	maxZ            float64
	compiled        bool
	compiledTick    uint64
	compiledQX      float64
	compiledQY      float64
	compiledQZ      float64
	compiledQW      float64
}

// NewVertices3D creates a Vertices3D instance from a thing configuration and its materials.
// It validates model parts, links, action maps, and initializes face and transform buffers.
func NewVertices3D(cfg *config.Thing, materials *materials.Materials) *Vertices3D {
	v := &Vertices3D{
		id:              cfg.Id,
		mass:            cfg.Mass,
		restitution:     cfg.Restitution,
		friction:        cfg.Friction,
		restCombine:     cfg.RestitutionCombine,
		frictionCombine: cfg.FrictionCombine,
		gForce:          cfg.GForce,
		solver:          nil,
		links:           articulated_dynamics.NewLinks(),
		joints:          articulated_dynamics.NewJoints(),
		elements:        NewVertices3DElements(cfg, materials),
		actions:         NewVertices3DActions(),
		currentAction:   -1,
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
	if err := v.joints.Setup(v.elements.vVolume.solver, configJoints, v.links); err != nil {
		panic(err)
	}
	if err := v.actions.Setup(cfg.Model3D, v.elements); err != nil {
		panic(err)
	}

	v.Compile(0)
	if v.minZ <= v.maxZ {
		if baseZ := -(v.minZ + v.maxZ) * 0.5; math.Abs(baseZ) > 1e-6 {
			v.joints.SetBaseOffsetZ(baseZ)
			v.compiled = false
			v.Compile(0)
		}
	}

	v.elements.UpdateEntityBounds()
	return v
}

// Setup initializes model elements, links, joints, and actions, compiles bounds, and returns any setup error.
func (v *Vertices3D) Setup(thing IThing) error {
	if err := v.elements.Setup(thing); err != nil {
		return err
	}
	if err := v.elements.vVolume.Setup(thing); err != nil {
		return err
	}
	v.solver = v.elements.GetSolver()
	v.ent = v.solver.GetEntity()
	v.joints.SetOnDetach(v.handlePartDetached)
	return nil
}

// SetOnDetach registers the callback invoked when a limb detaches into a rigid Volume.
func (v *Vertices3D) SetOnDetach(fn DetachVolumeFunc) {
	v.onDetach = fn
}

// SetOnRagdoll registers callbacks used to spawn per-limb dynamic Volumes and register PGS JointConstraints.
func (v *Vertices3D) SetOnRagdoll(spawnPart RagdollPartSpawnFunc, addJoint RagdollJointAddFunc) {
	v.onSpawnRagdoll = spawnPart
	v.onAddJoint = addJoint
}

// GetSolver returns the collision solver associated with the model.
func (v *Vertices3D) GetSolver() *collisions.Solver {
	return v.solver
}

// QueryOverlaps checks for overlaps between the entry's active volume and object, invoking callback for each match.
func (v *Vertices3D) QueryOverlaps(object aabb.IObject, callback func(object aabb.IObject) bool) {
	if v.ragdollRootVol != nil {
		v.ragdollRootVol.QueryOverlaps(object, callback)
		return
	}
	v.elements.vVolume.QueryOverlaps(object, callback)
}

// GetAABB returns the bounding box of the associated entity, or nil if no entity is available.
func (v *Vertices3D) GetAABB() *aabb.AABB {
	ent := v.ent
	ab := ent.GetAABB()
	if v.minX <= v.maxX {
		cx, cy, cz := ent.GetCenter()
		ab.Rebuild(cx+v.minX, cy+v.minY, cz+v.minZ, cx+v.maxX, cy+v.maxY, cz+v.maxZ)
	}
	return ab
}

// SetRagdollMode freezes the model, decouples parts into individual centered Volumes and Solvers,
// and links parent/child pairs with bilateral PGS JointConstraints in the Coordinator.
func (v *Vertices3D) SetRagdollMode() {
	if v.joints.IsRagdoll() {
		return
	}
	// Ensure part volumes and joint world transforms are compiled before freezing.
	v.compiled = false
	v.Compile(v.compiledTick)

	v.elements.Freeze()
	v.joints.ActivateRagdoll()

	numParts := v.elements.Len()
	if numParts == 0 {
		return
	}

	ent := v.ent
	bodyCx, bodyCy, bodyCz := ent.GetCenter()
	pvx, pvy, pvz := ent.GetVelocity()
	hitDx, hitDy, hitDz, hitMag, hitPart := v.joints.LastHitImpulse()
	if hitDx*hitDx+hitDy*hitDy+hitDz*hitDz < 1e-6 {
		ax0, ay0, _ := ent.GetAxes0()
		hitDx, hitDy, hitDz = ax0, ay0, 0
	}
	hitKick := 35.0
	fallSpin := 3.5
	if hitMag > 0 {
		scale := math.Min(hitMag/500000.0, 2.0)
		hitKick = math.Max(hitKick, scale*65.0)
		fallSpin = math.Max(fallSpin, scale*4.5)
	}
	groupID := ent.GetId()

	partSolvers := make([]*collisions.Solver, numParts)
	partThings := make([]IThing, numParts)
	partSrcVols := make([]*Volume, numParts)
	partMCs := make([]geometry.XYZ, numParts)
	partQuats := make([]*aabb.Quaternion, numParts)

	for i := 0; i < numParts; i++ {
		if i > 0 && v.joints.IsPartDetached(i) {
			v.elements.ClearPart(i)
			continue
		}
		entry := v.elements.GetEntry(i)
		srcVol := v.elements.volsA[i]
		if srcVol == nil && len(entry.volumes) > 0 {
			srcVol = entry.volumes[0]
		}
		partSrcVols[i] = srcVol

		qPart := aabb.NewQuaternionIdentity()
		v.joints.GetRotationA(i).ToQuaternion(qPart)
		partQuats[i] = qPart

		restitution := math.Max(v.restitution, 0.20)
		friction := math.Max(v.friction, 0.55)
		gForce := v.gForce
		if gForce <= 0 {
			gForce = 9.8 * 50.0
		}
		partMass := v.mass
		if i > 0 {
			partMass = math.Max(v.mass*0.25, 10.0)
		}

		// Build centered rigid Volume (root allocates no extra Cage since v.solver runs root physics).
		cVol, mcx, mcy, mcz := v.buildCenteredSinglePartVolume(i, srcVol, i > 0, partMass, restitution, friction, gForce)
		partMCs[i] = geometry.XYZ{X: mcx, Y: mcy, Z: mcz}
		cVol.Rebuild()

		origX, origY, origZ := v.joints.GetOriginsA(i)
		rot := v.joints.GetRotationA(i)
		rcx, rcy, rcz := rot.RotatePoint(mcx, mcy, mcz)
		worldX := bodyCx + origX + rcx
		worldY := bodyCy + origY + rcy
		worldZ := bodyCz + origZ + rcz
		qx, qy, qz, qw := qPart.Get()

		vx, vy, vz := pvx, pvy, pvz
		var wx, wy, wz float64
		if i == 0 || i == hitPart {
			vx += hitDx * hitKick
			vy += hitDy * hitKick
			vz += hitDz*hitKick + 25.0
			wx = -hitDy * fallSpin
			wy = hitDx * fallSpin
		}

		if i == 0 {
			v.ragdollRootVol = cVol
			v.ragdollRootRot = articulated_dynamics.NewTagAxisIdentity()
			v.elements.lerpT = 0.0

			faces := cVol.GetFaces()
			minX, minY, minZ := faces.GetMin()
			w, h, d := faces.GetSize()
			cx, cy, cz := faces.GetCentroid()
			ixx, ixy, ixz, iyy, iyz, izz := faces.GetInertia()
			v.solver.Rebuild(minX, minY, minZ, w, h, d, cx, cy, cz, ixx, ixy, ixz, iyy, iyz, izz)
			v.solver.SetMaxStep(0.0)
			v.solver.SetCollisionGroup(groupID)
			ent.SetKeepUpright(false)
			ent.MoveCenterTo(worldX, worldY, worldZ)
			ent.SetRotationQuaternion(qx, qy, qz, qw)
			ent.SetV(vx, vy, vz)
			ent.SetAngularVelocity(wx, wy, wz)
			partSolvers[0] = v.solver
		} else {
			v.elements.ClearPart(i)
			if v.onSpawnRagdoll != nil {
				childThing := v.onSpawnRagdoll(cVol, worldX, worldY, worldZ, qx, qy, qz, qw, vx, vy, vz, wx, wy, wz, groupID)
				partThings[i] = childThing
				if childThing != nil {
					partSolvers[i] = childThing.GetSolver()
				}
			}
		}
	}

	if v.onAddJoint != nil {
		for i := 1; i < numParts; i++ {
			if partSolvers[i] == nil {
				continue
			}
			pIdx := v.links.GetParent(i)
			if pIdx < 0 || partSolvers[pIdx] == nil {
				continue
			}

			var ax, ay, az float64
			restQ := aabb.NewQuaternionIdentity()
			parentVol := partSrcVols[pIdx]
			tagIdx := v.links.GetTagIndex(i)
			if tagIdx >= 0 && parentVol != nil {
				if tagAxis := parentVol.GetVertexTag(tagIdx); tagAxis != nil {
					tx, ty, tz := tagAxis.GetOrigin()
					ax = tx - partMCs[pIdx].X
					ay = ty - partMCs[pIdx].Y
					az = tz - partMCs[pIdx].Z
					tagAxis.ToQuaternion(restQ)
				}
			}
			bx := -partMCs[i].X
			by := -partMCs[i].Y
			bz := -partMCs[i].Z

			jCfg := v.joints.GetJoint(i)
			jc := collisions.NewJointConstraint(
				partSolvers[pIdx], partSolvers[i],
				ax, ay, az, bx, by, bz,
				restQ,
				jCfg.GetMaxAngleRagdoll(),
				jCfg.GetDamping(),
			)
			v.onAddJoint(jc, partThings[i], jCfg.CanDetach(), jCfg.GetDetachThreshold())
		}
	}

	v.compiled = false
	v.Compile(v.compiledTick)
}

// SetAction selects action idx and propagates mapped sub-actions to parts in execution order.
func (v *Vertices3D) SetAction(idx int) {
	if idx < 0 {
		return
	}
	if v.currentAction == idx {
		return
	}
	if v.joints.IsRagdoll() {
		return
	}
	v.currentAction = idx
	v.compiled = false
	enteringRagdoll := false
	eoLen := v.links.ExecutionOrderLen()
	for eoIdx := 0; eoIdx < eoLen; eoIdx++ {
		i := v.links.ExecutionOrderAt(eoIdx)
		el := v.elements.GetEntry(i)
		if el == nil {
			continue
		}
		if i > 0 && v.joints.IsPartDetached(i) {
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

// ApplyImpact routes a localized 3D impact force (fx, fy, fz) at world point (px, py, pz)
// to the closest articulated part, exciting the 5+ joint DoFs along the kinematic chain, jolting ragdoll bodies,
// and triggering dismemberment on critical or post-mortem hits.
func (v *Vertices3D) ApplyImpact(fx, fy, fz, px, py, pz float64) {
	v.compiled = false
	v.joints.ApplyImpact(fx, fy, fz, px, py, pz)
}

// Compile updates all model parts for the tick and writes transformed frame data into render buffers.
// It resolves parent tag links, integrates 11-DoF joint articulations and dismembered limbs, and clears stale faces.
func (v *Vertices3D) Compile(tick uint64) bool {
	ent := v.elements.vVolume.GetSolver().GetEntity()
	cx, cy, cz := ent.GetCenter()
	qx, qy, qz, qw := ent.GetQRotation().Get()

	if v.ragdollRootVol != nil {
		if v.compiled && qx == v.compiledQX && qy == v.compiledQY && qz == v.compiledQZ && qw == v.compiledQW {
			if v.minX <= v.maxX {
				ent.GetAABB().Rebuild(cx+v.minX, cy+v.minY, cz+v.minZ, cx+v.maxX, cy+v.maxY, cz+v.maxZ)
			}
			return false
		}
		v.compiled = true
		v.compiledTick = tick
		v.compiledQX, v.compiledQY, v.compiledQZ, v.compiledQW = qx, qy, qz, qw

		isIdent := qx == 0.0 && qy == 0.0 && qz == 0.0 && qw == 1.0
		if !isIdent {
			v.ragdollRootRot.SetFromQuaternion(ent.GetQRotation())
		}
		minX, minY, minZ := math.MaxFloat64, math.MaxFloat64, math.MaxFloat64
		maxX, maxY, maxZ := -math.MaxFloat64, -math.MaxFloat64, -math.MaxFloat64
		srcFaces := v.ragdollRootVol.GetFaces()
		offset := v.elements.partFaceOffsets[0]
		count := srcFaces.Len()
		if limit := v.elements.partFaceCounts[0]; limit < count {
			count = limit
		}
		for j := 0; j < count; j++ {
			src := srcFaces.At(j)
			dst := v.elements.facesA.At(offset + j)
			dst.material = src.material
			dst.u = src.u
			dst.v = src.v
			if isIdent {
				dst.CopyPoints(src)
			} else {
				v.ragdollRootRot.TransformPointsWithMatrix(dst.tri, src.tri, 0, 0, 0)
			}
			minX, minY, minZ, maxX, maxY, maxZ = dst.GetMinMax(minX, minY, minZ, maxX, maxY, maxZ)
		}
		if minX > maxX {
			minX, minY, minZ = 0, 0, 0
			maxX, maxY, maxZ = 0, 0, 0
		}
		v.minX, v.minY, v.minZ = minX, minY, minZ
		v.maxX, v.maxY, v.maxZ = maxX, maxY, maxZ
		ent.GetAABB().Rebuild(cx+minX, cy+minY, cz+minZ, cx+maxX, cy+maxY, cz+maxZ)
		return false
	}

	if v.compiled && v.compiledTick == tick && qx == v.compiledQX && qy == v.compiledQY && qz == v.compiledQZ && qw == v.compiledQW {
		if v.minX <= v.maxX {
			ent.GetAABB().Rebuild(cx+v.minX, cy+v.minY, cz+v.minZ, cx+v.maxX, cy+v.maxY, cz+v.maxZ)
		}
		return false
	}
	v.compiled = true
	v.compiledTick = tick
	v.compiledQX, v.compiledQY, v.compiledQZ, v.compiledQW = qx, qy, qz, qw

	v.joints.StepJoints(tick)

	minX, minY, minZ := math.MaxFloat64, math.MaxFloat64, math.MaxFloat64
	maxX, maxY, maxZ := -math.MaxFloat64, -math.MaxFloat64, -math.MaxFloat64

	eoLen := v.links.ExecutionOrderLen()
	for eoIdx := 0; eoIdx < eoLen; eoIdx++ {
		i := v.links.ExecutionOrderAt(eoIdx)

		if i > 0 && v.joints.IsPartDetached(i) {
			v.elements.ClearPart(i)
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

		origAX, origAY, origAZ := v.joints.GetOriginsA(i)
		origBX, origBY, origBZ := v.joints.GetOriginsB(i)
		rotA := v.joints.GetRotationA(i)
		rotB := v.joints.GetRotationB(i)

		pMinX, pMinY, pMinZ, pMaxX, pMaxY, pMaxZ := v.elements.Compile(i, origAX, origAY, origAZ, origBX, origBY, origBZ, rotA, rotB)
		if pMinX < minX {
			minX = pMinX
		}
		if pMinY < minY {
			minY = pMinY
		}
		if pMinZ < minZ {
			minZ = pMinZ
		}
		if pMaxX > maxX {
			maxX = pMaxX
		}
		if pMaxY > maxY {
			maxY = pMaxY
		}
		if pMaxZ > maxZ {
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

// GetUIFaces returns the current face sets, interpolation factor, and render mode.
func (v *Vertices3D) GetUIFaces() (*Faces, *Faces, float64, float64) {
	return v.elements.facesA, v.elements.facesB, v.elements.lerpT, RenderModeModel3D
}

func (v *Vertices3D) createConfigJoints(cfg *config.Model3D, elements *Vertices3DElements) ([]*articulated_dynamics.Joint, error) {
	numParts := len(cfg.Parts)
	joints := make([]*articulated_dynamics.Joint, numParts)
	for i := 0; i < numParts; i++ {
		element := elements.GetEntry(i)
		if element == nil {
			return nil, fmt.Errorf("entry %d is nil", i)
		}
		// TODO FROM CONFIG default partRadius
		j := articulated_dynamics.NewJoint2(element, 8.0)
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
			j.SetDofScale(0, 0, 0)
			j.SetMaxAngleAlive(0)
			j.SetMaxAngleRagdoll(0)
			j.SetStiffness(80.0)
			j.SetDamping(10.0)
			j.SetCanDetach(false)
		} else {
			tag := v.links.GetTag(i)
			z := cfg.Joints.Get(tag)
			j.SetDofScale(z.DofScale.X, z.DofScale.Y, z.DofScale.Z)
			j.SetMaxAngleAlive(z.MaxAngleAlive)
			j.SetMaxAngleRagdoll(z.MaxAngleRagdoll)
			j.SetStiffness(z.Stiffness)
			j.SetDamping(z.Damping)
			j.SetDetachThreshold(z.DetachThreshold)
			j.SetCanDetach(z.CanDetach)
		}
	}
	return joints, nil
}

// createConfigLinks creates and initializes physics.Link objects for each part in the given config.Model3D structure.
// Returns the created links or an error if initialization fails.
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

// handlePartDetached packages the frozen subtree rooted at partIdx into an autonomous,
// origin-centered dynamic Volume and hands it off to v.onDetach for 6-DoF rigid-body simulation.
func (v *Vertices3D) handlePartDetached(partIdx int, origX, origY, origZ, qx, qy, qz, qw, vx, vy, vz, wx, wy, wz float64) {
	numParts := v.elements.Len()
	inSubtree := make([]bool, numParts)
	localOrigins := make([]geometry.XYZ, numParts)
	localRots := make([]*articulated_dynamics.TagAxis, numParts)
	partVols := make([]*Volume, numParts)

	eoLen := v.links.ExecutionOrderLen()
	for eoIdx := 0; eoIdx < eoLen; eoIdx++ {
		i := v.links.ExecutionOrderAt(eoIdx)
		if i == partIdx {
			inSubtree[i] = true
			localRots[i] = articulated_dynamics.NewTagAxisIdentity()
		} else if pIdx := v.links.GetParent(i); pIdx >= 0 && inSubtree[pIdx] && !v.joints.IsDetached(i) {
			inSubtree[i] = true
			localRots[i] = articulated_dynamics.NewTagAxisIdentity()
			parentVol := partVols[pIdx]
			tagIdx := v.links.GetTagIndex(i)
			if tagIdx >= 0 && parentVol != nil {
				if tagAxis := parentVol.GetVertexTag(tagIdx); tagAxis != nil {
					rx, ry, rz := localRots[pIdx].RotatePointByTag(tagAxis)
					localOrigins[i] = geometry.XYZ{
						X: localOrigins[pIdx].X + rx,
						Y: localOrigins[pIdx].Y + ry,
						Z: localOrigins[pIdx].Z + rz,
					}
					localRots[pIdx].MultiplyMatrix(tagAxis, localRots[i])
				} else {
					localOrigins[i] = localOrigins[pIdx]
					localRots[i].Copy(localRots[pIdx])
				}
			} else {
				localOrigins[i] = localOrigins[pIdx]
				localRots[i].Copy(localRots[pIdx])
			}
		} else {
			continue
		}

		v.elements.ClearPart(i)
		entry := v.elements.GetEntry(i)
		srcVol := entry.volA
		if srcVol == nil && len(entry.volumes) > 0 {
			srcVol = entry.volumes[0]
		}
		partVols[i] = srcVol
	}

	v.compiled = false
	if v.onDetach == nil {
		return
	}

	scratch := geometry.NewTriangle3D([3]geometry.XYZ{})
	minX, minY, minZ := math.MaxFloat64, math.MaxFloat64, math.MaxFloat64
	maxX, maxY, maxZ := -math.MaxFloat64, -math.MaxFloat64, -math.MaxFloat64

	for i := 0; i < numParts; i++ {
		if !inSubtree[i] || partVols[i] == nil {
			continue
		}
		ox, oy, oz := localOrigins[i].X, localOrigins[i].Y, localOrigins[i].Z
		rot := localRots[i]
		srcFaces := partVols[i].GetFaces()
		for fIdx := 0; fIdx < srcFaces.Len(); fIdx++ {
			src := srcFaces.At(fIdx)
			rot.TransformPointsWithMatrix(scratch, src.tri, ox, oy, oz)
			for p := 0; p < 3; p++ {
				pt := scratch.Points[p]
				if pt.X < minX {
					minX = pt.X
				}
				if pt.X > maxX {
					maxX = pt.X
				}
				if pt.Y < minY {
					minY = pt.Y
				}
				if pt.Y > maxY {
					maxY = pt.Y
				}
				if pt.Z < minZ {
					minZ = pt.Z
				}
				if pt.Z > maxZ {
					maxZ = pt.Z
				}
			}
		}
	}

	if minX > maxX {
		return
	}

	mcx := (minX + maxX) * 0.5
	mcy := (minY + maxY) * 0.5
	mcz := (minZ + maxZ) * 0.5

	mass := math.Max(v.mass*0.20, 5.0)
	restitution := math.Max(v.restitution, 0.30)
	friction := math.Max(v.friction, 0.45)
	gForce := v.gForce
	if gForce <= 0 {
		gForce = 9.8 * 50.0
	}

	volId := fmt.Sprintf("%s_part_%d", v.id, partIdx)
	detachedVol := NewVolumeDynamic(partIdx, volId, "thing", mass, restitution, friction, v.restCombine, v.frictionCombine, gForce)

	for i := 0; i < numParts; i++ {
		if !inSubtree[i] || partVols[i] == nil {
			continue
		}
		ox, oy, oz := localOrigins[i].X-mcx, localOrigins[i].Y-mcy, localOrigins[i].Z-mcz
		rot := localRots[i]
		srcFaces := partVols[i].GetFaces()
		for fIdx := 0; fIdx < srcFaces.Len(); fIdx++ {
			src := srcFaces.At(fIdx)
			rot.TransformPointsWithMatrix(scratch, src.tri, ox, oy, oz)
			fc := NewFace(scratch.Points, src.label, src.material)
			fc.u = src.u
			fc.v = src.v
			fc.lockUV = src.lockUV
			detachedVol.AddFace(fc)
		}
	}

	partAxis := articulated_dynamics.NewTagAxisIdentity()
	var partQ aabb.Quaternion
	partQ.Set(qx, qy, qz, qw)
	partAxis.SetFromQuaternion(&partQ)
	rcx, rcy, rcz := partAxis.RotatePoint(mcx, mcy, mcz)

	bodyCx, bodyCy, bodyCz := v.ent.GetCenter()
	worldX := bodyCx + origX + rcx
	worldY := bodyCy + origY + rcy
	worldZ := bodyCz + origZ + rcz

	// Tangential linear boost at center of mass: v_cm = v_pivot + omega x r_cm.
	vx += wy*rcz - wz*rcy
	vy += wz*rcx - wx*rcz
	vz += wx*rcy - wy*rcx

	v.onDetach(detachedVol, worldX, worldY, worldZ, qx, qy, qz, qw, vx, vy, vz, wx, wy, wz)
}

func (v *Vertices3D) buildCenteredSinglePartVolume(
	partIdx int,
	srcVol *Volume,
	dynamic bool,
	mass, restitution, friction, gForce float64,
) (*Volume, float64, float64, float64) {
	volId := fmt.Sprintf("%s_ragdoll_%d", v.id, partIdx)
	cVol := newVolume(dynamic, partIdx, volId, "thing", mass, restitution, friction, v.restCombine, v.frictionCombine, gForce)
	if srcVol == nil || srcVol.GetFaces().Len() == 0 {
		return cVol, 0, 0, 0
	}

	minX, minY, minZ := math.MaxFloat64, math.MaxFloat64, math.MaxFloat64
	maxX, maxY, maxZ := -math.MaxFloat64, -math.MaxFloat64, -math.MaxFloat64
	srcFaces := srcVol.GetFaces()
	for fIdx := 0; fIdx < srcFaces.Len(); fIdx++ {
		src := srcFaces.At(fIdx)
		minX, minY, minZ, maxX, maxY, maxZ = src.GetMinMax(minX, minY, minZ, maxX, maxY, maxZ)
	}
	if minX > maxX {
		return cVol, 0, 0, 0
	}

	mcx := (minX + maxX) * 0.5
	mcy := (minY + maxY) * 0.5
	mcz := (minZ + maxZ) * 0.5

	scratch := geometry.NewTriangle3D([3]geometry.XYZ{})
	for fIdx := 0; fIdx < srcFaces.Len(); fIdx++ {
		src := srcFaces.At(fIdx)
		for p := 0; p < 3; p++ {
			scratch.Points[p].X = src.tri.Points[p].X - mcx
			scratch.Points[p].Y = src.tri.Points[p].Y - mcy
			scratch.Points[p].Z = src.tri.Points[p].Z - mcz
		}
		fc := NewFace(scratch.Points, src.label, src.material)
		fc.u = src.u
		fc.v = src.v
		fc.lockUV = src.lockUV
		cVol.AddFace(fc)
	}
	return cVol, mcx, mcy, mcz
}
