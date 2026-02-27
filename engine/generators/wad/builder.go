package wad

import (
	"fmt"
	"math"
	"sort"
	"strconv"

	"github.com/markel1974/godoom/engine/generators/wad/lumps"
	"github.com/markel1974/godoom/engine/model"
)

// ScaleFactorFloor defines scale factor for floor heights
const ScaleFactorFloor = 2.0

// ScaleFactorCeil defines scale factor for ceiling heights
const ScaleFactorCeil = 2.0

// 0.1 è il minimo matematico per assorbire il drift CSG di bsp.Traverse
// Tolleranza a 0.1 per il matching dei portali
const tolerance = 0.1

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
	sectors := b.scanSubSectors(level, bsp)

	p1 := level.Things[0]
	for _, t := range level.Things {
		if t.Type == 1 {
			p1 = t
			break
		}
	}

	_, p1Sector, _ := bsp.FindSector(p1.X, p1.Y, bsp.root)
	p1Pos := model.XY{X: float64(p1.X), Y: float64(-p1.Y)} // Player position invertita per l'engine
	p1Angle := float64(p1.Angle)

	player := model.NewConfigPlayer(p1Pos, p1Angle, strconv.Itoa(int(p1Sector)))
	root := model.NewConfigRoot(sectors, player, nil, 8.0, true)

	return root, nil
}

// scanSubSectors generates and returns a slice of ConfigSector objects by analyzing subsectors and applying transformations.
func (b *Builder) scanSubSectors(level *Level, bsp *BSP) []*model.ConfigSector {
	// 1. Define the global perimeter of the level (Doom Coordinates)
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

	// 2. Traverse the entire BSP tree to generate perfect polygons (Spazio Nativo)
	subsectorPolys := make(map[uint16]Polygons)
	if len(level.Nodes) > 0 {
		bsp.Traverse(level, uint16(len(level.Nodes)-1), rootBBox, subsectorPolys)
	}

	// 3. T-Junction elimination: (Spazio Nativo)
	b.eliminateTJunctions(level, subsectorPolys)

	// 4. ConfigSectors creation (Spazio Nativo, nessuna alterazione Y)
	numSS := len(level.SubSectors)
	miSectors := make([]*model.ConfigSector, numSS)
	for i := 0; i < numSS; i++ {
		sectorRef, _ := level.GetSectorFromSubSector(uint16(i))
		ds := level.Sectors[sectorRef]
		floor := SnapFloat(float64(ds.FloorHeight) / ScaleFactorFloor)
		ceil := SnapFloat(float64(ds.CeilingHeight) / ScaleFactorCeil)
		miSector := &model.ConfigSector{
			Id: strconv.Itoa(i), Floor: floor, Ceil: ceil, Textures: true, Tag: strconv.Itoa(int(sectorRef)),
		}

		miSector.TextureUpper = "wall2.ppm"
		miSector.TextureWall = "wall.ppm"
		miSector.TextureLower = "floor2.ppm"
		miSector.TextureCeil = "ceil.ppm"
		miSector.TextureFloor = "floor.ppm"
		miSector.TextureScaleFactor = 10.0
		miSector.Textures = true

		poly := subsectorPolys[uint16(i)]
		for j := 0; j < len(poly); j++ {
			p1 := poly[j]
			p2 := poly[(j+1)%len(poly)]
			// Inserimento con coordinate native DOOM
			seg := model.NewConfigSegment(miSector.Id, model.DefinitionUnknown, p1, p2)
			miSector.Segments = append(miSector.Segments, seg)
		}
		miSectors[i] = miSector
	}

	// 5. Apply Textures and Topological Identification (Spazio Nativo)
	b.applyWadAndLinks(level, miSectors)

	// 6. ALTERAZIONE FINALE DEL MODELLO
	// Solo dopo aver validato il BSP applichiamo la mutazione per l'engine
	for _, sector := range miSectors {
		if sector == nil {
			continue
		}
		for _, seg := range sector.Segments {
			// Inversione Y per il motore
			seg.Start.Y = SnapFloat(-seg.Start.Y)
			seg.End.Y = SnapFloat(-seg.End.Y)
		}
		// Il Winding Order viene calcolato sulla geometria finale invertita
		b.forceWindingOrder(sector.Segments, false)
	}

	return miSectors
}

func (b *Builder) eliminateTJunctions(level *Level, subsectorPolys map[uint16]Polygons) {
	var allVerts Polygons

	for _, poly := range subsectorPolys {
		allVerts = append(allVerts, poly...)
	}

	for _, v := range level.Vertexes {
		interX := SnapFloat(float64(v.XCoord))
		// Coordinata nativa senza inversione
		interY := SnapFloat(float64(v.YCoord))
		allVerts = append(allVerts, model.XY{
			X: interX,
			Y: interY,
		})
	}

	for ssIdx, poly := range subsectorPolys {
		var newPoly Polygons
		for i := 0; i < len(poly); i++ {
			p1 := poly[i]
			p2 := poly[(i+1)%len(poly)]

			var splits Polygons
			for _, v := range allVerts {
				if b.distPointToSegment(v, p1, p2) < tolerance {
					dot := (v.X-p1.X)*(p2.X-p1.X) + (v.Y-p1.Y)*(p2.Y-p1.Y)
					lenSq := (p2.X-p1.X)*(p2.X-p1.X) + (p2.Y-p1.Y)*(p2.Y-p1.Y)

					if dot > tolerance && dot < lenSq-tolerance {
						splits = append(splits, v)
					}
				}
			}

			sort.Slice(splits, func(i, j int) bool {
				return EuclideanDistance(p1, splits[i]) < EuclideanDistance(p1, splits[j])
			})

			newPoly = append(newPoly, p1)
			for _, sp := range splits {
				if EuclideanDistance(newPoly[len(newPoly)-1], sp) > tolerance {
					newPoly = append(newPoly, sp)
				}
			}
		}
		subsectorPolys[ssIdx] = newPoly
	}
}

// applyWadAndLinks processes map sectors, assigning textures, tags, and neighbor relationships based on WAD data and BSP output.
func (b *Builder) applyWadAndLinks(level *Level, miSectors []*model.ConfigSector) {
	for i, miSector := range miSectors {
		if miSector == nil {
			continue
		}
		ss := level.SubSectors[i]

		for _, seg := range miSector.Segments {
			mid := model.XY{
				X: SnapFloat((seg.Start.X + seg.End.X) / 2.0),
				Y: SnapFloat((seg.Start.Y + seg.End.Y) / 2.0),
			}

			wadSeg := b.findOverlappingWadSeg(level, mid, ss)

			foundNeighbor := false
			for j, otherSector := range miSectors {
				if i == j || otherSector == nil {
					continue
				}
				for _, otherSeg := range otherSector.Segments {
					if EuclideanDistance(seg.Start, otherSeg.End) < tolerance && EuclideanDistance(seg.End, otherSeg.Start) < tolerance {
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
					seg.Upper = side.UpperTexture
					seg.Middle = side.MiddleTexture
					seg.Lower = side.LowerTexture
				}
				seg.Tag = strconv.Itoa(int(line.Flags))

				if line.Flags&0x0004 == 0 {
					seg.Kind = model.DefinitionWall // = 2
				} else if foundNeighbor {
					seg.Kind = model.DefinitionJoin // = 3
				} else {
					seg.Kind = model.DefinitionWall
				}
			} else {
				if foundNeighbor {
					seg.Kind = model.DefinitionJoin // = 3
					seg.Tag = "bsp_split"
				} else {
					seg.Kind = model.DefinitionUnknown // = 0 (Open)
					seg.Tag = "open"
				}
			}
		}
	}
}

// distPointToSegment calculates the shortest distance from a point to a line segment in 2D space.
func (b *Builder) distPointToSegment(p model.XY, v model.XY, w model.XY) float64 {
	l2 := EuclideanDistance(v, w) * EuclideanDistance(v, w)
	if l2 == 0 {
		return EuclideanDistance(p, v)
	}
	t := ((p.X-v.X)*(w.X-v.X) + (p.Y-v.Y)*(w.Y-v.Y)) / l2
	t = math.Max(0, math.Min(1, t))
	proj := model.XY{X: v.X + t*(w.X-v.X), Y: v.Y + t*(w.Y-v.Y)}
	return EuclideanDistance(p, proj)
}

// findOverlappingWadSeg searches for a WAD segment in the given subsector whose centerline is within a close distance to the specified point.
func (b *Builder) findOverlappingWadSeg(level *Level, mid model.XY, ss *lumps.SubSector) *lumps.Seg {
	for i := int16(0); i < ss.NumSegments; i++ {
		wadSeg := level.Segments[ss.StartSeg+i]
		v1 := level.Vertexes[wadSeg.VertexStart]
		v2 := level.Vertexes[wadSeg.VertexEnd]

		interX1 := SnapFloat(float64(v1.XCoord))
		// Coordinata nativa senza inversione
		interY1 := SnapFloat(float64(v1.YCoord))
		interX2 := SnapFloat(float64(v2.XCoord))
		interY2 := SnapFloat(float64(v2.YCoord))

		w1 := model.XY{X: interX1, Y: interY1}
		w2 := model.XY{X: interX2, Y: interY2}

		if b.distPointToSegment(mid, w1, w2) < tolerance {
			return level.Segments[ss.StartSeg+i]
		}
	}
	return nil
}

// forceWindingOrder modifies the orientation of a set of line segments to enforce a desired winding order.
func (b *Builder) forceWindingOrder(segments []*model.ConfigSegment, wantClockwise bool) {
	if len(segments) < 3 {
		return
	}

	area := 0.0
	for _, seg := range segments {
		area += (seg.End.X - seg.Start.X) * (seg.End.Y + seg.Start.Y)
	}

	isClockwise := area > 0

	if isClockwise == wantClockwise {
		return
	}

	for i, j := 0, len(segments)-1; i < j; i, j = i+1, j-1 {
		segments[i], segments[j] = segments[j], segments[i]
	}

	for _, seg := range segments {
		seg.Start, seg.End = seg.End, seg.Start
	}
}
