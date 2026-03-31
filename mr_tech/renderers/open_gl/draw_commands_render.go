package open_gl

import (
	"math"
	"sort"
	"unsafe"

	"github.com/go-gl/gl/v3.3-core/gl"
)

// ByMaterial is a slice of pointers to DrawCommand, sorted based on texture, normal texture, and emissive texture IDs.
type ByMaterial []*DrawCommand

func (a ByMaterial) Len() int { return len(a) }

func (a ByMaterial) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

func (a ByMaterial) Less(i, j int) bool {
	if a[i].texId != a[j].texId {
		return a[i].texId < a[j].texId
	}
	if a[i].normTexId != a[j].normTexId {
		return a[i].normTexId < a[j].normTexId
	}
	return a[i].emissiveTexId < a[j].emissiveTexId
}

// DrawCommandsRender manages batched rendering commands using multi-draw elements for indexed geometry.
type DrawCommandsRender struct {
	dc []*DrawCommand
	mc []int32
	mi []unsafe.Pointer
}

// NewDrawCommandsRender creates and returns a new instance of DrawCommandsRender with pre-allocated capacity.
func NewDrawCommandsRender() *DrawCommandsRender {
	const startSize = 128
	return &DrawCommandsRender{
		mc: make([]int32, startSize),
		mi: make([]unsafe.Pointer, startSize),
		dc: make([]*DrawCommand, startSize),
	}
}

// Prepare initializes the DrawCommandsRender instance with a list of DrawCommand objects and sorts them by material properties.
func (w *DrawCommandsRender) Prepare(dc []*DrawCommand, sortRequired bool) {
	dcLen := len(dc)
	if dcLen >= cap(w.mc) {
		newSize := dcLen * 2
		w.mc = make([]int32, newSize)
		w.mi = make([]unsafe.Pointer, newSize)
		w.dc = make([]*DrawCommand, newSize)
	}
	w.dc = w.dc[:dcLen]
	copy(w.dc, dc)
	if sortRequired {
		sort.Sort(ByMaterial(w.dc))
	}
}

// Render processes and renders a list of DrawCommand objects into multi-draw elements for efficient batching.
func (w *DrawCommandsRender) Render() {
	if len(w.dc) == 0 {
		return
	}
	index := int32(0)
	var lastTex, lastNorm, lastEmiss uint32 = math.MaxUint32, math.MaxUint32, math.MaxUint32
	for _, cmd := range w.dc {
		if cmd.indexCount <= 0 {
			continue
		}
		if cmd.texId != lastTex || cmd.normTexId != lastNorm || cmd.emissiveTexId != lastEmiss {
			if index > 0 {
				gl.MultiDrawElements(gl.TRIANGLES, &w.mc[0], gl.UNSIGNED_INT, &w.mi[0], index)
				index = 0
			}
			if lastTex != cmd.texId {
				gl.ActiveTexture(gl.TEXTURE0)
				gl.BindTexture(gl.TEXTURE_2D, cmd.texId)
				lastTex = cmd.texId
			}
			if lastNorm != cmd.normTexId {
				gl.ActiveTexture(gl.TEXTURE1)
				gl.BindTexture(gl.TEXTURE_2D, cmd.normTexId)
				lastNorm = cmd.normTexId
			}
			if lastEmiss != cmd.emissiveTexId {
				gl.ActiveTexture(gl.TEXTURE5)
				gl.BindTexture(gl.TEXTURE_2D, cmd.emissiveTexId)
				lastEmiss = cmd.emissiveTexId
			}
		}
		w.mc[index] = cmd.indexCount
		w.mi[index] = gl.PtrOffset(int(cmd.firstIndex * 4))
		index++
	}
	if index > 0 {
		gl.MultiDrawElements(gl.TRIANGLES, &w.mc[0], gl.UNSIGNED_INT, &w.mi[0], index)
	}
}
