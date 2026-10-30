package q3

import (
	"strings"

	"github.com/markel1974/godoom/mr_tech/config"
)

// ConfigureMaterial assigns physically differentiated specular and detail texture properties
// based on Quake 3 shader parameters and texture naming conventions.
func ConfigureMaterial(mat *config.Material, texName string, sh *Shader) {
	if mat == nil {
		return
	}
	spec, shine, dScale, dFactor := HeuristicMaterialProperties(texName, sh)
	mat.SpecularBoost = spec
	mat.Shininess = shine
	// If a shader stage specifically defined a detail scale, keep it
	if mat.DetailScale == 1.0 {
		mat.DetailScale = dScale
	}
	mat.DetailFactor = dFactor
}

// ConfigureThingMaterial assigns physically differentiated specular and shininess properties for Quake 3 entities/models.
func ConfigureThingMaterial(mat *config.Material, classname string) {
	if mat == nil {
		return
	}
	spec, shine := HeuristicThingMaterialProperties(classname)
	mat.SpecularBoost = spec
	mat.Shininess = shine
	mat.DetailFactor = 0.0 // Preserve clean original model skin texture
}

// HeuristicMaterialProperties deduces realistic physical specular and detail texture properties
// based on Quake 3 shader surface parameters and canonical texture naming conventions.
func HeuristicMaterialProperties(texName string, sh *Shader) (float64, float64, float32, float32) {
	// Sky
	if (sh != nil && sh.sky) || strings.Contains(texName, "sky") || strings.Contains(texName, "skies") {
		return 0.0, 0.0, 1.0, 0.0
	}

	name := strings.TrimSpace(strings.ToLower(texName))
	if name == "" || name == "-" {
		return 0.0, 0.0, 1.0, 0.0
	}

	// 1. SurfaceParms defined in Quake 3 shader script
	if sh != nil && len(sh.surfaceParms) > 0 {
		if sh.surfaceParms["nodraw"] {
			return 0.0, 0.0, 1.0, 0.0
		}
		if sh.surfaceParms["water"] || sh.surfaceParms["lava"] || sh.surfaceParms["slime"] || sh.surfaceParms["fog"] {
			if strings.Contains(name, "teleport") || strings.Contains(name, "warp") {
				return 0.90, 110.0, 1.0, 0.0
			}
			return 0.78, 95.0, 1.0, 0.0
		}
		if sh.surfaceParms["glass"] {
			return 0.92, 120.0, 24.0, 0.08
		}
		if sh.surfaceParms["metalsteps"] {
			if sh.surfaceParms["slick"] {
				return 0.85, 95.0, 22.0, 0.15
			}
			return 0.70, 80.0, 22.0, 0.20
		}
		if sh.surfaceParms["flesh"] {
			return 0.35, 30.0, 14.0, 0.28
		}
		if sh.surfaceParms["woodsteps"] {
			return 0.12, 16.0, 16.0, 0.26
		}
		if sh.surfaceParms["dust"] || sh.surfaceParms["sand"] {
			return 0.04, 8.0, 16.0, 0.40
		}
	}

	// 2. Texture name / path heuristics
	// Liquids
	if strings.Contains(name, "liquids") || strings.Contains(name, "water") ||
		strings.Contains(name, "lava") || strings.Contains(name, "slime") {
		return 0.78, 95.0, 1.0, 0.0
	}

	// SFX, Energy fields, Jump pads, Teleporters
	if strings.Contains(name, "sfx") || strings.Contains(name, "bounce") ||
		strings.Contains(name, "jumppad") || strings.Contains(name, "portal") ||
		strings.Contains(name, "beam") || strings.Contains(name, "teleport") {
		return 0.85, 110.0, 28.0, 0.12
	}

	// Lights & Emissive Panels
	if strings.Contains(name, "light") || strings.Contains(name, "lamp") ||
		strings.Contains(name, "glow") || strings.Contains(name, "neon") {
		return 0.85, 115.0, 32.0, 0.10
	}

	// Tech Base & Industrial Metals
	if strings.Contains(name, "base_wall") || strings.Contains(name, "base_floor") ||
		strings.Contains(name, "base_door") || strings.Contains(name, "base_trim") ||
		strings.Contains(name, "base_support") || strings.Contains(name, "metal") ||
		strings.Contains(name, "iron") || strings.Contains(name, "steel") ||
		strings.Contains(name, "plate") || strings.Contains(name, "pipe") ||
		strings.Contains(name, "wire") || strings.Contains(name, "grate") ||
		strings.Contains(name, "panel") {
		return 0.68, 76.0, 22.0, 0.22
	}

	// Gothic Trim, Rusted Iron, Columns, Doors
	if strings.Contains(name, "gothic_trim") || strings.Contains(name, "gothic_door") ||
		strings.Contains(name, "ironcross") || strings.Contains(name, "rust") ||
		strings.Contains(name, "column") || strings.Contains(name, "pillar") {
		return 0.55, 60.0, 20.0, 0.24
	}

	// Polished Marble, Shrines & Altars
	if strings.Contains(name, "marb") || strings.Contains(name, "shrine") ||
		strings.Contains(name, "altar") {
		return 0.38, 52.0, 18.0, 0.18
	}

	// Gothic Walls, Floors, Bricks, Castles, Blocks
	if strings.Contains(name, "gothic_wall") || strings.Contains(name, "gothic_floor") ||
		strings.Contains(name, "gothic_block") || strings.Contains(name, "castle") ||
		strings.Contains(name, "brick") || strings.Contains(name, "stone") ||
		strings.Contains(name, "rock") || strings.Contains(name, "masonry") ||
		strings.Contains(name, "arch") || strings.Contains(name, "step") {
		return 0.05, 10.0, 16.0, 0.38
	}

	// Organics & Skin
	if strings.Contains(name, "organics") || strings.Contains(name, "skin") ||
		strings.Contains(name, "flesh") || strings.Contains(name, "tongue") ||
		strings.Contains(name, "blood") {
		return 0.35, 30.0, 14.0, 0.28
	}

	// Terrain, Dirt, Sand, Mud
	if strings.Contains(name, "terrain") || strings.Contains(name, "dirt") ||
		strings.Contains(name, "sand") || strings.Contains(name, "mud") ||
		strings.Contains(name, "gravel") {
		return 0.04, 8.0, 16.0, 0.40
	}

	// Default generic fallback
	return 0.15, 25.0, 16.0, 0.25
}

// HeuristicThingMaterialProperties determines specular properties for Quake 3 entities, items, and player models.
func HeuristicThingMaterialProperties(classname string) (float64, float64) {
	c := strings.ToLower(classname)

	// Metallic Armor & Powerups
	if strings.Contains(c, "armor") || strings.Contains(c, "quad") ||
		strings.Contains(c, "enviro") || strings.Contains(c, "haste") ||
		strings.Contains(c, "invis") || strings.Contains(c, "regen") ||
		strings.Contains(c, "flight") || strings.Contains(c, "teleporter") ||
		strings.Contains(c, "ammo") {
		return 0.82, 95.0
	}

	// Metallic Weapons
	if strings.HasPrefix(c, "weapon_") {
		return 0.78, 85.0
	}

	// Health pickups
	if strings.Contains(c, "health") || strings.Contains(c, "medkit") {
		return 0.50, 60.0
	}

	// Bot and player models (Sarge, Visor, etc.)
	if strings.Contains(c, "bot") || strings.Contains(c, "player") {
		return 0.45, 50.0
	}

	// Generic Thing fallback
	return 0.35, 40.0
}
