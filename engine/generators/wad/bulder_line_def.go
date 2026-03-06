package wad

import (
	"fmt"
	"math"
	"sort"
	"strconv"

	"github.com/markel1974/godoom/engine/generators/wad/lumps"
	"github.com/markel1974/godoom/engine/model"
)

// ScaleFactorLineDef defines the default scaling factor applied to line definitions when building configurations.
const ScaleFactorLineDef = 25.0

// ScaleFactorCeilFloorLineDef defines the scaling factor for converting WAD sector heights into engine-compatible units.
const ScaleFactorCeilFloorLineDef = 4.0

// Point represents a 2D point with X and Y coordinates as floating-point values.
type Point struct {
	X float64
	Y float64
}

// ToModelXY converts the Point instance into a model.XY structure with corresponding X and Y coordinates.
func (p Point) ToModelXY() model.XY { return model.XY{X: p.X, Y: p.Y} }

// Edge represents a connection between two vertices with additional metadata for level building in a 3D game engine.
type Edge struct {
	V1, V2 uint16
	LDIdx  int
	IsLeft bool
}

// EdgeKey represents a unique key for identifying an edge in a 2D space using its start (X1, Y1) and end (X2, Y2) coordinates.
type EdgeKey struct {
	X1, Y1, X2, Y2 float64
}

// PolygonDef defines a polygon with an outer boundary and optional holes, each represented as slices of Point structs.
type PolygonDef struct {
	Outer []Point
	Holes [][]Point
}

// BuilderLineDef encapsulates functionalities for handling and manipulating line definitions in a WAD file.
type BuilderLineDef struct {
	w *WAD
}

// NewBuilderLineDef creates and returns a new instance of BuilderLineDef struct.
func NewBuilderLineDef() *BuilderLineDef {
	return &BuilderLineDef{}
}

// Setup initializes the level configuration, builds sectors, places the player, and processes level objects.
func (bld *BuilderLineDef) Setup(wadFile string, levelNumber int) (*model.ConfigRoot, error) {
	bld.w = New()
	if err := bld.w.Load(wadFile); err != nil {
		return nil, err
	}
	levelNames := bld.w.GetLevels()
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

	t := bld.w.GetTextures()
	return model.NewConfigRoot(sectors, player, things, ScaleFactorLineDef, true, t), nil
}

// buildSectorsFromLineDefs constructs and returns a slice of ConfigSector objects from the LineDef data in the given Level.
func (bld *BuilderLineDef) buildSectorsFromLineDefs(level *Level) []*model.ConfigSector {
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
				// PROTEZIONE 1: Scartiamo i triangoli degeneri
				//if isDegenerate(tri[0], tri[1], tri[2]) {
				// continue
				//}

				cSector := bld.buildConfigSector(wadSector, secIdx, loopIdx, triIdx)
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
					// PROTEZIONE 2: Previene i Muri Fantasma!
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

// buildConfigSector constructs a ConfigSector from a given WAD Sector, assigning textures, light level, and other attributes.
func (bld *BuilderLineDef) buildConfigSector(wadSector *lumps.Sector, secIdx uint16, loopIdx int, triIdx int) *model.ConfigSector {
	sectorId := fmt.Sprintf("s%d_l%d_t%d", secIdx, loopIdx, triIdx)
	miSector := model.NewConfigSector(sectorId)
	miSector.Floor = float64(wadSector.FloorHeight) / ScaleFactorCeilFloorLineDef
	miSector.Ceil = float64(wadSector.CeilingHeight) / ScaleFactorCeilFloorLineDef
	miSector.Tag = strconv.Itoa(int(secIdx))
	miSector.TextureCeil = CreateFlatId(wadSector.CeilingPic)
	miSector.TextureFloor = CreateFlatId(wadSector.FloorPic)
	miSector.TextureScaleFactor = 10.0
	miSector.LightLevel = float64(wadSector.LightLevel) / 255
	return miSector
}

// buildConfigSegment constructs a configuration segment from a pair of points and sector-related edges, adjusting textures and kind.
func (bld *BuilderLineDef) buildConfigSegment(level *Level, sectorId string, p1, p2 Point, sectorEdges []Edge) (*model.ConfigSegment, bool) {
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
			seg.TextureMiddle = CreateTextureId(side.MiddleTexture)
			seg.TextureUpper = CreateTextureId(side.UpperTexture)
			seg.TextureLower = CreateTextureId(side.LowerTexture)
			if ld.Flags&(1<<2) != 0 {
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

// resolveSectorId determines the closest or containing sector ID for a given point using geometric computations.
func (bld *BuilderLineDef) resolveSectorId(p Point, sectors []*model.ConfigSector) string {
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

// GEOMETRY

// traceLoops identifies and organizes closed loops from a set of edges into polygons, distinguishing between outer and hole polygons.
func traceLoops(level *Level, edges []Edge) []PolygonDef {
	// Ottimizzazione: O(1) array access invece di map[uint16][]Edge
	adj := make([][]Edge, len(level.Vertexes))
	for _, e := range edges {
		adj[e.V1] = append(adj[e.V1], e)
	}

	// Ottimizzazione: flat bitmask/boolean array (LDIdx << 1 | IsLeft) invece di map[Edge]bool
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

// triangulate decomposes a simple polygon into a list of triangles using the ear-clipping method.
func triangulate(poly []Point) [][]Point {
	n := len(poly)
	if n < 3 {
		return nil
	}

	// Ottimizzazione: preallocazione esatta del numero di triangoli risultanti (N-2)
	triangles := make([][]Point, 0, n-2)
	working := make([]Point, n)
	copy(working, poly)

	if getWinding(working) < 0 {
		for i, j := 0, n-1; i < j; i, j = i+1, j-1 {
			working[i], working[j] = working[j], working[i]
		}
	}

	for len(working) > 2 {
		earFound := false
		for i := 0; i < len(working); i++ {
			prev := working[(i+len(working)-1)%len(working)]
			curr := working[i]
			next := working[(i+1)%len(working)]

			if isEar(prev, curr, next, working) {
				triangles = append(triangles, []Point{prev, curr, next})

				// Ottimizzazione: in-place slice shrinking per azzerare le allocazioni di append
				copy(working[i:], working[i+1:])
				working = working[:len(working)-1]

				earFound = true
				break
			}
		}

		if !earFound {
			bestIdx := 0
			minCp := math.MaxFloat64
			for i := 0; i < len(working); i++ {
				prev := working[(i+len(working)-1)%len(working)]
				curr := working[i]
				next := working[(i+1)%len(working)]

				cp := (curr.X-prev.X)*(next.Y-curr.Y) - (curr.Y-prev.Y)*(next.X-curr.X)
				if cp < minCp {
					minCp = cp
					bestIdx = i
				}
			}
			prev := working[(bestIdx+len(working)-1)%len(working)]
			curr := working[bestIdx]
			next := working[(bestIdx+1)%len(working)]

			triangles = append(triangles, []Point{prev, curr, next})

			copy(working[bestIdx:], working[bestIdx+1:])
			working = working[:len(working)-1]
		}
	}
	return triangles
}

// isEar determines if the triangle formed by points a, b, and c is an "ear" in the polygon poly.
// An "ear" is a triangle that is part of the polygon's triangulation and does not contain other points inside it.
// isEar determines if the triangle defined by points a, b, and c is an "ear".
func isEar(a, b, c Point, poly []Point) bool {
	if a.X == c.X && a.Y == c.Y {
		return true
	}

	cp := (b.X-a.X)*(c.Y-b.Y) - (b.Y-a.Y)*(c.X-b.X)
	if cp >= 0 {
		return false
	}

	// Calcolo AABB del triangolo per fast rejection nel loop di scansione vertici
	minX, maxX := a.X, a.X
	if b.X < minX {
		minX = b.X
	} else if b.X > maxX {
		maxX = b.X
	}
	if c.X < minX {
		minX = c.X
	} else if c.X > maxX {
		maxX = c.X
	}

	minY, maxY := a.Y, a.Y
	if b.Y < minY {
		minY = b.Y
	} else if b.Y > maxY {
		maxY = b.Y
	}
	if c.Y < minY {
		minY = c.Y
	} else if c.Y > maxY {
		maxY = c.Y
	}

	for _, p := range poly {
		// Fast rejection AABB: evita il dot-product se il punto è fuori dai limiti
		if p.X < minX || p.X > maxX || p.Y < minY || p.Y > maxY {
			continue
		}

		if (p.X == a.X && p.Y == a.Y) || (p.X == b.X && p.Y == b.Y) || (p.X == c.X && p.Y == c.Y) {
			continue
		}

		if pointInTriangleExact(p, a, b, c) {
			return false
		}
	}
	return true
}

// mergeHoles connects interior holes to the outer boundary of a polygon into a single contiguous loop of points.
func mergeHoles(def PolygonDef) []Point {
	if len(def.Holes) == 0 {
		return def.Outer
	}

	// Ottimizzazione 1: Calcolo esatto della capacità finale per azzerare le re-allocazioni dinamiche
	totalLen := len(def.Outer)
	for _, h := range def.Holes {
		totalLen += len(h) + 2 // +2 per i vertici di bridge (andata e ritorno)
	}

	outer := make([]Point, len(def.Outer), totalLen)
	copy(outer, def.Outer)

	// Ordina i buchi da destra a sinistra per garantire coerenza topologica nel bridge
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

			// Ottimizzazione 2: Fast rejection. Calcolo la distanza in O(1) prima
			// di lanciare isVisible (che è O(N) per ogni segmento).
			dist := distanceSq(holePoint, op)
			if dist < minDist {
				if isVisible(holePoint, op, hole, outer) {
					minDist = dist
					bestOuterIdx = i
				}
			}
		}

		// Fallback topologico per settori non-manifold o intersezioni anomale
		if bestOuterIdx == -1 {
			bestOuterIdx = 0
			for i, op := range outer {
				dist := distanceSq(holePoint, op)
				if dist < minDist {
					minDist = dist
					bestOuterIdx = i
				}
			}
		}

		// Ottimizzazione 3: In-place memory shifting sfruttando la capacity pre-allocata.
		// Nessuna allocazione heap aggiuntiva per i nuovi bridge.
		oldLen := len(outer)
		spliceLen := len(hole) + 2
		outer = append(outer, make([]Point, spliceLen)...)

		// Shift in avanti degli elementi a destra del punto di inserimento
		copy(outer[bestOuterIdx+1+spliceLen:], outer[bestOuterIdx+1:oldLen])

		// Ricostruzione lineare del bridge
		insertPos := bestOuterIdx + 1
		for i := 0; i < len(hole); i++ {
			outer[insertPos+i] = hole[(holeIdx+i)%len(hole)]
		}
		outer[insertPos+len(hole)] = holePoint
		outer[insertPos+len(hole)+1] = outer[bestOuterIdx]
	}

	return outer
}

// getWinding calculates the winding order of a polygon, returning 1 for counter-clockwise, -1 for clockwise, and 0 if undefined.
func getWinding(poly []Point) int64 {
	var area float64
	for i := 0; i < len(poly); i++ {
		p1, p2 := poly[i], poly[(i+1)%len(poly)]
		area += (p2.X - p1.X) * (p2.Y + p1.Y)
	}
	if area > 0 {
		return 1
	}
	if area < 0 {
		return -1
	}
	return 0
}

// signedArea calculates the signed area of a polygon represented by an array of Point structs.
// Positive value indicates counter-clockwise order, negative indicates clockwise.
func signedArea(poly []Point) float64 {
	var area float64
	for i := 0; i < len(poly); i++ {
		p1, p2 := poly[i], poly[(i+1)%len(poly)]
		area += p1.X*p2.Y - p2.X*p1.Y
	}
	return area / 2.0
}

// maxX returns the maximum X coordinate value among the points in the given polygon.
func maxPointsX(poly []Point) float64 {
	max := poly[0].X
	for _, p := range poly {
		if p.X > max {
			max = p.X
		}
	}
	return max
}

// distanceSq calculates the squared Euclidean distance between two points p1 and p2.
func distanceSq(p1, p2 Point) float64 {
	dx := p1.X - p2.X
	dy := p1.Y - p2.Y
	return dx*dx + dy*dy
}

// pointInPolygon determines if a point is inside a polygon using the ray-casting algorithm.
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

// isVisible determines if a direct, unobstructed line of sight exists between points p1 and p2, considering obstacles in hole and outer.
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

// segmentsIntersect checks if line segments intersect, implementing an AABB fast rejection.
func segmentsIntersect(p1, q1, p2, q2 Point) bool {
	// AABB Fast Rejection: scarta O(1) prima del calcolo dell'orientamento
	var minX1, maxX1, minY1, maxY1 float64
	if p1.X < q1.X {
		minX1, maxX1 = p1.X, q1.X
	} else {
		minX1, maxX1 = q1.X, p1.X
	}
	var minX2, maxX2 float64
	if p2.X < q2.X {
		minX2, maxX2 = p2.X, q2.X
	} else {
		minX2, maxX2 = q2.X, p2.X
	}
	if maxX1 < minX2 || minX1 > maxX2 {
		return false
	}

	if p1.Y < q1.Y {
		minY1, maxY1 = p1.Y, q1.Y
	} else {
		minY1, maxY1 = q1.Y, p1.Y
	}
	var minY2, maxY2 float64
	if p2.Y < q2.Y {
		minY2, maxY2 = p2.Y, q2.Y
	} else {
		minY2, maxY2 = q2.Y, p2.Y
	}
	if maxY1 < minY2 || minY1 > maxY2 {
		return false
	}

	// Exact orientation check
	o1 := orientation(p1, q1, p2)
	o2 := orientation(p1, q1, q2)
	o3 := orientation(p2, q2, p1)
	o4 := orientation(p2, q2, q1)
	return o1 != o2 && o3 != o4
}

// orientation determines the orientation of the triplet (p, q, r).
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

// pointInTriangleExact determines if a point lies exactly within or on the edges of a triangle defined by three vertices.
func pointInTriangleExact(p, a, b, c Point) bool {
	cp1 := (b.X-a.X)*(p.Y-a.Y) - (b.Y-a.Y)*(p.X-a.X)
	cp2 := (c.X-b.X)*(p.Y-b.Y) - (c.Y-b.Y)*(p.X-b.X)
	cp3 := (a.X-c.X)*(p.Y-c.Y) - (a.Y-c.Y)*(p.X-c.X)

	return (cp1 >= 0 && cp2 >= 0 && cp3 >= 0) || (cp1 <= 0 && cp2 <= 0 && cp3 <= 0)
}

// pointInTriangle determines if a point p lies within the triangle defined by vertices a, b, and c.
// Uses cross-product method to verify point-side relationships and includes an epsilon tolerance for precision.
func pointInTriangle(p, a, b, c Point) bool {
	cp1 := (b.X-a.X)*(p.Y-a.Y) - (b.Y-a.Y)*(p.X-a.X)
	cp2 := (c.X-b.X)*(p.Y-b.Y) - (c.Y-b.Y)*(p.X-b.X)
	cp3 := (a.X-c.X)*(p.Y-c.Y) - (a.Y-c.Y)*(p.X-c.X)

	const eps = 0.5
	return (cp1 >= -eps && cp2 >= -eps && cp3 >= -eps) || (cp1 <= eps && cp2 <= eps && cp3 <= eps)
}

/*
// isDegenerate checks if three points form a degenerate triangle.
func isDegenerate(a, b, c Point) bool {
	if a.X == b.X && a.Y == b.Y {
		return true
	}
	if b.X == c.X && b.Y == c.Y {
		return true
	}
	if c.X == a.X && c.Y == a.Y {
		return true
	}
	return ((b.X-a.X)*(c.Y-b.Y) - (b.Y-a.Y)*(c.X-b.X)) == 0
}
*/
