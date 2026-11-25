package open_gl

import (
	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/model"
	"github.com/markel1974/godoom/mr_tech/model/lights"
	"github.com/markel1974/godoom/mr_tech/physics/aabb"
)

// BuilderLights collects and classifies visible lights around the camera into FrameLights.
type BuilderLights struct {
	fl           *FrameLights
	lightBB      *aabb.BoundingBox
	pX, pY, pZ   float64
	frustumFront *aabb.Frustum
}

// NewBuilderLights creates and initializes a BuilderLights instance using shared BuilderContext configuration.
func NewBuilderLights(bc *BuilderContext) *BuilderLights {
	enableShadowLights := false
	if bc.cal != nil {
		enableShadowLights = bc.cal.EnableShadowLights
	}
	return &BuilderLights{
		fl:      NewFrameLights(bc.ctx, 1024, enableShadowLights),
		lightBB: aabb.NewBoundingBox(),
	}
}

// GetFrameLights returns the FrameLights instance associated with this BuilderLights.
func (bl *BuilderLights) GetFrameLights() *FrameLights {
	return bl.fl
}

// Process gathers lights overlapping the camera light box and classifies them against frustumFront.
func (bl *BuilderLights) Process(ls *model.Lights, frustumFront *aabb.Frustum, pX, pY, pZ float64, w, h, d float64) {
	bl.fl.DeepReset()
	bl.fl.Prepare()
	if ls == nil {
		return
	}
	bl.frustumFront = frustumFront
	bl.pX, bl.pY, bl.pZ = pX, pY, pZ
	bl.lightBB.RebuildCenter(pX, pY, pZ, w, h, d)
	ls.QueryOverlaps(bl.lightBB, bl.doQuery)
}

func (bl *BuilderLights) doQuery(object aabb.IObject) bool {
	light, ok := object.(*lights.Light)
	if !ok || light == nil || light.GetKind() == config.LightKindNone {
		return false
	}
	distSq := light.GetDistSq(bl.pX, bl.pY, bl.pZ)
	isFront := bl.frustumFront != nil && light.GetAABB().IntersectFrustum(bl.frustumFront)
	bl.fl.Create(light, distSq, isFront)
	return false
}
