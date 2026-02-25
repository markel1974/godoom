package wad

import (
	"errors"
	"fmt"
	"math"
	"os"
	"strconv"

	lumps2 "github.com/markel1974/godoom/engine/generator/wad/lumps"
	"github.com/markel1974/godoom/engine/model"
)

//L'implementazione richiede la stesura di un algoritmo di clipping poligonale (Sutherland-Hodgman)
// per fendere un bounding box globale iterativamente lungo l'albero BSP.

// Builder2 is responsible for constructing and managing game level data including textures, levels, and BSP trees.
type Builder2 struct {
	w        *WAD
	textures map[string]bool
	level    *Level
	bsp      *BSP
}

// NewBuilder initializes and returns a new instance of Builder with default values.
func NewBuilder2() *Builder2 {
	return &Builder2{
		textures: make(map[string]bool),
		level:    nil,
	}
}

// Setup initializes the Builder by loading a WAD file and preparing the data for a specific level. Returns model.InputConfig or an error.
func (b *Builder2) Setup(wadFile string, levelNumber int) (*model.InputConfig, error) {
	b.w = New()
	if err := b.w.Load(wadFile); err != nil {
		return nil, err
	}
	levelNames := b.w.GetLevels()
	if len(levelNames) == 0 {
		return nil, errors.New("error: No levels found")
	}
	levelIdx := levelNumber - 1
	if levelIdx >= len(levelNames) {
		return nil, errors.New(fmt.Sprintf("error: No such level number %d", levelNumber))
	}
	levelName := levelNames[levelIdx]
	fmt.Printf("Loading level %s ...\n", levelName)

	var err error
	b.level, err = b.w.GetLevel(levelName)
	if err != nil {
		fmt.Printf("error: %s\n", err)
		os.Exit(1)
	}

	b.bsp = NewBsp(b.level)

	// 1. Decostruzione top-down degli iperpiani dei NODES
	hulls := b.ExtractSubSectorHulls()

	// 2. Costruzione della topologia InputSector
	sectors := b.buildPortalTopology(hulls)

	for textureId := range b.textures {
		if _, err = b.w.GetTextureImage(textureId); err != nil {
			fmt.Println("Texture error:", textureId, err.Error())
		}
	}

	p1 := b.level.Things[1]
	//position := model.XY{X: float64(p1.X), Y: float64(-p1.Y)}
	// CORREZIONE 1: Riscalatura posizione Player per il clipping engine
	position := model.XY{X: float64(p1.X) / ScaleFactor, Y: float64(-p1.Y) / ScaleFactor}
	_, playerSSectorId, _ := b.bsp.FindSector(p1.X, p1.Y)

	fmt.Printf("PLAYER POSITION: SubSector %d\n", playerSSectorId)

	cfg := &model.InputConfig{
		DisableLoop: true,
		ScaleFactor: ScaleFactor,
		Sectors:     sectors,
		Player: &model.InputPlayer{
			Position: position,
			Angle:    float64(p1.Angle),
			Sector:   strconv.Itoa(int(playerSSectorId)),
		},
	}

	return cfg, nil
}

// ExtractSubSectorHulls generates and returns a map of subsector IDs to their respective convex hulls as polygons.
func (b *Builder2) ExtractSubSectorHulls() map[uint16]Polygon {
	hulls := make(map[uint16]Polygon)
	if len(b.level.Nodes) == 0 {
		return hulls
	}

	rootNodeIdx := uint16(len(b.level.Nodes) - 1)
	// Bounding box globale enorme per iniziare il clipping
	const maxBBox = 65536.0
	globalPoly := Polygon{
		{X: -maxBBox, Y: -maxBBox},
		{X: maxBBox, Y: -maxBBox},
		{X: maxBBox, Y: maxBBox},
		{X: -maxBBox, Y: maxBBox},
	}

	var traverse func(nodeIdx uint16, currentPoly Polygon)
	traverse = func(nodeIdx uint16, currentPoly Polygon) {
		if nodeIdx&subSectorBit != 0 {
			ssId := nodeIdx & ^subSectorBit
			//hulls[ssId] = currentPoly
			//return

			// CORREZIONE 2: Riscalatura dei vertici della Hull generata dal BSP
			scaledPoly := make(Polygon, len(currentPoly))
			for i, pt := range currentPoly {
				scaledPoly[i] = model.XY{X: pt.X / ScaleFactor, Y: pt.Y / ScaleFactor}
			}
			hulls[ssId] = scaledPoly
			return
		}

		node := b.level.Nodes[nodeIdx]
		nx, ny := float64(node.X), float64(-node.Y)
		ndx, ndy := float64(node.DX), float64(-node.DY)

		// Child[0] è Right (Front), Child[1] è Left (Back)
		right := b.clipPolygon(currentPoly, nx, ny, ndx, ndy, true)
		left := b.clipPolygon(currentPoly, nx, ny, ndx, ndy, false)

		traverse(node.Child[0], right)
		traverse(node.Child[1], left)
	}

	traverse(rootNodeIdx, globalPoly)
	return hulls
}

// clipPolygon clips a polygon against a dividing line defined by a normal and direction vector, returning the clipped polygon.
// poly is the input polygon to clip. nx, ny define the origin of the dividing line. ndx, ndy define the line's direction.
// rightSide determines if the right side (true) or left side (false) of the line is kept in the output.
func (b *Builder2) clipPolygon(poly Polygon, nx, ny, ndx, ndy float64, rightSide bool) Polygon {
	var out Polygon
	if len(poly) == 0 {
		return out
	}
	isInside := func(pt model.XY) bool {
		cross := (pt.X-nx)*ndy - (pt.Y-ny)*ndx
		if rightSide {
			return cross <= 0
		}
		return cross >= 0
	}
	intersect := func(p1, p2 model.XY) model.XY {
		x1, y1, x2, y2 := p1.X, p1.Y, p2.X, p2.Y
		x3, y3, x4, y4 := nx, ny, nx+ndx, ny+ndy
		den := (x1-x2)*(y3-y4) - (y1-y2)*(x3-x4)
		if math.Abs(den) < 1e-7 {
			return p1
		}
		t := ((x1-x3)*(y3-y4) - (y1-y3)*(x3-x4)) / den
		return model.XY{X: x1 + t*(x2-x1), Y: y1 + t*(y2-y1)}
	}
	prevPt := poly[len(poly)-1]
	prevInside := isInside(prevPt)
	for _, currPt := range poly {
		currInside := isInside(currPt)
		if currInside {
			if !prevInside {
				out = append(out, intersect(prevPt, currPt))
			}
			out = append(out, currPt)
		} else if prevInside {
			out = append(out, intersect(prevPt, currPt))
		}
		prevPt, prevInside = currPt, currInside
	}
	return out
}

// buildPortalTopology generates a list of InputSectors based on the given hulls and the level's subsectors and sectors.
func (b *Builder2) buildPortalTopology(hulls map[uint16]Polygon) []*model.InputSector {
	var out []*model.InputSector

	for ssId, poly := range hulls {
		subSector := b.level.SubSectors[ssId]
		sectorRef, ok := b.level.GetSectorFromSubSector(ssId)
		if !ok {
			continue
		}
		doomSector := b.level.Sectors[sectorRef]

		idStr := strconv.Itoa(int(ssId))
		mSector := model.NewInputSector(idStr)
		mSector.Floor = float64(doomSector.FloorHeight) / ScaleFactor
		mSector.Ceil = float64(doomSector.CeilingHeight) / ScaleFactor
		mSector.FloorTexture = doomSector.FloorPic
		mSector.CeilTexture = doomSector.CeilingPic

		mSector.Tag = fmt.Sprintf("%f", float64(doomSector.LightLevel)/255.0)

		for i := 0; i < len(poly); i++ {
			p1, p2 := poly[i], poly[(i+1)%len(poly)]
			foundSeg := false

			// Check se esiste un SEG fisico del WAD
			for j := int16(0); j < subSector.NumSegments; j++ {
				seg := b.level.Segments[subSector.StartSeg+j]
				v1 := b.level.Vertexes[seg.VertexStart]
				v2 := b.level.Vertexes[seg.VertexEnd]

				// CORREZIONE 3: Confronto con tolleranza riscalata
				if b.isPointEqualScaled(p1, v1) && b.isPointEqualScaled(p2, v2) {
					mSeg := b.createInputSegmentFromSeg(idStr, seg)
					mSector.Segments = append(mSector.Segments, mSeg)
					foundSeg = true
					break
				}
				/*
					if b.isPointEqual(p1, v1) && b.isPointEqual(p2, v2) {
						mSeg := b.createInputSegmentFromSeg(idStr, seg)
						mSector.Segments = append(mSector.Segments, mSeg)
						foundSeg = true
						break
					}

				*/
			}

			// Se è un bordo senza SEG, è un portale invisibile generato dai NODES
			if !foundSeg {
				// Probing per i portali invisibili (già riscalato)
				mid := model.XY{X: (p1.X + p2.X) / 2, Y: (p1.Y + p2.Y) / 2}
				dx, dy := p2.X-p1.X, p2.Y-p1.Y
				mag := math.Sqrt(dx*dx + dy*dy)
				if mag > 1e-4 {
					// Probe rimpicciolito coerentemente con ScaleFactor
					probe := model.XY{X: mid.X + (dy/mag)*(0.1/ScaleFactor), Y: mid.Y - (dx/mag)*(0.1/ScaleFactor)}
					// Riportiamo in unità Doom per FindSector
					_, neighborSSId, _ := b.bsp.FindSector(int16(probe.X*ScaleFactor), int16(-probe.Y*ScaleFactor))

					if neighborSSId != ssId {
						mSeg := model.NewInputSegment(idStr, DefinitionVoid, p1, p2)
						mSeg.Neighbor = strconv.Itoa(int(neighborSSId))
						mSector.Segments = append(mSector.Segments, mSeg)
					}
				}
			}
		}
		out = append(out, mSector)
	}
	return out
}

// isPointEqual returns true if the coordinates of the model.XY point and the lumps2.Vertex point are equal within a small tolerance.
func (b *Builder2) isPointEqual(p model.XY, v *lumps2.Vertex) bool {
	return math.Abs(p.X-float64(v.XCoord)) < 1e-2 && math.Abs(p.Y-float64(-v.YCoord)) < 1e-2
}

// createInputSegmentFromSeg creates an InputSegment from a given segment and associates it with a parent ID.
// It derives the segment's type, defines textures used, and updates the Builder's texture map.
func (b *Builder2) createInputSegmentFromSeg(parentId string, seg *lumps2.Seg) *model.InputSegment {
	lineDef := b.level.LineDefs[seg.LineDef]
	v1 := b.level.Vertexes[seg.VertexStart]
	v2 := b.level.Vertexes[seg.VertexEnd]

	// CORREZIONE 4: Riscalatura dei vertici dei segmenti fisici
	s := model.XY{X: float64(v1.XCoord) / ScaleFactor, Y: float64(-v1.YCoord) / ScaleFactor}
	e := model.XY{X: float64(v2.XCoord) / ScaleFactor, Y: float64(-v2.YCoord) / ScaleFactor}

	//s := model.XY{X: float64(v1.XCoord), Y: float64(-v1.YCoord)}
	//e := model.XY{X: float64(v2.XCoord), Y: float64(-v2.YCoord)}

	kind := DefinitionWall
	if lineDef.Flags&int16(lumps2.TwoSided) != 0 {
		kind = DefinitionVoid
	}

	is := model.NewInputSegment(parentId, kind, s, e)

	_, side := b.level.SegmentSideDef(seg, lineDef)
	if side != nil {
		is.Upper = side.UpperTexture
		is.Middle = side.MiddleTexture
		is.Lower = side.LowerTexture
		b.textures[side.UpperTexture] = true
		b.textures[side.MiddleTexture] = true
		b.textures[side.LowerTexture] = true
	}

	return is
}

func (b *Builder2) isPointEqualScaled(p model.XY, v *lumps2.Vertex) bool {
	return math.Abs(p.X-(float64(v.XCoord)/ScaleFactor)) < 1e-2 &&
		math.Abs(p.Y-(float64(-v.YCoord)/ScaleFactor)) < 1e-2
}
