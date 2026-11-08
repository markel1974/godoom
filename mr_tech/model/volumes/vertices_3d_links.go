package volumes

import (
	"fmt"

	"github.com/markel1974/godoom/mr_tech/config"
)

// linkInvalidParent marks a link as disconnected or without a valid parent.
const linkInvalidParent = -1

// Defines DFS traversal states used to track node visitation progress.
const (
	dfsUnvisited = 0
	dfsVisiting  = 1
	dfsVisited   = 2
)

// Vertices3DLink represents a hierarchical attachment from a model part to its parent through a tag.
type Vertices3DLink struct {
	parent int
	tag    string
}

// NewVertices3DLink creates a link to the specified parent element and tag.
func NewVertices3DLink(parent int, tag string) *Vertices3DLink {
	return &Vertices3DLink{
		parent: parent,
		tag:    tag,
	}
}

// Vertices3DLinks manages hierarchical parent connections and update ordering for linked 3D model parts.
type Vertices3DLinks struct {
	data           []*Vertices3DLink
	tagIndices     []int
	executionOrder []int
}

// NewVertices3DLinks creates an empty Vertices3DLinks collection.
func NewVertices3DLinks() *Vertices3DLinks {
	ls := &Vertices3DLinks{}
	return ls
}

// Setup initializes link data, validates parent relationships, resolves tag indices, and builds execution order.
func (ls *Vertices3DLinks) Setup(cfg *config.Model3D, elements []*Vertices3DEntry) error {
	numParts := len(elements)
	ls.data = make([]*Vertices3DLink, numParts)
	ls.tagIndices = make([]int, numParts)
	ls.executionOrder = make([]int, 0, numParts)

	for i := 0; i < numParts; i++ {
		if i < len(cfg.Links) {
			ls.data[i] = NewVertices3DLink(cfg.Links[i].Parent, cfg.Links[i].Tag)
		} else {
			ls.data[i] = NewVertices3DLink(linkInvalidParent, "")
		}
	}

	// Reject invalid parents and self-links
	for i := 0; i < numParts; i++ {
		if ls.data[i].parent == i {
			fmt.Printf("[warning] Vertices3D: part %d links to itself, disconnecting link\n", i)
			ls.data[i].parent = linkInvalidParent
			ls.data[i].tag = ""
		} else if ls.data[i].parent >= numParts {
			fmt.Printf("[warning] Vertices3D: part %d links to invalid parent %d (numParts=%d), disconnecting link\n", i, ls.data[i].parent, numParts)
			ls.data[i].parent = linkInvalidParent
			ls.data[i].tag = ""
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
				fmt.Printf("[warning] Vertices3D: circular link detected (cycle start at part %d, closed by part %d -> %d), disconnecting link\n", curr, prev, ls.GetParent(prev))
				ls.Invalidate(prev)
				break
			}
			pathIndex[curr] = len(path)
			path = append(path, curr)
			curr = ls.GetParent(curr)
		}
		for _, node := range path {
			visited[node] = true
		}
	}

	for i, link := range ls.data {
		ls.tagIndices[i] = -1
		if link.parent >= 0 && link.parent < numParts && elements[link.parent] != nil {
			tagIdx := elements[link.parent].GetTagIndex(link.tag)
			ls.tagIndices[i] = tagIdx
			if tagIdx < 0 {
				fmt.Printf("[warning] Vertices3D: tag %q not found on parent %d for part %d, inheriting parent origin and rotation without tag offset\n", link.tag, link.parent, i)
			} else {
				for frameIdx, vol := range elements[link.parent].volumes {
					if vol != nil {
						if !vol.HasVertexTag(tagIdx) {
							fmt.Printf("[warning] Vertices3D: tag %q (idx %d) missing in frame %d of parent %d for part %d, inheriting parent origin and rotation without tag offset\n", link.tag, tagIdx, frameIdx, link.parent, i)
						}
					}
				}
			}
		}
	}

	dfsState := make([]int, numParts)
	var visit func(int)
	visit = func(node int) {
		if node < 0 || node >= numParts || dfsState[node] == dfsVisited {
			return
		}
		if dfsState[node] == dfsVisiting {
			fmt.Printf("[warning] Vertices3D: cycle detected during DFS at node %d, disconnecting link\n", node)
			ls.Invalidate(node)
			return
		}
		dfsState[node] = dfsVisiting
		pIdx := ls.GetParent(node)
		if pIdx >= 0 && pIdx < numParts && pIdx != node {
			visit(pIdx)
		}
		dfsState[node] = dfsVisited
		ls.executionOrder = append(ls.executionOrder, node)
	}

	// Ensure all nodes are visited (including disconnected sub-trees)
	for i := 0; i < numParts; i++ {
		if dfsState[i] == dfsUnvisited {
			visit(i)
		}
	}

	return nil
}

// Invalidate disconnects the link at index i by clearing its parent and tag.
// It does nothing when i is outside the stored link range.
func (ls *Vertices3DLinks) Invalidate(i int) {
	if i >= 0 && i < len(ls.data) {
		ls.data[i].parent = -1
		ls.data[i].tag = ""
	}
}

// GetParent returns the parent index for i, or the invalid parent value if i is out of range.
func (ls *Vertices3DLinks) GetParent(i int) int {
	if i >= 0 && i < len(ls.data) {
		return ls.data[i].parent
	}
	return linkInvalidParent
}
