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
	dc           []*DrawCommand
	multiCounts  []int32
	multiIndices []unsafe.Pointer
}

// NewDrawCommandsRender creates and returns a new instance of DrawCommandsRender with pre-allocated capacity.
func NewDrawCommandsRender() *DrawCommandsRender {
	return &DrawCommandsRender{
		multiCounts:  make([]int32, 0, 128),
		multiIndices: make([]unsafe.Pointer, 0, 128),
	}
}

// Prepare initializes the DrawCommandsRender instance with a list of DrawCommand objects and sorts them by material properties.
func (w *DrawCommandsRender) Prepare(dc []*DrawCommand) {
	w.dc = dc
	if len(w.dc) == 0 {
		return
	}
	// Sorting in-place tramite interfaccia (zero reflection)
	sort.Sort(ByMaterial(w.dc))
}

// Render processes and renders a list of DrawCommand objects into multi-draw elements for efficient batching.
func (w *DrawCommandsRender) Render() {
	if len(w.dc) == 0 {
		return
	}

	// Re-slicing logico: capacity mantenuta, length a 0. Zero GC.
	w.multiCounts = w.multiCounts[:0]
	w.multiIndices = w.multiIndices[:0]

	var lastTex, lastNorm, lastEmiss uint32 = math.MaxUint32, math.MaxUint32, math.MaxUint32

	for _, cmd := range w.dc {
		if cmd.indexCount <= 0 {
			continue
		}

		if cmd.texId != lastTex || cmd.normTexId != lastNorm || cmd.emissiveTexId != lastEmiss {
			// Flush del materiale precedente
			if len(w.multiCounts) > 0 {
				gl.MultiDrawElements(gl.TRIANGLES, &w.multiCounts[0], gl.UNSIGNED_INT, &w.multiIndices[0], int32(len(w.multiCounts)))
				// Reset per il prossimo materiale
				w.multiCounts = w.multiCounts[:0]
				w.multiIndices = w.multiIndices[:0]
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

		// Memorizza il numero di indici e l'offset in byte all'interno dell'EBO (uint32 = 4 byte)
		w.multiCounts = append(w.multiCounts, cmd.indexCount)
		w.multiIndices = append(w.multiIndices, gl.PtrOffset(int(cmd.firstIndex*4)))
	}

	// Flush della coda finale
	if len(w.multiCounts) > 0 {
		gl.MultiDrawElements(gl.TRIANGLES, &w.multiCounts[0], gl.UNSIGNED_INT, &w.multiIndices[0], int32(len(w.multiCounts)))
	}
}
