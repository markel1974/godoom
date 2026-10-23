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

	for _, sector := range level.Sectors {
		if len(sector.Id) == 0 {
			continue
		}

		secFloorY := sector.FloorY
		secCeilY := sector.CeilingY

		if flag, ok := sector.GetFlag(1, 0x80000000); ok && flag != 0 {
			bestFloorY := sector.FloorY
			foundAdjacent := false
			for _, wall := range sector.Walls {
				if wall.Adjoin < 0 ||
					wall.Adjoin >= len(level.Sectors) {
					continue
				}
				adjacent := level.Sectors[wall.Adjoin]
				if !foundAdjacent || adjacent.FloorY > bestFloorY {
					bestFloorY = adjacent.FloorY
					foundAdjacent = true
				}
			}
			if foundAdjacent {
				secFloorY = bestFloorY
			}
		}

		bestFloorWall, floorGrad, floorGX, floorGY, floorQuota, hasSlopedFloor := b.computePlane(level, sector, false)

		if hasSlopedFloor {
			secFloorY = floorQuota
		}

		bestCeilWall, ceilGrad, ceilGX, ceilGY, ceilQuota, hasSlopedCeil := b.computePlane(level, sector, true)

		if hasSlopedCeil {
			secCeilY = ceilQuota
		}

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
				if !b.validWallVertices(sector, wall) {
					fmt.Println("INVALID VERTEX", sector.Id, wallIdx)
					continue
				}

				v1 := sector.Vertices[wall.LeftVertex]
				v2 := sector.Vertices[wall.RightVertex]

				globalVertices = append(globalVertices, v1, v2)

				cSeg := config.NewConfigSegment(sector.Id, config.SegmentUnknown, v1, v2)

				if hasSlopedFloor && wallIdx == bestFloorWall {
					cSeg.SlopedFloorRef = true
				}

				if hasSlopedCeil && wallIdx == bestCeilWall {
					cSeg.SlopedCeilingRef = true
				}

				if wall.Adjoin == -1 {
					texIdx := wall.MidTexture

					if !validTexture(texIdx) {
						switch {
						case validTexture(wall.TopTexture):
							texIdx = wall.TopTexture
						case validTexture(wall.BotTexture):
							texIdx = wall.BotTexture
						default:
							fmt.Printf("WARNING MISSING WALL_TEXTURE sector=%s wall=%d mid=%d top=%d bot=%d\n", sector.Id, wallIdx, wall.MidTexture, wall.TopTexture, wall.BotTexture)
							cSeg.Kind = config.SegmentUnknown
							cSector.Segments[wallIdx] = cSeg
							continue
						}
					}

					texName := level.GetTexture(texIdx)
					names, _ := b.archive.AddTexture(texName)

					cSeg.Kind = config.SegmentWall
					cSeg.Middle = config.NewConfigMaterial(names, config.MaterialKindLoop, scaleTextureW, scaleTextureH, 0, 0)
				} else {
					targetSecIdx := wall.Adjoin

					if targetSecIdx < 0 ||
						targetSecIdx >= len(level.Sectors) {
						fmt.Println("INVALID ADJOIN", sector.Id, wallIdx)
						continue
					}

					adjacent := level.Sectors[targetSecIdx]

					if wall.DAdjoin >= 0 && wall.DAdjoin < len(level.Sectors) {
						upperSec := level.Sectors[wall.Adjoin]
						lowerSec := level.Sectors[wall.DAdjoin]
						if math.Abs(sector.FloorY-lowerSec.FloorY) <=
							math.Abs(sector.FloorY-upperSec.FloorY) {
							targetSecIdx = wall.DAdjoin
							adjacent = lowerSec
						}
					}

					if sector.IsSky() && adjacent.IsSky() {
						cSeg.Upper = config.NewConfigMaterial(nil, config.MaterialKindSky, scaleTextureW, scaleTextureH, 0, 0)
					} else if wall.DAdjoin >= 0 && sector.IsSky() {
						cSeg.Upper = config.NewConfigMaterial(nil, config.MaterialKindSky, scaleTextureW, scaleTextureH, 0, 0)
					} else if !sector.IsSky() || !adjacent.IsSky() {
						topTexIdx := wall.TopTexture
						if validTexture(topTexIdx) {
							texName := level.GetTexture(topTexIdx)
							names, _ := b.archive.AddTexture(texName)
							cSeg.Upper = config.NewConfigMaterial(names, config.MaterialKindLoop, scaleTextureW, scaleTextureH, 0, 0)
						}
					}

					botTexIdx := wall.BotTexture
					if !validTexture(botTexIdx) {
						if b.eMode == EngineModeOutlaws && wall.Flags&1 == 0 && validTexture(wall.MidTexture) {
							adjoinSec := level.Sectors[targetSecIdx]
							floorDiff := sector.FloorY - adjoinSec.FloorY
							if floorDiff > 0.05 && floorDiff <= 5.0 {
								botTexIdx = wall.MidTexture
							}
						}
					}
					if validTexture(botTexIdx) {
						texName := level.GetTexture(botTexIdx)
						names, _ := b.archive.AddTexture(texName)
						cSeg.Lower = config.NewConfigMaterial(names, config.MaterialKindLoop, scaleTextureW, scaleTextureH, 0, 0)
					}

					// Texture middle solo per portali con flag esplicito.
					if wall.Flags&1 != 0 && validTexture(wall.MidTexture) {
						texName := level.GetTexture(wall.MidTexture)
						names, _ := b.archive.AddTexture(texName)
						cSeg.Middle = config.NewConfigMaterial(names, config.MaterialKindLoop, scaleTextureW, scaleTextureH, 0, 0)
					}
				}

				cSector.Segments[wallIdx] = cSeg
			}
		}

		if hasSlopedFloor {
			cSector.SlopedFloorGradient = floorGrad
			cSector.SlopedFloorGX = floorGX
			cSector.SlopedFloorGY = floorGY
		}

		if hasSlopedCeil {
			cSector.SlopedCeilingGradient = ceilGrad
			cSector.SlopedCeilingGX = ceilGX
			cSector.SlopedCeilingGY = ceilGY
		}

		configSectors = append(configSectors, cSector)
	}

	var secMap map[string]*config.Sector
	for _, sec := range level.Sectors {
		for _, w := range sec.Walls {
			if w.DAdjoin >= 0 && w.Adjoin >= 0 &&
				w.DAdjoin < len(level.Sectors) && w.Adjoin < len(level.Sectors) {
				upperSec := level.Sectors[w.Adjoin]
				lowerSec := level.Sectors[w.DAdjoin]
				if secMap == nil {
					secMap = make(map[string]*config.Sector)
					for _, cs := range configSectors {
						secMap[cs.Id] = cs
					}
				}
				if cs, ok := secMap[lowerSec.Id]; ok {
					cs.AddDependency(upperSec.Id)
				}
			}
		}
	}
	//if secMap != nil {
	//	b.closeRoofGaps(level, secMap)
	//	}
	configSectors = b.reorderSectorsForDAdjoin(configSectors)

	return configSectors, globalVertices, lights, nil
}

// closeRoofGaps removes the thin slab left between the ceiling of a lower sector (room, porch) and the
// floor of the upper sector stacked on it through a double adjoin (roof). The ceiling of the lower sector is
// aligned to the roof plane (when the two planes are parallel), so that the perimeter walls join the roof exactly.
func (b *World) closeRoofGaps(level *lumps.Level, secMap map[string]*config.Sector) {
	//COMPLETAMENTE SBAGLIATO FARE QUESTO
	const maxGap = 1.0 * scaleSectorH

	floorRef := func(cs *config.Sector) (geometry.XY, bool) {
		for _, seg := range cs.Segments {
			if seg != nil && seg.SlopedFloorRef {
				return seg.Start, true
			}
		}
		return geometry.XY{}, false
	}
	ceilRef := func(cs *config.Sector) (geometry.XY, bool) {
		for _, seg := range cs.Segments {
			if seg != nil && seg.SlopedCeilingRef {
				return seg.Start, true
			}
		}
		return geometry.XY{}, false
	}

	done := make(map[string]bool)
	for _, sec := range level.Sectors {
		for _, w := range sec.Walls {
			if w.DAdjoin < 0 || w.Adjoin < 0 || w.DAdjoin >= len(level.Sectors) || w.Adjoin >= len(level.Sectors) {
				continue
			}
			cUp, okUp := secMap[level.Sectors[w.Adjoin].Id]
			cLow, okLow := secMap[level.Sectors[w.DAdjoin].Id]
			if !okUp || !okLow || done[cLow.Id] {
				continue
			}

			// Upper floor plane evaluated on the reference point of the lower ceiling plane.
			zRoof := cUp.FloorY
			if cUp.SlopedFloorGradient != 0 || cLow.SlopedCeilingGradient != 0 {
				if cUp.SlopedFloorGradient == 0 || cLow.SlopedCeilingGradient == 0 {
					continue
				}
				if math.Abs(cUp.SlopedFloorGX-cLow.SlopedCeilingGX) > 0.01 || math.Abs(cUp.SlopedFloorGY-cLow.SlopedCeilingGY) > 0.01 {
					continue
				}
				pu, ok1 := floorRef(cUp)
				pl, ok2 := ceilRef(cLow)
				if !ok1 || !ok2 {
					continue
				}
				zRoof = cUp.FloorY + cUp.SlopedFloorGX*(pl.X-pu.X) + cUp.SlopedFloorGY*(pl.Y-pu.Y)
			}

			gap := zRoof - cLow.CeilY
			if gap > 1e-6 && gap <= maxGap {
				cLow.CeilY = zRoof
				if cUp.SlopedFloorGradient != 0 {
					cLow.SlopedCeilingGradient = cUp.SlopedFloorGradient
					cLow.SlopedCeilingGX = cUp.SlopedFloorGX
					cLow.SlopedCeilingGY = cUp.SlopedFloorGY
				}
				done[cLow.Id] = true
			}
		}
	}
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

// validWallVertices checks if a wall's vertices are within the valid range of the sector's vertex list. Returns true if valid.
func (b *World) validWallVertices(sec *lumps.Sector, wall *lumps.Wall) bool {
	return wall.LeftVertex >= 0 &&
		wall.RightVertex >= 0 &&
		wall.LeftVertex < len(sec.Vertices) &&
		wall.RightVertex < len(sec.Vertices)
}

// computePlane calculates the slope plane of a sector ceiling or floor, returning optimal wall index, gradient, quota, and status.
func (b *World) computePlane(level *lumps.Level, sec *lumps.Sector, isCeil bool) (int, float64, float64, float64, float64, bool) {
	sl := sec.SlopedFloor
	if isCeil {
		sl = sec.SlopedCeiling
	}

	if sl == nil || sl.SectorIndex < 0 || sl.SectorIndex >= len(level.Sectors) {
		return -1, 0, 0, 0, 0, false
	}

	pSec := level.Sectors[sl.SectorIndex]
	if sl.WallIndex < 0 || sl.WallIndex >= len(pSec.Walls) {
		return -1, 0, 0, 0, 0, false
	}

	pWall := pSec.Walls[sl.WallIndex]
	if !b.validWallVertices(pSec, pWall) {
		return -1, 0, 0, 0, 0, false
	}

	pv1 := pSec.Vertices[pWall.LeftVertex]
	pv2 := pSec.Vertices[pWall.RightVertex]

	pdx := pv2.X - pv1.X
	pdy := pv2.Y - pv1.Y
	pLen := math.Hypot(pdx, pdy)
	if pLen == 0 {
		return -1, 0, 0, 0, 0, false
	}

	var pnX, pnY float64
	if pSec.IsCCW() {
		pnX, pnY = -pdy/pLen, pdx/pLen
	} else {
		pnX, pnY = pdy/pLen, -pdx/pLen
	}

	grad := sl.GetGradient()
	gx := pnX * grad
	gy := pnY * grad

	baseZ := pSec.FloorY
	if isCeil {
		baseZ = pSec.CeilingY
	}

	bestWall := -1
	bestAbsDot := -1.0
	bestDot := 0.0

	if sl.SectorIndex == sec.Index && b.validWallVertices(sec, pWall) {
		bestWall = sl.WallIndex
		bestDot = 1.0
	} else {
		for wIdx, w := range sec.Walls {
			if !b.validWallVertices(sec, w) {
				continue
			}

			v1 := sec.Vertices[w.LeftVertex]
			v2 := sec.Vertices[w.RightVertex]

			dx := v2.X - v1.X
			dy := v2.Y - v1.Y
			wLen := math.Hypot(dx, dy)
			if wLen < 1e-4 {
				continue
			}

			var wnX, wnY float64
			if sec.IsCCW() {
				wnX, wnY = -dy/wLen, dx/wLen
			} else {
				wnX, wnY = dy/wLen, -dx/wLen
			}

			dot := wnX*pnX + wnY*pnY
			if math.Abs(dot) > bestAbsDot {
				bestAbsDot = math.Abs(dot)
				bestDot = dot
				bestWall = wIdx
			}
		}
	}

	if bestWall < 0 {
		return -1, 0, 0, 0, 0, false
	}

	vStart := sec.Vertices[sec.Walls[bestWall].LeftVertex]
	quotaStart := baseZ -
		(gx*(vStart.X-pv1.X) +
			gy*(vStart.Y-pv1.Y))

	effGrad := grad
	if bestDot < 0 {
		effGrad = -grad
	}

	return bestWall, effGrad, gx, gy, quotaStart, true
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
