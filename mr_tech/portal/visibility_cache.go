package portal

import "github.com/markel1974/godoom/mr_tech/model"

// Span represents an interval with a start point (X1) and an end point (X2) on a 1-dimensional axis.
type Span struct {
	X1, X2 float64
}

// VisibilityCache stores calculated visibility spans for sectors to optimize rendering and minimize redundant calculations.
// It uses a pre-allocated buffer to reduce garbage collection during span merges.
type VisibilityCache struct {
	cache map[*model.Sector][]Span
	temp  []Span // Buffer di swap pre-allocato per le fusioni (zero GC)
}

// NewVisibilityCache initializes and returns a pointer to a new VisibilityCache with pre-allocated internal structures.
func NewVisibilityCache() *VisibilityCache {
	return &VisibilityCache{
		cache: make(map[*model.Sector][]Span, 1024), // Warm-up immediato della mappa
		temp:  make([]Span, 0, 128),
	}
}

// Clear removes all entries from the cache without deallocating underlying slice memory, preserving allocated capacity.
func (v *VisibilityCache) Clear() {
	// Zero-allocation clear: manteniamo le chiavi e la memoria fisica (capacity) delle slice
	for k := range v.cache {
		v.cache[k] = v.cache[k][:0]
	}
}

// Get retrieves the spans associated with the given sector and a boolean indicating their existence in the cache.
func (v *VisibilityCache) Get(s *model.Sector) ([]Span, bool) {
	spans, ok := v.cache[s]
	if ok && len(spans) == 0 {
		return nil, false // Gestisce logicamente gli span azzerati dal Clear()
	}
	return spans, ok
}

// Add merges the given range [x1, x2] into the cached spans for the specified sector, avoiding overlap and redundancies.
func (v *VisibilityCache) Add(s *model.Sector, x1 float64, x2 float64) {
	spans := v.cache[s]
	v.temp = v.temp[:0] // Reset del buffer di swap (costo: 1 ciclo di clock)
	inserted := false

	for _, span := range spans {
		if inserted {
			v.temp = append(v.temp, span)
			continue
		}
		if x2 < span.X1 {
			v.temp = append(v.temp, Span{X1: x1, X2: x2}, span)
			inserted = true
		} else if x1 > span.X2 {
			v.temp = append(v.temp, span)
		} else {
			if span.X1 < x1 {
				x1 = span.X1
			}
			if span.X2 > x2 {
				x2 = span.X2
			}
		}
	}
	if !inserted {
		v.temp = append(v.temp, Span{X1: x1, X2: x2})
	}

	// Sostituzione in-place: copia fisica dei dati senza allocazioni heap
	if cap(spans) < len(v.temp) {
		spans = make([]Span, len(v.temp), len(v.temp)*2+4)
	} else {
		spans = spans[:len(v.temp)]
	}
	copy(spans, v.temp)
	v.cache[s] = spans
}

// IsVisible checks if the range [x1, x2] is not entirely covered by any spans in the specified sector. Returns true if visible.
func (v *VisibilityCache) IsVisible(s *model.Sector, x1 float64, x2 float64) bool {
	spans, ok := v.Get(s)
	if !ok {
		return true
	}
	for _, span := range spans {
		if x1 >= span.X1 && x2 <= span.X2 {
			return false
		}
	}
	return true
}
