package config

// MD3 represents a multi-part 3D model used in Quake 3 (e.g. players).
type MD3 struct {
	Lower  *MD1
	Upper  *MD1
	Head   *MD1
	Weapon *MD1

	LegsActions  map[string][2]int
	TorsoActions map[string][2]int
}

// NewMD3 creates a new MD3 structure holding multiple parts.
func NewMD3(lower, upper, head, weapon *MD1) *MD3 {
	return &MD3{
		Lower:        lower,
		Upper:        upper,
		Head:         head,
		Weapon:       weapon,
		LegsActions:  make(map[string][2]int),
		TorsoActions: make(map[string][2]int),
	}
}
