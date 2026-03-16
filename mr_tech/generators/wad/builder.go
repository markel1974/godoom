package wad

import (
	"fmt"
	"math"
	"strconv"

	"github.com/markel1974/godoom/mr_tech/generators/wad/geometry"
	"github.com/markel1974/godoom/mr_tech/generators/wad/lumps"
	"github.com/markel1974/godoom/mr_tech/model"
)

const TextureScaleW = 4.0
const TextureScaleH = 10.0

// ScaleFactorLineDef defines the scale factor applied to line definitions for coordinate normalization in the configuration.
const ScaleFactorLineDef = 25.0

// ScaleFactorCeilFloorLineDef is a constant scaling factor used to convert floor and ceiling heights into game unit measurements.
const ScaleFactorCeilFloorLineDef = 8.0

const SkyPicture = "F_SKY1"

// Builder provides utilities for constructing or processing line definitions within a WAD file.
type Builder struct {
}

// NewBuilder initializes and returns a new instance of Builder.
func NewBuilder() *Builder {
	return &Builder{}
}

// Setup initializes the level configuration by loading data from a WAD file and constructing sectors, player, and things.
func (bld *Builder) Setup(wadFile string, levelNumber int) (*model.ConfigRoot, error) {
	wad := New()
	if err := wad.Load(wadFile); err != nil {
		return nil, err
	}
	texHandler := wad.GetTextures()
	levelNames := wad.GetLevels()
	if levelNumber < 1 || levelNumber > len(levelNames) {
		return nil, fmt.Errorf("invalid level number: %d", levelNumber)
	}
	level, err := wad.GetLevel(levelNames[levelNumber-1])
	if err != nil {
		return nil, err
	}

	sectors := bld.buildSectorsFromLineDefs(level, texHandler)

	grid := NewSpatialGrid(sectors, 256.0)

	pX, pY, pAngle := float64(0), float64(0), float64(0)
	var things []*model.ConfigThing

	for i, t := range level.Things {
		tX := float64(t.X)
		tY := float64(t.Y)
		tAngle := float64(t.Angle)
		if t.Type == 1 || t.Type == 2 || t.Type == 3 || t.Type == 4 || t.Type == 11 {
			if t.Type == 1 {
				pX, pY, pAngle = tX, tY, tAngle
			}
			continue
		}
		// Risoluzione dei fotogrammi (Fallback su frame mancante)
		sd, hasAnim := _spriteDictionary[int(t.Type)]
		var frames []string
		if !hasAnim {
			fmt.Printf("WARNING No animation found for thing type %d, using default sprite\n", t.Type)
			frames = []string{"UNKNOWN"}
		} else {
			frames = sd.Sprites
		}
		tSectorId := grid.ResolveSectorId(geometry.Point{X: tX, Y: tY})
		tId := fmt.Sprintf("t_%d", i)
		anim := model.NewConfigAnimation(texHandler.SpriteCreateAnimation(frames), model.AnimationKindLoop, TextureScaleW/70, TextureScaleH/70)
		cfgThing := model.NewConfigThing(tId, model.XY{X: tX, Y: -tY}, tAngle, int(t.Type), tSectorId, sd.Mass, sd.Radius, sd.Height, anim)
		things = append(things, cfgThing)
	}

	playerSectorId := grid.ResolveSectorId(geometry.Point{X: pX, Y: pY})
	player := model.NewConfigPlayer(model.XY{X: pX, Y: -pY}, pAngle, playerSectorId, 20.0/radiusF, 100.0)

	return model.NewConfigRoot(sectors, player, things, ScaleFactorLineDef, true, texHandler), nil
}

// buildConfigSector converts a WAD sector to a ConfigSector, assigning texture, height, light level, and ID properties.
func (bld *Builder) buildConfigSector(level *Level, wadSector *lumps.Sector, texHandler *Textures, secIdx uint16, loopIdx int, triIdx int, edges []geometry.Edge) *model.ConfigSector {
	const openAllDoors = true
	sectorId := fmt.Sprintf("s%d_l%d_t%d", secIdx, loopIdx, triIdx)
	miSector := model.NewConfigSector(sectorId, bld.convertLight(wadSector.LightLevel), model.LightKindSpot)
	miSector.FloorY = float64(wadSector.FloorHeight) / ScaleFactorCeilFloorLineDef
	ceilHeight := float64(wadSector.CeilingHeight)
	if openAllDoors {
		ceilHeight = bld.calculateOpenDoorCeil(level, secIdx, wadSector, edges)
	}
	miSector.CeilY = ceilHeight / ScaleFactorCeilFloorLineDef
	miSector.Tag = strconv.Itoa(int(secIdx))
	ceilingType := model.AnimationKindLoop
	floorType := model.AnimationKindLoop
	if wadSector.CeilingPic == SkyPicture {
		ceilingType = model.AnimationKindSky
	}
	if wadSector.FloorPic == SkyPicture {
		floorType = model.AnimationKindSky
	}
	miSector.Ceil = model.NewConfigAnimation(texHandler.FlatCreateAnimation(wadSector.CeilingPic), ceilingType, TextureScaleW, TextureScaleH)
	miSector.Floor = model.NewConfigAnimation(texHandler.FlatCreateAnimation(wadSector.FloorPic), floorType, TextureScaleW, TextureScaleH)
	return miSector
}

// buildConfigSegment generates a ConfigSegment based on a level's geometry, sector ID, points, and sector edges.
// It identifies if a matching edge exists, adjusts Y-coordinates, sets texture details, and determines the segment kind.
func (bld *Builder) buildConfigSegment(level *Level, texHandler *Textures, sectorId string, p1, p2 geometry.Point, sectorEdges []geometry.Edge) (*model.ConfigSegment, bool) {
	mp1 := model.XY{X: p1.X, Y: p1.Y}
	mp2 := model.XY{X: p2.X, Y: p2.Y}

	seg := model.NewConfigSegment(sectorId, model.DefinitionWall, mp1, mp2, "")
	for _, e := range sectorEdges {
		v1, v2 := level.Vertexes[e.V1], level.Vertexes[e.V2]
		w1 := geometry.Point{X: float64(v1.XCoord), Y: float64(v1.YCoord)}
		w2 := geometry.Point{X: float64(v2.XCoord), Y: float64(v2.YCoord)}

		if (p1 == w1 && p2 == w2) || (p1 == w2 && p2 == w1) {
			ld := level.LineDefs[e.LDIdx]

			sideIdx := ld.SideDefRight
			if e.IsLeft {
				sideIdx = ld.SideDefLeft
			}
			side := level.SideDefs[sideIdx]

			middleT := texHandler.TextureCreateAnimation(side.MiddleTexture)
			upperT := texHandler.TextureCreateAnimation(side.UpperTexture)
			lowerT := texHandler.TextureCreateAnimation(side.LowerTexture)
			seg.Middle = model.NewConfigAnimation(middleT, model.AnimationKindLoop, TextureScaleW, TextureScaleH)
			seg.Upper = model.NewConfigAnimation(upperT, model.AnimationKindLoop, TextureScaleW, TextureScaleH)
			seg.Lower = model.NewConfigAnimation(lowerT, model.AnimationKindLoop, TextureScaleW, TextureScaleH)

			frontSector := level.Sectors[side.SectorRef]
			// SKY HACK VERTICALE:
			if ld.HasFlag(lumps.TwoSided) {
				backSideIdx := ld.SideDefLeft
				if e.IsLeft {
					backSideIdx = ld.SideDefRight
				}

				if backSideIdx != -1 {
					backSide := level.SideDefs[backSideIdx]
					backSector := level.Sectors[backSide.SectorRef]
					// If BOTH sectors have the ceiling set to sky, the upper wall is invisible/sky.
					if frontSector.CeilingPic == SkyPicture && backSector.CeilingPic == SkyPicture {
						seg.Upper = model.NewConfigAnimation(nil, model.AnimationKindNone, TextureScaleW, TextureScaleH)
					}
					// Extension for floors (e.g. moats that show sky at the bottom)
					if frontSector.FloorPic == SkyPicture && backSector.FloorPic == SkyPicture {
						seg.Lower = model.NewConfigAnimation(nil, model.AnimationKindNone, TextureScaleW, TextureScaleH)
					}
				}
			}

			if ld.HasFlag(2) {
				seg.Kind = model.DefinitionJoin
			}
			seg.Start.Y, seg.End.Y = -seg.Start.Y, -seg.End.Y
			return seg, true
		}
	}
	seg.Kind = model.DefinitionJoin
	seg.Start.Y, seg.End.Y = -seg.Start.Y, -seg.End.Y
	return seg, false
}

// convertLight normalizes a light level from the WAD file to a float between 0.0 and 1.0, or returns -1.0 for dim levels below 16.
func (bld *Builder) convertLight(lightLevel int16) float64 {
	rawLight := float64(lightLevel)
	// Soglia ambientale
	if rawLight < 16 {
		return -1.0
	}
	// Ritorna l'intensità lineare normalizzata
	return rawLight / 255.0
}

func (bld *Builder) calculateOpenDoorCeil(level *Level, secIdx uint16, wadSector *lumps.Sector, edges []geometry.Edge) float64 {
	isDoor := false
	lowestAdjCeil := int16(math.MaxInt16)
	hasAdjacent := false

	for _, e := range edges {
		ld := level.LineDefs[e.LDIdx]

		// 1. Check if the segment has a typical Doom door Action Special
		// Classic specials: 1 (DR), 26 (DR), 27, 28, 31 (D1), 32, 33, 34, 46, 117, 118
		special := ld.Function
		if special == 1 || special == 31 || special == 117 || special == 118 || special == 26 || special == 32 || special == 46 {
			isDoor = true
		}

		// 2. Navigate the segment sides to find the adjacent sector
		if ld.SideDefRight != -1 && ld.SideDefLeft != -1 {
			adjSideIdx := ld.SideDefRight
			if level.SideDefs[adjSideIdx].SectorRef == secIdx {
				adjSideIdx = ld.SideDefLeft
			}

			adjSectorRef := level.SideDefs[adjSideIdx].SectorRef
			adjSector := level.Sectors[adjSectorRef]

			if adjSector.CeilingHeight < lowestAdjCeil {
				lowestAdjCeil = adjSector.CeilingHeight
				hasAdjacent = true
			}
		}
	}

	// If it's a confirmed door (or a collapsed sector with valid adjacencies) we calculate the elevation
	if (isDoor && (wadSector.CeilingHeight <= wadSector.FloorHeight && hasAdjacent)) && lowestAdjCeil != math.MaxInt16 {
		targetCeil := lowestAdjCeil - 4
		if targetCeil < wadSector.FloorHeight {
			targetCeil = lowestAdjCeil // Emergency fallback if -4 penetrates the floor
		}
		return float64(targetCeil)
	}

	// Return the original height if it's not a door
	return float64(wadSector.CeilingHeight)
}

// buildSectorsFromLineDefs processes linedefs in a level to build and return a list of ConfigSector objects.
func (bld *Builder) buildSectorsFromLineDefs(level *Level, texHandler *Textures) []*model.ConfigSector {
	sectorToEdges := make(map[uint16][]geometry.Edge)
	for i, ld := range level.LineDefs {
		if ld.SideDefRight != -1 {
			s := level.SideDefs[ld.SideDefRight].SectorRef
			sectorToEdges[s] = append(sectorToEdges[s], geometry.Edge{V1: uint16(ld.VertexStart), V2: uint16(ld.VertexEnd), LDIdx: i, IsLeft: false})
		}
		if ld.SideDefLeft != -1 {
			s := level.SideDefs[ld.SideDefLeft].SectorRef
			sectorToEdges[s] = append(sectorToEdges[s], geometry.Edge{V1: uint16(ld.VertexEnd), V2: uint16(ld.VertexStart), LDIdx: i, IsLeft: true})
		}
	}

	var cSectors []*model.ConfigSector
	const quantize = 1000
	edgeMap := make(map[geometry.QuantizedEdgeKey]string)
	wadLines := make(map[*model.ConfigSegment]bool)

	for secIdx, edges := range sectorToEdges {
		wadSector := level.Sectors[secIdx]
		vertexes := make(geometry.Polygon, len(level.Vertexes))
		for idx, l := range level.Vertexes {
			vertexes[idx] = geometry.Point{X: float64(l.XCoord), Y: float64(l.YCoord)}
		}
		polygonDefs := vertexes.TraceLoops(edges, len(level.LineDefs))

		//if secIdx == 7 {
		//	fmt.Printf("Polygon: %v\n", edges)
		//}

		for loopIdx, def := range polygonDefs {
			mergedPoly := def.BridgeHoles()

			triangles := mergedPoly.Triangulate(int(secIdx))

			for triIdx, tri := range triangles {
				cSector := bld.buildConfigSector(level, wadSector, texHandler, secIdx, loopIdx, triIdx, edges)
				for k := 0; k < 3; k++ {
					p1 := tri[k]
					p2 := tri[(k+1)%3]
					cSeg, isWadLine := bld.buildConfigSegment(level, texHandler, cSector.Id, p1, p2, edges)
					cSector.Segments = append(cSector.Segments, cSeg)
					wadLines[cSeg] = isWadLine
					key := geometry.NewQuantizedEdgeKey(cSeg.Start.X, cSeg.Start.Y, cSeg.End.X, cSeg.End.Y, quantize)
					edgeMap[key] = cSeg.Parent
				}
				cSectors = append(cSectors, cSector)
			}
		}
	}

	for _, cf := range cSectors {
		for _, cs := range cf.Segments {
			if cs.Kind == model.DefinitionJoin {
				reverseKey := geometry.NewQuantizedEdgeKey(cs.End.X, cs.End.Y, cs.Start.X, cs.Start.Y, quantize)
				if neighborId, exists := edgeMap[reverseKey]; exists {
					cs.Neighbor = neighborId
				} else {
					fmt.Printf("WARNING: Missing edge for join segment: %s - %s (%v)\n", cs.Id, cs.Tag, reverseKey)
					// PROTECTION 2: Prevents Ghost Walls!
					isWadLine := wadLines[cs]
					if isWadLine {
						cs.Kind = model.DefinitionWall
					} else {
						cs.Neighbor = cs.Parent
					}
				}
			}
		}
	}

	return cSectors
}
