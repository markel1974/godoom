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
	volumes  *Volumes
	things   *Things
	player   *ThingPlayer
	lights   []*Light
	entities *Entities
}

// NewCompiler initializes and returns a new instance of Compiler with default nil-initialized fields.
func NewCompiler() *Compiler {
	return &Compiler{
		volumes:  nil,
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

	r.volumes, totalSegments = r.compile2d(cfg, animations)

	cfg.Player.Position.Scale(scale)

	for _, t := range cfg.Things {
		t.Position.Scale(scale)
	}

	//for _, l := range r.lights {
	//	l.pos.Scale(scale)
	//}

	for _, sect := range r.volumes.GetVolumes() {
		//legacy lights scale
		sect.Light.pos.Scale(scale)

		//sect.CeilY /= scale
		//sect.FloorY /= scale

		//vertex scale
		for _, face := range sect.GetFaces() {
			face.Scale2D(scale)
		}
	}

	//after scaling

	var err error

	r.volumes.CreateTree()

	r.lights, err = r.compileSectorsLights(r.volumes)
	if err != nil {
		return err
	}

	r.entities = NewEntities(uint(1 + len(cfg.Things)))

	if r.things, err = NewThings(cfg.Things, r.volumes, r.entities, animations); err != nil {
		return err
	}

	if r.player, err = NewThingPlayer(cfg.Player, r.volumes, r.entities, false); err != nil {
		return err
	}

	fmt.Printf("Scan complete sectors: %d, segments: %d\n", r.volumes.Len(), totalSegments)

	return nil
}

// GetEntities returns the Entities instance managed by the Compiler.
func (r *Compiler) GetEntities() *Entities {
	return r.entities
}

// GetVolumes retrieves the Volumes instance associated with the current Compiler object.
func (r *Compiler) GetVolumes() *Volumes {
	return r.volumes
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

// compile2d constructs and processes game sectors based on configuration data, animations, and geometry relationships.
func (r *Compiler) compile2d(cfg *config.ConfigRoot, anim *Animations) (*Volumes, int) {
	modelSectorId := 0
	var container []*Volume
	totalPolygons := 0
	var fixFaces []*Face
	facesTree := physics.NewAABBTree(1024)
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
				s := NewVolume(modelSectorId, cs.Id, cs.FloorY, anim.GetAnimation(cs.Floor), cs.CeilY, anim.GetAnimation(cs.Ceil), cs.Tag)
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
					start := geometry.XYZ{X: p1.X, Y: p1.Y, Z: 0}
					end := geometry.XYZ{X: p2.X, Y: p2.Y, Z: 0}
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
					face := NewFaceSegment(nil, config.DefinitionUnknown, start, end, tag, upper, middle, lower)
					if kind == config.DefinitionWall {
						face.SetKind(config.DefinitionWall)
					} else {
						fixFaces = append(fixFaces, face)
					}
					face.Rebuild()
					facesTree.InsertObject(face)
					s.AddFace(face)
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
	for _, seg := range fixFaces {
		// already linked
		if seg.GetKind() == config.DefinitionJoin || seg.GetKind() == config.DefinitionWall {
			continue
		}
		bestDistSq := math.MaxFloat64
		var bestNeighborFace *Face
		facesTree.QueryOverlaps(seg, func(object physics.IAABB) bool {
			overlapFace, ok := object.(*Face)
			if !ok || overlapFace.GetParent() == seg.GetParent() {
				return false
			}
			start := seg.GetStart()
			end := seg.GetEnd()
			overlapStart := overlapFace.GetStart()
			overlapEnd := overlapFace.GetEnd()
			dx1 := start.X - overlapEnd.X
			dy1 := start.Y - overlapEnd.Y
			dx2 := end.X - overlapStart.X
			dy2 := end.Y - overlapStart.Y
			distSq := (dx1 * dx1) + (dy1 * dy1) + (dx2 * dx2) + (dy2 * dy2)
			if distSq < bestDistSq {
				bestDistSq = distSq
				bestNeighborFace = overlapFace
			}
			return false
		})
		if bestNeighborFace != nil {
			// Link reciproco (O(N/2)
			bestNeighborFace.SetKind(config.DefinitionJoin)
			bestNeighborFace.SetNeighbor(seg.GetParent())
			seg.SetKind(config.DefinitionJoin)
			seg.SetNeighbor(bestNeighborFace.GetParent())
		} else {
			seg.SetKind(config.DefinitionWall)
			seg.SetNeighbor(nil)
		}
	}

	return NewVolumes(container), totalPolygons
}

// compileLights processes and merges adjacent sectors with similar properties into unified lighting areas.
func (r *Compiler) compileSectorsLights(sectors *Volumes) ([]*Light, error) {
	// --- RAGGRUPPAMENTO AREE (MERGE DEI CENTROIDI DI LUCE) ---
	// Unifica i triangoli adiacenti che appartengono allo stesso settore macroscopico.
	visited := make(map[string]bool)
	var out []*Light
	for sectIdx, sect := range sectors.GetVolumes() {
		if visited[sect.GetId()] {
			continue
		}
		// Utilizziamo un algoritmo di Flood Fill per trovare tutti i settori connessi
		var areaSectors []*Volume
		queue := []*Volume{sect}
		visited[sect.GetId()] = true

		for len(queue) > 0 {
			curr := queue[0]
			queue = queue[1:]
			areaSectors = append(areaSectors, curr)

			// Controlla i vicini di questo settore
			for _, seg := range curr.GetFaces() {
				if seg.GetKind() != config.DefinitionWall && seg.GetNeighbor() != nil {
					n := seg.GetNeighbor()
					if !visited[n.GetId()] {
						// Condizione di "Stessa Area": adiacenti e con stesse quote/luci
						if n.GetCeilY() == curr.GetCeilY() && n.GetFloorY() == curr.GetFloorY() && n.Light.intensity == curr.Light.intensity {
							visited[n.GetId()] = true
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
				for _, face := range s.GetFaces() {
					start := face.GetStart()
					end := face.GetEnd()
					x0, y0 := start.X, start.Y
					x1, y1 := end.X, end.Y
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

			globalCenter := geometry.XYZ{X: sumX / totalArea, Y: sumY / totalArea, Z: 0.0}

			// Assegniamo il nuovo centro luce globale a tutti i frammenti dell'area
			for _, s := range areaSectors {
				s.Light.pos.X = globalCenter.X
				s.Light.pos.Y = globalCenter.Y
			}
			first := areaSectors[0]
			light := NewLight()
			sector := r.volumes.QueryPoint2d(first.Light.pos.X, first.Light.pos.Y)
			if sector == nil {
				sector = first
				fmt.Printf("Warning: sector not found for light position (idx:%d x:%f, y:%f)\n", sectIdx, first.Light.pos.X, first.Light.pos.Y)
			}
			light.Setup(sector, first.Light.intensity, first.Light.kind, globalCenter, first.GetCeilY())
			out = append(out, light)
		} else if len(areaSectors) == 1 {
			first := areaSectors[0]
			light := NewLight()
			light.Setup(first, first.Light.intensity, first.Light.kind, first.GetCentroid(), first.GetCeilY())
			out = append(out, light)
		}
	}
	return out, nil
}
