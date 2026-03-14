package world

import (
	"encoding/json"
	"fmt"
	"math"
	rnd "math/rand"
	"os"

	"github.com/markel1974/godoom/engine/model"
	"github.com/markel1974/godoom/engine/utils"
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
	sector := model.NewConfigSector(utils.NextUUId())
	sector.FloorY = floor
	sector.CeilY = ceil

	sector.Animations.Floor = model.NewConfigAnimation([]string{availableFloor[random(0, len(availableFloor)-1)]}, model.AnimationKindLoop)
	sector.Animations.Ceil = model.NewConfigAnimation([]string{availableCeil[random(0, len(availableCeil)-1)]}, model.AnimationKindLoop)
	sector.Animations.ScaleFactor = 50.0

	sector.Light.Intensity = rnd.Float64()
	sector.Light.Kind = model.LightKindSpot

	pts := [4]model.XY{
		{X: x, Y: y},
		{X: x + max, Y: y},
		{X: x + max, Y: y + max},
		{X: x, Y: y + max},
	}

	for i := 0; i < 4; i++ {
		start := pts[i]
		end := pts[(i+1)%4]

		// Allocazione corretta tramite costruttore
		seg := model.NewConfigSegment("", model.DefinitionUnknown, start, end)
		seg.Neighbor = "unknown"

		seg.Animations.Upper = model.NewConfigAnimation([]string{availableUpper[random(0, len(availableUpper)-1)]}, model.AnimationKindLoop)
		seg.Animations.Lower = model.NewConfigAnimation([]string{availableLower[random(0, len(availableLower)-1)]}, model.AnimationKindLoop)
		seg.Animations.Middle = model.NewConfigAnimation([]string{availableWall[random(0, len(availableWall)-1)]}, model.AnimationKindLoop)

		sector.Segments = append(sector.Segments, seg)
	}

	return sector
}

func Generate() (*model.ConfigRoot, error) {
	basePath := "resources" + string(os.PathSeparator) + "textures" + string(os.PathSeparator)
	t, _ := NewTextures(basePath)

	//return generateSimple(t, 16, 16)
	return generateDungeon(t, 80, 80, 8.0)
}

// GenerateSimple creates a new game configuration with sectors, a player, and randomized structures based on grid dimensions.
func generateSimple(t *Textures, maxX int, maxY int) (*model.ConfigRoot, error) {
	configPlayer := &model.ConfigPlayer{}
	cfg := model.NewConfigRoot(nil, configPlayer, nil, 0, false, t)
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
func generateDungeon(t *Textures, gridWidth int, gridHeight int, cellSize float64) (*model.ConfigRoot, error) {
	cfg := model.NewConfigRoot(nil, &model.ConfigPlayer{}, nil, 0, false, t)

	// 1. Generazione Logica (Drunkard's Walk)
	grid := make([][]bool, gridWidth)
	for i := range grid {
		grid[i] = make([]bool, gridHeight)
	}

	cx, cy := gridWidth/2, gridHeight/2
	roomsCount := (gridWidth * gridHeight) / 3 // Densità del dungeon
	dirs := []struct{ dx, dy int }{{0, -1}, {1, 0}, {0, 1}, {-1, 0}}

	for i := 0; i < roomsCount; i++ {
		grid[cx][cy] = true
		d := dirs[random(0, 3)]
		cx += d.dx
		cy += d.dy
		// Clamp per non uscire dalla mappa
		if cx < 1 {
			cx = 1
		} else if cx >= gridWidth-1 {
			cx = gridWidth - 2
		}
		if cy < 1 {
			cy = 1
		} else if cy >= gridHeight-1 {
			cy = gridHeight - 2
		}
	}

	// 2. Creazione Settori e Altitudini
	sectorGrid := make([][]*model.ConfigSector, gridWidth)
	for i := range sectorGrid {
		sectorGrid[i] = make([]*model.ConfigSector, gridHeight)
	}

	for x := 0; x < gridWidth; x++ {
		for y := 0; y < gridHeight; y++ {
			if !grid[x][y] {
				continue
			}

			id := fmt.Sprintf("cell_%d_%d", x, y)
			sector := model.NewConfigSector(id)

			// Creiamo un dislivello progressivo dal centro per simulare gradini/colline
			distFromCenter := math.Abs(float64(x-gridWidth/2)) + math.Abs(float64(y-gridHeight/2))
			sector.FloorY = distFromCenter * 1.5 // Altezza gradino
			sector.CeilY = sector.FloorY + randomF(20.0, 100.0)

			sector.Animations.Floor = model.NewConfigAnimation([]string{availableFloor[random(0, len(availableFloor)-1)]}, model.AnimationKindLoop)
			sector.Animations.Ceil = model.NewConfigAnimation([]string{availableCeil[random(0, len(availableCeil)-1)]}, model.AnimationKindLoop)
			sector.Animations.ScaleFactor = 50.0

			sector.Light.Intensity = randomF(0.2, 1.0)
			sector.Light.Kind = model.LightKindSpot

			sectorGrid[x][y] = sector
			cfg.Sectors = append(cfg.Sectors, sector)
		}
	}

	// 3. Generazione Topologica (Edge e Portali)
	for x := 0; x < gridWidth; x++ {
		for y := 0; y < gridHeight; y++ {
			sector := sectorGrid[x][y]
			if sector == nil {
				continue
			}

			px, py := float64(x)*cellSize, float64(y)*cellSize

			// Definiamo i 4 bordi (Nord, Est, Sud, Ovest)
			edges := []struct {
				nx, ny int
				p1, p2 model.XY
			}{
				{x, y - 1, model.XY{X: px, Y: py}, model.XY{X: px + cellSize, Y: py}},                       // Nord
				{x + 1, y, model.XY{X: px + cellSize, Y: py}, model.XY{X: px + cellSize, Y: py + cellSize}}, // Est
				{x, y + 1, model.XY{X: px + cellSize, Y: py + cellSize}, model.XY{X: px, Y: py + cellSize}}, // Sud
				{x - 1, y, model.XY{X: px, Y: py + cellSize}, model.XY{X: px, Y: py}},                       // Ovest
			}

			for _, e := range edges {
				neighborId := "unknown"
				kind := model.DefinitionWall // Assume sia un muro solido

				// Controllo adiacenze per aprire il portale
				if e.nx >= 0 && e.nx < gridWidth && e.ny >= 0 && e.ny < gridHeight {
					if neighbor := sectorGrid[e.nx][e.ny]; neighbor != nil {
						neighborId = neighbor.Id
						kind = model.DefinitionJoin // Il bordo diventa un portale
					}
				}

				seg := model.NewConfigSegment("", kind, e.p1, e.p2)
				seg.Neighbor = neighborId
				seg.Animations.Upper = model.NewConfigAnimation([]string{availableUpper[random(0, len(availableUpper)-1)]}, model.AnimationKindLoop)
				seg.Animations.Lower = model.NewConfigAnimation([]string{availableLower[random(0, len(availableLower)-1)]}, model.AnimationKindLoop)
				seg.Animations.Middle = model.NewConfigAnimation([]string{availableWall[random(0, len(availableWall)-1)]}, model.AnimationKindLoop)

				sector.Segments = append(sector.Segments, seg)
			}
		}
	}

	// 4. Spawn del Giocatore al centro esatto
	cfg.Player.Sector = fmt.Sprintf("cell_%d_%d", gridWidth/2, gridHeight/2)
	cfg.Player.Position = model.XY{X: float64(gridWidth/2)*cellSize + cellSize/2, Y: float64(gridHeight/2)*cellSize + cellSize/2}
	cfg.Player.Angle = 0.0

	return cfg, nil
}
