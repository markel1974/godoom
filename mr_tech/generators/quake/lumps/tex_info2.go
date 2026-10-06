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

// Light checks if the surface has the SURF_LIGHT flag.
func (t TexInfo2) Light() bool {
	return (t.Flags & surfLight) != 0
}

// Slick checks if the surface has the SURF_SLICK flag.
func (t TexInfo2) Slick() bool {
	return (t.Flags & surfSlick) != 0
}

// IsSky determines whether the texture has the SURF_SKY flag set.
func (t TexInfo2) IsSky() bool {
	return (t.Flags & surfSky) != 0
}

// IsWarp determines whether the surface has the SURF_WARP flag set.
//
// Quake II uses SURF_WARP for surfaces such as water, slime and lava.
// This identifies a warped/liquid-style surface, but does not by itself
// distinguish the liquid type.
func (t TexInfo2) IsWarp() bool {
	return (t.Flags & surfWarp) != 0
}

// IsTrans33 determines whether the surface has 33% transparency.
func (t TexInfo2) IsTrans33() bool {
	return (t.Flags & surfTrans33) != 0
}

// IsTrans66 determines whether the surface has 66% transparency.
func (t TexInfo2) IsTrans66() bool {
	return (t.Flags & surfTrans66) != 0
}

// IsFlowing determines whether the texture is marked as flowing.
func (t TexInfo2) IsFlowing() bool {
	return (t.Flags & surfFlowing) != 0
}

// IsHint determines whether the surface is a BSP hint surface.
func (t TexInfo2) IsHint() bool {
	return (t.Flags & surfHint) != 0
}

// IsSkip determines whether the surface has the SURF_SKIP flag.
func (t TexInfo2) IsSkip() bool {
	return (t.Flags & surfSkip) != 0
}

// NoLight determines whether the surface has the SURF_NOLIGHT flag.
func (t TexInfo2) NoLight() bool {
	return (t.Flags & surfNoLight) != 0
}

// IsBump determines whether the surface has the SURF_BUMP flag.
func (t TexInfo2) IsBump() bool {
	return (t.Flags & surfBump) != 0
}

// IsMetal determines whether the surface has the SURF_METAL flag.
func (t TexInfo2) IsMetal() bool {
	return (t.Flags & surfMetal) != 0
}

// IsGlass determines whether the surface has the SURF_GLASS flag.
func (t TexInfo2) IsGlass() bool {
	return (t.Flags & surfGlass) != 0
}

// NoImpact determines whether the surface has the SURF_NOIMPACT flag.
func (t TexInfo2) NoImpact() bool {
	return (t.Flags & surfNoImpact) != 0
}

// HasAlpha determines whether the surface has the SURF_ALPHA flag.
func (t TexInfo2) HasAlpha() bool {
	return (t.Flags & surfAlpha) != 0
}

// IsDust determines whether the surface has the SURF_DUST flag.
func (t TexInfo2) IsDust() bool {
	return (t.Flags & surfDust) != 0
}
