package main

import "encoding/json"

type ConfigXY struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type ConfigXYZ struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

type ConfigNeighbor struct {
	ConfigXY
	Id string `json:"id"`
}

type ConfigSector struct {
	Id           string            `json:"id"`
	Ceil         float64           `json:"ceil"`
	Floor        float64           `json:"floor"`
	Textures     bool              `json:"textures"`
	FloorTexture string            `json:"floorTexture"`
	CeilTexture  string            `json:"ceilTexture"`
	UpperTexture string            `json:"upperTexture"`
	LowerTexture string            `json:"lowerTexture"`
	WallTexture  string            `json:"wallTexture"`
	Neighbors    []*ConfigNeighbor `json:"neighbors"`
}

type ConfigLight struct {
	Where  ConfigXYZ `json:"where"`
	Light  ConfigXYZ `json:"light"`
	Sector string    `json:"sector"`
}

type ConfigPlayer struct {
	Position ConfigXY `json:"position"`
	Angle    float64  `json:"angle"`
	Sector   string   `json:"sector"`
}

type Config struct {
	Sectors []*ConfigSector `json:"sectors"`
	Lights  []*ConfigLight  `json:"lights"`
	Player  *ConfigPlayer   `json:"player"`
}

func ParseJsonData(source []byte) (*Config, error) {
	cfg := &Config{}
	if err := json.Unmarshal(source, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

var availableCeil = []string{"ceil.ppm", "ceil2.ppm", "ceil2_norm.ppm"}

//var availableFloor = []string{"floor.ppm", "ceil.ppm", "ceil2.ppm"}
var availableFloor = []string{"floor.ppm"}
var availableUpper = []string{"wall2.ppm"}
var availableLower = []string{"wall2.ppm"}
var availableWall = []string{"wall2.ppm"}

func createCube(x float64, y float64, max float64, floor float64, ceil float64) *ConfigSector {
	sector := &ConfigSector{Id: NextUUId(), Floor: floor, Ceil: ceil}
	sector.Textures = true
	sector.FloorTexture = availableFloor[random(0, len(availableFloor)-1)]
	sector.CeilTexture = availableCeil[random(0, len(availableCeil)-1)]
	sector.UpperTexture = availableUpper[random(0, len(availableUpper)-1)]
	sector.LowerTexture = availableLower[random(0, len(availableLower)-1)]
	sector.WallTexture = availableWall[random(0, len(availableWall)-1)]
	for c := 0; c < 4; c++ {
		xy := ConfigXY{X: 0, Y: 0}
		switch c {
		case 0:
			xy.X = x
			xy.Y = y
		case 1:
			xy.X = x + max
			xy.Y = y
		case 2:
			xy.X = x + max
			xy.Y = y + max
		case 3:
			xy.X = x
			xy.Y = y + max
		}
		neighbor := &ConfigNeighbor{ConfigXY: xy, Id: "wall"}
		sector.Neighbors = append(sector.Neighbors, neighbor)
	}
	return sector
}

/*
func randomData() (*Config, error) {
	cfg := & Config{ Sectors:  nil, player: &ConfigPlayer{} }
	s1 := createCube(0, 0, 8, 0, 20)
	s1.Id = "root"
	cfg.Sectors = append(cfg.Sectors, s1)

	s2 := createCube(8, 0, 8, 0, 20)
	s1.Id = "toor"
	cfg.Sectors = append(cfg.Sectors, s2)
	for x := 1; x < 128; x ++ {
		ceil := randomF(15, 30)
		floor := randomF(0, 2)
		s3 := createCube(0, 8 * float64(x), 8, floor, ceil)
		cfg.Sectors = append(cfg.Sectors, s3)
	}
	cfg.player.Position.X = 1
	cfg.player.Position.Y = 1
	cfg.player.Sector = s1.Id

	return cfg, nil
}

*/

func GenerateWorld(maxX int, maxY int) (*Config, error) {
	cfg := &Config{Sectors: nil, Player: &ConfigPlayer{}}
	s1 := createCube(0, 0, 8, 0, 20)
	s1.Id = "root"
	cfg.Sectors = append(cfg.Sectors, s1)

	s2 := createCube(8, 0, 8, 0, 20)
	s1.Id = "toor"
	cfg.Sectors = append(cfg.Sectors, s2)

	for x := 1; x < maxX; x++ {
		for y := 1; y < maxY; y++ {
			create := random(0, 5)
			if x == 1 || y == 1 || create > 2 {
				ceil := randomF(15, 30)
				floor := randomF(0, 2)
				s3 := createCube(float64(x)*8, 8*float64(y), 8, floor, ceil)
				cfg.Sectors = append(cfg.Sectors, s3)
			}
		}
	}

	cfg.Player.Position.X = 1
	cfg.Player.Position.Y = 1
	cfg.Player.Sector = s1.Id

	return cfg, nil
}
