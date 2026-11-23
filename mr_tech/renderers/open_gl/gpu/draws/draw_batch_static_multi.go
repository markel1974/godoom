package draws

import (
	"unsafe"

	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
)

// DrawBatchStaticMulti groups compatible static indexed draw ranges for a single multi-draw call.
type DrawBatchStaticMulti struct {
	vao            uint32
	depthWrite     int
	polygonOffset  int
	cullMode       int
	hasCutout      bool
	mc             []int32
	mi             []uintptr
	len            int32
	mcPtr          *int32
	miPtr          *unsafe.Pointer
	firstOffsetPtr unsafe.Pointer
	lastEndIndex   int32
}

// NewDrawBatchStaticMulti creates a static multi-draw batch with preallocated draw count and index offset buffers.
func NewDrawBatchStaticMulti() *DrawBatchStaticMulti {
	const initCap = 64
	b := &DrawBatchStaticMulti{
		mc: make([]int32, initCap),
		mi: make([]uintptr, initCap),
	}
	b.mcPtr = &b.mc[0]
	b.miPtr = (*unsafe.Pointer)(unsafe.Pointer(&b.mi[0]))
	return b
}

// Reset prepares the batch for a new VAO and render state, clearing accumulated draw ranges.
func (b *DrawBatchStaticMulti) Reset(vao uint32, depthWrite, polygonOffset, cullMode int, hasCutout bool) {
	b.vao = vao
	b.depthWrite = depthWrite
	b.polygonOffset = polygonOffset
	b.cullMode = cullMode
	b.hasCutout = hasCutout
	b.len = 0
	b.lastEndIndex = -1
	b.firstOffsetPtr = nil
}

// AppendRange adds an indexed draw range to the batch, merging it with the previous range when contiguous.
func (b *DrawBatchStaticMulti) AppendRange(ctx api.IContext, firstIndex, indexCount int32) {
	if b.len > 0 && b.lastEndIndex == firstIndex {
		b.mc[b.len-1] += indexCount
		b.lastEndIndex += indexCount
		return
	}
	if int(b.len) >= len(b.mc) {
		newLen := max(len(b.mc)*2, 64)
		newMC := make([]int32, newLen)
		copy(newMC, b.mc[:b.len])
		newMI := make([]uintptr, newLen)
		copy(newMI, b.mi[:b.len])
		b.mc = newMC
		b.mi = newMI
		b.mcPtr = &b.mc[0]
		b.miPtr = (*unsafe.Pointer)(unsafe.Pointer(&b.mi[0]))
	}
	if b.len == 0 {
		b.firstOffsetPtr = ctx.PtrOffset(int(firstIndex) * 4)
	}
	b.mc[b.len] = indexCount
	b.mi[b.len] = uintptr(firstIndex) * 4
	b.len++
	b.lastEndIndex = firstIndex + indexCount
}

// Draw renders the batch with DrawElements for one range or MultiDrawElements for multiple ranges.
func (b *DrawBatchStaticMulti) Draw(ctx api.IContext) {
	if b.len == 1 {
		ctx.DrawElements(api.TRIANGLES, b.mc[0], api.UNSIGNED_INT, b.firstOffsetPtr)
		return
	}
	ctx.MultiDrawElements(api.TRIANGLES, b.mcPtr, api.UNSIGNED_INT, b.miPtr, b.len)
}
