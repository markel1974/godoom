package builder

// shadowThingCache is a structure for caching ThingRenderCache objects and managing lifecycle through reuse or cleanup.
type shadowThingCache struct {
	slots         map[uint64]*ThingRenderCache
	free          []*ThingRenderCache
	lastSweepTick uint64
}

// retrieve retrieves or creates a ThingRenderCache for the specified entity ID, updating its last seen frame.
func (c *shadowThingCache) retrieve(entID uint64, frame uint64) *ThingRenderCache {
	if c.slots == nil {
		c.slots = make(map[uint64]*ThingRenderCache, 128)
	}
	if entry, ok := c.slots[entID]; ok {
		entry.lastSeenFrame = frame
		return entry
	}
	var entry *ThingRenderCache
	if n := len(c.free); n > 0 {
		entry = c.free[n-1]
		c.free = c.free[:n-1]
	} else {
		entry = NewThingRenderCache()
	}
	entry.entID = entID
	entry.inUse = true
	entry.valid = false
	entry.pFaces = nil
	entry.pNextFaces = nil
	entry.lastSeenFrame = frame
	entry.compileFrame = 0
	entry.committedFrame = 0
	entry.committedBaseIdx = -1
	entry.fv.DeepReset()
	c.slots[entID] = entry
	return entry
}

// sweep removes stale entries from the shadowThingCache based on frame ticks and reclaims unused memory.
func (c *shadowThingCache) sweep(frame uint64) {
	if len(c.slots) == 0 || frame-c.lastSweepTick < 256 {
		return
	}
	c.lastSweepTick = frame
	for id, entry := range c.slots {
		if frame-entry.lastSeenFrame > 256 {
			entry.inUse = false
			entry.valid = false
			entry.pFaces = nil
			entry.pNextFaces = nil
			entry.fv.DeepReset()
			delete(c.slots, id)
			c.free = append(c.free, entry)
		}
	}
}
