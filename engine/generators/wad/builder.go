package wad

import (
	"fmt"
	"math"
	"sort"
	"strconv"

	"github.com/markel1974/godoom/engine/generators/wad/lumps"
	"github.com/markel1974/godoom/engine/model"
)

// ScaleFactor defines the scaling ratio applied to level geometry during processing, adjusting dimensions for consistency.
const ScaleFactor = 25.0

// ScaleFactorCeilFloor represents the scalar value used for normalizing floor and ceiling height calculations.
const ScaleFactorCeilFloor = 4.0

// tolerance defines the allowable margin of error for geometric calculations, typically used for comparing distances.
const tolerance = 0.1

// Polygons represents a slice of 2D points, each defined by X and Y coordinates, used for modeling geometric shapes.
type Polygons []model.XY

// EuclideanDistance computes the Euclidean distance between two points in 2D space.
func EuclideanDistance(p1 model.XY, p2 model.XY) float64 {
	return math.Hypot(p2.X-p1.X, p2.Y-p1.Y)
}

// SnapFloat rounds a floating-point value to four decimal places for precision control in floating-point arithmetic.
func SnapFloat(val float64) float64 {
	return math.Round(val*10000.0) / 10000.0
}

// PolygonSplit splits a polygon into two parts using a partition line defined by its normal and delta values.
// The front polygon includes vertices on or in front of the line, and the back polygon includes vertices behind it.
// Returns the cleaned front and back polygons, where cleaning removes redundant points based on a tolerance threshold.
func PolygonSplit(poly Polygons, nx int16, ny int16, ndx int16, ndy int16) (Polygons, Polygons) {
	var front Polygons
	var back Polygons

	if len(poly) < 3 {
		return nil, nil
	}

	fnx, fny := float64(nx), float64(ny)
	fndx, fndy := float64(ndx), float64(ndy)

	isFront := make([]bool, len(poly))
	for i, p := range poly {
		// In Doom il lato "front" della partizione è definito da val <= 0
		val := fndx*(p.Y-fny) - fndy*(p.X-fnx)
		// Margine per la stabilità in virgola mobile sulle coordinate native
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

		if f1 != f2 {
			dx, dy := p2.X-p1.X, p2.Y-p1.Y
			den := fndy*dx - fndx*dy
			if math.Abs(den) > 1e-10 {
				u := (fndx*(p1.Y-fny) - fndy*(p1.X-fnx)) / den
				interX := SnapFloat(p1.X + u*dx)
				interY := SnapFloat(p1.Y + u*dy)
				inter := model.XY{X: interX, Y: interY}
				//inter := model.XY{X: p1.X + u*dx, Y: p1.Y + u*dy}
				front = append(front, inter)
				back = append(back, inter)
			}
		}
	}
	return PolygonClean(front), PolygonClean(back)
}

// PolygonClean removes duplicate points from the input polygon and eliminates redundant vertices based on a tolerance threshold.
func PolygonClean(poly Polygons) Polygons {
	if len(poly) < 3 {
		return nil
	}
	var res Polygons
	for _, p := range poly {
		// La tolleranza 0.01 è perfetta se lavori in scala originale Doom [-32768, 32768]
		if len(res) == 0 || EuclideanDistance(res[len(res)-1], p) > tolerance {
			res = append(res, p)
		}
	}
	if len(res) > 1 && EuclideanDistance(res[0], res[len(res)-1]) <= tolerance {
		res = res[:len(res)-1]
	}
	if len(res) < 3 {
		return nil
	}
	return res
}

// Builder structures and initializes resources for WAD file processing, including levels, textures, and configuration data.
type Builder struct {
	w        *WAD
	textures map[string]bool
}

// NewBuilder creates a new Builder instance with an initialized texture map and returns a pointer to it.
func NewBuilder() *Builder {
	return &Builder{textures: make(map[string]bool)}
}

// Setup initializes the Builder with the specified WAD file and level, returning a ConfigRoot or an error if setup fails.
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

// scanSubSectors processes BSP nodes to generate configuration sectors for the game engine based on level geometry details.
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

	// 2. Traverse BSP (Spazio Nativo)
	subsectorPolys := make(map[uint16]Polygons)
	if len(level.Nodes) > 0 {
		bsp.Traverse(level, uint16(len(level.Nodes)-1), rootBBox, subsectorPolys)
	}

	// 3. T-Junction elimination (Spazio Nativo)
	b.eliminateTJunctions(level, subsectorPolys)

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

		poly := subsectorPolys[i]
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
			seg.Start.Y = SnapFloat(-seg.Start.Y)
			seg.End.Y = SnapFloat(-seg.End.Y)
		}
		b.forceWindingOrder(sector.Segments, false)
	}

	return miSectors
}

// eliminateTJunctions processes polygon data to eliminate T-junctions by splitting edges at intersecting vertices.
func (b *Builder) eliminateTJunctions(level *Level, subsectorPolys map[uint16]Polygons) {
	var allVerts Polygons
	for _, poly := range subsectorPolys {
		allVerts = append(allVerts, poly...)
	}
	for _, v := range level.Vertexes {
		allVerts = append(allVerts, model.XY{X: SnapFloat(float64(v.XCoord)), Y: SnapFloat(float64(v.YCoord))})
	}

	for ssIdx, poly := range subsectorPolys {
		var newPoly Polygons
		for i := 0; i < len(poly); i++ {
			p1, p2 := poly[i], poly[(i+1)%len(poly)]
			var splits Polygons
			dx, dy := p2.X-p1.X, p2.Y-p1.Y
			lenSq := dx*dx + dy*dy

			if lenSq > 0 {
				for _, v := range allVerts {
					if b.distPointToSegment(v, p1, p2) < tolerance {
						t := ((v.X-p1.X)*dx + (v.Y-p1.Y)*dy) / lenSq
						if t > 0.001 && t < 0.999 {
							splits = append(splits, v)
						}
					}
				}
			}
			sort.Slice(splits, func(i, j int) bool { return EuclideanDistance(p1, splits[i]) < EuclideanDistance(p1, splits[j]) })
			newPoly = append(newPoly, p1)
			for _, sp := range splits {
				if EuclideanDistance(newPoly[len(newPoly)-1], sp) > 0.001 {
					newPoly = append(newPoly, sp)
				}
			}
		}
		subsectorPolys[ssIdx] = newPoly
	}
}

// applyWadAndLinks processes segments in the given level and links them with WAD data and nearby sectors.
func (b *Builder) applyWadAndLinks(level *Level, miSectors []*model.ConfigSector) {
	for i, miSector := range miSectors {
		if miSector == nil {
			continue
		}
		ss := level.SubSectors[i]
		for _, seg := range miSector.Segments {
			mid := model.XY{X: SnapFloat((seg.Start.X + seg.End.X) / 2.0), Y: SnapFloat((seg.Start.Y + seg.End.Y) / 2.0)}
			wadSeg := b.findOverlappingWadSeg(level, mid, ss)

			foundNeighbor := false
			for j, otherSector := range miSectors {
				if i == j || otherSector == nil {
					continue
				}
				for _, otherSeg := range otherSector.Segments {
					if EuclideanDistance(seg.Start, otherSeg.End) < tolerance && EuclideanDistance(seg.End, otherSeg.Start) < tolerance ||
						EuclideanDistance(seg.Start, otherSeg.Start) < tolerance && EuclideanDistance(seg.End, otherSeg.End) < tolerance {
						seg.Neighbor = otherSector.Id
						foundNeighbor = true
						break
					}
				}
				if foundNeighbor {
					break
				}
			}

			if wadSeg != nil {
				line := level.LineDefs[wadSeg.LineDef]
				_, side := level.SegmentSideDef(wadSeg, line)
				if side != nil {
					seg.Upper, seg.Middle, seg.Lower = side.UpperTexture, side.MiddleTexture, side.LowerTexture
				}
				seg.Tag = strconv.Itoa(int(line.Flags))
				if line.Flags&0x0004 == 0 {
					seg.Kind = model.DefinitionWall
				} else if foundNeighbor {
					seg.Kind = model.DefinitionJoin
				} else {
					seg.Kind = model.DefinitionWall
				}
			} else {
				if foundNeighbor {
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

// distPointToSegment computes the shortest distance from a point `p` to a line segment defined by points `v` and `w`.
func (b *Builder) distPointToSegment(p, v, w model.XY) float64 {
	l2 := EuclideanDistance(v, w) * EuclideanDistance(v, w)
	if l2 == 0 {
		return EuclideanDistance(p, v)
	}
	t := math.Max(0, math.Min(1, ((p.X-v.X)*(w.X-v.X)+(p.Y-v.Y)*(w.Y-v.Y))/l2))
	return EuclideanDistance(p, model.XY{X: v.X + t*(w.X-v.X), Y: v.Y + t*(w.Y-v.Y)})
}

// findOverlappingWadSeg searches for a segment in the given SubSector that overlaps with the specified midpoint within a tolerance.
func (b *Builder) findOverlappingWadSeg(level *Level, mid model.XY, ss *lumps.SubSector) *lumps.Seg {
	for i := int16(0); i < ss.NumSegments; i++ {
		wadSeg := level.Segments[ss.StartSeg+i]
		v1, v2 := level.Vertexes[wadSeg.VertexStart], level.Vertexes[wadSeg.VertexEnd]
		w1 := model.XY{X: float64(v1.XCoord), Y: float64(v1.YCoord)}
		w2 := model.XY{X: float64(v2.XCoord), Y: float64(v2.YCoord)}
		// Usiamo 1.0 come tolleranza sicura per lo spazio nativo Doom
		if b.distPointToSegment(mid, w1, w2) < 1.0 {
			return level.Segments[ss.StartSeg+i]
		}
	}
	return nil
}

// forceWindingOrder ensures that the order of the given segments matches the specified winding order (clockwise or counter-clockwise).
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
