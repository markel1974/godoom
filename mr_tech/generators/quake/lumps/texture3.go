package lumps

// Texture3 represents a texture in a 3D environment.
// It consists of a name, flags, and contents metadata.
// Name is a fixed-size identifier for the texture.
// Flags define specific properties or behavior for the texture.
// Contents describes how the texture interacts within the environment.
type Texture3 struct {
	Name     [64]byte
	Flags    uint32
	Contents uint32
}
