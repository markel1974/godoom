package builder

import (
	"unsafe"

	"github.com/markel1974/godoom/mr_tech/model/volumes"
	"github.com/markel1974/godoom/mr_tech/physics/rigid_body"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/gpu/buffers"
)

// ShadowRigidEntry represents a data structure for managing a shadow's rigid body transformations and associated metadata.
type ShadowRigidEntry struct {
	entID          uint64
	inUse          bool
	seenFrame      uint64
	lastSeenFrame  uint64
	matValid       bool
	cx, cy, cz     float64
	qx, qy, qz, qw float64
	mat            [buffers.InstanceMatrixFloats]float32
	faces          *volumes.Faces
	facesVer       uint64
	facesLen       int
	rm             *buffers.ResidentMesh
	hash           uint64
}

// updateShadowRigidEntry updates the shadow rigid entry's transform matrix, cached mesh, and hash if necessary.

func (rEntry *ShadowRigidEntry) Update(entID uint64, ent *rigid_body.Entity, rigidFaces *volumes.Faces, bThings *Things, shadowFrame uint64) uint64 {
	tPosX, tPosY, zBot := ent.GetCenter()
	qx, qy, qz, qw := ent.GetQRotation().Get()
	if !rEntry.matValid || rEntry.cx != tPosX || rEntry.cy != tPosY || rEntry.cz != zBot || rEntry.qx != qx || rEntry.qy != qy || rEntry.qz != qz || rEntry.qw != qw {
		rEntry.cx, rEntry.cy, rEntry.cz = tPosX, tPosY, zBot
		rEntry.qx, rEntry.qy, rEntry.qz, rEntry.qw = qx, qy, qz, qw
		buffers.BuildTransformMatrix(rEntry.mat[:], tPosX, tPosY, zBot, qx, qy, qz, qw)
		rEntry.matValid = true
	}

	facesVer := rigidFaces.GetVersion()
	facesLen := rigidFaces.Len()
	if rEntry.rm == nil || rEntry.faces != rigidFaces || rEntry.facesVer != facesVer || rEntry.facesLen != facesLen {
		rEntry.rm = bThings.FindCachedMesh(rigidFaces)
		rEntry.faces = rigidFaces
		rEntry.facesVer = facesVer
		rEntry.facesLen = facesLen
	}

	h := mixU64(entID, mixPtrHash(unsafe.Pointer(rigidFaces)))
	h = mixU64(h, facesVer)
	h = mixU64(h, uint64(facesLen))
	h = mixU64(h, f32BitsStable(float32(tPosX)))
	h = mixU64(h, f32BitsStable(float32(tPosY)))
	h = mixU64(h, f32BitsStable(float32(zBot)))
	h = mixU64(h, f32BitsStable(float32(qx)))
	h = mixU64(h, f32BitsStable(float32(qy)))
	h = mixU64(h, f32BitsStable(float32(qz)))
	h = mixU64(h, f32BitsStable(float32(qw)))
	if rm := rEntry.rm; rm != nil && rm.AnimLen > 0 {
		for i := 0; i < rm.AnimLen; i++ {
			h = mixU64(h, mixPtrHash(unsafe.Pointer(rm.AnimMats[i].GetCurrentFrame())))
		}
	}
	rEntry.hash = h
	rEntry.seenFrame = shadowFrame
	return h
}
