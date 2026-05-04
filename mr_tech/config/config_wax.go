package config

type WAXFrame struct {
	SizeX     int
	SizeY     int
	InsertX   int
	InsertY   int
	TextureID string
	Flip      bool
}

type WAXView struct {
	Frames []*WAXFrame
}

// WAX represents a configurable and dynamic structure potentially associated with animations or materials in the system.
type WAX struct {
	Views []*WAXView
}

// NewWAX creates and returns a new instance of the WAX struct.
func NewWAX() *WAX {
	return &WAX{}
}
