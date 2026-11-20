package rigid_body

import (
	"math"
	"testing"
)

func TestMaterialCombineSymmetry(t *testing.T) {
	modes := []CombineMode{CombineAverage, CombineMin, CombineMultiply, CombineMax}
	values := []float64{0.0, 0.2, 0.5, 0.8, 1.0}

	for _, mode1 := range modes {
		for _, mode2 := range modes {
			for _, r1 := range values {
				for _, r2 := range values {
					m1 := NewPhysicsMaterial(r1, r1*0.8, mode1, mode1, 0.016)
					m2 := NewPhysicsMaterial(r2, r2*0.8, mode2, mode2, 0.016)

					// Commutativity: m1.CombineRestitution(m2) must equal m2.CombineRestitution(m1)
					rest12 := m1.CombineRestitution(m2)
					rest21 := m2.CombineRestitution(m1)
					if math.Abs(rest12-rest21) > 1e-12 {
						t.Fatalf("restitution combine not symmetric: m1=(%v, %v), m2=(%v, %v) -> 12=%v != 21=%v",
							r1, mode1, r2, mode2, rest12, rest21)
					}

					// Commutativity: m1.CombineFriction(m2) must equal m2.CombineFriction(m1)
					fric12 := m1.CombineFriction(m2)
					fric21 := m2.CombineFriction(m1)
					if math.Abs(fric12-fric21) > 1e-12 {
						t.Fatalf("friction combine not symmetric: m1=(%v, %v), m2=(%v, %v) -> 12=%v != 21=%v",
							r1, mode1, r2, mode2, fric12, fric21)
					}
				}
			}
		}
	}
}

func TestMaterialNilSafety(t *testing.T) {
	m := NewPhysicsMaterial(0.7, 0.5, CombineAverage, CombineAverage, 0.016)
	if m.CombineRestitution(nil) != 0.7 {
		t.Fatalf("expected 0.7 for m.CombineRestitution(nil)")
	}
	var nilMat *PhysicsMaterial
	if nilMat.CombineRestitution(m) != 0.7 {
		t.Fatalf("expected 0.7 for nil.CombineRestitution(m)")
	}
	if nilMat.CombineRestitution(nil) != 0.0 {
		t.Fatalf("expected 0.0 for nil.CombineRestitution(nil)")
	}
}
