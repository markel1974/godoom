package lumps

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

func (s *Sprite) IsTwoSided() bool {
	const twoSidedBit = 64 // sp.CStat & 64 == 0
	return s.CStat&twoSidedBit == 0
}

func (s *Sprite) IsCenteredVertically() bool {
	const centeredVerticallyBit = 128
	return s.CStat&centeredVerticallyBit != 0
}

type SpriteType int16

const (
	WallAlignedSprite  = SpriteType(16) // stands vertically along ang
	FloorAlignedSprite = SpriteType(32) // flat on floor/ceiling
)

func (s *Sprite) Type() SpriteType {
	return SpriteType(s.CStat & 48)
}
