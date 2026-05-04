package model

import (
	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/model/geometry"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// IVertices represents the interface for handling vertices, including retrieval, transformations, and related operations.
type IVertices interface {
	GetVertices(uint64) ([]*Face, []*Face, float64)

	GetVolume() *Volume

	SetAction(idx int)

	GetDisplacement() (float64, float64, float64)

	GetBillboard() float64

	SetThing(t IThing)
}

// VerticesFactory returns an implementation of IVertices based on the provided Thing configuration and material.
func VerticesFactory(cfg *config.Thing, pos geometry.XYZ, material *textures.Material) IVertices {
	if cfg.MD2 != nil {
		return NewVerticesMD2(cfg, pos, material)
	}
	if cfg.WAX != nil {
		return NewVerticesWAX(cfg, pos, material)
	}
	return NewVerticesSprite(cfg, pos, material)
}
