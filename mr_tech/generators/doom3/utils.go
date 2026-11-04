package doom3

import (
	"strings"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/generators/doom3/proc"
)

func ToMaterial(texNameLC string) config.MaterialKind {
	kind := config.MaterialKindSolid
	if texNameLC == "translucent" {
		kind = config.MaterialKindSolid //TODO config.MaterialKindTranslucent
	} else if strings.Contains(texNameLC, "sky") {
		kind = config.MaterialKindSky
	} else if strings.Contains(texNameLC, "water") || strings.Contains(texNameLC, "liquid") {
		kind = config.MaterialKindWater
	} else if strings.Contains(texNameLC, "slime") {
		kind = config.MaterialKindSlime
	} else if strings.Contains(texNameLC, "common/collision") || strings.Contains(texNameLC, "common/nodraw") || strings.Contains(texNameLC, "common/caulk") || strings.Contains(texNameLC, "common/trig") {
		kind = config.MaterialKindNone
	}
	return kind
}

// CreateConfigMaterial creates and returns a configured `config.Material` based on the provided material name.
func CreateConfigMaterial(def *proc.MaterialDef) *config.Material {
	//def := materials.Resolve(materialName)
	tex := []string{def.DiffuseMap}
	mat := config.NewConfigMaterial(def.Name, tex, ToMaterial(def.Kind2))
	if def.NormalMap != "" {
		mat.NormalMap = def.NormalMap
	}
	if def.SpecularMap != "" {
		mat.SpecularMap = def.SpecularMap
		mat.SpecularBoost = 0.4
		mat.Shininess = 32.0
	}
	return mat
}
