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

// IsMasked determines if the wall has a masked or grating texture based on its CStat bitmask.
func (w *Wall) IsMasked() bool {
	return w.CStat&16 != 0
}

// IsOneWay determines if the wall is a one-way wall based on its CStat property. Returns true if the condition is met.
func (w *Wall) IsOneWay() bool {
	return w.CStat&32 != 0
}

// IsTranslucent checks if the wall has a translucent property by evaluating the corresponding CStat flag.
func (w *Wall) IsTranslucent() bool {
	return w.CStat&128 != 0
}
