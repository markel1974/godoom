package wad

import (
	"fmt"
	"math"
	"strconv"

	"github.com/markel1974/godoom/mr_tech/generators/wad/lumps"
	"github.com/markel1974/godoom/mr_tech/model/config"
	"github.com/markel1974/godoom/mr_tech/model/geometry"
)

// Edge represents a connection between two vertices, usually part of the boundary of a sector or polygon in the level.
type Edge struct {
	V1Idx  int16
	V2Idx  int16
	LdIdx  int
	IsLeft bool
}

// BuilderNew is a structure used for creating and configuring game levels, sectors, objects, and player entities.
type BuilderNew struct {
}

// NewBuilderNew creates and returns a new instance of BuilderNew.
func NewBuilderNew() *BuilderNew {
	return &BuilderNew{}
}

// Setup initializes the level configuration by loading WAD file, processing level data, and building sectors, player, and things.
func (bld *BuilderNew) Setup(wadFile string, levelNumber int) (*config.ConfigRoot, error) {
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
	//grid := NewSpatialGrid(sectors, 256.0)
	things := bld.buildThings(level, texHandler)
	player := bld.buildPlayer(level)
	return config.NewConfigRoot(sectors, player, things, ScaleFactorLineDef, false, texHandler), nil
}

// buildConfigSector creates a ConfigSector object based on level data, sector properties, textures, and geometry edges.
func (bld *BuilderNew) buildConfigSector(level *Level, lumpSector *lumps.Sector, texHandler *Textures, secIdx uint16, loopIdx int, triIdx int, edges []Edge) *config.ConfigSector {
	const openAllDoors = true
	sectorId := fmt.Sprintf("s%d_l%d_t%d", secIdx, loopIdx, triIdx)
	miSector := config.NewConfigSector(sectorId, bld.convertLight(lumpSector.LightLevel), config.LightKindAmbient)
	miSector.FloorY = float64(lumpSector.FloorHeight) / ScaleFactorCeilFloorLineDef
	ceilHeight := float64(lumpSector.CeilingHeight)
	if openAllDoors {
		ceilHeight = bld.calculateOpenDoorCeil(level, secIdx, lumpSector, edges)
	}
	miSector.CeilY = ceilHeight / ScaleFactorCeilFloorLineDef
	miSector.Tag = strconv.Itoa(int(secIdx))
	ceilingType := config.AnimationKindLoop
	floorType := config.AnimationKindLoop
	if lumpSector.CeilingPic == SkyPicture {
		ceilingType = config.AnimationKindSky
		miSector.Light.Kind = config.LightKindOpenAir
	}
	if lumpSector.FloorPic == SkyPicture {
		floorType = config.AnimationKindSky
		miSector.Light.Kind = config.LightKindOpenAir
	}
	miSector.Ceil = config.NewConfigAnimation(texHandler.FlatCreateAnimation(lumpSector.CeilingPic), ceilingType, TextureScaleW, TextureScaleH)
	miSector.Floor = config.NewConfigAnimation(texHandler.FlatCreateAnimation(lumpSector.FloorPic), floorType, TextureScaleW, TextureScaleH)
	return miSector
}

// createSectorsEdges generates a map associating each sector with its edges based on the level's LineDefs and SideDefs.
func (bld *BuilderNew) createSectorsEdges(level *Level) map[uint16][]Edge {
	sectorsEdges := make(map[uint16][]Edge)
	for i, ld := range level.LineDefs {
		if ld.SideDefRight != -1 {
			s := level.SideDefs[ld.SideDefRight].SectorRef
			sectorsEdges[s] = append(sectorsEdges[s], Edge{V1Idx: ld.VertexStart, V2Idx: ld.VertexEnd, LdIdx: i, IsLeft: false})
		}
		if ld.SideDefLeft != -1 {
			s := level.SideDefs[ld.SideDefLeft].SectorRef
			sectorsEdges[s] = append(sectorsEdges[s], Edge{V1Idx: ld.VertexEnd, V2Idx: ld.VertexStart, LdIdx: i, IsLeft: true})
		}
	}
	return sectorsEdges
}

// buildSectors constructs configuration sectors from a game level, mapping edges and geometry, and returns them as a slice.
func (bld *BuilderNew) buildSectors(level *Level, texHandler *Textures) []*config.ConfigSector {
	var cSectors []*config.ConfigSector

	sectorsEdges := bld.createSectorsEdges(level)
	for secIdx, edges := range sectorsEdges {
		lumpSector := level.Sectors[secIdx]

		// loopIdx e triIdx non esistono più, passiamo 0
		cSector := bld.buildConfigSector(level, lumpSector, texHandler, secIdx, 0, 0, edges)
		for _, e := range edges {
			v1 := level.Vertexes[e.V1Idx]
			v2 := level.Vertexes[e.V2Idx]

			p1 := geometry.XY{X: float64(v1.XCoord), Y: float64(v1.YCoord)}
			p2 := geometry.XY{X: float64(v2.XCoord), Y: float64(v2.YCoord)}

			// Armonizzazione: Se è IsLeft, invertiamo Start ed End per mantenere il Winding Order CCW nativo
			if e.IsLeft {
				p1, p2 = p2, p1
			}

			// buildSegment mappa l'edge direttamente, senza più cercare l'intersezione e senza restituire bool
			cSeg := bld.buildSegment(level, texHandler, cSector.Id, p1, p2, e)
			cSector.Segments = append(cSector.Segments, cSeg)
		}

		cSectors = append(cSectors, cSector)
	}

	return cSectors
}

// buildSegment creates a ConfigSegment for a given Edge using level data, textures, and positional information.
func (bld *BuilderNew) buildSegment(level *Level, texHandler *Textures, sectorId string, p1, p2 geometry.XY, e Edge) *config.ConfigSegment {
	mp1 := geometry.XY{X: p1.X, Y: p1.Y}
	mp2 := geometry.XY{X: p2.X, Y: p2.Y}

	seg := config.NewConfigSegment(sectorId, config.DefinitionWall, mp1, mp2, "")
	ld := level.LineDefs[e.LdIdx]

	sideIdx := ld.SideDefRight
	if e.IsLeft {
		sideIdx = ld.SideDefLeft
	}
	side := level.SideDefs[sideIdx]

	middleT := texHandler.TextureCreateAnimation(side.MiddleTexture)
	upperT := texHandler.TextureCreateAnimation(side.UpperTexture)
	lowerT := texHandler.TextureCreateAnimation(side.LowerTexture)
	seg.Middle = config.NewConfigAnimation(middleT, config.AnimationKindLoop, TextureScaleW, TextureScaleH)
	seg.Upper = config.NewConfigAnimation(upperT, config.AnimationKindLoop, TextureScaleW, TextureScaleH)
	seg.Lower = config.NewConfigAnimation(lowerT, config.AnimationKindLoop, TextureScaleW, TextureScaleH)

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
				seg.Upper = config.NewConfigAnimation(nil, config.AnimationKindNone, TextureScaleW, TextureScaleH)
			}
			// Extension for floors (e.g. moats that show sky at the bottom)
			if frontSector.FloorPic == SkyPicture && backSector.FloorPic == SkyPicture {
				seg.Lower = config.NewConfigAnimation(nil, config.AnimationKindNone, TextureScaleW, TextureScaleH)
			}
		}
	}

	if ld.HasFlag(2) {
		seg.Kind = config.DefinitionJoin
	}

	seg.Start.Y, seg.End.Y = -seg.Start.Y, -seg.End.Y

	return seg
}

// buildThings processes the Things in a level and creates config.ConfigThing objects for valid types.
func (bld *BuilderNew) buildThings(level *Level, texHandler *Textures) []*config.ConfigThing {
	var things []*config.ConfigThing
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
		//tSectorId := grid.ResolveSectorId(geometry.XY{X: tX, Y: tY})
		tId := fmt.Sprintf("t_%d", i)
		anim := config.NewConfigAnimation(texHandler.SpriteCreateAnimation(frames), config.AnimationKindLoop, TextureScaleW/70, TextureScaleH/70)
		cfgThing := config.NewConfigThing(tId, geometry.XY{X: tX, Y: -tY}, tAngle, sd.Kind, sd.Mass, sd.Radius, sd.Height, sd.Speed, anim)
		things = append(things, cfgThing)
	}
	return things
}

// buildPlayer initializes and returns a new player configuration based on the level data and spatial grid.
func (bld *BuilderNew) buildPlayer(level *Level) *config.ConfigPlayer {
	pX, pY, pAngle := float64(0), float64(0), float64(0)
	for _, t := range level.Things {
		if t.Type == 1 {
			pX, pY, pAngle = float64(t.X), float64(t.Y), float64(t.Angle)
			break
		}
	}
	//playerSectorId := grid.ResolveSectorId(geometry.XY{X: pX, Y: pY})
	player := config.NewConfigPlayer(geometry.XY{X: pX, Y: -pY}, pAngle, 20.0/radiusF, 100.0)
	return player
}

// calculateOpenDoorCeil determines the ceiling height for sectors that act as doors based on adjacency and sector properties.
func (bld *BuilderNew) calculateOpenDoorCeil(level *Level, secIdx uint16, wadSector *lumps.Sector, edges []Edge) float64 {
	isDoor := false
	lowestAdjCeil := int16(math.MaxInt16)
	hasAdjacent := false

	for _, e := range edges {
		ld := level.LineDefs[e.LdIdx]
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

// convertLight converts an ambient light level from an integer to a normalized float64 intensity value (0.0 to 1.0).
// If the light level is below the threshold of 16, it returns -1.0 as an error or indicator.
func (bld *BuilderNew) convertLight(lightLevel int16) float64 {
	rawLight := float64(lightLevel)
	// Ambient threshold
	if rawLight < 16 {
		return -1.0
	}
	// Returns normalized linear intensity
	return rawLight / 255.0
}
