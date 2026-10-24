package lumps

// Wall represents a wall structure with spatial coordinates, rendering attributes, and connectivity to other walls or sectors.
type Wall struct {
	X          int32
	Y          int32
	Point2     int16
	NextWall   int16
	NextSector int16
	CStat      int16
	PicNum     int16
	OverPicNum int16
	Shade      int8
	Pal        uint8
	XRepeat    uint8
	YRepeat    uint8
	XPanning   uint8
	YPanning   uint8
	LoTag      int16
	HiTag      int16
	Extra      int16
}

// Build Engine Wall CStat bitmask constants
const (
	CStatWallBlocking      = 1 << 0 // Bit 0: Block players/monsters
	CStatWallBottomAligned = 1 << 1 // Bit 1: Bottom aligned texture
	CStatWallRedMarker     = 1 << 2 // Bit 2: Red marker (EDF/Map editor)
	CStatWallPegTop        = 1 << 3 // Bit 3: Don't peg texture to floor
	CStatWallMasked        = 1 << 4 // Bit 4: Masked / grating wall texture
	CStatWallOneWay        = 1 << 5 // Bit 5: One-way wall (blocks only from front side)
	CStatWallXFlip         = 1 << 6 // Bit 6: Flip wall texture horizontally
	CStatWallTranslucent   = 1 << 7 // Bit 7: Translucent wall rendering
	CStatWallYFlip         = 1 << 8 // Bit 8: Flip wall texture vertically
	CStatWallAlignBottom   = 1 << 9 // Bit 9: Align texture to bottom
)

// IsBlocking checks if the wall blocks players and monsters.
func (w *Wall) IsBlocking() bool {
	return w.CStat&CStatWallBlocking != 0
}

// IsBottomAligned checks if the wall texture is bottom-aligned.
func (w *Wall) IsBottomAligned() bool {
	return w.CStat&CStatWallBottomAligned != 0
}

// IsRedMarker checks if the red marker flag is set on the wall.
func (w *Wall) IsRedMarker() bool {
	return w.CStat&CStatWallRedMarker != 0
}

// IsPegTop checks if the wall texture is pegged to the top.
func (w *Wall) IsPegTop() bool {
	return w.CStat&CStatWallPegTop != 0
}

// IsMasked determines if the wall has a masked or grating texture.
func (w *Wall) IsMasked() bool {
	return w.CStat&CStatWallMasked != 0
}

// IsOneWay determines if the wall is a one-way wall.
func (w *Wall) IsOneWay() bool {
	return w.CStat&CStatWallOneWay != 0
}

// HasHorizontalFlip checks if the wall texture is flipped horizontally.
func (w *Wall) HasHorizontalFlip() bool {
	return w.CStat&CStatWallXFlip != 0
}

// IsTranslucent checks if the wall has translucent rendering enabled.
func (w *Wall) IsTranslucent() bool {
	return w.CStat&CStatWallTranslucent != 0
}

// HasVerticalFlip checks if the wall texture is flipped vertically.
func (w *Wall) HasVerticalFlip() bool {
	return w.CStat&CStatWallYFlip != 0
}

// IsAlignBottom checks if the wall texture alignment is set to bottom.
func (w *Wall) IsAlignBottom() bool {
	return w.CStat&CStatWallAlignBottom != 0
}
