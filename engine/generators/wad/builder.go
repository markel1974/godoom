package wad

import (
	"fmt"
	"math"
	"sort"
	"strconv"

	"github.com/markel1974/godoom/engine/generators/wad/lumps"
	"github.com/markel1974/godoom/engine/model"
)

// ScaleFactor defines a constant multiplier used to scale dimensions within the configuration system.
const ScaleFactor = 25.0

// ScaleFactorCeilFloor is a constant used to scale ceiling and floor height calculations.
const ScaleFactorCeilFloor = 4.0

type Point struct {
	X                 float64
	Y                 float64
	OverlappedSegment *lumps.Seg
	Sector            *lumps.Sector
	SectorRef         uint16
}

func (p Point) ToModelXY() model.XY {
	return model.XY{X: p.X, Y: p.Y}
}

type Points []Point

// Polygon represents a collection of points defining a shape in 2D space.
type Polygon struct {
	Id        string
	Sector    *lumps.Sector
	SectorRef uint16
	Points    Points
}

type Polygons []Polygon

// EdgeKey is used for O(1) topological neighbor lookups (Half-Edge dictionary)
type EdgeKey struct {
	X1, Y1, X2, Y2 float64
}

// Builder is responsible for constructing configuration data from a WAD file.
type Builder struct {
	w        *WAD
	textures map[string]bool
}

// NewBuilder creates and returns a new Builder instance.
func NewBuilder() *Builder {
	return &Builder{textures: make(map[string]bool)}
}

// Setup initializes and configures the game level.
func (b *Builder) Setup(wadFile string, levelNumber int) (*model.ConfigRoot, error) {
	b.w = New()
	if err := b.w.Load(wadFile); err != nil {
		return nil, err
	}
	levelNames := b.w.GetLevels()
	if levelNumber-1 >= len(levelNames) {
		return nil, fmt.Errorf("level index out of bounds: %d", levelNumber)
	}

	level, err := b.w.GetLevel(levelNames[levelNumber-1])
	if err != nil {
		return nil, err
	}

	bsp := NewBsp(level)
	sectors := b.scanSubSectors(level, bsp)

	p1 := level.Things[0]
	for _, t := range level.Things {
		if t.Type == 1 {
			p1 = t
			break
		}
	}

	_, p1Sector, _ := bsp.FindSector(p1.X, p1.Y, bsp.root)
	p1Pos := model.XY{X: float64(p1.X), Y: float64(-p1.Y)}
	p1Angle := float64(p1.Angle)

	player := model.NewConfigPlayer(p1Pos, p1Angle, strconv.Itoa(int(p1Sector)))
	root := model.NewConfigRoot(sectors, player, nil, ScaleFactor, true)

	return root, nil
}

// scanSubSectors processes the BSP tree to create configuration sectors.
func (b *Builder) scanSubSectors(level *Level, bsp *BSP) []*model.ConfigSector {
	const doomMax = 32768.0
	const doomMargin = 256.0
	minX, minY, maxX, maxY := doomMax, doomMax, -doomMax, -doomMax
	for _, v := range level.Vertexes {
		if float64(v.XCoord) < minX {
			minX = float64(v.XCoord)
		}
		if float64(v.XCoord) > maxX {
			maxX = float64(v.XCoord)
		}
		if float64(v.YCoord) < minY {
			minY = float64(v.YCoord)
		}
		if float64(v.YCoord) > maxY {
			maxY = float64(v.YCoord)
		}
	}

	rootBBox := Points{
		{X: minX - doomMargin, Y: minY - doomMargin},
		{X: maxX + doomMargin, Y: minY - doomMargin},
		{X: maxX + doomMargin, Y: maxY + doomMargin},
		{X: minX - doomMargin, Y: maxY + doomMargin},
	}

	numSS := uint16(len(level.SubSectors))
	levelVerts := make(Points, len(level.Vertexes))

	for i, v := range level.Vertexes {
		levelVerts[i] = Point{X: float64(v.XCoord), Y: float64(v.YCoord)}
	}

	// 1. Traverse BSP (Spazio Nativo)
	traversedPoints := make(map[uint16]Points)
	if len(level.Nodes) > 0 {
		bsp.Traverse(level, uint16(len(level.Nodes)-1), rootBBox, traversedPoints)
	}

	traversedPolys := make(map[uint16]Polygon)

	// 2. Ancoraggio Metadati di Superficie (Pre-Split)
	for i := uint16(0); i < numSS; i++ {
		sectorRef, _ := level.GetSectorFromSubSector(i)
		ds := level.Sectors[sectorRef]
		points := traversedPoints[i]

		for j1 := 0; j1 < len(points); j1++ {
			p1 := points[j1]
			p1.OverlappedSegment = nil
			p1.Sector = ds
			p1.SectorRef = sectorRef
			points[j1] = p1
		}
		traversedPolys[i] = Polygon{Id: strconv.Itoa(int(i)), Sector: ds, SectorRef: sectorRef, Points: points}
	}

	// 3. Vertex Snapping Topologico (Elimina il drift FP64)
	PolygonsSnap(levelVerts, traversedPolys)

	var allVerts Points
	for _, poly := range traversedPolys {
		allVerts = append(allVerts, poly.Points...)
	}
	allVerts = append(allVerts, levelVerts...)

	// 4. T-Junction elimination (Crea i sottomultipli esatti per le colonne)
	b.eliminateTJunctions(allVerts, traversedPolys)

	// 5. Ancoraggio Metadati GLOBALE Sector-Aware (Post-Split)
	for i := uint16(0); i < numSS; i++ {
		poly := traversedPolys[i]
		for j1 := 0; j1 < len(poly.Points); j1++ {
			p1 := poly.Points[j1]
			p2 := poly.Points[(j1+1)%len(poly.Points)]

			// Ricerca globale filtrata per appartenenza al settore corrente
			p1.OverlappedSegment = b.findGlobalWadSeg(level, p1.ToModelXY(), p2.ToModelXY(), poly.SectorRef)
			poly.Points[j1] = p1
		}
		traversedPolys[i] = poly
	}

	b.runDiagnostics(level, traversedPolys, numSS)

	// 6. Generazione Half-Edge Map per adiacenze O(1)
	edgeToSector := make(map[EdgeKey]string)
	for i := uint16(0); i < numSS; i++ {
		poly := traversedPolys[i]
		sectorId := strconv.Itoa(int(i))
		for j := 0; j < len(poly.Points); j++ {
			p1 := poly.Points[j]
			p2 := poly.Points[(j+1)%len(poly.Points)]
			edgeToSector[EdgeKey{p1.X, p1.Y, p2.X, p2.Y}] = sectorId
		}
	}

	// 7. Export verso le strutture dell'Engine
	miSectors := b.buildEngineSectors(level, traversedPolys, numSS, edgeToSector)

	// 8. ALTERAZIONE FINALE: Trasformazione coordinate
	for _, sector := range miSectors {
		if sector == nil {
			continue
		}
		for _, seg := range sector.Segments {
			seg.Start.Y = -seg.Start.Y
			seg.End.Y = -seg.End.Y
		}
		b.forceWindingOrder(sector.Segments, false)
	}

	return miSectors
}

// eliminateTJunctions refines subsector polygons by splitting edges to eliminate T-junctions.
func (b *Builder) eliminateTJunctions(allVerts Points, subsectorPolys map[uint16]Polygon) {
	for ssIdx, poly := range subsectorPolys {
		var newPoly Polygon
		newPoly.Id = poly.Id
		newPoly.SectorRef = poly.SectorRef
		newPoly.Sector = poly.Sector

		for i1 := 0; i1 < len(poly.Points); i1++ {
			i2 := (i1 + 1) % len(poly.Points)
			start := poly.Points[i1]
			end := poly.Points[i2]
			var splits Points
			dx := end.X - start.X
			dy := end.Y - start.Y
			if lenSq := (dx * dx) + (dy * dy); lenSq > 0 {
				for _, v := range allVerts {
					// Tolleranza estesa a 1.5 per allinearsi al Vertex Snapping
					if b.distPointToSegment(v, start, end) <= 1.5 {
						t := ((v.X-start.X)*dx + (v.Y-start.Y)*dy) / lenSq
						if t > 0.001 && t < 0.999 {
							out := Point{
								X:                 v.X,
								Y:                 v.Y,
								OverlappedSegment: nil, // Risolto al mapping
								Sector:            start.Sector,
								SectorRef:         start.SectorRef,
							}
							splits = append(splits, out)
						}
					}
				}
			}
			sort.Slice(splits, func(i, j int) bool {
				dxi, dyi := splits[i].X-start.X, splits[i].Y-start.Y
				dxj, dyj := splits[j].X-start.X, splits[j].Y-start.Y
				return (dxi*dxi + dyi*dyi) < (dxj*dxj + dyj*dyj)
			})

			newPoly.Points = append(newPoly.Points, start)
			for _, sp := range splits {
				last := newPoly.Points[len(newPoly.Points)-1]
				dxSp, dySp := sp.X-last.X, sp.Y-last.Y
				if (dxSp*dxSp + dySp*dySp) > 0.000001 {
					newPoly.Points = append(newPoly.Points, sp)
				}
			}
		}
		subsectorPolys[ssIdx] = newPoly
	}
}

// buildEngineSectors risolve metadati e adiacenze operando sui poligoni nativi.
func (b *Builder) buildEngineSectors(level *Level, traversedPolys map[uint16]Polygon, numSS uint16, edgeMap map[EdgeKey]string) []*model.ConfigSector {
	miSectors := make([]*model.ConfigSector, numSS)

	for i := uint16(0); i < numSS; i++ {
		poly := traversedPolys[i]
		sectorId := strconv.Itoa(int(i))

		miSector := &model.ConfigSector{
			Id:           sectorId,
			Floor:        SnapFloat(float64(poly.Sector.FloorHeight) / ScaleFactorCeilFloor),
			Ceil:         SnapFloat(float64(poly.Sector.CeilingHeight) / ScaleFactorCeilFloor),
			Tag:          strconv.Itoa(int(poly.SectorRef)),
			TextureUpper: "wall2.ppm", TextureWall: "wall.ppm", TextureLower: "floor2.ppm",
			TextureCeil: "ceil.ppm", TextureFloor: "floor.ppm", TextureScaleFactor: 10.0,
			Textures: true,
		}

		for j := 0; j < len(poly.Points); j++ {
			p1 := poly.Points[j]
			p2 := poly.Points[(j+1)%len(poly.Points)]

			seg := model.NewConfigSegment(miSector.Id, model.DefinitionUnknown, p1.ToModelXY(), p2.ToModelXY())
			wadSeg := p1.OverlappedSegment

			// O(1) Topological Lookup (vettore inverso P2 -> P1)
			reverseKey := EdgeKey{p2.X, p2.Y, p1.X, p1.Y}
			if neighborId, exists := edgeMap[reverseKey]; exists {
				seg.Neighbor = neighborId
			}

			seg.Kind = model.DefinitionWall

			if wadSeg != nil {
				line := level.LineDefs[wadSeg.LineDef]
				_, side := level.SegmentSideDef(wadSeg, line)
				if side != nil {
					seg.Upper, seg.Middle, seg.Lower = side.UpperTexture, side.MiddleTexture, side.LowerTexture
				}
				seg.Tag = strconv.Itoa(int(line.Flags))
				if (line.Flags & 0x0004) == 0 {
					seg.Kind = model.DefinitionWall
				} else if seg.Neighbor != "" {
					seg.Kind = model.DefinitionJoin
				}
			} else {
				if seg.Neighbor != "" {
					seg.Kind = model.DefinitionJoin
					seg.Tag = "bsp_split"
				} else {
					seg.Kind = model.DefinitionUnknown
					seg.Tag = "open"
				}
			}

			miSector.Segments = append(miSector.Segments, seg)
		}
		miSectors[i] = miSector
	}

	return miSectors
}

// distPointToSegment calculates the shortest distance from a point to a line segment.
func (b *Builder) distPointToSegment(p Point, v Point, w Point) float64 {
	dx, dy := w.X-v.X, w.Y-v.Y
	l2 := dx*dx + dy*dy
	if l2 == 0 {
		pdx, pdy := p.X-v.X, p.Y-v.Y
		return math.Sqrt(pdx*pdx + pdy*pdy)
	}
	t := math.Max(0, math.Min(1, ((p.X-v.X)*dx+(p.Y-v.Y)*dy)/l2))
	projX := v.X + t*dx
	projY := v.Y + t*dy
	pdx, pdy := p.X-projX, p.Y-projY
	return math.Sqrt(pdx*pdx + pdy*pdy)
}

// findGlobalWadSeg esegue una ricerca globale limitando i match ai Seg pertinenti al Sector corrente.
func (b *Builder) findGlobalWadSeg(level *Level, p1 model.XY, p2 model.XY, targetSectorRef uint16) *lumps.Seg {
	const epsilonSq = 1.0

	var bestMatch *lumps.Seg
	var maxOverlap float64

	for i := 0; i < len(level.Segments); i++ {
		wadSeg := level.Segments[i]
		v1 := level.Vertexes[wadSeg.VertexStart]
		v2 := level.Vertexes[wadSeg.VertexEnd]

		w1 := model.XY{X: float64(v1.XCoord), Y: float64(v1.YCoord)}
		w2 := model.XY{X: float64(v2.XCoord), Y: float64(v2.YCoord)}

		dx := w2.X - w1.X
		dy := w2.Y - w1.Y
		lenSq := dx*dx + dy*dy

		if lenSq == 0 {
			continue
		}

		// 1. Check Collinearità
		cross1 := dx*(p1.Y-w1.Y) - dy*(p1.X-w1.X)
		if (cross1*cross1)/lenSq > epsilonSq {
			continue
		}

		cross2 := dx*(p2.Y-w1.Y) - dy*(p2.X-w1.X)
		if (cross2*cross2)/lenSq > epsilonSq {
			continue
		}

		// 2. Proiezione parametrica
		t1 := ((p1.X-w1.X)*dx + (p1.Y-w1.Y)*dy) / lenSq
		t2 := ((p2.X-w1.X)*dx + (p2.Y-w1.Y)*dy) / lenSq

		if t1 > t2 {
			t1, t2 = t2, t1
		}

		// 3. Verifica Sovrapposizione Assoluta
		overlapStart := math.Max(0.0, t1)
		overlapEnd := math.Min(1.0, t2)
		overlapLength := (overlapEnd - overlapStart) * math.Sqrt(lenSq)

		// Cattura l'intersezione fisica reale
		if overlapLength > 0.5 {
			line := level.LineDefs[wadSeg.LineDef]
			_, side := level.SegmentSideDef(wadSeg, line)

			// Controlla che il Seg trovato faccia parte ESATTAMENTE del settore che stiamo analizzando
			if side != nil && side.SectorRef == targetSectorRef {
				return wadSeg // Match geometrico E semantico assoluto
			}

			// Salva per fallback se il Nodebuilder ha fatto un pasticcio semantico
			if overlapLength > maxOverlap {
				maxOverlap = overlapLength
				bestMatch = wadSeg
			}
		}
	}
	return bestMatch // Ritorna il miglior match geometrico come fallback
}

// runDiagnostics calcola il delta tra i Seg nativi allocati nel SubSector e quelli catturati geometricamente.
func (b *Builder) runDiagnostics(level *Level, traversedPolys map[uint16]Polygon, numSS uint16) {
	missingTotal := 0
	alienTotal := 0

	for i := uint16(0); i < numSS; i++ {
		ss := level.SubSectors[i]

		// 1. Estrazione Segments attesi (Verità nativa del WAD)
		expectedSegs := make(map[int]bool)
		for j := int16(0); j < ss.NumSegments; j++ {
			expectedSegs[int(ss.StartSeg+j)] = true
		}

		// 2. Estrazione Segments catturati dalla nostra pipeline spaziale
		capturedSegs := make(map[int]bool)
		poly := traversedPolys[i]
		for _, p := range poly.Points {
			if p.OverlappedSegment != nil {
				// Risoluzione O(N) del puntatore all'indice reale nel WAD
				for k := 0; k < len(level.Segments); k++ {
					if level.Segments[k] == p.OverlappedSegment {
						capturedSegs[k] = true
						break
					}
				}
			}
		}

		// 3. Calcolo dei Delta
		var missing []int
		for segIdx := range expectedSegs {
			if !capturedSegs[segIdx] {
				missing = append(missing, segIdx)
			}
		}

		var alien []int
		for segIdx := range capturedSegs {
			if !expectedSegs[segIdx] {
				alien = append(alien, segIdx)
			}
		}

		// 4. Dump dell'anomalia
		if len(missing) > 0 || len(alien) > 0 {
			fmt.Printf("[DIAGNOSTIC] SubSector %d (SectorRef %d):\n", i, poly.SectorRef)
			if len(missing) > 0 {
				missingTotal += len(missing)
				fmt.Printf("  -> MANCANTI (%d): %v\n", len(missing), missing)
				for _, m := range missing {
					s := level.Segments[m]
					v1 := level.Vertexes[s.VertexStart]
					v2 := level.Vertexes[s.VertexEnd]
					fmt.Printf("     Seg %d: (%.0f, %.0f) -> (%.0f, %.0f) LineDef: %d\n",
						m, float64(v1.XCoord), float64(v1.YCoord), float64(v2.XCoord), float64(v2.YCoord), s.LineDef)
				}
			}
			if len(alien) > 0 {
				alienTotal += len(alien)
				fmt.Printf("  -> ALIENI (%d): %v (Catturati ma non assegnati dal Nodebuilder a questo SS)\n", len(alien), alien)
			}
		}
	}

	fmt.Printf("\n[DIAGNOSTIC SUMMARY] Totale Seg Nativi Persi: %d | Totale Seg Alieni Catturati: %d\n", missingTotal, alienTotal)

	//os.Exit(1)
}

// forceWindingOrder adjusts the winding order of given segments.
func (b *Builder) forceWindingOrder(segments []*model.ConfigSegment, wantClockwise bool) {
	if len(segments) < 3 {
		return
	}
	area := 0.0
	for _, seg := range segments {
		area += (seg.End.X - seg.Start.X) * (seg.End.Y + seg.Start.Y)
	}
	if (area > 0) != wantClockwise {
		for i, j := 0, len(segments)-1; i < j; i, j = i+1, j-1 {
			segments[i], segments[j] = segments[j], segments[i]
		}
		for _, seg := range segments {
			seg.Start, seg.End = seg.End, seg.Start
		}
	}
}

// SnapFloat rounds a floating-point number.
func SnapFloat(val float64) float64 {
	return math.Round(val*10000.0) / 10000.0
}

// PolygonSplit splits a polygon into two sub-polygons based on a partition line.
func PolygonSplit(poly Points, nx int16, ny int16, ndx int16, ndy int16) (Points, Points) {
	if len(poly) < 3 {
		return nil, nil
	}

	const epsilonSq = 0.01

	fnx, fny := float64(nx), float64(ny)
	fndx, fndy := float64(ndx), float64(ndy)
	lenSq := fndx*fndx + fndy*fndy

	if lenSq == 0 {
		return poly, nil
	}

	type vertexInfo struct {
		p    Point
		dist float64
		side int
	}

	vertices := make([]vertexInfo, 0, len(poly))
	frontCount, backCount := 0, 0

	for _, p := range poly {
		cross := fndx*(p.Y-fny) - fndy*(p.X-fnx)
		distSq := (cross * cross) / lenSq
		side := 0

		if distSq > epsilonSq {
			if cross <= 0 {
				side = 1
				frontCount++
			} else {
				side = -1
				backCount++
			}
		}
		vertices = append(vertices, vertexInfo{p, cross, side})
	}

	if backCount == 0 {
		return PolygonClean(poly), nil
	}
	if frontCount == 0 {
		return nil, PolygonClean(poly)
	}

	var front, back Points

	for i := 0; i < len(vertices); i++ {
		v1 := vertices[i]
		v2 := vertices[(i+1)%len(vertices)]

		if v1.side >= 0 {
			front = append(front, v1.p)
		}
		if v1.side <= 0 {
			back = append(back, v1.p)
		}

		if (v1.side > 0 && v2.side < 0) || (v1.side < 0 && v2.side > 0) {
			u := v1.dist / (v1.dist - v2.dist)
			inter := Point{
				X:                 v1.p.X + u*(v2.p.X-v1.p.X),
				Y:                 v1.p.Y + u*(v2.p.Y-v1.p.Y),
				OverlappedSegment: nil,
				Sector:            v1.p.Sector,
				SectorRef:         v1.p.SectorRef,
			}
			front = append(front, inter)
			back = append(back, inter)
		}
	}

	return PolygonClean(front), PolygonClean(back)
}

// PolygonClean removes redundant points.
func PolygonClean(poly Points) Points {
	if len(poly) < 3 {
		return nil
	}
	var res Points
	tolSq := 0.1 * 0.1

	for _, p := range poly {
		if len(res) == 0 {
			res = append(res, p)
			continue
		}
		last := res[len(res)-1]
		dx, dy := p.X-last.X, p.Y-last.Y
		if (dx*dx + dy*dy) > tolSq {
			res = append(res, p)
		}
	}

	if len(res) > 1 {
		first, last := res[0], res[len(res)-1]
		dx, dy := last.X-first.X, last.Y-first.Y
		if (dx*dx + dy*dy) <= tolSq {
			res = res[:len(res)-1]
		}
	}

	if len(res) < 3 {
		return nil
	}
	return res
}

// PolygonsSnap collassa i vertici approssimati sulle coordinate native esatte prendendo il più vicino.
func PolygonsSnap(vertexes Points, subsectorPolys map[uint16]Polygon) {
	snapVertex := func(p Point) Point {
		best := p
		minDistSq := 2.25 // Distanza max 1.5 unità

		for _, wv := range vertexes {
			dx, dy := p.X-wv.X, p.Y-wv.Y
			distSq := dx*dx + dy*dy
			if distSq <= minDistSq {
				minDistSq = distSq
				// Conserva i metadati di superficie
				best = Point{
					X:                 wv.X,
					Y:                 wv.Y,
					OverlappedSegment: p.OverlappedSegment,
					Sector:            p.Sector,
					SectorRef:         p.SectorRef,
				}
			}
		}
		return best
	}

	for i, poly := range subsectorPolys {
		for j, p := range poly.Points {
			poly.Points[j] = snapVertex(p)
		}
		points := PolygonClean(poly.Points)

		z := subsectorPolys[i]
		z.Points = points
		subsectorPolys[i] = z
	}
}
