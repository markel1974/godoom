package builder

import (
	"github.com/markel1974/godoom/mr_tech/model/volumes"
	"github.com/markel1974/godoom/mr_tech/physics/rigid_body"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/gpu/buffers"
)

// ShadowRigidEntry tracks cached state for a rigid shadow-casting entity across frames.
type ShadowRigidEntry struct {
	entID         uint64
	inUse         bool
	seenFrame     uint64
	lastSeenFrame uint64
	isRigidActive bool
	hash          uint64

	matValid bool
	mat      [buffers.InstanceMatrixFloats]float32
	faces    *volumes.Faces
	rm       *buffers.ResidentMesh

	stream *ThingRenderCache
}

// Reset reinitializes the entry for the entity and frame, clearing cached mesh, transform, hash, and stream state.
func (rEntry *ShadowRigidEntry) Reset(entID uint64, frame uint64) {
	rEntry.entID = entID
	rEntry.inUse = true
	rEntry.seenFrame = 0
	rEntry.lastSeenFrame = frame
	rEntry.isRigidActive = false
	rEntry.hash = 0
	rEntry.matValid = false
	rEntry.faces = nil
	rEntry.rm = nil
	if rEntry.stream != nil {
		rEntry.stream.valid = false
		rEntry.stream.pFaces = nil
		rEntry.stream.pNextFaces = nil
		rEntry.stream.compileFrame = 0
		rEntry.stream.committedFrame = 0
		rEntry.stream.committedBaseIdx = -1
		rEntry.stream.fv.DeepReset()
	}
}

// Release marks the entry unused and clears cached rigid state, mesh references, and any stream cache data.
func (rEntry *ShadowRigidEntry) Release() {
	rEntry.inUse = false
	rEntry.isRigidActive = false
	rEntry.hash = 0
	rEntry.matValid = false
	rEntry.faces = nil
	rEntry.rm = nil
	if rEntry.stream != nil {
		rEntry.stream.inUse = false
		rEntry.stream.valid = false
		rEntry.stream.pFaces = nil
		rEntry.stream.pNextFaces = nil
		rEntry.stream.fv.DeepReset()
	}
}

// EnsureStreamCache returns the entry's render cache, creating it if needed.
// It marks the cache in use and syncs its entity ID and last-seen frame.
func (rEntry *ShadowRigidEntry) EnsureStreamCache() *ThingRenderCache {
	if rEntry.stream == nil {
		rEntry.stream = NewThingRenderCache()
	}
	rEntry.stream.entID = rEntry.entID
	rEntry.stream.inUse = true
	rEntry.stream.lastSeenFrame = rEntry.lastSeenFrame
	return rEntry.stream
}

// SyncRigid updates the entry transform and cached mesh from the rigid entity and marks it active.
func (rEntry *ShadowRigidEntry) SyncRigid(ent *rigid_body.Entity, rigidFaces *volumes.Faces, bThings *Things) {
	tPosX, tPosY, zBot := ent.GetCenter()
	qx, qy, qz, qw := ent.GetQRotation().Get()
	buffers.BuildTransformMatrix(rEntry.mat[:], tPosX, tPosY, zBot, qx, qy, qz, qw)
	rEntry.matValid = true
	if rEntry.faces != rigidFaces || rEntry.rm == nil {
		rEntry.rm = bThings.FindCachedMesh(rigidFaces)
		rEntry.faces = rigidFaces
	}
	rEntry.isRigidActive = true
}
