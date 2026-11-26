package builder

import (
	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/gpu/buffers"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/gpu/draws"
)

// ShadowInstRange identifies a contiguous set of shadow instances within a resident mesh buffer.
type ShadowInstRange struct {
	mesh           *buffers.ResidentMesh
	baseByteOffset int32
	instanceCount  int32
}

// ShadowTileBuilder builds draw command data for rendering shadow tiles.
type ShadowTileBuilder struct {
	dcRender      *draws.DrawCommandsRender
	streamDC      *draws.DrawCommands
	staticItems   buffers.PtrPool[draws.DrawItemStatic]
	instItems     buffers.PtrPool[draws.DrawItemInstanced]
	instRanges2   []ShadowInstRange
	instRangesLen int
}

// NewShadowTileBuilder creates a builder for collecting and preparing opaque shadow tile draw commands.
func NewShadowTileBuilder(ctx api.IContext) *ShadowTileBuilder {
	return &ShadowTileBuilder{
		dcRender:    draws.NewDrawCommandsRender(ctx, "shadow_tile", false),
		streamDC:    draws.NewDrawCommands(config.BlendModeOpaque, 128),
		instRanges2: make([]ShadowInstRange, 32),
	}
}

// Reset clears accumulated shadow tile draw state and returns internal pools and command buffers to empty reuse state.
func (tb *ShadowTileBuilder) Reset() {
	tb.staticItems.Reset()
	tb.instItems.Reset()
	tb.instRangesLen = 0
	tb.streamDC.DeepReset()
}

// Add appends an instanced mesh range with its base byte offset and instance count for shadow rendering.
func (tb *ShadowTileBuilder) Add(mesh *buffers.ResidentMesh, baseByteOffset int32, instanceCount int32) {
	if tb.instRangesLen >= len(tb.instRanges2) {
		newLen := max(len(tb.instRanges2)*2, 32)
		z := make([]ShadowInstRange, newLen)
		copy(z, tb.instRanges2)
		tb.instRanges2 = z
	}
	ptr := &tb.instRanges2[tb.instRangesLen]
	ptr.mesh = mesh
	ptr.baseByteOffset = baseByteOffset
	ptr.instanceCount = instanceCount

	tb.instRangesLen++
}

// Prepare builds shadow draw items for opaque submeshes and submits them to the draw command renderer.
func (tb *ShadowTileBuilder) Prepare(shadowInstVBO uint32) {
	for rIdx := range tb.instRangesLen {
		r := &tb.instRanges2[rIdx]
		for sIdx := 0; sIdx < r.mesh.SubMeshesLen; sIdx++ {
			sub := r.mesh.SubMeshes[sIdx]
			if sub.IndexCount == 0 || sub.BlendMode != config.BlendModeOpaque {
				continue
			}
			item := tb.instItems.Acquire()
			item.Mesh = r.mesh
			item.Sub = sub
			item.InstanceVBO = shadowInstVBO
			item.BaseByteOffset = r.baseByteOffset
			item.InstanceCount = r.instanceCount
		}
	}
	tb.dcRender.Prepare(tb.streamDC.GetDrawCommands())
	tb.dcRender.SetShadowResidentDraws(tb.staticItems.Items, tb.staticItems.Len, tb.instItems.Items, tb.instItems.Len)
}
