package quickhull

type meshBuilder struct {
	// Mesh data
	faces     []meshBuilderFace
	halfEdges []HalfEdge

	// When the mesh is modified and Faces and half edges are removed from it, we do not actually remove them from the container vectors.
	// Insted, they are marked as disabled which means that the indices can be reused when we need to add new Faces and half edges to the mesh.
	// We store the free indices in the following vectors.
	disabledFaces     []int
	disabledHalfEdges []int
}

func (mb *meshBuilder) addFace() int {
	if nDisabled := len(mb.disabledFaces); nDisabled > 0 {
		lastDisabledIndex := nDisabled - 1
		index := mb.disabledFaces[lastDisabledIndex]
		f := &mb.faces[index]
		assertTrue(f.isDisabled())
		assertTrue(f.pointsOnPositiveSide == nil)
		f.mostDistantPointDist = 0
		mb.disabledFaces = mb.disabledFaces[:lastDisabledIndex]

		return index
	}

	mb.faces = append(mb.faces, newMeshBuilderFace())
	return len(mb.faces) - 1
}

func (mb *meshBuilder) addHalfEdge() int {
	if nDisabled := len(mb.disabledHalfEdges); nDisabled > 0 {
		lastDisabledIndex := nDisabled - 1
		index := mb.disabledHalfEdges[lastDisabledIndex]
		mb.disabledHalfEdges = mb.disabledHalfEdges[:lastDisabledIndex]

		return index
	}

	mb.halfEdges = append(mb.halfEdges, HalfEdge{})
	return len(mb.halfEdges) - 1
}

func (mb *meshBuilder) disableFace(faceIndex int) (pointsOnPositiveSide []int) {
	f := &mb.faces[faceIndex]
	f.disable()
	mb.disabledFaces = append(mb.disabledFaces, faceIndex)
	pointsOnPositiveSide = f.pointsOnPositiveSide
	f.pointsOnPositiveSide = nil
	return
}

func (mb *meshBuilder) disableHalfEdge(heIndex int) {
	he := &mb.halfEdges[heIndex]
	he.disable()
	mb.disabledHalfEdges = append(mb.disabledHalfEdges, heIndex)
}

func (mb *meshBuilder) vertexIndicesOfFace(f meshBuilderFace) [3]int {
	var v [3]int
	he := mb.halfEdges[f.halfEdgeIndex]
	v[0] = he.EndVertex
	he = mb.halfEdges[he.Next]
	v[1] = he.EndVertex
	he = mb.halfEdges[he.Next]
	v[2] = he.EndVertex
	return v
}

func (mb *meshBuilder) vertexIndicesOfHalfEdge(he HalfEdge) [2]int {
	return [2]int{mb.halfEdges[he.Opp].EndVertex, he.EndVertex}
}
func (mb *meshBuilder) halfEdgeIndicesOfFace(f meshBuilderFace) [3]int {
	second := mb.halfEdges[f.halfEdgeIndex].Next
	return [3]int{f.halfEdgeIndex, second, mb.halfEdges[second].Next}
}

// Create a mesh with initial tetrahedron ABCD. Dot product of AB with the normal of triangle ABC should be negative.
func newMeshBuilder(a, b, c, d int) meshBuilder {
	// Create halfedges
	var mb meshBuilder

	ab := HalfEdge{
		EndVertex: b,
		Opp:       6,
		Face:      0,
		Next:      1,
	}
	mb.halfEdges = append(mb.halfEdges, ab)

	bc := HalfEdge{
		EndVertex: c,
		Opp:       9,
		Face:      0,
		Next:      2,
	}
	mb.halfEdges = append(mb.halfEdges, bc)

	ca := HalfEdge{
		EndVertex: a,
		Opp:       3,
		Face:      0,
		Next:      0,
	}
	mb.halfEdges = append(mb.halfEdges, ca)

	ac := HalfEdge{
		EndVertex: c,
		Opp:       2,
		Face:      1,
		Next:      4,
	}
	mb.halfEdges = append(mb.halfEdges, ac)

	cd := HalfEdge{
		EndVertex: d,
		Opp:       11,
		Face:      1,
		Next:      5,
	}
	mb.halfEdges = append(mb.halfEdges, cd)

	da := HalfEdge{
		EndVertex: a,
		Opp:       7,
		Face:      1,
		Next:      3,
	}
	mb.halfEdges = append(mb.halfEdges, da)

	ba := HalfEdge{
		EndVertex: a,
		Opp:       0,
		Face:      2,
		Next:      7,
	}
	mb.halfEdges = append(mb.halfEdges, ba)

	ad := HalfEdge{
		EndVertex: d,
		Opp:       5,
		Face:      2,
		Next:      8,
	}
	mb.halfEdges = append(mb.halfEdges, ad)

	db := HalfEdge{
		EndVertex: b,
		Opp:       10,
		Face:      2,
		Next:      6,
	}
	mb.halfEdges = append(mb.halfEdges, db)

	cb := HalfEdge{
		EndVertex: b,
		Opp:       1,
		Face:      3,
		Next:      10,
	}
	mb.halfEdges = append(mb.halfEdges, cb)

	bd := HalfEdge{
		EndVertex: d,
		Opp:       8,
		Face:      3,
		Next:      11,
	}
	mb.halfEdges = append(mb.halfEdges, bd)

	dc := HalfEdge{
		EndVertex: c,
		Opp:       4,
		Face:      3,
		Next:      9,
	}
	mb.halfEdges = append(mb.halfEdges, dc)

	// Create Faces
	abc := meshBuilderFace{
		halfEdgeIndex: 0,
	}
	mb.faces = append(mb.faces, abc)

	acd := meshBuilderFace{
		halfEdgeIndex: 3,
	}
	mb.faces = append(mb.faces, acd)

	bad := meshBuilderFace{
		halfEdgeIndex: 6,
	}
	mb.faces = append(mb.faces, bad)

	cbd := meshBuilderFace{
		halfEdgeIndex: 9,
	}
	mb.faces = append(mb.faces, cbd)

	return mb
}

type meshBuilderFace struct {
	halfEdgeIndex                   int
	plane                           plane
	mostDistantPoint                int
	mostDistantPointDist            float64
	visibilityCheckedOnIteration    int
	isVisibleFaceOnCurrentIteration bool
	inFaceStack                     bool
	horizonEdgesOnCurrentIteration  byte // Bit for each half edge assigned to this Face, each being 0 or 1 depending on whether the edge belongs to horizon edge
	pointsOnPositiveSide            []int
}

func (mbf *meshBuilderFace) disable() {
	mbf.halfEdgeIndex = disabledInt
}

func (mbf *meshBuilderFace) isDisabled() bool {
	return mbf.halfEdgeIndex == disabledInt
}

func newMeshBuilderFace() meshBuilderFace {
	return meshBuilderFace{halfEdgeIndex: disabledInt}
}
