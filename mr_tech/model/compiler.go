package model

import (
	"errors"
	"fmt"
	"math"
	"strings"
)

// DefinitionJoin represents a join action in a system with a numeric value of 3.
// DefinitionVoid represents a void action in a system with a numeric value of 1.
// DefinitionWall represents a wall action in a system with a numeric value of 2.
// DefinitionUnknown represents an undefined state in a system with a numeric value of 0.
const (
	DefinitionJoin    = 3
	DefinitionVoid    = 1
	DefinitionWall    = 2
	DefinitionUnknown = 0
)

// Compiler represents a core game engine component for managing sectors, game objects, player interactions, and entities.
type Compiler struct {
	sectors  *Sectors
	things   *Things
	player   *ThingPlayer
	lights   []*Light
	entities *Entities
}

// NewCompiler initializes and returns a new instance of Compiler with default nil-initialized fields.
func NewCompiler() *Compiler {
	return &Compiler{
		sectors:  nil,
		things:   nil,
		player:   nil,
		entities: nil,
		lights:   nil,
	}
}

// Compile initializes and processes game data from the provided configuration, returning an error if compilation fails.
func (r *Compiler) Compile(cfg *ConfigRoot) error {
	var totalSegments int
	scale := cfg.ScaleFactor
	if scale < 1 {
		scale = 1
	}

	animations := NewAnimations(cfg.Textures)

	r.sectors, totalSegments = r.compileSectors(cfg, animations)

	r.lights = r.compileSectorsLights(r.sectors)

	cfg.Player.Position.Scale(scale)

	for _, t := range cfg.Things {
		t.Position.Scale(scale)
	}

	for _, l := range r.lights {
		l.pos.Scale(scale)
	}

	for _, sect := range r.sectors.GetSectors() {
		//legacy lights scale
		sect.Light.pos.Scale(scale)

		//vertex scale
		for s := 0; s < len(sect.Segments); s++ {
			sect.Segments[s].Start.Scale(scale)
			sect.Segments[s].End.Scale(scale)
		}
	}

	//after scaling

	var err error

	r.sectors.CreateTree()

	r.entities = NewEntities(uint(1 + len(cfg.Things)))

	if r.things, err = NewThings(cfg.Things, r.sectors, r.entities, animations); err != nil {
		return err
	}

	pSector := r.sectors.GetSector(cfg.Player.Sector)
	if pSector == nil {
		return fmt.Errorf("can't find player sector at %s", cfg.Player.Sector)
	}
	r.player = NewThingPlayer(cfg.Player, pSector, r.sectors, r.entities, false)

	fmt.Printf("Scan complete sectors: %d, segments: %d\n", r.sectors.Len(), totalSegments)

	return nil
}

// GetEntities returns the Entities instance managed by the Compiler.
func (r *Compiler) GetEntities() *Entities {
	return r.entities
}

// GetSectors retrieves the Sectors instance associated with the current Compiler object.
func (r *Compiler) GetSectors() *Sectors {
	return r.sectors
}

// GetThings returns the Things instance managed by the Compiler.
func (r *Compiler) GetThings() *Things {
	return r.things
}

// GetPlayer returns the player object associated with the compiler instance.
func (r *Compiler) GetPlayer() *ThingPlayer {
	return r.player
}

// GetLights retrieves the list of Light objects managed by the Compiler.
func (r *Compiler) GetLights() []*Light {
	return r.lights
}

// GetSector retrieves a Sector by its ID. Returns an error if the Sector is not found.
func (r *Compiler) GetSector(sectorId string) (*Sector, error) {
	s := r.sectors.GetSector(sectorId)
	if s == nil {
		return nil, errors.New(fmt.Sprintf("invalid Sector: %s", sectorId))
	}
	return s, nil
}

// compileSectors processes the sector configurations and animations to construct and return the compiled Sectors and total segments.
func (r *Compiler) compileSectors(cfg *ConfigRoot, anim *Animations) (*Sectors, int) {
	modelSectorId := uint16(0)
	var container []*Sector
	for idx, cs := range cfg.Sectors {
		var segments []*Segment
		var tags []string
		for _, cn := range cs.Segments {
			tags = append(tags, cn.Tag)
			aUpper := anim.GetAnimation(cn.Upper)
			aMiddle := anim.GetAnimation(cn.Middle)
			aLower := anim.GetAnimation(cn.Lower)
			seg := NewSegment(cn.Neighbor, nil, cn.Kind, cn.Start, cn.End, cn.Tag, aUpper, aMiddle, aLower)
			segments = append(segments, seg)
		}

		if len(segments) == 0 {
			fmt.Printf("Sector %s (idx: %d): vertices as zero len, removing\n", cs.Id, idx)
			continue
		}

		texFloor := anim.GetAnimation(cs.Floor)
		texCeil := anim.GetAnimation(cs.Ceil)

		s := NewSector(modelSectorId, cs.Id, segments, texFloor, texCeil)
		modelSectorId++
		s.Tag = cs.Tag + "[" + strings.Join(tags, ";") + "]"
		s.CeilY = cs.CeilY
		s.FloorY = cs.FloorY
		s.Light = NewLight()
		if cs.Light != nil {
			s.Light.Setup(cs.Light.Intensity, cs.Light.Kind, s.GetCentroid(), cs.FloorY+cs.CeilY)
		}
		container = append(container, s)
	}

	sectors := NewSectors(container)

	totalSegments := 0
	for _, sect := range sectors.GetSectors() {
		for _, seg := range sect.Segments {
			totalSegments++
			if seg.Kind != DefinitionWall {
				if s := sectors.GetSector(seg.Ref); s != nil {
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
		for _, sector := range sectors.GetSectors() {
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
		lineDefsCache := sectors.MakeSegmentsCache()
		for _, sector := range sectors.GetSectors() {
			for np, s := range sector.Segments {
				if s.Kind != DefinitionWall {
					if ld, ok := lineDefsCache[s.MakeReverseEdgeKey()]; ok {
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
	return sectors, totalSegments
}

// compileLights processes and merges adjacent sectors with similar properties into unified lighting areas.
func (r *Compiler) compileSectorsLights(sectors *Sectors) []*Light {
	// --- RAGGRUPPAMENTO AREE (MERGE DEI CENTROIDI DI LUCE) ---
	// Unifica i triangoli adiacenti che appartengono allo stesso settore macroscopico.
	visited := make(map[string]bool)
	var out []*Light
	for _, sect := range sectors.GetSectors() {
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
					if n := sectors.GetSector(seg.Ref); n != nil {
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
			var sumX, sumY, totalArea float64
			for _, s := range areaSectors {
				// Calcola l'area del triangolo (prodotto vettoriale)
				area := 0.0
				for i := range s.Segments {
					x0, y0 := s.Segments[i].Start.X, s.Segments[i].Start.Y
					x1, y1 := s.Segments[i].End.X, s.Segments[i].End.Y
					area += (x0 * y1) - (x1 * y0)
				}
				area = math.Abs(area * 0.5)
				sumX += s.Light.pos.X * area
				sumY += s.Light.pos.Y * area
				totalArea += area
			}

			globalCenter := XY{X: sumX / totalArea, Y: sumY / totalArea}

			// Assegniamo il nuovo centro luce globale a tutti i frammenti dell'area
			for _, s := range areaSectors {
				s.Light.pos.X = globalCenter.X
				s.Light.pos.Y = globalCenter.Y
			}

			first := areaSectors[0]
			light := NewLight()
			light.Setup(first.Light.intensity, first.Light.kind, globalCenter, first.FloorY+first.CeilY)
			out = append(out, light)
		}
	}
	return out
}
