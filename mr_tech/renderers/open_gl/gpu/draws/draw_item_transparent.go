package draws

import "github.com/markel1974/godoom/mr_tech/renderers/open_gl/gpu/buffers"

// TransparentKind identifies how a transparent draw item is sourced and rendered.
type TransparentKind uint8

// TransparentKind values define how transparent geometry is rendered.
const (
	TransparentStreaming TransparentKind = iota
	TransparentStatic
	TransparentInstanced
)

// DrawItemTransparent describes a queued transparent draw with sorting and render-state metadata.
type DrawItemTransparent struct {
	Kind           TransparentKind
	DistSq         float32
	SortOrder      int
	CullMode       int
	HasCutout      bool
	PolygonOffset  bool
	FirstIndex     int32
	IndexCount     int32
	Mesh           *buffers.ResidentMesh
	Sub            *buffers.ResidentSubMesh
	InstanceVBO    uint32
	BaseByteOffset int32
	InstanceCount  int32
}
