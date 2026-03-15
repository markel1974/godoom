package portal

import "github.com/markel1974/godoom/mr_tech/model"

// Span represents a range in 1D space defined by two float64 values, X1 and X2, where X1 is the start and X2 is the end.
type Span struct {
	X1, X2 float64
}

// VisibilityCache is a structure used to track visible spans for each sector during the current frame.
type VisibilityCache struct {
	// Tracker degli span per settore per il frame corrente
	cache map[*model.Sector][]Span
}

// NewVisibilityCache initializes and returns a new VisibilityCache with an empty cache map.
func NewVisibilityCache() *VisibilityCache {
	return &VisibilityCache{
		cache: make(map[*model.Sector][]Span),
	}
}

// Clear removes all elements from the VisibilityCache, resetting its internal state.
func (v *VisibilityCache) Clear() {
	for k := range v.cache {
		delete(v.cache, k)
	}
}

// Get retrieves the Span list associated with the given Sector from the visibility cache and indicates if it exists.
func (v *VisibilityCache) Get(s *model.Sector) ([]Span, bool) {
	spans, ok := v.cache[s]
	return spans, ok
}

// Add appends a new Span defined by x1 and x2 to the cache for the given Sector.
func (v *VisibilityCache) Add(s *model.Sector, x1 float64, x2 float64) {
	var merged []Span
	inserted := false

	for _, span := range v.cache[s] {
		if inserted {
			merged = append(merged, span)
			continue
		}
		if x2 < span.X1 {
			// Nessuna sovrapposizione, prima dello span corrente
			merged = append(merged, Span{x1, x2}, span)
			inserted = true
		} else if x1 > span.X2 {
			// Nessuna sovrapposizione, dopo lo span corrente
			merged = append(merged, span)
		} else {
			// Sovrapposizione: espandi i limiti
			if span.X1 < x1 {
				x1 = span.X1
			}
			if span.X2 > x2 {
				x2 = span.X2
			}
		}
	}
	if !inserted {
		merged = append(merged, Span{x1, x2})
	}
	v.cache[s] = merged
}

// IsVisible checks if the specified interval [x1, x2] of a sector is fully visible based on the cached spans.
func (v *VisibilityCache) IsVisible(s *model.Sector, x1 float64, x2 float64) bool {
	spans, ok := v.Get(s)
	if !ok {
		return true
	}
	for _, span := range spans {
		// Se il nuovo intervallo è completamente contenuto in uno esistente, non è visibile
		if x1 >= span.X1 && x2 <= span.X2 {
			return false
		}
	}
	return true
}
