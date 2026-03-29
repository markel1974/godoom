package open_gl

import (
	"math"
	"sort"

	"github.com/go-gl/gl/v3.3-core/gl"
)

// ByMaterial is a slice of pointers to DrawCommand, sorted based on texture, normal texture, and emissive texture IDs.
type ByMaterial []*DrawCommand

// Len returns the number of elements in the ByMaterial collection.
func (a ByMaterial) Len() int { return len(a) }

// Swap swaps the elements with indexes i and j within the ByMaterial slice.
func (a ByMaterial) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

// Less compares two DrawCommand elements based on their texId, normTexId, and emissiveTexId in priority order.
func (a ByMaterial) Less(i, j int) bool {
	if a[i].texId != a[j].texId {
		return a[i].texId < a[j].texId
	}
	if a[i].normTexId != a[j].normTexId {
		return a[i].normTexId < a[j].normTexId
	}
	return a[i].emissiveTexId < a[j].emissiveTexId
}

// DrawCommandsRender manages batched rendering commands using arrays of starting indices and counts for each draw call.
type DrawCommandsRender struct {
	multiFirsts []int32
	multiCounts []int32
}

// NewDrawCommandsRender creates and returns a new instance of DrawCommandsRender with initialized slice containers.
func NewDrawCommandsRender() *DrawCommandsRender {
	return &DrawCommandsRender{}
}

// Render processes and renders a list of DrawCommand objects into multi-draw arrays for efficient batching.
func (w *DrawCommandsRender) Render(dc []*DrawCommand) {
	if len(dc) == 0 {
		return
	}

	// Sorting in-place tramite interfaccia (zero reflection)
	sort.Sort(ByMaterial(dc))

	counter := 0

	var lastTex, lastNorm, lastEmiss uint32 = math.MaxUint32, math.MaxUint32, math.MaxUint32

	for _, cmd := range dc {
		if cmd.vertexCount <= 0 {
			continue
		}

		if cmd.texId != lastTex || cmd.normTexId != lastNorm || cmd.emissiveTexId != lastEmiss {
			if counter > 0 {
				gl.MultiDrawArrays(gl.TRIANGLES, &w.multiFirsts[0], &w.multiCounts[0], int32(counter))
				counter = 0
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

		if counter >= len(w.multiFirsts) {
			w.Grow()
		}
		w.multiFirsts[counter] = cmd.firstVertex
		w.multiCounts[counter] = cmd.vertexCount
		counter++
	}

	if counter > 0 {
		gl.MultiDrawArrays(gl.TRIANGLES, &w.multiFirsts[0], &w.multiCounts[0], int32(counter))
	}
}

// Grow doubles the capacity of the internal slices or initializes them with a default size if empty.
func (w *DrawCommandsRender) Grow() {
	newSize := len(w.multiFirsts) * 2
	if newSize == 0 {
		newSize = 128
	}
	newMultiFirsts := make([]int32, newSize)
	copy(newMultiFirsts, w.multiFirsts)
	w.multiFirsts = newMultiFirsts

	newMultiCounts := make([]int32, newSize)
	copy(newMultiCounts, w.multiCounts)
	w.multiCounts = newMultiCounts
}
