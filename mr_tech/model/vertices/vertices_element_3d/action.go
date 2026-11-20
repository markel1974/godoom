package vertices_element_3d

import (
	"strings"
)

// Action describes a named vertex animation frame range and whether playback should clamp at its end.
type Action struct {
	Name  string
	Start int
	End   int
	Clamp bool
}

// NewAction creates a Action with the given name and frame interval, with clamping disabled.
func NewAction(name string, start int, end int) *Action {
	v := &Action{
		Name:  name,
		Start: start,
		End:   end,
		Clamp: false,
	}
	//TODO MOVE IN GENERATORS!!!!!
	lower := strings.ToLower(strings.TrimSpace(name))
	if strings.Contains(lower, "death") || strings.Contains(lower, "die") || strings.Contains(lower, "dead") {
		v.Clamp = true
	}
	return v
}
