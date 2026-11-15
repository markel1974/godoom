package aabb

// ContactPoint represents a collision contact position and its penetration depth.
type ContactPoint struct {
	x, y, z     float64
	penetration float64
}

// Set updates the contact point coordinates and penetration depth.
func (cp *ContactPoint) Set(x, y, z, penetration float64) {
	cp.x = x
	cp.y = y
	cp.z = z
	cp.penetration = penetration
}

// SetPenetration sets the contact point's penetration depth.
func (cp *ContactPoint) SetPenetration(penetration float64) {
	cp.penetration = penetration
}

// Get returns the x, y, z coordinates and penetration depth.
func (cp *ContactPoint) Get() (float64, float64, float64, float64) {
	return cp.x, cp.y, cp.z, cp.penetration
}
