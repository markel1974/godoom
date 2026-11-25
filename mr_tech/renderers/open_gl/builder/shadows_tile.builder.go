package builder

import (
	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/gpu/buffers"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/gpu/draws"
)

// ShadowInstRange represents a range of shadow instances associated with a specific resident mesh and rendering properties.
type ShadowInstRange struct {
	mesh           *buffers.ResidentMesh
	baseByteOffset int32
	instanceCount  int32
}

// ShadowTileBuilder is responsible for constructing shadow tiles used in rendering pipelines.
type ShadowTileBuilder struct {
	dcRender    *draws.DrawCommandsRender
	streamDC    *draws.DrawCommands
	staticItems buffers.PtrPool[draws.DrawItemStatic]
	instItems   buffers.PtrPool[draws.DrawItemInstanced]
	instRanges  []ShadowInstRange
}

// NewShadowTileBuilder creates and initializes a ShadowTileBuilder instance with rendering and draw command setups.
func NewShadowTileBuilder(ctx api.IContext) *ShadowTileBuilder {
	return &ShadowTileBuilder{
		dcRender:   draws.NewDrawCommandsRender(ctx, "shadow_tile", false),
		streamDC:   draws.NewDrawCommands(config.BlendModeOpaque, 128),
		instRanges: make([]ShadowInstRange, 0, 32),
	}
}

// Prepare processes shadow instance ranges and initializes draw commands for rendering with given instance VBO.
func (tb *ShadowTileBuilder) Prepare(shadowInstVBO uint32) {
	for rIdx := range tb.instRanges {
		r := &tb.instRanges[rIdx]
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
