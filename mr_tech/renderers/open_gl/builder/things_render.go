package builder

import (
	"github.com/markel1974/godoom/mr_tech/physics/rigid_body"
)

const defaultThingCacheCap = 4096

// ThingCacheRef records a cached thing reference and generation used to detect render cache changes.
type ThingCacheRef struct {
	id  uint64
	gen uint64
}

// ThingsRender manages preallocated per-thing render cache slots and fixed reference arrays with zero hot-path allocations.
type ThingsRender struct {
	entries       []*ThingRenderCache
	freeSlots     []int
	freeLen       int
	nextSlot      int
	thingFrameID  uint64
	prevThingRefs []ThingCacheRef
	prevLen       int
	currThingRefs []ThingCacheRef
	currLen       int
}

// NewThingsRender creates a ThingsRender with preallocated cache slots and frame reference arrays.
func NewThingsRender() *ThingsRender {
	return NewThingsRenderWithCap(defaultThingCacheCap)
}

// NewThingsRenderWithCap creates a ThingsRender preallocated for initCap simultaneous cached things.
func NewThingsRenderWithCap(initCap int) *ThingsRender {
	if initCap < 16 {
		initCap = 16
	}
	entries := make([]*ThingRenderCache, initCap)
	for i := range entries {
		entries[i] = NewThingRenderCache()
	}
	return &ThingsRender{
		entries:       entries,
		freeSlots:     make([]int, initCap),
		freeLen:       0,
		nextSlot:      0,
		thingFrameID:  0,
		prevThingRefs: make([]ThingCacheRef, initCap),
		prevLen:       0,
		currThingRefs: make([]ThingCacheRef, initCap),
		currLen:       0,
	}
}

// Prepare starts a new thing render frame and resets the current reference count.
func (w *ThingsRender) Prepare() {
	w.thingFrameID++
	w.currLen = 0
}

// Retrieve returns the preallocated render cache slot bound to ent, assigning or recycling a slot if needed.
func (w *ThingsRender) Retrieve(ent *rigid_body.Entity) *ThingRenderCache {
	entID := ent.GetId()
	slot := ent.GetRenderMarker()
	if slot >= 0 && slot < w.nextSlot {
		entry := w.entries[slot]
		if entry.inUse && entry.entID == entID {
			entry.lastSeenFrame = w.thingFrameID
			return entry
		}
	}

	if w.freeLen > 0 {
		w.freeLen--
		slot = w.freeSlots[w.freeLen]
	} else {
		if w.nextSlot >= len(w.entries) {
			w.growEntries()
		}
		slot = w.nextSlot
		w.nextSlot++
	}

	entry := w.entries[slot]
	entry.entID = entID
	entry.inUse = true
	entry.valid = false
	entry.pFaces = nil
	entry.pNextFaces = nil
	entry.lastSeenFrame = w.thingFrameID
	entry.compileFrame = 0
	entry.committedFrame = 0
	entry.committedBaseIdx = -1
	entry.fv.DeepReset()
	ent.SetRenderMarker(slot)
	return entry
}

// PeekIfCurrent returns ent's cached entry without modifying tracking state if already visited in the current thing frame.
func (w *ThingsRender) PeekIfCurrent(ent *rigid_body.Entity) *ThingRenderCache {
	if ent == nil {
		return nil
	}
	entID := ent.GetId()
	slot := ent.GetRenderMarker()
	if slot >= 0 && slot < w.nextSlot {
		entry := w.entries[slot]
		if entry.inUse && entry.entID == entID && entry.lastSeenFrame == w.thingFrameID {
			return entry
		}
	}
	return nil
}

// AddRef records the entry's entity ID and current cache generation in the preallocated frame reference array.
func (w *ThingsRender) AddRef(entry *ThingRenderCache) {
	if w.currLen >= len(w.currThingRefs) {
		w.growRefs()
	}
	w.currThingRefs[w.currLen] = ThingCacheRef{id: entry.entID, gen: entry.gen}
	w.currLen++
}

// Commit finalizes the frame's thing references, swaps the double-buffered reference arrays on changes,
// and periodically recycles slots for entities no longer seen recently.
func (w *ThingsRender) Commit() bool {
	if w.thingFrameID&255 == 0 {
		for slot := 0; slot < w.nextSlot; slot++ {
			entry := w.entries[slot]
			if entry.inUse && w.thingFrameID-entry.lastSeenFrame > 256 {
				entry.inUse = false
				entry.valid = false
				entry.pFaces = nil
				entry.pNextFaces = nil
				entry.fv.DeepReset()
				w.freeSlots[w.freeLen] = slot
				w.freeLen++
			}
		}
	}

	thingsChanged := w.currLen != w.prevLen
	if !thingsChanged {
		for i := 0; i < w.currLen; i++ {
			if w.currThingRefs[i] != w.prevThingRefs[i] {
				thingsChanged = true
				break
			}
		}
	}
	if thingsChanged {
		w.prevThingRefs, w.currThingRefs = w.currThingRefs, w.prevThingRefs
		w.prevLen = w.currLen
	}
	return thingsChanged
}

// growEntries expands the cache entry pool and free-slot stack if capacity is exceeded.
func (w *ThingsRender) growEntries() {
	oldCap := len(w.entries)
	newCap := max(oldCap*2, 16)
	newEntries := make([]*ThingRenderCache, newCap)
	copy(newEntries, w.entries)
	for i := oldCap; i < newCap; i++ {
		newEntries[i] = NewThingRenderCache()
	}
	w.entries = newEntries

	newFree := make([]int, newCap)
	copy(newFree, w.freeSlots[:w.freeLen])
	w.freeSlots = newFree
}

// growRefs expands both reference tracking buffers together if active things per frame exceed initial capacity.
func (w *ThingsRender) growRefs() {
	oldCap := len(w.currThingRefs)
	newCap := max(oldCap*2, 16)
	newCurr := make([]ThingCacheRef, newCap)
	copy(newCurr, w.currThingRefs[:w.currLen])
	w.currThingRefs = newCurr

	newPrev := make([]ThingCacheRef, newCap)
	copy(newPrev, w.prevThingRefs[:w.prevLen])
	w.prevThingRefs = newPrev
}
