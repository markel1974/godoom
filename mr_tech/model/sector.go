package model

import (
	"encoding/json"
	"math"

	"github.com/markel1974/godoom/mr_tech/mathematic"
	"github.com/markel1974/godoom/mr_tech/physics"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// Sector represents a 3D space defined by its boundaries, texture animations, lighting, and associated metadata.
type Sector struct {
	ModelId       uint16
	Id            string
	Segments      []*Segment
	Tag           string
	FloorY        float64
	CeilY         float64
	Ceil          *textures.Animation
	Floor         *textures.Animation
	Light         *Light
	usage         int
	compileId     uint64
	references    map[uint64]bool
	VisibleSpans  [][2]float64
	aabb          *physics.AABB
	LastCompileId uint64
}

// NewSector initializes and returns a new Sector instance with specified parameters including model ID, ID, segments, floor, and ceiling.
func NewSector(modelId uint16, id string, segments []*Segment, floor *textures.Animation, ceil *textures.Animation) *Sector {
	s := &Sector{
		ModelId:    modelId,
		Id:         id,
		CeilY:      0,
		FloorY:     0,
		Segments:   segments,
		usage:      0,
		compileId:  0,
		references: make(map[uint64]bool),
		Ceil:       ceil,
		Floor:      floor,
	}
	return s
}

// Reference updates the sector's compile ID if it differs or increments its usage count if it matches.
func (s *Sector) Reference(compileId uint64) {
	if compileId != s.compileId {
		s.compileId = compileId
		s.usage = 0
		s.references = make(map[uint64]bool)
	} else {
		s.usage++
	}
}

// GetCompileId retrieves the unique compile ID associated with the Sector instance.
func (s *Sector) GetCompileId() uint64 {
	return s.compileId
}

// GetUsage retrieves the current usage count for the Sector instance.
func (s *Sector) GetUsage() int {
	return s.usage
}

// Add registers the given ID in the sector's references map by setting its value to true.
func (s *Sector) Add(id uint64) {
	s.references[id] = true
}

// Has checks if the given `id` exists in the `references` map and returns true if it does, otherwise false.
func (s *Sector) Has(id uint64) bool {
	_, ok := s.references[id]
	return ok
}

// IsVisible determines if a range [x1, x2] is not occluded and visible based on the provided identifier id.
func (s *Sector) IsVisible(x1 float64, x2 float64, id uint64) bool {
	if s.LastCompileId != id {
		s.VisibleSpans = s.VisibleSpans[:0]
		s.LastCompileId = id
	}
	for _, span := range s.VisibleSpans {
		// Se lo span da testare è interamente contenuto in uno span fuso, è occluso.
		if x1 >= span[0] && x2 <= span[1] {
			return false
		}
	}
	return true
}

// AddSpan merges a new span defined by x1 and x2 into the VisibleSpans of the Sector, ensuring proper ordering and overlap handling.
func (s *Sector) AddSpan(x1 float64, x2 float64) {
	var merged [][2]float64
	inserted := false

	for _, span := range s.VisibleSpans {
		if inserted {
			merged = append(merged, span)
			continue
		}

		if x2 < span[0] {
			// Inserimento a sinistra (mantiene l'ordinamento)
			merged = append(merged, [2]float64{x1, x2})
			merged = append(merged, span)
			inserted = true
		} else if x1 > span[1] {
			// Nessuna sovrapposizione
			merged = append(merged, span)
		} else {
			// Sovrapposizione: fusione dei limiti
			if span[0] < x1 {
				x1 = span[0]
			}
			if span[1] > x2 {
				x2 = span[1]
			}
		}
	}

	if !inserted {
		merged = append(merged, [2]float64{x1, x2})
	}

	s.VisibleSpans = merged
}

// LocateSector identifies the Sector containing the point (px, py) by traversing convex polygons linked via Segments.
func (s *Sector) LocateSector(px, py float64) *Sector {
	curr := s
	const maxSteps = 16 // Safeguard per loop infiniti da approssimazioni floating-point
	for step := 0; step < maxSteps; step++ {
		inside := true
		for _, seg := range curr.Segments {
			// Assumendo che < 0 indichi il semispazio "esterno" all'edge
			if mathematic.PointSideF(px, py, seg.Start.X, seg.Start.Y, seg.End.X, seg.End.Y) < 0 {
				if seg.Sector == nil {
					// Hit boundary esterno della mesh
					return nil
				}
				// Transizione: il punto è oltre questo segmento, saltiamo al vicino
				curr = seg.Sector
				inside = false
				break
			}
		}
		// Se il punto non è risultato all'esterno di nessun segmento,
		// per definizione è all'interno del poligono convesso corrente.
		if inside {
			return curr
		}
	}
	// Limite di walk superato (possibile ping-pong tra settori per edge-cases FP)

	//TODO
	//Quando questa funzione restituisce nil,
	//il chiamante saprà che la ricerca locale è fallita (punto teletrasportato troppo lontano, o uscito dalla mappa).
	//In quel ramo if eseguira la query globale contro l'AABB tree per riagganciare la referenza corretta con un costo logaritmico.
	return nil
}

// Print serializes the Sector into a JSON string, optionally indented, including its segments, floor, and ceiling data.
func (s *Sector) Print(indent bool) string {
	type printerSegment struct {
		Start XY
		End   XY
		Ref   string
		Kind  int
		Tag   string
	}
	type printerSector struct {
		ModelId  uint16
		Id       string
		Floor    float64
		Ceil     float64
		Segments []*printerSegment
	}

	p := printerSector{ModelId: s.ModelId, Id: s.Id, Floor: s.FloorY, Ceil: s.CeilY}
	for _, z := range s.Segments {
		ps := &printerSegment{Start: z.Start, End: z.End, Ref: z.Ref, Kind: z.Kind, Tag: z.Tag}
		p.Segments = append(p.Segments, ps)
	}
	if indent {
		d, _ := json.MarshalIndent(p, "", "  ")
		return string(d)
	}
	d, _ := json.Marshal(p)
	return string(d)
}

// GetAABB returns the axis-aligned bounding box (AABB) associated with the Sector instance.
func (s *Sector) GetAABB() *physics.AABB {
	return s.aabb
}

// ComputeAABB calculates and updates the axis-aligned bounding box (AABB) for the sector based on its segments.
func (s *Sector) ComputeAABB() {
	if len(s.Segments) == 0 {
		s.aabb = physics.NewAABB(0, 0, 0, 0, 0, 0)
		return
	}

	minX, minY := math.MaxFloat64, math.MaxFloat64
	maxX, maxY := -math.MaxFloat64, -math.MaxFloat64

	for _, seg := range s.Segments {
		if seg.Start.X < minX {
			minX = seg.Start.X
		}
		if seg.Start.X > maxX {
			maxX = seg.Start.X
		}
		if seg.Start.Y < minY {
			minY = seg.Start.Y
		}
		if seg.Start.Y > maxY {
			maxY = seg.Start.Y
		}

		if seg.End.X < minX {
			minX = seg.End.X
		}
		if seg.End.X > maxX {
			maxX = seg.End.X
		}
		if seg.End.Y < minY {
			minY = seg.End.Y
		}
		if seg.End.Y > maxY {
			maxY = seg.End.Y
		}
	}

	// Assumendo 2D o altezza gestita separatamente; Z può riflettere floor/ceiling se necessario
	s.aabb = physics.NewAABB(minX, minY, 0, maxX, maxY, 0)
}

// ClipVelocity adjusts the velocity vector (velX, velY) for collision detection and resolution within the current sector.
func (s *Sector) ClipVelocity(viewX float64, viewY float64, pX float64, pY float64, velX float64, velY float64, headPos float64, kneePos float64) (float64, float64) {
	const maxIter = 3

	// Micro-loop per predizione collisioni multiple
	for iter := 0; iter < maxIter; iter++ {
		hit := false

		for _, seg := range s.Segments {
			start := seg.Start
			end := seg.End

			if mathematic.IntersectLineSegmentsF(viewX, viewY, pX, pY, start.X, start.Y, end.X, end.Y) {
				holeLow := 9e9
				holeHigh := -9e9
				if seg.Sector != nil {
					holeLow = mathematic.MaxF(s.FloorY, seg.Sector.FloorY)
					holeHigh = mathematic.MinF(s.CeilY, seg.Sector.CeilY)
				}

				if holeHigh < headPos || holeLow > kneePos {
					xd := end.X - start.X
					yd := end.Y - start.Y
					lenSq := xd*xd + yd*yd

					if lenSq > 0 {
						dot := velX*xd + velY*yd
						velX = (xd * dot) / lenSq
						velY = (yd * dot) / lenSq

						invLen := 1.0 / math.Sqrt(lenSq)
						nx := -yd * invLen
						ny := xd * invLen

						epsilon := 0.005
						velX += nx * epsilon
						velY += ny * epsilon
					}
					hit = true
					break // Vettore modificato, ri-valuta contro la geometria
				}
			}
		}
		if !hit {
			break // Traiettoria stabilizzata
		}
	}
	return velX, velY
}
