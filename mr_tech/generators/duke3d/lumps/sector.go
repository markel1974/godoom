package lumps

// Sector represents a section of a game level, including spatial, visual, and connectivity attributes.
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

// IsSlopedFloor checks if the floor of the sector is sloped based on its FloorStat and FloorHeiNum properties.
func (s *Sector) IsSlopedFloor() bool {
	return s.FloorStat&2 != 0 && s.FloorHeiNum != 0
}

// IsSlopedCeiling checks if the ceiling of the sector is sloped based on its status and height properties.
func (s *Sector) IsSlopedCeiling() bool {
	return s.CeilingStat&2 != 0 && s.CeilingHeiNum != 0
}

// IsSkyCeiling checks if the sector's ceiling has the "sky" property based on its CeilingStat bitmask.
func (s *Sector) IsSkyCeiling() bool {
	return s.CeilingStat&1 != 0
}

// IsSkyFloor checks if the sector's floor is marked as a sky surface based on its FloorStat property.
func (s *Sector) IsSkyFloor() bool {
	return s.FloorStat&1 != 0
}
