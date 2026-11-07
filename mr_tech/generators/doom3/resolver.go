package doom3

import (
	"strings"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/generators/doom3/proc"
)

const DefaultMonitorTexture = "models/mapobjects/filler/monitor_d"

// ToMaterial determines the material kind based on the provided lowercase texture name and returns the corresponding MaterialKind.
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

// ResolveConfigMaterial creates and returns a configured `config.Material` based on the provided material name.
func ResolveConfigMaterial(def *proc.MaterialDef) *config.Material {
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

// IsGuiSurface checks if a material definition or material name represents a dynamic GUI surface.
func IsGuiSurface(def *proc.MaterialDef, matName string) bool {
	if def != nil && def.GuiSurf != "" {
		return true
	}
	matLower := strings.ToLower(matName)
	return strings.Contains(matLower, "entitygui") || strings.Contains(matLower, "pda_gui")
}

// ResolveGuiPath determines the .gui file path for a GUI surface based on entity properties or material definition.
func ResolveGuiPath(def *proc.MaterialDef, matName string, ent *proc.MapEntity) string {
	guiSurf := ""
	if def != nil {
		guiSurf = strings.ToLower(def.GuiSurf)
	}
	matLower := strings.ToLower(matName)

	if guiSurf == "entity3" || strings.HasSuffix(matLower, "entitygui3") {
		if ent != nil {
			if g, ok := ent.GetProperty("gui3"); ok && len(g) > 0 {
				return g
			}
		}
		return ""
	}
	if guiSurf == "entity2" || strings.HasSuffix(matLower, "entitygui2") {
		if ent != nil {
			if g, ok := ent.GetProperty("gui2"); ok && len(g) > 0 {
				return g
			}
		}
		return ""
	}
	if guiSurf != "" && guiSurf != "entity" {
		return guiSurf
	}
	if ent != nil {
		if g, ok := ent.GetProperty("gui"); ok && len(g) > 0 {
			return g
		}
		if g, ok := ent.GetProperty("gui_name"); ok && len(g) > 0 {
			return g
		}
	}
	return ""
}

// ResolveGuiBackgroundForSurface returns the resolved texture/material name for a surface if it is a GUI surface.
func ResolveGuiBackgroundForSurface(arc proc.IPkArchive, materials *proc.Materials, matName string, ent *proc.MapEntity) string {
	def := materials.Resolve(matName)
	if !IsGuiSurface(def, matName) {
		return ""
	}
	guiPath := ResolveGuiPath(def, matName, ent)
	if len(guiPath) > 0 && arc != nil {
		if bg := proc.ExtractGuiBackground(arc, guiPath); len(bg) > 0 {
			return bg
		}
	}
	return DefaultMonitorTexture
}

// ResolveGuiMaterial resolves and creates a config.Material for a GUI surface, or returns nil if not a GUI surface.
func ResolveGuiMaterial(arc proc.IPkArchive, materials *proc.Materials, def *proc.MaterialDef, ent *proc.MapEntity) (*config.Material, string) {
	if def == nil {
		return nil, ""
	}
	if !IsGuiSurface(def, def.Name) {
		return nil, ""
	}

	bg := ResolveGuiBackgroundForSurface(arc, materials, def.Name, ent)
	if len(bg) == 0 {
		bg = DefaultMonitorTexture
	}
	bgDef := materials.Resolve(bg)
	mat := ResolveConfigMaterial(bgDef)
	return mat, bgDef.Name
}
