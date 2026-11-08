package volumes

import (
	"fmt"
	"math"
	"strings"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/model/materials"
	"github.com/markel1974/godoom/mr_tech/physics"
	"github.com/markel1974/godoom/mr_tech/physics/aabb"
)

// Vertices3D represents a composite animated 3D model with linked parts, volumes, faces, and 11-DoF ragdoll/dismemberment physics state.
type Vertices3D struct {
	facesA            *Faces
	facesB            *Faces
	totalFaces        int
	partFaceOffsets   []int
	partFaceCounts    []int
	viewVolume        *Volume
	entity            *physics.Entity
	currentAction     int
	actions           [][]int
	elements          []*Vertices3DEntry
	links             []config.Model3DLink
	linkTagIndices    []int
	executionOrder    []int
	volsA             []*Volume
	volsB             []*Volume
	originsA          []geometry.XYZ
	originsB          []geometry.XYZ
	rotationsA        []*TagAxis
	rotationsB        []*TagAxis
	lerpT             float64
	joints            *PartJoints
	lastTick          uint64
	hasActiveDynamics bool
	isRagdoll         bool
	ragdollCollapse   bool
	dismemberedCount  int
	lastHitPart       int
	lastHitForce      float64
	lastHitDirX       float64
	lastHitDirY       float64
	lastHitDirZ       float64
	fallAxisX         float64
	fallAxisY         float64
	fallAngle         float64
	fallVel           float64
	rootRollAng       float64
	rootRollVel       float64
	rootBounceZ       float64
	rootBounceVz      float64
	rootDropZ         float64
	hasRootTransform  bool
	rootQuat          *physics.Quaternion
	rootFallQuat      *physics.Quaternion
	rootRollQuat      *physics.Quaternion
	rootCombinedQuat  *physics.Quaternion
}

// NewVertices3D creates a Vertices3D instance from a thing configuration and its materials.
// It validates model parts, links, action maps, and initializes face and transform buffers.
func NewVertices3D(thing IThing, cfg *config.Thing, materials *materials.Materials) *Vertices3D {
	if cfg.Model3D == nil {
		panic(fmt.Sprintf("no Model3D for thing %s", cfg.Id))
	}
	if len(cfg.Model3D.Parts) == 0 {
		panic(fmt.Sprintf("Model3D for thing %s has no parts", cfg.Id))
	}

	numParts := len(cfg.Model3D.Parts)
	elements := make([]*Vertices3DEntry, numParts)
	partFaceOffsets := make([]int, numParts)
	partFaceCounts := make([]int, numParts)
	var totalFaces int

	for i, part := range cfg.Model3D.Parts {
		if part == nil {
			continue
		}
		c := cfg.Clone()
		c.Model3DEntry = part
		c.Model3D = nil
		entry := NewVertices3DEntry(thing, c, materials)
		if i > 0 && elements[0] != nil {
			entry.salt = elements[0].salt
		}
		elements[i] = entry

		if len(entry.volumes) > 0 && entry.volumes[0] != nil {
			count := entry.volumes[0].GetFaceCount()
			partFaceOffsets[i] = totalFaces
			partFaceCounts[i] = count
			totalFaces += count
		}
	}

	// Ensure links slice matches parts count
	links := make([]config.Model3DLink, numParts)
	for i := 0; i < numParts; i++ {
		if i < len(cfg.Model3D.Links) {
			links[i] = cfg.Model3D.Links[i]
		} else {
			links[i] = config.Model3DLink{Parent: -1, Tag: ""}
		}
	}

	// Reject invalid parents and self-links
	for i := 0; i < numParts; i++ {
		if links[i].Parent == i {
			fmt.Printf("[warning] Vertices3D %s: part %d links to itself, disconnecting link\n", cfg.Id, i)
			links[i].Parent = -1
			links[i].Tag = ""
		} else if links[i].Parent >= numParts {
			fmt.Printf("[warning] Vertices3D %s: part %d links to invalid parent %d (numParts=%d), disconnecting link\n", cfg.Id, i, links[i].Parent, numParts)
			links[i].Parent = -1
			links[i].Tag = ""
		}
	}

	// Reject circular dependencies by recording node positions in the current chain
	visited := make([]bool, numParts)
	pathIndex := make([]int, numParts)
	path := make([]int, 0, numParts)
	for i := 0; i < numParts; i++ {
		if visited[i] {
			continue
		}
		for k := range pathIndex {
			pathIndex[k] = -1
		}
		path = path[:0]
		curr := i
		for curr >= 0 && curr < numParts {
			if visited[curr] {
				break
			}
			if pos := pathIndex[curr]; pos >= 0 {
				// Cycle detected starting at path[pos] (curr) and closed by path[len(path)-1] -> curr.
				// Disconnect exclusively the edge closing the cycle, preserving upstream links.
				prev := path[len(path)-1]
				fmt.Printf("[warning] Vertices3D %s: circular link detected (cycle start at part %d, closed by part %d -> %d), disconnecting link\n", cfg.Id, curr, prev, links[prev].Parent)
				links[prev].Parent = -1
				links[prev].Tag = ""
				break
			}
			pathIndex[curr] = len(path)
			path = append(path, curr)
			curr = links[curr].Parent
		}
		for _, node := range path {
			visited[node] = true
		}
	}

	// Validate and normalize ActionMaps
	rawActionMaps := cfg.Model3D.ActionMaps
	if rawActionMaps == nil || len(rawActionMaps) == 0 {
		fmt.Printf("[warning] Vertices3D %s: Model3D has no ActionMaps defined\n", cfg.Id)
	} else if len(rawActionMaps) != numParts {
		fmt.Printf("[warning] Vertices3D %s: ActionMaps count (%d) does not match parts count (%d)\n", cfg.Id, len(rawActionMaps), numParts)
	}

	expectedActions := 0
	if cfg.Model3D != nil && len(cfg.Model3D.ActionDefinitions) > 0 {
		expectedActions = len(cfg.Model3D.ActionDefinitions)
	} else if len(rawActionMaps) > 0 && len(rawActionMaps[0]) > 0 {
		expectedActions = len(rawActionMaps[0])
	} else if len(elements) > 0 && elements[0] != nil {
		expectedActions = len(elements[0].actions)
	}

	actions := make([][]int, numParts)
	for i := 0; i < numParts; i++ {
		var action []int
		if i < len(rawActionMaps) && rawActionMaps[i] != nil {
			action = make([]int, len(rawActionMaps[i]))
			copy(action, rawActionMaps[i])
			if expectedActions > 0 && len(action) < expectedActions {
				fmt.Printf("[warning] Vertices3D %s: part %d action map has %d entries, expected at least %d, falling back to action 0 for missing entries\n", cfg.Id, i, len(action), expectedActions)
				padded := make([]int, expectedActions)
				copy(padded, action)
				action = padded
			}
		} else {
			fmt.Printf("[warning] Vertices3D %s: missing action map for part %d, falling back to action 0\n", cfg.Id, i)
			action = make([]int, expectedActions)
		}

		if elements[i] != nil {
			numActions := len(elements[i].actions)
			for actionIdx, targetIdx := range action {
				if targetIdx < 0 || (numActions > 0 && targetIdx >= numActions) {
					fmt.Printf("[warning] Vertices3D %s: part %d action %d maps to invalid sub-action %d (max %d), falling back to action 0\n", cfg.Id, i, actionIdx, targetIdx, numActions-1)
					action[actionIdx] = 0
				}
			}
		}
		actions[i] = action
	}

	// Resolve link tag indices
	linkTagIndices := make([]int, numParts)
	for i, link := range links {
		linkTagIndices[i] = -1
		if link.Parent >= 0 && link.Parent < numParts && elements[link.Parent] != nil {
			tagIdx := elements[link.Parent].GetTagIndex(link.Tag)
			linkTagIndices[i] = tagIdx
			if tagIdx < 0 {
				fmt.Printf("[warning] Vertices3D %s: tag %q not found on parent %d for part %d, inheriting parent origin and rotation without tag offset\n", cfg.Id, link.Tag, link.Parent, i)
			} else {
				for frameIdx, vol := range elements[link.Parent].volumes {
					if vol != nil {
						if !vol.HasVertexTag(tagIdx) {
							fmt.Printf("[warning] Vertices3D %s: tag %q (idx %d) missing in frame %d of parent %d for part %d, inheriting parent origin and rotation without tag offset\n", cfg.Id, link.Tag, tagIdx, frameIdx, link.Parent, i)
						}
					}
				}
			}
		}
	}

	// Build topological execution order based on Links using DFS
	const (
		dfsUnvisited = 0
		dfsVisiting  = 1
		dfsVisited   = 2
	)
	executionOrder := make([]int, 0, numParts)
	dfsState := make([]int, numParts)
	var visit func(int)
	visit = func(node int) {
		if node < 0 || node >= numParts || dfsState[node] == dfsVisited {
			return
		}
		if dfsState[node] == dfsVisiting {
			fmt.Printf("[warning] Vertices3D %s: cycle detected during DFS at node %d, disconnecting link\n", cfg.Id, node)
			links[node].Parent = -1
			links[node].Tag = ""
			return
		}
		dfsState[node] = dfsVisiting
		pIdx := links[node].Parent
		if pIdx >= 0 && pIdx < numParts && pIdx != node {
			visit(pIdx)
		}
		dfsState[node] = dfsVisited
		executionOrder = append(executionOrder, node)
	}

	// Ensure all nodes are visited (including disconnected sub-trees)
	for i := 0; i < numParts; i++ {
		if dfsState[i] == dfsUnvisited {
			visit(i)
		}
	}

	v := &Vertices3D{
		elements:         elements,
		totalFaces:       totalFaces,
		partFaceOffsets:  partFaceOffsets,
		partFaceCounts:   partFaceCounts,
		currentAction:    -1,
		actions:          actions,
		facesA:           NewFaces(totalFaces, true),
		facesB:           NewFaces(totalFaces, true),
		links:            links,
		linkTagIndices:   linkTagIndices,
		executionOrder:   executionOrder,
		volsA:            make([]*Volume, numParts),
		volsB:            make([]*Volume, numParts),
		originsA:         make([]geometry.XYZ, numParts),
		originsB:         make([]geometry.XYZ, numParts),
		rotationsA:       make([]*TagAxis, numParts),
		rotationsB:       make([]*TagAxis, numParts),
		joints:           NewPartJoints(numParts),
		lastHitPart:      -1,
		fallAxisX:        1.0,
		rootQuat:         physics.NewQuaternionIdentity(),
		rootFallQuat:     physics.NewQuaternionIdentity(),
		rootRollQuat:     physics.NewQuaternionIdentity(),
		rootCombinedQuat: physics.NewQuaternionIdentity(),
	}

	for i := 0; i < numParts; i++ {
		v.rotationsA[i] = NewTagAxis(geometry.XYZ{}, _im)
		v.rotationsB[i] = NewTagAxis(geometry.XYZ{}, _im)
	}
	v.initPartJoints()

	// Initialize face buffers using partFaceOffsets to decouple buffer layout from execution order
	for i, el := range v.elements {
		if el == nil {
			panic("nil element found in vertices_3d")
		}
		if len(el.volumes) == 0 {
			panic("nil volumes found in vertices_3d")
		}

		if el.volumes[0] == nil {
			panic("nil volumes[0] found in vertices_3d")
		}
		target := el.volumes[0]
		count := target.GetFaceCount()
		offset := v.partFaceOffsets[i]
		for j := 0; j < count; j++ {
			src := target.GetFace(j)
			idx := offset + j
			v.facesA.Set(idx, &Face{material: src.material, u: src.u, v: src.v})
			v.facesB.Set(idx, &Face{material: src.material, u: src.u, v: src.v})
		}
	}

	if numParts == 1 && v.elements[0] != nil {
		v.viewVolume = v.elements[0].viewVolume
		v.entity = v.elements[0].VerticesGetEntity()
		v.VerticesCompile(0)
	} else if numParts > 1 {
		v.VerticesCompile(0)
		compVol := NewVolume(0, fmt.Sprintf("%s_md3_composite", cfg.Id), "thing", cfg.Mass, cfg.Restitution, cfg.Friction, cfg.GForce)
		for idx := 0; idx < v.facesA.Len(); idx++ {
			src := v.facesA.At(idx)
			if src == nil {
				continue
			}
			fCopy := NewFace(src.ClonePoints(), fmt.Sprintf("%s_md3_f_%d", cfg.Id, idx), src.material)
			u0, v0 := src.GetUV0()
			u1, v1 := src.GetUV1()
			u2, v2 := src.GetUV2()
			fCopy.SetUV(u0, v0, u1, v1, u2, v2)
			fCopy.LockUV(true)
			compVol.AddFace(fCopy)
		}
		compVol.Rebuild()
		compVol.SetThing(thing)
		v.viewVolume = compVol
		v.entity = compVol.GetEntity()
	}

	return v
}

// initPartJoints configures anatomical joint parameters, local bounds, and pre-allocated matrices for each part.
func (v *Vertices3D) initPartJoints() {
	for i := range v.joints.data {
		j := v.joints.data[i]
		j.quat.SetIdentity()
		j.detQuat.SetIdentity()
		j.axis.Reset()
		j.combinedA.Reset()
		j.combinedB.Reset()
		j.partRadius = 8.0

		if i < len(v.elements) && v.elements[i] != nil && len(v.elements[i].volumes) > 0 && v.elements[i].volumes[0] != nil {
			vol := v.elements[i].volumes[0]
			fc := vol.GetFaceCount()
			if fc > 0 {
				minX, minY, minZ := math.MaxFloat64, math.MaxFloat64, math.MaxFloat64
				maxX, maxY, maxZ := -math.MaxFloat64, -math.MaxFloat64, -math.MaxFloat64
				for fIdx := 0; fIdx < fc; fIdx++ {
					f := vol.GetFace(fIdx)
					minX, minY, minZ, maxX, maxY, maxZ = f.GetMinMax(minX, minY, minZ, maxX, maxY, maxZ)
				}
				j.localCenter = geometry.XYZ{X: (minX + maxX) * 0.5, Y: (minY + maxY) * 0.5, Z: (minZ + maxZ) * 0.5}
				dx, dy, dz := maxX-minX, maxY-minY, maxZ-minZ
				r := math.Max(dx, math.Max(dy, dz)) * 0.5
				j.partRadius = math.Max(4.0, math.Min(48.0, r))
			}
		}

		tagLow := ""
		if i < len(v.links) {
			tagLow = strings.ToLower(v.links[i].Tag)
		}

		switch {
		case i == 0 || v.links[i].Parent < 0:
			j.dofScaleX, j.dofScaleY, j.dofScaleZ = 0, 0, 0
			j.maxAngleAlive = 0
			j.maxAngleRagdoll = 0
			j.stiffness = 80.0
			j.damping = 10.0
			j.canDetach = false

		case strings.Contains(tagLow, "torso") || strings.Contains(tagLow, "chest") ||
			strings.Contains(tagLow, "waist") || strings.Contains(tagLow, "pelvis") ||
			strings.Contains(tagLow, "hips") || strings.Contains(tagLow, "belly") ||
			strings.Contains(tagLow, "spine") || strings.Contains(tagLow, "shoulders") ||
			strings.Contains(tagLow, "upper"):
			// Torso / Spine: 3 DoF (Pitch, Roll, Yaw), core trunk stays connected
			j.dofScaleX, j.dofScaleY, j.dofScaleZ = 1.0, 1.0, 0.85
			j.maxAngleAlive = 0.48
			j.maxAngleRagdoll = 0.85
			j.stiffness = 58.0
			j.damping = 7.5
			j.canDetach = false

		case strings.Contains(tagLow, "head") || strings.Contains(tagLow, "neck") || strings.Contains(tagLow, "jaw"):
			// Head / Neck: 2-3 DoF with decapitation threshold
			j.dofScaleX, j.dofScaleY, j.dofScaleZ = 1.0, 1.0, 0.75
			j.maxAngleAlive = 0.60
			j.maxAngleRagdoll = 1.15
			j.stiffness = 48.0
			j.damping = 6.5
			//j.canDetach = true
			j.canDetach = false
			j.detachThreshold = 1.25

		case strings.Contains(tagLow, "lloarm") || strings.Contains(tagLow, "rloarm") ||
			strings.Contains(tagLow, "lhand") || strings.Contains(tagLow, "rhand") ||
			strings.Contains(tagLow, "lloleg") || strings.Contains(tagLow, "rloleg") ||
			strings.Contains(tagLow, "lfoot") || strings.Contains(tagLow, "rfoot"):
			// Distal limbs (forearms, hands, lower legs): 1-2 DoF hinge, detachable
			j.dofScaleX, j.dofScaleY, j.dofScaleZ = 1.0, 0.90, 0.40
			j.maxAngleAlive = 0.72
			j.maxAngleRagdoll = 1.35
			j.stiffness = 40.0
			j.damping = 5.5
			//j.canDetach = true
			j.canDetach = false
			j.detachThreshold = 1.05

		case strings.Contains(tagLow, "weapon"):
			// Held weapon: disarms/flies off on hit
			j.dofScaleX, j.dofScaleY, j.dofScaleZ = 1.0, 1.0, 0.65
			j.maxAngleAlive = 0.68
			j.maxAngleRagdoll = 1.35
			j.stiffness = 42.0
			j.damping = 5.5
			//j.canDetach = true
			j.canDetach = false
			j.detachThreshold = 0.95

		default:
			// Upper arms and upper legs (luparm, ruparm, lupleg, rupleg, tail, etc.): 2-3 DoF ball joint, detachable on heavy/ragdoll hits
			j.dofScaleX, j.dofScaleY, j.dofScaleZ = 1.0, 1.0, 0.65
			j.maxAngleAlive = 0.62
			j.maxAngleRagdoll = 1.20
			j.stiffness = 46.0
			j.damping = 6.2
			//j.canDetach = true
			j.canDetach = false
			j.detachThreshold = 1.30
		}
	}
}

// VerticesGetVolume returns the composite view volume of the model, or nil if unavailable.
func (v *Vertices3D) VerticesGetVolume() *Volume {
	if v.viewVolume != nil {
		return v.viewVolume
	}
	if len(v.elements) > 0 && v.elements[0] != nil {
		return v.elements[0].viewVolume
	}
	return nil
}

// VerticesGetEntity returns the physics entity for the composite model, or the first element's entity if unset.
func (v *Vertices3D) VerticesGetEntity() *physics.Entity {
	if v.entity != nil {
		return v.entity
	}
	if len(v.elements) > 0 && v.elements[0] != nil {
		return v.elements[0].VerticesGetEntity()
	}
	return nil
}

// GetAABB returns the bounding box of the associated entity, or nil if no entity is available.
func (v *Vertices3D) GetAABB() *aabb.AABB {
	if e := v.VerticesGetEntity(); e != nil {
		return e.GetAABB()
	}
	return nil
}

// VerticesSetAction selects action idx and propagates mapped sub-actions to parts in execution order.
// Passing idx == -2 triggers procedural ragdoll death and collapse when no baked death animation exists.
func (v *Vertices3D) VerticesSetAction(idx int) {
	if idx == -2 {
		if v.currentAction == idx && v.isRagdoll {
			return
		}
		v.currentAction = idx
		for _, el := range v.elements {
			if el != nil {
				el.FreezeCurrentFrame()
			}
		}
		v.ragdollCollapse = true
		v.activateRagdoll()
		return
	}
	if idx < 0 {
		return
	}
	if v.currentAction == idx {
		return
	}
	v.currentAction = idx
	enteringRagdoll := false
	for _, i := range v.executionOrder {
		if i < 0 || i >= len(v.elements) {
			continue
		}
		el := v.elements[i]
		if el == nil {
			continue
		}
		if i > 0 && v.isPartOrAncestorDetached(i) {
			continue
		}
		if i < len(v.actions) && idx < len(v.actions[i]) {
			targetAction := v.actions[i][idx]
			if targetAction >= 0 && targetAction < len(el.actions) {
				el.VerticesSetAction(targetAction)
				if el.IsActionClamped(targetAction) {
					enteringRagdoll = true
				}
			} else {
				fmt.Printf("[warning] Vertices3D: part %d action %d maps to invalid sub-action %d (max %d), falling back to action 0\n",
					i, idx, targetAction, len(el.actions)-1)
				el.VerticesSetAction(0)
			}
		} else {
			fmt.Printf("[warning] Vertices3D: part %d has no mapping for action %d, falling back to action 0\n", i, idx)
			el.VerticesSetAction(0)
		}
	}
	if enteringRagdoll && !v.isRagdoll {
		v.activateRagdoll()
	}
}

// isPartOrAncestorDetached reports whether part i or any ancestor along its parent chain is detached.
func (v *Vertices3D) isPartOrAncestorDetached(i int) bool {
	curr := i
	for steps := 0; steps < len(v.joints.data) && curr > 0 && curr < len(v.joints.data); steps++ {
		if v.joints.data[curr].detached {
			return true
		}
		if curr < len(v.links) {
			curr = v.links[curr].Parent
		} else {
			break
		}
	}
	return false
}

// activateRagdoll transitions all connected joints into relaxed ragdoll physics, initiates root collapse if needed,
// and dismembers any critically hit limb on death.
func (v *Vertices3D) activateRagdoll() {
	v.isRagdoll = true
	v.hasActiveDynamics = true

	dx, dy, dz := v.lastHitDirX, v.lastHitDirY, v.lastHitDirZ
	if dx*dx+dy*dy+dz*dz < 1e-6 {
		dx = 1.0
	}

	horizLen := math.Hypot(dx, dy)
	if horizLen > 1e-4 {
		v.fallAxisX = -dy / horizLen
		v.fallAxisY = dx / horizLen
	} else {
		v.fallAxisX = 1.0
		v.fallAxisY = 0.0
	}

	if v.ragdollCollapse && v.fallAngle == 0 {
		v.fallAngle = 0.08
		v.fallVel = 3.8
	}
	v.rootRollVel += 2.6
	v.rootBounceVz = 48.0

	// If the killing blow hit a detachable extremity with sufficient accumulated stress, sever it on death
	if v.dismemberedCount < 3 && v.lastHitPart > 0 && v.lastHitPart < len(v.joints.data) {
		hj := v.joints.data[v.lastHitPart]
		if hj.canDetach && !hj.detached && hj.damage >= hj.detachThreshold*0.45 {
			v.detachPart(v.lastHitPart, dx, dy, dz, v.lastHitForce)
		}
	}

	for i := 1; i < len(v.joints.data); i++ {
		j := v.joints.data[i]
		if j.detached {
			continue
		}
		// Compute gravitational/impact slump target in the joint's local frame
		tx := dy*0.55 - dz*0.25
		ty := -dx*0.55 + dz*0.40
		tz := (dx - dy) * 0.30
		lx, ly, lz := v.rotationsA[i].RotatePointInverse(tx, ty, tz)
		sign := 1.0
		if i&1 == 1 {
			sign = -0.90
		}
		j.ragdollTargetX = math.Max(-j.maxAngleRagdoll*0.85, math.Min(j.maxAngleRagdoll*0.85, lx*j.dofScaleX*0.95))
		j.ragdollTargetY = math.Max(-j.maxAngleRagdoll*0.85, math.Min(j.maxAngleRagdoll*0.85, (ly+0.28*sign)*j.dofScaleY*0.95))
		j.ragdollTargetZ = math.Max(-j.maxAngleRagdoll*0.70, math.Min(j.maxAngleRagdoll*0.70, lz*j.dofScaleZ*0.80))

		j.velX += j.ragdollTargetX * 5.0
		j.velY += j.ragdollTargetY * 5.0
		j.velZ += j.ragdollTargetZ * 4.0
	}
}

// VerticesApplyImpact routes a localized 3D impact force (fx, fy, fz) at world point (px, py, pz)
// to the closest articulated part, exciting the 5+ joint DoFs along the kinematic chain, jolting ragdoll bodies,
// and triggering dismemberment on critical or post-mortem hits.
func (v *Vertices3D) VerticesApplyImpact(fx, fy, fz, px, py, pz float64) {
	if len(v.elements) == 0 {
		return
	}
	fMag := math.Sqrt(fx*fx + fy*fy + fz*fz)
	if fMag < 1e-6 {
		return
	}
	dirX, dirY, dirZ := fx/fMag, fy/fMag, fz/fMag

	v.lastHitForce = fMag
	v.lastHitDirX, v.lastHitDirY, v.lastHitDirZ = dirX, dirY, dirZ
	v.hasActiveDynamics = true

	// Normalize force magnitude to angular joint impulse (calibrated to weaponForce = 500,000)
	normImpulse := math.Min(fMag/500000.0, 1.6)
	if normImpulse < 0.45 {
		normImpulse = 0.45
	}

	// Always jolt root corpse when already in ragdoll state (even for single-mesh or torso hits)
	if v.isRagdoll {
		if v.ragdollCollapse && v.fallAngle > 1.15 {
			// Make lying corpse pop/shudder on impact
			v.fallVel -= 2.4 * normImpulse
		} else {
			v.fallVel += 2.8 * normImpulse
		}
		crossZ := dirX*v.fallAxisY - dirY*v.fallAxisX
		if math.Abs(crossZ) < 0.25 {
			crossZ = 0.55
		}
		v.rootRollVel += crossZ * 5.5 * normImpulse
		v.rootBounceVz = math.Max(v.rootBounceVz, 60.0*normImpulse)
	}

	if len(v.elements) <= 1 {
		return
	}

	cx, cy, cz := v.VerticesGetCenter()
	relPx, relPy, relPz := px-cx, py-cy, pz-cz

	// Locate the closest non-detached articulated part to the impact ray/point
	bestPart := -1
	bestScore := math.MaxFloat64
	bestDetachPart := -1
	bestDetachScore := math.MaxFloat64

	for i := 0; i < len(v.elements); i++ {
		if i > 0 && v.isPartOrAncestorDetached(i) {
			continue
		}
		j := v.joints.data[i]
		lcX, lcY, lcZ := v.rotationsA[i].RotatePoint(j.localCenter.X, j.localCenter.Y, j.localCenter.Z)
		partCx := v.originsA[i].X + lcX
		partCy := v.originsA[i].Y + lcY
		partCz := v.originsA[i].Z + lcZ

		vx := partCx - relPx
		vy := partCy - relPy
		vz := partCz - relPz

		// Decompose vector from ray entry point to part center into longitudinal (along ray) and perpendicular components
		along := vx*dirX + vy*dirY + vz*dirZ
		perpX := vx - along*dirX
		perpY := vy - along*dirY
		perpZ := vz - along*dirZ
		perpDist := math.Sqrt(perpX*perpX + perpY*perpY + perpZ*perpZ)
		rad := math.Max(j.partRadius, 6.0)

		score := (perpDist / rad) + math.Abs(along)/(rad*3.5)
		// Favor articulated child limbs over root part 0 when the ray passes near the child limb
		if i > 0 && perpDist <= rad*1.8 {
			score *= 0.60
		}
		if score < bestScore {
			bestScore = score
			bestPart = i
		}
		if i > 0 && j.canDetach && !j.detached && score < bestDetachScore {
			bestDetachScore = score
			bestDetachPart = i
		}
	}

	if bestPart < 0 {
		return
	}

	// If root part 0 was matched, also excite its primary torso child if present
	targetPart := bestPart
	if targetPart == 0 {
		for i := 1; i < len(v.links); i++ {
			if v.links[i].Parent == 0 && !v.joints.data[i].detached {
				targetPart = i
				break
			}
		}
	}

	v.lastHitPart = targetPart

	// Propagate joint torque up the kinematic chain (hit limb -> parent -> grandparent)
	curr := targetPart
	atten := 1.0
	for depth := 0; depth < 3 && curr > 0 && curr < len(v.joints.data); depth++ {
		j := v.joints.data[curr]
		if j.detached {
			break
		}
		pivX, pivY, pivZ := v.originsA[curr].X, v.originsA[curr].Y, v.originsA[curr].Z
		lcX, lcY, lcZ := v.rotationsA[curr].RotatePoint(j.localCenter.X, j.localCenter.Y, j.localCenter.Z)

		// Blend impact lever arm with limb center-of-mass lever arm so direct hits always bend the joint backward
		rx := 0.45*(relPx-pivX) + 0.55*lcX
		ry := 0.45*(relPy-pivY) + 0.55*lcY
		rz := 0.45*(relPz-pivZ) + 0.55*lcZ
		rLen := math.Sqrt(rx*rx + ry*ry + rz*rz)
		if rLen < 1e-4 {
			rx, ry, rz = 0, 0, j.partRadius
			rLen = j.partRadius
		}

		// Unit torque direction τ = (r / ||r||) × dir
		tx := (ry*dirZ - rz*dirY) / rLen
		ty := (rz*dirX - rx*dirZ) / rLen
		tz := (rx*dirY - ry*dirX) / rLen
		if tx*tx+ty*ty+tz*tz < 0.04 {
			// Fallback orthogonal bend axis when shot is purely radial along the bone
			tx, ty, tz = -dirY, dirX, 0.30
		}

		// Transform torque into the joint's local coordinate system
		ltx, lty, ltz := v.rotationsA[curr].RotatePointInverse(tx, ty, tz)
		kick := 9.5 * normImpulse * atten
		if v.isRagdoll {
			kick *= 1.45
		}

		const maxJointVel = 14.0
		j.velX = math.Max(-maxJointVel, math.Min(maxJointVel, j.velX+ltx*kick*j.dofScaleX))
		j.velY = math.Max(-maxJointVel, math.Min(maxJointVel, j.velY+lty*kick*j.dofScaleY))
		j.velZ = math.Max(-maxJointVel, math.Min(maxJointVel, j.velZ+ltz*kick*j.dofScaleZ))

		if j.canDetach && v.dismemberedCount < 3 {
			stress := 0.75 * normImpulse * atten
			if v.isRagdoll {
				stress *= 1.65
			}
			j.damage += stress
			if j.damage >= j.detachThreshold {
				v.detachPart(curr, dirX, dirY, dirZ, fMag)
				break
			}
		}

		curr = v.links[curr].Parent
		atten *= 0.60
	}

	// When shooting a ragdoll corpse on the torso, also whip all connected limbs and stress the nearest detachable limb
	if v.isRagdoll {
		for i := 1; i < len(v.joints.data); i++ {
			j := v.joints.data[i]
			if i == targetPart || j.detached {
				continue
			}
			sign := 1.0
			if i&1 == 1 {
				sign = -1.0
			}
			j.velX += (dirY * 3.2 * normImpulse * sign) * j.dofScaleX
			j.velY += (-dirX * 3.2 * normImpulse) * j.dofScaleY
			j.velZ += (2.2 * normImpulse * sign) * j.dofScaleZ
		}
		if bestDetachPart > 0 && bestDetachPart != targetPart && bestDetachScore < 3.2 && v.dismemberedCount < 3 {
			dj := v.joints.data[bestDetachPart]
			dj.damage += 0.85 * normImpulse
			if dj.damage >= dj.detachThreshold {
				v.detachPart(bestDetachPart, dirX, dirY, dirZ, fMag)
			}
		}
	}
}

// detachPart severs part idx from its parent joint and launches it on an independent 6-DoF ballistic & tumbling trajectory.
func (v *Vertices3D) detachPart(idx int, dirX, dirY, dirZ, fMag float64) {
	if idx <= 0 || idx >= len(v.joints.data) || v.dismemberedCount >= 3 {
		return
	}
	j := v.joints.data[idx]
	if j.detached || !j.canDetach {
		return
	}
	j.detached = true
	j.detSettled = false
	v.dismemberedCount++
	v.hasActiveDynamics = true

	for i := 0; i < len(v.elements); i++ {
		if i == idx || v.isPartOrAncestorDetached(i) {
			if v.elements[i] != nil {
				v.elements[i].FreezeCurrentFrame()
			}
		}
	}

	cx, cy, cz := v.VerticesGetCenter()
	j.detWorldX = cx + v.originsA[idx].X
	j.detWorldY = cy + v.originsA[idx].Y
	j.detWorldZ = cz + v.originsA[idx].Z

	halfD := 28.0
	if v.entity != nil {
		_, _, halfD = v.entity.GetSizeCenter()
	}
	j.detFloorZ = (cz - halfD) + math.Min(j.partRadius*0.30, 5.0)
	if j.detWorldZ < j.detFloorZ {
		j.detWorldZ = j.detFloorZ
	}

	v.rotationsA[idx].ToQuaternion(j.detQuat)

	// Ballistic launch velocity along impact ray + upward pop + slight lateral scatter
	launchSpeed := 125.0 + math.Min(fMag/500000.0, 2.5)*70.0
	sideSign := 1.0
	if idx&1 == 1 {
		sideSign = -1.0
	}
	j.detVx = dirX*launchSpeed - dirY*(25.0*sideSign)
	j.detVy = dirY*launchSpeed + dirX*(25.0*sideSign)
	j.detVz = math.Max(dirZ*launchSpeed+105.0, 75.0)

	// 3D tumbling angular velocity
	j.detWx = -dirY*8.5 + sideSign*3.5
	j.detWy = dirX*8.5 - 3.0
	j.detWz = sideSign * 7.0
}

// stepJoints integrates root ragdoll collapse/jolts, all 11-DoF articulated joint spring-dampers,
// and severed limb ballistic trajectories over timestep dt.
func (v *Vertices3D) stepJoints(tick uint64) {
	if tick == 0 {
		return
	}
	if v.lastTick == 0 {
		v.lastTick = tick
		return
	}
	if tick <= v.lastTick {
		return
	}
	dt := float64(tick-v.lastTick) / 120.0
	v.lastTick = tick
	if !v.hasActiveDynamics && !v.isRagdoll {
		return
	}
	if dt > 1.0/30.0 {
		dt = 1.0 / 30.0
	}

	anyActive := false

	// 1. Root ragdoll collapse & post-mortem impact jolt integration
	if v.isRagdoll {
		rootActive := false
		if v.ragdollCollapse {
			const maxFallAngle = 1.48 // ~85 degrees onto floor
			if v.fallAngle < maxFallAngle || math.Abs(v.fallVel) > 0.02 {
				rootActive = true
				grav := 18.0 * math.Sin(math.Max(v.fallAngle, 0.15))
				v.fallVel += grav * dt
				v.fallVel *= math.Pow(0.45, dt)
				v.fallAngle += v.fallVel * dt
				if v.fallAngle >= maxFallAngle {
					v.fallAngle = maxFallAngle
					if v.fallVel > 0.45 {
						v.fallVel = -v.fallVel * 0.25
					} else {
						v.fallVel = 0.0
					}
				} else if v.fallAngle < 0.0 {
					v.fallAngle = 0.0
					v.fallVel = 0.0
				}
			}
		}

		// Secondary roll/jolt oscillation when corpse is shot
		if math.Abs(v.rootRollAng) > 1e-4 || math.Abs(v.rootRollVel) > 1e-4 {
			rootActive = true
			alphaRoll := -32.0*v.rootRollAng - 5.5*v.rootRollVel
			v.rootRollVel += alphaRoll * dt
			v.rootRollAng += v.rootRollVel * dt
			if math.Abs(v.rootRollAng) > 0.55 {
				v.rootRollAng = math.Copysign(0.55, v.rootRollAng)
				v.rootRollVel *= -0.35
			}
			if math.Abs(v.rootRollAng) < 1e-3 && math.Abs(v.rootRollVel) < 1e-3 {
				v.rootRollAng = 0
				v.rootRollVel = 0
			}
		}

		// Vertical pop/bounce when corpse is hit
		if v.rootBounceZ > 0 || math.Abs(v.rootBounceVz) > 1e-3 {
			rootActive = true
			v.rootBounceVz -= 480.0 * dt
			v.rootBounceZ += v.rootBounceVz * dt
			if v.rootBounceZ <= 0 {
				v.rootBounceZ = 0
				if v.rootBounceVz < -22.0 {
					v.rootBounceVz = -v.rootBounceVz * 0.28
				} else {
					v.rootBounceVz = 0
				}
			}
		}

		halfD := 24.0
		if v.entity != nil {
			_, _, halfD = v.entity.GetSizeCenter()
		}
		collapseDrop := 0.0
		if v.ragdollCollapse {
			collapseDrop = -math.Max(halfD-6.0, 0.0) * math.Sin(v.fallAngle)
		}
		v.rootDropZ = collapseDrop + v.rootBounceZ

		if v.fallAngle > 1e-4 || math.Abs(v.rootRollAng) > 1e-4 {
			v.rootFallQuat.SetAxisAngle(v.fallAxisX, v.fallAxisY, 0, v.fallAngle)
			if math.Abs(v.rootRollAng) > 1e-4 {
				v.rootRollQuat.SetAxisAngle(-v.fallAxisY, v.fallAxisX, 0.35, v.rootRollAng)
				v.rootRollQuat.Multiplies(v.rootFallQuat, v.rootQuat)
			} else {
				v.rootQuat.CopyFrom(v.rootFallQuat)
			}
			v.hasRootTransform = true
		}

		if rootActive {
			anyActive = true
		}
	}

	// 2. Detached limbs and connected 5+ DoF joint spring-dampers
	for i := 1; i < len(v.joints.data); i++ {
		j := v.joints.data[i]
		if j.detached {
			if !j.detSettled {
				anyActive = true
				const limbGravity = 560.0
				j.detVz -= limbGravity * dt

				airDamp := math.Pow(0.65, dt)
				j.detVx *= airDamp
				j.detVy *= airDamp

				j.detWorldX += j.detVx * dt
				j.detWorldY += j.detVy * dt
				j.detWorldZ += j.detVz * dt

				if j.detWorldZ <= j.detFloorZ {
					j.detWorldZ = j.detFloorZ
					if j.detVz < -28.0 {
						j.detVz = -j.detVz * 0.34
						j.detWx += -j.detVy * 0.04
						j.detWy += j.detVx * 0.04
					} else {
						j.detVz = 0.0
					}
					groundFric := math.Pow(0.015, dt)
					j.detVx *= groundFric
					j.detVy *= groundFric

					angFric := math.Pow(0.025, dt)
					j.detWx *= angFric
					j.detWy *= angFric
					j.detWz *= angFric

					if j.detVz == 0.0 && (j.detVx*j.detVx+j.detVy*j.detVy) < 1.0 &&
						(j.detWx*j.detWx+j.detWy*j.detWy+j.detWz*j.detWz) < 0.02 {
						j.detVx, j.detVy, j.detVz = 0, 0, 0
						j.detWx, j.detWy, j.detWz = 0, 0, 0
						j.detSettled = true
					}
				}
				j.detQuat.IntegrateAngular(j.detWx, j.detWy, j.detWz, dt)
			}
			continue
		}

		// Connected articulated joint (PD spring-damper + cone joint limits)
		ks := j.stiffness
		kd := j.damping
		maxAng := j.maxAngleAlive
		var targetX, targetY, targetZ float64

		if v.isRagdoll {
			ks *= 0.12
			kd *= 0.55
			maxAng = j.maxAngleRagdoll
			targetX, targetY, targetZ = j.ragdollTargetX, j.ragdollTargetY, j.ragdollTargetZ
		} else if j.damage > 0 {
			j.damage = math.Max(0.0, j.damage-0.10*dt)
		}

		alphaX := -ks*(j.angX-targetX) - kd*j.velX
		alphaY := -ks*(j.angY-targetY) - kd*j.velY
		alphaZ := -ks*(j.angZ-targetZ) - kd*j.velZ

		j.velX += alphaX * dt
		j.velY += alphaY * dt
		j.velZ += alphaZ * dt

		j.angX += j.velX * dt
		j.angY += j.velY * dt
		j.angZ += j.velZ * dt

		angMagSq := j.angX*j.angX + j.angY*j.angY + j.angZ*j.angZ
		if maxAng > 0.0 && angMagSq > maxAng*maxAng {
			angMag := math.Sqrt(angMagSq)
			nx, ny, nz := j.angX/angMag, j.angY/angMag, j.angZ/angMag
			j.angX, j.angY, j.angZ = nx*maxAng, ny*maxAng, nz*maxAng
			angMagSq = maxAng * maxAng
			vDotN := j.velX*nx + j.velY*ny + j.velZ*nz
			if vDotN > 0.0 {
				j.velX -= 1.30 * vDotN * nx
				j.velY -= 1.30 * vDotN * ny
				j.velZ -= 1.30 * vDotN * nz
			}
		}

		diffX := j.angX - targetX
		diffY := j.angY - targetY
		diffZ := j.angZ - targetZ
		diffSq := diffX*diffX + diffY*diffY + diffZ*diffZ
		velMagSq := j.velX*j.velX + j.velY*j.velY + j.velZ*j.velZ

		if diffSq > 1e-6 || velMagSq > 1e-5 {
			anyActive = true
		}

		if angMagSq > 1e-6 {
			angMag := math.Sqrt(angMagSq)
			j.quat.SetAxisAngle(j.angX/angMag, j.angY/angMag, j.angZ/angMag, angMag)
			j.axis.SetFromQuaternion(j.quat)
			j.hasDeflection = true
		} else {
			j.axis.Reset()
			j.hasDeflection = false
		}

		if diffSq <= 1e-6 && velMagSq <= 1e-5 {
			j.angX, j.angY, j.angZ = targetX, targetY, targetZ
			j.velX, j.velY, j.velZ = 0, 0, 0
		}
	}
	v.hasActiveDynamics = anyActive
}

// VerticesGetCenter returns the center coordinates of the composite entity, or zero coordinates if none is available.
func (v *Vertices3D) VerticesGetCenter() (float64, float64, float64) {
	if v.entity != nil {
		return v.entity.GetCenter()
	}
	if len(v.elements) > 0 && v.elements[0] != nil {
		return v.elements[0].VerticesGetCenter()
	}
	return 0, 0, 0
}

// VerticesCompile updates all model parts for the tick and writes transformed frame data into render buffers.
// It resolves parent tag links, integrates 11-DoF joint articulations and dismembered limbs, and clears stale faces.
func (v *Vertices3D) VerticesCompile(tick uint64) bool {
	v.stepJoints(tick)
	cx, cy, cz := v.VerticesGetCenter()

	for _, i := range v.executionOrder {
		element := v.elements[i]
		_ = element.VerticesCompile(tick)
		volA, volB, lerpT := element.GetVolumes()
		if i == 0 {
			v.lerpT = lerpT
		}

		v.volsA[i] = volA
		v.volsB[i] = volB

		pIdx := -1
		if i < len(v.links) {
			pIdx = v.links[i].Parent
		}

		if i > 0 && v.joints.data[i].detached {
			j := v.joints.data[i]
			v.originsA[i].X = j.detWorldX - cx
			v.originsA[i].Y = j.detWorldY - cy
			v.originsA[i].Z = j.detWorldZ - cz
			v.originsB[i] = v.originsA[i]
			v.rotationsA[i].SetFromQuaternion(j.detQuat)
			v.rotationsB[i].SetFromQuaternion(j.detQuat)
		} else if pIdx >= 0 && pIdx < len(v.elements) && pIdx != i && v.volsA[pIdx] != nil && v.volsB[pIdx] != nil {
			tagIdx := -1
			if i < len(v.linkTagIndices) {
				tagIdx = v.linkTagIndices[i]
			}
			j := v.joints.data[i]
			if tagIdx >= 0 {
				if tagAxisA := v.volsA[pIdx].GetVertexTag(tagIdx); tagAxisA != nil {
					rotOrigAX, rotOrigAY, rotOrigAZ := v.rotationsA[pIdx].RotatePointByTag(tagAxisA)
					v.originsA[i].X = v.originsA[pIdx].X + rotOrigAX
					v.originsA[i].Y = v.originsA[pIdx].Y + rotOrigAY
					v.originsA[i].Z = v.originsA[pIdx].Z + rotOrigAZ
					if j.hasDeflection {
						v.rotationsA[pIdx].MultiplyMatrix(tagAxisA, j.combinedA)
						j.combinedA.MultiplyMatrix(j.axis, v.rotationsA[i])
					} else {
						v.rotationsA[pIdx].MultiplyMatrix(tagAxisA, v.rotationsA[i])
					}
				} else {
					fmt.Printf("[warning] Vertices3D: tag %q (idx %d) not found in frame A of parent %d for part %d, inheriting parent origin and rotation without tag offset\n", v.links[i].Tag, tagIdx, pIdx, i)
					v.originsA[i] = v.originsA[pIdx]
					if j.hasDeflection {
						v.rotationsA[pIdx].MultiplyMatrix(j.axis, v.rotationsA[i])
					} else {
						*v.rotationsA[i] = *v.rotationsA[pIdx]
					}
				}
				if tagAxisB := v.volsB[pIdx].GetVertexTag(tagIdx); tagAxisB != nil {
					rotOrigBX, rotOrigBY, rotOrigBZ := v.rotationsB[pIdx].RotatePointByTag(tagAxisB)
					v.originsB[i].X = v.originsB[pIdx].X + rotOrigBX
					v.originsB[i].Y = v.originsB[pIdx].Y + rotOrigBY
					v.originsB[i].Z = v.originsB[pIdx].Z + rotOrigBZ
					if j.hasDeflection {
						v.rotationsB[pIdx].MultiplyMatrix(tagAxisB, j.combinedB)
						j.combinedB.MultiplyMatrix(j.axis, v.rotationsB[i])
					} else {
						v.rotationsB[pIdx].MultiplyMatrix(tagAxisB, v.rotationsB[i])
					}
				} else {
					fmt.Printf("[warning] Vertices3D: tag %q (idx %d) not found in frame B of parent %d for part %d, inheriting parent origin and rotation without tag offset\n", v.links[i].Tag, tagIdx, pIdx, i)
					v.originsB[i] = v.originsB[pIdx]
					if j.hasDeflection {
						v.rotationsB[pIdx].MultiplyMatrix(j.axis, v.rotationsB[i])
					} else {
						*v.rotationsB[i] = *v.rotationsB[pIdx]
					}
				}
			} else {
				fmt.Printf("[warning] Vertices3D: missing tag %q on parent %d for part %d, inheriting parent origin and rotation without tag offset\n", v.links[i].Tag, pIdx, i)
				v.originsA[i] = v.originsA[pIdx]
				v.originsB[i] = v.originsB[pIdx]
				if j.hasDeflection {
					v.rotationsA[pIdx].MultiplyMatrix(j.axis, v.rotationsA[i])
					v.rotationsB[pIdx].MultiplyMatrix(j.axis, v.rotationsB[i])
				} else {
					*v.rotationsA[i] = *v.rotationsA[pIdx]
					*v.rotationsB[i] = *v.rotationsB[pIdx]
				}
			}
		} else {
			v.reset(i)
		}

		offset := v.partFaceOffsets[i]
		partCount := v.partFaceCounts[i]

		countA := volA.GetFaceCount()
		countB := volB.GetFaceCount()

		count := countA
		if countB < count {
			count = countB
		}
		if partCount < count {
			count = partCount
		}

		origAX, origAY, origAZ := v.originsA[i].X, v.originsA[i].Y, v.originsA[i].Z
		origBX, origBY, origBZ := v.originsB[i].X, v.originsB[i].Y, v.originsB[i].Z
		rotA := v.rotationsA[i]
		rotB := v.rotationsB[i]

		isIdentA := rotA.IsIdentity()
		isIdentB := rotB.IsIdentity()

		for j := 0; j < count; j++ {
			srcA := volA.GetFace(j)
			dstA := v.facesA.At(offset + j)
			if isIdentA {
				if origAX == 0 && origAY == 0 && origAZ == 0 {
					dstA.CopyPoints(srcA)
				} else {
					dstA.TransformPoints(srcA, origAX, origAY, origAZ)
				}
			} else {
				rotA.TransformPointsWithMatrix(dstA, srcA, origAX, origAY, origAZ)
			}

			srcB := volB.GetFace(j)
			dstB := v.facesB.At(offset + j)
			if isIdentB {
				if origBX == 0 && origBY == 0 && origBZ == 0 {
					dstB.CopyPoints(srcB)
				} else {
					dstB.TransformPoints(srcB, origBX, origBY, origBZ)
				}
			} else {
				rotB.TransformPointsWithMatrix(dstB, srcB, origBX, origBY, origBZ)
			}
		}

		// Zero out any leftover faces in this part's buffer slot to avoid displaying stale frame data
		for j := count; j < partCount; j++ {
			if faceA := v.facesA.At(offset + j); faceA != nil {
				faceA.Zero()
			}
			if faceB := v.facesB.At(offset + j); faceB != nil {
				faceB.Zero()
			}
		}
	}
	return false
}

// VerticesGet returns the current face sets, interpolation factor, and render mode.
func (v *Vertices3D) VerticesGet() (*Faces, *Faces, float64, float64) {
	return v.facesA, v.facesB, v.lerpT, RenderModeModel3D
}

// reset clears the origin offsets and restores the rotation axes at index i to the entity's orientation (or identity if unset),
// composing any active procedural root ragdoll collapse/jolt transform.
func (v *Vertices3D) reset(i int) {
	v.originsA[i].X, v.originsA[i].Y, v.originsA[i].Z = 0, 0, v.rootDropZ
	v.originsB[i].X, v.originsB[i].Y, v.originsB[i].Z = 0, 0, v.rootDropZ
	if v.entity != nil {
		q := v.entity.GetQuaternion()
		if v.hasRootTransform {
			v.rootQuat.Multiplies(q, v.rootCombinedQuat)
			v.rotationsA[i].SetFromQuaternion(v.rootCombinedQuat)
			v.rotationsB[i].SetFromQuaternion(v.rootCombinedQuat)
		} else {
			v.rotationsA[i].SetFromQuaternion(q)
			v.rotationsB[i].SetFromQuaternion(q)
		}
	} else if v.hasRootTransform {
		v.rotationsA[i].SetFromQuaternion(v.rootQuat)
		v.rotationsB[i].SetFromQuaternion(v.rootQuat)
	} else {
		v.rotationsA[i].Reset()
		v.rotationsB[i].Reset()
	}
}

// deepReset clears cached volumes and resets origin and rotation state for the part at i.
func (v *Vertices3D) deepReset(i int) {
	v.volsA[i] = nil
	v.volsB[i] = nil
	v.reset(i)
}
