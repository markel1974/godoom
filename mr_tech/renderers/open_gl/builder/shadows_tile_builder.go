package builder

import (
	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/gpu/buffers"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/gpu/draws"
)

// ShadowInstRange describes a contiguous range of shadow instances within a resident mesh buffer.
type ShadowInstRange struct {
	mesh           *buffers.ResidentMesh
	baseByteOffset int32
	instanceCount  int32
}

// GetMesh returns the resident mesh assigned to the shadow instance range.
func (s *ShadowInstRange) GetMesh() *buffers.ResidentMesh {
	return s.mesh
}

// Assign sets the mesh, base byte offset, and instance count for this shadow instance range.
func (s *ShadowInstRange) Assign(mesh *buffers.ResidentMesh, baseByteOffset int32, instanceCount int32) {
	s.mesh = mesh
	s.baseByteOffset = baseByteOffset
	s.instanceCount = instanceCount
}

// ShadowTileBuilder builds draw command data for shadow tile rendering.
type ShadowTileBuilder struct {
	dcRender      *draws.DrawCommandsRender
	streamDC      *draws.DrawCommands
	staticItems   buffers.PtrPool[draws.DrawItemStatic]
	instItems     buffers.PtrPool[draws.DrawItemInstanced]
	instRanges    []ShadowInstRange
	instRangesLen int
}

// NewShadowTileBuilder creates a ShadowTileBuilder initialized for shadow tile draw command rendering.
func NewShadowTileBuilder(ctx api.IContext) *ShadowTileBuilder {
	return &ShadowTileBuilder{
		dcRender:   draws.NewDrawCommandsRender(ctx, "shadow_tile", false),
		streamDC:   draws.NewDrawCommands(config.BlendModeOpaque, 128),
		instRanges: make([]ShadowInstRange, 32),
	}
}

// Reset clears accumulated shadow tile state and draw commands while retaining allocated storage for reuse.
func (tb *ShadowTileBuilder) Reset() {
	tb.staticItems.Reset()
	tb.instItems.Reset()
	tb.instRangesLen = 0
	tb.streamDC.DeepReset()
}

// GetStreamDC returns the stream draw commands used to collect non-instanced shadow draw batches.
func (tb *ShadowTileBuilder) GetStreamDC() *draws.DrawCommands {
	return tb.streamDC
}

// Add records an instanced mesh range for later shadow tile preparation.
func (tb *ShadowTileBuilder) Add(mesh *buffers.ResidentMesh, baseByteOffset int32, instanceCount int32) {
	if tb.instRangesLen >= len(tb.instRanges) {
		newLen := max(len(tb.instRanges)*2, 32)
		z := make([]ShadowInstRange, newLen)
		copy(z, tb.instRanges)
		tb.instRanges = z
	}
	ptr := &tb.instRanges[tb.instRangesLen]
	ptr.Assign(mesh, baseByteOffset, instanceCount)

	tb.instRangesLen++
}

// Prepare builds shadow draw items for opaque instanced submeshes and submits them to the draw command renderer.
func (tb *ShadowTileBuilder) Prepare(shadowInstVBO uint32) {
	for rIdx := range tb.instRangesLen {
		ptr := &tb.instRanges[rIdx]
		for sIdx := 0; sIdx < ptr.mesh.SubMeshesLen; sIdx++ {
			sub := ptr.mesh.SubMeshes[sIdx]
			if sub.IndexCount == 0 || sub.BlendMode != config.BlendModeOpaque {
				continue
			}
			item := tb.instItems.Acquire()
			item.Assign(ptr.mesh, sub, shadowInstVBO, ptr.baseByteOffset, ptr.instanceCount)
		}
	}
	tb.dcRender.Prepare(tb.streamDC.GetDrawCommands())
	tb.dcRender.SetShadowResidentDraws(tb.staticItems.Items, tb.staticItems.Len, tb.instItems.Items, tb.instItems.Len)
}

// PushStaticSubMeshes adds opaque static submeshes in the given index range to the shadow tile builder.
func (tb *ShadowTileBuilder) PushStaticSubMeshes(mesh *buffers.ResidentMesh, startIdx, endIdx int) {
	for sIdx := startIdx; sIdx < endIdx; sIdx++ {
		sub := mesh.SubMeshes[sIdx]
		if sub.IndexCount == 0 || sub.BlendMode != config.BlendModeOpaque {
			continue
		}
		item := tb.staticItems.Acquire()
		item.Mesh = mesh
		item.Sub = sub
	}
}
