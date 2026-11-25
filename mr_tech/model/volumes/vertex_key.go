package volumes

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/geometry"
)

const (
	deformMaxSubdivisionPasses = 12
	deformMaxFaces             = 65536
	deformMinEdge              = 1.0
	minEdgeSq                  = (deformMinEdge * 2.0) * (deformMinEdge * 2.0)
)

type VertexKey struct {
	x, y, z uint32
}

func MakeVertexKey(p geometry.XYZ32) VertexKey {
	if p.X == 0 {
		p.X = 0
	}
	if p.Y == 0 {
		p.Y = 0
	}
	if p.Z == 0 {
		p.Z = 0
	}

	return VertexKey{
		x: math.Float32bits(p.X),
		y: math.Float32bits(p.Y),
		z: math.Float32bits(p.Z),
	}
}

type EdgeKey struct {
	a, b VertexKey
}

func MakeEdgeKey(a, b geometry.XYZ32) EdgeKey {
	lessVertex := func(a, b VertexKey) bool {
		if a.x != b.x {
			return a.x < b.x
		}
		if a.y != b.y {
			return a.y < b.y
		}
		return a.z < b.z
	}
	ka, kb := MakeVertexKey(a), MakeVertexKey(b)
	if lessVertex(kb, ka) {
		ka, kb = kb, ka
	}
	return EdgeKey{a: ka, b: kb}
}
