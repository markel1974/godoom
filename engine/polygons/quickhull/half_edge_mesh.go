package quickhull

// HalfEdgeMesh is a mesh consisting of half edges.
// See: https://www.openmesh.org/media/Documentations/OpenMesh-6.3-Documentation/a00010.html
type HalfEdgeMesh struct {
	Vertices  []Vector
	Faces     []Face
	HalfEdges []HalfEdge
}

// HalfEdge is a half edge.
// See: https://www.openmesh.org/media/Documentations/OpenMesh-6.3-Documentation/a00010.html
type HalfEdge struct {
	EndVertex int // Index of end vertex
	Opp       int // Index of opposite HalfEdge
	Face      int // Index of Face it belongs to
	Next      int // Index of next HalfEdge
}

const disabledInt = ^int(0)

func (he *HalfEdge) disable() {
	he.EndVertex = disabledInt
}

func (he HalfEdge) isDisabled() bool {
	return he.EndVertex == disabledInt
}

// Face of a half edge.
// See: https://www.openmesh.org/media/Documentations/OpenMesh-6.3-Documentation/a00010.html
type Face struct {
	HalfEdge int // Index of a bounding HalfEdge
}

func newHalfEdgeMesh(builder meshBuilder, vertices []Vector) HalfEdgeMesh {
	var heMesh HalfEdgeMesh

	faceMapping := make(map[int]int)
	halfEdgeMapping := make(map[int]int)
	vertexMapping := make(map[int]int)

	for i, f := range builder.faces {
		if f.isDisabled() {
			continue
		}

		heMesh.Faces = append(heMesh.Faces, Face{HalfEdge: f.halfEdgeIndex})
		faceMapping[i] = len(heMesh.Faces) - 1

		heIndicies := builder.halfEdgeIndicesOfFace(f)
		for _, heIndex := range heIndicies {
			vertexIndex := builder.halfEdges[heIndex].EndVertex
			if _, contains := vertexMapping[vertexIndex]; !contains {
				heMesh.Vertices = append(heMesh.Vertices, vertices[vertexIndex])
				vertexMapping[vertexIndex] = len(heMesh.Vertices) - 1
			}
		}
	}

	for i, he := range builder.halfEdges {
		if he.isDisabled() {
			continue
		}

		heMesh.HalfEdges = append(heMesh.HalfEdges, he)
		halfEdgeMapping[i] = len(heMesh.HalfEdges) - 1
	}

	for i := range heMesh.Faces {
		_, contains := halfEdgeMapping[heMesh.Faces[i].HalfEdge]
		assertTrue(contains)
		heMesh.Faces[i].HalfEdge = halfEdgeMapping[heMesh.Faces[i].HalfEdge]
	}

	for i := range heMesh.HalfEdges {
		heMesh.HalfEdges[i].Face = faceMapping[heMesh.HalfEdges[i].Face]
		heMesh.HalfEdges[i].Opp = halfEdgeMapping[heMesh.HalfEdges[i].Opp]
		heMesh.HalfEdges[i].Next = halfEdgeMapping[heMesh.HalfEdges[i].Next]
		heMesh.HalfEdges[i].EndVertex = vertexMapping[heMesh.HalfEdges[i].EndVertex]
	}

	return heMesh
}
