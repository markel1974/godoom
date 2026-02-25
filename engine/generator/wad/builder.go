package wad

import (
	"errors"
	"fmt"
	"math"
	"strconv"

	"github.com/markel1974/godoom/engine/generator/wad/lumps"
	"github.com/markel1974/godoom/engine/model"
)

type Builder struct {
	w        *WAD
	textures map[string]bool
	level    *Level
	bsp      *BSP
}

func NewBuilder() *Builder {
	return &Builder{
		textures: make(map[string]bool),
		level:    nil,
	}
}

func (b *Builder) Setup(wadFile string, levelNumber int) (*model.InputConfig, error) {
	b.w = New()
	if err := b.w.Load(wadFile); err != nil {
		return nil, err
	}
	levelNames := b.w.GetLevels()
	if len(levelNames) == 0 {
		return nil, errors.New("error: No levels found")
	}
	levelName := levelNames[levelNumber-1]

	var err error
	b.level, err = b.w.GetLevel(levelName)
	if err != nil {
		return nil, err
	}

	b.bsp = NewBsp(b.level)

	// 1. Decostruzione top-down: operiamo in unità Doom intere per massima precisione
	hulls := b.ExtractSubSectorHulls()

	// 2. Costruzione della topologia: qui applichiamo ScaleFactor
	sectors := b.buildPortalTopology(hulls)

	p1 := b.level.Things[1]
	// Posizione Player riscalata coerentemente
	position := model.XY{X: float64(p1.X) / ScaleFactor, Y: float64(-p1.Y) / ScaleFactor}
	_, playerSSectorId, _ := b.bsp.FindSector(p1.X, p1.Y)

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

func (b *Builder) ExtractSubSectorHulls() map[uint16]Polygon {
	hulls := make(map[uint16]Polygon)
	if len(b.level.Nodes) == 0 {
		return hulls
	}

	rootNodeIdx := uint16(len(b.level.Nodes) - 1)
	const maxBBox = 32768.0
	// Iniziamo con un poligono in unità Doom
	globalPoly := Polygon{
		{X: -maxBBox, Y: -maxBBox},
		{X: maxBBox, Y: -maxBBox},
		{X: maxBBox, Y: maxBBox},
		{X: -maxBBox, Y: maxBBox},
	}

	var traverse func(nodeIdx uint16, currentPoly Polygon)
	traverse = func(nodeIdx uint16, currentPoly Polygon) {
		if nodeIdx&subSectorBit != 0 {
			hulls[nodeIdx&^subSectorBit] = currentPoly
			return
		}

		node := b.level.Nodes[nodeIdx]
		// Nota: nx, ny, ndx, ndy in unità Doom originali per il clipping
		nx, ny := float64(node.X), float64(-node.Y)
		ndx, ndy := float64(node.DX), float64(-node.DY)

		// Doom BSP: Child[0] è Right/Front, Child[1] è Left/Back
		right := b.clipPolygon(currentPoly, nx, ny, ndx, ndy, true)
		left := b.clipPolygon(currentPoly, nx, ny, ndx, ndy, false)

		if len(right) > 2 {
			traverse(node.Child[0], right)
		}
		if len(left) > 2 {
			traverse(node.Child[1], left)
		}
	}

	traverse(rootNodeIdx, globalPoly)
	return hulls
}

func (b *Builder) clipPolygon(poly Polygon, nx, ny, ndx, ndy float64, rightSide bool) Polygon {
	var out Polygon
	if len(poly) == 0 {
		return out
	}

	// Matematicamente corretto: test del segno del prodotto vettoriale 2D
	isInside := func(pt model.XY) bool {
		// Normalizzato per Doom: (pt - origin) x direction
		val := (pt.X-nx)*ndy - (pt.Y-ny)*ndx
		if rightSide {
			return val <= 0
		}
		return val >= 0
	}

	intersect := func(p1, p2 model.XY) model.XY {
		// Retta A (segmento poligono): p1 + u*(p2-p1)
		// Retta B (partizione BSP): (nx,ny) + v*(ndx,ndy)
		x1, y1, x2, y2 := p1.X, p1.Y, p2.X, p2.Y
		den := (x1-x2)*ndy - (y1-y2)*ndx
		if math.Abs(den) < 1e-9 {
			return p1
		}
		t := ((x1-nx)*ndy - (y1-ny)*ndx) / den
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

func (b *Builder) buildPortalTopology(hulls map[uint16]Polygon) []*model.InputSector {
	var out []*model.InputSector

	for ssId, poly := range hulls {
		sectorRef, ok := b.level.GetSectorFromSubSector(ssId)
		if !ok {
			continue
		}
		doomSector := b.level.Sectors[sectorRef]
		subSector := b.level.SubSectors[ssId]
		idStr := strconv.Itoa(int(ssId))

		mSector := model.NewInputSector(idStr)
		mSector.Floor = float64(doomSector.FloorHeight) / ScaleFactor
		mSector.Ceil = float64(doomSector.CeilingHeight) / ScaleFactor

		// Tag usato per la luce (normalizzato 0..1)
		mSector.Tag = fmt.Sprintf("%f", float64(doomSector.LightLevel)/255.0)

		// 1. Aggiungiamo i segmenti REALI dal WAD
		// Questi sono i muri visibili con texture
		for j := int16(0); j < subSector.NumSegments; j++ {
			seg := b.level.Segments[subSector.StartSeg+j]
			mSector.Segments = append(mSector.Segments, b.createInputSegmentFromSeg(idStr, seg))
		}

		// 2. Usiamo la Hull clippata per chiudere il poligono (Portali invisibili)
		// Se un lato della Hull non ha un corrispettivo nei Segs, è un'apertura BSP.
		for i := 0; i < len(poly); i++ {
			uP1 := poly[i]
			uP2 := poly[(i+1)%len(poly)]

			// Evitiamo segmenti degeneri (micro-scarti del clipping)
			if b.dist(uP1, uP2) < 0.5 {
				continue
			}

			// Verifichiamo se questo lato della hull esiste già come SEG reale
			exists := false
			for j := int16(0); j < subSector.NumSegments; j++ {
				seg := b.level.Segments[subSector.StartSeg+j]
				v1 := b.level.Vertexes[seg.VertexStart]
				v2 := b.level.Vertexes[seg.VertexEnd]

				// Matching geometrico del segmento (non solo dei punti)
				if b.isSegMatch(uP1, uP2, v1, v2) {
					exists = true
					break
				}
			}

			if !exists {
				// Generiamo un portale invisibile (DefinitionVoid)
				p1 := model.XY{X: uP1.X / ScaleFactor, Y: uP1.Y / ScaleFactor}
				p2 := model.XY{X: uP2.X / ScaleFactor, Y: uP2.Y / ScaleFactor}

				mSeg := model.NewInputSegment(idStr, DefinitionVoid, p1, p2)

				// Calcolo del vicino tramite probe nel BSP
				midX := (uP1.X + uP2.X) / 2
				midY := (uP1.Y + uP2.Y) / 2
				dx, dy := uP2.X-uP1.X, uP2.Y-uP1.Y
				mag := math.Sqrt(dx*dx + dy*dy)

				// Spostamento infinitesimale verso l'esterno della Hull
				probeX := midX + (dy/mag)*0.5
				probeY := midY - (dx/mag)*0.5

				_, neighborSSId, _ := b.bsp.FindSector(int16(probeX), int16(-probeY))
				if neighborSSId != ssId && neighborSSId < 32768 {
					mSeg.Neighbor = strconv.Itoa(int(neighborSSId))
				}

				mSector.Segments = append(mSector.Segments, mSeg)
			}
		}
		out = append(out, mSector)
	}
	return out
}

// Supporto per matching robusto
func (b *Builder) isSegMatch(p1, p2 model.XY, v1, v2 *lumps.Vertex) bool {
	vv1 := model.XY{X: float64(v1.XCoord), Y: float64(-v1.YCoord)}
	vv2 := model.XY{X: float64(v2.XCoord), Y: float64(-v2.YCoord)}

	// Confronto midpoint per evitare errori di floating point sui vertici clippati
	midH := model.XY{X: (p1.X + p2.X) / 2, Y: (p1.Y + p2.Y) / 2}
	midV := model.XY{X: (vv1.X + vv2.X) / 2, Y: (vv1.Y + vv2.Y) / 2}

	return b.dist(midH, midV) < 1.0 // Tolleranza di 1 unità Doom
}

func (b *Builder) dist(c, d model.XY) float64 {
	return math.Sqrt(math.Pow(c.X-d.X, 2) + math.Pow(c.Y-d.Y, 2))
}

func (b *Builder) isPointEqual(p model.XY, v *lumps.Vertex) bool {
	return math.Abs(p.X-float64(v.XCoord)) < 1e-1 && math.Abs(p.Y-float64(-v.YCoord)) < 1e-1
}

func (b *Builder) createInputSegmentFromSeg(parentId string, seg *lumps.Seg) *model.InputSegment {
	lineDef := b.level.LineDefs[seg.LineDef]
	v1, v2 := b.level.Vertexes[seg.VertexStart], b.level.Vertexes[seg.VertexEnd]

	s := model.XY{X: float64(v1.XCoord) / ScaleFactor, Y: float64(-v1.YCoord) / ScaleFactor}
	e := model.XY{X: float64(v2.XCoord) / ScaleFactor, Y: float64(-v2.YCoord) / ScaleFactor}

	kind := DefinitionWall
	if lineDef.Flags&int16(lumps.TwoSided) != 0 {
		kind = DefinitionVoid
	}

	is := model.NewInputSegment(parentId, kind, s, e)
	// ... (caricamento texture rimosso per brevità, mantieni il tuo originale)
	return is
}
