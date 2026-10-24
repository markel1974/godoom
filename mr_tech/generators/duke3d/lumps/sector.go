package lumps

// StatParallaxing represents a surface that displays a sky or parallaxing background.
// StatSloped indicates that the surface is sloped.
// StatSwapXY specifies swapping of texture X and Y coordinates.
// StatDoubleSmooshiness doubles the vertical smooshiness or scaling factor of the surface.
// StatXFlip enables horizontal flipping of the texture.
// StatYFlip enables vertical flipping of the texture.
// StatAlignFirstWall aligns the texture to the first wall.
const (
	StatParallaxing       = 1 << 0 // Bit 0: Surface displays sky / parallaxing background
	StatSloped            = 1 << 1 // Bit 1: Surface is sloped
	StatSwapXY            = 1 << 2 // Bit 2: Swap texture X and Y coordinates
	StatDoubleSmooshiness = 1 << 3 // Bit 3: Double vertical smooshiness / scaling factor
	StatXFlip             = 1 << 4 // Bit 4: Flip texture horizontally
	StatYFlip             = 1 << 5 // Bit 5: Flip texture vertically
	StatAlignFirstWall    = 1 << 6 // Bit 6: Align texture to the first wall
)

// Sector represents a map sector with properties for walls, floor, ceiling, and other sector-specific attributes.
type Sector struct {
	WallPtr         int16
	WallNum         int16
	CeilingZ        int32
	FloorZ          int32
	CeilingStat     int16
	FloorStat       int16
	CeilingPicNum   int16
	CeilingHeiNum   int16
	CeilingShade    int8
	CeilingPal      uint8
	CeilingXPanning uint8
	CeilingYPanning uint8
	FloorPicNum     int16
	FloorHeiNum     int16
	FloorShade      int8
	FloorPal        uint8
	FloorXPanning   uint8
	FloorYPanning   uint8
	Visibility      uint8
	Filler          uint8
	LoTag           int16
	HiTag           int16
	Extra           int16
}

// IsSlopedFloor checks if the sector's floor is sloped based on its FloorStat and FloorHeiNum properties.
func (s *Sector) IsSlopedFloor() bool {
	return s.FloorStat&StatSloped != 0 && s.FloorHeiNum != 0
}

// IsSlopedCeiling checks if the sector's ceiling is sloped by evaluating its ceiling status and height properties.
func (s *Sector) IsSlopedCeiling() bool {
	return s.CeilingStat&StatSloped != 0 && s.CeilingHeiNum != 0
}

// IsSkyCeiling returns true if the ceiling has a sky/parallaxing texture, indicated by the StatParallaxing flag.
func (s *Sector) IsSkyCeiling() bool {
	return s.CeilingStat&StatParallaxing != 0
}

// IsSkyFloor checks if the floor of the sector is parallaxed, indicating a sky-like floor rendering.
func (s *Sector) IsSkyFloor() bool {
	return s.FloorStat&StatParallaxing != 0
}

// SwapXYFloor checks if the floor texture is swapped along the X and Y axes based on the floor's status flags.
func (s *Sector) SwapXYFloor() bool {
	return s.FloorStat&StatSwapXY != 0
}

// SwapXYCeiling checks if the X and Y axes of the ceiling are swapped by evaluating the StatSwapXY flag in CeilingStat.
func (s *Sector) SwapXYCeiling() bool {
	return s.CeilingStat&StatSwapXY != 0
}

// DoubleSmooshinessFloor checks if the floor has the double smooshiness flag set in the FloorStat field.
func (s *Sector) DoubleSmooshinessFloor() bool {
	return s.FloorStat&StatDoubleSmooshiness != 0
}

// DoubleSmooshinessCeiling checks if the ceiling of the sector has the "double smooshiness" property enabled.
func (s *Sector) DoubleSmooshinessCeiling() bool {
	return s.CeilingStat&StatDoubleSmooshiness != 0
}

// XFlipFloor checks if the floor has the X-Flip flag set in its status. Returns true if the flag is set, otherwise false.
func (s *Sector) XFlipFloor() bool {
	return s.FloorStat&StatXFlip != 0
}

// XFlipCeiling checks if the ceiling texture is horizontally flipped based on its CeilingStat flag.
func (s *Sector) XFlipCeiling() bool {
	return s.CeilingStat&StatXFlip != 0
}

// YFlipFloor checks if the Y-axis of the floor texture is flipped by inspecting the FloorStat field.
func (s *Sector) YFlipFloor() bool {
	return s.FloorStat&StatYFlip != 0
}

// YFlipCeiling checks if the Y-flip flag is set for the ceiling of the sector.
func (s *Sector) YFlipCeiling() bool {
	return s.CeilingStat&StatYFlip != 0
}

// AlignFirstWallFloor checks if the floor alignment flag with the first wall is set in the sector's FloorStat field.
func (s *Sector) AlignFirstWallFloor() bool {
	return s.FloorStat&StatAlignFirstWall != 0
}

// AlignFirstWallCeiling checks if the ceiling texture aligns with the first wall based on the ceiling stat flags.
func (s *Sector) AlignFirstWallCeiling() bool {
	return s.CeilingStat&StatAlignFirstWall != 0
}
