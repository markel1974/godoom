package model

import (
	"errors"
	"fmt"
	"math"
	"strings"
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

// Compiler represents a 3D map compiler that manages sectors, their heights, and an internal cache for fast lookups.
type Compiler struct {
	sectors          *Sectors
	things           []*Thing
	player           *Player
	sectorsMaxHeight float64
	entities         *Entities
}

// NewCompiler initializes and returns a new instance of the Compiler type with default values.
func NewCompiler() *Compiler {
	return &Compiler{
		sectors:          nil,
		things:           nil,
		player:           nil,
		sectorsMaxHeight: 0,
		entities:         nil,
	}
}

// Setup initializes the sectors and segments for the compiler based on the provided configuration.
func (r *Compiler) Setup(cfg *ConfigRoot) error {
	var totalSegments int

	r.sectors, totalSegments = r.compileSectors(cfg)

	r.compileLights(r.sectors)

	r.sectorsMaxHeight = r.scale(cfg, r.sectors)

	var err error

	r.entities = NewEntities(uint(1 + len(cfg.Things)))

	if r.things, err = r.createThings(cfg, r.sectors, r.entities); err != nil {
		return err
	}

	if r.player, err = r.createPlayer(cfg.Player, r.sectors, r.entities); err != nil {
		return err
	}

	fmt.Printf("Scan complete sectors: %d, segments: %d\n", r.sectors.Len(), totalSegments)

	return nil
}

// GetEntities retrieves the Entities instance managed by the Compiler.
func (r *Compiler) GetEntities() *Entities {
	return r.entities
}

// GetSectors retrieves the list of sectors associated with the Compiler instance.
func (r *Compiler) GetSectors() *Sectors {
	return r.sectors
}

// GetThings retrieves the list of compiled things associated with the Compiler instance.
func (r *Compiler) GetThings() []*Thing {
	return r.things
}

// GetPlayer retrieves the Player instance associated with the Compiler.
func (r *Compiler) GetPlayer() *Player {
	return r.player
}

// GetSector retrieves a Sector from the cache using the provided sectorId. Returns an error if the sectorId is invalid.
func (r *Compiler) GetSector(sectorId string) (*Sector, error) {
	s := r.sectors.GetSector(sectorId)
	if s == nil {
		return nil, errors.New(fmt.Sprintf("invalid Sector: %s", sectorId))
	}
	return s, nil
}

// GetMaxHeight returns the maximum height difference between the floor and ceiling among all sectors in the Compiler.
func (r *Compiler) GetMaxHeight() float64 {
	return r.sectorsMaxHeight
}

func (r *Compiler) compileSectors(cfg *ConfigRoot) (*Sectors, int) {
	modelSectorId := uint16(0)
	var container []*Sector
	for idx, cs := range cfg.Sectors {
		var segments []*Segment
		var tags []string
		for _, cn := range cs.Segments {
			tags = append(tags, cn.Tag)
			aUpper := cfg.GetAnimation(cn.Upper)
			aMiddle := cfg.GetAnimation(cn.Middle)
			aLower := cfg.GetAnimation(cn.Lower)
			seg := NewSegment(cn.Neighbor, nil, cn.Kind, cn.Start, cn.End, cn.Tag, aUpper, aMiddle, aLower)
			segments = append(segments, seg)
		}

		if len(segments) == 0 {
			fmt.Printf("Sector %s (idx: %d): vertices as zero len, removing\n", cs.Id, idx)
			continue
		}

		texFloor := cfg.GetAnimation(cs.Floor)
		texCeil := cfg.GetAnimation(cs.Ceil)

		s := NewSector(modelSectorId, cs.Id, segments, texFloor, texCeil)
		modelSectorId++
		s.Tag = cs.Tag + "[" + strings.Join(tags, ";") + "]"
		s.CeilY = cs.CeilY
		s.FloorY = cs.FloorY
		s.Light = NewLight()
		if cs.Light != nil {
			lXY := cs.GetCentroid()
			lightZ := (cs.FloorY + cs.CeilY) * 1.3
			//TODO TERMINARE CON TUTTI I TIPI DI LUCE
			if cs.Light.Kind == LightKindAmbient {
				lightZ = (cs.FloorY + cs.CeilY) * 1000
			}
			s.Light.Setup(cs.Light.Intensity, cs.Light.Kind, XYZ{X: lXY.X, Y: lXY.Y, Z: lightZ})
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
	return sectors, totalSegments
}

func (r *Compiler) compileLights(sectors *Sectors) {
	// --- RAGGRUPPAMENTO AREE (MERGE DEI CENTROIDI DI LUCE) ---
	// Unifica i triangoli adiacenti che appartengono allo stesso settore macroscopico.
	visited := make(map[string]bool)
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

			globalCenterX := sumX / totalArea
			globalCenterY := sumY / totalArea

			// Assegniamo il nuovo centro luce globale a tutti i frammenti dell'area
			for _, s := range areaSectors {
				s.Light.pos.X = globalCenterX
				s.Light.pos.Y = globalCenterY
			}
		}
	}
}

func (r *Compiler) createPlayer(cfg *ConfigPlayer, sectors *Sectors, entities *Entities) (*Player, error) {
	pSector := r.sectors.GetSector(cfg.Sector)
	if pSector == nil {
		return nil, fmt.Errorf("can't find player sector at %s", cfg.Sector)
	}
	player := NewPlayer(cfg, pSector, sectors, entities, false)
	return player, nil
}

// compileThings processes raw ConfigThing entities, resolving their sector references and storing them in the compiler.
func (r *Compiler) createThings(cfg *ConfigRoot, sectors *Sectors, entities *Entities) ([]*Thing, error) {
	var things []*Thing
	for _, ct := range cfg.Things {
		sector := sectors.GetSector(ct.Sector)
		if sector == nil {
			return nil, fmt.Errorf("can't find thing sector at %s", ct.Sector)
		}
		thing := NewThing(ct, cfg.GetAnimation(ct.Animation), sector, sectors, entities)
		things = append(things, thing)
	}
	return things, nil
}

// finalize adjusts player position and sector dimensions based on the scale factor and calculates the maximum sector height.
func (r *Compiler) scale(cfg *ConfigRoot, sectors *Sectors) float64 {
	scale := cfg.ScaleFactor
	if scale < 1 {
		scale = 1
	}

	cfg.Player.Position.Scale(scale)

	for _, t := range cfg.Things {
		t.Position.Scale(scale)
	}

	sectorsMaxHeight := float64(0)
	for _, sect := range sectors.GetSectors() {
		//lights scale
		sect.Light.pos.ScaleXY(scale)
		//vertex scale
		for s := 0; s < len(sect.Segments); s++ {
			sect.Segments[s].Start.Scale(scale)
			sect.Segments[s].End.Scale(scale)
		}
		//maxHeight
		if h := math.Abs(sect.CeilY - sect.FloorY); h > r.sectorsMaxHeight {
			sectorsMaxHeight = h
		}
	}
	return sectorsMaxHeight
}
