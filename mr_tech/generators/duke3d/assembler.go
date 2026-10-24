package duke3d

import (
	"fmt"
	"sort"

	"github.com/markel1974/godoom/mr_tech/generators/duke3d/lumps"
)

// WorldAssembler analyzes Duke3D map topology and assembles disconnected sector islands
// (such as rooftops, room-over-room layers, and isolated sub-maps connected via SE 7)
// into a single, continuous, coherent 3D world space.
type WorldAssembler struct {
	level *lumps.Level
}

// NewWorldAssembler creates an assembler instance for the specified level.
func NewWorldAssembler(level *lumps.Level) *WorldAssembler {
	return &WorldAssembler{level: level}
}

// Assemble detects disconnected sector clusters and aligns them using matching Sector Effector (SE 7) anchors.
func (a *WorldAssembler) Assemble() {
	if a.level == nil || len(a.level.Sectors) == 0 {
		return
	}

	numSects := len(a.level.Sectors)

	// Build adjacency graph from sector wall portals
	adj := make([]map[int]bool, numSects)
	for sIdx := 0; sIdx < numSects; sIdx++ {
		sec := a.level.Sectors[sIdx]
		wStart := int(sec.WallPtr)
		wCount := int(sec.WallNum)
		adj[sIdx] = make(map[int]bool)
		for i := 0; i < wCount; i++ {
			if wStart+i >= len(a.level.Walls) {
				continue
			}
			w := a.level.Walls[wStart+i]
			if w.NextSector >= 0 && int(w.NextSector) < numSects {
				adj[sIdx][int(w.NextSector)] = true
			}
		}
	}

	// Identify connected components
	visited := make([]bool, numSects)
	var components [][]int
	for sIdx := 0; sIdx < numSects; sIdx++ {
		if visited[sIdx] {
			continue
		}
		var comp []int
		queue := []int{sIdx}
		visited[sIdx] = true
		for len(queue) > 0 {
			curr := queue[0]
			queue = queue[1:]
			comp = append(comp, curr)
			for neighbor := range adj[curr] {
				if !visited[neighbor] {
					visited[neighbor] = true
					queue = append(queue, neighbor)
				}
			}
		}
		components = append(components, comp)
	}

	if len(components) <= 1 {
		return
	}

	// Sort components by size descending (largest first is main world)
	sort.Slice(components, func(i, j int) bool {
		return len(components[i]) > len(components[j])
	})

	// Find SE 7 (Teleport / room-over-room linkage) sprites
	type se7Sprite struct {
		secNum int
		hiTag  int16
		x, y   int32
	}
	var se7List []se7Sprite
	for _, sp := range a.level.Sprites {
		if sp.PicNum == 1 && sp.LoTag == 7 {
			se7List = append(se7List, se7Sprite{
				secNum: int(sp.SectNum),
				hiTag:  sp.HiTag,
				x:      sp.X,
				y:      sp.Y,
			})
		}
	}

	placedSectors := make(map[int]bool)
	for _, sIdx := range components[0] {
		placedSectors[sIdx] = true
	}

	placedComponents := make(map[int]bool)
	placedComponents[0] = true

	// Iteratively match sub-components to already placed components
	for {
		progress := false
		for cIdx := 1; cIdx < len(components); cIdx++ {
			if placedComponents[cIdx] {
				continue
			}
			comp := components[cIdx]
			compSet := make(map[int]bool, len(comp))
			for _, sIdx := range comp {
				compSet[sIdx] = true
			}

			// Look for an SE 7 in this component matching an SE 7 in placed sectors
			var matchFound bool
			var shiftX, shiftY int32

			for _, subSp := range se7List {
				if !compSet[subSp.secNum] {
					continue
				}
				for _, placedSp := range se7List {
					if !placedSectors[placedSp.secNum] {
						continue
					}
					if subSp.hiTag == placedSp.hiTag {
						shiftX = placedSp.x - subSp.x
						shiftY = placedSp.y - subSp.y
						matchFound = true
						break
					}
				}
				if matchFound {
					break
				}
			}

			if matchFound {
				// Translate all walls in this component
				for _, sIdx := range comp {
					sec := a.level.Sectors[sIdx]
					wStart := int(sec.WallPtr)
					wCount := int(sec.WallNum)
					for i := 0; i < wCount; i++ {
						if wStart+i < len(a.level.Walls) {
							a.level.Walls[wStart+i].X += shiftX
							a.level.Walls[wStart+i].Y += shiftY
						}
					}
				}

				// Translate sprites in this component
				for i := range a.level.Sprites {
					if compSet[int(a.level.Sprites[i].SectNum)] {
						a.level.Sprites[i].X += shiftX
						a.level.Sprites[i].Y += shiftY
					}
				}

				// If player starts in this component, translate player start
				if compSet[int(a.level.Header.CurSectNum)] {
					a.level.Header.PosX += shiftX
					a.level.Header.PosY += shiftY
				}

				// Check for artificial skybox sectors (e.g. sec 310 in E1L1) enclosing this island
				for _, sIdx := range comp {
					sec := a.level.Sectors[sIdx]
					if sec.IsSkyFloor() && sec.IsSkyCeiling() {
						wStart := int(sec.WallPtr)
						wCount := int(sec.WallNum)
						allInComp := true
						for i := 0; i < wCount; i++ {
							if wStart+i < len(a.level.Walls) {
								ns := int(a.level.Walls[wStart+i].NextSector)
								if ns >= 0 && !compSet[ns] {
									allInComp = false
									break
								}
							}
						}
						// If this sector only connects inward to the sub-component
						if allInComp && wCount <= 16 {
							// Convert interior walls pointing to this skybox into 1-sided exterior walls
							for wi := 0; wi < len(a.level.Walls); wi++ {
								if int(a.level.Walls[wi].NextSector) == sIdx {
									a.level.Walls[wi].NextSector = -1
									// Copy texture from reverse wall if current wall PicNum is 0
									nw := int(a.level.Walls[wi].NextWall)
									if a.level.Walls[wi].PicNum == 0 && nw >= 0 && nw < len(a.level.Walls) {
										if a.level.Walls[nw].PicNum != 0 {
											a.level.Walls[wi].PicNum = a.level.Walls[nw].PicNum
										}
									}
									a.level.Walls[wi].NextWall = -1
								}
							}
							// Zero WallNum so the artificial skybox sector won't be exported
							a.level.Sectors[sIdx].WallNum = 0
							fmt.Printf("Duke3D Assembler: stripped artificial skybox sector %d\n", sIdx)
						}
					}
				}

				for _, sIdx := range comp {
					placedSectors[sIdx] = true
				}
				placedComponents[cIdx] = true
				progress = true
				fmt.Printf("Duke3D Assembler: aligned sub-component %d (%d sectors) with shift (%d, %d)\n", cIdx, len(comp), shiftX, shiftY)
			}
		}
		if !progress {
			break
		}
	}
}
