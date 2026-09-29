package lumps

const (
	surfSky2    = 0x4
	surfNoDraw2 = 0x80
)

// TexInfo2 represents texture mapping information for Quake 2 BSP files.
// Vecs defines two texture vectors used for UV mapping calculations.
// Flags holds attributes for the surface such as visibility or rendering properties.
// Value specifies additional data for the texture, often used for switches or animations.
// TextureName is the name of the texture, stored as a null-terminated string.
// NextTexInfo holds the index of the next texture in the chain, or -1 if none.
type TexInfo2 struct {
	Vecs        [2][4]float32
	Flags       uint32
	Value       uint32
	TextureName [32]byte
	NextTexInfo int32
}

// NoDraw checks if the texture is marked with the SurfNoDraw2 flag, indicating it should not be rendered.
func (t TexInfo2) NoDraw() bool {
	return (t.Flags & surfNoDraw2) != 0
}

// IsSky determines whether the texture has the SurfSky2 flag set, indicating it represents a sky surface.
func (t TexInfo2) IsSky() bool {
	return (t.Flags & surfSky2) != 0
}
