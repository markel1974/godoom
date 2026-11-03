package proc

// File represents the complete parsed structure of a Doom 3 .proc file.
type File struct {
	Version int
	Models  []*Model
	Portals []*Portal
}

// NewProcFile initializes a new empty File instance.
func NewProcFile() *File {
	return &File{
		Models:  make([]*Model, 0),
		Portals: make([]*Portal, 0),
	}
}
