package config

// WAX represents a configurable and dynamic structure potentially associated with animations or materials in the system.
type WAX struct {
	Materials []*Material
}

// NewWAX creates and returns a new instance of the WAX struct.
func NewWAX() *WAX {
	return &WAX{}
}
