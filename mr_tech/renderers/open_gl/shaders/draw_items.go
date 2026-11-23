package shaders

import (
	"cmp"
	"slices"

	"github.com/markel1974/godoom/mr_tech/config"
)

// TransparentKind identifies how a transparent draw item is sourced and rendered.
type TransparentKind uint8

// TransparentKind values define how transparent geometry is rendered.
const (
	TransparentStreaming TransparentKind = iota
	TransparentStatic
	TransparentInstanced
)

// StaticDrawItem pairs a GPU-resident mesh with one of its submesh ranges for static rendering.
type StaticDrawItem struct {
	Mesh *ResidentMesh
	Sub  *ResidentSubMesh
}

// InstancedDrawItem describes an instanced draw command for a resident submesh and its instance buffer range.
type InstancedDrawItem struct {
	Mesh           *ResidentMesh
	Sub            *ResidentSubMesh
	InstanceVBO    uint32
	BaseByteOffset int32
	InstanceCount  int32
}

// TransparentDrawItem describes a queued transparent draw with sorting and render-state metadata.
type TransparentDrawItem struct {
	Kind           TransparentKind
	DistSq         float32
	SortOrder      int
	CullMode       int
	HasCutout      bool
	PolygonOffset  bool
	FirstIndex     int32
	IndexCount     int32
	Mesh           *ResidentMesh
	Sub            *ResidentSubMesh
	InstanceVBO    uint32
	BaseByteOffset int32
	InstanceCount  int32
}

// TransparentDrawItems manages a reusable collection of transparent draw items for sorting and rendering.
type TransparentDrawItems struct {
	data           []*TransparentDrawItem
	transparentLen int
	prevLen        int
	dynPool        PtrPool[TransparentDrawItem]
}

// NewTransparentDrawItems creates an empty TransparentDrawItems collection with initial storage for n items.
func NewTransparentDrawItems(n int) *TransparentDrawItems {
	if n <= 0 {
		n = 16
	}
	return &TransparentDrawItems{
		data:           make([]*TransparentDrawItem, n),
		transparentLen: 0,
	}
}

// Len returns the number of active transparent draw items.
func (w *TransparentDrawItems) Len() int {
	return w.transparentLen
}

// At returns the active transparent draw item at index i.
func (w *TransparentDrawItems) At(i int) *TransparentDrawItem {
	return w.data[i]
}

// Reset clears the transparent draw item list without releasing its backing storage.
func (w *TransparentDrawItems) Reset() {
	w.transparentLen = 0
}

// Begin starts a fresh transparent item collection pass while recording prior length for stale reference trimming.
func (w *TransparentDrawItems) Begin() {
	w.prevLen = w.transparentLen
	w.transparentLen = 0
	w.dynPool.Reset()
}

// End trims stale pointer references past the current length and sorts active items back-to-front.
func (w *TransparentDrawItems) End() {
	w.Trim(w.prevLen)
	w.prevLen = 0
	w.SortTransparentDrawItems()
}

// Clear releases all active transparent draw item references and resets pooled dynamic state.
func (w *TransparentDrawItems) Clear() {
	for i := 0; i < w.transparentLen; i++ {
		w.data[i] = nil
	}
	w.transparentLen = 0
	w.prevLen = 0
	w.dynPool.Reset()
}

// Trim clears stale transparent draw item references from the current length up to prevTransLen.
func (w *TransparentDrawItems) Trim(prevTransLen int) {
	limit := min(prevTransLen, len(w.data))
	for i := w.transparentLen; i < limit; i++ {
		w.data[i] = nil
	}
}

// Add appends a transparent draw item, growing the internal storage when needed.
func (w *TransparentDrawItems) Add(it *TransparentDrawItem) {
	if w.transparentLen >= len(w.data) {
		newLen := max(len(w.data)*2, 32)
		grown := make([]*TransparentDrawItem, newLen)
		copy(grown, w.data)
		w.data = grown
	}
	w.data[w.transparentLen] = it
	w.transparentLen++
}

// AddStatic queues prebuilt static transparent submesh items, updating camera-relative squared distances.
func (w *TransparentDrawItems) AddStatic(src *PtrPool[TransparentDrawItem], camX, camY, camZ float32) {
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
func (w *TransparentDrawItems) AddInstanced(mesh *ResidentMesh, sub *ResidentSubMesh, instanceVBO uint32, baseByteOffset int32, cx, cy, cz, camX, camY, camZ float32) {
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
func (w *TransparentDrawItems) AddCommands(cmds []*DrawCommand, camX, camY, camZ float32) {
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
func (w *TransparentDrawItems) SortTransparentDrawItems() {
	n := w.transparentLen
	if n <= 1 {
		return
	}
	slices.SortStableFunc(w.data[:n], func(a, b *TransparentDrawItem) int {
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
