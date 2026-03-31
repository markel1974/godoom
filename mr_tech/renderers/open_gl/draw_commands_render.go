package open_gl

import (
	"unsafe"

	"github.com/go-gl/gl/v3.3-core/gl"
)

// DrawCommandsRender manages batched GPU drawing commands, storing index counts and memory offsets for rendering.
type DrawCommandsRender struct {
	mc    []int32
	mi    []unsafe.Pointer
	count int32
}

// NewDrawCommandsRender initializes and returns a new instance of DrawCommandsRender with preallocated internal arrays.
func NewDrawCommandsRender() *DrawCommandsRender {
	return &DrawCommandsRender{
		mc:    make([]int32, 1024),
		mi:    make([]unsafe.Pointer, 1024),
		count: 0,
	}
}

// Prepare initializes the draw command buffers and sets up data for rendering based on the provided draw commands.
func (w *DrawCommandsRender) Prepare(dc []*DrawCommand) {
	dcLen := len(dc)
	w.count = int32(dcLen)
	if dcLen == 0 {
		return
	}
	if dcLen > cap(w.mc) {
		w.mc = make([]int32, dcLen*2)
		w.mi = make([]unsafe.Pointer, dcLen*2)
	}
	for i, cmd := range dc {
		w.mc[i] = cmd.indexCount
		w.mi[i] = gl.PtrOffset(int(cmd.firstIndex * 4))
	}
}

// Render executes the rendering process for the prepared draw commands using multi-draw elements in OpenGL.
func (w *DrawCommandsRender) Render() {
	if w.count == 0 {
		return
	}
	gl.MultiDrawElements(gl.TRIANGLES, &w.mc[0], gl.UNSIGNED_INT, &w.mi[0], w.count)
}
