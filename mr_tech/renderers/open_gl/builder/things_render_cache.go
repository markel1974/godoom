package builder

import (
	"github.com/markel1974/godoom/mr_tech/model/volumes"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/gpu/draws"
)

// ThingRenderCache stores cached render state for reusing generated thing frame vertices across frames.
type ThingRenderCache struct {
	fv               *draws.FrameVertices
	pFaces           *volumes.Faces
	pNextFaces       *volumes.Faces
	facesVer         uint64
	nextVer          uint64
	facesLen         int
	nextLen          int
	lerp             float32
	yaw              float32
	renderMode       float32
	oX, oY, oZ       float32
	qX, qY, qZ       float32
	qW               float32
	isRigid          bool
	entID            uint64
	gen              uint64
	lastSeenFrame    uint64
	compileFrame     uint64
	committedFrame   uint64
	committedBaseIdx int32
	inUse            bool
	valid            bool
}

// NewThingRenderCache creates a render cache with CPU-only staging frame vertices for thing rendering.
func NewThingRenderCache() *ThingRenderCache {
	return &ThingRenderCache{
		fv:               draws.NewStagingFrameVertices(128),
		committedBaseIdx: -1,
	}
}

// Reset invalidates the render cache, clears face references and frame state, and deeply resets its frame vertices.
func (tr *ThingRenderCache) Reset() {
	tr.valid = false
	tr.pFaces = nil
	tr.pNextFaces = nil
	tr.compileFrame = 0
	tr.committedFrame = 0
	tr.committedBaseIdx = -1
	tr.fv.DeepReset()
}

// Release marks the render cache unused, invalidates it, clears face references, and resets frame vertices.
func (tr *ThingRenderCache) Release() {
	tr.inUse = false
	tr.valid = false
	tr.pFaces = nil
	tr.pNextFaces = nil
	tr.fv.DeepReset()
}

// GetFrameVertices returns the cached frame vertices for this render cache.
func (tr *ThingRenderCache) GetFrameVertices() *draws.FrameVertices {
	return tr.fv
}

// GetGen returns the monotonic cache revision counter for this entry.
func (tr *ThingRenderCache) GetGen() uint64 {
	return tr.gen
}

// Equal reports whether the cache is valid and matches the supplied render state without needing animation rebuild.
func (tr *ThingRenderCache) Equal(pFaces *volumes.Faces, pNextFaces *volumes.Faces, facesVer uint64, nextVer uint64, facesLen int, nextLen int, lerp float32, yaw float32, renderModeF float32, oX float32, oY float32, oZ float32) bool {
	return tr.valid &&
		!tr.isRigid &&
		tr.pFaces == pFaces &&
		tr.pNextFaces == pNextFaces &&
		tr.facesVer == facesVer &&
		tr.nextVer == nextVer &&
		tr.facesLen == facesLen &&
		tr.nextLen == nextLen &&
		tr.lerp == lerp &&
		tr.yaw == yaw &&
		tr.renderMode == renderModeF &&
		tr.oX == oX && tr.oY == oY && tr.oZ == oZ &&
		!tr.fv.NeedsAnimRebuild()
}

// EqualRigid reports whether a rigid instanced entity's template, transform, and orientation match the previous frame.
func (tr *ThingRenderCache) EqualRigid(pFaces *volumes.Faces, facesVer uint64, facesLen int, oX, oY, oZ, qX, qY, qZ, qW float32) bool {
	return tr.valid &&
		tr.isRigid &&
		tr.pFaces == pFaces &&
		tr.facesVer == facesVer &&
		tr.facesLen == facesLen &&
		tr.oX == oX && tr.oY == oY && tr.oZ == oZ &&
		tr.qX == qX && tr.qY == qY && tr.qZ == qZ && tr.qW == qW
}

// Assign updates the cached render inputs, marks the cache valid, and advances its revision counter.
func (tr *ThingRenderCache) Assign(pFaces *volumes.Faces, pNextFaces *volumes.Faces, facesVer uint64, nextVer uint64, facesLen int, nextLen int, lerp float32, yaw float32, renderModeF float32, oX float32, oY float32, oZ float32) {
	tr.isRigid = false
	tr.pFaces = pFaces
	tr.pNextFaces = pNextFaces
	tr.facesVer = facesVer
	tr.nextVer = nextVer
	tr.facesLen = facesLen
	tr.nextLen = nextLen
	tr.lerp = lerp
	tr.yaw = yaw
	tr.renderMode = renderModeF
	tr.oX, tr.oY, tr.oZ = oX, oY, oZ
	tr.valid = true
	tr.gen++
}

// AssignRigid updates a rigid instanced entity's transform cache and advances its revision counter.
func (tr *ThingRenderCache) AssignRigid(pFaces *volumes.Faces, facesVer uint64, facesLen int, oX, oY, oZ, qX, qY, qZ, qW float32) {
	tr.isRigid = true
	tr.pFaces = pFaces
	tr.pNextFaces = nil
	tr.facesVer = facesVer
	tr.nextVer = 0
	tr.facesLen = facesLen
	tr.nextLen = 0
	tr.oX, tr.oY, tr.oZ = oX, oY, oZ
	tr.qX, tr.qY, tr.qZ, tr.qW = qX, qY, qZ, qW
	tr.valid = true
	tr.gen++
}
