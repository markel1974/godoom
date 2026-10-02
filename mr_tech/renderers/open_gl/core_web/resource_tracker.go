//go:build js && wasm

package core_web

// ResourceTracker manages a collection of reusable resources and tracks free slots within the collection.
type ResourceTracker struct {
	items []js.Value
	free  []uint32
}

// NewResourceTracker creates and returns a new instance of resourceTracker with a reserved null value in its items list.
func NewResourceTracker() resourceTracker {
	return resourceTracker{items: []js.Value{js.Null()}} // 0 is reserved/null
}

// Add inserts the given `js.Value` into the tracker and returns its index. Reuses an existing free index if available.
func (r *ResourceTracker) Add(val js.Value) uint32 {
	if len(r.free) > 0 {
		idx := r.free[len(r.free)-1]
		r.free = r.free[:len(r.free)-1]
		r.items[idx] = val
		return idx
	}
	r.items = append(r.items, val)
	return uint32(len(r.items) - 1)
}

// Get retrieves the `js.Value` at the specified index `idx` in the `ResourceTracker`. Returns `js.Null()` if the index is invalid.
func (r *ResourceTracker) Get(idx uint32) js.Value {
	if idx == 0 || int(idx) >= len(r.items) {
		return js.Null()
	}
	return r.items[idx]
}

// Remove removes the resource at the specified index and marks the index as free for reuse.
func (r *ResourceTracker) Remove(idx uint32) {
	if idx > 0 && int(idx) < len(r.items) {
		r.items[idx] = js.Null()
		r.free = append(r.free, idx)
	}
}
