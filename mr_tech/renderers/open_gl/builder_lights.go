package open_gl

import (
	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/model"
	"github.com/markel1974/godoom/mr_tech/model/lights"
	"github.com/markel1974/godoom/mr_tech/physics/aabb"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
)

// BuilderLights represents a structure for managing lights and their bounding box in a rendering scene.
type BuilderLights struct {
	fl           *FrameLights
	lightBB      *aabb.BoundingBox
	pX, pY, pZ   float64
	frustumFront *aabb.Frustum
}

// NewBuilderLights creates and initializes a new BuilderLights instance with frame lights and a bounding box.
func NewBuilderLights(ctx api.IContext, cal *model.Calibration) *BuilderLights {
	return &BuilderLights{
		fl:      NewFrameLights(ctx, 1024, cal.EnableShadowLights),
		lightBB: aabb.NewBoundingBox(),
	}
}

// GetFrameLights returns the FrameLights instance associated with the BuilderLights.
func (bl *BuilderLights) GetFrameLights() *FrameLights {
	return bl.fl
}

// Process processes a collection of lights, applying spatial filtering and frustum intersection based on the given parameters.
func (bl *BuilderLights) Process(ls *model.Lights, frustumFront *aabb.Frustum, pX, pY, pZ float64, w, h, d float64) {
	bl.frustumFront = frustumFront
	bl.pX, bl.pY, bl.pZ = pX, pY, pZ
	bl.lightBB.RebuildCenter(pX, pY, pZ, w, h, d)
	bl.fl.DeepReset()
	bl.fl.Prepare()
	//counter := 0
	ls.QueryOverlaps(bl.lightBB, bl.doQuery)
	//fmt.Println("Processed lights:", ls.Len(), counter)
}

func (bl *BuilderLights) doQuery(object aabb.IObject) bool {
	light := object.(*lights.Light)
	if light.GetKind() == config.LightKindNone {
		return false
	}
	distSq := light.GetDistSq(bl.pX, bl.pY, bl.pZ)
	isFront := light.GetAABB().IntersectFrustum(bl.frustumFront)
	bl.fl.Create(light, distSq, isFront)
	//counter++
	return false
}
