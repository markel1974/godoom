package wolfstein

import (
	"fmt"
	"math"
	"strconv"

	"github.com/markel1974/godoom/mr_tech/model/config"
	"github.com/markel1974/godoom/mr_tech/model/geometry"
)

// floorCeilScaleW defines the scaling factor for floor and ceiling textures in the horizontal direction.
const floorCeilScaleW = 10.0

// floorCeilScaleH defines the height scaling factor for floor and ceiling animations in the configuration.
const floorCeilScaleH = 10.0

// isDoor determines if a given cell value represents a door based on its range in the map data.
func isDoor(cell uint16) bool {
	return cell >= 90 && cell <= 101
}

// isThing rileva nemici (108-143) e oggetti/decorazioni (23-79)
func isThing(cell uint16) bool {
	return (cell >= 108 && cell <= 143) || (cell >= 23 && cell <= 79)
}

// Parser represents a data structure used to parse and manage grid-based map configurations and their associated metadata.
type Parser struct {
	tileSize     float64
	sectorHeight float64
	gridWidth    int
	gridHeight   int
	mapData      []uint16
	sectorIds    [][]string
	openDoors    bool
}

// NewParser initializes and returns a new instance of Parser with the specified tile size, sector height, and door state.
func NewParser(tileSize float64, sectorHeight float64, openDoors bool) *Parser {
	return &Parser{
		tileSize:     tileSize,
		sectorHeight: sectorHeight,
		openDoors:    openDoors,
	}
}

// Parse generates a ConfigRoot based on map dimensions and data, handling sectors, things, doors, and textures.
func (wp *Parser) Parse(width int, height int, md []uint16) (*config.ConfigRoot, error) {
	if len(md) != width*height {
		return nil, fmt.Errorf("mapData size does not match width * height")
	}
	texProvider, tErr := NewTextures()
	if tErr != nil {
		return nil, tErr
	}
	player := &config.ConfigPlayer{}
	root := config.NewConfigRoot(nil, player, nil, 1.0, false, texProvider)

	if err := wp.prepare(width, height, md); err != nil {
		return nil, err
	}

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			cell := wp.mapData[y*width+x]
			if cell != 0 && !isDoor(cell) && !isThing(cell) {
				continue
			}

			if isThing(cell) {
				pos := geometry.XY{
					X: float64(x)*wp.tileSize + wp.tileSize/2,
					Y: float64(y)*wp.tileSize + wp.tileSize/2,
				}

				var angle float64 = 0
				kind := config.ThingItemDef // Default per oggetti
				object := "item"

				// Se è un nemico, calcoliamo l'orientamento originale
				if cell >= 108 {
					object = "enemy"
					kind = config.ThingEnemyDef
					dir := (cell - 108) % 4
					switch dir {
					case 0:
						angle = math.Pi / 2
					case 1:
						angle = 0
					case 2:
						angle = -math.Pi / 2
					case 3:
						angle = math.Pi
					}
				}

				id := fmt.Sprintf("thing_%s_%d_%d", object, x, y)
				sequence := []string{"image194.png", "image195.png", "image196.png", "image197.png"}
				const scaleH = 0.08
				anim := config.NewConfigAnimation(sequence, config.AnimationKindLoop, scaleH, scaleH*2)

				// Raggio e massa possono variare: i nemici sono solidi, i pickup no (gestito nel runtime)
				thing := config.NewConfigThing(id, pos, angle, kind, 10.0, 1, 1, 0.01, anim)
				root.Things = append(root.Things, thing)

				cell = 0 // Libera la cella per il compilatore topologico
			}

			sid := wp.sectorIds[y][x]
			lightIntensity := 0.5 //rnd.Float64()
			cs := config.NewConfigSector(sid, lightIntensity, config.LightKindAmbient)
			cs.FloorY = 0
			cs.CeilY = wp.sectorHeight
			cs.Tag = "wolf_cell"

			cs.Floor = config.NewConfigAnimation([]string{"image91.png"}, config.AnimationKindLoop, floorCeilScaleW, floorCeilScaleH)
			cs.Ceil = config.NewConfigAnimation([]string{"image89.png"}, config.AnimationKindLoop, floorCeilScaleW, floorCeilScaleH)

			x0, x1 := float64(x)*wp.tileSize, float64(x+1)*wp.tileSize
			y0, y1 := float64(y)*wp.tileSize, float64(y+1)*wp.tileSize
			pTL := geometry.XY{X: x0, Y: y0}
			pTR := geometry.XY{X: x1, Y: y0}
			pBR := geometry.XY{X: x1, Y: y1}
			pBL := geometry.XY{X: x0, Y: y1}

			if isDoor(cell) {
				cs.Tag = "door"
				solidN := wp.isSolid(width, height, x, y-1)
				solidS := wp.isSolid(width, height, x, y+1)

				if solidN || solidS {
					pMidT := geometry.XY{X: x0 + wp.tileSize/2, Y: y0}
					pMidB := geometry.XY{X: x0 + wp.tileSize/2, Y: y1}

					wp.addSegment(cs, width, height, pTL, pMidT, x, y-1, cell)
					wp.addSegment(cs, width, height, pMidT, pTR, x, y-1, cell)
					wp.addSegment(cs, width, height, pTR, pBR, x+1, y, cell)
					wp.addSegment(cs, width, height, pBR, pMidB, x, y+1, cell)
					wp.addSegment(cs, width, height, pMidB, pBL, x, y+1, cell)
					wp.addSegment(cs, width, height, pBL, pTL, x-1, y, cell)

					if !wp.openDoors {
						doorSeg := config.NewConfigSegment(cs.Id, config.DefinitionWall, pMidT, pMidB)
						anim := config.NewConfigAnimation([]string{"image99.png"}, config.AnimationKindLoop, 10.0, 10.0)
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

					if !wp.openDoors {
						doorSeg := config.NewConfigSegment(cs.Id, config.DefinitionWall, pMidL, pMidR)
						anim := config.NewConfigAnimation([]string{"image98.png"}, config.AnimationKindLoop, 10.0, 50.0)
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

// prepare initializes the grid dimensions, map data, and sector IDs for the parser, preparing it for further processing.
func (wp *Parser) prepare(width int, height int, md []uint16) error {
	wp.gridWidth = width
	wp.gridHeight = height
	wp.mapData = md

	wp.sectorIds = make([][]string, height)
	for y := 0; y < height; y++ {
		wp.sectorIds[y] = make([]string, width)
		for x := 0; x < width; x++ {
			if wp.mapData[y*width+x] == 0 || isDoor(wp.mapData[y*width+x]) || isThing(wp.mapData[y*width+x]) {
				wp.sectorIds[y][x] = strconv.Itoa(y*width + x)
			}
		}
	}
	return nil
}

// addSegment adds a segment to the specified ConfigSector with geometry, texture, adjacency, and type information.
func (wp *Parser) addSegment(cs *config.ConfigSector, width, height int, start, end geometry.XY, nx, ny int, currentCell uint16) {
	kind := config.DefinitionUnknown
	isAdjDoor := false
	adj := uint16(1)
	if nx >= 0 && nx < width && ny >= 0 && ny < height {
		adj = wp.mapData[ny*width+nx]
		if adj == 0 || isDoor(adj) || isThing(adj) {
			if isDoor(adj) {
				isAdjDoor = true
			}
		} else {
			kind = config.DefinitionWall
		}
	}

	seg := config.NewConfigSegment(cs.Id, kind, start, end)

	// Calcolo normale: X costante -> Est/Ovest
	isEW := math.Abs(start.X-end.X) < 0.001
	var texName string

	if isDoor(currentCell) {
		if kind != config.DefinitionWall {
			idx := 98
			if isEW {
				idx++
			}
			texName = fmt.Sprintf("image%d.png", idx)
		} else {
			idx := 104
			if isEW {
				idx++
			}
			texName = fmt.Sprintf("image%d.png", idx)
		}
	} else {
		if kind == config.DefinitionWall {
			baseIdx := (int(adj) - 1) * 2
			if isEW {
				baseIdx++
			}
			texName = fmt.Sprintf("image%d.png", baseIdx)
		} else if isAdjDoor {
			idx := 98
			if isEW {
				idx++
			}
			texName = fmt.Sprintf("image%d.png", idx)
		}
	}

	if texName != "" {
		anim := config.NewConfigAnimation([]string{texName}, config.AnimationKindLoop, 7.0, 4.0)
		seg.Upper = anim
		seg.Middle = anim
		seg.Lower = anim
	}

	cs.Segments = append(cs.Segments, seg)
}

func (wp *Parser) isSolid(width, height, nx, ny int) bool {
	if nx < 0 || nx >= width || ny < 0 || ny >= height {
		return true
	}
	adj := wp.mapData[ny*width+nx]
	return adj != 0 && !isDoor(adj) && !isThing(adj)
}
