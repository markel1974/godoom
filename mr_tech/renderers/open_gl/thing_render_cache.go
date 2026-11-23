package open_gl

import "github.com/markel1974/godoom/mr_tech/model/volumes"

// ThingCacheRef records a cached thing reference and generation used to detect render cache changes.
type ThingCacheRef struct {
	id  uint64
	gen uint64
}

// ThingRenderCache stores cached render state for reusing generated thing frame vertices across frames.
type ThingRenderCache struct {
	fv            *FrameVertices
	pFaces        *volumes.Faces
	pNextFaces    *volumes.Faces
	facesVer      uint64
	nextVer       uint64
	facesLen      int
	nextLen       int
	lerp          float32
	yaw           float32
	renderMode    float32
	oX, oY, oZ    float32
	gen           uint64
	lastSeenFrame uint64
	valid         bool
}

// NewThingRenderCache creates a render cache with CPU-only staging frame vertices for thing rendering.
func NewThingRenderCache() *ThingRenderCache {
	return &ThingRenderCache{fv: NewStagingFrameVertices(128)}
}

// GetFrameVertices returns the cached frame vertices for this render cache.
func (tr *ThingRenderCache) GetFrameVertices() *FrameVertices {
	return tr.fv
}

// Equal reports whether the cache is valid and matches the supplied render state without needing animation rebuild.
func (tr *ThingRenderCache) Equal(pFaces *volumes.Faces, pNextFaces *volumes.Faces, facesVer uint64, nextVer uint64, facesLen int, nextLen int, lerp float32, yaw float32, renderModeF float32, oX float32, oY float32, oZ float32) bool {
	cacheHit := tr.valid &&
		tr.pFaces == pFaces &&
		tr.pNextFaces == pNextFaces &&
		tr.facesVer == facesVer &&
		tr.nextVer == nextVer &&
		tr.facesLen == facesLen &&
		tr.nextLen == nextLen &&
		tr.lerp == lerp &&
		tr.yaw == yaw &&
		tr.renderMode == renderModeF &&
		tr.oX == oX && tr.oY == oY && tr.oZ == oZ &&
		!tr.fv.NeedsAnimRebuild()
	return cacheHit
}

// Assign updates the cached render inputs and marks the cache valid.
// Assign increments the generation counter to reflect the new cached state.
func (tr *ThingRenderCache) Assign(pFaces *volumes.Faces, pNextFaces *volumes.Faces, facesVer uint64, nextVer uint64, facesLen int, nextLen int, lerp float32, yaw float32, renderModeF float32, oX float32, oY float32, oZ float32) {
	tr.pFaces = pFaces
	tr.pNextFaces = pNextFaces
	tr.facesVer = facesVer
	tr.nextVer = nextVer
	tr.facesLen = facesLen
	tr.nextLen = nextLen
	tr.lerp = lerp
	tr.yaw = yaw
	tr.renderMode = renderModeF
	tr.oX, tr.oY, tr.oZ = oX, oY, oZ
	tr.valid = true
	tr.gen++
}

// ThingsRender manages per-thing render caches and tracks frame references to detect changes between frames.
type ThingsRender struct {
	thingCache    map[uint64]*ThingRenderCache
	thingFrameID  uint64
	prevThingRefs []ThingCacheRef
	currThingRefs []ThingCacheRef
}

// NewThingsRender creates a ThingsRender with initialized caches and frame reference buffers.
func NewThingsRender() *ThingsRender {
	return &ThingsRender{
		thingCache:    make(map[uint64]*ThingRenderCache, 256), //TODO NON DEVE ESSERE UNA MAPPA MA UN ARRAY (MAI MAPPE NEGLI HOT PATH)
		thingFrameID:  0,
		prevThingRefs: make([]ThingCacheRef, 0), // TODO COMPLETAMENTE SBAGLIATO GLI ARRAY NON POSSONO ESSERE COSTRUITI AD OGNI AZIONE (MAI COSTRUITI NEGLI HOT PATH)
		currThingRefs: make([]ThingCacheRef, 0), // TODO COMPLETAMENTE SBAGLIATO GLI ARRAY NON POSSONO ESSERE COSTRUITI AD OGNI AZIONE (MAI COSTRUITI NEGLI HOT PATH)
	}
}

// Prepare starts a new thing render frame and clears the current reference list.
func (w *ThingsRender) Prepare() {
	w.thingFrameID++
	w.currThingRefs = w.currThingRefs[:0]
}

// Retrieve returns the render cache for entID, creating it if needed, and marks it seen in the current frame.
func (w *ThingsRender) Retrieve(entID uint64) *ThingRenderCache {
	entry := w.thingCache[entID]
	if entry == nil {
		entry = NewThingRenderCache()
		w.thingCache[entID] = entry
	}
	entry.lastSeenFrame = w.thingFrameID
	return entry
}

// AddRef records the entity's current cache generation for change detection during commit.
func (w *ThingsRender) AddRef(entID uint64, entry *ThingRenderCache) {
	w.currThingRefs = append(w.currThingRefs, ThingCacheRef{id: entID, gen: entry.gen})
}

// Commit finalizes the current thing references and reports whether they differ from the previous frame.
// Commit also periodically removes stale cached thing entries that have not been seen recently.
func (w *ThingsRender) Commit() bool {
	if w.thingFrameID&255 == 0 {
		for id, entry := range w.thingCache {
			if w.thingFrameID-entry.lastSeenFrame > 256 {
				delete(w.thingCache, id)
			}
		}
	}

	thingsChanged := len(w.currThingRefs) != len(w.prevThingRefs)
	if !thingsChanged {
		for i, ref := range w.currThingRefs {
			if w.prevThingRefs[i] != ref {
				thingsChanged = true
				break
			}
		}
	}
	if thingsChanged {
		if cap(w.prevThingRefs) < len(w.currThingRefs) {
			w.prevThingRefs = make([]ThingCacheRef, len(w.currThingRefs))
		} else {
			w.prevThingRefs = w.prevThingRefs[:len(w.currThingRefs)]
		}
		copy(w.prevThingRefs, w.currThingRefs)
	}
	return thingsChanged
}
