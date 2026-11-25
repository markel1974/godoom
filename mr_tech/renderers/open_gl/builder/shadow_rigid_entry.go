package builder

import (
	"unsafe"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/model/ithing"
	"github.com/markel1974/godoom/mr_tech/model/volumes"
	"github.com/markel1974/godoom/mr_tech/physics/rigid_body"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/gpu/buffers"
)

type staticRigidPoseChecker interface {
	IsStaticRigidPose() bool
}

// ShadowRigidEntry stores per-entity cached shadow state, rigid transform matrix, incremental hashes,
// cached interface assertions, and optional lazy streaming vertex cache to keep off-camera shadow queries O(1).
type ShadowRigidEntry struct {
	entID         uint64
	inUse         bool
	seenFrame     uint64
	lastSeenFrame uint64
	isRigidActive bool
	hash          uint64

	rigidSrcChecked bool
	rigidSrc        ithing.IRigidRenderSource
	staticChecked   bool
	staticCap       staticRigidPoseChecker
	staticFastOK    bool
	poseGen         uint64
	hasAnimMat      bool

	matValid   bool
	hashValid  bool
	cx, cy, cz float64
	qx, qy, qz float64
	qw         float64
	mat        [buffers.InstanceMatrixFloats]float32
	faces      *volumes.Faces
	facesVer   uint64
	facesLen   int
	rm         *buffers.ResidentMesh
	baseHash   uint64

	stream          *ThingRenderCache
	streamHashValid bool
	streamHashSrc   *ThingRenderCache
	streamHashGen   uint64
	streamHash      uint64
}

func (rEntry *ShadowRigidEntry) reset(entID uint64, frame uint64) {
	rEntry.entID = entID
	rEntry.inUse = true
	rEntry.seenFrame = 0
	rEntry.lastSeenFrame = frame
	rEntry.isRigidActive = false
	rEntry.hash = 0
	rEntry.rigidSrcChecked = false
	rEntry.rigidSrc = nil
	rEntry.staticChecked = false
	rEntry.staticCap = nil
	rEntry.staticFastOK = false
	rEntry.poseGen = 0
	rEntry.hasAnimMat = false
	rEntry.matValid = false
	rEntry.hashValid = false
	rEntry.faces = nil
	rEntry.facesVer = 0
	rEntry.facesLen = 0
	rEntry.rm = nil
	rEntry.baseHash = 0
	rEntry.streamHashValid = false
	rEntry.streamHashSrc = nil
	rEntry.streamHashGen = 0
	rEntry.streamHash = 0
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

func (rEntry *ShadowRigidEntry) release() {
	rEntry.inUse = false
	rEntry.rigidSrc = nil
	rEntry.rigidSrcChecked = false
	rEntry.staticCap = nil
	rEntry.staticChecked = false
	rEntry.staticFastOK = false
	rEntry.poseGen = 0
	rEntry.hasAnimMat = false
	rEntry.faces = nil
	rEntry.rm = nil
	rEntry.streamHashValid = false
	rEntry.streamHashSrc = nil
	if rEntry.stream != nil {
		rEntry.stream.inUse = false
		rEntry.stream.valid = false
		rEntry.stream.pFaces = nil
		rEntry.stream.pNextFaces = nil
		rEntry.stream.fv.DeepReset()
	}
}

// EnsureStreamCache returns this entity's dedicated off-camera streaming ThingRenderCache, allocating lazily on first use.
func (rEntry *ShadowRigidEntry) EnsureStreamCache() *ThingRenderCache {
	if rEntry.stream == nil {
		rEntry.stream = NewThingRenderCache()
	}
	rEntry.stream.entID = rEntry.entID
	rEntry.stream.inUse = true
	rEntry.stream.lastSeenFrame = rEntry.lastSeenFrame
	return rEntry.stream
}

func facesHasAnimatedMaterial(faces *volumes.Faces) bool {
	n := faces.Len()
	for i := range n {
		f := faces.At(i)
		if f == nil {
			continue
		}
		mat := f.GetMaterial()
		if mat != nil && mat.GetKind() != int(config.MaterialKindNone) && mat.IsAnimated() {
			return true
		}
	}
	return false
}

// Update updates the shadow rigid entry's transform matrix, cached mesh, and incremental hash only when inputs change.
func (rEntry *ShadowRigidEntry) Update(entID uint64, ent *rigid_body.Entity, rigidFaces *volumes.Faces, bThings *Things, shadowFrame uint64) uint64 {
	tPosX, tPosY, zBot := ent.GetCenter()
	qx, qy, qz, qw := ent.GetQRotation().Get()
	matChanged := !rEntry.matValid ||
		rEntry.cx != tPosX || rEntry.cy != tPosY || rEntry.cz != zBot ||
		rEntry.qx != qx || rEntry.qy != qy || rEntry.qz != qz || rEntry.qw != qw
	if matChanged {
		rEntry.cx, rEntry.cy, rEntry.cz = tPosX, tPosY, zBot
		rEntry.qx, rEntry.qy, rEntry.qz, rEntry.qw = qx, qy, qz, qw
		buffers.BuildTransformMatrix(rEntry.mat[:], tPosX, tPosY, zBot, qx, qy, qz, qw)
		rEntry.matValid = true
	}

	facesVer := rigidFaces.GetVersion()
	facesLen := rigidFaces.Len()
	facesChanged := !rEntry.hashValid || rEntry.rm == nil || rEntry.faces != rigidFaces || rEntry.facesVer != facesVer || rEntry.facesLen != facesLen
	if facesChanged {
		rm := bThings.FindCachedMesh(rigidFaces)
		rEntry.rm = rm
		rEntry.faces = rigidFaces
		rEntry.facesVer = facesVer
		rEntry.facesLen = facesLen
		if rm != nil {
			rEntry.hasAnimMat = rm.AnimLen > 0
		} else {
			rEntry.hasAnimMat = facesHasAnimatedMaterial(rigidFaces)
		}
	}
	rEntry.poseGen = ent.GetPoseGen()

	if matChanged || facesChanged || !rEntry.hashValid {
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
		rEntry.baseHash = h
		rEntry.hashValid = true
	}

	h := rEntry.baseHash
	if rm := rEntry.rm; rm != nil && rm.AnimLen > 0 {
		for i := 0; i < rm.AnimLen; i++ {
			h = mixU64(h, mixPtrHash(unsafe.Pointer(rm.AnimMats[i].GetCurrentFrame())))
		}
	}
	rEntry.hash = h
	rEntry.seenFrame = shadowFrame
	return h
}
