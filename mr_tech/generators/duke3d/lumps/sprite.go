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
