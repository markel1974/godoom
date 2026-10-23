package jedi

import (
	"fmt"
	"math"
	"strings"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/generators/jedi/lumps"
	"github.com/markel1974/godoom/mr_tech/geometry"
)

// World represents the primary context for managing game state, including archives and engine mode configuration.
type World struct {
	archive lumps.IArchive
	eMode   EngineMode
}

// NewWorld initializes and returns a new World instance using the provided game archive and engine mode.
func NewWorld(archive lumps.IArchive, eMode EngineMode) *World {
	w := &World{
		archive: archive,
		eMode:   eMode,
	}
	return w
}

// Build constructs the configuration for sectors, vertices, and lights based on the level data in the current context.
func (b *World) Build() ([]*config.Sector, geometry.Polygon, []*config.Light, error) {
	level := b.archive.GetLevel()
	validTexture := func(idx int) bool {
		return idx >= 0 && !b.isDefaultTexture(level.GetTexture(idx))
	}

	configSectors := make([]*config.Sector, 0, len(level.Sectors))

	totalVertices := 0
	for _, sec := range level.Sectors {
		totalVertices += len(sec.Walls)
	}

	globalVertices := make(geometry.Polygon, 0, totalVertices)
	var lights []*config.Light

	type Adjoin struct {
		Lower *lumps.Sector
		Upper *lumps.Sector
	}

	var adjoins []Adjoin

	for _, sector := range level.Sectors {
		if len(sector.Id) == 0 {
			continue
		}
		secFloorY, secCeilY := sector.Compute(level)
		lightLevel, lightFalloff := b.computeLight(sector)

		cSector := config.NewConfigSector(sector.Id, lightLevel, config.LightKindAmbient, lightFalloff)
		cSector.FloorY = -secFloorY * scaleSectorH
		cSector.CeilY = -secCeilY * scaleSectorH

		if sector.FloorTexture >= 0 {
			texName := level.GetTexture(sector.FloorTexture)
			names, _ := b.archive.AddTexture(texName)
			cSector.Floor = config.NewConfigMaterial(names, config.MaterialKindLoop, scaleTextureW, scaleTextureH, 0, 0)
		} else {
			fmt.Println("MISSING FLOOR_TEXTURE", sector.Id)
		}

		if sector.CeilingTexture >= 0 {
			texName := level.GetTexture(sector.CeilingTexture)
			names, _ := b.archive.AddTexture(texName)
			animKind := config.MaterialKindLoop
			if sector.IsSky() {
				animKind = config.MaterialKindSky
				cSector.Light.Kind = config.LightKindOpenAir
			}
			cSector.Ceil = config.NewConfigMaterial(names, animKind, scaleTextureW, scaleTextureH, 0, 0)
		} else {
			fmt.Println("MISSING CEILING_TEXTURE", sector.Id)
		}

		if len(sector.Walls) > 0 {
			cSector.Segments = make([]*config.Segment, len(sector.Walls))
			for wallIdx, wall := range sector.Walls {
				if !wall.ValidWallVertices(len(sector.Vertices)) {
					fmt.Println("INVALID VERTEX", sector.Id, wallIdx)
					continue
				}
				v1 := sector.Vertices[wall.LeftVertex]
				v2 := sector.Vertices[wall.RightVertex]
				globalVertices = append(globalVertices, v1, v2)

				cSeg := config.NewConfigSegment(sector.Id, config.SegmentUnknown, v1, v2)
				cSector.Segments[wallIdx] = cSeg

				if sector.HasSlopedFloor && wallIdx == sector.BestFloorWall {
					cSeg.SlopedFloorRef = true
				}
				if sector.HasSlopedCeil && wallIdx == sector.BestCeilWall {
					cSeg.SlopedCeilingRef = true
				}

				if wall.Adjoin < 0 {
					texIdx := wall.MidTexture
					if validTexture(texIdx) {
						texName := level.GetTexture(texIdx)
						names, _ := b.archive.AddTexture(texName)
						cSeg.Kind = config.SegmentWall
						cSeg.Middle = config.NewConfigMaterial(names, config.MaterialKindLoop, scaleTextureW, scaleTextureH, 0, 0)
					} else {
						fmt.Printf("WARNING MISSING WALL_TEXTURE sector=%s wall=%d mid=%d top=%d bot=%d\n", sector.Id, wallIdx, wall.MidTexture, wall.TopTexture, wall.BotTexture)
					}
				} else {
					targetSecIdx := wall.Adjoin
					adjacent := level.GetSector(targetSecIdx)
					if adjacent == nil {
						fmt.Println("INVALID ADJOIN", sector.Id, wallIdx)
						continue
					}
					if wall.Adjoin >= 0 && wall.DAdjoin >= 0 {
						upperSec := level.GetSector(wall.Adjoin)
						lowerSec := level.GetSector(wall.DAdjoin)
						if upperSec == nil || lowerSec == nil {
							fmt.Println("INVALID ADJOIN/DADJOIN", wall.Adjoin)
							continue
						}
						adjoins = append(adjoins, Adjoin{Lower: lowerSec, Upper: upperSec})
						if math.Abs(sector.FloorY-lowerSec.FloorY) <= math.Abs(sector.FloorY-upperSec.FloorY) {
							targetSecIdx = wall.DAdjoin
							adjacent = lowerSec
							upFloorY, _ := upperSec.Compute(level)
							cSeg.HasUpperLimitZ = true
							cSeg.UpperLimitZ = -upFloorY * scaleSectorH
						}
					}

					topTexIdx := wall.TopTexture
					if wall.DAdjoin >= 0 && !validTexture(topTexIdx) && validTexture(wall.MidTexture) {
						topTexIdx = wall.MidTexture
					}

					if validTexture(topTexIdx) {
						texName := level.GetTexture(topTexIdx)
						names, _ := b.archive.AddTexture(texName)
						cSeg.Upper = config.NewConfigMaterial(names, config.MaterialKindLoop, scaleTextureW, scaleTextureH, 0, 0)
					} else if sector.IsSky() && adjacent.IsSky() {
						cSeg.Upper = config.NewConfigMaterial(nil, config.MaterialKindSky, scaleTextureW, scaleTextureH, 0, 0)
					}

					botTexIdx := wall.BotTexture
					if validTexture(botTexIdx) {
						texName := level.GetTexture(botTexIdx)
						names, _ := b.archive.AddTexture(texName)
						cSeg.Lower = config.NewConfigMaterial(names, config.MaterialKindLoop, scaleTextureW, scaleTextureH, 0, 0)
					}

					// Texture middle solo per portali con flag esplicito.
					if wall.GetFlags(1) {
						if validTexture(wall.MidTexture) {
							texName := level.GetTexture(wall.MidTexture)
							names, _ := b.archive.AddTexture(texName)
							cSeg.Middle = config.NewConfigMaterial(names, config.MaterialKindLoop, scaleTextureW, scaleTextureH, 0, 0)
						}
					}
				}
			}
		}

		if sector.HasSlopedFloor {
			cSector.SlopedFloorGradient = sector.FloorGrad
			cSector.SlopedFloorGX = sector.FloorGX
			cSector.SlopedFloorGY = sector.FloorGY
		}

		if sector.HasSlopedCeil {
			cSector.SlopedCeilingGradient = sector.CeilGrad
			cSector.SlopedCeilingGX = sector.CeilGX
			cSector.SlopedCeilingGY = sector.CeilGY
		}

		configSectors = append(configSectors, cSector)
	}

	if len(adjoins) > 0 {
		secMap := make(map[string]*config.Sector)
		for _, cs := range configSectors {
			secMap[cs.Id] = cs
		}
		for _, adj := range adjoins {
			if cs, ok := secMap[adj.Lower.Id]; ok {
				cs.AddDependency(adj.Upper.Id)
			} else {
				fmt.Println("sector not found in map:", adj.Lower.Id)
			}
		}
	}

	configSectors = b.reorderSectorsForDAdjoin(configSectors)

	return configSectors, globalVertices, lights, nil
}

// reorderSectorsForDAdjoin topologically sorts configSectors so that for any sector
// with dependencies in DependsOn, that sector appears after all sectors it depends on.
// This ensures that for stacked portals (DAdjoin), lower sectors appear after upper
// sectors in configSectors, allowing compiler segment adjacency resolution to correctly
// pair exterior portals with ground-level sectors rather than roofs.
// TODO MOVE IN COMPILER
func (b *World) reorderSectorsForDAdjoin(configSectors []*config.Sector) []*config.Sector {
	n := len(configSectors)
	if n == 0 {
		return configSectors
	}

	idToOrigIdx := make(map[string]int, n)
	for i, cs := range configSectors {
		idToOrigIdx[cs.Id] = i
	}

	adj := make(map[int][]int)
	inDegree := make([]int, n)

	for lowIdx, cs := range configSectors {
		for _, upId := range cs.DependsOn {
			if upIdx, ok := idToOrigIdx[upId]; ok && upIdx != lowIdx {
				exists := false
				for _, existing := range adj[upIdx] {
					if existing == lowIdx {
						exists = true
						break
					}
				}
				if !exists {
					adj[upIdx] = append(adj[upIdx], lowIdx)
					inDegree[lowIdx]++
				}
			}
		}
	}

	result := make([]*config.Sector, 0, n)
	ready := make([]bool, n)
	for i := 0; i < n; i++ {
		if inDegree[i] == 0 {
			ready[i] = true
		}
	}

	for len(result) < n {
		nextIdx := -1
		for i := 0; i < n; i++ {
			if ready[i] {
				nextIdx = i
				break
			}
		}
		if nextIdx == -1 {
			// In case of a cycle, append any remaining sectors preserving their relative order.
			for i := 0; i < n; i++ {
				if inDegree[i] > 0 {
					result = append(result, configSectors[i])
				}
			}
			break
		}
		ready[nextIdx] = false
		inDegree[nextIdx] = -1
		result = append(result, configSectors[nextIdx])
		for _, child := range adj[nextIdx] {
			inDegree[child]--
			if inDegree[child] == 0 {
				ready[child] = true
			}
		}
	}

	return result
}

// computeLight computes the light level and falloff for a given sector based on its properties and scaling factors.
func (b *World) computeLight(sector *lumps.Sector) (float64, float64) {
	lightLevel := sector.LightLevel * scaleLight
	if lightLevel < 2.2 {
		lightLevel = 2.2
	}
	lightFalloff := lightLevel * scaleLightFalloff
	return lightLevel, lightFalloff
}

// isDefaultTexture checks if the given texture name is empty or matches the default texture names "DEFAULT.PCX" or "DEFAULT.BM".
func (b *World) isDefaultTexture(texName string) bool {
	return len(texName) == 0 || strings.EqualFold(texName, "DEFAULT.PCX") || strings.EqualFold(texName, "DEFAULT.BM")
}
