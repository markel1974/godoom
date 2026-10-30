package q1

import (
	"strings"

	"github.com/markel1974/godoom/mr_tech/config"
)

// ConfigureFaceMaterial assigns physically differentiated specular and detail texture properties
// based on Quake 1 texture naming conventions.
func ConfigureFaceMaterial(mat *config.Material, texName string, isSky bool) {
	if mat == nil {
		return
	}
	spec, shine, dScale, dFactor := HeuristicMaterialProperties(texName, isSky)
	mat.SpecularBoost = spec
	mat.Shininess = shine
	mat.DetailScale = dScale
	mat.DetailFactor = dFactor
}

// ConfigureThingMaterial assigns physically differentiated specular and shininess properties for Quake 1 entities/models.
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
// based on canonical Quake 1 texture naming conventions.
func HeuristicMaterialProperties(texName string, isSky bool) (float64, float64, float32, float32) {
	if isSky {
		return 0.0, 0.0, 1.0, 0.0
	}

	name := strings.TrimSpace(strings.ToLower(texName))
	if name == "" || name == "-" {
		return 0.0, 0.0, 1.0, 0.0
	}

	// Sky
	if strings.Contains(name, "sky") {
		return 0.0, 0.0, 1.0, 0.0
	}

	// Liquids (Quake liquids start with '*')
	if len(name) > 0 && name[0] == '*' {
		if strings.Contains(name, "teleport") {
			return 0.90, 110.0, 1.0, 0.0
		}
		// *water, *slime, *lava
		return 0.75, 95.0, 1.0, 0.0
	}

	// Computer terminals, high-tech lights, screens
	if strings.Contains(name, "light") || strings.Contains(name, "lite") ||
		strings.Contains(name, "comp") || strings.Contains(name, "screen") ||
		strings.Contains(name, "ctrl") || strings.Contains(name, "terminal") {
		return 0.85, 115.0, 32.0, 0.12
	}

	// Metals, Iron, Tech Base Panels, Pipes, Chains, Grates, Doors, Buttons
	if strings.Contains(name, "metal") || strings.Contains(name, "met_") ||
		strings.Contains(name, "iron") || strings.Contains(name, "plat") ||
		strings.Contains(name, "door") || strings.Contains(name, "gate") ||
		strings.Contains(name, "pipe") || strings.Contains(name, "chain") ||
		strings.Contains(name, "grate") || strings.Contains(name, "c_met") ||
		strings.Contains(name, "w_met") || strings.Contains(name, "floor_met") ||
		strings.Contains(name, "base") || strings.Contains(name, "tech") ||
		strings.Contains(name, "slip") {
		return 0.68, 78.0, 22.0, 0.22
	}

	// Polished Marble, Runic Altars
	if strings.Contains(name, "marb") || strings.Contains(name, "altar") ||
		strings.Contains(name, "runic") {
		return 0.35, 48.0, 18.0, 0.18
	}

	// Flesh, Bones, Blood and Cthulhu organic surfaces
	if strings.Contains(name, "flesh") || strings.Contains(name, "bone") ||
		strings.Contains(name, "rib") || strings.Contains(name, "blood") ||
		strings.Contains(name, "skin") {
		return 0.32, 28.0, 14.0, 0.30
	}

	// Wood, Planks, Crates
	if strings.Contains(name, "wood") || strings.Contains(name, "plank") ||
		strings.Contains(name, "crate") || strings.Contains(name, "box") {
		return 0.12, 16.0, 16.0, 0.26
	}

	// Gothic Stone, Castle Bricks, Crypt, Ruins, Cliff, Rock, Steps, Masonry
	if strings.Contains(name, "brick") || strings.Contains(name, "stone") ||
		strings.Contains(name, "rock") || strings.Contains(name, "wall") ||
		strings.Contains(name, "ground") || strings.Contains(name, "floor") ||
		strings.Contains(name, "cliff") || strings.Contains(name, "ruin") ||
		strings.Contains(name, "crypt") || strings.Contains(name, "step") ||
		strings.Contains(name, "arch") || strings.Contains(name, "pillar") ||
		strings.Contains(name, "mon") || strings.Contains(name, "ceil") ||
		strings.Contains(name, "tomb") || strings.Contains(name, "c_run") ||
		strings.Contains(name, "r_wall") {
		return 0.05, 10.0, 16.0, 0.38
	}

	// Default generic fallback
	return 0.15, 30.0, 16.0, 0.25
}

// HeuristicThingMaterialProperties determines specular properties for Quake 1 entities and items.
func HeuristicThingMaterialProperties(classname string) (float64, float64) {
	c := strings.ToLower(classname)

	// Metallic Armor & Items
	if strings.Contains(c, "armor") || strings.Contains(c, "key") ||
		strings.Contains(c, "suit") || strings.Contains(c, "quaddama") ||
		strings.Contains(c, "invulner") {
		return 0.80, 85.0
	}

	// Metallic Weapons
	if strings.HasPrefix(c, "weapon_") || strings.Contains(c, "spike") {
		return 0.70, 75.0
	}

	// Armored / Mechanical monsters
	if c == "monster_enforcer" || c == "monster_knight" || c == "monster_hell_knight" {
		return 0.50, 58.0
	}

	// Organic monsters (Shambler, Ogre, Fiend, Scrag, Vore, Zombie, Dog, Fish, Bosses)
	return 0.25, 25.0
}
