package builder

import (
	"github.com/markel1974/godoom/mr_tech/model/ithing"
	"github.com/markel1974/godoom/mr_tech/model/volumes"
	"github.com/markel1974/godoom/mr_tech/physics/rigid_body"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/gpu/buffers"
)

// ShadowsRigidEntry tracks cached state for a rigid shadow-casting entity across frames.
type ShadowsRigidEntry struct {
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
func (re *ShadowsRigidEntry) Reset(entID uint64, frame uint64) {
	re.entID = entID
	re.inUse = true
	re.seenFrame = 0
	re.lastSeenFrame = frame
	re.isRigidActive = false
	re.hash = 0
	re.matValid = false
	re.faces = nil
	re.rm = nil
	if re.stream != nil {
		re.stream.Reset()
	}
}

// Release marks the entry unused and clears cached rigid state, mesh references, and any stream cache data.
func (re *ShadowsRigidEntry) Release() {
	re.inUse = false
	re.isRigidActive = false
	re.hash = 0
	re.matValid = false
	re.faces = nil
	re.rm = nil
	if re.stream != nil {
		re.stream.Release()
	}
}

// EnsureStreamCache returns the entry's render cache, creating it if needed.
// It marks the cache in use and syncs its entity ID and last-seen frame.
func (re *ShadowsRigidEntry) EnsureStreamCache() *ThingRenderCache {
	if re.stream == nil {
		re.stream = NewThingRenderCache()
	}
	re.stream.entID = re.entID
	re.stream.inUse = true
	re.stream.lastSeenFrame = re.lastSeenFrame
	return re.stream
}

// Update refreshes the entry once per shadow frame, syncing cached rigid shadow state when the hash changes.
func (re *ShadowsRigidEntry) Update(shadowFrame uint64, thing ithing.IThing, ent *rigid_body.Entity, hash uint64, bThings *Things) {
	if re.seenFrame != shadowFrame {
		if re.hash != hash {
			re.sync(thing, ent, hash, bThings)
		}
		re.seenFrame = shadowFrame
	}
}

// sync updates the shadow cache entry with the given hash and rigid mesh state.
// It clears rigid state when the thing has no rigid shadow faces.
func (re *ShadowsRigidEntry) sync(thing ithing.IThing, ent *rigid_body.Entity, hash uint64, bThings *Things) {
	re.hash = hash
	if rigidFaces := thing.GetRigidFaces(); rigidFaces != nil && rigidFaces.Len() > 0 {
		tPosX, tPosY, zBot := ent.GetCenter()
		qx, qy, qz, qw := ent.GetQRotation().Get()
		buffers.BuildTransformMatrix(re.mat[:], tPosX, tPosY, zBot, qx, qy, qz, qw)
		re.matValid = true
		if re.faces != rigidFaces || re.rm == nil {
			if bThings != nil {
				re.rm = bThings.FindCachedMesh(rigidFaces)
			}
			re.faces = rigidFaces
		}
		re.isRigidActive = true
		return
	}
	re.isRigidActive = false
	re.matValid = false
	re.faces = nil
	re.rm = nil
}
