package config

// Liquid represents a configurable entity that uses a Material for its visual properties and behaviors.
type Liquid struct {
	Material *Material
}

// NewConfigLiquid creates a new Liquid instance with the specified Material.
func NewConfigLiquid(material *Material) *Liquid {
	return &Liquid{Material: material}
}
