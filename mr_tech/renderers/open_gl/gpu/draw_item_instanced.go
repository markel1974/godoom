package gpu

// DrawItemInstanced describes an instanced draw command for a resident submesh and its instance buffer range.
type DrawItemInstanced struct {
	Mesh           *ResidentMesh
	Sub            *ResidentSubMesh
	InstanceVBO    uint32
	BaseByteOffset int32
	InstanceCount  int32
}
