package model

import (
	"fmt"

	"github.com/markel1974/godoom/mr_tech/model/config"
	"github.com/markel1974/godoom/mr_tech/model/geometry"
	"github.com/markel1974/godoom/mr_tech/physics"
)

// VertexNode represents a graph vertex with an identifier, 2D position, and an associated axis-aligned bounding box (AABB).
type VertexNode struct {
	Id   int
	aabb *physics.AABB
	geometry.XY
}

// NewVertexNode creates and returns a new VertexNode with the specified ID and 2D coordinates (XY) initialized with an AABB.
func NewVertexNode(id int, xy geometry.XY, eps float64) *VertexNode {
	//const eps = 0.001
	return &VertexNode{
		Id:   id,
		XY:   xy,
		aabb: physics.NewAABB(xy.X-eps, xy.Y-eps, 0, xy.X+eps, xy.Y+eps, 0),
	}
}

// GetAABB returns the axis-aligned bounding box (AABB) associated with the VertexNode.
func (v *VertexNode) GetAABB() *physics.AABB {
	return v.aabb
}

// VertexEdges represents a structure that associates a polygon with an AABB tree and its corresponding sector edges.
type VertexEdges struct {
	vertexes     geometry.Polygon
	tree         *physics.AABBTree
	sectorsEdges [][]geometry.Edge
	eps          float64
}

// NewVertexEdges creates and initializes a new instance of VertexEdges with an empty vertex list and a new AABBTree.
func NewVertexEdges(eps float64) *VertexEdges {
	return &VertexEdges{
		tree:         physics.NewAABBTree(1024),
		vertexes:     nil,
		sectorsEdges: nil,
		eps:          eps,
	}
}

func (t *VertexEdges) getOrAddVertex(p geometry.XY) int {
	foundId := -1
	t.tree.QueryPoint(p.X, p.Y, func(object physics.IAABB) bool {
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
	idx := len(t.vertexes)
	t.vertexes = append(t.vertexes, point)
	vNode := NewVertexNode(idx, point, t.eps)
	t.tree.InsertObject(vNode)
	return idx
}

// Construct builds the vertex and edge data structures from the provided configuration sectors.
func (t *VertexEdges) Construct(config *config.ConfigRoot) {
	// STEP 1: Priming dell'AABB Tree
	// Se il pool esiste, lo usiamo per "pre-occupare" lo spazio.
	// Questo garantisce che ogni getOrAddVertex successivo faccia uno SNAP perfetto.
	if len(config.Vertices) > 0 {
		for _, point := range config.Vertices {
			idx := len(t.vertexes)
			t.vertexes = append(t.vertexes, point)
			vNode := NewVertexNode(idx, point, t.eps)
			t.tree.InsertObject(vNode)
		}
	}

	// STEP 2: Costruzione degli Edges (Constraints)
	// Ora t.sectorsEdges viene popolato usando il pool appena creato (o creandone uno al volo)
	t.sectorsEdges = make([][]geometry.Edge, len(config.Sectors))

	for configIdx, cs := range config.Sectors {
		var edges []geometry.Edge
		for i, cn := range cs.Segments {
			// getOrAddVertex cercherà nell'AABB Tree.
			// Grazie al priming dello Step 1, troverà gli indici dei vertici originali.
			vStart := t.getOrAddVertex(cn.Start)
			vEnd := t.getOrAddVertex(cn.End)

			edges = append(edges, geometry.Edge{
				V1Idx: vStart,
				V2Idx: vEnd,
				Index: i, // L'indice del segmento nel settore
			})
		}
		t.sectorsEdges[configIdx] = edges
	}
}

// GetTriangles returns a collection of triangulated polygons for the specified sector index or an error if the index is invalid.
// The returned polygons represent non-overlapping triangle groups derived from the sector's edges.
func (t *VertexEdges) GetTriangles(configIdx int) ([][]geometry.Polygon, error) {
	if configIdx < 0 || configIdx >= len(t.sectorsEdges) {
		return nil, fmt.Errorf("invalid sector index %d", configIdx)
	}
	edges := t.sectorsEdges[configIdx]
	triContainer := t.vertexes.TriangulateEdges(edges, len(t.vertexes))
	return triContainer, nil
}
