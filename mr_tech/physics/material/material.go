package material

import "math"

// CombineMode defines how physical properties (restitution, friction) of two contacting surfaces are combined.
type CombineMode uint8

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

// PhysicsMaterial defines the surface physical characteristics for rigid body collisions and friction.
type PhysicsMaterial struct {
	Restitution        float64
	Friction           float64
	RestitutionCombine CombineMode
	FrictionCombine    CombineMode
}

// NewPhysicsMaterial creates a new PhysicsMaterial with clamped restitution [0, 1] and positive friction.
func NewPhysicsMaterial(restitution, friction float64, restCombine, frictCombine CombineMode) PhysicsMaterial {
	if restitution < 0.0 {
		restitution = 0.0
	} else if restitution > 1.0 {
		restitution = 1.0
	}
	if friction < 0.0 {
		friction = 0.0
	}
	return PhysicsMaterial{
		Restitution:        restitution,
		Friction:           friction,
		RestitutionCombine: restCombine,
		FrictionCombine:    frictCombine,
	}
}

// DefaultPhysicsMaterial returns standard non-bouncy solid surface properties (concrete-like).
func DefaultPhysicsMaterial() PhysicsMaterial {
	return PhysicsMaterial{
		Restitution:        0.0,
		Friction:           0.5,
		RestitutionCombine: CombineAverage,
		FrictionCombine:    CombineAverage,
	}
}

// CombineRestitution combines the restitution coefficients of two materials according to their combine modes.
// When combine modes differ, the higher-priority mode (Max > Multiply > Min > Average) dominates.
func CombineRestitution(m1, m2 PhysicsMaterial) float64 {
	mode := m1.RestitutionCombine
	if m2.RestitutionCombine > mode {
		mode = m2.RestitutionCombine
	}
	switch mode {
	case CombineMin:
		return math.Min(m1.Restitution, m2.Restitution)
	case CombineMultiply:
		return m1.Restitution * m2.Restitution
	case CombineMax:
		return math.Max(m1.Restitution, m2.Restitution)
	case CombineAverage:
		fallthrough
	default:
		return (m1.Restitution + m2.Restitution) * 0.5
	}
}

// CombineFriction combines the friction coefficients of two materials according to their combine modes.
// When combine modes differ, the higher-priority mode (Max > Multiply > Min > Average) dominates.
func CombineFriction(m1, m2 PhysicsMaterial) float64 {
	mode := m1.FrictionCombine
	if m2.FrictionCombine > mode {
		mode = m2.FrictionCombine
	}
	switch mode {
	case CombineMin:
		return math.Min(m1.Friction, m2.Friction)
	case CombineMultiply:
		return math.Sqrt(m1.Friction * m2.Friction)
	case CombineMax:
		return math.Max(m1.Friction, m2.Friction)
	case CombineAverage:
		fallthrough
	default:
		return (m1.Friction + m2.Friction) * 0.5
	}
}
