package dungeon

import (
	"encoding/json"
	"fmt"
	"math"
	rnd "math/rand"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/generators/common"
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/utils"
)

const (
	lightFalloff = 100.0
	cellSize     = 16.0
)

const (
	AspectRatio  = 1.333
	ScaleSectorH = 0.8
	PlayerHeight = 15.0 * ScaleSectorH
	PlayerRadius = 4.0
	PlayerSpeed  = 800.0
	PlayerMass   = 50.0

	GForce    = 9.8 * 50
	JumpForce = GForce * 8
)

const (
	texUnitSize = 32.0
	scaleTex    = 1024.0 / texUnitSize
)

// availableCeil contains a list of available ceiling texture file names in PPM format.
var _availableCeil = []string{"ceil.ppm", "ceil2.ppm", "ceil2_norm.ppm"}

// availableFloor represents a list of texture filenames that can be used as floor textures in the game environment.
var _availableFloor = []string{"floor.ppm"}

// availableUpper contains a list of file names for textures used as the upper surface in sectors.
var _availableUpper = []string{"wall2.ppm"}

// availableLower stores a list of texture file names used for lower wall surfaces in the level configuration.
var _availableLower = []string{"wall2.ppm"}

// availableWall holds a list of wall texture file names available for sector configurations.
var _availableWall = []string{"wall2.ppm"}

// random generates a random integer between the specified min (inclusive) and max (inclusive) values.
func random(min int, max int) int {
	return rnd.Intn(max-min+1) + min
}

// randomF generates a random float64 value between the specified min and max bounds using a uniform distribution.
func randomF(min float64, max float64) float64 {
	return min + rnd.Float64()*(max-min)
}

// Builder provides methods to construct and generate a configuration tree for the application.
type Builder struct {
}

// NewBuilder creates and returns a new instance of Builder.
func NewBuilder() *Builder {
	return &Builder{}
}

// Build generates and returns the root configuration for the application or system, along with any encountered errors.
func (b *Builder) Build(res common.IResources, level int) (*config.Root, error) {
	t, _ := NewTextures(res)
	//return b.generateSimple(t, 16, 16)
	return b.generateDungeon(t, 32, 32, cellSize)
}

// createCube initializes and returns a Sector representing a cubical sector in a level with specified properties.
func (b *Builder) createCube(x float64, y float64, max float64, floor float64, ceil float64) *config.Sector {
	sector := config.NewConfigSector(utils.NextUUId(), rnd.Float64(), config.LightKindAmbient, lightFalloff)
	sector.FloorY = floor
	sector.CeilY = ceil

	floorT := []string{_availableFloor[random(0, len(_availableFloor)-1)]}
	ceilT := []string{_availableCeil[random(0, len(_availableCeil)-1)]}
	sector.Floor = config.NewConfigMaterial(floorT, config.MaterialKindLoop, 0, 0)
	sector.Ceil = config.NewConfigMaterial(ceilT, config.MaterialKindLoop, 0, 0)
	sector.FloorAlign = config.NewUVAlignSurface(false, geometry.XY{}, geometry.XY{}, false, false, false, geometry.XY{}, scaleTex, scaleTex)
	sector.CeilAlign = config.NewUVAlignSurface(false, geometry.XY{}, geometry.XY{}, false, false, false, geometry.XY{}, scaleTex, scaleTex)

	pts := [4]geometry.XY{
		{X: x, Y: y},
		{X: x + max, Y: y},
		{X: x + max, Y: y + max},
		{X: x, Y: y + max},
	}

	for i := 0; i < 4; i++ {
		start := pts[i]
		end := pts[(i+1)%4]

		seg := config.NewConfigSegment("", config.SegmentUnknown, start, end)

		upperT := []string{_availableUpper[random(0, len(_availableUpper)-1)]}
		lowerT := []string{_availableLower[random(0, len(_availableLower)-1)]}
		middleT := []string{_availableWall[random(0, len(_availableWall)-1)]}
		seg.Upper = config.NewConfigMaterial(upperT, config.MaterialKindLoop, 0, 0)
		seg.Lower = config.NewConfigMaterial(lowerT, config.MaterialKindLoop, 0, 0)
		seg.Middle = config.NewConfigMaterial(middleT, config.MaterialKindLoop, 0, 0)

		wallLen := math.Hypot(end.X-start.X, end.Y-start.Y)
		panning := geometry.XY{}
		seg.UVAlign = config.NewUVAlignVertical(false, false, false, panning, wallLen, scaleTex)
		seg.UVAlignUpper = config.NewUVAlignVertical(false, false, false, panning, wallLen, scaleTex)
		seg.UVAlignLower = config.NewUVAlignVertical(false, false, false, panning, wallLen, scaleTex)

		sector.Segments = append(sector.Segments, seg)
	}

	return sector
}

// GenerateSimple creates a new game configuration with sectors, a player, and randomized structures based on grid dimensions.
func (b *Builder) generateSimple(t *Textures, maxX int, maxY int) (*config.Root, error) {
	p := NewPlayer(0, 0, 0)
	player2 := p.Build()
	_ = common.NewPlayer(player2)
	cal := config.NewConfigCalibration(0, 0, 0, 0, 0, 0, true)
	cal.AspectRatio = AspectRatio

	scaleFactor := geometry.XYZ{X: 1, Y: 1, Z: 1}
	cfg := config.NewConfigRoot(cal, nil, player2, nil, scaleFactor, t)
	s1 := b.createCube(0, 0, 8, 0, 20)
	s1.Id = "root"
	cfg.Sectors = append(cfg.Sectors, s1)

	s2 := b.createCube(8, 0, 8, 0, 20)
	s1.Id = "toor"
	cfg.Sectors = append(cfg.Sectors, s2)

	for x := 1; x < maxX; x++ {
		for y := 1; y < maxY; y++ {
			create := random(0, 5)
			if x == 1 || y == 1 || create > 2 {
				ceil := randomF(15, 30)
				floor := randomF(0, 2)
				s3 := b.createCube(float64(x)*8, 8*float64(y), 8, floor, ceil)
				cfg.Sectors = append(cfg.Sectors, s3)
			}
		}
	}

	cfg.Player.Position.X = 1
	cfg.Player.Position.Y = 1

	return cfg, nil
}

func (b *Builder) generateDungeon(t *Textures, gridWidth int, gridHeight int, cellSize float64) (*config.Root, error) {
	p := NewPlayer(0, 0, 0)
	player := p.Build()
	_ = common.NewPlayer(player)
	cal := config.NewConfigCalibration(0, 0, 0, 0, 0, 0, true)
	cal.AspectRatio = AspectRatio

	scale := geometry.XYZ{X: 1, Y: 1, Z: 1}
	cfg := config.NewConfigRoot(cal, nil, player, nil, scale, t)

	//  Generazione Logica (Drunkard's Walk)
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

	// Creazione Settori e Altitudini
	sectorGrid := make([][]*config.Sector, gridWidth)
	for i := range sectorGrid {
		sectorGrid[i] = make([]*config.Sector, gridHeight)
	}

	for x := 0; x < gridWidth; x++ {
		for y := 0; y < gridHeight; y++ {
			if !grid[x][y] {
				continue
			}

			id := fmt.Sprintf("cell_%d_%d", x, y)
			lightIntensity := randomF(0.1, 0.5)
			sector := config.NewConfigSector(id, lightIntensity, config.LightKindAmbient, lightFalloff)

			// Creiamo un dislivello progressivo dal centro per simulare gradini/colline
			distFromCenter := math.Abs(float64(x-gridWidth/2)) + math.Abs(float64(y-gridHeight/2))
			sector.FloorY = distFromCenter * 1.5 // Altezza gradino
			sector.CeilY = sector.FloorY + randomF(20.0, 100.0)

			floorT := []string{_availableFloor[random(0, len(_availableFloor)-1)]}
			ceilT := []string{_availableCeil[random(0, len(_availableCeil)-1)]}
			sector.Floor = config.NewConfigMaterial(floorT, config.MaterialKindLoop, 0, 0)
			sector.Ceil = config.NewConfigMaterial(ceilT, config.MaterialKindLoop, 0, 0)

			sector.FloorAlign = config.NewUVAlignSurface(false, geometry.XY{}, geometry.XY{}, false, false, false, geometry.XY{}, scaleTex, scaleTex)
			sector.CeilAlign = config.NewUVAlignSurface(false, geometry.XY{}, geometry.XY{}, false, false, false, geometry.XY{}, scaleTex, scaleTex)

			sectorGrid[x][y] = sector
			cfg.Sectors = append(cfg.Sectors, sector)
		}
	}

	// Generazione Topologica (Edge e Portali)
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
				p1, p2 geometry.XY
			}{
				{x, y - 1, geometry.XY{X: px, Y: py}, geometry.XY{X: px + cellSize, Y: py}},                       // Nord
				{x + 1, y, geometry.XY{X: px + cellSize, Y: py}, geometry.XY{X: px + cellSize, Y: py + cellSize}}, // Est
				{x, y + 1, geometry.XY{X: px + cellSize, Y: py + cellSize}, geometry.XY{X: px, Y: py + cellSize}}, // Sud
				{x - 1, y, geometry.XY{X: px, Y: py + cellSize}, geometry.XY{X: px, Y: py}},                       // Ovest
			}

			for _, e := range edges {
				kind := config.SegmentUnknown
				hasNeighbor := false
				// Controllo adiacenze per aprire il portale
				if e.nx >= 0 && e.nx < gridWidth && e.ny >= 0 && e.ny < gridHeight {
					if neighbor := sectorGrid[e.nx][e.ny]; neighbor != nil {
						hasNeighbor = true
					}
				}
				if !hasNeighbor {
					kind = config.SegmentWall
				}

				seg := config.NewConfigSegment("", kind, e.p1, e.p2)
				wallLen := math.Hypot(e.p2.X-e.p1.X, e.p2.Y-e.p1.Y) * scaleTex
				panning := geometry.XY{}

				if kind == config.SegmentWall {
					middleT := []string{_availableWall[random(0, len(_availableWall)-1)]}
					seg.Middle = config.NewConfigMaterial(middleT, config.MaterialKindLoop, 0, 0)
					seg.UVAlign = config.NewUVAlignVertical(false, false, false, panning, wallLen, scaleTex)
				} else {
					upperT := []string{_availableUpper[random(0, len(_availableUpper)-1)]}
					lowerT := []string{_availableLower[random(0, len(_availableLower)-1)]}
					seg.Upper = config.NewConfigMaterial(upperT, config.MaterialKindLoop, 0, 0)
					seg.Lower = config.NewConfigMaterial(lowerT, config.MaterialKindLoop, 0, 0)
					seg.UVAlignUpper = config.NewUVAlignVertical(false, false, false, panning, wallLen, scaleTex)
					seg.UVAlignLower = config.NewUVAlignVertical(false, false, false, panning, wallLen, scaleTex)
				}

				sector.Segments = append(sector.Segments, seg)
			}
		}
	}

	// Spawn del Giocatore al centro esatto
	cfg.Player.Position = geometry.XYZ{X: float64(gridWidth/2)*cellSize + cellSize/2, Y: float64(gridHeight/2)*cellSize + cellSize/2, Z: 0}
	cfg.Player.Angle = 0.0

	return cfg, nil
}

// ParseJsonData parses a JSON-encoded byte array into a Root struct and returns it or an error on failure.
func (b *Builder) parseJsonData(source []byte) (*config.Root, error) {
	cfg := &config.Root{}
	if err := json.Unmarshal(source, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
