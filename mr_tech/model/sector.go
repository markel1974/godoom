package model

import (
	"encoding/json"

	"github.com/markel1974/godoom/mr_tech/mathematic"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// Sector represents a 3D environment component, defined by its boundaries, textures, lighting, and associated segments.
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
	LastCompileId uint64
}

// NewSector initializes and returns a new Sector instance with the given modelId, id, and segments.
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

// Reference resets or updates the Sector's state based on the provided compileId, managing usage count and references.
func (s *Sector) Reference(compileId uint64) {
	if compileId != s.compileId {
		s.compileId = compileId
		s.usage = 0
		s.references = make(map[uint64]bool)
	} else {
		s.usage++
	}
}

// GetCompileId returns the current compile ID associated with the Sector instance.
func (s *Sector) GetCompileId() uint64 {
	return s.compileId
}

// GetUsage returns the current usage count of the Sector.
func (s *Sector) GetUsage() int {
	return s.usage
}

// Add marks the specified ID as referenced in the sector's internal references map.
func (s *Sector) Add(id uint64) {
	s.references[id] = true
}

// Has checks whether the given id exists in the Sector's references map and returns true if it is found, false otherwise.
func (s *Sector) Has(id uint64) bool {
	_, ok := s.references[id]
	return ok
}

// IsVisible determines if the range [x1, x2] is visible, updating the visibility spans if the compile ID has changed.
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

// AddSpan inserisce un nuovo segmento di occlusione (es. muro solido disegnato)
// fondendolo con eventuali span adiacenti o sovrapposti in tempo reale.
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

// LocateSector esegue un topological walk partendo dal settore 's'.
// Ritorna il settore contenente (px, py), oppure nil se il punto è fuori dalla mesh
// o se si innesca un ciclo (precisione FP), caso in cui il chiamante deve usare l'AABB tree.
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

// Print converts the Sector instance into a JSON-formatted string, optionally indented for readability.
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
