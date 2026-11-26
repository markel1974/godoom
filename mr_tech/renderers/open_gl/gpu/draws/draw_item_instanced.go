package draws

import "github.com/markel1974/godoom/mr_tech/renderers/open_gl/gpu/buffers"

// DrawItemInstanced describes an instanced draw command for a resident submesh and its instance buffer range.
type DrawItemInstanced struct {
	Mesh           *buffers.ResidentMesh
	Sub            *buffers.ResidentSubMesh
	InstanceVBO    uint32
	BaseByteOffset int32
	InstanceCount  int32
}

func (item *DrawItemInstanced) Assign(mesh *buffers.ResidentMesh, sub *buffers.ResidentSubMesh, instanceVBO uint32, baseByteOffset int32, instanceCount int32) {
	item.Mesh = mesh
	item.Sub = sub
	item.InstanceVBO = instanceVBO
	item.BaseByteOffset = baseByteOffset
	item.InstanceCount = instanceCount
}
