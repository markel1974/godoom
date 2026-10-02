package config

// Model3DLink defines the hierarchical connection between a parent element and a child element via a tag.
type Model3DLink struct {
	Parent int
	Tag    string
}

// Model3D represents a multi-part 3D model used in Quake 3 (e.g. players).
type Model3D struct {
	Parts      []*Model3DEntry
	Links      []Model3DLink
	ActionMaps [][]int
}

// NewMD3 creates a new Model3D structure holding multiple parts.
func NewMD3(parts []*Model3DEntry, links []Model3DLink) *Model3D {
	return &Model3D{
		Parts: parts,
		Links: links,
	}
}
