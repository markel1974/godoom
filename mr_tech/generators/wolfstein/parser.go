package wolfstein

import (
	"fmt"
	"os"
	"strconv"

	"github.com/markel1974/godoom/mr_tech/model/config"
	"github.com/markel1974/godoom/mr_tech/model/geometry"
)

// Parser represents a structure for parsing and organizing map data into a configurable level format.
// TileSize specifies the dimensions of each tile in the map grid.
// Height represents the height of individual map sectors.
// mapData holds the raw data for the map's structure, represented as a slice of uint16 values.
// sectorIds contains unique identifiers for sectors, organized as a 2D grid of strings.
type Parser struct {
	tileSize  float64
	height    float64
	width     float64
	mapData   []uint16
	sectorIds [][]string
}

// NewParser initializes and returns a new Parser with the specified tile size and height.
func NewParser(tileSize float64) *Parser {
	return &Parser{
		tileSize: tileSize,
	}
}

// Parse initializes and processes map data, generating configuration sectors, animations, and vertices for the game level.
func (wp *Parser) Parse(width int, height int, md []uint16) (*config.ConfigRoot, error) {
	if len(md) != width*height {
		return nil, fmt.Errorf("mapData size does not match width * height")
	}
	basePath := "resources" + string(os.PathSeparator) + "textures" + string(os.PathSeparator)
	texProvider, tErr := NewTextures(basePath)
	if tErr != nil {
		return nil, tErr
	}
	player := &config.ConfigPlayer{}
	root := config.NewConfigRoot(nil, player, nil, 1.0, false, texProvider)
	if err := wp.prepare(width, height, md); err != nil {
		return nil, err
	}

	// Animazioni di base per i flats (pavimento/soffitto)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			cell := wp.mapData[y*width+x]
			if cell != 0 {
				continue // Ignoriamo i blocchi solidi (saranno i bordi dei settori vuoti)
			}
			sid := wp.sectorIds[y][x]
			cs := config.NewConfigSector(sid, 1.0, config.LightKindAmbient)
			cs.FloorY = 0
			cs.CeilY = wp.height
			cs.Floor = config.NewConfigAnimation([]string{"floor.ppm"}, config.AnimationKindLoop, 100.0, 100.0)
			cs.Ceil = config.NewConfigAnimation([]string{"ceil.ppm"}, config.AnimationKindLoop, 200.0, 200.0)
			cs.Tag = "wolf_cell"
			// Vertici della cella (TopLeft, TopRight, BottomRight, BottomLeft)
			x0, x1 := float64(x)*wp.tileSize, float64(x+1)*wp.tileSize
			y0, y1 := float64(y)*wp.tileSize, float64(y+1)*wp.tileSize
			pTL := geometry.XY{X: x0, Y: y0}
			pTR := geometry.XY{X: x1, Y: y0}
			pBR := geometry.XY{X: x1, Y: y1}
			pBL := geometry.XY{X: x0, Y: y1}
			// Cicliamo in ordine orario/antiorario per chiudere il perimetro del settore
			// Nord
			wp.addSegment(cs, width, height, pTL, pTR, x, y-1)
			// Est
			wp.addSegment(cs, width, height, pTR, pBR, x+1, y)
			// Sud
			wp.addSegment(cs, width, height, pBR, pBL, x, y+1)
			// Ovest
			wp.addSegment(cs, width, height, pBL, pTL, x-1, y)
			root.Sectors = append(root.Sectors, cs)
			// Aggiungiamo i vertici alla lista globale per il compilatore (VertexEdges)
			root.Vertices = append(root.Vertices, pTL, pTR, pBR, pBL)
		}
	}

	player.Position = root.Sectors[0].Segments[0].End
	return root, nil
}

func (wp *Parser) prepare(width int, height int, md []uint16) error {
	if len(md) != width*height {
		return fmt.Errorf("mapData size does not match width * height")
	}
	wp.height = float64(height)
	wp.width = float64(width)
	wp.mapData = md
	// Mappatura pre-pass per generare gli ID univoci dei settori (celle vuote)
	wp.sectorIds = make([][]string, height)
	for y := 0; y < height; y++ {
		wp.sectorIds[y] = make([]string, width)
		for x := 0; x < width; x++ {
			if wp.mapData[y*width+x] == 0 {
				wp.sectorIds[y][x] = strconv.Itoa(y*width + x)
			}
		}
	}
	return nil
}

// addSegment adds a new segment to the provided ConfigSector based on map adjacency, geometry, and texture information.
func (wp *Parser) addSegment(cs *config.ConfigSector, width, height int, start, end geometry.XY, nx, ny int) {
	kind := config.DefinitionWall
	neighborId := ""
	texId := uint16(1)
	if nx >= 0 && nx < width && ny >= 0 && ny < height {
		adj := wp.mapData[ny*width+nx]
		if adj == 0 {
			kind = config.DefinitionJoin
			neighborId = wp.sectorIds[ny][nx]
		} else {
			texId = adj
		}
	}
	seg := config.NewConfigSegment(cs.Id, kind, start, end, neighborId)
	if kind == config.DefinitionWall {
		texName := fmt.Sprintf("wall%d.ppm", texId)
		anim := config.NewConfigAnimation([]string{texName}, config.AnimationKindLoop, 10.0, 50.0)
		seg.Middle = anim
	}
	cs.Segments = append(cs.Segments, seg)
}
