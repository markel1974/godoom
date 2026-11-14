package aabb

// ContactPoint represents a single contact vertex and its penetration depth in world space.
type ContactPoint struct {
	X, Y, Z     float64
	Penetration float64
}

// Set updates all coordinates and the penetration depth of this contact point in-place.
func (cp *ContactPoint) Set(x, y, z, penetration float64) {
	cp.X = x
	cp.Y = y
	cp.Z = z
	cp.Penetration = penetration
}
