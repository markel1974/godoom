package open_gl

import (
	"unsafe"

	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
)

// RenderBatch is a structure to hold batched rendering data such as index counts and memory pointers.
// It is used to optimize rendering by grouping draw calls with similar states together.
type RenderBatch struct {
	mc            []int32
	mi            []uintptr
	depthWrite    int
	polygonOffset int
	len           int
	mcPtr         *int32
	miPtr         *unsafe.Pointer
}

// NewRenderBatch creates and initializes a new RenderBatch with default starting size for internal slices.
func NewRenderBatch() *RenderBatch {
	const startSize = 4096
	rb := &RenderBatch{
		mc:  nil,
		mi:  nil,
		len: 0,
	}
	rb.rebuild(startSize)
	return rb
}

// Reset resets the RenderBatch, clearing its length and setting depthWrite and polygonOffset to the provided values.
func (rb *RenderBatch) Reset(depthWrite int, polygonOffset int) {
	rb.len = 0
	rb.depthWrite = depthWrite
	rb.polygonOffset = polygonOffset
}

// Add appends a new draw command to the RenderBatch with specified index count and EBO byte offset.
func (rb *RenderBatch) Add(indexCount int32, byteOffset int32) {
	if rb.len >= len(rb.mc) {
		rb.grow()
	}
	rb.mc[rb.len] = indexCount
	rb.mi[rb.len] = uintptr(byteOffset)
	rb.len++
}

// rebuild recreates and resizes the internal slices mc and mi to the specified length, ensuring a minimum size of 1.
func (rb *RenderBatch) rebuild(newLen int) {
	if newLen < 1 {
		newLen = 1
	}
	rb.mc = make([]int32, newLen)
	rb.mi = make([]uintptr, newLen)
	rb.mcPtr = &rb.mc[0]
	rb.miPtr = (*unsafe.Pointer)(unsafe.Pointer(&rb.mi[0]))
}

// grow increases the capacity of the RenderBatch's internal buffers to accommodate additional draw commands.
func (rb *RenderBatch) grow() {
	mc, mi := rb.mc, rb.mi
	newLen := rb.len * 2
	rb.rebuild(newLen)
	copy(rb.mc, mc)
	copy(rb.mi, mi)
}

// DrawCommandsRender organizes and manages render batches for executing draw commands in a graphics context.
type DrawCommandsRender struct {
	ctx        api.IContext
	batches    []*RenderBatch
	batchesLen int
	isAdditive bool
	name       string
}

// NewDrawCommandsRender creates and initializes a new DrawCommandsRender with specified context and batch type.
func NewDrawCommandsRender(ctx api.IContext, name string, isAdditive bool) *DrawCommandsRender {
	const startSize = 16
	dcr := &DrawCommandsRender{
		ctx:        ctx,
		name:       name,
		batches:    make([]*RenderBatch, startSize),
		isAdditive: isAdditive,
	}
	for idx := range dcr.batches {
		dcr.batches[idx] = NewRenderBatch()
	}
	return dcr
}

// HasCommands returns true if there are render batches available to process; otherwise, it returns false.
func (w *DrawCommandsRender) HasCommands() bool {
	return w.batchesLen > 0
}

// Prepare organizes and processes an array of DrawCommand instances into renderable batches for efficient rendering.
func (w *DrawCommandsRender) Prepare(dc []*DrawCommand) {
	w.batchesLen = 0
	if len(dc) == 0 {
		return
	}

	currentDepthWrite := -1
	currentPolygonOffset := -1

	var currentBatch *RenderBatch

	for _, cmd := range dc {
		if cmd.indexCount == 0 {
			continue
		}

		var depthWrite int
		var polygonOffset int

		if cmd.material != nil {
			if w.isAdditive {
				depthWrite = 0
			} else if cmd.material.GetDepthWrite() {
				depthWrite = 1
			} else {
				depthWrite = 0
			}

			if cmd.material.GetPolygonOffset() {
				polygonOffset = 1
			} else {
				polygonOffset = 0
			}
		} else {
			if w.isAdditive {
				depthWrite = 0
			} else {
				depthWrite = 1
			}
			polygonOffset = 0
		}

		if depthWrite != currentDepthWrite || polygonOffset != currentPolygonOffset || currentBatch == nil {
			if w.batchesLen >= len(w.batches) {
				w.grow()
			}
			currentBatch = w.batches[w.batchesLen]
			currentBatch.Reset(depthWrite, polygonOffset)
			w.batchesLen++
			currentDepthWrite = depthWrite
			currentPolygonOffset = polygonOffset
		}
		currentBatch.Add(cmd.indexCount, cmd.firstIndex*4)
	}
}

// Render executes the rendering process for all batched draw commands, managing depth writes and polygon offset states.
func (w *DrawCommandsRender) Render() {
	if w.batchesLen == 0 {
		return
	}

	currentDepthWrite := -1 // -1 means unknown
	currentPolygonOffset := -1

	//totalDraws := 0
	for i := 0; i < w.batchesLen; i++ {
		b2 := w.batches[i]
		if b2.depthWrite != currentDepthWrite {
			if b2.depthWrite == 1 {
				w.ctx.DepthMask(true)
			} else {
				w.ctx.DepthMask(false)
			}
			currentDepthWrite = b2.depthWrite
		}
		if b2.polygonOffset != currentPolygonOffset {
			if b2.polygonOffset == 1 {
				w.ctx.Enable(api.POLYGON_OFFSET_FILL)
				w.ctx.PolygonOffset(-1.0, -1.0)
			} else {
				w.ctx.Disable(api.POLYGON_OFFSET_FILL)
			}
			currentPolygonOffset = b2.polygonOffset
		}
		//totalDraws += int(b2.len)
		w.ctx.MultiDrawElements(api.TRIANGLES, b2.mcPtr, api.UNSIGNED_INT, b2.miPtr, int32(b2.len))
	}
	if currentDepthWrite != 1 && !w.isAdditive {
		w.ctx.DepthMask(true)
	}
	if currentPolygonOffset == 1 {
		w.ctx.Disable(api.POLYGON_OFFSET_FILL)
	}

	//fmt.Printf("[%s] batches=%d draws=%d additive=%v\n", w.name, w.batchesLen, totalDraws, w.isAdditive, )
}

func (w *DrawCommandsRender) grow() {
	batches := w.batches
	newLen := len(w.batches) * 2
	w.batches = make([]*RenderBatch, newLen)
	copy(w.batches, batches)
	for i := len(batches); i < newLen; i++ {
		w.batches[i] = NewRenderBatch()
	}
}
