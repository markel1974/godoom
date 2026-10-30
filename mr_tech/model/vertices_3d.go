package model

import (
	"fmt"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/physics"
	"github.com/markel1974/godoom/mr_tech/physics/aabb"
)

var _identityMatrix = [3]geometry.XYZ{
	{X: 1, Y: 0, Z: 0},
	{X: 0, Y: 1, Z: 0},
	{X: 0, Y: 0, Z: 1},
}

// Vertices3D represents a 3D structure composed of faces, volumes, origins, and associated metadata for spatial modeling.
type Vertices3D struct {
	facesA          []*Face
	facesB          []*Face
	facesAPtr       *[]*Face
	facesBPtr       *[]*Face
	totalFaces      int
	partFaceOffsets []int
	partFaceCounts  []int
	entity          *physics.Entity
	currentAction   int
	actionMaps      [][]int
	elements        []*Vertices3DEntry
	links           []config.Model3DLink
	linkTagIndices  []int
	executionOrder  []int
	volsA           []*Volume
	volsB           []*Volume
	originsA        []geometry.XYZ
	originsB        []geometry.XYZ
	rotationsA      [][3]geometry.XYZ
	rotationsB      [][3]geometry.XYZ
}

// NewVertices3D initializes and returns a new Vertices3D object based on the provided Thing configuration and materials.
func NewVertices3D(cfg *config.Thing, materials *Materials) *Vertices3D {
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
		entry := NewVertices3DEntry(c, materials)
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
				fmt.Printf("[warning] Vertices3D %s: circular link detected (cycle start at part %d, closed by part %d -> %d), disconnecting link\n",
					cfg.Id, curr, prev, links[prev].Parent)
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
		expectedActions = len(elements[0].actionIntervals)
	}

	actionMaps := make([][]int, numParts)
	for i := 0; i < numParts; i++ {
		var aMap []int
		if i < len(rawActionMaps) && rawActionMaps[i] != nil {
			aMap = make([]int, len(rawActionMaps[i]))
			copy(aMap, rawActionMaps[i])
			if expectedActions > 0 && len(aMap) < expectedActions {
				fmt.Printf("[warning] Vertices3D %s: part %d action map has %d entries, expected at least %d, falling back to action 0 for missing entries\n",
					cfg.Id, i, len(aMap), expectedActions)
				padded := make([]int, expectedActions)
				copy(padded, aMap)
				aMap = padded
			}
		} else {
			fmt.Printf("[warning] Vertices3D %s: missing action map for part %d, falling back to action 0\n", cfg.Id, i)
			aMap = make([]int, expectedActions)
		}

		if elements[i] != nil {
			numActions := len(elements[i].actionIntervals)
			for actionIdx, targetIdx := range aMap {
				if targetIdx < 0 || (numActions > 0 && targetIdx >= numActions) {
					fmt.Printf("[warning] Vertices3D %s: part %d action %d maps to invalid sub-action %d (max %d), falling back to action 0\n",
						cfg.Id, i, actionIdx, targetIdx, numActions-1)
					aMap[actionIdx] = 0
				}
			}
		}
		actionMaps[i] = aMap
	}

	// Resolve link tag indices
	linkTagIndices := make([]int, numParts)
	for i, link := range links {
		linkTagIndices[i] = -1
		if link.Parent >= 0 && link.Parent < numParts && elements[link.Parent] != nil {
			tagIdx := elements[link.Parent].GetTagIndex(link.Tag)
			linkTagIndices[i] = tagIdx
			if tagIdx < 0 {
				fmt.Printf("[warning] Vertices3D %s: tag %q not found on parent %d for part %d, inheriting parent origin and rotation without tag offset\n",
					cfg.Id, link.Tag, link.Parent, i)
			} else {
				for frameIdx, vol := range elements[link.Parent].volumes {
					if vol != nil {
						if _, _, ok := vol.GetVertexTag(tagIdx); !ok {
							fmt.Printf("[warning] Vertices3D %s: tag %q (idx %d) missing in frame %d of parent %d for part %d, inheriting parent origin and rotation without tag offset\n",
								cfg.Id, link.Tag, tagIdx, frameIdx, link.Parent, i)
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

	rotationsA := make([][3]geometry.XYZ, numParts)
	rotationsB := make([][3]geometry.XYZ, numParts)
	for i := 0; i < numParts; i++ {
		rotationsA[i] = _identityMatrix
		rotationsB[i] = _identityMatrix
	}

	v := &Vertices3D{
		elements:        elements,
		totalFaces:      totalFaces,
		partFaceOffsets: partFaceOffsets,
		partFaceCounts:  partFaceCounts,
		currentAction:   -1,
		actionMaps:      actionMaps,
		facesA:          make([]*Face, totalFaces),
		facesB:          make([]*Face, totalFaces),
		links:           links,
		linkTagIndices:  linkTagIndices,
		executionOrder:  executionOrder,
		volsA:           make([]*Volume, numParts),
		volsB:           make([]*Volume, numParts),
		originsA:        make([]geometry.XYZ, numParts),
		originsB:        make([]geometry.XYZ, numParts),
		rotationsA:      rotationsA,
		rotationsB:      rotationsB,
	}
	v.facesAPtr = &v.facesA
	v.facesBPtr = &v.facesB

	// Initialize face buffers using partFaceOffsets to decouple buffer layout from execution order
	for i, el := range v.elements {
		if el == nil || len(el.volumes) == 0 || el.volumes[0] == nil {
			continue
		}
		//faces, count := el.volumes[0].GetFaces()
		target := el.volumes[0]
		count := target.GetFaceCount()
		offset := v.partFaceOffsets[i]
		for j := 0; j < count; j++ {
			src := target.GetFace(j) //(*faces)[j]
			idx := offset + j

			v.facesA[idx] = &Face{
				material: src.material,
				u:        src.u,
				v:        src.v,
			}
			v.facesB[idx] = &Face{
				material: src.material,
				u:        src.u,
				v:        src.v,
			}
		}
	}

	if len(v.elements) > 0 && v.elements[0] != nil {
		v.entity = v.elements[0].GetEntity()
	}

	return v
}

// GetVolume retrieves the first Volume instance associated with the Vertices3D object.
func (v *Vertices3D) GetVolume() *Volume {
	if len(v.elements) > 0 && v.elements[0] != nil {
		return v.elements[0].viewVolume
	}
	return nil
}

// GetEntity retrieves the associated physics.Entity instance of the Vertices3D object.
func (v *Vertices3D) GetEntity() *physics.Entity {
	if len(v.elements) > 0 && v.elements[0] != nil {
		return v.elements[0].GetEntity()
	}
	return v.entity
}

// GetAABB returns the axis-aligned bounding box (AABB) of the Vertices3D instance by delegating to its associated entity.
func (v *Vertices3D) GetAABB() *aabb.AABB {
	if e := v.GetEntity(); e != nil {
		return e.GetAABB()
	}
	return nil
}

// SetAction updates the current action of Vertices3D and propagates the action change to all associated elements.
func (v *Vertices3D) SetAction(idx int) {
	if idx < 0 {
		return
	}
	if v.currentAction == idx {
		return
	}
	v.currentAction = idx
	for _, i := range v.executionOrder {
		if i < 0 || i >= len(v.elements) {
			continue
		}
		el := v.elements[i]
		if el == nil {
			continue
		}
		if i < len(v.actionMaps) && idx < len(v.actionMaps[i]) {
			targetAction := v.actionMaps[i][idx]
			if targetAction >= 0 && targetAction < len(el.actionIntervals) {
				el.SetAction(targetAction)
			} else {
				fmt.Printf("[warning] Vertices3D: part %d action %d maps to invalid sub-action %d (max %d), falling back to action 0\n",
					i, idx, targetAction, len(el.actionIntervals)-1)
				el.SetAction(0)
			}
		} else {
			fmt.Printf("[warning] Vertices3D: part %d has no mapping for action %d, falling back to action 0\n", i, idx)
			el.SetAction(0)
		}
	}
}

// GetVerticesCenter returns the displacement vector (dx, dy, dz) of the first element in the Vertices3D instance.
func (v *Vertices3D) GetVerticesCenter() (float64, float64, float64) {
	if len(v.elements) > 0 && v.elements[0] != nil {
		return v.elements[0].GetVerticesCenter()
	}
	return 0, 0, 0
}

// GetRenderMode returns the current render mode as a float64, used to determine how the 3D model is rendered.
func (v *Vertices3D) GetRenderMode() float64 {
	return RenderModeModel3D
}

// SetThing assigns the provided IThing instance to all non-nil entries in the elements slice of Vertices3D and updates cached entity.
func (v *Vertices3D) SetThing(t IThing) {
	for _, e := range v.elements {
		if e != nil {
			e.SetThing(t)
		}
	}
	if len(v.elements) > 0 && v.elements[0] != nil {
		v.entity = v.elements[0].GetEntity()
	}
}

// GetVertices computes and returns transformed vertex data for a specified tick, including faces, counts, interpolation, and render mode.
func (v *Vertices3D) GetVertices(tick uint64) (*[]*Face, int, *[]*Face, int, float64, float64) {
	var lerpTRet float64

	reset := func(i int) {
		v.volsA[i] = nil
		v.volsB[i] = nil
		v.originsA[i].X, v.originsA[i].Y, v.originsA[i].Z = 0, 0, 0
		v.originsB[i].X, v.originsB[i].Y, v.originsB[i].Z = 0, 0, 0
		v.rotationsA[i] = _identityMatrix
		v.rotationsB[i] = _identityMatrix
	}

	for _, i := range v.executionOrder {
		el := v.elements[i]
		if el == nil {
			reset(i)
			continue
		}
		volA, volB, lerpT := el.GetVolumesAt(tick)
		if volA == nil && volB == nil {
			reset(i)
			continue
		}
		// Fallback to the available volume if one is nil
		if volA == nil {
			volA = volB
		}
		if volB == nil {
			volB = volA
		}

		v.volsA[i] = volA
		v.volsB[i] = volB

		if i == 0 {
			lerpTRet = lerpT
		}

		pIdx := -1
		if i < len(v.links) {
			pIdx = v.links[i].Parent
		}

		if pIdx >= 0 && pIdx < len(v.elements) && pIdx != i && v.volsA[pIdx] != nil && v.volsB[pIdx] != nil {
			tagIdx := -1
			if i < len(v.linkTagIndices) {
				tagIdx = v.linkTagIndices[i]
			}
			if tagIdx >= 0 {
				tagOrigA, tagAxisA, okA := v.volsA[pIdx].GetVertexTag(tagIdx)
				tagOrigB, tagAxisB, okB := v.volsB[pIdx].GetVertexTag(tagIdx)
				if okA {
					rotOrigA := v3dRotatePoint(v.rotationsA[pIdx], tagOrigA)
					v.originsA[i].X = v.originsA[pIdx].X + rotOrigA.X
					v.originsA[i].Y = v.originsA[pIdx].Y + rotOrigA.Y
					v.originsA[i].Z = v.originsA[pIdx].Z + rotOrigA.Z
					v.rotationsA[i] = v3dMultiplyMatrix(tagAxisA, v.rotationsA[pIdx])
				} else {
					fmt.Printf("[warning] Vertices3D: tag %q (idx %d) not found in frame A of parent %d for part %d, inheriting parent origin and rotation without tag offset\n", v.links[i].Tag, tagIdx, pIdx, i)
					v.originsA[i] = v.originsA[pIdx]
					v.rotationsA[i] = v.rotationsA[pIdx]
				}
				if okB {
					rotOrigB := v3dRotatePoint(v.rotationsB[pIdx], tagOrigB)
					v.originsB[i].X = v.originsB[pIdx].X + rotOrigB.X
					v.originsB[i].Y = v.originsB[pIdx].Y + rotOrigB.Y
					v.originsB[i].Z = v.originsB[pIdx].Z + rotOrigB.Z
					v.rotationsB[i] = v3dMultiplyMatrix(tagAxisB, v.rotationsB[pIdx])
				} else {
					fmt.Printf("[warning] Vertices3D: tag %q (idx %d) not found in frame B of parent %d for part %d, inheriting parent origin and rotation without tag offset\n", v.links[i].Tag, tagIdx, pIdx, i)
					v.originsB[i] = v.originsB[pIdx]
					v.rotationsB[i] = v.rotationsB[pIdx]
				}
			} else {
				fmt.Printf("[warning] Vertices3D: missing tag %q on parent %d for part %d, inheriting parent origin and rotation without tag offset\n", v.links[i].Tag, pIdx, i)
				v.originsA[i] = v.originsA[pIdx]
				v.originsB[i] = v.originsB[pIdx]
				v.rotationsA[i] = v.rotationsA[pIdx]
				v.rotationsB[i] = v.rotationsB[pIdx]
			}
		} else {
			v.originsA[i].X, v.originsA[i].Y, v.originsA[i].Z = 0, 0, 0
			v.originsB[i].X, v.originsB[i].Y, v.originsB[i].Z = 0, 0, 0
			v.rotationsA[i] = _identityMatrix
			v.rotationsB[i] = _identityMatrix
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

		origA := v.originsA[i]
		origB := v.originsB[i]
		rotA := v.rotationsA[i]
		rotB := v.rotationsB[i]

		isIdentA := v3dIsIdentity(rotA)
		isIdentB := v3dIsIdentity(rotB)

		for j := 0; j < count; j++ {
			srcA := volA.GetFace(j)
			dstA := v.facesA[offset+j]
			if isIdentA {
				if origA.X == 0 && origA.Y == 0 && origA.Z == 0 {
					v3dCopyFacePoints(dstA, srcA)
				} else {
					v3dTransformPoints(dstA, srcA, origA)
				}
			} else {
				v3dTransformPointsWithMatrix(dstA, srcA, origA, rotA)
			}

			srcB := volB.GetFace(j)
			dstB := v.facesB[offset+j]
			if isIdentB {
				if origB.X == 0 && origB.Y == 0 && origB.Z == 0 {
					v3dCopyFacePoints(dstB, srcB)
				} else {
					v3dTransformPoints(dstB, srcB, origB)
				}
			} else {
				v3dTransformPointsWithMatrix(dstB, srcB, origB, rotB)
			}
		}

		// Zero out any leftover faces in this part's buffer slot to avoid displaying stale frame data
		for j := count; j < partCount; j++ {
			v3dZeroFace(v.facesA[offset+j])
			v3dZeroFace(v.facesB[offset+j])
		}
	}

	return v.facesAPtr, v.totalFaces, v.facesBPtr, v.totalFaces, lerpTRet, v.GetRenderMode()
}

// v3dTransformPoints transforms the points of src Face to a new position relative to origin, storing them in dst.
func v3dTransformPoints(dst *Face, src *Face, origin geometry.XYZ) {
	dst.tri[0].X = src.tri[0].X + origin.X
	dst.tri[0].Y = src.tri[0].Y + origin.Y
	dst.tri[0].Z = src.tri[0].Z + origin.Z
	dst.tri[1].X = src.tri[1].X + origin.X
	dst.tri[1].Y = src.tri[1].Y + origin.Y
	dst.tri[1].Z = src.tri[1].Z + origin.Z
	dst.tri[2].X = src.tri[2].X + origin.X
	dst.tri[2].Y = src.tri[2].Y + origin.Y
	dst.tri[2].Z = src.tri[2].Z + origin.Z
}

// v3dTransformPointsWithMatrix rotates the points of src Face by matrix m and translates them by origin, storing in dst.
func v3dTransformPointsWithMatrix(dst *Face, src *Face, origin geometry.XYZ, m [3]geometry.XYZ) {
	for k := 0; k < 3; k++ {
		rx := src.tri[k].X*m[0].X + src.tri[k].Y*m[1].X + src.tri[k].Z*m[2].X
		ry := src.tri[k].X*m[0].Y + src.tri[k].Y*m[1].Y + src.tri[k].Z*m[2].Y
		rz := src.tri[k].X*m[0].Z + src.tri[k].Y*m[1].Z + src.tri[k].Z*m[2].Z

		dst.tri[k].X = rx + origin.X
		dst.tri[k].Y = ry + origin.Y
		dst.tri[k].Z = rz + origin.Z
	}
}

// v3dCopyFacePoints copies the vertex coordinates from the source Face to the destination Face.
func v3dCopyFacePoints(dst *Face, src *Face) {
	dst.tri[0].X = src.tri[0].X
	dst.tri[0].Y = src.tri[0].Y
	dst.tri[0].Z = src.tri[0].Z
	dst.tri[1].X = src.tri[1].X
	dst.tri[1].Y = src.tri[1].Y
	dst.tri[1].Z = src.tri[1].Z
	dst.tri[2].X = src.tri[2].X
	dst.tri[2].Y = src.tri[2].Y
	dst.tri[2].Z = src.tri[2].Z
}

// v3dZeroFace collapses a face to zero coordinates so that it produces no visible geometry.
func v3dZeroFace(dst *Face) {
	if dst == nil {
		return
	}
	dst.tri[0] = geometry.XYZ{}
	dst.tri[1] = geometry.XYZ{}
	dst.tri[2] = geometry.XYZ{}
}

// v3dIsIdentity returns true if the matrix is an identity matrix.
func v3dIsIdentity(m [3]geometry.XYZ) bool {
	return m[0].X == 1 && m[0].Y == 0 && m[0].Z == 0 &&
		m[1].X == 0 && m[1].Y == 1 && m[1].Z == 0 &&
		m[2].X == 0 && m[2].Y == 0 && m[2].Z == 1
}

// v3dRotatePoint rotates point p by matrix m.
func v3dRotatePoint(m [3]geometry.XYZ, p geometry.XYZ) geometry.XYZ {
	return geometry.XYZ{
		X: p.X*m[0].X + p.Y*m[1].X + p.Z*m[2].X,
		Y: p.X*m[0].Y + p.Y*m[1].Y + p.Z*m[2].Y,
		Z: p.X*m[0].Z + p.Y*m[1].Z + p.Z*m[2].Z,
	}
}

// v3dMultiplyMatrix combines child and parent transformation matrices.
func v3dMultiplyMatrix(child, parent [3]geometry.XYZ) [3]geometry.XYZ {
	return [3]geometry.XYZ{
		v3dRotatePoint(parent, child[0]),
		v3dRotatePoint(parent, child[1]),
		v3dRotatePoint(parent, child[2]),
	}
}
