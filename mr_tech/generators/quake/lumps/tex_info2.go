package lumps

const (
	// Quake II surface flags.
	surfLight    = 0x00000001 // SURF_LIGHT
	surfSlick    = 0x00000002 // SURF_SLICK
	surfSky      = 0x00000004 // SURF_SKY
	surfWarp     = 0x00000008 // SURF_WARP
	surfTrans33  = 0x00000010 // SURF_TRANS33
	surfTrans66  = 0x00000020 // SURF_TRANS66
	surfFlowing  = 0x00000040 // SURF_FLOWING
	surfNoDraw   = 0x00000080 // SURF_NODRAW
	surfHint     = 0x00000100 // SURF_HINT
	surfSkip     = 0x00000200 // SURF_SKIP
	surfNoLight  = 0x00000400 // SURF_NOLIGHT
	surfBump     = 0x00000800 // SURF_BUMP
	surfMetal    = 0x00001000 // SURF_METAL
	surfGlass    = 0x00002000 // SURF_GLASS
	surfNoImpact = 0x00004000 // SURF_NOIMPACT
	surfAlpha    = 0x00008000 // SURF_ALPHA
	surfDust     = 0x00010000 // SURF_DUST
)

// TexInfo2 represents texture mapping information for Quake II BSP files.
//
// Vecs defines the two texture vectors used for UV mapping calculations.
//
// Flags holds surface attributes such as sky, warp, transparency,
// flowing textures, and rendering properties.
//
// Value specifies additional data associated with the texture,
// traditionally used by the BSP/game code for light or texture values.
//
// TextureName is the name of the texture, stored as a null-terminated
// string.
//
// NextTexInfo holds the index of the next texture in the animation/
// texture chain, or -1 if there is no next TexInfo.
type TexInfo2 struct {
	Vecs        [2][4]float32
	Flags       uint32
	Value       uint32
	TextureName [32]byte
	NextTexInfo int32
}

// NoDraw checks if the texture is marked with SURF_NODRAW.
func (t TexInfo2) NoDraw() bool {
	return (t.Flags & surfNoDraw) != 0
}
