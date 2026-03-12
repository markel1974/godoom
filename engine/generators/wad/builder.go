package wad

import (
	"fmt"
	"math"
	"sort"
	"strconv"

	"github.com/markel1974/godoom/engine/generators/wad/lumps"
	"github.com/markel1974/godoom/engine/model"
)

// ScaleFactorLineDef defines the scale factor applied to line definitions for coordinate normalization in the configuration.
const ScaleFactorLineDef = 25.0

// ScaleFactorCeilFloorLineDef is a constant scaling factor used to convert floor and ceiling heights into game unit measurements.
const ScaleFactorCeilFloorLineDef = 8.0

const SkyPicture = "F_SKY1"

// Point represents a 2D coordinate with X and Y as floating-point values.
type Point struct {
	X float64
	Y float64
}

// ToModelXY converts a Point instance to a model.XY structure with identical X and Y coordinate values.
func (p Point) ToModelXY() model.XY { return model.XY{X: p.X, Y: p.Y} }

// Triangle represents a geometric shape consisting of three vertices defined by points in 2D space.
type Triangle struct {
	A, B, C Point
}

// HasVertex checks whether the given point is one of the vertices of the triangle.
func (t Triangle) HasVertex(p Point) bool {
	return t.A == p || t.B == p || t.C == p
}

// HasEdge checks if the given edge exists in the triangle, considering both possible orientations of the edge.
func (t Triangle) HasEdge(e [2]Point) bool {
	eRev := [2]Point{e[1], e[0]}
	tEdges := [3][2]Point{{t.A, t.B}, {t.B, t.C}, {t.C, t.A}}
	for _, te := range tEdges {
		if te == e || te == eRev {
			return true
		}
	}
	return false
}

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

// PolygonDef represents a polygon with an outer boundary and optional holes defined as collections of points.
type PolygonDef struct {
	Outer []Point
	Holes [][]Point
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
		polygonDefs := traceLoops(level, edges)

		for loopIdx, def := range polygonDefs {
			mergedPoly := mergeHoles(def)
			triangles := triangulate(mergedPoly)

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
					fmt.Println("WARNING: Missing edge for join segment: ", cs.Parent)
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

		if pointInTriangle(p, v1, v2, v3) {
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

// maxPointsX returns the maximum X coordinate value from a slice of Point objects.
func maxPointsX(poly []Point) float64 {
	max := poly[0].X
	for _, p := range poly {
		if p.X > max {
			max = p.X
		}
	}
	return max
}

// mergeHoles connects holes and outer boundaries in a polygon definition into a single ordered list of points.
func mergeHoles(def PolygonDef) []Point {
	if len(def.Holes) == 0 {
		return def.Outer
	}

	// Optimization 1: Exact calculation of final capacity to eliminate dynamic reallocations
	totalLen := len(def.Outer)
	for _, h := range def.Holes {
		totalLen += len(h) + 2 // +2 for bridge vertices (forward and return)
	}

	outer := make([]Point, len(def.Outer), totalLen)
	copy(outer, def.Outer)

	// Sort holes from right to left to ensure topological consistency in bridging
	sort.Slice(def.Holes, func(i, j int) bool {
		return maxPointsX(def.Holes[i]) > maxPointsX(def.Holes[j])
	})

	for _, hole := range def.Holes {
		holeIdx := 0
		mX := hole[0].X
		for i := 1; i < len(hole); i++ {
			if hole[i].X > mX {
				mX = hole[i].X
				holeIdx = i
			}
		}
		holePoint := hole[holeIdx]
		bestOuterIdx := -1
		minDist := math.MaxFloat64

		for i, op := range outer {
			if op.X < holePoint.X {
				continue
			}

			// Optimization 2: Fast rejection. Calculate distance in O(1) before
			// launching isVisible (which is O(N) for each segment).
			if dist := distanceSq(holePoint, op); dist < minDist {
				if isVisible(holePoint, op, hole, outer) {
					minDist = dist
					bestOuterIdx = i
				}
			}
		}

		// Topological fallback for non-manifold sectors or anomalous intersections
		if bestOuterIdx == -1 {
			bestOuterIdx = 0
			for i, op := range outer {
				if dist := distanceSq(holePoint, op); dist < minDist {
					minDist = dist
					bestOuterIdx = i
				}
			}
		}

		// Optimization 3: In-place memory shifting leveraging pre-allocated capacity.
		// No additional heap allocation for new bridges.
		oldLen := len(outer)
		spliceLen := len(hole) + 2
		outer = append(outer, make([]Point, spliceLen)...)

		// Forward shift of elements to the right of the insertion point
		copy(outer[bestOuterIdx+1+spliceLen:], outer[bestOuterIdx+1:oldLen])

		// Linear reconstruction of the bridge
		insertPos := bestOuterIdx + 1
		for i := 0; i < len(hole); i++ {
			outer[insertPos+i] = hole[(holeIdx+i)%len(hole)]
		}
		outer[insertPos+len(hole)] = holePoint
		outer[insertPos+len(hole)+1] = outer[bestOuterIdx]
	}

	return outer
}

// isVisible checks if the line segment between p1 and p2 does not intersect any edge from the hole or outer polygon.
func isVisible(p1, p2 Point, hole, outer []Point) bool {
	for i := 0; i < len(outer); i++ {
		e1, e2 := outer[i], outer[(i+1)%len(outer)]
		if e1 == p1 || e1 == p2 || e2 == p1 || e2 == p2 {
			continue
		}
		if segmentsIntersect(p1, p2, e1, e2) {
			return false
		}
	}
	for i := 0; i < len(hole); i++ {
		e1, e2 := hole[i], hole[(i+1)%len(hole)]
		if e1 == p1 || e1 == p2 || e2 == p1 || e2 == p2 {
			continue
		}
		if segmentsIntersect(p1, p2, e1, e2) {
			return false
		}
	}
	return true
}

// distanceSq calculates and returns the squared Euclidean distance between two points p1 and p2.
func distanceSq(p1, p2 Point) float64 {
	dx := p1.X - p2.X
	dy := p1.Y - p2.Y
	return dx*dx + dy*dy
}

// pointInTriangle determines if a point p lies inside or on the edges of a triangle formed by vertices a, b, and c.
func pointInTriangle(p, a, b, c Point) bool {
	cp1 := (b.X-a.X)*(p.Y-a.Y) - (b.Y-a.Y)*(p.X-a.X)
	cp2 := (c.X-b.X)*(p.Y-b.Y) - (c.Y-b.Y)*(p.X-b.X)
	cp3 := (a.X-c.X)*(p.Y-c.Y) - (a.Y-c.Y)*(p.X-c.X)

	const eps = 0.5
	return (cp1 >= -eps && cp2 >= -eps && cp3 >= -eps) || (cp1 <= eps && cp2 <= eps && cp3 <= eps)
}

// traceLoops constructs closed polygon definitions (outers and holes) from a set of edges for a given Level.
func traceLoops(level *Level, edges []Edge) []PolygonDef {
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

	var rawLoops [][]Point

	for _, startEdge := range edges {
		vIdx := getVisitedIdx(startEdge)
		if visited[vIdx] {
			continue
		}

		var currentLoop []Point
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

	var outers [][]Point
	var holes [][]Point

	maxArea := 0.0
	outerSign := 1.0

	areas := make([]float64, len(rawLoops))
	for i, loop := range rawLoops {
		areas[i] = signedArea(loop)
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

	defs := make([]PolygonDef, len(outers))
	for i, o := range outers {
		defs[i] = PolygonDef{Outer: o}
	}

	for _, h := range holes {
		for i, def := range defs {
			if pointInPolygon(h[0], def.Outer) {
				defs[i].Holes = append(defs[i].Holes, h)
				break
			}
		}
	}

	return defs
}

// GEOMETRY & CDT ENGINE

// triangulate performs constrained Delaunay triangulation on a polygon with optional holes.
// It returns a list of triangles, each represented by 3 counterclockwise points.
func triangulate(poly []Point) [][]Point {
	if len(poly) < 3 {
		return nil
	}

	// 1. Deserialization: extract Outer and Holes from flat array separated by NaN
	var outer []Point
	var holes [][]Point
	var current []Point

	for _, p := range poly {
		if math.IsNaN(p.X) || math.IsNaN(p.Y) {
			if outer == nil {
				outer = current
			} else {
				holes = append(holes, current)
			}
			current = nil
		} else {
			current = append(current, p)
		}
	}
	if outer == nil {
		outer = current
	} else if len(current) > 0 {
		holes = append(holes, current)
	}

	// 2. Collect all valid vertices
	var points []Point
	points = append(points, outer...)
	for _, h := range holes {
		points = append(points, h...)
	}

	if len(points) < 3 {
		return nil
	}

	// 3. Unconstrained Delaunay Triangulation (Bowyer-Watson)
	mesh := bowyerWatson(points)

	// 4. Constraint Recovery (Lawson Edge Flipping)
	var constraints [][2]Point
	constraints = append(constraints, buildConstraints(outer)...)
	for _, hole := range holes {
		constraints = append(constraints, buildConstraints(hole)...)
	}
	mesh = recoverConstraints(mesh, constraints)

	// 5. Domain Culling
	var finalTriangles [][]Point
	for _, t := range mesh {
		// Calculate centroid for containment test
		centroid := Point{
			X: (t.A.X + t.B.X + t.C.X) / 3.0,
			Y: (t.A.Y + t.B.Y + t.C.Y) / 3.0,
		}

		// The triangle is valid if it's inside the perimeter and outside all holes
		if pointInPolygon(centroid, outer) {
			inHole := false
			for _, hole := range holes {
				if pointInPolygon(centroid, hole) {
					inHole = true
					break
				}
			}
			if !inHole {
				// Ensure counterclockwise (CCW) winding required by renderer
				if orientation(t.A, t.B, t.C) == 2 {
					finalTriangles = append(finalTriangles, []Point{t.A, t.C, t.B})
				} else {
					finalTriangles = append(finalTriangles, []Point{t.A, t.B, t.C})
				}
			}
		}
	}

	return finalTriangles
}

// buildConstraints generates edge constraints from a polygon by connecting consecutive vertices in cyclical order.
func buildConstraints(poly []Point) [][2]Point {
	var c [][2]Point
	if len(poly) < 3 {
		return c
	}
	for i := 0; i < len(poly); i++ {
		c = append(c, [2]Point{poly[i], poly[(i+1)%len(poly)]})
	}
	return c
}

// bowyerWatson performs the Bowyer-Watson algorithm for constructing a Delaunay triangulation from a set of points.
func bowyerWatson(points []Point) []Triangle {
	minX, minY := math.MaxFloat64, math.MaxFloat64
	maxX, maxY := -math.MaxFloat64, -math.MaxFloat64

	for _, p := range points {
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

	dx, dy := maxX-minX, maxY-minY
	deltaMax := math.Max(dx, dy)
	if deltaMax == 0 {
		deltaMax = 1.0
	}
	midX, midY := (minX+maxX)/2.0, (minY+maxY)/2.0

	// Bounding super-triangle that wraps the entire area
	stA := Point{X: midX - 20*deltaMax, Y: midY - deltaMax}
	stB := Point{X: midX, Y: midY + 20*deltaMax}
	stC := Point{X: midX + 20*deltaMax, Y: midY - deltaMax}

	triangles := []Triangle{{stA, stB, stC}}

	for _, p := range points {
		var badTriangles []Triangle
		var polygon [][2]Point

		for _, t := range triangles {
			// Force CCW orientation for exact InCircle determinant
			if orientation(t.A, t.B, t.C) == 2 {
				if inCircle(t.A, t.B, t.C, p) {
					badTriangles = append(badTriangles, t)
				}
			} else {
				if inCircle(t.A, t.C, t.B, p) {
					badTriangles = append(badTriangles, t)
				}
			}
		}

		for _, bt := range badTriangles {
			edges := [3][2]Point{{bt.A, bt.B}, {bt.B, bt.C}, {bt.C, bt.A}}
			for _, edge := range edges {
				shared := false
				for _, other := range badTriangles {
					if bt == other {
						continue
					}
					if other.HasEdge(edge) {
						shared = true
						break
					}
				}
				if !shared {
					polygon = append(polygon, edge)
				}
			}
		}

		var nextTriangles []Triangle
		for _, t := range triangles {
			isBad := false
			for _, bt := range badTriangles {
				if t == bt {
					isBad = true
					break
				}
			}
			if !isBad {
				nextTriangles = append(nextTriangles, t)
			}
		}
		triangles = nextTriangles

		for _, edge := range polygon {
			triangles = append(triangles, Triangle{edge[0], edge[1], p})
		}
	}

	var finalTriangles []Triangle
	for _, t := range triangles {
		if !t.HasVertex(stA) && !t.HasVertex(stB) && !t.HasVertex(stC) {
			finalTriangles = append(finalTriangles, t)
		}
	}

	return finalTriangles
}

// recoverConstraints enforces WAD segments on the mesh through Edge Flipping,
// preventing infinite loops on non-convex quadrilaterals.
func recoverConstraints(triangles []Triangle, constraints [][2]Point) []Triangle {
	const failsafeMax = 2000
	for _, c := range constraints {
		failsafe := 0
		for {
			failsafe++
			if failsafe > failsafeMax {
				break // Absolute failsafe: impossible/degenerate topology in the WAD
			}

			flipped := false

			for i, t := range triangles {
				edges := [3][2]Point{{t.A, t.B}, {t.B, t.C}, {t.C, t.A}}
				for _, e := range edges {
					if e[0] == c[0] || e[0] == c[1] || e[1] == c[0] || e[1] == c[1] {
						continue
					}

					if segmentsIntersect(e[0], e[1], c[0], c[1]) {
						adjIdx := -1
						for j, ot := range triangles {
							if i == j {
								continue
							}
							if ot.HasEdge(e) {
								adjIdx = j
								break
							}
						}

						if adjIdx != -1 {
							t1 := triangles[i]
							t2 := triangles[adjIdx]

							var pOpp1, pOpp2 Point
							for _, v := range []Point{t1.A, t1.B, t1.C} {
								if v != e[0] && v != e[1] {
									pOpp1 = v
									break
								}
							}
							for _, v := range []Point{t2.A, t2.B, t2.C} {
								if v != e[0] && v != e[1] {
									pOpp2 = v
									break
								}
							}

							// CONVEXITY TEST: the new diagonal (pOpp1-pOpp2)
							// must leave the old vertices (e[0], e[1]) on opposite sides.
							o1 := orientation(pOpp1, pOpp2, e[0])
							o2 := orientation(pOpp1, pOpp2, e[1])

							if o1 != o2 && o1 != 0 && o2 != 0 {
								// Convex quadrilateral: safe flip
								triangles[i] = Triangle{pOpp1, pOpp2, e[0]}
								triangles[adjIdx] = Triangle{pOpp1, pOpp2, e[1]}
								flipped = true
								break // Restart scanning with the new topology
							}
						}
					}
				}
				if flipped {
					break
				}
			}

			// If we scanned all triangles without making valid flips,
			// the constraint is resolved or blocked on unsolvable degeneracies.
			if !flipped {
				break
			}
		}
	}
	return triangles
}

// inCircle determines if a point `d` lies within the circumcircle of triangle formed by points `a`, `b`, and `c`.
func inCircle(a, b, c, d Point) bool {
	adx, ady := a.X-d.X, a.Y-d.Y
	bdx, bdy := b.X-d.X, b.Y-d.Y
	cdx, cdy := c.X-d.X, c.Y-d.Y

	abDet := adx*bdy - bdx*ady
	bcDet := bdx*cdy - cdx*bdy
	caDet := cdx*ady - adx*cdy

	aLift := adx*adx + ady*ady
	bLift := bdx*bdx + bdy*bdy
	cLift := cdx*cdx + cdy*cdy

	return aLift*bcDet+bLift*caDet+cLift*abDet > 0
}

// orientation calculates the orientation of three points (p, q, r).
// Returns 0 if collinear, 1 if clockwise, and 2 if counterclockwise.
func orientation(p, q, r Point) int {
	val := (q.Y-p.Y)*(r.X-q.X) - (q.X-p.X)*(r.Y-q.Y)
	if val == 0 {
		return 0
	}
	if val > 0 {
		return 1
	}
	return 2
}

// onSegment checks if point q lies on the line segment defined by points p and r.
func onSegment(p, q, r Point) bool {
	return q.X <= math.Max(p.X, r.X) && q.X >= math.Min(p.X, r.X) &&
		q.Y <= math.Max(p.Y, r.Y) && q.Y >= math.Min(p.Y, r.Y)
}

// segmentsIntersect checks whether two line segments (p1-q1 and p2-q2) intersect. Uses orientation and colinearity checks.
func segmentsIntersect(p1, q1, p2, q2 Point) bool {
	o1 := orientation(p1, q1, p2)
	o2 := orientation(p1, q1, q2)
	o3 := orientation(p2, q2, p1)
	o4 := orientation(p2, q2, q1)

	if o1 != o2 && o3 != o4 {
		return true
	}
	if o1 == 0 && onSegment(p1, p2, q1) {
		return true
	}
	if o2 == 0 && onSegment(p1, q2, q1) {
		return true
	}
	if o3 == 0 && onSegment(p2, p1, q2) {
		return true
	}
	if o4 == 0 && onSegment(p2, q1, q2) {
		return true
	}

	return false
}

// pointInPolygon determines whether a point is inside a given polygon using the ray-casting algorithm.
// p represents the point to test.
// poly is the array of Points defining the polygon, ordered either clockwise or counterclockwise.
// Returns true if the point is inside the polygon; otherwise, returns false.
func pointInPolygon(p Point, poly []Point) bool {
	inside := false
	for i, j := 0, len(poly)-1; i < len(poly); j, i = i, i+1 {
		xi, yi := poly[i].X, poly[i].Y
		xj, yj := poly[j].X, poly[j].Y
		if ((yi > p.Y) != (yj > p.Y)) && (p.X < (xj-xi)*(p.Y-yi)/(yj-yi)+xi) {
			inside = !inside
		}
	}
	return inside
}

// signedArea calculates the signed area of a polygon defined by a slice of Points, using the shoelace formula.
func signedArea(poly []Point) float64 {
	var area float64
	for i := 0; i < len(poly); i++ {
		p1, p2 := poly[i], poly[(i+1)%len(poly)]
		area += p1.X*p2.Y - p2.X*p1.Y
	}
	return area / 2.0
}
