package shaders

import (
	"cmp"
	"slices"
)

// TransparentKind identifies the source pipeline for a sorted translucent item.
type TransparentKind uint8

const (
	TransparentStreaming TransparentKind = iota
	TransparentStatic
	TransparentInstanced
)

// StaticDrawItem binds a GPU-resident static submesh for direct drawing.
type StaticDrawItem struct {
	Mesh *ResidentMesh
	Sub  *ResidentSubMesh
}

// InstancedDrawItem binds a contiguous slice of instances sharing one GPU-resident submesh.
type InstancedDrawItem struct {
	Mesh           *ResidentMesh
	Sub            *ResidentSubMesh
	InstanceVBO    uint32
	BaseByteOffset int32
	InstanceCount  int32
}

// TransparentDrawItem holds one translucent submesh/command sorted back-to-front per frame.
type TransparentDrawItem struct {
	Kind           TransparentKind
	DistSq         float32
	SortOrder      int
	CullMode       int
	HasCutout      bool
	PolygonOffset  bool
	FirstIndex     int32
	IndexCount     int32
	Mesh           *ResidentMesh
	Sub            *ResidentSubMesh
	InstanceVBO    uint32
	BaseByteOffset int32
	InstanceCount  int32
}

type TransparentDrawItems struct {
	data []*TransparentDrawItem
}

// SortTransparentDrawItems sorts translucent items[:n] strictly back-to-front (largest DistSq first),
// using SortOrder ascending as a deterministic co-planar tie-breaker.
func SortTransparentDrawItems(items []*TransparentDrawItem, n int) {
	if n <= 1 {
		return
	}
	slices.SortStableFunc(items[:n], func(a, b *TransparentDrawItem) int {
		diff := a.DistSq - b.DistSq
		if diff > 1e-2 {
			return -1
		}
		if diff < -1e-2 {
			return 1
		}
		return cmp.Compare(a.SortOrder, b.SortOrder)
	})
}
