package config

// MultiSprite represents a collection of materials used for multi-frame animations or visual compositions.
type MultiSprite struct {
	Materials []*Material
}

// NewMultiSprite creates and returns a new instance of MultiSprite with an initialized empty slice of Materials.
func NewMultiSprite() *MultiSprite {
	return &MultiSprite{}
}

// Add appends a Material to the MultiSprite's Material list and returns the index of the newly added Material.
func (ms *MultiSprite) Add(material *Material) int {
	counter := len(ms.Materials)
	ms.Materials = append(ms.Materials, material)
	return counter
}
