package wad

import (
	"fmt"
	"math"
	"sort"
	"strconv"

	"github.com/markel1974/godoom/engine/generators/wad/lumps"
	"github.com/markel1974/godoom/engine/model"
)

// Builder is responsible for constructing and managing game levels, textures, and BSP trees from WAD data.
type Builder struct {
	w        *WAD
	textures map[string]bool
}

// NewBuilder creates and returns a new instance of Builder with initialized textures mapping.
func NewBuilder() *Builder {
	return &Builder{textures: make(map[string]bool)}
}

// Setup initializes the Builder with the specified WAD file and level number, returning a ConfigRoot or an error.
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
	sectors := b.scanSubSectors(level)

	p1 := level.Things[0]

	for _, t := range level.Things {
		if t.Type == 1 {
			p1 = t
			break
		}
	}

	_, p1Sector, _ := bsp.FindSector(p1.X, p1.Y)
	p1Pos := model.XY{X: float64(p1.X) / ScaleFactor, Y: float64(-p1.Y) / ScaleFactor}
	p1Angle := float64(p1.Angle)

	player := model.NewConfigPlayer(p1Pos, p1Angle, strconv.Itoa(int(p1Sector)))
	root := model.NewConfigRoot(sectors, player, nil, ScaleFactor, true)

	return root, nil
}

// scanSubSectors generates and returns a slice of ConfigSector objects by analyzing subsectors and applying transformations.
func (b *Builder) scanSubSectors(level *Level) []*model.ConfigSector {
	// 1. Definiamo il perimetro globale del livello (Coordinate Doom)
	minX, minY, maxX, maxY := 32768.0, 32768.0, -32768.0, -32768.0
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

	margin := 256.0
	rootBBox := []model.XY{
		{X: minX - margin, Y: minY - margin},
		{X: maxX + margin, Y: minY - margin},
		{X: maxX + margin, Y: maxY + margin},
		{X: minX - margin, Y: maxY + margin},
	}

	// 2. Traversa tutto l'albero BSP per generare i poligoni perfetti
	subsectorPolys := make(map[uint16][]model.XY)
	if len(level.Nodes) > 0 {
		b.traverseBSP(level, uint16(len(level.Nodes)-1), rootBBox, subsectorPolys)
	}

	// 3. Convertiamo nel sistema di coordinate del motore (Y invertita e Scalata)
	for ssIdx, poly := range subsectorPolys {
		var scaled []model.XY
		for _, p := range poly {
			scaled = append(scaled, model.XY{X: p.X / ScaleFactor, Y: -p.Y / ScaleFactor})
		}
		subsectorPolys[ssIdx] = scaled
	}

	// 4. Eliminazione T-Junctions: Spezziamo i lati grandi in modo che combacino tutti 1:1
	b.eliminateTJunctions(level, subsectorPolys)

	// 5. Creazione ConfigSectors
	numSS := len(level.SubSectors)
	miSectors := make([]*model.ConfigSector, numSS)
	for i := 0; i < numSS; i++ {
		//ss := b.level.SubSectors[i]
		sectorRef, _ := level.GetSectorFromSubSector(uint16(i))
		ds := level.Sectors[sectorRef]

		miSector := &model.ConfigSector{
			Id:       strconv.Itoa(i),
			Floor:    float64(ds.FloorHeight) / ScaleFactor,
			Ceil:     float64(ds.CeilingHeight) / ScaleFactor,
			Textures: true,
			Tag:      strconv.Itoa(int(sectorRef)),
		}

		poly := subsectorPolys[uint16(i)]
		for j := 0; j < len(poly); j++ {
			p1 := poly[j]
			p2 := poly[(j+1)%len(poly)]
			seg := model.NewConfigSegment(miSector.Id, model.DefinitionUnknown, p1, p2)
			miSector.Segments = append(miSector.Segments, seg)
		}
		miSectors[i] = miSector
	}

	// 6. Applicazione Texture (da WAD) e Identificazione Topologica
	b.applyWadAndLinks(level, miSectors)

	for _, sector := range miSectors {
		b.forceWindingOrder(sector.Segments, false)
	}

	return miSectors
}

// traverseBSP traverses a BSP tree, splits input polygons, and associates them with subsectors in the output map.
func (b *Builder) traverseBSP(level *Level, nodeIdx uint16, poly []model.XY, out map[uint16][]model.XY) {
	if nodeIdx&0x8000 != 0 {
		ssIdx := nodeIdx &^ 0x8000
		out[ssIdx] = poly
		return
	}

	node := level.Nodes[nodeIdx]
	nx, ny := float64(node.X), float64(node.Y)
	ndx, ndy := float64(node.DX), float64(node.DY)

	front, back := b.splitPolygon(poly, nx, ny, ndx, ndy)
	if len(front) > 0 {
		b.traverseBSP(level, node.Child[0], front, out)
	}
	if len(back) > 0 {
		b.traverseBSP(level, node.Child[1], back, out)
	}
}

// splitPolygon splits a polygon into two parts based on a partition line described by its normal and point.
// poly is the input polygon defined as a slice of points (model.XY).
// nx, ny specify the base point of the partition line.
// ndx, ndy define the direction vector perpendicular to the partition line.
// Returns two slices of points (front and back) representing the two resulting polygons.
func (b *Builder) splitPolygon(poly []model.XY, nx, ny, ndx, ndy float64) (front, back []model.XY) {
	if len(poly) < 3 {
		return nil, nil
	}

	isFront := make([]bool, len(poly))
	for i, p := range poly {
		// In Doom il lato "front" della partizione è definito da val <= 0
		val := ndx*(p.Y-ny) - ndy*(p.X-nx)
		isFront[i] = val <= 1e-5
	}

	for i := 0; i < len(poly); i++ {
		p1 := poly[i]
		p2 := poly[(i+1)%len(poly)]
		f1 := isFront[i]
		f2 := isFront[(i+1)%len(poly)]

		if f1 {
			front = append(front, p1)
		} else {
			back = append(back, p1)
		}

		// Generazione del vertice sul taglio (intersezione)
		if f1 != f2 {
			dx, dy := p2.X-p1.X, p2.Y-p1.Y
			den := ndy*dx - ndx*dy
			if math.Abs(den) > 1e-10 {
				u := (ndx*(p1.Y-ny) - ndy*(p1.X-nx)) / den
				inter := model.XY{X: p1.X + u*dx, Y: p1.Y + u*dy}
				front = append(front, inter)
				back = append(back, inter)
			}
		}
	}
	return b.cleanPoly(front), b.cleanPoly(back)
}

// cleanPoly removes duplicate or close points from the polygon and ensures it has at least 3 vertices for validity.
func (b *Builder) cleanPoly(poly []model.XY) []model.XY {
	if len(poly) < 3 {
		return nil
	}
	var res []model.XY
	for _, p := range poly {
		if len(res) == 0 || b.dist(res[len(res)-1], p) > 0.01 {
			res = append(res, p)
		}
	}
	if len(res) > 1 && b.dist(res[0], res[len(res)-1]) <= 0.01 {
		res = res[:len(res)-1]
	}
	if len(res) < 3 {
		return nil
	}
	return res
}

// eliminateTJunctions resolves T-junctions in a set of subsector polygons by splitting edges into smaller segments at overlap points.
func (b *Builder) eliminateTJunctions(level *Level, subsectorPolys map[uint16][]model.XY) {
	var allVerts []model.XY
	for _, poly := range subsectorPolys {
		allVerts = append(allVerts, poly...)
	}

	for ssIdx, poly := range subsectorPolys {
		var newPoly []model.XY
		for i := 0; i < len(poly); i++ {
			p1 := poly[i]
			p2 := poly[(i+1)%len(poly)]

			var splits []model.XY
			for _, v := range allVerts {
				// Se il vertice giace strettamente DENTRO il segmento p1-p2
				if b.distPointToSegment(v, p1, p2) < 0.01 {
					dot := (v.X-p1.X)*(p2.X-p1.X) + (v.Y-p1.Y)*(p2.Y-p1.Y)
					lenSq := (p2.X-p1.X)*(p2.X-p1.X) + (p2.Y-p1.Y)*(p2.Y-p1.Y)
					if dot > 0.05 && dot < lenSq-0.05 {
						splits = append(splits, v)
					}
				}
			}

			sort.Slice(splits, func(i, j int) bool {
				return b.dist(p1, splits[i]) < b.dist(p1, splits[j])
			})

			newPoly = append(newPoly, p1)
			for _, sp := range splits {
				if b.dist(newPoly[len(newPoly)-1], sp) > 0.01 {
					newPoly = append(newPoly, sp)
				}
			}
		}
		subsectorPolys[ssIdx] = newPoly
	}
}

/*

func (b *Builder) eliminateTJunctions(level *Level, subsectorPolys map[uint16][]model.XY) {
	var allVerts []model.XY

	// 1. Raccogliamo i vertici generati dai tagli del BSP
	for _, poly := range subsectorPolys {
		allVerts = append(allVerts, poly...)
	}

	// 2. FIX FONDAMENTALE: Aggiungiamo i vertici fisici del livello
	// Questo costringerà i lati lunghi del BSP a "spezzarsi" esattamente
	// alle estremità dei segmenti fisici di Doom (WAD Segs)
	for _, v := range level.Vertexes {
		allVerts = append(allVerts, model.XY{
			X: float64(v.XCoord) / ScaleFactor,
			Y: float64(-v.YCoord) / ScaleFactor,
		})
	}

	for ssIdx, poly := range subsectorPolys {
		var newPoly []model.XY
		for i := 0; i < len(poly); i++ {
			p1 := poly[i]
			p2 := poly[(i+1)%len(poly)]

			var splits []model.XY
			for _, v := range allVerts {
				// Se il vertice giace strettamente DENTRO il segmento p1-p2
				if b.distPointToSegment(v, p1, p2) < 0.01 {
					dot := (v.X-p1.X)*(p2.X-p1.X) + (v.Y-p1.Y)*(p2.Y-p1.Y)
					lenSq := (p2.X-p1.X)*(p2.X-p1.X) + (p2.Y-p1.Y)*(p2.Y-p1.Y)

					// Ridotto il margine a 0.01 per catturare anche micro-muri
					if dot > 0.01 && dot < lenSq-0.01 {
						splits = append(splits, v)
					}
				}
			}

			sort.Slice(splits, func(i, j int) bool {
				return b.dist(p1, splits[i]) < b.dist(p1, splits[j])
			})

			newPoly = append(newPoly, p1)
			for _, sp := range splits {
				// Filtro per evitare duplicati causati da float vicinissimi
				if b.dist(newPoly[len(newPoly)-1], sp) > 0.005 {
					newPoly = append(newPoly, sp)
				}
			}
		}
		subsectorPolys[ssIdx] = newPoly
	}
}

*/

// applyWadAndLinks processes map sectors, assigning textures, tags, and neighbor relationships based on WAD data and BSP output.
func (b *Builder) applyWadAndLinks(level *Level, miSectors []*model.ConfigSector) {
	for i, miSector := range miSectors {
		if miSector == nil {
			continue
		}
		ss := level.SubSectors[i]

		for _, seg := range miSector.Segments {
			mid := model.XY{X: (seg.Start.X + seg.End.X) / 2.0, Y: (seg.Start.Y + seg.End.Y) / 2.0}

			// Trova se questo lato generato dal BSP è un vero muro del WAD
			wadSeg := b.findOverlappingWadSeg(level, mid, ss)

			// 1. Identifica il vicino (se c'è un match inverso geometrico esatto)
			foundNeighbor := false
			for j, otherSector := range miSectors {
				if i == j || otherSector == nil {
					continue
				}
				for _, otherSeg := range otherSector.Segments {
					if b.dist(seg.Start, otherSeg.End) < 0.05 && b.dist(seg.End, otherSeg.Start) < 0.05 {
						seg.Neighbor = otherSector.Id
						foundNeighbor = true
						break
					}
				}
				if foundNeighbor {
					break
				}
			}

			// 2. Classificazione basata su World.go e WAD
			if wadSeg != nil {
				line := level.LineDefs[wadSeg.LineDef]
				_, side := level.SegmentSideDef(wadSeg, line)

				if side != nil {
					seg.Upper = side.UpperTexture
					seg.Middle = side.MiddleTexture
					seg.Lower = side.LowerTexture
				}
				seg.Tag = strconv.Itoa(int(line.Flags))

				if line.Flags&0x0004 == 0 {
					// È una linea opaca a singolo lato
					seg.Kind = model.DefinitionWall // = 2
					//seg.Neighbor = "wall"
				} else if foundNeighbor {
					// È una Two-Sided che connette ad un altro settore
					seg.Kind = model.DefinitionJoin // = 3
				} else {
					// Edge case: Two-Sided rivolto verso il vuoto esterno
					seg.Kind = model.DefinitionWall
					//seg.Neighbor = "wall"
				}
			} else {
				// Il lato NON è nel WAD. È stato creato dal BSP per chiudere lo spazio.
				if foundNeighbor {
					// Portale implicito interno
					seg.Kind = model.DefinitionJoin // = 3
					seg.Tag = "bsp_split"
				} else {
					// Il lato tocca l'esterno della mappa (il BBox virtuale).
					// È un OPEN LOOP come richiesto in world.go!
					seg.Kind = model.DefinitionUnknown // = 0 (Open)
					//seg.Neighbor = "unknown"
					seg.Tag = "open"
				}
			}
		}
	}
}

// dist calculates the Euclidean distance between two points p1 and p2 in 2D space.
func (b *Builder) dist(p1, p2 model.XY) float64 {
	return math.Hypot(p2.X-p1.X, p2.Y-p1.Y)
}

// distPointToSegment calculates the shortest distance from a point to a line segment in 2D space.
func (b *Builder) distPointToSegment(p, v, w model.XY) float64 {
	l2 := b.dist(v, w) * b.dist(v, w)
	if l2 == 0 {
		return b.dist(p, v)
	}
	t := ((p.X-v.X)*(w.X-v.X) + (p.Y-v.Y)*(w.Y-v.Y)) / l2
	t = math.Max(0, math.Min(1, t))
	proj := model.XY{X: v.X + t*(w.X-v.X), Y: v.Y + t*(w.Y-v.Y)}
	return b.dist(p, proj)
}

// findOverlappingWadSeg searches for a WAD segment in the given subsector whose centerline is within a close distance to the specified point.
func (b *Builder) findOverlappingWadSeg(level *Level, mid model.XY, ss *lumps.SubSector) *lumps.Seg {
	for i := int16(0); i < ss.NumSegments; i++ {
		wadSeg := level.Segments[ss.StartSeg+i]
		v1 := level.Vertexes[wadSeg.VertexStart]
		v2 := level.Vertexes[wadSeg.VertexEnd]

		w1 := model.XY{X: float64(v1.XCoord) / ScaleFactor, Y: float64(-v1.YCoord) / ScaleFactor}
		w2 := model.XY{X: float64(v2.XCoord) / ScaleFactor, Y: float64(-v2.YCoord) / ScaleFactor}

		// Poiché i segmenti sono stati accorciati dalla risoluzione dei T-Junction,
		// controlliamo semplicemente la distanza dal loro centro alla retta originale del WAD.
		if b.distPointToSegment(mid, w1, w2) < 0.1 {
			return wadSeg
		}
	}
	return nil
}

// forceWindingOrder modifies the orientation of a set of line segments to enforce a desired winding order.
// The desired winding order is specified by the wantClockwise parameter (true for clockwise, false for counter-clockwise).
func (b *Builder) forceWindingOrder(segments []*model.ConfigSegment, wantClockwise bool) {
	if len(segments) < 3 {
		return
	}

	// 1. Calcolo dell'area con segno (Shoelace Formula)
	area := 0.0
	for _, seg := range segments {
		area += (seg.End.X - seg.Start.X) * (seg.End.Y + seg.Start.Y)
	}

	// area > 0 indica senso Orario (Clockwise)
	// area < 0 indica senso Antiorario (Counter-Clockwise)
	isClockwise := area > 0

	// 2. Se l'orientamento è già quello desiderato, non facciamo nulla
	if isClockwise == wantClockwise {
		return
	}

	// 3. Inversione dell'ordine: scambiamo le posizioni nell'array e invertiamo Start/End
	for i, j := 0, len(segments)-1; i < j; i, j = i+1, j-1 {
		segments[i], segments[j] = segments[j], segments[i]
	}

	// Invertiamo anche il verso dei singoli vettori per mantenere l'anello chiuso
	for _, seg := range segments {
		seg.Start, seg.End = seg.End, seg.Start
	}
}
