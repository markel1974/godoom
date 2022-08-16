package model

type XY struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type XYZ struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

type InputSegment struct {
	Start    XY     `json:"start"`
	End      XY     `json:"end"`
	Kind     int    `json:"kind"`
	Neighbor string `json:"neighbor"`
	Tag      string `json:"tag"`
	Upper    string `json:"upper"`
	Middle   string `json:"middle"`
	Lower    string `json:"lower"`
}

type InputSector struct {
	Id           string           `json:"id"`
	Ceil         float64          `json:"ceil"`
	Floor        float64          `json:"floor"`
	Textures     bool             `json:"textures"`
	FloorTexture string           `json:"floorTexture"`
	CeilTexture  string           `json:"ceilTexture"`
	UpperTexture string           `json:"upperTexture"`
	LowerTexture string           `json:"lowerTexture"`
	WallTexture  string           `json:"wallTexture"`
	Segments     []*InputSegment  `json:"segments"`
	Tag          string           `json:"tag"`
}

type InputLight struct {
	Where  XYZ    `json:"where"`
	Light  XYZ    `json:"light"`
	Sector string `json:"sector"`
}

type InputPlayer struct {
	Position XY      `json:"position"`
	Angle    float64 `json:"angle"`
	Sector   string  `json:"sector"`
}

type Input struct {
	Sectors           []*InputSector `json:"sectors"`
	Lights            []*InputLight  `json:"lights"`
	Player            *InputPlayer   `json:"player"`
	ScaleFactor       float64        `json:"scaleFactor"`
	DisableFix        bool           `json:"fix"`
}
