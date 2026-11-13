package rigid_body

import "math"

// CombineMode specifies how two material properties are combined during contact resolution.
type CombineMode uint8

// CombineMode values define how two material coefficients are merged.
const (
	// CombineAverage computes the arithmetic average: (a + b) * 0.5.
	CombineAverage CombineMode = iota
	// CombineMin takes the minimum value: min(a, b).
	CombineMin
	// CombineMultiply multiplies the values: a * b (for restitution) or sqrt(a * b) (for friction).
	CombineMultiply
	// CombineMax takes the maximum value: max(a, b).
	CombineMax
)

// PhysicsMaterial defines surface response properties for contacts, including bounce, friction, and combine behavior.
type PhysicsMaterial struct {
	restitution        float64
	friction           float64
	restitutionCombine CombineMode
	frictionCombine    CombineMode
	groundFriction     float64
	groundDamping      float64
}

// NewPhysicsMaterial creates a physics material with clamped restitution, nonnegative friction, combine modes, and dt damping.
func NewPhysicsMaterial(restitution, friction float64, restCombine, frictionCombine CombineMode, dt float64) *PhysicsMaterial {
	if restitution < 0.0 {
		restitution = 0.0
	} else if restitution > 1.0 {
		restitution = 1.0
	}
	if friction < 0.0 {
		friction = 0.0
	}
	return &PhysicsMaterial{
		groundFriction:     friction,
		groundDamping:      math.Pow(friction, dt),
		restitution:        restitution,
		friction:           friction,
		restitutionCombine: restCombine,
		frictionCombine:    frictionCombine,
	}
}

// SetDt updates ground damping using the given time step.
func (m1 *PhysicsMaterial) SetDt(dt float64) {
	m1.groundDamping = math.Pow(m1.groundFriction, dt)
}

// GetGroundDamping returns the precomputed damping factor applied while the material is on the ground.
func (m1 *PhysicsMaterial) GetGroundDamping() float64 {
	return m1.groundDamping
}

// CombineRestitution returns the combined restitution with m2 using the higher-priority combine mode.
func (m1 *PhysicsMaterial) CombineRestitution(m2 *PhysicsMaterial) float64 {
	mode := m1.restitutionCombine
	if m2.restitutionCombine > mode {
		mode = m2.restitutionCombine
	}
	switch mode {
	case CombineMin:
		return math.Min(m1.restitution, m2.restitution)
	case CombineMultiply:
		return m1.restitution * m2.restitution
	case CombineMax:
		return math.Max(m1.restitution, m2.restitution)
	case CombineAverage:
		fallthrough
	default:
		return (m1.restitution + m2.restitution) * 0.5
	}
}

// CombineFriction returns the combined friction value for this material and m2.
// It uses the higher-priority friction combine mode of the two materials.
func (m1 *PhysicsMaterial) CombineFriction(m2 *PhysicsMaterial) float64 {
	mode := m1.frictionCombine
	if m2.frictionCombine > mode {
		mode = m2.frictionCombine
	}
	switch mode {
	case CombineMin:
		return math.Min(m1.friction, m2.friction)
	case CombineMultiply:
		return math.Sqrt(m1.friction * m2.friction)
	case CombineMax:
		return math.Max(m1.friction, m2.friction)
	case CombineAverage:
		fallthrough
	default:
		return (m1.friction + m2.friction) * 0.5
	}
}
