package volumes

import (
	"fmt"

	"github.com/markel1974/godoom/mr_tech/config"
)

// linkInvalidParent marks a link as disconnected by indicating that it has no valid parent.
const linkInvalidParent = -1

// Link represents a hierarchical connection to a parent element identified by a tag.
type Link struct {
	parent int
	tag    string
}

// NewLink creates a link with the specified parent index and tag.
func NewLink(parent int, tag string) *Link {
	return &Link{
		parent: parent,
		tag:    tag,
	}
}

// Links stores parent-tag relationships for indexed model parts.
type Links struct {
	data []*Link
}

// NewLinks creates a fixed-size Links collection from link definitions, invalidating missing, self, or out-of-range parents.
func NewLinks(size int, links []config.Model3DLink) *Links {
	ls := &Links{
		data: make([]*Link, size),
	}
	for i := 0; i < size; i++ {
		if i < len(links) {
			ls.data[i] = NewLink(links[i].Parent, links[i].Tag)
		} else {
			ls.data[i] = NewLink(linkInvalidParent, "")
		}
	}

	// Reject invalid parents and self-links
	for i := 0; i < size; i++ {
		if ls.data[i].parent == i {
			fmt.Printf("[warning] Vertices3D: part %d links to itself, disconnecting link\n", i)
			ls.data[i].parent = linkInvalidParent
			ls.data[i].tag = ""
		} else if ls.data[i].parent >= size {
			fmt.Printf("[warning] Vertices3D: part %d links to invalid parent %d (numParts=%d), disconnecting link\n", i, ls.data[i].parent, size)
			ls.data[i].parent = linkInvalidParent
			ls.data[i].tag = ""
		}
	}
	return ls
}

// Invalidate marks the link at index i as invalid by clearing its parent and tag.
func (ls *Links) Invalidate(i int) {
	if i >= 0 && i < len(ls.data) {
		ls.data[i].parent = -1
		ls.data[i].tag = ""
	}
}

// GetParent returns the parent index for link i, or -1 if i is out of range.
func (ls *Links) GetParent(i int) int {
	if i >= 0 && i < len(ls.data) {
		return ls.data[i].parent
	}
	return linkInvalidParent
}
