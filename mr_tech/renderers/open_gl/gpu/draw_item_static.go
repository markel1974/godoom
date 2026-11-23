package gpu

// DrawItemStatic pairs a GPU-resident mesh with one of its submesh ranges for static rendering.
type DrawItemStatic struct {
	Mesh *ResidentMesh
	Sub  *ResidentSubMesh
}
