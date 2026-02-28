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

// tolerance defines the permissible margin of error for geometric calculations, such as distance comparisons or alignment.
const tolerance = 0.1

// Polygons represents a collection of XY points that define a closed or open polygon shape in 2D space.
type Polygons []model.XY

// Builder is responsible for constructing configuration data from a WAD file and its levels.
type Builder struct {
	w        *WAD
	textures map[string]bool
}

// NewBuilder creates and returns a new Builder instance with initialized textures map.
func NewBuilder() *Builder {
	return &Builder{textures: make(map[string]bool)}
}

// Setup initializes and configures the game level, including sectors, player position, and settings for the given WAD file.
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

// scanSubSectors processes the BSP tree to create configuration sectors from the level's subsectors and vertex data.
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

	rootBBox := Polygons{
		{X: minX - doomMargin, Y: minY - doomMargin},
		{X: maxX + doomMargin, Y: minY - doomMargin},
		{X: maxX + doomMargin, Y: maxY + doomMargin},
		{X: minX - doomMargin, Y: maxY + doomMargin},
	}

	levelVerts := make(Polygons, len(level.Vertexes))
	for i, v := range level.Vertexes {
		levelVerts[i] = model.XY{X: float64(v.XCoord), Y: float64(v.YCoord)}
	}

	// 2. Traverse BSP (Spazio Nativo)
	traversedPolys := make(map[uint16]Polygons)
	if len(level.Nodes) > 0 {
		bsp.Traverse(level, uint16(len(level.Nodes)-1), rootBBox, traversedPolys)
	}

	var allVerts Polygons
	for _, poly := range traversedPolys {
		allVerts = append(allVerts, poly...)
	}
	allVerts = append(allVerts, levelVerts...)

	// 3. T-Junction elimination (Spazio Nativo)
	b.eliminateTJunctions(allVerts, traversedPolys)

	// 3.5 Vertex Snapping Topologico
	PolygonsSnap(levelVerts, traversedPolys)

	// 4. ConfigSectors creation (Spazio Nativo)
	numSS := uint16(len(level.SubSectors))
	miSectors := make([]*model.ConfigSector, numSS)
	for i := uint16(0); i < numSS; i++ {
		sectorRef, _ := level.GetSectorFromSubSector(i)
		ds := level.Sectors[sectorRef]
		miSector := &model.ConfigSector{
			Id:           strconv.Itoa(int(i)),
			Floor:        SnapFloat(float64(ds.FloorHeight) / ScaleFactorCeilFloor),
			Ceil:         SnapFloat(float64(ds.CeilingHeight) / ScaleFactorCeilFloor),
			Tag:          strconv.Itoa(int(sectorRef)),
			TextureUpper: "wall2.ppm", TextureWall: "wall.ppm", TextureLower: "floor2.ppm",
			TextureCeil: "ceil.ppm", TextureFloor: "floor.ppm", TextureScaleFactor: 10.0,
			Textures: true,
		}

		poly := traversedPolys[i]
		for j := 0; j < len(poly); j++ {
			p1 := poly[j]
			p2 := poly[(j+1)%len(poly)]
			seg := model.NewConfigSegment(miSector.Id, model.DefinitionUnknown, p1, p2)
			miSector.Segments = append(miSector.Segments, seg)
		}
		miSectors[i] = miSector
	}

	// 5. Apply Textures and Links (Spazio Nativo)
	b.applyWadAndLinks(level, miSectors)

	// 6. ALTERAZIONE FINALE: Trasformazione in coordinate Engine
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

// eliminateTJunctions refines subsector polygons by splitting edges where vertices are close to eliminate T-junctions.
func (b *Builder) eliminateTJunctions(allVerts Polygons, subsectorPolys map[uint16]Polygons) {
	for ssIdx, poly := range subsectorPolys {
		var newPoly Polygons
		for i := 0; i < len(poly); i++ {
			p1, p2 := poly[i], poly[(i+1)%len(poly)]
			var splits Polygons
			dx := p2.X - p1.X
			dy := p2.Y - p1.Y
			if lenSq := (dx * dx) + (dy * dy); lenSq > 0 {
				for _, v := range allVerts {
					if b.distPointToSegment(v, p1, p2) < tolerance {
						t := ((v.X-p1.X)*dx + (v.Y-p1.Y)*dy) / lenSq
						if t > 0.001 && t < 0.999 {
							splits = append(splits, v)
						}
					}
				}
			}
			sort.Slice(splits, func(i, j int) bool {
				dxi, dyi := splits[i].X-p1.X, splits[i].Y-p1.Y
				dxj, dyj := splits[j].X-p1.X, splits[j].Y-p1.Y
				return (dxi*dxi + dyi*dyi) < (dxj*dxj + dyj*dyj)
			})

			newPoly = append(newPoly, p1)
			for _, sp := range splits {
				last := newPoly[len(newPoly)-1]
				dxSp, dySp := sp.X-last.X, sp.Y-last.Y
				if (dxSp*dxSp + dySp*dySp) > 0.000001 { // 0.001^2
					newPoly = append(newPoly, sp)
				}
			}
		}
		subsectorPolys[ssIdx] = newPoly
	}
}

// applyWadAndLinks processes subsectors by associating them with WAD data, setting neighbor links, and defining segment kinds.
func (b *Builder) applyWadAndLinks(level *Level, miSectors []*model.ConfigSector) {
	for idx, miSector := range miSectors {
		if miSector == nil {
			continue
		}
		//ss := level.SubSectors[idx]
		for _, seg := range miSector.Segments {
			wadSeg := b.findOverlappingWadSegFromSeg(level, seg.Start, seg.End)
			seg.Neighbor = b.findNeighbors(miSectors, seg.Start, seg.End, idx)
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
				} else if len(seg.Neighbor) > 0 {
					seg.Kind = model.DefinitionJoin
				}
			} else {
				if len(seg.Neighbor) > 0 {
					seg.Kind = model.DefinitionJoin
					seg.Tag = "bsp_split"
				} else {
					seg.Kind = model.DefinitionUnknown
					seg.Tag = "open"
				}
			}
		}
	}
}

// distPointToSegment calculates the shortest distance from a point to a line segment in 2D space.
func (b *Builder) distPointToSegment(p model.XY, v model.XY, w model.XY) float64 {
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

// findNeighbors identifies and returns the ID of a neighboring sector whose segment overlaps or aligns with the specified segment.
func (b *Builder) findNeighbors(miSectors []*model.ConfigSector, p1 model.XY, p2 model.XY, segIdx int) string {
	for j, otherSector := range miSectors {
		if segIdx == j || otherSector == nil {
			continue
		}
		for _, otherSeg := range otherSector.Segments {
			if IsCollinearOverlap(p1, p2, otherSeg.Start, otherSeg.End) {
				return otherSector.Id
			}
		}
	}
	return ""
}

// findOverlappingWadSegFromSeg esegue una ricerca globale sull'intero set di segmenti del livello.
func (b *Builder) findOverlappingWadSegFromSeg(level *Level, p1 model.XY, p2 model.XY) *lumps.Seg {
	const epsilonSq = 1.0 // Tolleranza al quadrato nello spazio nativo Doom

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

		// 1. Collinearità: verifica la distanza quadratica di p1 e p2 dalla retta del wadSeg
		cross1 := dx*(p1.Y-w1.Y) - dy*(p1.X-w1.X)
		if (cross1*cross1)/lenSq > epsilonSq {
			continue
		}

		cross2 := dx*(p2.Y-w1.Y) - dy*(p2.X-w1.X)
		if (cross2*cross2)/lenSq > epsilonSq {
			continue
		}

		// 2. Intersezione parametrica: proiezione di p1 e p2 su w1-w2
		t1 := ((p1.X-w1.X)*dx + (p1.Y-w1.Y)*dy) / lenSq
		t2 := ((p2.X-w1.X)*dx + (p2.Y-w1.Y)*dy) / lenSq

		if t1 > t2 {
			t1, t2 = t2, t1
		}

		margin := 1.0 / math.Sqrt(lenSq)

		if t1 >= -margin && t2 <= 1.0+margin && (t2-t1) > 0.001 {
			return level.Segments[i]
		}
	}
	return nil
}

// forceWindingOrder adjusts the winding order of given segments to match the desired orientation (clockwise or counterclockwise).
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

// SnapFloat rounds a floating-point number to four decimal places, helping to reduce numerical inaccuracies.
func SnapFloat(val float64) float64 {
	return math.Round(val*10000.0) / 10000.0
}

// PolygonSplit splits a polygon into two sub-polygons based on a defined partition line specified by its normal and delta values.
// Takes a polygon `poly` and partition line parameters `nx`, `ny`, `ndx`, `ndy`.
// Returns two Polygons: one in front of the partition line and one behind it, or nil if no splitting occurs.
func PolygonSplit(poly Polygons, nx int16, ny int16, ndx int16, ndy int16) (Polygons, Polygons) {
	if len(poly) < 3 {
		return nil, nil
	}

	// Tolleranza quadra (es. 0.1 * 0.1) calibrata per lo spazio nativo Doom
	const epsilonSq = 0.01

	fnx, fny := float64(nx), float64(ny)
	fndx, fndy := float64(ndx), float64(ndy)
	lenSq := fndx*fndx + fndy*fndy

	if lenSq == 0 {
		return poly, nil // Fallback di sicurezza in caso di vettore partizione nullo
	}

	type vertexInfo struct {
		p    model.XY
		dist float64
		side int // 1: Front, -1: Back, 0: On
	}

	vertices := make([]vertexInfo, 0, len(poly))
	frontCount, backCount := 0, 0

	for _, p := range poly {
		// Prodotto vettoriale (distanza non scalata)
		cross := fndx*(p.Y-fny) - fndy*(p.X-fnx)

		// Verifica di appartenenza alla retta (On-plane)
		distSq := (cross * cross) / lenSq
		side := 0

		if distSq > epsilonSq {
			if cross <= 0 {
				side = 1 // Front (Convenzione BSP: lato destro <= 0)
				frontCount++
			} else {
				side = -1 // Back
				backCount++
			}
		}
		vertices = append(vertices, vertexInfo{p, cross, side})
	}

	// Fast path: nessuna intersezione necessaria se i punti giacciono tutti da una parte
	if backCount == 0 {
		return PolygonClean(poly), nil
	}
	if frontCount == 0 {
		return nil, PolygonClean(poly)
	}

	var front, back Polygons

	for i := 0; i < len(vertices); i++ {
		v1 := vertices[i]
		v2 := vertices[(i+1)%len(vertices)]

		if v1.side >= 0 {
			front = append(front, v1.p)
		}
		if v1.side <= 0 {
			back = append(back, v1.p)
		}

		// Se il segmento attraversa la partizione (front <-> back), genera l'intersezione
		if (v1.side > 0 && v2.side < 0) || (v1.side < 0 && v2.side > 0) {
			u := v1.dist / (v1.dist - v2.dist)
			// FP64 puro, senza SnapFloat
			inter := model.XY{
				X: v1.p.X + u*(v2.p.X-v1.p.X),
				Y: v1.p.Y + u*(v2.p.Y-v1.p.Y),
			}
			front = append(front, inter)
			back = append(back, inter)
		}
	}

	return PolygonClean(front), PolygonClean(back)
}

// PolygonClean removes redundant points and ensures the polygon has at least three vertices or returns nil if invalid.
func PolygonClean(poly Polygons) Polygons {
	if len(poly) < 3 {
		return nil
	}
	var res Polygons
	tolSq := tolerance * tolerance

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

// IsCollinearOverlap determines if two line segments overlap collinearly in 2D space.
// It checks for parallelism, collinearity, and overlapping projections on the shared line.
func IsCollinearOverlap(s1, e1, s2, e2 model.XY) bool {
	dx1, dy1 := e1.X-s1.X, e1.Y-s1.Y
	dx2, dy2 := e2.X-s2.X, e2.Y-s2.Y

	// 1. Parallelismo: il prodotto vettoriale delle direzioni deve essere ~0
	if math.Abs(dx1*dy2-dy1*dx2) > 0.1 {
		return false
	}

	len1Sq := dx1*dx1 + dy1*dy1
	if len1Sq == 0 {
		return false
	}

	// 2. Collinearità: la distanza al quadrato di s2 dalla retta (s1, e1) deve essere minima
	crossLine := dx1*(s2.Y-s1.Y) - dy1*(s2.X-s1.X)
	if (crossLine*crossLine)/len1Sq > tolerance {
		return false
	}

	// 3. Sovrapposizione: le proiezioni parametriche t di s2 ed e2 su (s1, e1)
	// devono intersecare l'intervallo [0, 1] del segmento originale.
	t1 := ((s2.X-s1.X)*dx1 + (s2.Y-s1.Y)*dy1) / len1Sq
	t2 := ((e2.X-s1.X)*dx1 + (e2.Y-s1.Y)*dy1) / len1Sq

	if t1 > t2 {
		t1, t2 = t2, t1
	}

	// Calcolo del segmento di intersezione tra l'intervallo proiettato [t1, t2] e [0, 1]
	overlapStart := math.Max(0.0, t1)
	overlapEnd := math.Min(1.0, t2)

	// Un overlap è topologicamente valido per un portal solo se la porzione condivisa
	// supera una soglia dimensionale tangibile (es. 0.1% della lunghezza di s1-e1).
	// Questo previene falsi collegamenti ("micro-portali") tra settori dovuti al drift FP64.
	return (overlapEnd - overlapStart) > 0.001
}

func PolygonsSnap(wadVerts Polygons, subsectorPolys map[uint16]Polygons) {
	// Funzione di snap: se un vertice FP64 dista meno di 1.5 unità Doom
	// da un vertice WAD nativo, lo collassa sulla coordinata esatta.
	snapVertex := func(p model.XY) model.XY {
		for _, wv := range wadVerts {
			dx, dy := p.X-wv.X, p.Y-wv.Y
			if (dx*dx + dy*dy) <= 2.25 { // 1.5 al quadrato
				return wv
			}
		}
		return p
	}

	// Applica lo snap a tutti i poligoni risultanti dal BSP
	for i, poly := range subsectorPolys {
		for j, p := range poly {
			poly[j] = snapVertex(p)
		}
		subsectorPolys[i] = PolygonClean(poly) // Pulisce eventuali vertici collassati
	}
}

// EuclideanDistance calculates the Euclidean distance between two points in a 2D space.
//func EuclideanDistance(p1 model.XY, p2 model.XY) float64 {
//	return math.Hypot(p2.X-p1.X, p2.Y-p1.Y)
//}
