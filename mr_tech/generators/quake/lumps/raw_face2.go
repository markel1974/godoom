package lumps

import "github.com/markel1974/godoom/mr_tech/geometry"

// RawFace2 represents a 3D polygonal face with geometry points, UV mapping, texture name, and type classification.
// Points holds the vertices of the face as an array of 3D coordinates.
// UVs stores the UV mapping coordinates for texture placement.
// TexName specifies the name of the texture applied to the face.
// Kind denotes the type or category of the face as an integer.
type RawFace2 struct {
	Points  []geometry.XYZ
	UVs     [][2]float64
	TexName string
	Flags   uint32
}

// NewRawFace2 creates and returns a new RawFace2 instance with specified points, UV coordinates, texture name, and kind.
func NewRawFace2(pts []geometry.XYZ, uvs [][2]float64, tex string, flags uint32) *RawFace2 {
	return &RawFace2{Points: pts, UVs: uvs, TexName: tex, Flags: flags}
}

// Light checks if the surface has the SURF_LIGHT flag.
func (t RawFace2) Light() bool {
	return (t.Flags & surfLight) != 0
}

// Slick checks if the surface has the SURF_SLICK flag.
func (t RawFace2) Slick() bool {
	return (t.Flags & surfSlick) != 0
}

// IsSky determines whether the texture has the SURF_SKY flag set.
func (t RawFace2) IsSky() bool {
	return (t.Flags & surfSky) != 0
}

// IsWarp determines whether the surface has the SURF_WARP flag set.
//
// Quake II uses SURF_WARP for surfaces such as water, slime and lava.
// This identifies a warped/liquid-style surface, but does not by itself
// distinguish the liquid type.
func (t RawFace2) IsWarp() bool {
	return (t.Flags & surfWarp) != 0
}

// IsTrans33 determines whether the surface has 33% transparency.
func (t RawFace2) IsTrans33() bool {
	return (t.Flags & surfTrans33) != 0
}

// IsTrans66 determines whether the surface has 66% transparency.
func (t RawFace2) IsTrans66() bool {
	return (t.Flags & surfTrans66) != 0
}

// IsFlowing determines whether the texture is marked as flowing.
func (t RawFace2) IsFlowing() bool {
	return (t.Flags & surfFlowing) != 0
}

// NoDraw checks if the texture is marked with SURF_NODRAW.
func (t RawFace2) NoDraw() bool {
	return (t.Flags & surfNoDraw) != 0
}

// IsHint determines whether the surface is a BSP hint surface.
func (t RawFace2) IsHint() bool {
	return (t.Flags & surfHint) != 0
}

// IsSkip determines whether the surface has the SURF_SKIP flag.
func (t TexInfo2) IsSkip() bool {
	return (t.Flags & surfSkip) != 0
}

// NoLight determines whether the surface has the SURF_NOLIGHT flag.
func (t RawFace2) NoLight() bool {
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
