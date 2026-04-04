package model

import (
	"fmt"
	"math"

	"github.com/markel1974/godoom/mr_tech/model/config"
	"github.com/markel1974/godoom/mr_tech/model/geometry"
	"github.com/markel1974/godoom/mr_tech/model/mathematic"
	"github.com/markel1974/godoom/mr_tech/physics"
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
func (r *Compiler) Compile(cfg *config.ConfigRoot) error {
	var totalSegments int
	scale := cfg.ScaleFactor
	if scale < 1 {
		scale = 1
	}

	animations := NewAnimations(cfg.Textures)

	r.sectors, totalSegments = r.compileSectors(cfg, animations)

	cfg.Player.Position.Scale(scale)

	for _, t := range cfg.Things {
		t.Position.Scale(scale)
	}

	//for _, l := range r.lights {
	//	l.pos.Scale(scale)
	//}

	for _, sect := range r.sectors.GetSectors() {
		//legacy lights scale
		sect.Light.pos.Scale(scale)

		//sect.CeilY /= scale
		//sect.FloorY /= scale

		//vertex scale
		for s := 0; s < len(sect.Segments); s++ {
			sect.Segments[s].Start.Scale(scale)
			sect.Segments[s].End.Scale(scale)
		}
	}

	//after scaling

	var err error

	r.sectors.CreateTree()

	r.lights, err = r.compileSectorsLights(r.sectors)
	if err != nil {
		return err
	}

	r.entities = NewEntities(uint(1 + len(cfg.Things)))

	if r.things, err = NewThings(cfg.Things, r.sectors, r.entities, animations); err != nil {
		return err
	}

	if r.player, err = NewThingPlayer(cfg.Player, r.sectors, r.entities, false); err != nil {
		return err
	}

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

// compileSectors constructs and processes game sectors based on configuration data, animations, and geometry relationships.
func (r *Compiler) compileSectors(cfg *config.ConfigRoot, anim *Animations) (*Sectors, int) {
	modelSectorId := uint16(0)
	var container []*Sector
	totalPolygons := 0
	var fixSegments []*Segment
	segmentsTree := physics.NewAABBTree(1024)
	emptyAnim := anim.GetAnimation(nil)
	ve := NewVertexEdges(0.001)
	ve.Construct(cfg.Vertices, cfg.Sectors)

	for csIdx, cs := range cfg.Sectors {
		if len(cs.Segments) == 0 {
			continue
		}
		triContainer, _, err := ve.GetTriangles(csIdx)
		if err != nil {
			fmt.Println("Error retrieving polygons for sector", csIdx, ":", err.Error())
			continue
		}
		for _, triangles := range triContainer {
			for _, tri := range triangles {
				s := NewSector(modelSectorId, cs.Id, cs.FloorY, anim.GetAnimation(cs.Floor), cs.CeilY, anim.GetAnimation(cs.Ceil), cs.Tag)
				modelSectorId++
				container = append(container, s)
				// Mantiene il Winding Order consistente per ContainsPoint
				if mathematic.PointSideF(tri[2].X, tri[2].Y, tri[0].X, tri[0].Y, tri[1].X, tri[1].Y) < 0 {
					tri[1], tri[2] = tri[2], tri[1]
				}
				for k := 0; k < 3; k++ {
					p1 := tri[k]
					p2 := tri[(k+1)%3]
					var origSeg *config.ConfigSegment
					// Match topologico ESATTO
					for _, cn := range cs.Segments {
						if (p1 == cn.Start && p2 == cn.End) || (p1 == cn.End && p2 == cn.Start) {
							origSeg = cn
							break
						}
					}
					start := geometry.XY{X: p1.X, Y: p1.Y}
					end := geometry.XY{X: p2.X, Y: p2.Y}
					kind := config.DefinitionUnknown
					upper, middle, lower := emptyAnim, emptyAnim, emptyAnim
					tag := "unknown"
					if origSeg != nil {
						kind = origSeg.Kind
						tag = origSeg.Tag
						upper = anim.GetAnimation(origSeg.Upper)
						middle = anim.GetAnimation(origSeg.Middle)
						lower = anim.GetAnimation(origSeg.Lower)
					}
					s.AddTag(tag)
					seg := NewSegment(nil, config.DefinitionUnknown, start, end, tag, upper, middle, lower)
					if kind == config.DefinitionWall {
						seg.Kind = config.DefinitionWall
					} else {
						fixSegments = append(fixSegments, seg)
					}
					seg.ComputeAABB()
					segmentsTree.InsertObject(seg)
					s.AddSegment(seg)
				}
				s.Light = NewLight()
				if cs.Light != nil {
					s.Light.Setup(nil, cs.Light.Intensity, cs.Light.Kind, s.GetCentroid(), cs.FloorY+cs.CeilY)
				}
				totalPolygons++
			}
		}
	}

	// Risoluzione adiacenze
	for _, seg := range fixSegments {
		// already linked
		if seg.Kind == config.DefinitionJoin || seg.Kind == config.DefinitionWall {
			continue
		}
		bestDistSq := math.MaxFloat64
		var bestNeighborSeg *Segment
		segmentsTree.QueryOverlaps(seg, func(object physics.IAABB) bool {
			overlapSeg, ok := object.(*Segment)
			if !ok || overlapSeg.Parent == seg.Parent {
				return false
			}
			dx1 := seg.Start.X - overlapSeg.End.X
			dy1 := seg.Start.Y - overlapSeg.End.Y
			dx2 := seg.End.X - overlapSeg.Start.X
			dy2 := seg.End.Y - overlapSeg.Start.Y
			distSq := (dx1 * dx1) + (dy1 * dy1) + (dx2 * dx2) + (dy2 * dy2)
			if distSq < bestDistSq {
				bestDistSq = distSq
				bestNeighborSeg = overlapSeg
			}
			return false
		})
		if bestNeighborSeg != nil {
			// Link reciproco (O(N/2)
			bestNeighborSeg.Kind = config.DefinitionJoin
			bestNeighborSeg.SetNeighbor(seg.Parent)
			seg.Kind = config.DefinitionJoin
			seg.SetNeighbor(bestNeighborSeg.Parent)
		} else {
			seg.Kind = config.DefinitionWall
			seg.SetNeighbor(nil)
		}
	}

	return NewSectors(container), totalPolygons
}

// compileLights processes and merges adjacent sectors with similar properties into unified lighting areas.
func (r *Compiler) compileSectorsLights(sectors *Sectors) ([]*Light, error) {
	// --- RAGGRUPPAMENTO AREE (MERGE DEI CENTROIDI DI LUCE) ---
	// Unifica i triangoli adiacenti che appartengono allo stesso settore macroscopico.
	visited := make(map[string]bool)
	var out []*Light
	for sectIdx, sect := range sectors.GetSectors() {
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
				if seg.Kind != config.DefinitionWall && seg.Neighbor != nil {
					n := seg.Neighbor
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
			if totalArea == 0 {
				fmt.Println("total area is zero")
				continue
			}

			globalCenter := geometry.XY{X: sumX / totalArea, Y: sumY / totalArea}

			// Assegniamo il nuovo centro luce globale a tutti i frammenti dell'area
			for _, s := range areaSectors {
				s.Light.pos.X = globalCenter.X
				s.Light.pos.Y = globalCenter.Y
			}
			first := areaSectors[0]
			light := NewLight()
			sector := r.sectors.QueryPoint(first.Light.pos.X, first.Light.pos.Y)
			if sector == nil {
				sector = first
				fmt.Printf("Warning: sector not found for light position (idx:%d x:%f, y:%f)\n", sectIdx, first.Light.pos.X, first.Light.pos.Y)
			}
			light.Setup(sector, first.Light.intensity, first.Light.kind, globalCenter, first.CeilY)
			out = append(out, light)
		} else if len(areaSectors) == 1 {
			first := areaSectors[0]
			light := NewLight()
			light.Setup(first, first.Light.intensity, first.Light.kind, first.GetCentroid(), first.CeilY)
			out = append(out, light)
		}
	}
	return out, nil
}
