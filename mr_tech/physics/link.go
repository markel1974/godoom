package physics

import "strings"

// Link represents a connection with a parent index, a tag, and a resolved tag index.
type Link struct {
	parent   int
	tag      string
	tagIndex int
}

// NewLink creates and returns a new Link instance with a specified parent index and tag.
func NewLink(parent int, tag string) *Link {
	return &Link{
		parent:   parent,
		tag:      strings.ToLower(strings.TrimSpace(tag)),
		tagIndex: LinkInvalidIndex,
	}
}

// GetParent returns the parent index of the link.
func (vl *Link) GetParent() int {
	return vl.parent
}

// GetTag returns the tag associated with the link as a string.
func (vl *Link) GetTag() string {
	return vl.tag
}

// Invalidate clears the link's parent, tag, and tag index to reset its state and mark it as invalid.
func (vl *Link) Invalidate() {
	vl.parent = LinkInvalidIndex
	vl.tag = ""
	vl.tagIndex = LinkInvalidIndex
}

// SetTagIndex sets the tag index of the Link to the given integer value.
func (vl *Link) SetTagIndex(i int) {
	vl.tagIndex = i
}

// GetTagIndex returns the tag index associated with the Link instance.
func (vl *Link) GetTagIndex() int {
	return vl.tagIndex
}
