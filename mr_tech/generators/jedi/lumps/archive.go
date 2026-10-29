package lumps

// IArchive represents an interface for managing game archives, including parsing, level management, and resource retrieval.
type IArchive interface {
	Parse(dir string) error

	GetLevels() []string

	SetLevel(levelNumber int) error

	List(p string) ([]string, error)

	GetPayload(name string) ([]byte, error)

	GetTextures() *Textures

	GetLevel() *Level

	GetEntities() *Entities

	AddTexture(texName string) ([]string, error)

	AddRawTexture(texName string, sizeX int, sizeY int, pixels []byte, dump bool)

	Close() error
}
