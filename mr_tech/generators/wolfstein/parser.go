package wolfstein

import (
	"fmt"
	"os"
	"strconv"

	"github.com/markel1974/godoom/mr_tech/model/config"
	"github.com/markel1974/godoom/mr_tech/model/geometry"
)

// floorCeilScaleW defines a fixed scaling factor for animations applied to floor and ceiling textures.
const floorCeilScaleW = 50.0

// floorCeilScaleH defines the height scaling factor for floor and ceiling textures, used in rendering calculations.
const floorCeilScaleH = 50.0

// isDoor checks if the given cell value corresponds to a door by verifying if it is within the range [90, 101].
func isDoor(cell uint16) bool {
	return cell >= 90 && cell <= 101
}

// Parser represents a structure for parsing and managing map configurations for grid-based environments.
type Parser struct {
	tileSize     float64
	sectorHeight float64
	gridWidth    int
	gridHeight   int
	mapData      []uint16
	sectorIds    [][]string
	openDoors    bool
}

// NewParser creates and initializes a new Parser instance with the specified tile size, sector height, and door state.
func NewParser(tileSize float64, sectorHeight float64, openDoors bool) *Parser {
	return &Parser{
		tileSize:     tileSize,
		sectorHeight: sectorHeight,
		openDoors:    openDoors,
	}
}

// Parse converts the given map data into a configuration object suitable for use in the game's rendering and logic systems.
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
	isSolid := func(nx, ny int) bool {
		if nx < 0 || nx >= width || ny < 0 || ny >= height {
			return true
		}
		adj := wp.mapData[ny*width+nx]
		return adj != 0 && !isDoor(adj)
	}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			cell := wp.mapData[y*width+x]
			if cell != 0 && !isDoor(cell) {
				continue
			}
			sid := wp.sectorIds[y][x]
			cs := config.NewConfigSector(sid, 1.0, config.LightKindAmbient)
			cs.FloorY = 0
			cs.CeilY = wp.sectorHeight
			cs.Tag = "wolf_cell"
			if isDoor(cell) {
				cs.Tag = "door"
			}
			cs.Floor = config.NewConfigAnimation([]string{"floor.ppm"}, config.AnimationKindLoop, floorCeilScaleW, floorCeilScaleH)
			cs.Ceil = config.NewConfigAnimation([]string{"ceil.ppm"}, config.AnimationKindLoop, floorCeilScaleW, floorCeilScaleH)
			x0, x1 := float64(x)*wp.tileSize, float64(x+1)*wp.tileSize
			y0, y1 := float64(y)*wp.tileSize, float64(y+1)*wp.tileSize
			pTL := geometry.XY{X: x0, Y: y0}
			pTR := geometry.XY{X: x1, Y: y0}
			pBR := geometry.XY{X: x1, Y: y1}
			pBL := geometry.XY{X: x0, Y: y1}

			if isDoor(cell) {
				solidN := isSolid(x, y-1)
				solidS := isSolid(x, y+1)
				if solidN || solidS {
					pMidT := geometry.XY{X: x0 + wp.tileSize/2, Y: y0}
					pMidB := geometry.XY{X: x0 + wp.tileSize/2, Y: y1}
					wp.addSegment(cs, width, height, pTL, pMidT, x, y-1, cell)
					wp.addSegment(cs, width, height, pMidT, pTR, x, y-1, cell)
					wp.addSegment(cs, width, height, pTR, pBR, x+1, y, cell)
					wp.addSegment(cs, width, height, pBR, pMidB, x, y+1, cell)
					wp.addSegment(cs, width, height, pMidB, pBL, x, y+1, cell)
					wp.addSegment(cs, width, height, pBL, pTL, x-1, y, cell)
					// Switch: Generate the physical door wall only if OpenDoors is false
					if !wp.openDoors {
						doorSeg := config.NewConfigSegment(cs.Id, config.DefinitionWall, pMidT, pMidB, "")
						anim := config.NewConfigAnimation([]string{"door.ppm"}, config.AnimationKindLoop, floorCeilScaleW, floorCeilScaleH)
						doorSeg.Upper, doorSeg.Middle, doorSeg.Lower = anim, anim, anim
						cs.Segments = append(cs.Segments, doorSeg)
					}
					root.Vertices = append(root.Vertices, pTL, pTR, pBR, pBL, pMidT, pMidB)

				} else {
					pMidL := geometry.XY{X: x0, Y: y0 + wp.tileSize/2}
					pMidR := geometry.XY{X: x1, Y: y0 + wp.tileSize/2}
					wp.addSegment(cs, width, height, pTL, pTR, x, y-1, cell)
					wp.addSegment(cs, width, height, pTR, pMidR, x+1, y, cell)
					wp.addSegment(cs, width, height, pMidR, pBR, x+1, y, cell)
					wp.addSegment(cs, width, height, pBR, pBL, x, y+1, cell)
					wp.addSegment(cs, width, height, pBL, pMidL, x-1, y, cell)
					wp.addSegment(cs, width, height, pMidL, pTL, x-1, y, cell)
					// Switch: Generate the physical door wall only if OpenDoors is false
					if !wp.openDoors {
						doorSeg := config.NewConfigSegment(cs.Id, config.DefinitionWall, pMidL, pMidR, "")
						anim := config.NewConfigAnimation([]string{"door.ppm"}, config.AnimationKindLoop, floorCeilScaleW, floorCeilScaleH)
						doorSeg.Upper, doorSeg.Middle, doorSeg.Lower = anim, anim, anim
						cs.Segments = append(cs.Segments, doorSeg)
					}
					root.Vertices = append(root.Vertices, pTL, pTR, pBR, pBL, pMidL, pMidR)
				}
			} else {
				wp.addSegment(cs, width, height, pTL, pTR, x, y-1, cell)
				wp.addSegment(cs, width, height, pTR, pBR, x+1, y, cell)
				wp.addSegment(cs, width, height, pBR, pBL, x, y+1, cell)
				wp.addSegment(cs, width, height, pBL, pTL, x-1, y, cell)
				root.Vertices = append(root.Vertices, pTL, pTR, pBR, pBL)
			}

			root.Sectors = append(root.Sectors, cs)
		}
	}

	if len(root.Sectors) > 0 && len(root.Sectors[0].Segments) > 0 {
		player.Position = root.Sectors[0].Segments[0].End
	}
	return root, nil
}

// prepare initializes the grid dimensions, map data, and sector IDs for the parser. It processes doors and empty cells.
func (wp *Parser) prepare(width int, height int, md []uint16) error {
	wp.gridWidth = width
	wp.gridHeight = height
	wp.mapData = md
	wp.sectorIds = make([][]string, height)
	for y := 0; y < height; y++ {
		wp.sectorIds[y] = make([]string, width)
		for x := 0; x < width; x++ {
			if wp.mapData[y*width+x] == 0 || isDoor(wp.mapData[y*width+x]) {
				wp.sectorIds[y][x] = strconv.Itoa(y*width + x)
			}
		}
	}
	return nil
}

// addSegment appends a new ConfigSegment to a ConfigSector based on sector geometry, neighboring cells, and textures.
func (wp *Parser) addSegment(cs *config.ConfigSector, width, height int, start, end geometry.XY, nx, ny int, currentCell uint16) {
	kind := config.DefinitionWall
	neighborId := ""
	texName := "wall1.ppm"
	if nx >= 0 && nx < width && ny >= 0 && ny < height {
		adj := wp.mapData[ny*width+nx]
		if adj == 0 || isDoor(adj) {
			kind = config.DefinitionJoin
			neighborId = wp.sectorIds[ny][nx]
		} else {
			texName = fmt.Sprintf("wall%d.ppm", adj)
		}
	}
	seg := config.NewConfigSegment(cs.Id, kind, start, end, neighborId)
	// If the current cell is a door and this segment borders a solid wall,
	// it's the track on which the door will slide!
	if kind == config.DefinitionWall {
		if isDoor(currentCell) {
			texName = "doortrak.ppm"
		}
		anim := config.NewConfigAnimation([]string{texName}, config.AnimationKindLoop, 10.0, 50.0)
		seg.Upper, seg.Middle, seg.Lower = anim, anim, anim
	}
	cs.Segments = append(cs.Segments, seg)
}
