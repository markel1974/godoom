package model

import (
	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/physics"
)

// IVertices represents the interface for handling vertices, including retrieval, transformations, and related operations.
type IVertices interface {
	GetVertices(uint64) (*[]*Face, int, *[]*Face, int, float64, float64)

	GetVolume() *Volume

	GetAABB() *physics.AABB

	GetEntity() *physics.Entity

	SetAction(idx int)

	GetDisplacement() (float64, float64, float64)

	GetBillboard() float64

	SetThing(t IThing)
}

// VerticesFactory returns an implementation of IVertices based on the provided Thing configuration and material.
func VerticesFactory(thing IThing, cfg *config.Thing, materials *Materials) IVertices {
	var out IVertices
	if cfg.MD1 != nil {
		out = NewVerticesMD2(cfg, materials)
	} else if cfg.MultiSprite != nil {
		out = NewVerticesMultiSprite(cfg, materials)
	} else {
		out = NewVerticesSprite(cfg, materials)
	}
	out.SetThing(thing)
	return out
}
