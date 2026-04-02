package wad

import (
	"fmt"
	"math"
	"strconv"

	"github.com/markel1974/godoom/mr_tech/generators/wad/geometry"
	"github.com/markel1974/godoom/mr_tech/generators/wad/lumps"
	"github.com/markel1974/godoom/mr_tech/model"
)

// TextureScaleW defines the horizontal texture scaling factor used when creating texture animations or configurations.
const TextureScaleW = 4.0

// TextureScaleH defines the horizontal scale factor for texture mapping, used in rendering and configuration calculations.
const TextureScaleH = 10.0

// ScaleFactorLineDef defines the default scaling factor applied to line definitions in the level configuration.
const ScaleFactorLineDef = 25.0

// ScaleFactorCeilFloorLineDef defines the scale factor used for converting ceiling and floor heights in level configurations.
const ScaleFactorCeilFloorLineDef = 8.0

// SkyPicture represents a constant string identifier for the default sky texture in the level configuration.
const SkyPicture = "F_SKY1"

// _doors is a map where keys represent door action codes, and values describe their associated behavior or requirements.
var _doors = map[int16]string{
	1:   "DR	Door Open, Wait, Close",
	2:   "W1	Door Stay Open",
	3:   "W1	Door Close",
	4:   "W1	Door",
	16:  "W1	Door Close and Open",
	26:  "DR	Door Blue Key",
	27:  "DR	Door Yellow Key",
	28:  "DR	Door Red Key",
	29:  "S1	Door",
	31:  "D1	Door Stay Open",
	32:  "D1	Door Blue Key",
	33:  "D1	Door Red Key",
	34:  "D1	Door Yellow Key",
	42:  "SR	Door Close",
	46:  "GR	Door Also Monsters",
	50:  "S1	Door Close",
	63:  "SR	Door",
	117: "GR	Door Wait Raise Fast",
	118: "GR	Door Wait Close Fast",
	150: "S1	Door Wait Raise",
	151: "S1	Door Close Wait Open",
	175: "S1	Door Close and Open",
	196: "SR	Door Close then Open",
	197: "SR	Door Wait Close",
	198: "SR	Door Raise",
	199: "SR	Door Wait Raise",
	200: "SR	Door Close Wait Open",
	201: "SR	Door Wait Raise Silent",
	202: "SR	Door Wait Raise Fast",
	203: "SR	Door Wait Close Fast",
	204: "SR	Door Raise Fast",
	205: "SR	Door Close Fast",
	206: "SR	Door Open Fast",
	207: "SR	Door Close Wait Open Fast",
	323: "UNK   Door Raise (fast 150)",
	324: "UNK   Door Close (fast 150)",
	325: "UNK   Door Raise (slow 300)",
	326: "UNK   Door Close (slow 300)",
	327: "UNK   Door Closest (fast 150)",
	328: "UNK   Door Closest (slow 300)",
	329: "UNK   Door Locked Raise",
	330: "UNK   Door Locked Closest",
}

// Builder is a type responsible for constructing and configuring level data structures from resource files.
type Builder struct {
}

// NewBuilder creates and returns a new instance of the Builder type.
func NewBuilder() *Builder {
	return &Builder{}
}

// Setup initializes the game level configuration by loading WAD file data, processing textures, sectors, and player setup.
func (bld *Builder) Setup(wadFile string, levelNumber int) (*model.ConfigRoot, error) {
	wad := New()
	if err := wad.Load(wadFile); err != nil {
		return nil, err
	}
	levelIdx := levelNumber - 1
	levelNames := wad.GetLevels()
	if levelIdx < 0 || levelIdx >= len(levelNames) {
		return nil, fmt.Errorf("invalid level number: %d", levelIdx)
	}
	levelName := levelNames[levelIdx]
	level, err := wad.GetLevel(levelName)
	if err != nil {
		return nil, err
	}
	texHandler := wad.GetTextures()
	sectors := bld.buildSectors(level, texHandler)
	grid := NewSpatialGrid(sectors, 256.0)
	things := bld.buildThings(level, texHandler, grid)
	player := bld.buildPlayer(level, grid)
	return model.NewConfigRoot(sectors, player, things, ScaleFactorLineDef, false, texHandler), nil
}

// buildConfigSector creates a ConfigSector for a given sector using level data, textures, and geometric information.
func (bld *Builder) buildConfigSector(level *Level, lumpSector *lumps.Sector, texHandler *Textures, secIdx uint16, loopIdx int, triIdx int, edges []geometry.Edge) *model.ConfigSector {
	const openAllDoors = true
	sectorId := fmt.Sprintf("s%d_l%d_t%d", secIdx, loopIdx, triIdx)
	miSector := model.NewConfigSector(sectorId, bld.convertLight(lumpSector.LightLevel), model.LightKindAmbient)
	miSector.FloorY = float64(lumpSector.FloorHeight) / ScaleFactorCeilFloorLineDef
	ceilHeight := float64(lumpSector.CeilingHeight)
	if openAllDoors {
		ceilHeight = bld.calculateOpenDoorCeil(level, secIdx, lumpSector, edges)
	}
	miSector.CeilY = ceilHeight / ScaleFactorCeilFloorLineDef
	miSector.Tag = strconv.Itoa(int(secIdx))
	ceilingType := model.AnimationKindLoop
	floorType := model.AnimationKindLoop
	if lumpSector.CeilingPic == SkyPicture {
		ceilingType = model.AnimationKindSky
		miSector.Light.Kind = model.LightKindOpenAir
	}
	if lumpSector.FloorPic == SkyPicture {
		floorType = model.AnimationKindSky
		miSector.Light.Kind = model.LightKindOpenAir
	}
	miSector.Ceil = model.NewConfigAnimation(texHandler.FlatCreateAnimation(lumpSector.CeilingPic), ceilingType, TextureScaleW, TextureScaleH)
	miSector.Floor = model.NewConfigAnimation(texHandler.FlatCreateAnimation(lumpSector.FloorPic), floorType, TextureScaleW, TextureScaleH)
	return miSector
}

// buildSectors processes the level data to create and return a list of configuration sectors with their segments and properties.
func (bld *Builder) buildSectors(level *Level, texHandler *Textures) []*model.ConfigSector {
	const quantize = 1000

	var cSectors []*model.ConfigSector
	parentsContainer := make(map[geometry.QuantizedEdgeKey]string)
	edgeSegmentsContainer := make(map[*model.ConfigSegment]bool)

	totalLineDefs := len(level.LineDefs)
	vertexes := make(geometry.Polygon, len(level.Vertexes))
	for idx, l := range level.Vertexes {
		vertexes[idx] = geometry.Point{X: float64(l.XCoord), Y: float64(l.YCoord)}
	}

	sectorsEdges := bld.createSectorsEdge(level)
	for secIdx, edges := range sectorsEdges {
		lumpSector := level.Sectors[secIdx]
		polygonDefs := vertexes.TraceLoops(edges, totalLineDefs)
		for loopIdx, def := range polygonDefs {
			mergedPoly := def.BridgeHoles()
			triangles := mergedPoly.Triangulate()
			for triIdx, tri := range triangles {
				cSector := bld.buildConfigSector(level, lumpSector, texHandler, secIdx, loopIdx, triIdx, edges)
				for k := 0; k < 3; k++ {
					p1 := tri[k]
					p2 := tri[(k+1)%3]
					cSeg, matchEdges := bld.buildSegment(level, texHandler, cSector.Id, p1, p2, edges)
					cSector.Segments = append(cSector.Segments, cSeg)
					edgeSegmentsContainer[cSeg] = matchEdges
					edgeKey := geometry.NewQuantizedEdgeKey(cSeg.Start.X, cSeg.Start.Y, cSeg.End.X, cSeg.End.Y, quantize)
					parentsContainer[edgeKey] = cSeg.Parent
				}
				cSectors = append(cSectors, cSector)
			}
		}
	}

	for _, cf := range cSectors {
		for _, cs := range cf.Segments {
			if cs.Kind == model.DefinitionJoin {
				reverseKey := geometry.NewQuantizedEdgeKey(cs.End.X, cs.End.Y, cs.Start.X, cs.Start.Y, quantize)
				if neighborId, exists := parentsContainer[reverseKey]; exists {
					cs.Neighbor = neighborId
				} else {
					fmt.Printf("WARNING: Missing edge for join segment: %s - %s (%v)\n", cs.Id, cs.Tag, reverseKey)
					// Prevents ghost walls
					matchEdges := edgeSegmentsContainer[cs]
					if matchEdges {
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

// buildSegment creates a ConfigSegment for a wall section based on sector edges and textures, handling sky and two-sided flags.
// Returns the constructed ConfigSegment and a bool indicating if it matched any edge.
func (bld *Builder) buildSegment(level *Level, texHandler *Textures, sectorId string, p1, p2 geometry.Point, sectorEdges []geometry.Edge) (*model.ConfigSegment, bool) {
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
			// vertical sky hack
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

// buildConfigThing creates and returns a list of ConfigThing objects based on the given level, texture handler, and spatial grid.
func (bld *Builder) buildThings(level *Level, texHandler *Textures, grid *SpatialGrid) []*model.ConfigThing {
	var things []*model.ConfigThing
	for i, t := range level.Things {
		tX := float64(t.X)
		tY := float64(t.Y)
		tAngle := float64(t.Angle)
		if t.Type == 1 || t.Type == 2 || t.Type == 3 || t.Type == 4 || t.Type == 11 {
			continue
		}
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
		cfgThing := model.NewConfigThing(tId, model.XY{X: tX, Y: -tY}, tAngle, sd.Kind, tSectorId, sd.Mass, sd.Radius, sd.Height, sd.Speed, anim)
		things = append(things, cfgThing)
	}
	return things
}

// buildPlayer initializes and returns a ConfigPlayer based on the player's starting position and angle in the level.
func (bld *Builder) buildPlayer(level *Level, grid *SpatialGrid) *model.ConfigPlayer {
	pX, pY, pAngle := float64(0), float64(0), float64(0)
	for _, t := range level.Things {
		if t.Type == 1 {
			pX, pY, pAngle = float64(t.X), float64(t.Y), float64(t.Angle)
			break
		}
	}
	playerSectorId := grid.ResolveSectorId(geometry.Point{X: pX, Y: pY})
	player := model.NewConfigPlayer(model.XY{X: pX, Y: -pY}, pAngle, playerSectorId, 20.0/radiusF, 100.0)
	return player
}

// calculateOpenDoorCeil determines the ceiling height for an open door based on adjacent sectors and door conditions.
func (bld *Builder) calculateOpenDoorCeil(level *Level, secIdx uint16, wadSector *lumps.Sector, edges []geometry.Edge) float64 {
	isDoor := false
	lowestAdjCeil := int16(math.MaxInt16)
	hasAdjacent := false

	for _, e := range edges {
		ld := level.LineDefs[e.LDIdx]
		// Check if the segment has a typical Doom door Action Special
		special := ld.Function
		if special != 0 {
			if _, ok := _doors[special]; ok {
				isDoor = true
			}
		}
		// Navigate the segment sides to find the adjacent sector
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
	// If it's a confirmed door (or a collapsed sector with valid adiacencies), we calculate the elevation
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

// createSectorsEdge maps sector IDs to their respective edges by analyzing LineDefs and SideDefs in the given level.
func (bld *Builder) createSectorsEdge(level *Level) map[uint16][]geometry.Edge {
	sectorsEdges := make(map[uint16][]geometry.Edge)
	for i, ld := range level.LineDefs {
		if ld.SideDefRight != -1 {
			s := level.SideDefs[ld.SideDefRight].SectorRef
			sectorsEdges[s] = append(sectorsEdges[s], geometry.Edge{V1: uint16(ld.VertexStart), V2: uint16(ld.VertexEnd), LDIdx: i, IsLeft: false})
		}
		if ld.SideDefLeft != -1 {
			s := level.SideDefs[ld.SideDefLeft].SectorRef
			sectorsEdges[s] = append(sectorsEdges[s], geometry.Edge{V1: uint16(ld.VertexEnd), V2: uint16(ld.VertexStart), LDIdx: i, IsLeft: true})
		}
	}
	return sectorsEdges
}

// convertLight converts a given light level (int16) into a normalized float64 linear intensity value ranging from 0 to 1.
// Returns -1.0 if the light level is below a predefined threshold.
func (bld *Builder) convertLight(lightLevel int16) float64 {
	rawLight := float64(lightLevel)
	// Ambient threshold
	if rawLight < 16 {
		return -1.0
	}
	// Returns normalized linear intensity
	return rawLight / 255.0
}
