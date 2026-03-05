package wad

import (
	"fmt"
	"math"
	"sort"
	"strconv"

	"github.com/markel1974/godoom/engine/model"
)

type Fixed int64

func ToFixed(f float64) Fixed    { return Fixed(f * 65536) }
func (f Fixed) ToFloat() float64 { return float64(f) / 65536.0 }

const ScaleFactorLineDef = 25.0
const ScaleFactorCeilFloorLineDef = 4.0

type PointFixed struct {
	X Fixed
	Y Fixed
}

func (p PointFixed) ToModelXY() model.XY { return model.XY{X: p.X.ToFloat(), Y: p.Y.ToFloat()} }

type Edge struct {
	V1, V2 uint16
	LDIdx  int
	IsLeft bool
}

type EdgeKeyFixed struct {
	X1, Y1, X2, Y2 Fixed
}

type PolygonDef struct {
	Outer []PointFixed
	Holes [][]PointFixed
}

type BuilderLineDef struct {
	w *WAD
}

func NewBuilderLineDef() *BuilderLineDef {
	return &BuilderLineDef{}
}

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
	//var things []model.ConfigThing

	for _, t := range level.Things {
		if t.Type == 1 || t.Type == 2 || t.Type == 3 || t.Type == 4 || t.Type == 11 {
			if t.Type == 1 {
				pX, pY, pAngle = float64(t.X), float64(t.Y), float64(t.Angle)
			}
			continue
		}

		//tX, tY := float64(t.X), float64(t.Y)
		//tSectorId := bld.resolveSectorId(PointFixed{ToFixed(tX), ToFixed(tY)}, sectors)
		//cfgThing := model.NewConfigThing(
		//	fmt.Sprintf("t_%d", i),
		//	model.XY{X: tX, Y: -tY},
		//	float64(t.Angle)*(math.Pi/180.0), // In caso l'engine richieda radianti per le Things
		//	int(t.Type),
		//	tSectorId,
		//)
		//things = append(things, cfgThing)
	}

	playerSectorId := bld.resolveSectorId(PointFixed{ToFixed(pX), ToFixed(pY)}, sectors)
	player := model.NewConfigPlayer(
		model.XY{X: pX, Y: -pY},
		pAngle,
		playerSectorId,
	)

	return model.NewConfigRoot(sectors, player, nil, ScaleFactorLineDef, true), nil
}

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

	var allConfigSectors []*model.ConfigSector

	// Struttura potenziata per tracciare se il segmento è un VERO muro o una diagonale interna
	type exactSeg struct {
		SectorID  string
		Seg       *model.ConfigSegment
		P1, P2    PointFixed
		IsWadLine bool
	}
	var allExactSegs []*exactSeg

	for secIdx, edges := range sectorToEdges {
		wadSector := level.Sectors[secIdx]
		polygonDefs := bld.traceLoops(level, edges)

		for loopIdx, def := range polygonDefs {
			mergedPoly := bld.mergeHoles(def)
			triangles := bld.triangulate(mergedPoly)

			for triIdx, tri := range triangles {
				// PROTEZIONE 1: Scartiamo i triangoli degeneri (area nulla) che bucano il rasterizer
				if bld.isDegenerate(tri[0], tri[1], tri[2]) {
					continue
				}

				sectorId := fmt.Sprintf("s%d_l%d_t%d", secIdx, loopIdx, triIdx)
				miSector := &model.ConfigSector{
					Id:           sectorId,
					Floor:        float64(wadSector.FloorHeight) / ScaleFactorCeilFloorLineDef,
					Ceil:         float64(wadSector.CeilingHeight) / ScaleFactorCeilFloorLineDef,
					Tag:          strconv.Itoa(int(secIdx)),
					Textures:     true,
					TextureUpper: "wall2.ppm", TextureWall: "wall.ppm", TextureLower: "floor2.ppm",
					TextureCeil: "ceil.ppm", TextureFloor: "floor.ppm", TextureScaleFactor: 10.0,
				}

				for k := 0; k < 3; k++ {
					p1, p2 := tri[k], tri[(k+1)%3]

					seg := model.NewConfigSegment(sectorId, model.DefinitionWall, p1.ToModelXY(), p2.ToModelXY())
					isWadLine := bld.mapSegmentMetadata(seg, p1, p2, edges, level)

					seg.Start.Y, seg.End.Y = -seg.Start.Y, -seg.End.Y
					miSector.Segments = append(miSector.Segments, seg)

					allExactSegs = append(allExactSegs, &exactSeg{
						SectorID:  sectorId,
						Seg:       seg,
						P1:        p1,
						P2:        p2,
						IsWadLine: isWadLine,
					})
				}
				allConfigSectors = append(allConfigSectors, miSector)
			}
		}
	}

	edgeMap := make(map[EdgeKeyFixed]string)
	for _, es := range allExactSegs {
		k := EdgeKeyFixed{es.P1.X, es.P1.Y, es.P2.X, es.P2.Y}
		edgeMap[k] = es.SectorID
	}

	for _, es := range allExactSegs {
		if es.Seg.Kind == model.DefinitionJoin {
			reverseKey := EdgeKeyFixed{es.P2.X, es.P2.Y, es.P1.X, es.P1.Y}
			if neighborId, exists := edgeMap[reverseKey]; exists {
				es.Seg.Neighbor = neighborId
			} else {
				// PROTEZIONE 2: Previene i Muri Fantasma!
				if es.IsWadLine {
					es.Seg.Kind = model.DefinitionWall // Muro originale con settore adiacente mancante (errore di mappa)
				} else {
					es.Seg.Neighbor = es.SectorID // Diagonale interna orfana: diventa un auto-portale invisibile!
				}
			}
		}
	}

	return allConfigSectors
}

func (bld *BuilderLineDef) triangulate(poly []PointFixed) [][]PointFixed {
	var triangles [][]PointFixed
	working := make([]PointFixed, len(poly))
	copy(working, poly)

	// I poligoni nativi di Doom devono essere trattati in senso Orario (CW)
	if bld.getWinding(working) < 0 {
		for i, j := 0, len(working)-1; i < j; i, j = i+1, j-1 {
			working[i], working[j] = working[j], working[i]
		}
	}

	for len(working) > 2 {
		earFound := false
		for i := 0; i < len(working); i++ {
			prev := working[(i+len(working)-1)%len(working)]
			curr := working[i]
			next := working[(i+1)%len(working)]

			if bld.isEar(prev, curr, next, working) {
				triangles = append(triangles, []PointFixed{prev, curr, next})
				working = append(working[:i], working[i+1:]...)
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
				abx, aby := float64(curr.X-prev.X), float64(curr.Y-prev.Y)
				cbx, cby := float64(next.X-curr.X), float64(next.Y-curr.Y)
				cp := abx*cby - aby*cbx
				if cp < minCp {
					minCp = cp
					bestIdx = i
				}
			}
			prev := working[(bestIdx+len(working)-1)%len(working)]
			curr := working[bestIdx]
			next := working[(bestIdx+1)%len(working)]
			triangles = append(triangles, []PointFixed{prev, curr, next})
			working = append(working[:bestIdx], working[bestIdx+1:]...)
		}
	}
	return triangles
}

func (bld *BuilderLineDef) isEar(a, b, c PointFixed, poly []PointFixed) bool {
	if a.X == c.X && a.Y == c.Y {
		return true
	}

	abx, aby := float64(b.X-a.X), float64(b.Y-a.Y)
	cbx, cby := float64(c.X-b.X), float64(c.Y-b.Y)

	cp := abx*cby - aby*cbx

	if cp >= 0 {
		return false
	}

	for _, p := range poly {
		if p.X == a.X && p.Y == a.Y {
			continue
		}
		if p.X == b.X && p.Y == b.Y {
			continue
		}
		if p.X == c.X && p.Y == c.Y {
			continue
		}

		if bld.pointInTriangleExact(p, a, b, c) {
			return false
		}
	}
	return true
}

func (bld *BuilderLineDef) isDegenerate(a, b, c PointFixed) bool {
	if a.X == b.X && a.Y == b.Y {
		return true
	}
	if b.X == c.X && b.Y == c.Y {
		return true
	}
	if c.X == a.X && c.Y == a.Y {
		return true
	}

	abx, aby := float64(b.X-a.X), float64(b.Y-a.Y)
	cbx, cby := float64(c.X-b.X), float64(c.Y-b.Y)
	return (abx*cby - aby*cbx) == 0
}

func (bld *BuilderLineDef) getWinding(poly []PointFixed) int64 {
	var area float64
	for i := 0; i < len(poly); i++ {
		p1, p2 := poly[i], poly[(i+1)%len(poly)]
		area += float64(p2.X-p1.X) * float64(p2.Y+p1.Y)
	}
	if area > 0 {
		return 1
	}
	if area < 0 {
		return -1
	}
	return 0
}

func (bld *BuilderLineDef) mapSegmentMetadata(seg *model.ConfigSegment, p1, p2 PointFixed, sectorEdges []Edge, level *Level) bool {
	for _, e := range sectorEdges {
		v1, v2 := level.Vertexes[e.V1], level.Vertexes[e.V2]
		w1 := PointFixed{ToFixed(float64(v1.XCoord)), ToFixed(float64(v1.YCoord))}
		w2 := PointFixed{ToFixed(float64(v2.XCoord)), ToFixed(float64(v2.YCoord))}

		if (p1 == w1 && p2 == w2) || (p1 == w2 && p2 == w1) {
			ld := level.LineDefs[e.LDIdx]
			sideIdx := ld.SideDefRight
			if e.IsLeft {
				sideIdx = ld.SideDefLeft
			}

			side := level.SideDefs[sideIdx]
			seg.Middle, seg.Upper, seg.Lower = side.MiddleTexture, side.UpperTexture, side.LowerTexture
			seg.Kind = model.DefinitionWall
			if ld.Flags&(1<<2) != 0 {
				seg.Kind = model.DefinitionJoin
			}
			return true
		}
	}
	seg.Kind = model.DefinitionJoin
	return false
}

func (bld *BuilderLineDef) resolveSectorId(p PointFixed, sectors []*model.ConfigSector) string {
	var closestSector string
	var minDist = math.MaxFloat64

	px, py := p.X.ToFloat(), p.Y.ToFloat()

	for _, s := range sectors {
		if len(s.Segments) != 3 {
			continue
		}

		v1X := s.Segments[0].Start.X
		v1Y := -s.Segments[0].Start.Y
		v2X := s.Segments[1].Start.X
		v2Y := -s.Segments[1].Start.Y
		v3X := s.Segments[2].Start.X
		v3Y := -s.Segments[2].Start.Y

		v1 := PointFixed{ToFixed(v1X), ToFixed(v1Y)}
		v2 := PointFixed{ToFixed(v2X), ToFixed(v2Y)}
		v3 := PointFixed{ToFixed(v3X), ToFixed(v3Y)}

		if bld.pointInTriangle(p, v1, v2, v3) {
			return s.Id
		}

		cx := (v1X + v2X + v3X) / 3.0
		cy := (v1Y + v2Y + v3Y) / 3.0
		distSq := (cx-px)*(cx-px) + (cy-py)*(cy-py)

		if distSq < minDist {
			minDist = distSq
			closestSector = s.Id
		}
	}

	if closestSector != "" {
		return closestSector
	}
	return "0"
}

func (bld *BuilderLineDef) pointInTriangleExact(p, a, b, c PointFixed) bool {
	abx, aby := float64(b.X-a.X), float64(b.Y-a.Y)
	bcx, bcy := float64(c.X-b.X), float64(c.Y-b.Y)
	cax, cay := float64(a.X-c.X), float64(a.Y-c.Y)

	pax, pay := float64(p.X-a.X), float64(p.Y-a.Y)
	pbx, pby := float64(p.X-b.X), float64(p.Y-b.Y)
	pcx, pcy := float64(p.X-c.X), float64(p.Y-c.Y)

	cp1 := abx*pay - aby*pax
	cp2 := bcx*pby - bcy*pbx
	cp3 := cax*pcy - cay*pcx

	return (cp1 >= 0 && cp2 >= 0 && cp3 >= 0) || (cp1 <= 0 && cp2 <= 0 && cp3 <= 0)
}

func (bld *BuilderLineDef) pointInTriangle(p, a, b, c PointFixed) bool {
	abx, aby := float64(b.X-a.X), float64(b.Y-a.Y)
	bcx, bcy := float64(c.X-b.X), float64(c.Y-b.Y)
	cax, cay := float64(a.X-c.X), float64(a.Y-c.Y)

	pax, pay := float64(p.X-a.X), float64(p.Y-a.Y)
	pbx, pby := float64(p.X-b.X), float64(p.Y-b.Y)
	pcx, pcy := float64(p.X-c.X), float64(p.Y-c.Y)

	cp1 := abx*pay - aby*pax
	cp2 := bcx*pby - bcy*pbx
	cp3 := cax*pcy - cay*pcx

	const eps = 0.5
	return (cp1 >= -eps && cp2 >= -eps && cp3 >= -eps) || (cp1 <= eps && cp2 <= eps && cp3 <= eps)
}

func (bld *BuilderLineDef) traceLoops(level *Level, edges []Edge) []PolygonDef {
	adj := make(map[uint16][]Edge)
	for _, e := range edges {
		adj[e.V1] = append(adj[e.V1], e)
	}

	var rawLoops [][]PointFixed
	visited := make(map[Edge]bool)

	for _, startEdge := range edges {
		if visited[startEdge] {
			continue
		}

		var currentLoop []PointFixed
		curr := startEdge
		for {
			visited[curr] = true
			v := level.Vertexes[curr.V1]
			currentLoop = append(currentLoop, PointFixed{X: ToFixed(float64(v.XCoord)), Y: ToFixed(float64(v.YCoord))})

			nextOptions := adj[curr.V2]
			var nextEdge Edge
			found := false
			for _, o := range nextOptions {
				if !visited[o] {
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

	var outers [][]PointFixed
	var holes [][]PointFixed

	maxArea := 0.0
	outerSign := 1.0

	areas := make([]float64, len(rawLoops))
	for i, loop := range rawLoops {
		areas[i] = bld.signedArea(loop)
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
			if bld.pointInPolygon(h[0], def.Outer) {
				defs[i].Holes = append(defs[i].Holes, h)
				break
			}
		}
	}

	return defs
}

func (bld *BuilderLineDef) mergeHoles(def PolygonDef) []PointFixed {
	if len(def.Holes) == 0 {
		return def.Outer
	}

	outer := make([]PointFixed, len(def.Outer))
	copy(outer, def.Outer)

	sort.Slice(def.Holes, func(i, j int) bool {
		return bld.maxX(def.Holes[i]) > bld.maxX(def.Holes[j])
	})

	for _, hole := range def.Holes {
		holeIdx := 0
		maxX := hole[0].X
		for i, p := range hole {
			if p.X > maxX {
				maxX = p.X
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

			if bld.isVisible(holePoint, op, hole, outer) {
				dist := bld.distanceSq(holePoint, op)
				if dist < minDist {
					minDist = dist
					bestOuterIdx = i
				}
			}
		}

		if bestOuterIdx == -1 {
			bestOuterIdx = 0
			for i, op := range outer {
				dist := bld.distanceSq(holePoint, op)
				if dist < minDist {
					minDist = dist
					bestOuterIdx = i
				}
			}
		}

		newPoly := make([]PointFixed, 0, len(outer)+len(hole)+2)
		newPoly = append(newPoly, outer[:bestOuterIdx+1]...)

		for i := 0; i < len(hole); i++ {
			newPoly = append(newPoly, hole[(holeIdx+i)%len(hole)])
		}
		newPoly = append(newPoly, hole[holeIdx])
		newPoly = append(newPoly, outer[bestOuterIdx])

		if bestOuterIdx+1 < len(outer) {
			newPoly = append(newPoly, outer[bestOuterIdx+1:]...)
		}
		outer = newPoly
	}

	return outer
}

func (bld *BuilderLineDef) signedArea(poly []PointFixed) float64 {
	var area float64
	for i := 0; i < len(poly); i++ {
		p1, p2 := poly[i], poly[(i+1)%len(poly)]
		area += p1.X.ToFloat()*p2.Y.ToFloat() - p2.X.ToFloat()*p1.Y.ToFloat()
	}
	return area / 2.0
}

func (bld *BuilderLineDef) maxX(poly []PointFixed) Fixed {
	max := poly[0].X
	for _, p := range poly {
		if p.X > max {
			max = p.X
		}
	}
	return max
}

func (bld *BuilderLineDef) distanceSq(p1, p2 PointFixed) float64 {
	dx := p1.X.ToFloat() - p2.X.ToFloat()
	dy := p1.Y.ToFloat() - p2.Y.ToFloat()
	return dx*dx + dy*dy
}

func (bld *BuilderLineDef) pointInPolygon(p PointFixed, poly []PointFixed) bool {
	inside := false
	px, py := p.X.ToFloat(), p.Y.ToFloat()
	for i, j := 0, len(poly)-1; i < len(poly); j, i = i, i+1 {
		xi, yi := poly[i].X.ToFloat(), poly[i].Y.ToFloat()
		xj, yj := poly[j].X.ToFloat(), poly[j].Y.ToFloat()
		if ((yi > py) != (yj > py)) && (px < (xj-xi)*(py-yi)/(yj-yi)+xi) {
			inside = !inside
		}
	}
	return inside
}

func (bld *BuilderLineDef) isVisible(p1, p2 PointFixed, hole, outer []PointFixed) bool {
	for i := 0; i < len(outer); i++ {
		e1, e2 := outer[i], outer[(i+1)%len(outer)]
		if e1 == p1 || e1 == p2 || e2 == p1 || e2 == p2 {
			continue
		}
		if bld.segmentsIntersect(p1, p2, e1, e2) {
			return false
		}
	}
	for i := 0; i < len(hole); i++ {
		e1, e2 := hole[i], hole[(i+1)%len(hole)]
		if e1 == p1 || e1 == p2 || e2 == p1 || e2 == p2 {
			continue
		}
		if bld.segmentsIntersect(p1, p2, e1, e2) {
			return false
		}
	}
	return true
}

func (bld *BuilderLineDef) segmentsIntersect(p1, q1, p2, q2 PointFixed) bool {
	o1 := bld.orientation(p1, q1, p2)
	o2 := bld.orientation(p1, q1, q2)
	o3 := bld.orientation(p2, q2, p1)
	o4 := bld.orientation(p2, q2, q1)
	if o1 != o2 && o3 != o4 {
		return true
	}
	return false
}

func (bld *BuilderLineDef) orientation(p, q, r PointFixed) int {
	val := (q.Y.ToFloat()-p.Y.ToFloat())*(r.X.ToFloat()-q.X.ToFloat()) - (q.X.ToFloat()-p.X.ToFloat())*(r.Y.ToFloat()-q.Y.ToFloat())
	if val == 0 {
		return 0
	}
	if val > 0 {
		return 1
	}
	return 2
}
