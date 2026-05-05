package config

// Sprite represents a 2D object that is rendered using a Material for visual and animation properties.
type Sprite struct {
	Material *Material
}

// NewConfigSprite creates and returns a new Sprite instance with the specified Material.
func NewConfigSprite(material *Material) *Sprite {
	return &Sprite{Material: material}
}
