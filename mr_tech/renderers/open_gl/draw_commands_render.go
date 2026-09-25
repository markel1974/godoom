package open_gl

import (
	"unsafe"

	"github.com/go-gl/gl/v3.3-core/gl"
)

type RenderBatch struct {
	mc            []int32
	mi            []unsafe.Pointer
	depthWrite    int
	polygonOffset int
}

// DrawCommandsRender manages batched GPU drawing commands, storing index counts and memory offsets for rendering.
type DrawCommandsRender struct {
	batches         []RenderBatch
	batchesAdditive []RenderBatch
}

// NewDrawCommandsRender initializes and returns a new instance of DrawCommandsRender with preallocated internal arrays.
func NewDrawCommandsRender() *DrawCommandsRender {
	return &DrawCommandsRender{
		batches:         make([]RenderBatch, 0, 16),
		batchesAdditive: make([]RenderBatch, 0, 16),
	}
}

func (w *DrawCommandsRender) buildBatches(dc []*DrawCommand, isAdditive bool) []RenderBatch {
	var batches []RenderBatch
	if len(dc) == 0 {
		return batches
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
			if isAdditive {
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
			if isAdditive {
				depthWrite = 0
			} else {
				depthWrite = 1
			}
			polygonOffset = 0
		}

		if depthWrite != currentDepthWrite || polygonOffset != currentPolygonOffset || currentBatch == nil {
			batches = append(batches, RenderBatch{
				mc:            make([]int32, 0, 1024),
				mi:            make([]unsafe.Pointer, 0, 1024),
				depthWrite:    depthWrite,
				polygonOffset: polygonOffset,
			})
			currentBatch = &batches[len(batches)-1]
			currentDepthWrite = depthWrite
			currentPolygonOffset = polygonOffset
		}

		currentBatch.mc = append(currentBatch.mc, cmd.indexCount)
		currentBatch.mi = append(currentBatch.mi, gl.PtrOffset(int(cmd.firstIndex*4)))
	}
	return batches
}

// Prepare initializes the draw command buffers and sets up data for rendering based on the provided draw commands.
func (w *DrawCommandsRender) Prepare(dc []*DrawCommand) {
	w.batches = w.buildBatches(dc, false)
	w.batchesAdditive = w.buildBatches(dc, true)
}

// Render executes the rendering process for the prepared draw commands using multi-draw elements in OpenGL.
func (w *DrawCommandsRender) Render() {
	w.renderInternal(false)
}

// RenderAdditive executes the rendering process for the additive draw commands without forcing depth write.
func (w *DrawCommandsRender) RenderAdditive() {
	w.renderInternal(true)
}

func (w *DrawCommandsRender) renderInternal(isAdditive bool) {
	batches := w.batches
	if isAdditive {
		batches = w.batchesAdditive
	}

	if len(batches) == 0 {
		return
	}

	currentDepthWrite := -1 // -1 means unknown
	currentPolygonOffset := -1

	for i := 0; i < len(batches); i++ {
		b := &batches[i]

		if b.depthWrite != currentDepthWrite {
			if b.depthWrite == 1 {
				gl.DepthMask(true)
			} else {
				gl.DepthMask(false)
			}
			currentDepthWrite = b.depthWrite
		}

		if b.polygonOffset != currentPolygonOffset {
			if b.polygonOffset == 1 {
				gl.Enable(gl.POLYGON_OFFSET_FILL)
				gl.PolygonOffset(-1.0, -1.0)
			} else {
				gl.Disable(gl.POLYGON_OFFSET_FILL)
			}
			currentPolygonOffset = b.polygonOffset
		}

		gl.MultiDrawElements(gl.TRIANGLES, &b.mc[0], gl.UNSIGNED_INT, &b.mi[0], int32(len(b.mc)))
	}

	if currentDepthWrite != 1 && !isAdditive {
		gl.DepthMask(true)
	}
	if currentPolygonOffset == 1 {
		gl.Disable(gl.POLYGON_OFFSET_FILL)
	}
}
