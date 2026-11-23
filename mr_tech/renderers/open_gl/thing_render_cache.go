package open_gl

import "github.com/markel1974/godoom/mr_tech/model/volumes"

type ThingCacheRef struct {
	id  uint64
	gen uint64
}

type ThingRenderCache struct {
	fv            *FrameVertices
	pFaces        *volumes.Faces
	pNextFaces    *volumes.Faces
	facesVer      uint64
	nextVer       uint64
	facesLen      int
	nextLen       int
	lerp          float32
	yaw           float32
	renderMode    float32
	oX, oY, oZ    float32
	gen           uint64
	lastSeenFrame uint64
	valid         bool
}
