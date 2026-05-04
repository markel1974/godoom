package config

// Sprite represents a 2D object that is rendered using a Material for visual and animation properties.
type Sprite struct {
	Material *Material
}

// NewSprite creates and returns a new Sprite instance with the specified Material.
func NewSprite(material *Material) *Sprite {
	return &Sprite{Material: material}
}
