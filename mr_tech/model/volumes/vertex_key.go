package volumes

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/geometry"
)

const (
	deformMaxSubdivisionPasses = 12
	deformMaxFaces             = 65536
	deformMinEdge              = 1.0
)

type vertexKey struct {
	x, y, z uint64
}

type edgeKey struct {
	a, b vertexKey
}

func makeVertexKey(p geometry.XYZ) vertexKey {
	if p.X == 0 {
		p.X = 0
	}
	if p.Y == 0 {
		p.Y = 0
	}
	if p.Z == 0 {
		p.Z = 0
	}

	return vertexKey{
		x: math.Float64bits(p.X),
		y: math.Float64bits(p.Y),
		z: math.Float64bits(p.Z),
	}
}

func lessVertex(a, b vertexKey) bool {
	if a.x != b.x {
		return a.x < b.x
	}
	if a.y != b.y {
		return a.y < b.y
	}
	return a.z < b.z
}

func makeEdgeKey(a, b geometry.XYZ) edgeKey {
	ka, kb := makeVertexKey(a), makeVertexKey(b)
	if lessVertex(kb, ka) {
		ka, kb = kb, ka
	}
	return edgeKey{a: ka, b: kb}
}
