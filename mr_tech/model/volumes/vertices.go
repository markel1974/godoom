package volumes

import (
	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/model/materials"
	"github.com/markel1974/godoom/mr_tech/physics/aabb"
	"github.com/markel1974/godoom/mr_tech/physics/collisions"
)

const (
	RenderModeOpaque             = 0.0
	RenderModeLiquid             = 0.5
	RenderModeBillboard          = 1.0
	RenderModeBillboardSpherical = 1.1
	RenderModeModel3D            = 2.0
	RenderModeInterface          = 3.0
)

// IVertices represents the interface for handling vertices, including retrieval, transformations, and related operations.
type IVertices interface {
	Setup(thing IThing) error

	Compile(uint64) bool

	GetAABB() *aabb.AABB

	GetSolver() *collisions.Solver

	QueryOverlaps(object aabb.IObject, callback func(object aabb.IObject) bool)

	GetUIFaces() (*Faces, *Faces, float64, float64)

	SetAction(idx int)

	ApplyImpact(fx, fy, fz, px, py, pz float64)

	SetRagdollMode()
}

// CreateVertices returns an implementation of IVertices based on the provided Thing configuration and material.
func CreateVertices(cfg *config.Thing, materials *materials.Materials) IVertices {
	var out IVertices
	if cfg.Model3D != nil {
		out = NewVertices3D(cfg, materials)
	} else if cfg.Model3DEntry != nil {
		out = NewVertices3DEntry(cfg, materials)
	} else if cfg.MultiSprite != nil {
		out = NewVerticesMultiSprite(cfg, materials)
	} else {
		out = NewVerticesSprite(cfg, materials)
	}
	return out
}
