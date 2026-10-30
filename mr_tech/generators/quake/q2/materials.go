package q2

import (
	"strings"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/generators/quake/lumps"
)

// ConfigureFaceMaterial assigns physically differentiated specular and detail texture properties
// based on Quake 2 surface flags and texture naming conventions.
func ConfigureFaceMaterial(mat *config.Material, texName string, info lumps.TexInfo2) {
	if mat == nil {
		return
	}
	spec, shine, dScale, dFactor := HeuristicMaterialProperties(texName, info)
	mat.SpecularBoost = spec
	mat.Shininess = shine
	mat.DetailScale = dScale
	mat.DetailFactor = dFactor
}

// ConfigureThingMaterial assigns physically differentiated specular and shininess properties for Quake 2 entities/models.
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
// based on Quake 2 surface flags and canonical texture naming conventions.
func HeuristicMaterialProperties(texName string, info lumps.TexInfo2) (float64, float64, float32, float32) {
	// Sky
	if info.IsSky() {
		return 0.0, 0.0, 1.0, 0.0
	}

	name := strings.TrimSpace(strings.ToLower(texName))
	if name == "" || name == "-" {
		return 0.0, 0.0, 1.0, 0.0
	}

	if strings.Contains(name, "sky") {
		return 0.0, 0.0, 1.0, 0.0
	}

	// Liquids (SURF_WARP, SURF_FLOWING, or liquid names)
	if info.IsWarp() || info.IsFlowing() || (len(name) > 0 && name[0] == '*') ||
		strings.Contains(name, "water") || strings.Contains(name, "slime") ||
		strings.Contains(name, "lava") {
		if strings.Contains(name, "teleport") || strings.Contains(name, "warp") {
			return 0.90, 110.0, 1.0, 0.0
		}
		return 0.78, 95.0, 1.0, 0.0
	}

	// Glass surfaces (SURF_GLASS flag or glass name)
	if info.IsGlass() || strings.Contains(name, "glass") || strings.Contains(name, "window") {
		return 0.90, 120.0, 24.0, 0.08
	}

	// Lights & Emissive Panels (SURF_LIGHT flag or light name)
	if info.Light() || strings.Contains(name, "light") || strings.Contains(name, "lite") ||
		strings.Contains(name, "lamp") || strings.Contains(name, "glow") ||
		strings.Contains(name, "bulb") || strings.Contains(name, "neon") {
		return 0.85, 115.0, 32.0, 0.10
	}

	// Native Metal Flag
	if info.IsMetal() {
		if info.Slick() {
			return 0.85, 95.0, 22.0, 0.18
		}
		return 0.70, 78.0, 22.0, 0.22
	}

	// Computer terminals, high-tech Strogg consoles, screens
	if strings.Contains(name, "comp") || strings.Contains(name, "screen") ||
		strings.Contains(name, "ctrl") || strings.Contains(name, "panel") ||
		strings.Contains(name, "console") || strings.Contains(name, "display") ||
		strings.Contains(name, "tele") {
		return 0.78, 98.0, 28.0, 0.14
	}

	// Strogg Industrial & Base Metals, Plates, Grates, Pipes, Doors
	if strings.Contains(name, "base_wall") || strings.Contains(name, "base_floor") ||
		strings.Contains(name, "base_door") || strings.Contains(name, "base_button") ||
		strings.Contains(name, "base_trim") || strings.Contains(name, "metal") ||
		strings.Contains(name, "met_") || strings.Contains(name, "iron") ||
		strings.Contains(name, "steel") || strings.Contains(name, "plate") ||
		strings.Contains(name, "pipe") || strings.Contains(name, "wire") ||
		strings.Contains(name, "vent") || strings.Contains(name, "grate") ||
		strings.Contains(name, "chain") || strings.Contains(name, "ladder") ||
		strings.Contains(name, "beam") || strings.Contains(name, "hangar") ||
		strings.Contains(name, "c_met") || strings.Contains(name, "plat") ||
		strings.Contains(name, "door") || strings.Contains(name, "gate") ||
		strings.Contains(name, "support") || strings.Contains(name, "cargo") {
		return 0.68, 76.0, 22.0, 0.22
	}

	// Laboratory and Clean Tech Tiles
	if strings.Contains(name, "lab") || strings.Contains(name, "tile") ||
		strings.Contains(name, "clean") {
		return 0.48, 55.0, 20.0, 0.18
	}

	// Strogg Bio-Organic, Flesh, Vats, Guts
	if strings.Contains(name, "flesh") || strings.Contains(name, "bio") ||
		strings.Contains(name, "blood") || strings.Contains(name, "vat") ||
		strings.Contains(name, "meat") || strings.Contains(name, "tube") ||
		strings.Contains(name, "skin") {
		return 0.35, 32.0, 14.0, 0.28
	}

	// Natural terrain, rock, dirt, mud, sand, cliffs, caves
	if strings.Contains(name, "rock") || strings.Contains(name, "dirt") ||
		strings.Contains(name, "sand") || strings.Contains(name, "mud") ||
		strings.Contains(name, "cliff") || strings.Contains(name, "cave") ||
		strings.Contains(name, "ground") || strings.Contains(name, "earth") ||
		strings.Contains(name, "terrain") || strings.Contains(name, "grass") {
		return 0.04, 8.0, 16.0, 0.40
	}

	// Concrete, bricks, masonry, stone
	if strings.Contains(name, "brick") || strings.Contains(name, "concrete") ||
		strings.Contains(name, "cement") || strings.Contains(name, "stone") ||
		strings.Contains(name, "c_") || strings.Contains(name, "wall") ||
		strings.Contains(name, "floor") || strings.Contains(name, "ceil") {
		return 0.07, 14.0, 16.0, 0.35
	}

	// Default generic fallback
	return 0.15, 25.0, 16.0, 0.25
}

// HeuristicThingMaterialProperties determines specular properties for Quake 2 entities, items, and monsters.
func HeuristicThingMaterialProperties(classname string) (float64, float64) {
	c := strings.ToLower(classname)

	// Metallic Armor, Powerups, Ammo & Keys
	if strings.Contains(c, "armor") || strings.Contains(c, "shield") ||
		strings.Contains(c, "screen") || strings.Contains(c, "quad") ||
		strings.Contains(c, "invulner") || strings.Contains(c, "silencer") ||
		strings.Contains(c, "breather") || strings.Contains(c, "enviro") ||
		strings.Contains(c, "ammo_") || strings.Contains(c, "bandolier") ||
		strings.Contains(c, "pack") || strings.Contains(c, "adrenal") {
		return 0.82, 90.0
	}

	// Metallic Weapons
	if strings.HasPrefix(c, "weapon_") {
		return 0.75, 80.0
	}

	// Vehicles, Dishes, Turrets & Metallic Barrels
	if strings.Contains(c, "viper") || strings.Contains(c, "dish") ||
		strings.Contains(c, "ship") || strings.Contains(c, "turret") ||
		strings.Contains(c, "explobox") {
		return 0.70, 75.0
	}

	// Strogg Cyborgs & Heavily Armored Mechanical Monsters
	if strings.Contains(c, "tank") || strings.Contains(c, "gladiat") ||
		strings.Contains(c, "gunner") || strings.Contains(c, "infantry") ||
		strings.Contains(c, "soldier") || strings.Contains(c, "chick") ||
		strings.Contains(c, "flyer") || strings.Contains(c, "floater") ||
		strings.Contains(c, "hover") || strings.Contains(c, "boss") ||
		strings.Contains(c, "jorg") || strings.Contains(c, "command") {
		return 0.55, 65.0
	}

	// Organic/Mutated Monsters
	if strings.Contains(c, "berserk") || strings.Contains(c, "parasite") ||
		strings.Contains(c, "mutant") || strings.Contains(c, "brain") ||
		strings.Contains(c, "flipper") || strings.Contains(c, "medic") {
		return 0.35, 35.0
	}

	// Dead bodies, Gibs
	if strings.Contains(c, "dead") || strings.Contains(c, "gib") {
		return 0.25, 20.0
	}

	// Generic Thing fallback
	return 0.30, 30.0
}
