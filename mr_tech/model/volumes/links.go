package volumes

import (
	"fmt"
	"strings"
)

// linkInvalidIndex marks a link as disconnected or without a valid parent.
const linkInvalidIndex = -1

// Defines DFS traversal states used to track node visitation progress.
const (
	dfsUnvisited = 0
	dfsVisiting  = 1
	dfsVisited   = 2
)

// Link represents a hierarchical attachment from a model part to its parent through a tag.
type Link struct {
	parent   int
	tag      string
	tagIndex int
}

// NewLink creates a link to the specified parent element and tag.
func NewLink(parent int, tag string) *Link {
	return &Link{
		parent:   parent,
		tag:      tag,
		tagIndex: linkInvalidIndex,
	}
}

// Invalidate disconnects the link by clearing its parent and tag.
func (vl *Link) Invalidate() {
	vl.parent = linkInvalidIndex
	vl.tag = ""
	vl.tagIndex = linkInvalidIndex
}

// SetTagIndex sets the tag index value for the Link object to the specified integer.
func (vl *Link) SetTagIndex(i int) {
	vl.tagIndex = i
}

// GetTagIndex returns the current tag index value of the Link instance.
func (vl *Link) GetTagIndex() int {
	return vl.tagIndex
}

// Links manages hierarchical parent connections and update ordering for linked 3D model parts.
type Links struct {
	data           []*Link
	executionOrder []int
}

// NewLinks creates an empty Links collection.
func NewLinks() *Links {
	ls := &Links{}
	return ls
}

// Setup initializes link data, validates parent relationships, resolves tag indices, and builds execution order.
func (ls *Links) Setup(links []*Link) error {
	numParts := len(links)
	ls.data = make([]*Link, numParts)
	ls.executionOrder = make([]int, 0, numParts)

	copy(ls.data, links)

	//Initialization
	// Reject invalid parents and self-links
	for i := 0; i < numParts; i++ {
		if ls.data[i].parent == i {
			fmt.Printf("[warning] Vertices3D: part %d links to itself, disconnecting link\n", i)
			ls.data[i].Invalidate()
		} else if ls.data[i].parent >= numParts {
			fmt.Printf("[warning] Vertices3D: part %d links to invalid parent %d (numParts=%d), disconnecting link\n", i, ls.data[i].parent, numParts)
			ls.data[i].Invalidate()
		}
	}

	// Reject circular dependencies by recording node positions in the current chain
	visited := make([]bool, numParts)
	pathIndex := make([]int, numParts)
	path := make([]int, 0, numParts)
	invalidIndices := make([]int, numParts)

	for k := range invalidIndices {
		invalidIndices[k] = linkInvalidIndex
	}

	for i := 0; i < numParts; i++ {
		if visited[i] {
			continue
		}
		copy(pathIndex, invalidIndices)
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
func (ls *Links) Invalidate(i int) {
	if i >= 0 && i < len(ls.data) {
		ls.data[i].Invalidate()
	}
}

// GetParent returns the parent index for i, or the invalid parent value if i is out of range.
func (ls *Links) GetParent(i int) int {
	if i >= 0 && i < len(ls.data) {
		return ls.data[i].parent
	}
	return linkInvalidIndex
}

// HasParent reports whether i is within the stored link range.
func (ls *Links) HasParent(i int) bool {
	if i >= 0 && i < len(ls.data) {
		return true
	}
	return false
}

// GetTag returns the lowercase tag for i, or an empty string when i is out of range.
func (ls *Links) GetTag(i int) string {
	if i >= 0 && i < len(ls.data) {
		return strings.ToLower(ls.data[i].tag)
	}
	return ""
}

// GetTagIndex returns the resolved tag index for i, or the invalid index when i is out of range.
func (ls *Links) GetTagIndex(i int) int {
	if i >= 0 && i < len(ls.data) {
		return ls.data[i].GetTagIndex()
	}
	return linkInvalidIndex
}
