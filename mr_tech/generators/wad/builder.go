package wad

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/generators/common"
	"github.com/markel1974/godoom/mr_tech/generators/wad/lumps"
	"github.com/markel1974/godoom/mr_tech/geometry"
)

// AspectRatio defines the fixed width-to-height ratio used for rendering or configuration purposes, set to 1.5.
const AspectRatio = 1.5

// ScaleLight is a constant factor used to scale light intensity values within the lighting system calculations.
const ScaleLight = 0.015

// ScaleSectorH is a constant used to scale the height values of sectors in the configuration.
const ScaleSectorH = 1.0

// ScaleTextureW represents the width scaling factor applied to textures during rendering or material configuration.
const ScaleTextureW = 1.0

// ScaleTextureH represents the vertical texture scaling factor, typically used for aligning textures in rendering operations.
const ScaleTextureH = 1.0

// ScaleWThings defines the horizontal scaling factor applied to the material dimensions of "Thing" entities in the game world.
const ScaleWThings = 1.0

// ScaleHThings represents the scaling factor for the height of "Thing" entities in the game configuration.
const ScaleHThings = 1.0

// GForce represents the standard gravitational force multiplier applied to game entities, with a value of 9.8 * 8.
const GForce = 9.8 * 50

// playerHeight defines the height of a player entity in game units, used for calculations related to geometry and physics.
const playerHeight = 62.0

// playerRadius defines the constant radius of the player character used for collision detection and bounding operations.
const playerRadius = 30

// playerSpeed defines the constant speed value for the player, represented in units per second.
const playerSpeed = 1800

// playerMass defines the mass of the player entity in the game, used for physics calculations and movement behavior.
const playerMass = 50

// SkyPicture represents the texture string identifier used for sky rendering in sectors and segments.
const SkyPicture = "F_SKY1"

// openAllDoors determines whether all doors in the level should be opened automatically during sector configuration.
const openAllDoors = true

// Edge represents a line in a 2D space connecting two points, with metadata about its relationship to sectors and sidedefs.
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

// Builder represents a utility for constructing configuration objects from level data in a WAD file.
type Builder struct {
}

// NewBuilder creates and returns a new instance of Builder.
func NewBuilder() *Builder {
	return &Builder{}
}

// Build generates a Root configuration by loading level data from a WAD file and populating sectors, things, and player.
func (bld *Builder) Build(fs common.IResources, wadFile string, levelNumber int) (*config.Root, error) {
	wad := New()
	if err := wad.Load(fs, wadFile); err != nil {
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
	texHandler := wad.CreateTextures()

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
		if lThing.Type == 1 || lThing.Type == 2 || lThing.Type == 3 || lThing.Type == 4 || lThing.Type == 11 {
			continue
		}
		thing, tErr := bld.buildThings(lThing, i, texHandler)
		if tErr != nil {
			fmt.Println(tErr.Error())
		}
		things = append(things, thing)
	}

	player := bld.buildPlayer(level)
	cal := config.NewConfigCalibration(0, 0, 0, 0, 0, 0, true)
	cal.AspectRatio = AspectRatio
	scaleFactor := geometry.XYZ{X: 1.0, Y: 1.0, Z: 1.0}
	cr := config.NewConfigRoot(cal, sectors, player, things, scaleFactor, texHandler)
	cr.Vertices = vertexes

	crosshair := common.NewSimpleCrosshair()
	texHandler.AddDirect(crosshair.GetTexture())
	cr.AddUIElement(crosshair.Create())

	return cr, nil
}

// buildSector creates and configures a Sector with specified properties such as light levels, textures, and geometry.
func (bld *Builder) buildSector(sectorId string, lightLevel int16, floorPic string, floorY float64, ceilPic string, ceilY float64, texHandler *Textures, edges []Edge) *config.Sector {
	ceilingType := config.MaterialKindLoop
	floorType := config.MaterialKindLoop
	light, kind, falloff, r, g, b := bld.heuristicLight(lightLevel, ceilPic, ceilY, floorPic, floorY, edges)

	//if v, ok := bld.heuristicMaterial(floorPic); ok {
	//	floorType = v
	//}

	miSector := config.NewConfigSector(sectorId, light, kind, falloff)
	miSector.Light.R = r
	miSector.Light.G = g
	miSector.Light.B = b

	miSector.FloorY = floorY * ScaleSectorH
	miSector.CeilY = ceilY * ScaleSectorH
	miSector.Tag = sectorId
	if ceilPic == SkyPicture {
		ceilingType = config.MaterialKindSky
		miSector.Light.Kind = config.LightKindOpenAir
	}
	if floorPic == SkyPicture {
		floorType = config.MaterialKindSky
		miSector.Light.Kind = config.LightKindOpenAir
	}
	miSector.Ceil = config.NewConfigMaterial("miSector.Ceil", texHandler.FlatCreateAnimation(ceilPic), ceilingType)
	miSector.Floor = config.NewConfigMaterial("miSector.Floor", texHandler.FlatCreateAnimation(floorPic), floorType)
	bld.configureMaterial(miSector.Ceil, ceilPic)
	bld.configureMaterial(miSector.Floor, floorPic)
	miSector.FloorAlign = config.NewUVAlignSurface(false, geometry.XY{}, geometry.XY{}, false, false, false, geometry.XY{}, ScaleTextureW, ScaleTextureH)
	miSector.CeilAlign = config.NewUVAlignSurface(false, geometry.XY{}, geometry.XY{}, false, false, false, geometry.XY{}, ScaleTextureW, ScaleTextureH)
	return miSector
}

// buildSegment creates and configures a new segment for a given sector and edge, applying texture and rendering adjustments.
func (bld *Builder) buildSegment(sectorId string, e Edge, texHandler *Textures) *config.Segment {
	seg := config.NewConfigSegment(sectorId, config.SegmentUnknown, e.P1, e.P2)
	middleT := texHandler.TextureCreateAnimation(e.SideDef.MiddleTexture)
	upperT := texHandler.TextureCreateAnimation(e.SideDef.UpperTexture)
	lowerT := texHandler.TextureCreateAnimation(e.SideDef.LowerTexture)
	seg.Middle = config.NewConfigMaterial("seg.Middle", middleT, config.MaterialKindLoop)
	seg.Upper = config.NewConfigMaterial("seg.Upper", upperT, config.MaterialKindLoop)
	seg.Lower = config.NewConfigMaterial("seg.Lower", lowerT, config.MaterialKindLoop)
	bld.configureMaterial(seg.Middle, e.SideDef.MiddleTexture)
	bld.configureMaterial(seg.Upper, e.SideDef.UpperTexture)
	bld.configureMaterial(seg.Lower, e.SideDef.LowerTexture)

	wallTexLen := math.Hypot(e.P2.X-e.P1.X, e.P2.Y-e.P1.Y)
	panning := geometry.XY{X: float64(e.SideDef.XOffset), Y: float64(e.SideDef.YOffset)}

	middleAlignBottom := e.LineDef.HasFlag(lumps.LowerUnpegged)
	upperAlignBottom := !e.LineDef.HasFlag(lumps.UpperUnpegged)
	lowerAlignBottom := e.LineDef.HasFlag(lumps.LowerUnpegged)

	seg.UVAlign = config.NewUVAlignVertical(middleAlignBottom, false, false, panning, wallTexLen, ScaleTextureH)
	seg.UVAlignUpper = config.NewUVAlignVertical(upperAlignBottom, false, false, panning, wallTexLen, ScaleTextureH)
	seg.UVAlignLower = config.NewUVAlignVertical(lowerAlignBottom, false, false, panning, wallTexLen, ScaleTextureH)

	// vertical sky hack
	if e.LineDef.HasFlag(lumps.TwoSided) && e.BackSector != nil {
		// If BOTH sectors have the ceiling set to sky, the upper wall is invisible/sky.
		if e.Sector.CeilingPic == SkyPicture && e.BackSector.CeilingPic == SkyPicture {
			seg.Upper = config.NewConfigMaterial("seg.Upper", nil, config.MaterialKindNone)
		}
		// Extension for floors (e.g. moats that show sky at the bottom)
		if e.Sector.FloorPic == SkyPicture && e.BackSector.FloorPic == SkyPicture {
			seg.Lower = config.NewConfigMaterial("seg.Lower =", nil, config.MaterialKindNone)
		}
	}
	if !e.LineDef.HasFlag(2) {
		seg.Kind = config.SegmentWall
	}
	// Inversione Y per l'allineamento con il motore di rendering
	//seg.Start.Y, seg.End.Y = -seg.Start.Y, -seg.End.Y
	return seg
}

// buildThings creates a configuration object for a game "Thing" entity with its properties and associated materials.
func (bld *Builder) buildThings(t *lumps.Thing, i int, texHandler *Textures) (*config.Thing, error) {
	tX := float64(t.X)
	tY := float64(t.Y)
	tAngle := float64(t.Angle)
	sd, found := _spriteDictionary[int(t.Type)]
	if !found {
		return nil, fmt.Errorf("[warning] buildThings sprite not found %d\n", int(t.Type))
	}

	animations := config.NewEnemyAnimations()
	multiAngleAnim := CreateAnimation(texHandler, sd.Sprite)
	ms := config.NewMultiSprite()
	ms.Clamps = []int{3, 4}

	if sd.Dead {
		sd.Kind = config.ThingItemDef
		angleAnim := multiAngleAnim[EnemyActionDead]
		if len(angleAnim[0]) == 0 {
			return nil, fmt.Errorf("[warning] buildThings dead not found %d\n", int(t.Type))
		}
		actionFrames := angleAnim[0]
		latestFrame := actionFrames[len(actionFrames)-1]
		mat := config.NewConfigMaterial("anim.dead", []string{latestFrame}, config.MaterialKindLoop)
		bld.configureThingMaterial(mat, sd.Sprite)
		_ = ms.Add(mat)
	} else {
		for idx, angleAnim := range multiAngleAnim {
			//for _, view := range temporalFrames {
			if len(angleAnim[0]) == 0 {
				continue
			}
			actionFrames := angleAnim[0]
			mat := config.NewConfigMaterial("anim", actionFrames, config.MaterialKindLoop)
			bld.configureThingMaterial(mat, sd.Sprite)
			animId := ms.Add(mat)
			//TODO PATCH
			//animId = 0
			for _, ad := range EnemyStateAdapter[idx] {
				animations.Add(ad, animId, len(actionFrames))
			}
		}
	}

	if ms.Len() == 0 {
		return nil, fmt.Errorf("[warning] buildThings no material found %d\n", int(t.Type))
	}

	tId := fmt.Sprintf("%s_%d", sd.Sprite, i)
	cfgThing := config.NewConfigThing(tId, geometry.XYZ{X: tX, Y: tY, Z: 0}, tAngle, sd.Kind, sd.Mass, sd.Radius, sd.Height, sd.Speed)
	cfgThing.MultiSprite = ms
	if cfgThing.Kind == config.ThingEnemyDef {
		cfgThing.GForce = GForce
		cfgThing.JumpForce = GForce * 6
		cfgThing.WakeupDistance = 400
		_ = common.NewEnemy(cfgThing, animations)
	} else {
		cfgThing.GForce = GForce
		_ = common.NewItem(cfgThing)
	}
	return cfgThing, nil
}

// buildPlayer creates and returns a Player object configured with position, angle, mass, speed, radius, and height.
func (bld *Builder) buildPlayer(level *Level) *config.Player {
	pX, pY, pAngle := float64(0), float64(0), float64(0)
	for _, t := range level.Things {
		if t.Type == 1 {
			pX, pY, pAngle = float64(t.X), float64(t.Y), float64(t.Angle)
			break
		}
	}

	player := config.NewConfigPlayer(geometry.XYZ{X: pX, Y: pY, Z: 0}, pAngle, playerMass, playerSpeed, playerRadius, playerHeight)
	_ = common.NewPlayer(player)
	player.GForce = GForce
	player.JumpForce = GForce * 8
	player.Impulse = 0.13 //mod

	player.Flash.ZFar = 8192
	player.Flash.Intensity = 0.02
	player.Flash.Falloff = 2000
	player.Flash.OffsetX = 0.2
	player.Flash.OffsetY = 0.1
	player.Bobbing.SwayScale = 2.0
	player.Bobbing.SwayOffsetX = 20
	player.Bobbing.SwayOffsetY = -0.9
	player.Bobbing.MaxAmplitudeX = playerHeight * 0.15
	player.Bobbing.MaxAmplitudeY = playerHeight * 0.15
	player.Bobbing.StrideLength = 0.048 //0.0008 // FREQUENZA: 1000 * 0.0007 = 0.7 rad/frame.

	// Breathing
	player.Bobbing.IdleAmpX = 0.9  //mod 0.9
	player.Bobbing.IdleAmpY = 0.9  //mod 0.9
	player.Bobbing.IdleDrift = 3.0 //2.0 - 0.01

	player.Bobbing.SpeedLerp = 0.30 // Reattività istantanea alla velocità
	player.Bobbing.AmpLerp = 0.20
	player.Bobbing.ImpactMax = 1000.0
	player.Bobbing.ImpactScale = 0.02   // ATTERRAGGIO: 1000 * 0.02 = 20 unità di scuotimento verticale
	player.Bobbing.SpringTension = 0.20 // Molla più rigida (ritorno rapido)
	player.Bobbing.SpringDamping = 0.80
	player.Bobbing.TiltAmp = 0.05

	return player
}

// calculateOpenDoorCeil computes the ceiling height of a sector if it functions as a door, falling back to the sector's original height.
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

// createSectorsEdges constructs a 2D slice of edges for each sector in a given level based on its LineDefs and Vertexes.
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

// configureMaterial assigns physically differentiated specular and detail texture properties based on the texture name.
func (bld *Builder) configureMaterial(mat *config.Material, texName string) {
	if mat == nil {
		return
	}
	spec, shine, dScale, dFactor := bld.heuristicMaterialProperties(texName)
	mat.SpecularBoost = spec
	mat.Shininess = shine
	mat.DetailScale = dScale
	mat.DetailFactor = dFactor
}

// configureThingMaterial assigns physically differentiated specular and shininess properties for game entities/sprites.
func (bld *Builder) configureThingMaterial(mat *config.Material, spriteName string) {
	if mat == nil {
		return
	}
	spec, shine := bld.heuristicThingMaterialProperties(spriteName)
	mat.SpecularBoost = spec
	mat.Shininess = shine
	mat.DetailFactor = 0.0 // Preserve clean 2D sprite art
}

// heuristicMaterialProperties deduces realistic physical specular and detail texture properties (scale & factor)
// based on canonical Doom 1 & Doom 2 texture naming conventions.
func (bld *Builder) heuristicMaterialProperties(texName string) (float64, float64, float32, float32) {
	name := strings.TrimSpace(strings.ToUpper(texName))
	if name == "" || name == "-" {
		return 0.0, 0.0, 1.0, 0.0
	}

	// Sky: no specular reflections or detail noise
	if name == SkyPicture || strings.Contains(name, "SKY") {
		return 0.0, 0.0, 1.0, 0.0
	}

	// Liquids: highly reflective, fluid mirror surface without rock/metal grain
	if strings.Contains(name, "NUKAGE") || strings.Contains(name, "SLIME") ||
		strings.Contains(name, "LAVA") || strings.Contains(name, "BLOOD") ||
		strings.Contains(name, "FWATER") || strings.Contains(name, "WATER") {
		return 0.75, 95.0, 1.0, 0.0
	}

	// Glass, Screens, High-tech lights, Computer terminals: intense sharp highlights, subtle phosphor grain
	if strings.HasPrefix(name, "LITE") || strings.Contains(name, "COMP") ||
		strings.Contains(name, "CONS") || strings.Contains(name, "GLASS") {
		return 0.85, 110.0, 32.0, 0.12
	}

	// Metals, Tech panels, Doors, Pipes, Steels, Crates: clear metallic response and brushed fine grain
	if strings.HasPrefix(name, "STARTAN") || strings.HasPrefix(name, "TEKWALL") ||
		strings.HasPrefix(name, "SILVER") || strings.HasPrefix(name, "SHAWN") ||
		strings.Contains(name, "DOOR") || strings.Contains(name, "SUPPORT") ||
		strings.Contains(name, "PIPE") || strings.Contains(name, "BRONZE") ||
		strings.Contains(name, "METAL") || strings.Contains(name, "PANEL") ||
		strings.Contains(name, "SPACE") || strings.Contains(name, "PLAT") ||
		strings.Contains(name, "STEP") || strings.HasPrefix(name, "FLOOR4_") ||
		strings.HasPrefix(name, "FLOOR5_") || strings.HasPrefix(name, "CEIL5_") ||
		strings.HasPrefix(name, "GATE") || strings.HasPrefix(name, "CRAT") {
		return 0.65, 75.0, 24.0, 0.22
	}

	// Polished Marble / Temple stones: smooth surface with moderate highlight and delicate veins
	if strings.HasPrefix(name, "MARB") || strings.Contains(name, "MARBLE") ||
		strings.HasPrefix(name, "DEM1_") || strings.HasPrefix(name, "DEM2_") {
		return 0.35, 45.0, 20.0, 0.18
	}

	// Flesh, Skin, Organic hell walls: moist, damp organic sheen and porous skin micro-texture
	if strings.HasPrefix(name, "SKIN") || strings.Contains(name, "SNAKE") ||
		strings.Contains(name, "SKSPINE") || strings.Contains(name, "SP_DUDE") ||
		strings.Contains(name, "SP_FACE") || strings.Contains(name, "SKFACES") ||
		strings.HasPrefix(name, "SFLAT") {
		return 0.30, 28.0, 14.0, 0.30
	}

	// Wood: soft diffuse reflection and grain texture
	if strings.Contains(name, "WOOD") || strings.Contains(name, "PANCASE") ||
		strings.Contains(name, "PANBORDER") || strings.HasPrefix(name, "CEIL1_") {
		return 0.12, 16.0, 18.0, 0.25
	}

	// Rough Stone, Bricks, Cement, Dirt, Soil, Rocks, Grass: matte surface with pronounced porous detail
	if strings.Contains(name, "BROWN") || strings.Contains(name, "BRICK") ||
		strings.Contains(name, "STONE") || strings.Contains(name, "ROCK") ||
		strings.Contains(name, "GRAY") || strings.Contains(name, "STUCCO") ||
		strings.Contains(name, "ASHWALL") || strings.HasPrefix(name, "FLAT1") ||
		strings.HasPrefix(name, "FLAT5") || strings.HasPrefix(name, "FLAT10") ||
		strings.HasPrefix(name, "FLOOR0_") || strings.Contains(name, "GRASS") ||
		strings.Contains(name, "DIRT") || strings.Contains(name, "MUCK") {
		return 0.05, 8.0, 16.0, 0.38
	}

	// Default generic fallback
	return 0.15, 30.0, 16.0, 0.25
}

// heuristicThingMaterialProperties determines specular properties for items, weapons, barrels, and creatures.
func (bld *Builder) heuristicThingMaterialProperties(spriteName string) (float64, float64) {
	s := strings.TrimSpace(strings.ToUpper(spriteName))
	if len(s) >= 4 {
		s = s[:4]
	}

	// Tech, metal items: barrels, armor, keys, weapons
	if s == "BAR1" || s == "BEXP" || // Barrels
		s == "ARM1" || s == "ARM2" || s == "BON1" || s == "BON2" || // Armor & Helmets
		s == "BFUG" || s == "CSAW" || s == "LAUN" || s == "MGUN" || s == "PLAS" || s == "SHOT" || s == "SGN2" || // Weapons
		s == "BKEY" || s == "RKEY" || s == "YKEY" || s == "BSKU" || s == "RSKU" || s == "YSKU" || // Keys & Skulls
		s == "SOUL" || s == "MEGA" || s == "PINV" || s == "PSTR" || s == "PINS" || s == "SUIT" { // Artifacts & Tech
		return 0.70, 75.0
	}

	// Cyborg / Mechanical Bosses (Cyberdemon, Spider Mastermind, Arachnotron)
	if s == "CYBR" || s == "SPID" || s == "BSPI" {
		return 0.45, 55.0
	}

	// Organic monsters (Imp, Cacodemon, Pinky Demon, Hell Knight, Baron, etc.)
	return 0.25, 25.0
}

// heuristicMaterial determines the material type and whether it matches known liquid kinds based on the texture name.
func (bld *Builder) heuristicMaterial(floorPic string) (config.MaterialKind, bool) {
	floorPic = strings.TrimSpace(strings.ToUpper(floorPic))
	// Acido / Radioattività (Nukage, Slime) -> Verde
	if strings.Contains(floorPic, "NUKAGE") || strings.Contains(floorPic, "SLIME") {
		return config.MaterialKindLiquid, true
	}
	// Lava / Sangue -> Rosso
	if strings.Contains(floorPic, "LAVA") || strings.Contains(floorPic, "BLOOD") {
		return config.MaterialKindLiquid, true
	}
	// Acqua / Computer -> Blu
	if strings.Contains(floorPic, "FWATER") {
		return config.MaterialKindLiquid, true
	}
	return config.MaterialKindNone, false
}

// heuristicLight determines the light intensity, kind, falloff, and color values for a scene based on provided parameters.
func (bld *Builder) heuristicLight(lightLevel int16, ceilPic string, ceilY float64, floorPic string, floorY float64, edges []Edge) (float64, config.LightKind, float64, float64, float64, float64) {
	intensity := float64(lightLevel) * ScaleLight
	kind := config.LightKindAmbient
	falloff := ((ceilY - floorY) * ScaleSectorH) * 2.0
	r, g, b := 1.0, 0.95, 0.9

	floorPic = strings.TrimSpace(strings.ToUpper(floorPic))
	ceilPic = strings.TrimSpace(strings.ToUpper(ceilPic))

	// --- EURISTICA DEL COLORE ---
	// 1. Acido / Radioattività (Nukage, Slime) -> Verde
	if strings.Contains(floorPic, "NUKAGE") || strings.Contains(floorPic, "SLIME") {
		r, g, b = 0.2, 1.0, 0.2
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
		intensity *= 3.0
		if falloff < 10.0 {
			falloff = 10.0
		}
	}
	return intensity, kind, falloff, r, g, b
}
