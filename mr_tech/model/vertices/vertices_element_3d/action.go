package vertices_element_3d

// Action describes a named vertex animation frame range and whether playback should clamp at its end.
type Action struct {
	Name  string
	Start int
	End   int
	Clamp bool
}

// NewAction creates a Action with the given name and frame interval, with clamping disabled.
func NewAction(name string, start int, end int) *Action {
	return &Action{
		Name:  name,
		Start: start,
		End:   end,
		Clamp: false,
	}
}
