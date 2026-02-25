package portal

// Span represents a continuous range with start and end points X1 and X2.
type Span struct {
	X1, X2 float64
}

// SectorTracker manages a collection of spans to track visibility in a linear range.
type SectorTracker struct {
	spans []Span
}

// IsVisible checks if the range [x1, x2] is not completely occluded by any spans in the SectorTracker.
func (st *SectorTracker) IsVisible(x1 float64, x2 float64) bool {
	for _, s := range st.spans {
		if x1 >= s.X1 && x2 <= s.X2 {
			return false // Completamente occluso
		}
	}
	return true
}
