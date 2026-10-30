package textures

// ITextures provides methods to retrieve texture names and access textures by their names.
type ITextures interface {
	GetNames() []string

	GetFrames(name []string) []*Texture
}
