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
		if idx < 0 {
			fmt.Printf("invalid texture index: %d", idx)
			return false
		}
		tex := level.GetTexture(idx)
		return !b.isDefaultTexture(tex)
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

		var vFirst1, vFirst2 geometry.XY
		if len(sector.Walls) > 0 && sector.Walls[0].ValidWallVertices(len(sector.Vertices)) {
			vFirst1 = sector.Vertices[sector.Walls[0].LeftVertex]
			vFirst2 = sector.Vertices[sector.Walls[0].RightVertex]
		}
		scalePlane := 8.0 / scaleFactor
		fPanning := geometry.XY{X: sector.FloorOffsetX * 8.0, Y: sector.FloorOffsetY * 8.0}
		cSector.FloorAlign = config.NewUVAlignSurface(false, vFirst1, vFirst2, false, false, false, fPanning, scalePlane, scalePlane)

		cPanning := geometry.XY{X: sector.CeilOffsetX * 8.0, Y: sector.CeilOffsetY * 8.0}
		cSector.CeilAlign = config.NewUVAlignSurface(false, vFirst1, vFirst2, false, false, false, cPanning, scalePlane, scalePlane)

		if sector.FloorTexture >= 0 {
			texName := level.GetTexture(sector.FloorTexture)
			names, _ := b.archive.AddTexture(texName)
			animKind := config.MaterialKindLoop
			if b.isLiquid(texName) || sector.IsWater() {
				animKind = config.MaterialKindLiquid
			}
			cSector.Floor = config.NewConfigMaterial(names, animKind, 0, 0)
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
			} else if b.isLiquid(texName) {
				animKind = config.MaterialKindLiquid
			}
			cSector.Ceil = config.NewConfigMaterial(names, animKind, 0, 0)
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

				wallLenDFU := math.Hypot(v2.X-v1.X, v2.Y-v1.Y)
				wallTexLen := wallLenDFU * 8.0

				var midPanning, topPanning, botPanning geometry.XY
				if validTexture(wall.MidTexture) {
					midPanning = geometry.XY{X: wall.MidOffsetX * 8.0, Y: wall.MidOffsetY * 8.0}
				}
				if validTexture(wall.TopTexture) {
					topPanning = geometry.XY{X: wall.TopOffsetX * 8.0, Y: wall.TopOffsetY * 8.0}
				}
				if validTexture(wall.BotTexture) {
					botPanning = geometry.XY{X: wall.BotOffsetX * 8.0, Y: wall.BotOffsetY * 8.0}
				}

				if wall.Adjoin < 0 {
					texIdx := wall.MidTexture
					if validTexture(texIdx) {
						texName := level.GetTexture(texIdx)
						names, _ := b.archive.AddTexture(texName)
						cSeg.Kind = config.SegmentWall
						animKind := config.MaterialKindLoop
						if b.isLiquid(texName) {
							animKind = config.MaterialKindLiquid
						}
						cSeg.Middle = config.NewConfigMaterial(names, animKind, 0, 0)
					}
					//else {
					//	fmt.Printf("WARNING MISSING WALL_TEXTURE sector=%s wall=%d mid=%d top=%d bot=%d\n", sector.Id, wallIdx, wall.MidTexture, wall.TopTexture, wall.BotTexture)
					//}
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
							cSeg.HasUpperLimitZ = true
							cSeg.UpperLimitZS = upperSec.EvaluateFloorZ(level, v1, scaleSectorH)
							cSeg.UpperLimitZE = upperSec.EvaluateFloorZ(level, v2, scaleSectorH)
						}
					}

					topTexIdx := wall.TopTexture
					if wall.DAdjoin >= 0 && !validTexture(topTexIdx) && validTexture(wall.MidTexture) {
						topTexIdx = wall.MidTexture
						topPanning = midPanning
					} else if wall.TopOffsetX == 0 && wall.TopOffsetY == 0 && wall.TopTexture == wall.MidTexture && validTexture(wall.MidTexture) {
						topPanning = midPanning
					}

					if adjacent != nil {
						_, adjCeilY := adjacent.Compute(level)
						lintel := -adjCeilY
						floor := -secFloorY
						if wall.TopOffsetY != 0 && lintel > floor && wall.TopOffsetY >= (lintel-floor) {
							effTopOffsetY := wall.TopOffsetY - (lintel - floor)
							topPanning.Y = effTopOffsetY * 8.0
						}
					}

					if validTexture(topTexIdx) {
						texName := level.GetTexture(topTexIdx)
						names, _ := b.archive.AddTexture(texName)
						animKind := config.MaterialKindLoop
						if b.isLiquid(texName) {
							animKind = config.MaterialKindLiquid
						}
						cSeg.Upper = config.NewConfigMaterial(names, animKind, 0, 0)
					} else if sector.IsSky() && adjacent.IsSky() {
						cSeg.Upper = config.NewConfigMaterial(nil, config.MaterialKindSky, 0, 0)
					}

					botTexIdx := wall.BotTexture
					if validTexture(botTexIdx) {
						texName := level.GetTexture(botTexIdx)
						names, _ := b.archive.AddTexture(texName)
						animKind := config.MaterialKindLoop
						if b.isLiquid(texName) {
							animKind = config.MaterialKindLiquid
						}
						cSeg.Lower = config.NewConfigMaterial(names, animKind, 0, 0)
					}

					// Texture middle solo per portali con flag esplicito.
					if wall.GetFlags(1) {
						if validTexture(wall.MidTexture) {
							texName := level.GetTexture(wall.MidTexture)
							names, _ := b.archive.AddTexture(texName)
							animKind := config.MaterialKindLoop
							if b.isLiquid(texName) {
								animKind = config.MaterialKindLiquid
							}
							cSeg.Middle = config.NewConfigMaterial(names, animKind, 0, 0)
						}
					}
				}

				cSeg.UVAlign = config.NewUVAlignVertical(false, false, false, midPanning, wallTexLen, scaleSectorW)
				cSeg.UVAlign.SetRefZ(cSector.CeilY)
				cSeg.UVAlignUpper = config.NewUVAlignVertical(false, false, false, topPanning, wallTexLen, scaleSectorW)
				cSeg.UVAlignUpper.SetRefZ(cSector.CeilY)
				cSeg.UVAlignLower = config.NewUVAlignVertical(false, false, false, botPanning, wallTexLen, scaleSectorW)
				cSeg.UVAlignLower.SetRefZ(cSector.CeilY)

				if sector.HasSlopedFloor && wallIdx == sector.BestFloorWall {
					cSeg.SlopedFloorRef = true
				}
				if sector.HasSlopedCeil && wallIdx == sector.BestCeilWall {
					cSeg.SlopedCeilingRef = true
				}
			}
		}

		if sector.HasSlopedFloor {
			cSector.SlopeFloor.Gradient = sector.FloorGrad
			cSector.SlopeFloor.G.X = sector.FloorGX
			cSector.SlopeFloor.G.Y = sector.FloorGY
		}

		if sector.HasSlopedCeil {
			cSector.SlopeCeiling.Gradient = sector.CeilGrad
			cSector.SlopeCeiling.G.X = sector.CeilGX
			cSector.SlopeCeiling.G.Y = sector.CeilGY
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

	//configSectors = b.reorderSectorsForDAdjoin(configSectors)

	return configSectors, globalVertices, lights, nil
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

// isLiquid checks if a texture name corresponds to liquid / water / slime / lava.
func (b *World) isLiquid(texName string) bool {
	u := strings.ToUpper(strings.TrimSpace(texName))
	return strings.Contains(u, "WATER") ||
		strings.Contains(u, "EXWAT") ||
		strings.Contains(u, "LAVA") ||
		strings.Contains(u, "SLIME") ||
		strings.Contains(u, "WFALL") ||
		strings.Contains(u, "POOL") ||
		strings.Contains(u, "RIVER") ||
		strings.Contains(u, "STREAM")
}
