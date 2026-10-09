package model

import (
	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/physics"
	"github.com/markel1974/godoom/mr_tech/physics/aabb"
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
	GetVertices(uint64) (*[]*Face, int, *[]*Face, int, float64, float64)

	GetVolume() *Volume

	GetAABB() *aabb.AABB

	GetEntity() *physics.Entity

	SetAction(idx int)

	GetDisplacement() (float64, float64, float64)

	GetRenderMode() float64

	SetThing(t IThing)
}

// VerticesFactory returns an implementation of IVertices based on the provided Thing configuration and material.
func VerticesFactory(thing IThing, cfg *config.Thing, materials *Materials) IVertices {
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
	out.SetThing(thing)
	return out
}
