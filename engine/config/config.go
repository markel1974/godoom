package config

type XY struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type XYZ struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

type Neighbor struct {
	XY
	Id string `json:"id"`
}

type Sector struct {
	Id           string      `json:"id"`
	Ceil         float64     `json:"ceil"`
	Floor        float64     `json:"floor"`
	Textures     bool        `json:"textures"`
	FloorTexture string      `json:"floorTexture"`
	CeilTexture  string      `json:"ceilTexture"`
	UpperTexture string      `json:"upperTexture"`
	LowerTexture string      `json:"lowerTexture"`
	WallTexture  string      `json:"wallTexture"`
	Neighbors    []*Neighbor `json:"neighbors"`
}

type Light struct {
	Where  XYZ `json:"where"`
	Light  XYZ `json:"light"`
	Sector string    `json:"sector"`
}

type Player struct {
	Position XY `json:"position"`
	Angle    float64  `json:"angle"`
	Sector   string   `json:"sector"`
}

type Config struct {
	Sectors []*Sector `json:"sectors"`
	Lights  []*Light  `json:"lights"`
	Player  *Player   `json:"player"`
	Compile bool      `json:"player"`
}
