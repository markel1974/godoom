package draws

import (
	"unsafe"
)

// DrawBatch is a structure to hold batched rendering data such as index counts and memory pointers.
// It is used to optimize rendering by grouping draw calls with similar states together.
type DrawBatch struct {
	mc            []int32
	mi            []uintptr
	depthWrite    int
	polygonOffset int
	hasCutout     int
	cullMode      int
	len           int
	mcPtr         *int32
	miPtr         *unsafe.Pointer
}

// NewDrawBatch creates and initializes a new DrawBatch with default starting size for internal slices.
func NewDrawBatch() *DrawBatch {
	const startSize = 4096
	rb := &DrawBatch{
		mc:  nil,
		mi:  nil,
		len: 0,
	}
	rb.rebuild(startSize)
	return rb
}

// Reset resets the DrawBatch, clearing its length and setting depthWrite, polygonOffset, cutout, and cull mode.
func (rb *DrawBatch) Reset(depthWrite int, polygonOffset int, hasCutout int, cullMode int) {
	rb.len = 0
	rb.depthWrite = depthWrite
	rb.polygonOffset = polygonOffset
	rb.hasCutout = hasCutout
	rb.cullMode = cullMode
}

// Add appends a new draw command to the DrawBatch with specified index count and EBO byte offset.
func (rb *DrawBatch) Add(indexCount int32, byteOffset int32) {
	if rb.len >= len(rb.mc) {
		rb.grow()
	}
	rb.mc[rb.len] = indexCount
	rb.mi[rb.len] = uintptr(byteOffset)
	rb.len++
}

// rebuild recreates and resizes the internal slices mc and mi to the specified length, ensuring a minimum size of 1.
func (rb *DrawBatch) rebuild(newLen int) {
	if newLen < 1 {
		newLen = 1
	}
	rb.mc = make([]int32, newLen)
	rb.mi = make([]uintptr, newLen)
	rb.mcPtr = &rb.mc[0]
	rb.miPtr = (*unsafe.Pointer)(unsafe.Pointer(&rb.mi[0]))
}

// grow increases the capacity of the DrawBatch's internal buffers to accommodate additional draw commands.
func (rb *DrawBatch) grow() {
	mc, mi := rb.mc, rb.mi
	newLen := rb.len * 2
	rb.rebuild(newLen)
	copy(rb.mc, mc)
	copy(rb.mi, mi)
}
