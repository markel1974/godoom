package lumps

// SpriteType represents the type/category of a sprite based on specific configuration bits in its CStat property.
type SpriteType int16

// Build Engine Sprite CStat bitmask constants
const (
	CStatBlocking         = 1 << 0 // Bit 0: Block players/monsters/shots
	CStatTransparent      = 1 << 1 // Bit 1: Translucent sprite rendering (transparency)
	CStatXFlip            = 1 << 2 // Bit 2: Flip sprite horizontally
	CStatYFlip            = 1 << 3 // Bit 3: Flip sprite vertically
	CStatAlignmentMask    = 3 << 4 // Bits 4-5: Sprite alignment type mask
	CStatOneSided         = 1 << 6 // Bit 6: One-sided sprite (invisible from back)
	CStatCenterVertically = 1 << 7 // Bit 7: Center vertically around Z coordinate
	CStatHitscan          = 1 << 8 // Bit 8: Hitscan blocking
	CStatTransAlt         = 1 << 9 // Bit 9: Alternative translucency (33% / 66%)
	CStatInvisible        = 1 << 15
)

// Sprite alignment types defined by bits 4 and 5 (values 0, 16, 32).
const (
	FaceSprite  = SpriteType(0 << 4) // Faces the player/camera (billboard)
	WallSprite  = SpriteType(1 << 4) // Stands vertically along ang
	FloorSprite = SpriteType(2 << 4) // Flat on floor/ceiling
)

// Sprite represents an entity in the game world with position, appearance, motion, and tagging attributes.
type Sprite struct {
	X        int32
	Y        int32
	Z        int32
	CStat    int16
	PicNum   int16
	Shade    int8
	Pal      uint8
	ClipDist uint8
	Filler   uint8
	XRepeat  uint8
	YRepeat  uint8
	XOffset  int8
	YOffset  int8
	SectNum  int16
	StatNum  int16
	Ang      int16
	Owner    int16
	XVel     int16
	YVel     int16
	ZVel     int16
	LoTag    int16
	HiTag    int16
	Extra    int16
}

// IsBlocking determines if the sprite is blocking other entities.
func (s *Sprite) IsBlocking() bool {
	return s.CStat&CStatBlocking != 0
}

// IsTransparent checks if the sprite has translucent rendering enabled.
func (s *Sprite) IsTransparent() bool {
	return s.CStat&CStatTransparent != 0
}

// HasHorizontalFlip determines if the sprite's horizontal flip bit is set.
func (s *Sprite) HasHorizontalFlip() bool {
	return s.CStat&CStatXFlip != 0
}

// HasVerticalFlip determines if the sprite is vertically flipped.
func (s *Sprite) HasVerticalFlip() bool {
	return s.CStat&CStatYFlip != 0
}

// IsTwoSided checks if the sprite is two-sided (not one-sided).
func (s *Sprite) IsTwoSided() bool {
	return s.CStat&CStatOneSided == 0
}

// GetType returns the structural alignment type of the sprite.
func (s *Sprite) GetType() SpriteType {
	return SpriteType(s.CStat & CStatAlignmentMask)
}

// HasVerticalAlignment checks if the sprite is vertically centered around its Z coordinate.
func (s *Sprite) HasVerticalAlignment() bool {
	return s.CStat&CStatCenterVertically != 0
}

// IsInvisible checks if the sprite is completely invisible (bit 15).
func (s *Sprite) IsInvisible() bool {
	return uint16(s.CStat)&CStatInvisible != 0
}

// IsInvisibleHitscan checks if the sprite is invisible to hitscans / blockable hits.
func (s *Sprite) IsInvisibleHitscan() bool {
	return s.CStat&CStatHitscan != 0
}
