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
	Parent   string      `json:"id"`
	Id       string      `json:"id"`
	Start    XY          `json:"start"`
	End      XY          `json:"end"`
	Kind     int         `json:"kind"`
	Neighbor string      `json:"neighbor"`
	Tag      string      `json:"tag"`
	Upper    string      `json:"upper"`
	Middle   string      `json:"middle"`
	Lower    string      `json:"lower"`
}



func NewInputSegment(parent string, kind int, s XY, e XY) * InputSegment {
	is := &InputSegment{
		Parent:   parent,
		Id:       NextUUId(),
		Start:    s,
		End:      e,
		Kind:     kind,
		Neighbor: "",
		Tag:      "",
		Upper:    "",
		Middle:   "",
		Lower:    "",
	}
	return is
}


func (is * InputSegment) Clone() * InputSegment {
	out := NewInputSegment(is.Parent, is.Kind, is.Start, is.End)
	out.Neighbor = is.Neighbor
	out.Tag = is.Tag
	out.Upper = is.Upper
	out.Middle = is.Middle
	out.Lower = is.Lower
	return out
}

func (is * InputSegment) EqualCoords(tst * InputSegment) bool {
	ret := is.Start.X == tst.Start.X && is.Start.Y == tst.Start.Y && is.End.X == tst.End.X && is.End.Y == tst.End.Y
	return ret
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
	DisableLoop       bool           `json:"disableLoop"`
}
