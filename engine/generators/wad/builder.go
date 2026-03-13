package wad

import (
	"fmt"
	"math"
	"strconv"

	"github.com/markel1974/godoom/engine/generators/wad/lumps"
	"github.com/markel1974/godoom/engine/model"
)

// ScaleFactorLineDef defines the scale factor applied to line definitions for coordinate normalization in the configuration.
const ScaleFactorLineDef = 25.0

// ScaleFactorCeilFloorLineDef is a constant scaling factor used to convert floor and ceiling heights into game unit measurements.
const ScaleFactorCeilFloorLineDef = 8.0

const SkyPicture = "F_SKY1"

// Edge represents a connection between two vertices in a graph or geometric structure.
// V1 and V2 are the indices of the vertices connected by the edge.
// LDIdx is an index associated with a line definition in the context of the level structure.
// IsLeft indicates whether the edge corresponds to the left side of a line definition.
type Edge struct {
	V1, V2 uint16
	LDIdx  int
	IsLeft bool
}

// EdgeKey represents a unique key for an edge defined by its start and end points in 2D space.
type EdgeKey struct {
	X1, Y1, X2, Y2 float64
}

// Builder provides utilities for constructing or processing line definitions within a WAD file.
type Builder struct {
	w        *WAD
	textures *Textures
}

// NewBuilder initializes and returns a new instance of Builder.
func NewBuilder() *Builder {
	return &Builder{}
}

// Setup initializes the level configuration by loading data from a WAD file and constructing sectors, player, and things.
func (bld *Builder) Setup(wadFile string, levelNumber int) (*model.ConfigRoot, error) {
	bld.w = New()
	if err := bld.w.Load(wadFile); err != nil {
		return nil, err
	}
	bld.textures = bld.w.GetTextures()
	levelNames := bld.w.GetLevels()
	if levelNumber < 1 || levelNumber > len(levelNames) {
		return nil, fmt.Errorf("invalid level number: %d", levelNumber)
	}
	level, err := bld.w.GetLevel(levelNames[levelNumber-1])
	if err != nil {
		return nil, err
	}

	sectors := bld.buildSectorsFromLineDefs(level)

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
		tSectorId := bld.resolveSectorId(Point{tX, tY}, sectors)
		tId := fmt.Sprintf("t_%d", i)
		cfgThing := model.NewConfigThing(tId, model.XY{X: tX, Y: -tY}, tAngle, int(t.Type), tSectorId)
		things = append(things, cfgThing)
	}

	playerSectorId := bld.resolveSectorId(Point{pX, pY}, sectors)
	player := model.NewConfigPlayer(
		model.XY{X: pX, Y: -pY},
		pAngle,
		playerSectorId,
	)

	return model.NewConfigRoot(sectors, player, things, ScaleFactorLineDef, true, bld.textures), nil
}

// buildSectorsFromLineDefs processes linedefs in a level to build and return a list of ConfigSector objects.
func (bld *Builder) buildSectorsFromLineDefs(level *Level) []*model.ConfigSector {
	sectorToEdges := make(map[uint16][]Edge)
	for i, ld := range level.LineDefs {
		if ld.SideDefRight != -1 {
			s := level.SideDefs[ld.SideDefRight].SectorRef
			sectorToEdges[s] = append(sectorToEdges[s], Edge{uint16(ld.VertexStart), uint16(ld.VertexEnd), i, false})
		}
		if ld.SideDefLeft != -1 {
			s := level.SideDefs[ld.SideDefLeft].SectorRef
			sectorToEdges[s] = append(sectorToEdges[s], Edge{uint16(ld.VertexEnd), uint16(ld.VertexStart), i, true})
		}
	}

	var cSectors []*model.ConfigSector
	edgeMap := make(map[EdgeKey]string)
	wadLines := make(map[*model.ConfigSegment]bool)

	for secIdx, edges := range sectorToEdges {
		wadSector := level.Sectors[secIdx]
		polygonDefs := bld.traceLoops(level, edges)

		if secIdx == 7 {
			fmt.Printf("Polygon: %v\n", edges)
		}

		for loopIdx, def := range polygonDefs {
			mergedPoly := def.BridgeHoles()

			triangles := mergedPoly.Triangulate(int(secIdx))

			for triIdx, tri := range triangles {
				cSector := bld.buildConfigSector(level, wadSector, secIdx, loopIdx, triIdx, edges)
				for k := 0; k < 3; k++ {
					p1 := tri[k]
					p2 := tri[(k+1)%3]
					cSeg, isWadLine := bld.buildConfigSegment(level, cSector.Id, p1, p2, edges)
					cSector.Segments = append(cSector.Segments, cSeg)
					wadLines[cSeg] = isWadLine
					key := EdgeKey{cSeg.Start.X, cSeg.Start.Y, cSeg.End.X, cSeg.End.Y}
					edgeMap[key] = cSeg.Parent
				}
				cSectors = append(cSectors, cSector)
			}
		}
	}

	for _, cf := range cSectors {
		for _, cs := range cf.Segments {
			if cs.Kind == model.DefinitionJoin {
				reverseKey := EdgeKey{cs.End.X, cs.End.Y, cs.Start.X, cs.Start.Y}
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

// buildConfigSector converts a WAD sector to a ConfigSector, assigning texture, height, light level, and ID properties.
func (bld *Builder) buildConfigSector(level *Level, wadSector *lumps.Sector, secIdx uint16, loopIdx int, triIdx int, edges []Edge) *model.ConfigSector {
	const openAllDoors = true
	sectorId := fmt.Sprintf("s%d_l%d_t%d", secIdx, loopIdx, triIdx)
	miSector := model.NewConfigSector(sectorId)
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
	miSector.Animations.Ceil = model.NewConfigAnimation(bld.textures.FlatCreateAnimation(wadSector.CeilingPic), ceilingType)
	miSector.Animations.Floor = model.NewConfigAnimation(bld.textures.FlatCreateAnimation(wadSector.FloorPic), floorType)
	miSector.Animations.ScaleFactor = 10.0
	miSector.Light.Intensity = bld.convertLight(wadSector.LightLevel)
	miSector.Light.Kind = model.LightKindSpot
	return miSector
}

// buildConfigSegment generates a ConfigSegment based on a level's geometry, sector ID, points, and sector edges.
// It identifies if a matching edge exists, adjusts Y-coordinates, sets texture details, and determines the segment kind.
func (bld *Builder) buildConfigSegment(level *Level, sectorId string, p1, p2 Point, sectorEdges []Edge) (*model.ConfigSegment, bool) {
	seg := model.NewConfigSegment(sectorId, model.DefinitionWall, p1.ToModelXY(), p2.ToModelXY())
	for _, e := range sectorEdges {
		v1, v2 := level.Vertexes[e.V1], level.Vertexes[e.V2]
		w1 := Point{float64(v1.XCoord), float64(v1.YCoord)}
		w2 := Point{float64(v2.XCoord), float64(v2.YCoord)}

		if (p1 == w1 && p2 == w2) || (p1 == w2 && p2 == w1) {
			ld := level.LineDefs[e.LDIdx]

			sideIdx := ld.SideDefRight
			if e.IsLeft {
				sideIdx = ld.SideDefLeft
			}
			side := level.SideDefs[sideIdx]

			seg.Animations.Middle = model.NewConfigAnimation(bld.textures.TextureCreateAnimation(side.MiddleTexture), model.AnimationKindLoop)
			seg.Animations.Upper = model.NewConfigAnimation(bld.textures.TextureCreateAnimation(side.UpperTexture), model.AnimationKindLoop)
			seg.Animations.Lower = model.NewConfigAnimation(bld.textures.TextureCreateAnimation(side.LowerTexture), model.AnimationKindLoop)

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
						seg.Animations.Upper = model.NewConfigAnimation(nil, model.AnimationKindNone)
					}
					// Extension for floors (e.g. moats that show sky at the bottom)
					if frontSector.FloorPic == SkyPicture && backSector.FloorPic == SkyPicture {
						seg.Animations.Lower = model.NewConfigAnimation(nil, model.AnimationKindNone)
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

func (bld *Builder) calculateOpenDoorCeil(level *Level, secIdx uint16, wadSector *lumps.Sector, edges []Edge) float64 {
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

// resolveSectorId determines the sector ID for a given point by checking triangle containment or finding the nearest sector.
func (bld *Builder) resolveSectorId(p Point, sectors []*model.ConfigSector) string {
	if len(sectors) == 0 {
		return ""
	}
	var minDist = math.MaxFloat64
	closestSector := sectors[0].Id

	for _, s := range sectors {
		if len(s.Segments) != 3 {
			continue
		}

		v1 := Point{s.Segments[0].Start.X, -s.Segments[0].Start.Y}
		v2 := Point{s.Segments[1].Start.X, -s.Segments[1].Start.Y}
		v3 := Point{s.Segments[2].Start.X, -s.Segments[2].Start.Y}

		if PointInTriangle(p, v1, v2, v3) {
			return s.Id
		}

		cx := (v1.X + v2.X + v3.X) / 3.0
		cy := (v1.Y + v2.Y + v3.Y) / 3.0
		distSq := (cx-p.X)*(cx-p.X) + (cy-p.Y)*(cy-p.Y)

		if distSq < minDist {
			minDist = distSq
			closestSector = s.Id
		}
	}
	return closestSector
}

// traceLoops constructs closed polygon definitions (outers and holes) from a set of edges for a given Level.
func (bld *Builder) traceLoops(level *Level, edges []Edge) []ComplexPolygon {
	// Optimization: O(1) array access instead of map[uint16][]Edge
	adj := make([][]Edge, len(level.Vertexes))
	for _, e := range edges {
		adj[e.V1] = append(adj[e.V1], e)
	}

	// Optimization: flat bitmask/boolean array (LDIdx << 1 | IsLeft) instead of map[Edge]bool
	visited := make([]bool, len(level.LineDefs)*2)

	getVisitedIdx := func(e Edge) int {
		idx := e.LDIdx << 1
		if e.IsLeft {
			idx |= 1
		}
		return idx
	}

	var rawLoops []Polygon

	for _, startEdge := range edges {
		vIdx := getVisitedIdx(startEdge)
		if visited[vIdx] {
			continue
		}

		var currentLoop Polygon
		curr := startEdge
		for {
			visited[getVisitedIdx(curr)] = true
			v := level.Vertexes[curr.V1]
			currentLoop = append(currentLoop, Point{X: float64(v.XCoord), Y: float64(v.YCoord)})

			nextOptions := adj[curr.V2]
			var nextEdge Edge
			found := false
			for _, o := range nextOptions {
				if !visited[getVisitedIdx(o)] {
					nextEdge = o
					found = true
					break
				}
			}

			if !found || nextEdge.V1 == startEdge.V1 {
				break
			}
			curr = nextEdge
		}
		if len(currentLoop) >= 3 {
			rawLoops = append(rawLoops, currentLoop)
		}
	}

	if len(rawLoops) == 0 {
		return nil
	}

	var outers []Polygon
	var holes []Polygon

	maxArea := 0.0
	outerSign := 1.0

	areas := make([]float64, len(rawLoops))
	for i, loop := range rawLoops {
		areas[i] = loop.SignedArea()
		absArea := math.Abs(areas[i])
		if absArea > maxArea {
			maxArea = absArea
			if areas[i] < 0 {
				outerSign = -1.0
			} else {
				outerSign = 1.0
			}
		}
	}

	for i, loop := range rawLoops {
		if (areas[i] < 0 && outerSign < 0) || (areas[i] > 0 && outerSign > 0) {
			outers = append(outers, loop)
		} else {
			holes = append(holes, loop)
		}
	}

	defs := make([]ComplexPolygon, len(outers))
	for i, o := range outers {
		defs[i] = ComplexPolygon{Outer: o}
	}

	for _, h := range holes {
		for i, def := range defs {
			if def.Outer.PointInPolygon(h[0]) {
				defs[i].Holes = append(defs[i].Holes, h)
				break
			}
		}
	}

	return defs
}

// GEOMETRY & CDT ENGINE
