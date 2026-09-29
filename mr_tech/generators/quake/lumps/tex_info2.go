package lumps

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
