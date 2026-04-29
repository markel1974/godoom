package wolfstein

import "github.com/markel1974/godoom/mr_tech/config"

// Builder represents a configurable utility type that constructs complex data structures or configurations.
type Builder struct {
}

// NewBuilder initializes and returns a new instance of Builder.
func NewBuilder() *Builder {
	return &Builder{}
}

// Build constructs a Root configuration by parsing original map data through the Parser, based on the specified level.
func (b *Builder) Build(level int) (*config.Root, error) {
	w, h, data := GetOriginalMapData()
	wp := NewParser(8, 15, true)
	return wp.Parse(w, h, data)
}
