package builder

import "github.com/markel1974/godoom/mr_tech/physics/rigid_body"

const defaultShadowCacheCap = 128

// ShadowRigidCache manages O(1) slot-indexed per-entity shadow entries via Entity.GetShadowMarker
// with free-slot recycling and zero map lookups in hot paths.
type ShadowRigidCache struct {
	entries       []*ShadowRigidEntry
	freeSlots     []int
	freeLen       int
	nextSlot      int
	lastSweepTick uint64
}

func (c *ShadowRigidCache) initPool(initCap int) {
	if initCap < defaultShadowCacheCap {
		initCap = defaultShadowCacheCap
	}
	c.entries = make([]*ShadowRigidEntry, initCap)
	for i := range c.entries {
		c.entries[i] = &ShadowRigidEntry{}
	}
	c.freeSlots = make([]int, initCap)
}

// Retrieve returns the preallocated shadow entry bound to ent in O(1), assigning or recycling a slot if needed.
func (c *ShadowRigidCache) Retrieve(ent *rigid_body.Entity, frame uint64) *ShadowRigidEntry {
	if c.entries == nil {
		c.initPool(defaultShadowCacheCap)
	}
	entID := ent.GetId()
	slot := ent.GetShadowMarker()
	if slot >= 0 && slot < c.nextSlot {
		entry := c.entries[slot]
		if entry.inUse && entry.entID == entID {
			entry.lastSeenFrame = frame
			return entry
		}
	}

	if c.freeLen > 0 {
		c.freeLen--
		slot = c.freeSlots[c.freeLen]
	} else {
		if c.nextSlot >= len(c.entries) {
			c.grow()
		}
		slot = c.nextSlot
		c.nextSlot++
	}

	entry := c.entries[slot]
	entry.Reset(entID, frame)
	ent.SetShadowMarker(slot)
	return entry
}

// Sweep recycles stale entries not seen for more than 256 frames back onto the free stack.
func (c *ShadowRigidCache) Sweep(frame uint64) {
	if c.nextSlot == 0 || frame-c.lastSweepTick < 256 {
		return
	}
	c.lastSweepTick = frame
	for slot := 0; slot < c.nextSlot; slot++ {
		entry := c.entries[slot]
		if entry.inUse && frame-entry.lastSeenFrame > 256 {
			entry.Release()
			c.freeSlots[c.freeLen] = slot
			c.freeLen++
		}
	}
}

func (c *ShadowRigidCache) grow() {
	oldCap := len(c.entries)
	newCap := max(oldCap*2, defaultShadowCacheCap)
	newEntries := make([]*ShadowRigidEntry, newCap)
	copy(newEntries, c.entries)
	for i := oldCap; i < newCap; i++ {
		newEntries[i] = &ShadowRigidEntry{}
	}
	c.entries = newEntries

	newFree := make([]int, newCap)
	copy(newFree, c.freeSlots[:c.freeLen])
	c.freeSlots = newFree
}
