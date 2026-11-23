package draws

import (
	"unsafe"

	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/gpu/buffers"
)

// InstShadowCommand stores a precomputed instanced draw entry for shadow atlas rendering.
type InstShadowCommand struct {
	mesh           *buffers.ResidentMesh
	instanceVBO    uint32
	baseByteOffset int32
	instanceCount  int32
	indexCount     int32
	indexOffsetPtr unsafe.Pointer
	depthWrite     int
	polygonOffset  int
	cullMode       int
	hasCutout      bool
}
