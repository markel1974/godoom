package model

import (
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/markel1974/godoom/engine/textures"
)

// DefinitionJoin represents the join type with an assigned value of 3.
// DefinitionVoid represents the void type with an assigned value of 1.
// DefinitionWall represents the wall type with an assigned value of 2.
// DefinitionUnknown represents an unknown type with an assigned value of 0.
const (
	DefinitionJoin    = 3
	DefinitionVoid    = 1
	DefinitionWall    = 2
	DefinitionUnknown = 0
)

// segment represents an internal 2D segment structure with start and end points, sector reference, and index.
type segment struct {
	start  XY
	end    XY
	sector *Sector
	np     int
}

// edgeKey represents a unique key for an edge in 2D space, defined by the coordinates of its start and end points.
type edgeKey struct {
	x1, y1, x2, y2 float64
}

// makeEdgeKey generates a unique edgeKey for an edge defined by two points, used for identifying edges in a 2D space.
func makeEdgeKey(start XY, end XY) edgeKey {
	return edgeKey{
		x1: start.X,
		y1: start.Y,
		x2: end.X,
		y2: end.Y,
	}
}

// Compiler represents a 3D map compiler that manages sectors, their heights, and an internal cache for fast lookups.
type Compiler struct {
	sectors          []*Sector
	sectorsMaxHeight float64
	cache            map[string]*Sector
	textures         textures.ITextures
}

// NewCompiler initializes and returns a new instance of the Compiler type with default values.
func NewCompiler() *Compiler {
	return &Compiler{
		sectors:          nil,
		sectorsMaxHeight: 0,
		cache:            make(map[string]*Sector),
	}
}

// Setup initializes the sectors and segments for the compiler based on the provided configuration.
func (r *Compiler) Setup(cfg *ConfigRoot) error {
	r.textures = cfg.Textures
	modelSectorId := uint16(0)
	for idx, cs := range cfg.Sectors {
		var segments []*Segment
		var tags []string
		for _, cn := range cs.Segments {
			tags = append(tags, cn.Tag)
			aUpper := cfg.GetAnimation(cn.Animations.Upper)
			aMiddle := cfg.GetAnimation(cn.Animations.Middle)
			aLower := cfg.GetAnimation(cn.Animations.Lower)
			segments = append(segments, NewSegment(cn.Neighbor, nil, cn.Kind, cn.Start, cn.End, cn.Tag, aUpper, aMiddle, aLower))
		}

		if len(segments) == 0 {
			fmt.Printf("Sector %s (idx: %d): vertices as zero len, removing\n", cs.Id, idx)
			continue
		}

		texFloor := cfg.GetAnimation(cs.Animations.Floors)
		texCeil := cfg.GetAnimation(cs.Animations.Ceils)
		texScaleFactor := cs.Animations.ScaleFactor

		s := NewSector(modelSectorId, cs.Id, segments, texFloor, texCeil, texScaleFactor)
		modelSectorId++
		s.Tag = cs.Tag + "[" + strings.Join(tags, ";") + "]"
		s.CeilY = cs.CeilY
		s.FloorY = cs.FloorY
		s.Light = NewLight()
		if cs.Light != nil {
			lXY := cs.GetCentroid()
			s.Light.Setup(cs.Light.Intensity, cs.Light.Kind, XYZ{X: lXY.X, Y: lXY.Y, Z: s.CeilY})
		}
		r.sectors = append(r.sectors, s)
		r.cache[cs.Id] = s
	}

	for _, sect := range r.sectors {
		for _, seg := range sect.Segments {
			if seg.Kind != DefinitionWall {
				if s, ok := r.cache[seg.Ref]; ok {
					seg.SetSector(s.Id, s)
				} else {
					//fmt.Println("OUT", segment.Ref)
					//os.Exit(-1)
				}
			}
		}
	}

	if !cfg.DisableLoop {
		//Verify Loop
		for _, sector := range r.sectors {
			if len(sector.Segments) == 1 {
				continue
			}
			vFirst := sector.Segments[0]
			vLast := sector.Segments[len(sector.Segments)-1]
			hasLoop := vFirst.Start.X == vLast.End.X && vFirst.Start.Y == vLast.End.Y
			if !hasLoop {
				fmt.Printf("creating loop for Sector %s\n", sector.Id)
				k := vLast.Copy()
				k.Start = k.End
				k.End = vFirst.Start
				sector.Segments = append(sector.Segments, k)
			}
		}

		//Rescan:
		// Verify that for each edge that has a neighbor, the neighbor has this same neighbor.
		fixed := 0
		undefined := 0
		lineDefsCache := r.makeSegmentsCache()
		for _, sector := range r.sectors {
			for np, s := range sector.Segments {
				if s.Kind != DefinitionWall {
					if ld, ok := lineDefsCache[makeEdgeKey(s.End, s.Start)]; ok {
						if s.Ref != ld.sector.Id {
							fmt.Printf("p1 - Sector %s (segment: %d): Neighbor behind line (%g, %g) - (%g, %g) should be %s, %s found instead. Fixing...\n", sector.Id, np, s.Start.X, s.Start.Y, s.End.X, s.End.Y, ld.sector.Id, s.Ref)
							if s.Kind == DefinitionUnknown {
								s.Kind = DefinitionJoin
							}
							s.SetSector(ld.sector.Id, ld.sector)
							fixed++
						}
					} else {
						s.Kind = DefinitionWall
						s.SetSector("", nil)

						fmt.Printf("p1 - Sector %s (segment: %d): Neighbor behind line (%g, %g) - (%g, %g) %s %s. Opposite not found\n", sector.Id, np, s.Start.X, s.Start.Y, s.End.X, s.End.Y, s.Ref, s.Tag)
						undefined++
					}
				}
			}
		}
		fmt.Println("undefined:", undefined, "fixed:", fixed)
	}

	// --- RAGGRUPPAMENTO AREE (MERGE DEI CENTROIDI DI LUCE) ---
	// Unifica i triangoli adiacenti che appartengono allo stesso settore macroscopico.
	visited := make(map[string]bool)
	for _, sect := range r.sectors {
		if visited[sect.Id] {
			continue
		}

		// Utilizziamo un algoritmo di Flood Fill per trovare tutti i settori connessi
		var areaSectors []*Sector
		queue := []*Sector{sect}
		visited[sect.Id] = true

		for len(queue) > 0 {
			curr := queue[0]
			queue = queue[1:]
			areaSectors = append(areaSectors, curr)

			// Controlla i vicini di questo settore
			for _, seg := range curr.Segments {
				if seg.Kind != DefinitionWall && seg.Ref != "" {
					if n, ok := r.cache[seg.Ref]; ok {
						if !visited[n.Id] {
							// Condizione di "Stessa Area": adiacenti e con stesse quote/luci
							if n.CeilY == curr.CeilY && n.FloorY == curr.FloorY && n.Light.intensity == curr.Light.intensity {
								visited[n.Id] = true
								queue = append(queue, n)
							}
						}
					}
				}
			}
		}

		// Se l'area è composta da più poligoni, calcoliamo un baricentro globale
		if len(areaSectors) > 1 {
			var sumX, sumY float64
			for _, s := range areaSectors {
				// Sommiamo i baricentri dei singoli triangoli calcolati in precedenza
				sumX += s.Light.pos.X
				sumY += s.Light.pos.Y
			}

			globalCenterX := sumX / float64(len(areaSectors))
			globalCenterY := sumY / float64(len(areaSectors))

			// Assegniamo il nuovo centro luce globale a tutti i frammenti dell'area
			for _, s := range areaSectors {
				s.Light.pos.X = globalCenterX
				s.Light.pos.Y = globalCenterY
			}
		}
	}

	r.finalize(cfg)

	fmt.Println("Scan complete")

	return nil
}

// finalize adjusts player position and sector dimensions based on the scale factor and calculates the maximum sector height.
func (r *Compiler) finalize(cfg *ConfigRoot) {
	scale := cfg.ScaleFactor
	if scale < 1 {
		scale = 1
	}

	cfg.Player.Position.Scale(scale)

	r.sectorsMaxHeight = 0
	for _, sect := range r.sectors {
		//lights scale
		sect.Light.pos.ScaleXY(scale)
		//vertex scale
		for s := 0; s < len(sect.Segments); s++ {
			sect.Segments[s].Start.Scale(scale)
			sect.Segments[s].End.Scale(scale)
		}
		//maxHeight
		if h := math.Abs(sect.CeilY - sect.FloorY); h > r.sectorsMaxHeight {
			r.sectorsMaxHeight = h
		}
	}
}

// GetSectors retrieves the list of sectors associated with the Compiler instance.
func (r *Compiler) GetSectors() []*Sector {
	return r.sectors
}

// Get retrieves a Sector from the cache using the provided sectorId. Returns an error if the sectorId is invalid.
func (r *Compiler) Get(sectorId string) (*Sector, error) {
	s, ok := r.cache[sectorId]
	if !ok {
		return nil, errors.New(fmt.Sprintf("invalid Sector: %s", sectorId))
	}
	return s, nil
}

// GetMaxHeight returns the maximum height difference between the floor and ceiling among all sectors in the Compiler.
func (r *Compiler) GetMaxHeight() float64 {
	return r.sectorsMaxHeight
}

// makeSegmentsCache creates and returns a map associating unique edge keys to their corresponding segments.
func (r *Compiler) makeSegmentsCache() map[edgeKey]*segment {
	t := make(map[edgeKey]*segment)
	for _, sect := range r.sectors {
		for np := 0; np < len(sect.Segments); np++ {
			s := sect.Segments[np]
			hash := makeEdgeKey(s.Start, s.End)
			ld := &segment{sector: sect, np: np, start: s.Start, end: s.End}
			if fld, ok := t[hash]; ok {
				if sect.Id != fld.sector.Id {
					//fmt.Println("line segment already added", sect.Id, fld.Sector.Id, hash, np)
				}
			} else {
				t[hash] = ld
			}
		}
	}
	return t
}
