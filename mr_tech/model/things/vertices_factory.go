package things

import (
	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/model/materials"
	"github.com/markel1974/godoom/mr_tech/model/vertices"
	"github.com/markel1974/godoom/mr_tech/model/vertices/vertices_3d"
	"github.com/markel1974/godoom/mr_tech/model/vertices/vertices_3d_entry"
	"github.com/markel1974/godoom/mr_tech/model/vertices/vertices_multi_sprite"
	"github.com/markel1974/godoom/mr_tech/model/vertices/vertices_sprite"
)

// CreateVertices returns an implementation of IVertices based on the provided Thing configuration and material.
func CreateVertices(cfg *config.Thing, materials *materials.Materials) vertices.IVertices {
	var out vertices.IVertices
	if cfg.Model3D != nil {
		out = vertices_3d.NewVertices3D(cfg, materials)
	} else if cfg.Model3DEntry != nil {
		out = vertices_3d_entry.NewVertices3DEntry(cfg, materials)
	} else if cfg.MultiSprite != nil {
		out = vertices_multi_sprite.NewMultiSprite(cfg, materials)
	} else {
		out = vertices_sprite.NewVerticesSprite(cfg, materials)
	}
	return out
}
