package wad

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/generators/wad/lumps"
	"github.com/markel1974/godoom/mr_tech/model/geometry"
)

// WorldScaleFactor defines a constant scaling factor used to standardize and convert game world dimensions to real-world units.
const WorldScaleFactor = 25.0

// WallScaleW is a constant scaling factor applied to the width of wall textures in the game world rendering system.
const WallScaleW = WorldScaleFactor * 0.16

// ScaleCeilFloorLineDef is a constant used to scale ceiling and floor heights relative to WorldScaleFactor.
const ScaleCeilFloorLineDef = WorldScaleFactor * 0.32

// ScaleWallH defines the vertical scaling factor for wall height, determined as 40% of the global WorldScaleFactor.
const ScaleWallH = WorldScaleFactor * 0.4

// ScaleWThings is a constant representing a factor for scaling the width of "thing" objects in the game world.
const ScaleWThings = WorldScaleFactor * 0.002

// ScaleHThings defines the height scale factor for things in the game world, derived from WorldScaleFactor.
const ScaleHThings = WorldScaleFactor * 0.006

// SkyPicture represents the identifier for sky-related ceiling or floor textures in the game's level configuration.
const SkyPicture = "F_SKY1"

const openAllDoors = true

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

// Setup initializes the configuration for a specific level in the WAD file and returns a Root or an error.
func (bld *Builder) Setup(wadFile string, levelNumber int) (*config.Root, error) {
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
	var sectors []*config.Sector
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
		cSector := bld.buildSector(sectorId, light, floorPic, floorY, ceilPic, ceilY, texHandler, edges)

		for _, edge := range edges {
			cSeg := bld.buildSegment(sectorId, edge, texHandler)
			cSector.Segments = append(cSector.Segments, cSeg)
		}
		sectors = append(sectors, cSector)
	}

	var things []*config.Thing
	for i, lThing := range level.Things {
		if thing := bld.buildThings(lThing, i, texHandler); thing != nil {
			things = append(things, thing)
		}
	}

	player := bld.buildPlayer(level)
	cal := config.NewConfigCalibration(false, 0, 0, 0, 0, 0, 0, true)
	cr := config.NewConfigRoot(cal, sectors, player, things, WorldScaleFactor, texHandler)
	cr.Vertices = vertexes

	return cr, nil
}

// buildConfigSector constructs and returns a Sector for a given level sector, including floor, ceiling, and lighting data.
func (bld *Builder) buildSector(sectorId string, lightLevel int16, floorPic string, floorY float64, ceilPic string, ceilY float64, texHandler *Textures, edges []Edge) *config.Sector {
	ceilingType := config.AnimationKindLoop
	floorType := config.AnimationKindLoop
	light, kind, falloff, r, g, b := bld.heuristicLight(lightLevel, ceilPic, ceilY, floorPic, floorY, edges)

	miSector := config.NewConfigSector(sectorId, light, kind, falloff)
	miSector.Light.R = r
	miSector.Light.G = g
	miSector.Light.B = b

	miSector.FloorY = floorY / ScaleCeilFloorLineDef
	miSector.CeilY = ceilY / ScaleCeilFloorLineDef
	miSector.Tag = sectorId
	if ceilPic == SkyPicture {
		ceilingType = config.AnimationKindSky
		miSector.Light.Kind = config.LightKindOpenAir
	}
	if floorPic == SkyPicture {
		floorType = config.AnimationKindSky
		miSector.Light.Kind = config.LightKindOpenAir
	}
	miSector.Ceil = config.NewConfigAnimation(texHandler.FlatCreateAnimation(ceilPic), ceilingType, WallScaleW, ScaleWallH)
	miSector.Floor = config.NewConfigAnimation(texHandler.FlatCreateAnimation(floorPic), floorType, WallScaleW, ScaleWallH)
	return miSector
}

// buildSegment constructs a Segment for a given edge within a sector, including wall textures and alignment adjustments.
func (bld *Builder) buildSegment(sectorId string, e Edge, texHandler *Textures) *config.Segment {
	seg := config.NewConfigSegment(sectorId, config.SegmentUnknown, e.P1, e.P2)
	middleT := texHandler.TextureCreateAnimation(e.SideDef.MiddleTexture)
	upperT := texHandler.TextureCreateAnimation(e.SideDef.UpperTexture)
	lowerT := texHandler.TextureCreateAnimation(e.SideDef.LowerTexture)
	seg.Middle = config.NewConfigAnimation(middleT, config.AnimationKindLoop, WallScaleW, ScaleWallH)
	seg.Upper = config.NewConfigAnimation(upperT, config.AnimationKindLoop, WallScaleW, ScaleWallH)
	seg.Lower = config.NewConfigAnimation(lowerT, config.AnimationKindLoop, WallScaleW, ScaleWallH)
	// vertical sky hack
	if e.LineDef.HasFlag(lumps.TwoSided) && e.BackSector != nil {
		// If BOTH sectors have the ceiling set to sky, the upper wall is invisible/sky.
		if e.Sector.CeilingPic == SkyPicture && e.BackSector.CeilingPic == SkyPicture {
			seg.Upper = config.NewConfigAnimation(nil, config.AnimationKindNone, WallScaleW, ScaleWallH)
		}
		// Extension for floors (e.g. moats that show sky at the bottom)
		if e.Sector.FloorPic == SkyPicture && e.BackSector.FloorPic == SkyPicture {
			seg.Lower = config.NewConfigAnimation(nil, config.AnimationKindNone, WallScaleW, ScaleWallH)
		}
	}
	if !e.LineDef.HasFlag(2) {
		seg.Kind = config.SegmentWall
	}
	// Inversione Y per l'allineamento con il motore di rendering
	seg.Start.Y, seg.End.Y = -seg.Start.Y, -seg.End.Y
	return seg
}

// buildThings generates a list of config.Thing objects from a level's things, excluding specific types (1, 2, 3, 4, 11).
func (bld *Builder) buildThings(t *lumps.Thing, i int, texHandler *Textures) *config.Thing {
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
	anim := config.NewConfigAnimation(texHandler.SpriteCreateAnimation(frames), config.AnimationKindLoop, ScaleWThings, ScaleHThings)
	cfgThing := config.NewConfigThing(tId, geometry.XYZ{X: tX, Y: -tY, Z: 0}, tAngle, sd.Kind, sd.Mass, sd.Radius, sd.Height, sd.Speed, anim)
	return cfgThing
}

// buildPlayer initializes and returns a Player instance based on the first thing of type 1 found in the level.
func (bld *Builder) buildPlayer(level *Level) *config.Player {
	pX, pY, pAngle := float64(0), float64(0), float64(0)
	for _, t := range level.Things {
		if t.Type == 1 {
			pX, pY, pAngle = float64(t.X), float64(t.Y), float64(t.Angle)
			break
		}
	}

	const playerHeight = 175.0 / WorldScaleFactor
	//const playerRadius = 2 / WorldScaleFactor
	//const playerHeight = 25 / WorldScaleFactor
	const playerRadius = 2 / WorldScaleFactor
	const playerSpeed = 2000 / WorldScaleFactor
	const playerMass = 8

	player := config.NewConfigPlayer(geometry.XYZ{X: pX, Y: -pY, Z: 0}, pAngle, playerMass, playerSpeed, playerRadius, playerHeight)
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

// Estendiamo l'euristica per restituire anche R, G, B
func (bld *Builder) heuristicLight(lightLevel int16, ceilPic string, ceilY float64, floorPic string, floorY float64, edges []Edge) (float64, config.LightKind, float64, float64, float64, float64) {
	intensity := float64(lightLevel) * 0.008
	kind := config.LightKindAmbient
	falloff := (ceilY - floorY) / ScaleCeilFloorLineDef * 1.5

	// Default: Luce bianca/calda
	r, g, b := 1.0, 0.95, 0.9

	// --- EURISTICA DEL COLORE ---
	// 1. Acido / Radioattività (Nukage, Slime) -> Verde
	if strings.Contains(floorPic, "NUKAGE") || strings.Contains(floorPic, "SLIME") {
		r, g, b = 0.2, 1.0, 0.2
		// Se c'è acido a terra, illuminiamo l'ambiente di verde!
	}
	// 2. Lava / Sangue -> Rosso
	if strings.Contains(floorPic, "LAVA") || strings.Contains(floorPic, "BLOOD") || strings.Contains(ceilPic, "RED") {
		r, g, b = 1.0, 0.3, 0.1
	}
	// 3. Acqua / Computer -> Blu
	if strings.Contains(floorPic, "FWATER") || strings.Contains(ceilPic, "BLUE") || strings.Contains(ceilPic, "COMP") {
		r, g, b = 0.2, 0.5, 1.0
	}
	// --- EURISTICA SPOTLIGHT ---
	minX, maxX, minY, maxY := math.MaxFloat64, -math.MaxFloat64, math.MaxFloat64, -math.MaxFloat64
	for _, e := range edges {
		for _, p := range []geometry.XY{e.P1, e.P2} {
			if p.X < minX {
				minX = p.X
			}
			if p.X > maxX {
				maxX = p.X
			}
			if p.Y < minY {
				minY = p.Y
			}
			if p.Y > maxY {
				maxY = p.Y
			}
		}
	}
	width := maxX - minX
	height := maxY - minY

	isSmall := width < 128 && height < 128
	isBright := lightLevel > 192
	isLightTexture := false
	lightTextures := []string{"LITE", "TLITE", "CEIL1_1", "CEIL1_2", "GLOW", "LAMP"}
	for _, t := range lightTextures {
		if strings.Contains(ceilPic, t) {
			isLightTexture = true
			break
		}
	}
	if (isSmall && isLightTexture) || (isSmall && isBright) {
		kind = config.LightKindSpot
		intensity *= 2.0
		if falloff < 10.0 {
			falloff = 10.0
		}
	}
	return intensity, kind, falloff, r, g, b
}
