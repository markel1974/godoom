package model

import (
	"github.com/markel1974/godoom/mr_tech/model/config"
	"github.com/markel1974/godoom/mr_tech/model/geometry"
	"github.com/markel1974/godoom/mr_tech/physics"
)

// VertexNode represents a graph vertex with a unique ID, geometric position, and an associated bounding box.
type VertexNode struct {
	Id   int
	aabb *physics.AABB
	geometry.XY
}

// NewVertexNode creates a new VertexNode with the given ID and coordinates, initializing its bounding box (AABB).
func NewVertexNode(id int, xy geometry.XY) *VertexNode {
	const eps = 0.001
	return &VertexNode{
		Id:   id,
		XY:   xy,
		aabb: physics.NewAABB(xy.X-eps, xy.Y-eps, 0, xy.X+eps, xy.Y+eps, 0),
	}
}

// GetAABB retrieves the axis-aligned bounding box (AABB) associated with the VertexNode.
func (v *VertexNode) GetAABB() *physics.AABB {
	return v.aabb
}

// VertexEdges represents a structure that facilitates vertex and edge compilation from sector configurations.
type VertexEdges struct {
}

// NewVertexEdges creates and returns a new instance of the VertexEdges structure.
func NewVertexEdges() *VertexEdges {
	return &VertexEdges{}
}

// Construct constructs a polygon and associated edges from the given sectors using a spatial acceleration structure.
func (t *VertexEdges) Construct(cSectors []*config.ConfigSector) (geometry.Polygon, [][]geometry.Edge) {
	tree := physics.NewAABBTree(1024)
	var vertexes geometry.Polygon
	getOrAddVertex := func(p geometry.XY) int {
		foundId := -1
		tree.QueryPoint(p.X, p.Y, func(object physics.IAABB) bool {
			if v, ok := object.(*VertexNode); ok {
				if v.X == p.X && v.Y == p.Y {
					foundId = v.Id
					return true
				}
			}
			return false
		})
		if foundId >= 0 {
			return foundId
		}
		point := geometry.XY{X: p.X, Y: p.Y}
		idx := len(vertexes)
		vertexes = append(vertexes, point)
		vNode := NewVertexNode(idx, point)
		tree.InsertObject(vNode)
		return idx
	}

	sectorsEdges := make([][]geometry.Edge, len(cSectors))
	// 1. Costruzione dei bordi (vincoli) per il triangolatore
	for idx, cs := range cSectors {
		var edges []geometry.Edge
		for i, cn := range cs.Segments {
			vStart := getOrAddVertex(cn.Start)
			vEnd := getOrAddVertex(cn.End)
			edges = append(edges, geometry.Edge{V1Idx: vStart, V2Idx: vEnd, LdIdx: i, IsLeft: false})
		}
		sectorsEdges[idx] = edges
	}
	return vertexes, sectorsEdges
}
