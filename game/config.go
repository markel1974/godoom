package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

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

/*
func ParseJsonData() (* Config, error) {
	data := []byte(stub)
	cfg := &Config{}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
*/

var availableCeil = []string{"ceil.ppm", "ceil2.ppm", "ceil2_norm.ppm"}

//var availableFloor = []string{"floor.ppm", "ceil.ppm", "ceil2.ppm"}
var availableFloor = []string{"floor.ppm"}
var availableUpper = []string{"wall2.ppm"}
var availableLower = []string{"wall2.ppm"}
var availableWall = []string{"wall2.ppm"}

func createCube(x float64, y float64, max float64, floor float64, ceil float64) *ConfigSector {
	sector := &ConfigSector{Id: NextUUId(), Floor: floor, Ceil: ceil}

	//textures := random(0, 1)
	textures := 1

	if textures == 1 {
		sector.Textures = true
	}
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

func randomData() (*Config, error) {
	cfg := &Config{Sectors: nil, Player: &ConfigPlayer{}}
	s1 := createCube(0, 0, 8, 0, 20)
	s1.Id = "root"
	cfg.Sectors = append(cfg.Sectors, s1)

	s2 := createCube(8, 0, 8, 0, 20)
	s1.Id = "toor"
	cfg.Sectors = append(cfg.Sectors, s2)

	for x := 1; x < 16; x++ {
		for y := 1; y < 16; y++ {
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

func ParseOldData(id string) (*Config, error) {
	var cfgVertices []XY
	cfg := &Config{
		Sectors: nil,
		Player:  &ConfigPlayer{},
	}

	oldData := strings.Split(id, "\n")

	configSectorIdx := 0

	for _, data := range oldData {
		var verb string
		var word string
		r := strings.NewReader(data)
		if _, err := fmt.Fscanf(r, "%s", &verb); err != nil {
			continue
		}
		switch verb {
		case "#":
			continue

		case "vertex":
			for {
				var vertexY float64
				if _, err := fmt.Fscanf(r, "%f", &vertexY); err != nil {
					if err != io.EOF {
						return nil, err
					}
					break
				}
				for {
					xy := XY{Y: vertexY}
					if _, err := fmt.Fscanf(r, "%f", &xy.X); err != nil {
						if err != io.EOF {
							return nil, err
						}
						break
					}
					//fmt.Printf("idx: %d, x: %f, y: %f\n", len(cfgVertices), xy.X, xy.Y)
					cfgVertices = append(cfgVertices, xy)
				}
			}

		case "sector":
			if cfgVertices == nil {
				return nil, errors.New(fmt.Sprintf("nil vertices"))
			}
			cs := &ConfigSector{}
			cs.Id = strconv.Itoa(configSectorIdx)
			configSectorIdx++
			_, err := fmt.Fscanf(r, "%f%f", &cs.Floor, &cs.Ceil)
			if err != nil {
				return nil, err
			}
			var numbers []int
			for {
				if _, err := fmt.Fscanf(r, "%32s", &word); err != nil {
					break
				}
				var val int
				if word[0] != 'x' {
					val, _ = strconv.Atoi(word)
				} else {
					val = -1
				}
				numbers = append(numbers, val)
			}
			if len(numbers) == 0 || len(numbers)%2 > 0 {
				return nil, errors.New("empty sector number")
			}
			//numbers viene diviso a metà perche la prima parte contiene i riferimenti ai vertici e la seconda i Neighbors
			m := len(numbers) / 2
			for idx := 0; idx < m; idx++ {
				vertexId := numbers[idx]
				neighborId := numbers[idx+m]
				if vertexId < 0 || vertexId >= len(cfgVertices) {
					return nil, errors.New(fmt.Sprintf("invalid vertex number: %d max: %d", vertexId, len(cfgVertices)))
				}
				neighbor := &ConfigNeighbor{
					ConfigXY: ConfigXY{X: cfgVertices[vertexId].X, Y: cfgVertices[vertexId].Y},
					Id:       strconv.Itoa(neighborId),
				}
				cs.Neighbors = append(cs.Neighbors, neighbor)
				cs.WallTexture = "wall2.ppm"
				cs.LowerTexture = "wall.ppm"
				cs.UpperTexture = "wall3.ppm"
				cs.FloorTexture = "floor.ppm"
				cs.CeilTexture = "ceil.ppm"
				cs.Textures = true
			}
			cfg.Sectors = append(cfg.Sectors, cs)

		case "light":
			l := &ConfigLight{}
			_, _ = fmt.Fscanf(r, "%f %f %f %s %f %f %f", &l.Where.X, &l.Where.Z, &l.Where.Y, &l.Sector, &l.Light.X, &l.Light.Y, &l.Light.Z)
			cfg.Lights = append(cfg.Lights, l)

		case "player":
			_, _ = fmt.Fscanf(r, "%f %f %f %s", &cfg.Player.Position.X, &cfg.Player.Position.Y, &cfg.Player.Angle, &cfg.Player.Sector)

		default:
			continue
		}
	}

	if cfgVertices == nil {
		return nil, errors.New(fmt.Sprintf("nil vertices"))
	}

	out, _ := json.MarshalIndent(cfg, "", " ")
	fmt.Println(string(out))

	/*
		var ranges  = []int{ 3, 14, 27, 45 }
		for _, r := range ranges {
			out, _ := json.MarshalIndent(cfg.Vertices[r], "", " ")
			fmt.Println(string(out))
		}
	*/

	return cfg, nil
}
