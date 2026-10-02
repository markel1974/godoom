package config

// Model3D represents a multi-part 3D model used in Quake 3 (e.g. players).
type Model3D struct {
	Lower  *Model3DEntry
	Upper  *Model3DEntry
	Head   *Model3DEntry
	Weapon *Model3DEntry

	LegsActions  map[string][2]int
	TorsoActions map[string][2]int
}

// NewMD3 creates a new Model3D structure holding multiple parts.
func NewMD3(lower, upper, head, weapon *Model3DEntry) *Model3D {
	return &Model3D{
		Lower:        lower,
		Upper:        upper,
		Head:         head,
		Weapon:       weapon,
		LegsActions:  make(map[string][2]int),
		TorsoActions: make(map[string][2]int),
	}
}
