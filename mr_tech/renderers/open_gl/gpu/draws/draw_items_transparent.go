package draws

import (
	"cmp"
	"slices"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/gpu/buffers"
)

// DrawItemsTransparent manages a reusable collection of transparent draw items for sorting and rendering.
type DrawItemsTransparent struct {
	data           []*DrawItemTransparent
	transparentLen int
	prevLen        int
	dynPool        buffers.PtrPool[DrawItemTransparent]
}

// NewDrawItemsTransparent creates an empty DrawItemsTransparent collection with initial storage for n items.
func NewDrawItemsTransparent(n int) *DrawItemsTransparent {
	if n <= 0 {
		n = 16
	}
	return &DrawItemsTransparent{
		data:           make([]*DrawItemTransparent, n),
		transparentLen: 0,
	}
}

// Len returns the number of active transparent draw items.
func (w *DrawItemsTransparent) Len() int {
	return w.transparentLen
}

// At returns the active transparent draw item at index i.
func (w *DrawItemsTransparent) At(i int) *DrawItemTransparent {
	return w.data[i]
}

// Reset clears the transparent draw item list without releasing its backing storage.
func (w *DrawItemsTransparent) Reset() {
	w.transparentLen = 0
}

// Begin starts a fresh transparent item collection pass while recording prior length for stale reference trimming.
func (w *DrawItemsTransparent) Begin() {
	w.prevLen = w.transparentLen
	w.transparentLen = 0
	w.dynPool.Reset()
}

// End trims stale pointer references past the current length and sorts active items back-to-front.
func (w *DrawItemsTransparent) End() {
	w.Trim(w.prevLen)
	w.prevLen = 0
	w.SortTransparentDrawItems()
}

// Clear releases all active transparent draw item references and resets pooled dynamic state.
func (w *DrawItemsTransparent) Clear() {
	for i := 0; i < w.transparentLen; i++ {
		w.data[i] = nil
	}
	w.transparentLen = 0
	w.prevLen = 0
	w.dynPool.Reset()
}

// Trim clears stale transparent draw item references from the current length up to prevTransLen.
func (w *DrawItemsTransparent) Trim(prevTransLen int) {
	limit := min(prevTransLen, len(w.data))
	for i := w.transparentLen; i < limit; i++ {
		w.data[i] = nil
	}
}

// Add appends a transparent draw item, growing the internal storage when needed.
func (w *DrawItemsTransparent) Add(it *DrawItemTransparent) {
	if w.transparentLen >= len(w.data) {
		newLen := max(len(w.data)*2, 32)
		grown := make([]*DrawItemTransparent, newLen)
		copy(grown, w.data)
		w.data = grown
	}
	w.data[w.transparentLen] = it
	w.transparentLen++
}

// AddStatic queues prebuilt static transparent submesh items, updating camera-relative squared distances.
func (w *DrawItemsTransparent) AddStatic(src *buffers.PtrPool[DrawItemTransparent], camX, camY, camZ float32) {
	for i := 0; i < src.Len; i++ {
		it := src.Items[i]
		sub := it.Sub
		dx := sub.CentroidX - camX
		dy := sub.CentroidY - camY
		dz := sub.CentroidZ - camZ
		it.DistSq = dx*dx + dy*dy + dz*dz
		w.Add(it)
	}
}

// AddInstanced acquires a dynamic transparent item for a single rigid instance submesh and queues it for sorting.
func (w *DrawItemsTransparent) AddInstanced(mesh *buffers.ResidentMesh, sub *buffers.ResidentSubMesh, instanceVBO uint32, baseByteOffset int32, cx, cy, cz, camX, camY, camZ float32) {
	dx := cx - camX
	dy := cy - camY
	dz := cz - camZ
	it := w.dynPool.Acquire()
	it.Kind = TransparentInstanced
	it.DistSq = dx*dx + dy*dy + dz*dz
	it.SortOrder = sub.SortOrder
	it.CullMode = sub.CullMode
	it.HasCutout = sub.HasCutout
	it.PolygonOffset = sub.PolygonOffset
	it.FirstIndex = 0
	it.IndexCount = 0
	it.Mesh = mesh
	it.Sub = sub
	it.InstanceVBO = instanceVBO
	it.BaseByteOffset = baseByteOffset
	it.InstanceCount = 1
	w.Add(it)
}

// AddCommands acquires dynamic transparent items for streaming translucent DrawCommands and queues them for sorting.
func (w *DrawItemsTransparent) AddCommands(cmds []*DrawCommand, camX, camY, camZ float32) {
	for _, cmd := range cmds {
		if cmd.GetIndexCount() == 0 {
			continue
		}
		mat := cmd.GetMaterial()
		cx, cy, cz := cmd.GetCentroid()
		dx := cx - camX
		dy := cy - camY
		dz := cz - camZ
		sortOrder := 0
		cullMode := config.CullFront
		hasCutout := false
		polyOff := false
		if mat != nil {
			sortOrder = mat.GetSort()
			cullMode = mat.GetCullMode()
			hasCutout = mat.HasCutout()
			polyOff = mat.GetPolygonOffset()
		}
		it := w.dynPool.Acquire()
		it.Kind = TransparentStreaming
		it.DistSq = dx*dx + dy*dy + dz*dz
		it.SortOrder = sortOrder
		it.CullMode = cullMode
		it.HasCutout = hasCutout
		it.PolygonOffset = polyOff
		it.FirstIndex = cmd.GetFirstIndex()
		it.IndexCount = cmd.GetIndexCount()
		it.Mesh = nil
		it.Sub = nil
		it.InstanceVBO = 0
		it.BaseByteOffset = 0
		it.InstanceCount = 0
		w.Add(it)
	}
}

// SortTransparentDrawItems stably orders transparent draw items back-to-front, using sort order for near-equal distances.
func (w *DrawItemsTransparent) SortTransparentDrawItems() {
	n := w.transparentLen
	if n <= 1 {
		return
	}
	slices.SortStableFunc(w.data[:n], func(a, b *DrawItemTransparent) int {
		diff := a.DistSq - b.DistSq
		if diff > 1e-2 {
			return -1
		}
		if diff < -1e-2 {
			return 1
		}
		return cmp.Compare(a.SortOrder, b.SortOrder)
	})
}
