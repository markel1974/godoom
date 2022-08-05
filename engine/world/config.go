package world

import (
	"encoding/json"
	"github.com/markel1974/godoom/engine/config"
	rnd "math/rand"
)

func random(min int, max int) int {
	return rnd.Intn(max-min+1) + min
}

func randomF(min float64, max float64) float64 {
	return min + rnd.Float64()*(max-min)
}

func ParseJsonData(source []byte) (*config.Config, error) {
	cfg := &config.Config{}
	if err := json.Unmarshal(source, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

var availableCeil = []string{"ceil.ppm", "ceil2.ppm", "ceil2_norm.ppm"}
var availableFloor = []string{"floor.ppm"}
var availableUpper = []string{"wall2.ppm"}
var availableLower = []string{"wall2.ppm"}
var availableWall = []string{"wall2.ppm"}

func createCube(x float64, y float64, max float64, floor float64, ceil float64) *config.Sector {
	sector := &config.Sector{Id: NextUUId(), Floor: floor, Ceil: ceil}
	sector.Textures = true
	sector.FloorTexture = availableFloor[random(0, len(availableFloor)-1)]
	sector.CeilTexture = availableCeil[random(0, len(availableCeil)-1)]
	sector.UpperTexture = availableUpper[random(0, len(availableUpper)-1)]
	sector.LowerTexture = availableLower[random(0, len(availableLower)-1)]
	sector.WallTexture = availableWall[random(0, len(availableWall)-1)]
	for c := 0; c < 4; c++ {
		xy := config.XY{X: 0, Y: 0}
		switch c {
		case 0: xy.X = x; xy.Y = y
		case 1:	xy.X = x + max; xy.Y = y
		case 2:	xy.X = x + max;	xy.Y = y + max
		case 3:	xy.X = x; xy.Y = y + max
		}
		neighbor := &config.Neighbor{XY: xy, Id: "wall"}
		sector.Neighbors = append(sector.Neighbors, neighbor)
	}
	return sector
}

func Generate(maxX int, maxY int) (*config.Config, error) {
	cfg := &config.Config{Sectors: nil, Player: &config.Player{}}
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
