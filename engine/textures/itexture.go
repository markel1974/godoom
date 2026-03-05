package textures

// ITextures represents an interface for managing and retrieving texture resources by name.
type ITextures interface {
	Get(name string) *Texture
}
