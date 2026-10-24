package lumps

// SpriteType represents the type/category of a sprite based on specific configuration bits in its CStat property.
type SpriteType int16

// FaceSprite represents a sprite with a face-on orientation.
// WallAlignedSprite represents a sprite that stands vertically, aligned along an angle.
// FloorAlignedSprite represents a sprite aligned flat on the floor or ceiling.
const (
	FaceSprite  = SpriteType(0)
	WallSprite  = SpriteType(16) // stands vertically along ang
	FloorSprite = SpriteType(32) // flat on floor/ceiling
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

// IsBlocking determines if the sprite is blocking other entities based on its CStat field. Returns true if blocking is enabled.
func (s *Sprite) IsBlocking() bool {
	const blockingBit = 1
	return s.CStat&blockingBit != 0
}

// IsTransparent checks if the sprite is transparent by evaluating the transparent bit in its CStat field. Returns true if transparent.
func (s *Sprite) IsTransparent() bool {
	const transparentBit = 2
	return s.CStat&transparentBit != 0
}

// HasHorizontalFlip determines if the sprite's horizontal flip bit is set in its CStat field. Returns true if flipped, otherwise false.
func (s *Sprite) HasHorizontalFlip() bool {
	const hFlipBit = 4
	return s.CStat&hFlipBit != 0
}

// HasVerticalFlip determines if the sprite is vertically flipped based on its CStat property. Returns true if flipped.
func (s *Sprite) HasVerticalFlip() bool {
	const vFlipBit = 8
	return s.CStat&vFlipBit != 0
}

// IsTwoSided checks if the sprite is two-sided (CStat & 64 == 0). Returns true if two-sided, false otherwise.
func (s *Sprite) IsTwoSided() bool {
	const twoSidedBit = 64 // sp.CStat & 64 == 0
	return s.CStat&twoSidedBit == 0
}

// GetType returns the type of the sprite based on specific bits in the CStat field.
func (s *Sprite) GetType() SpriteType {
	const typeMask = int16(FaceSprite) | int16(WallSprite) | int16(FloorSprite)
	return SpriteType(s.CStat & typeMask)
}

// HasVerticalAlignment checks if the sprite is vertically aligned by evaluating the centered vertically bit in CStat. Returns true if aligned.
func (s *Sprite) HasVerticalAlignment() bool {
	const verticalAlignmentBit = 128
	return s.CStat&verticalAlignmentBit != 0
}
