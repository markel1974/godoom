package model

import (
	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/model/geometry"
)

// IVertices represents the interface for handling vertices, including retrieval, transformations, and related operations.
type IVertices interface {
	GetVertices(uint64) ([]*Face, int, []*Face, int, float64)

	GetVolume() *Volume

	SetAction(idx int)

	GetDisplacement() (float64, float64, float64)

	GetBillboard() float64

	SetThing(t IThing)
}

// VerticesFactory returns an implementation of IVertices based on the provided Thing configuration and material.
func VerticesFactory(thing IThing, cfg *config.Thing, pos geometry.XYZ, materials *Materials) IVertices {
	var out IVertices
	if cfg.MD1 != nil {
		out = NewVerticesMD2(cfg, pos, materials)
	} else if cfg.MultiSprite != nil {
		out = NewVerticesMultiSprite(cfg, pos, materials)
	} else {
		out = NewVerticesSprite(cfg, pos, materials)
	}
	out.SetThing(thing)
	return out
}
