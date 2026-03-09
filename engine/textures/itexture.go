package textures

// ITextures represents an interface for managing and retrieving texture resources by name.
type ITextures interface {
	GetNames() []string
	Get(name []string) []*Texture
}
