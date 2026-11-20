package things

import (
	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/model/ithing"
	"github.com/markel1974/godoom/mr_tech/model/materials"
	"github.com/markel1974/godoom/mr_tech/model/vertices/vertices_element_3d"
	"github.com/markel1974/godoom/mr_tech/model/vertices/vertices_multi_element_3d"
	"github.com/markel1974/godoom/mr_tech/model/vertices/vertices_multi_sprite"
	"github.com/markel1974/godoom/mr_tech/model/vertices/vertices_sprite"
)

// CreateVertices returns an implementation of IVertices based on the provided Thing configuration and material.
func CreateVertices(cfg *config.Thing, materials *materials.Materials) ithing.IVertices {
	var out ithing.IVertices
	if cfg.MultiElement3D != nil {
		out = vertices_multi_element_3d.NewMultiElement3D(cfg, materials)
	} else if cfg.Element3D != nil {
		out = vertices_element_3d.NewElement3D(cfg, materials)
	} else if cfg.MultiSprite != nil {
		out = vertices_multi_sprite.NewMultiSprite(cfg, materials)
	} else {
		out = vertices_sprite.NewVerticesSprite(cfg, materials)
	}
	return out
}
