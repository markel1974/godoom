package world

import (
	"encoding/json"
	rnd "math/rand"

	"github.com/markel1974/godoom/engine/model"
)

// random generates a random integer between the specified min (inclusive) and max (inclusive) values.
func random(min int, max int) int {
	return rnd.Intn(max-min+1) + min
}

// randomF generates a random float64 value between the specified min and max bounds using a uniform distribution.
func randomF(min float64, max float64) float64 {
	return min + rnd.Float64()*(max-min)
}

// ParseJsonData parses a JSON-encoded byte array into a ConfigRoot struct and returns it or an error on failure.
func ParseJsonData(source []byte) (*model.ConfigRoot, error) {
	cfg := &model.ConfigRoot{}
	if err := json.Unmarshal(source, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// availableCeil contains a list of available ceiling texture file names in PPM format.
var availableCeil = []string{"ceil.ppm", "ceil2.ppm", "ceil2_norm.ppm"}

// availableFloor represents a list of texture filenames that can be used as floor textures in the game environment.
var availableFloor = []string{"floor.ppm"}

// availableUpper contains a list of file names for textures used as the upper surface in sectors.
var availableUpper = []string{"wall2.ppm"}

// availableLower stores a list of texture file names used for lower wall surfaces in the level configuration.
var availableLower = []string{"wall2.ppm"}

// availableWall holds a list of wall texture file names available for sector configurations.
var availableWall = []string{"wall2.ppm"}

// createCube initializes and returns a ConfigSector representing a cubical sector in a level with specified properties.
func createCube(x float64, y float64, max float64, floor float64, ceil float64) *model.ConfigSector {
	sector := &model.ConfigSector{Id: model.NextUUId(), Floor: floor, Ceil: ceil}
	sector.Textures = true
	sector.TextureFloor = availableFloor[random(0, len(availableFloor)-1)]
	sector.TextureCeil = availableCeil[random(0, len(availableCeil)-1)]
	sector.TextureUpper = availableUpper[random(0, len(availableUpper)-1)]
	sector.TextureLower = availableLower[random(0, len(availableLower)-1)]
	sector.TextureWall = availableWall[random(0, len(availableWall)-1)]
	sector.TextureScaleFactor = 50.0
	const SegmentMax = 4
	for c := 0; c < SegmentMax; c++ {
		xy := model.XY{X: 0, Y: 0}
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

		if c == 0 {
			neighbor := &model.ConfigSegment{Start: xy, End: xy, Neighbor: "unknown", Kind: model.DefinitionUnknown}
			sector.Segments = append(sector.Segments, neighbor)
		} else if c == SegmentMax-1 {
			prev := sector.Segments[c-1]
			prev.End = xy
		} else {
			prev := sector.Segments[c-1]
			prev.End = xy
			neighbor := &model.ConfigSegment{Start: xy, End: xy, Neighbor: "unknown", Kind: model.DefinitionUnknown}
			sector.Segments = append(sector.Segments, neighbor)
		}
	}
	return sector
}

// Generate creates a new game configuration with sectors, a player, and randomized structures based on grid dimensions.
func Generate(maxX int, maxY int) (*model.ConfigRoot, error) {
	configPlayer := &model.ConfigPlayer{}
	cfg := model.NewConfigRoot(nil, configPlayer, nil, 0, false)
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
