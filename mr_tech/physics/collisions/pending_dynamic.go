package collisions

import "github.com/markel1974/godoom/mr_tech/physics/aabb"

// pendingDynamicContact stores the immutable narrow-phase contact result evaluated for a dynamic pair
// during parallel read-only acquisition before being merged deterministically in partner-ID order.
type pendingDynamicContact struct {
	partner      *Cage
	rFace        IFace
	recipFace    IFace
	normalX      float64
	normalY      float64
	normalZ      float64
	penetration  float64
	rMaxZ        float64
	selfFloor    bool
	partnerFloor bool
	wasSleeping  bool
	hit          bool
	pointCount   int
	points       [aabb.MaxCandidatePoints][4]float64
}

// sortPendingDynamicContacts sorts pending[0:n] in ascending order of partner cage ID (tie-breaking on entity ID)
// so Gauss-Seidel PGS receives a deterministic contact order independent of worker scheduling.
func sortPendingDynamicContacts(pending []pendingDynamicContact) {
	for i := 1; i < len(pending); i++ {
		cur := pending[i]
		curID := cur.partner.id
		curEntID := cur.partner.entity.GetId()
		j := i - 1
		for j >= 0 {
			p := pending[j].partner
			if p.id < curID || (p.id == curID && p.entity.GetId() <= curEntID) {
				break
			}
			pending[j+1] = pending[j]
			j--
		}
		pending[j+1] = cur
	}
}
