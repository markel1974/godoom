package collisions

import "github.com/markel1974/godoom/mr_tech/physics/aabb"

// PendingDynamicContact stores a buffered dynamic contact result for later symmetric application.
type PendingDynamicContact struct {
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

// NewPendingDynamicContact creates an empty pending dynamic contact.
func NewPendingDynamicContact() *PendingDynamicContact {
	return &PendingDynamicContact{}
}

// PendingDynamicContacts stores reusable pending dynamic contact records for collision processing.
type PendingDynamicContacts struct {
	init      int
	container []*PendingDynamicContact
	len       int
}

// NewPendingDynamicContacts creates a reusable pending dynamic contact collection with the given initial capacity.
func NewPendingDynamicContacts(init int) *PendingDynamicContacts {
	p := &PendingDynamicContacts{
		init:      init,
		container: make([]*PendingDynamicContact, init),
		len:       0,
	}
	for i := 0; i < len(p.container); i++ {
		p.container[i] = NewPendingDynamicContact()
	}
	return p
}

// Reset clears the pending dynamic contact count while retaining allocated storage.
func (pdc *PendingDynamicContacts) Reset() {
	pdc.len = 0
}

// Len returns the number of pending dynamic contacts.
func (pdc *PendingDynamicContacts) Len() int {
	return pdc.len
}

// At returns the pending dynamic contact stored at idx.
func (pdc *PendingDynamicContacts) At(idx int) *PendingDynamicContact {
	return pdc.container[idx]
}

// Current returns the pending dynamic contact at the current write position, growing storage if needed.
func (pdc *PendingDynamicContacts) Current() *PendingDynamicContact {
	if pdc.len >= len(pdc.container) {
		n := make([]*PendingDynamicContact, nextGrowCap(len(pdc.container), pdc.init))
		copy(n, pdc.container)
		pdc.container = n
	}
	return pdc.container[pdc.len]
}

// Next advances the pending contact count after the current entry has been populated.
func (pdc *PendingDynamicContacts) Next() {
	pdc.len++
}

// Sort orders pending dynamic contacts by partner ID, then by partner entity ID.
func (pdc *PendingDynamicContacts) Sort() {
	if pdc.len <= 1 {
		return
	}
	for i := 1; i < pdc.len; i++ {
		cur := pdc.container[i]
		curID := cur.partner.id
		curEntID := cur.partner.entity.GetId()
		j := i - 1
		for j >= 0 {
			p := pdc.container[j].partner
			if p.id < curID || (p.id == curID && p.entity.GetId() <= curEntID) {
				break
			}
			pdc.container[j+1] = pdc.container[j]
			j--
		}
		pdc.container[j+1] = cur
	}
}
