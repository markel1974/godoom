package open_gl

import (
	"unsafe"

	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
)

// RenderBatch is a structure to hold batched rendering data such as index counts and memory pointers.
// It is used to optimize rendering by grouping draw calls with similar states together.
type RenderBatch struct {
	mc            []int32
	mi            []unsafe.Pointer
	depthWrite    int
	polygonOffset int
	len           int
}

// NewRenderBatch creates and initializes a new RenderBatch with default starting size for internal slices.
func NewRenderBatch() *RenderBatch {
	const startSize = 1024
	return &RenderBatch{
		mc:  make([]int32, startSize),
		mi:  make([]unsafe.Pointer, startSize),
		len: 0,
	}
}

// Reset resets the RenderBatch, clearing its length and setting depthWrite and polygonOffset to the provided values.
func (rb *RenderBatch) Reset(depthWrite int, polygonOffset int) {
	rb.len = 0
	rb.depthWrite = depthWrite
	rb.polygonOffset = polygonOffset
}

// Add appends a new draw command to the RenderBatch with specified index count and pointer.
func (rb *RenderBatch) Add(indexCount int32, pointer unsafe.Pointer) {
	if rb.len >= len(rb.mc) {
		mc := rb.mc
		mi := rb.mi
		newLen := rb.len * 2
		rb.mc = make([]int32, newLen)
		rb.mi = make([]unsafe.Pointer, newLen)
		copy(rb.mc, mc)
		copy(rb.mi, mi)
	}
	rb.mc[rb.len] = indexCount
	rb.mi[rb.len] = pointer
	rb.len++
}

// DrawCommandsRender organizes and manages render batches for executing draw commands in a graphics context.
type DrawCommandsRender struct {
	ctx        api.IContext
	batches    []*RenderBatch
	batchesLen int
	isAdditive bool
}

// NewDrawCommandsRender creates and initializes a new DrawCommandsRender with specified context and batch type.
func NewDrawCommandsRender(ctx api.IContext, isAdditive bool) *DrawCommandsRender {
	const startSize = 16
	dcr := &DrawCommandsRender{
		ctx:        ctx,
		batches:    make([]*RenderBatch, startSize),
		isAdditive: isAdditive,
	}
	for idx := range dcr.batches {
		dcr.batches[idx] = NewRenderBatch()
	}
	return dcr
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
				batches := w.batches
				newLen := len(w.batches) * 2
				w.batches = make([]*RenderBatch, newLen)
				copy(w.batches, batches)
				for i := len(batches); i < newLen; i++ {
					w.batches[i] = NewRenderBatch()
				}
			}
			currentBatch = w.batches[w.batchesLen]
			currentBatch.Reset(depthWrite, polygonOffset)
			w.batchesLen++
			currentDepthWrite = depthWrite
			currentPolygonOffset = polygonOffset
		}
		currentBatch.Add(cmd.indexCount, w.ctx.PtrOffset(int(cmd.firstIndex*4)))
	}
}

// Render executes the rendering process for all batched draw commands, managing depth writes and polygon offset states.
func (w *DrawCommandsRender) Render() {
	if w.batchesLen == 0 {
		return
	}

	currentDepthWrite := -1 // -1 means unknown
	currentPolygonOffset := -1

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

		w.ctx.MultiDrawElements(api.TRIANGLES, &b2.mc[0], api.UNSIGNED_INT, &b2.mi[0], int32(b2.len))
	}

	if currentDepthWrite != 1 && !w.isAdditive {
		w.ctx.DepthMask(true)
	}
	if currentPolygonOffset == 1 {
		w.ctx.Disable(api.POLYGON_OFFSET_FILL)
	}
}

// HasCommands returns true if there are render batches available to process; otherwise, it returns false.
func (w *DrawCommandsRender) HasCommands() bool {
	return w.batchesLen > 0
}
