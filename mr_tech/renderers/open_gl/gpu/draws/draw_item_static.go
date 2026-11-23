package draws

import "github.com/markel1974/godoom/mr_tech/renderers/open_gl/gpu/buffers"

// DrawItemStatic pairs a GPU-resident mesh with one of its submesh ranges for static rendering.
type DrawItemStatic struct {
	Mesh *buffers.ResidentMesh
	Sub  *buffers.ResidentSubMesh
}
