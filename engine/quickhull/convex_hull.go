package quickhull

import (
)

type ConvexHull struct {
	optimizedVertexBuffer []Vector
	Vertices              []Vector
	Indices               []int
}

func (hull ConvexHull) Triangles() [][3]Vector {
	triangles := make([][3]Vector, len(hull.Indices)/3)

	for i, idx := range hull.Indices {
		triangles[i/3][i%3] = hull.Vertices[idx]
	}

	return triangles
}

func newConvexHull(mesh meshBuilder, pointCloud []Vector, ccw bool, useOriginalIndices bool) ConvexHull {
	var hull ConvexHull

	faceProcessed := make([]bool, len(mesh.faces))
	var faceStack []int
	for i, f := range mesh.faces {
		if !f.isDisabled() {
			faceStack = append(faceStack, i)
			break
		}
	}
	if len(faceStack) == 0 {
		return hull
	}

	finalMeshFaceCount := len(mesh.faces) - len(mesh.disabledFaces)
	hull.Indices = make([]int, 0, finalMeshFaceCount*3)
	vertexIndexMapping := make(map[int]int) // Map vertex indices from original point cloud to the new mesh vertex indices

	for len(faceStack) > 0 {
		lastFaceIndex := len(faceStack) - 1
		var top int
		top, faceStack = faceStack[lastFaceIndex], faceStack[:lastFaceIndex]
		topFace := mesh.faces[top]
		assertTrue(!topFace.isDisabled())
		if faceProcessed[top] {
			continue
		}

		faceProcessed[top] = true
		halfEdges := mesh.halfEdgeIndicesOfFace(topFace)
		adjacent := []int{mesh.halfEdges[mesh.halfEdges[halfEdges[0]].Opp].Face, mesh.halfEdges[mesh.halfEdges[halfEdges[1]].Opp].Face, mesh.halfEdges[mesh.halfEdges[halfEdges[2]].Opp].Face}
		for _, a := range adjacent {
			if !faceProcessed[a] && !mesh.faces[a].isDisabled() {
				faceStack = append(faceStack, a)
			}
		}

		vertices := mesh.vertexIndicesOfFace(topFace)
		if !useOriginalIndices {
			for i, v := range vertices {
				it, found := vertexIndexMapping[v]
				if !found {
					hull.optimizedVertexBuffer = append(hull.optimizedVertexBuffer, pointCloud[v])
					addedIdx := len(hull.optimizedVertexBuffer) - 1
					vertexIndexMapping[v] = addedIdx
					vertices[i] = addedIdx
				} else {
					vertices[i] = it
				}
			}
		}

		hull.Indices = append(hull.Indices, vertices[0])
		if ccw {
			hull.Indices = append(hull.Indices, vertices[2])
			hull.Indices = append(hull.Indices, vertices[1])
		} else {
			hull.Indices = append(hull.Indices, vertices[1])
			hull.Indices = append(hull.Indices, vertices[2])
		}
	}

	if useOriginalIndices {
		hull.Vertices = pointCloud
	} else {
		hull.Vertices = hull.optimizedVertexBuffer
	}

	return hull
}
