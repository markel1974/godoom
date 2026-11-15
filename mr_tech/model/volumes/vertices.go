package volumes

import (
	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/model/materials"
	"github.com/markel1974/godoom/mr_tech/physics/aabb"
	"github.com/markel1974/godoom/mr_tech/physics/rigid_body"
)

const (
	RenderModeOpaque             = 0.0
	RenderModeLiquid             = 0.5
	RenderModeBillboard          = 1.0
	RenderModeBillboardSpherical = 1.1
	RenderModeModel3D            = 2.0
	RenderModeInterface          = 3.0
)

type IContainerVolume interface {
	QueryOverlaps(object aabb.IObject, callback func(object aabb.IObject) bool)

	GetEntity() *rigid_body.Entity
}

// IVertices represents the interface for handling vertices, including retrieval, transformations, and related operations.
type IVertices interface {
	GetAABB() *aabb.AABB

	Compile(uint64) bool

	GetUIFaces() (*Faces, *Faces, float64, float64)

	GetContainerVolume() IContainerVolume

	SetAction(idx int)

	ApplyImpact(fx, fy, fz, px, py, pz float64)

	SetRagdollMode()
}

// VerticesFactory returns an implementation of IVertices based on the provided Thing configuration and material.
func VerticesFactory(thing IThing, cfg *config.Thing, materials *materials.Materials) IVertices {
	var out IVertices
	if cfg.Model3D != nil {
		out = NewVertices3D(thing, cfg, materials)
	} else if cfg.Model3DEntry != nil {
		out = NewVertices3DEntry(thing, cfg, materials)
	} else if cfg.MultiSprite != nil {
		out = NewVerticesMultiSprite(thing, cfg, materials)
	} else {
		out = NewVerticesSprite(thing, cfg, materials)
	}
	return out
}
