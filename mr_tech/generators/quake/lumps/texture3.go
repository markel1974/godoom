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

// IsSky determines if the RawFace3 object represents a sky surface by checking if the SURF_SKY flag is set in Info.Flags.
func (t *Texture3) IsSky() bool {
	return (t.Flags & 0x4) != 0 // SURF_SKY
}
