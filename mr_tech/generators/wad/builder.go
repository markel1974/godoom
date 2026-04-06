package wad

import (
	"fmt"
	"math"
	"strconv"

	"github.com/markel1974/godoom/mr_tech/generators/wad/lumps"
	"github.com/markel1974/godoom/mr_tech/model/config"
	"github.com/markel1974/godoom/mr_tech/model/geometry"
)

// TextureScaleW defines the horizontal scaling factor for textures, used in rendering and configuration processes.
const TextureScaleW = 4.0

// TextureScaleH defines the horizontal scaling factor for textures, typically used to adjust their visual dimensions.
const TextureScaleH = 10.0

// ScaleFactorLineDef defines the scaling factor applied to line definitions during level configuration processing.
const ScaleFactorLineDef = 25.0

// ScaleFactorCeilFloorLineDef defines the scaling factor for converting ceiling and floor heights in level configurations.
const ScaleFactorCeilFloorLineDef = 8.0

// SkyPicture represents the identifier for sky-related ceiling or floor textures in the game's level configuration.
const SkyPicture = "F_SKY1"

const openAllDoors = true

// _doors is a map that associates specific action special IDs (int16) with corresponding door behaviors (string descriptions).
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

// Edge represents a connection between two points (P1 and P2) with metadata such as Sidedef, Linedef, and associated Sectors.
type Edge struct {
	P1         geometry.XY
	P2         geometry.XY
	SideDef    *lumps.SideDef
	LineDef    *lumps.LineDef
	Sector     *lumps.Sector
	V1         geometry.XY
	V2         geometry.XY
	BackSector *lumps.Sector
}

// Builder is a type responsible for constructing and configuring game assets like sectors, things, and players.
type Builder struct {
}

// NewBuilder creates and returns a new instance of Builder.
func NewBuilder() *Builder {
	return &Builder{}
}

// Setup initializes the configuration for a specific level in the WAD file and returns a ConfigRoot or an error.
func (bld *Builder) Setup(wadFile string, levelNumber int) (*config.ConfigRoot, error) {
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
	vertexes := make(geometry.Polygon, len(level.Vertexes))
	for idx, l := range level.Vertexes {
		vertexes[idx] = geometry.XY{X: float64(l.XCoord), Y: float64(l.YCoord)}
	}

	sectorsEdges := bld.createSectorsEdges(level, vertexes)
	var sectors []*config.ConfigSector
	for secIdx, edges := range sectorsEdges {
		if edges == nil {
			continue
		}
		lSector := level.Sectors[secIdx]
		light := lSector.LightLevel
		ceilPic := lSector.CeilingPic
		floorPic := lSector.FloorPic
		floorY := float64(lSector.FloorHeight)
		ceilY := float64(lSector.CeilingHeight)
		if openAllDoors {
			ceilY = bld.calculateOpenDoorCeil(level, uint16(secIdx), lSector, edges)
		}
		sectorId := strconv.Itoa(secIdx)
		cSector := bld.buildSector(sectorId, light, floorPic, floorY, ceilPic, ceilY, texHandler)
		for _, edge := range edges {
			cSeg := bld.buildSegment(sectorId, edge, texHandler)
			cSector.Segments = append(cSector.Segments, cSeg)
		}
		sectors = append(sectors, cSector)
	}

	var things []*config.ConfigThing
	for i, lThing := range level.Things {
		if thing := bld.buildThings(lThing, i, texHandler); thing != nil {
			things = append(things, thing)
		}
	}

	player := bld.buildPlayer(level)
	cr := config.NewConfigRoot(sectors, player, things, ScaleFactorLineDef, false, texHandler)
	cr.Vertices = vertexes

	return cr, nil
}

// buildConfigSector constructs and returns a ConfigSector for a given level sector, including floor, ceiling, and lighting data.
func (bld *Builder) buildSector(sectorId string, lightLevel int16, floorPic string, floorY float64, ceilPic string, ceilY float64, texHandler *Textures) *config.ConfigSector {
	ceilingType := config.AnimationKindLoop
	floorType := config.AnimationKindLoop
	const falloff = 10.0
	miSector := config.NewConfigSector(sectorId, bld.convertLight(lightLevel), config.LightKindAmbient, falloff)
	miSector.FloorY = floorY / ScaleFactorCeilFloorLineDef
	miSector.CeilY = ceilY / ScaleFactorCeilFloorLineDef
	miSector.Tag = sectorId
	if ceilPic == SkyPicture {
		ceilingType = config.AnimationKindSky
		miSector.Light.Kind = config.LightKindOpenAir
	}
	if floorPic == SkyPicture {
		floorType = config.AnimationKindSky
		miSector.Light.Kind = config.LightKindOpenAir
	}
	miSector.Ceil = config.NewConfigAnimation(texHandler.FlatCreateAnimation(ceilPic), ceilingType, TextureScaleW, TextureScaleH)
	miSector.Floor = config.NewConfigAnimation(texHandler.FlatCreateAnimation(floorPic), floorType, TextureScaleW, TextureScaleH)
	return miSector
}

// buildSegment constructs a ConfigSegment for a given edge within a sector, including wall textures and alignment adjustments.
func (bld *Builder) buildSegment(sectorId string, e Edge, texHandler *Textures) *config.ConfigSegment {
	seg := config.NewConfigSegment(sectorId, config.SegmentUnknown, e.P1, e.P2)
	middleT := texHandler.TextureCreateAnimation(e.SideDef.MiddleTexture)
	upperT := texHandler.TextureCreateAnimation(e.SideDef.UpperTexture)
	lowerT := texHandler.TextureCreateAnimation(e.SideDef.LowerTexture)
	seg.Middle = config.NewConfigAnimation(middleT, config.AnimationKindLoop, TextureScaleW, TextureScaleH)
	seg.Upper = config.NewConfigAnimation(upperT, config.AnimationKindLoop, TextureScaleW, TextureScaleH)
	seg.Lower = config.NewConfigAnimation(lowerT, config.AnimationKindLoop, TextureScaleW, TextureScaleH)
	// vertical sky hack
	if e.LineDef.HasFlag(lumps.TwoSided) && e.BackSector != nil {
		// If BOTH sectors have the ceiling set to sky, the upper wall is invisible/sky.
		if e.Sector.CeilingPic == SkyPicture && e.BackSector.CeilingPic == SkyPicture {
			seg.Upper = config.NewConfigAnimation(nil, config.AnimationKindNone, TextureScaleW, TextureScaleH)
		}
		// Extension for floors (e.g. moats that show sky at the bottom)
		if e.Sector.FloorPic == SkyPicture && e.BackSector.FloorPic == SkyPicture {
			seg.Lower = config.NewConfigAnimation(nil, config.AnimationKindNone, TextureScaleW, TextureScaleH)
		}
	}
	if !e.LineDef.HasFlag(2) {
		seg.Kind = config.SegmentWall
	}
	// Inversione Y per l'allineamento con il motore di rendering
	seg.Start.Y, seg.End.Y = -seg.Start.Y, -seg.End.Y
	return seg
}

// buildThings generates a list of config.ConfigThing objects from a level's things, excluding specific types (1, 2, 3, 4, 11).
func (bld *Builder) buildThings(t *lumps.Thing, i int, texHandler *Textures) *config.ConfigThing {
	tX := float64(t.X)
	tY := float64(t.Y)
	tAngle := float64(t.Angle)
	if t.Type == 1 || t.Type == 2 || t.Type == 3 || t.Type == 4 || t.Type == 11 {
		return nil
	}
	sd, hasAnim := _spriteDictionary[int(t.Type)]
	var frames []string
	if !hasAnim {
		fmt.Printf("WARNING No animation found for thing type %d, using default sprite\n", t.Type)
		frames = []string{"UNKNOWN"}
	} else {
		frames = sd.Sprites
	}
	tId := fmt.Sprintf("t_%d", i)
	anim := config.NewConfigAnimation(texHandler.SpriteCreateAnimation(frames), config.AnimationKindLoop, TextureScaleW/70, TextureScaleH/70)
	cfgThing := config.NewConfigThing(tId, geometry.XY{X: tX, Y: -tY}, tAngle, sd.Kind, sd.Mass, sd.Radius, sd.Height, sd.Speed, anim)
	return cfgThing
}

// buildPlayer initializes and returns a ConfigPlayer instance based on the first thing of type 1 found in the level.
func (bld *Builder) buildPlayer(level *Level) *config.ConfigPlayer {
	pX, pY, pAngle := float64(0), float64(0), float64(0)
	for _, t := range level.Things {
		if t.Type == 1 {
			pX, pY, pAngle = float64(t.X), float64(t.Y), float64(t.Angle)
			break
		}
	}
	player := config.NewConfigPlayer(geometry.XY{X: pX, Y: -pY}, pAngle, 20.0/radiusF, 100.0)
	return player
}

// calculateOpenDoorCeil calculates the ceiling height for an open door or collapsed sector based on adjacent sectors and door checks.
func (bld *Builder) calculateOpenDoorCeil(level *Level, secIdx uint16, sector *lumps.Sector, edges []Edge) float64 {
	isDoor := false
	lowestAdjCeil := int16(math.MaxInt16)
	hasAdjacent := false

	for _, e := range edges {
		// Check if the segment has a typical Doom door Action Special
		if _, ok := _doors[e.LineDef.Function]; ok {
			isDoor = true
		}
		// Navigate the segment sides to find the adjacent sector
		if e.LineDef.SideDefRight != -1 && e.LineDef.SideDefLeft != -1 {
			adjSideIdx := e.LineDef.SideDefRight
			if level.SideDefs[adjSideIdx].SectorRef == secIdx {
				adjSideIdx = e.LineDef.SideDefLeft
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
	if (isDoor && (sector.CeilingHeight <= sector.FloorHeight && hasAdjacent)) && lowestAdjCeil != math.MaxInt16 {
		targetCeil := lowestAdjCeil - 4
		if targetCeil < sector.FloorHeight {
			targetCeil = lowestAdjCeil // Emergency fallback if -4 penetrates the floor
		}
		return float64(targetCeil)
	}
	// Return the original height if it's not a door
	return float64(sector.CeilingHeight)
}

// createSectorsEdges constructs edges for each sector in the provided level using its LineDefs and vertex data.
func (bld *Builder) createSectorsEdges(level *Level, vertexes geometry.Polygon) [][]Edge {
	sectorsEdges := make([][]Edge, len(level.Sectors))
	add := func(ld *lumps.LineDef, sdIdx int16, isLeft bool) {
		if sdIdx < 0 || int(sdIdx) >= len(level.SideDefs) {
			fmt.Printf("WARNING: Invalid side index %d\n", sdIdx)
			return
		}
		sd := level.SideDefs[sdIdx]
		if sd.SectorRef < 0 || int(sd.SectorRef) >= len(sectorsEdges) {
			fmt.Printf("WARNING: Invalid sector reference %d\n", sd.SectorRef)
			return
		}
		sector := level.Sectors[sd.SectorRef]
		vStart := ld.VertexStart
		vEnd := ld.VertexEnd
		backSideIdx := ld.SideDefLeft
		vertexStart := level.Vertexes[vStart]
		vertexEnd := level.Vertexes[vEnd]
		edge := Edge{
			P1:         vertexes[vStart],
			P2:         vertexes[vEnd],
			V1:         geometry.XY{X: float64(vertexStart.XCoord), Y: float64(vertexStart.YCoord)},
			V2:         geometry.XY{X: float64(vertexEnd.XCoord), Y: float64(vertexEnd.YCoord)},
			LineDef:    ld,
			Sector:     sector,
			SideDef:    sd,
			BackSector: nil,
		}
		if isLeft {
			edge.P1, edge.P2 = edge.P2, edge.P1
			edge.V1, edge.V2 = edge.V2, edge.V1
			backSideIdx = ld.SideDefRight
		}
		if backSideIdx != -1 {
			backSide := level.SideDefs[backSideIdx]
			edge.BackSector = level.Sectors[backSide.SectorRef]
		}
		sectorsEdges[sd.SectorRef] = append(sectorsEdges[sd.SectorRef], edge)
	}
	for _, ld := range level.LineDefs {
		if ld.SideDefRight != -1 {
			add(ld, ld.SideDefRight, false)
		}
		if ld.SideDefLeft != -1 {
			add(ld, ld.SideDefLeft, true)
		}
	}
	return sectorsEdges
}

// convertLight converts a light level from an integer value to a normalized float intensity ranging from 0.0 to 1.0.
// If the light level is below the ambient threshold (16), it returns -1.0 to indicate insufficient illumination.
func (bld *Builder) convertLight(lightLevel int16) float64 {
	rawLight := float64(lightLevel)
	// Ambient threshold
	if rawLight < 16 {
		return -1.0
	}
	// Returns normalized linear intensity
	return rawLight / 255.0
}
