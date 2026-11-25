package builder

// ShadowRigidCache manages a cache of shadowRigidEntry objects with lifecycle control and periodic sweeping logic.
// It includes a pool of reusable entries to minimize allocations and a map for efficient retrieval by unique identifiers.
// lastSweepTick tracks the frame of the last cleanup operation to determine when to Sweep stale entries.
type ShadowRigidCache struct {
	slots         map[uint64]*ShadowRigidEntry
	free          []*ShadowRigidEntry
	lastSweepTick uint64
}

// Retrieve retrieves or creates a shadowRigidEntry for the given entity ID and updates its last seen frame.
func (c *ShadowRigidCache) Retrieve(entID uint64, frame uint64) *ShadowRigidEntry {
	if c.slots == nil {
		c.slots = make(map[uint64]*ShadowRigidEntry, 128)
	}
	if entry, ok := c.slots[entID]; ok {
		entry.lastSeenFrame = frame
		return entry
	}
	var entry *ShadowRigidEntry
	if n := len(c.free); n > 0 {
		entry = c.free[n-1]
		c.free = c.free[:n-1]
	} else {
		entry = &ShadowRigidEntry{}
	}
	entry.entID = entID
	entry.inUse = true
	entry.seenFrame = 0
	entry.lastSeenFrame = frame
	entry.matValid = false
	entry.faces = nil
	entry.facesVer = 0
	entry.facesLen = 0
	entry.rm = nil
	entry.hash = 0
	c.slots[entID] = entry
	return entry
}

// Sweep removes stale entries from the cache if they have not been accessed for more than 256 frames.
func (c *ShadowRigidCache) Sweep(frame uint64) {
	if len(c.slots) == 0 || frame-c.lastSweepTick < 256 {
		return
	}
	c.lastSweepTick = frame
	for id, entry := range c.slots {
		if frame-entry.lastSeenFrame > 256 {
			entry.inUse = false
			entry.faces = nil
			entry.rm = nil
			delete(c.slots, id)
			c.free = append(c.free, entry)
		}
	}
}
