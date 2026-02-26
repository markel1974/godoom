package wad

import (
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"

	"github.com/markel1974/godoom/engine/generators/wad/lumps"
	"github.com/markel1974/godoom/engine/model"
)

// Builder is responsible for constructing and managing game levels and textures using WAD data structures.
type Builder struct {
	w        *WAD
	textures map[string]bool
	level    *Level
	bsp      *BSP
}

// NewBuilder creates and returns a new instance of Builder1 with initialized textures map.
func NewBuilder() *Builder {
	return &Builder{textures: make(map[string]bool)}
}

// Setup initializes the Builder by loading a WAD file and setting up the level configuration based on the provided level number.
func (b *Builder) Setup(wadFile string, levelNumber int) (*model.ConfigRoot, error) {
	b.w = New()
	if err := b.w.Load(wadFile); err != nil {
		return nil, err
	}
	levelNames := b.w.GetLevels()
	if levelNumber-1 >= len(levelNames) {
		return nil, fmt.Errorf("level index out of bounds: %d", levelNumber)
	}

	var err error
	b.level, err = b.w.GetLevel(levelNames[levelNumber-1])
	if err != nil {
		fmt.Printf("error: %s\n", err)
		os.Exit(1)
	}

	b.bsp = NewBsp(b.level)
	sectors := b.scanSubSectors()

	for _, sector := range sectors {
		b.forceWindingOrder(sector.Segments, false)
	}

	p1 := b.level.Things[1]
	position := model.XY{X: float64(p1.X) / ScaleFactor, Y: float64(-p1.Y) / ScaleFactor}
	_, playerSSId, _ := b.bsp.FindSector(p1.X, p1.Y)

	player := model.NewConfigPlayer(position, float64(p1.Angle), strconv.Itoa(int(playerSSId)))
	root := model.NewConfigRoot(sectors, player, nil, ScaleFactor, true)

	return root, nil
}

// scanSubSectors generates and returns a list of ConfigSector objects by processing BSP data and subsector polygons.
func (b *Builder) scanSubSectors() []*model.ConfigSector {
	// 1. Definiamo il perimetro globale del livello (Coordinate Doom)
	minX, minY, maxX, maxY := 32768.0, 32768.0, -32768.0, -32768.0
	for _, v := range b.level.Vertexes {
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
	if len(b.level.Nodes) > 0 {
		b.traverseBSP(uint16(len(b.level.Nodes)-1), rootBBox, subsectorPolys)
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
	b.eliminateTJunctions(subsectorPolys)

	// 5. Creazione ConfigSectors
	numSS := len(b.level.SubSectors)
	miSectors := make([]*model.ConfigSector, numSS)
	for i := 0; i < numSS; i++ {
		//ss := b.level.SubSectors[i]
		sectorRef, _ := b.level.GetSectorFromSubSector(uint16(i))
		ds := b.level.Sectors[sectorRef]

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
	b.applyWadAndLinks(miSectors)

	return miSectors
}

// traverseBSP recursively traverses the BSP tree, partitioning polygons and grouping by subsector indices.
func (b *Builder) traverseBSP(nodeIdx uint16, poly []model.XY, out map[uint16][]model.XY) {
	if nodeIdx&0x8000 != 0 {
		ssIdx := nodeIdx &^ 0x8000
		out[ssIdx] = poly
		return
	}

	node := b.level.Nodes[nodeIdx]
	nx, ny := float64(node.X), float64(node.Y)
	ndx, ndy := float64(node.DX), float64(node.DY)

	front, back := b.splitPolygon(poly, nx, ny, ndx, ndy)
	if len(front) > 0 {
		b.traverseBSP(node.Child[0], front, out)
	}
	if len(back) > 0 {
		b.traverseBSP(node.Child[1], back, out)
	}
}

// splitPolygon splits a convex polygon into two sub-polygons based on a splitting line defined by nx, ny, ndx, and ndy.
// It returns the points of the front and back sub-polygons after the split. Polygons with fewer than 3 vertices return nil.
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

// cleanPoly simplifies a polygon by removing consecutive points closer than a threshold and ensuring it remains closed.
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

// eliminateTJunctions identifies and resolves T-junctions in the subsector polygons by splitting edges at intersection points.
func (b *Builder) eliminateTJunctions(subsectorPolys map[uint16][]model.XY) {
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

// applyWadAndLinks associates map sectors with BSP sectors, resolving neighbors, textures, and classifications.
func (b *Builder) applyWadAndLinks(miSectors []*model.ConfigSector) {
	for i, miSector := range miSectors {
		if miSector == nil {
			continue
		}
		ss := b.level.SubSectors[i]

		for _, seg := range miSector.Segments {
			mid := model.XY{X: (seg.Start.X + seg.End.X) / 2.0, Y: (seg.Start.Y + seg.End.Y) / 2.0}

			// Trova se questo lato generato dal BSP è un vero muro del WAD
			wadSeg := b.findOverlappingWadSeg(mid, ss)

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
				line := b.level.LineDefs[wadSeg.LineDef]
				_, side := b.level.SegmentSideDef(wadSeg, line)

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

// distPointToSegment calculates the shortest distance between a point p and a line segment defined by endpoints v and w.
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

// findOverlappingWadSeg identifies and returns the WAD segment overlapping the given midpoint within the specified subsector.
func (b *Builder) findOverlappingWadSeg(mid model.XY, ss *lumps.SubSector) *lumps.Seg {
	for i := int16(0); i < ss.NumSegments; i++ {
		wadSeg := b.level.Segments[ss.StartSeg+i]
		v1 := b.level.Vertexes[wadSeg.VertexStart]
		v2 := b.level.Vertexes[wadSeg.VertexEnd]

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
