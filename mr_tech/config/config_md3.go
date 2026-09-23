package config

// MD3 represents a multi-part 3D model used in Quake 3 (e.g. players).
type MD3 struct {
	Lower *MD1
	Upper *MD1
	Head  *MD1
	// Weapon *MD1 // Optional, not implemented yet

	// Animation config
	LegsStartFrame int
	LegsNumFrames  int
	LegsLoopFrames int
	LegsFPS        int

	TorsoStartFrame int
	TorsoNumFrames  int
	TorsoLoopFrames int
	TorsoFPS        int
}

// NewMD3 creates a new MD3 structure holding multiple parts.
func NewMD3(lower, upper, head *MD1) *MD3 {
	return &MD3{
		Lower: lower,
		Upper: upper,
		Head:  head,
	}
}
