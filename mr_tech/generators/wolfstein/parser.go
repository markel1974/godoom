package wolfstein

import (
	"fmt"
	"math"
	"strconv"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/model/geometry"
)

// floorCeilScaleW defines the default scaling factor for floor and ceiling texture animations in the configuration.
const floorCeilScaleW = 10.0

// floorCeilScaleH defines the scale factor applied to the height of floor and ceiling textures in a sector.
const floorCeilScaleH = 10.0

func isWall(cell uint16) bool {
	return cell >= 1 && cell <= 99
}

// isDoor checks if the given cell value corresponds to a door, defined by the range 90 to 101 inclusive.
func isDoor(cell uint16) bool {
	return cell >= 100 && cell <= 109
}

func isThing2(cell uint16) bool {
	//136 - 183 objects | 183 - 197 enemies (with animation)
	return (cell >= 136 && cell <= 183) || isEnemy(cell)
}

func isEnemy(cell uint16) bool {
	return cell >= 183 && cell <= 197
}

// isThing determines if a given cell value represents a "thing" based on predefined ranges of values.
func isThingOrEnemy(cell uint16) bool {
	//136 - 183 objects | 183 - 197 enemies (with animation)
	return isThing2(cell) || isEnemy(cell)
}

// Parser represents a structure used for parsing and managing 2D grid-based map data, including sectors and entities.
type Parser struct {
	tileSize     float64
	sectorHeight float64
	gridWidth    int
	gridHeight   int
	mapData      []uint16
	sectorIds    [][]string
	openDoors    bool
}

// NewParser creates a new Parser instance with specified tile size, sector height, and door openness settings.
func NewParser(tileSize float64, sectorHeight float64, openDoors bool) *Parser {
	return &Parser{
		tileSize:     tileSize,
		sectorHeight: sectorHeight,
		openDoors:    openDoors,
	}
}

// Parse constructs a Root by processing a map grid of given dimensions and metadata, generating sectors and objects.
func (wp *Parser) Parse(width int, height int, md []uint16) (*config.Root, error) {
	const useEnemy = true
	if len(md) != width*height {
		return nil, fmt.Errorf("mapData size does not match width * height")
	}
	texProvider, tErr := NewTextures()
	if tErr != nil {
		return nil, tErr
	}

	cal := config.NewConfigCalibration(false, 0, 0, 0, 0, 0, 0, true)
	root := config.NewConfigRoot(cal, nil, nil, nil, 1.0, texProvider)
	if err := wp.prepare(width, height, md); err != nil {
		return nil, err
	}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			cell := wp.mapData[y*width+x]
			if isWall(cell) {
				continue
			}
			if isThingOrEnemy(cell) {
				pos := geometry.XYZ{
					X: float64(x)*wp.tileSize + wp.tileSize/2,
					Y: float64(y)*wp.tileSize + wp.tileSize/2,
					Z: 0,
				}
				var angle float64 = 0
				kind := config.ThingItemDef
				object := "item"
				sequence := []string{fmt.Sprintf("image%d.png", cell)}
				// Se è un nemico, calcoliamo l'orientamento originale
				if isEnemy(cell) {
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
					sequence = []string{"image194.png", "image195.png", "image196.png", "image197.png"}
				}
				id := fmt.Sprintf("thing_%s_%d_%d", object, x, y)
				const scaleH = 0.08
				anim := config.NewConfigAnimation(sequence, config.AnimationKindLoop, scaleH, scaleH*2)
				if useEnemy {
					thing := config.NewConfigThing(id, pos, angle, kind, 10.0, 1, 1, 6, anim)
					root.Things = append(root.Things, thing)
				}
				cell = 0 // Libera la cella per il compilatore topologico
			}

			const falloff = 10.0
			const lightIntensity = 1.5
			sid := wp.sectorIds[y][x]
			cs := config.NewConfigSector(sid, lightIntensity, config.LightKindAmbient, falloff)
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
				_, wallN := wp.isWall(width, height, x, y-1)
				_, wallS := wp.isWall(width, height, x, y+1)
				if wallN || wallS {
					pMidT := geometry.XY{X: x0 + wp.tileSize/2, Y: y0}
					pMidB := geometry.XY{X: x0 + wp.tileSize/2, Y: y1}
					wp.addSegment(cs, width, height, pTL, pMidT, x, y-1, cell)
					wp.addSegment(cs, width, height, pMidT, pTR, x, y-1, cell)
					wp.addSegment(cs, width, height, pTR, pBR, x+1, y, cell)
					wp.addSegment(cs, width, height, pBR, pMidB, x, y+1, cell)
					wp.addSegment(cs, width, height, pMidB, pBL, x, y+1, cell)
					wp.addSegment(cs, width, height, pBL, pTL, x-1, y, cell)
					if !wp.openDoors {
						doorSeg := config.NewConfigSegment(cs.Id, config.SegmentWall, pMidT, pMidB)
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
						doorSeg := config.NewConfigSegment(cs.Id, config.SegmentWall, pMidL, pMidR)
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

	var playerPos geometry.XYZ
	if len(root.Sectors) > 0 && len(root.Sectors[0].Segments) > 0 {
		pos := root.Sectors[0].Segments[0].End
		playerPos = geometry.XYZ{X: pos.X, Y: pos.Y, Z: 0}
	}
	root.Player = config.NewConfigPlayer(playerPos, 0, 8, 4, 20)
	return root, nil
}

// prepare initializes the Parser with the given grid dimensions and map data, and generates sector IDs for relevant cells.
func (wp *Parser) prepare(width int, height int, md []uint16) error {
	wp.gridWidth = width
	wp.gridHeight = height
	wp.mapData = md
	wp.sectorIds = make([][]string, height)
	for y := 0; y < height; y++ {
		wp.sectorIds[y] = make([]string, width)
		for x := 0; x < width; x++ {
			if target, wall := wp.isWall(width, height, x, y); !wall {
				wp.sectorIds[y][x] = strconv.Itoa(target)
			}
		}
	}
	return nil
}

// addSegment adds a new segment to the specified Sector using level geometry, neighbors, and texture information.
func (wp *Parser) addSegment(cs *config.Sector, width, height int, start, end geometry.XY, nx, ny int, currentCell uint16) {
	kind := config.SegmentUnknown
	isAdjDoor := false
	cell := uint16(1)
	if _, wall := wp.isWall(width, height, nx, ny); wall {
		kind = config.SegmentWall
	} else {
		if isDoor(cell) {
			isAdjDoor = true
		}
	}
	seg := config.NewConfigSegment(cs.Id, kind, start, end)
	isEW := math.Abs(start.X-end.X) < 0.001 // Calcolo normale: X costante -> Est/Ovest
	var texName string
	if isDoor(currentCell) {
		if kind != config.SegmentWall {
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
		if kind == config.SegmentWall {
			idx := (int(cell) - 1) * 2
			if isEW {
				idx++
			}
			texName = fmt.Sprintf("image%d.png", idx)
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

// isSolid checks if a given map cell is solid based on its position and attributes from the map data.
func (wp *Parser) isWall(width, height, nx, ny int) (int, bool) {
	target := (ny * width) + nx
	if nx < 0 || nx >= width || ny < 0 || ny >= height {
		return target, true
	}
	cell := wp.mapData[target]
	return target, isWall(cell)
}
