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
	} else if strings.Contains(texNameLC, "lava") || strings.Contains(texNameLC, "magma") {
		kind = config.MaterialKindLava
	} else if strings.Contains(texNameLC, "slime") || strings.Contains(texNameLC, "acid") || strings.Contains(texNameLC, "nukage") {
		kind = config.MaterialKindSlime
	} else if strings.Contains(texNameLC, "blood") && strings.Contains(texNameLC, "liquid") {
		kind = config.MaterialKindBlood
	} else if strings.Contains(texNameLC, "water") || strings.Contains(texNameLC, "liquid") {
		kind = config.MaterialKindWater
	} else if strings.Contains(texNameLC, "common/collision") || strings.Contains(texNameLC, "common/nodraw") ||
		strings.Contains(texNameLC, "common/caulk") || strings.Contains(texNameLC, "common/trig") ||
		strings.Contains(texNameLC, "common/clip") || strings.Contains(texNameLC, "common/monster_clip") ||
		strings.Contains(texNameLC, "common/player_clip") || strings.Contains(texNameLC, "common/ik_clip") ||
		strings.Contains(texNameLC, "common/aas") || strings.Contains(texNameLC, "common/visportal") ||
		strings.Contains(texNameLC, "common/shadow") {
		kind = config.MaterialKindNone
	}
	return kind
}

// ResolveConfigMaterial creates and returns a configured `config.Material` based on the provided material definition.
func ResolveConfigMaterial(def *proc.MaterialDef) *config.Material {
	tex := []string{def.DiffuseMap}
	mat := config.NewConfigMaterial(def.Name, tex, ToMaterial(def.Kind))
	mat.DiffuseMap = def.DiffuseMap
	if def.NormalMap != "" {
		mat.NormalMap = def.NormalMap
	}
	if def.SpecularMap != "" {
		mat.SpecularMap = def.SpecularMap
	}
	ConfigureMaterial(mat, def.Name, def)
	return mat
}

// ConfigureMaterial assigns physically differentiated SpecularBoost, Shininess, DetailScale, and DetailFactor
// based on Doom 3 .mtr shader surface types and canonical texture naming conventions.
func ConfigureMaterial(mat *config.Material, matName string, def *proc.MaterialDef) {
	if mat == nil {
		return
	}
	spec, shine, dScale, dFactor := HeuristicMaterialProperties(matName, def)
	mat.SpecularBoost = spec
	mat.Shininess = shine
	mat.DetailScale = dScale
	mat.DetailFactor = dFactor
}

// ConfigureThingMaterial assigns physically differentiated SpecularBoost and Shininess for Doom 3 3D models
// (MD5 monsters/characters and LWO items/weapons/props) while setting DetailFactor to 0.0 to preserve clean model skins.
func ConfigureThingMaterial(mat *config.Material, name string, def *proc.MaterialDef) {
	if mat == nil {
		return
	}
	spec, shine := HeuristicThingMaterialProperties(name, def)
	mat.SpecularBoost = spec
	mat.Shininess = shine
	mat.DetailScale = 1.0
	mat.DetailFactor = 0.0
}

// HeuristicMaterialProperties deduces realistic physical specular and detail texture properties
// based on Doom 3 .mtr shader definitions and canonical UAC / Mars / Hell texture naming conventions.
func HeuristicMaterialProperties(matName string, def *proc.MaterialDef) (float64, float64, float32, float32) {
	name := strings.TrimSpace(strings.ToLower(matName))
	surfType := ""
	hasSpecMap := false
	hasGui := false
	if def != nil {
		if name == "" {
			name = strings.TrimSpace(strings.ToLower(def.Name))
		}
		surfType = strings.ToLower(def.SurfaceType)
		hasSpecMap = def.SpecularMap != ""
		hasGui = def.GuiSurf != ""
	}

	if name == "" || name == "-" {
		return 0.0, 0.0, 1.0, 0.0
	}

	// 1. Sky & Utility / Non-rendered surfaces
	if surfType == "sky" || strings.Contains(name, "sky") ||
		strings.Contains(name, "common/collision") || strings.Contains(name, "common/nodraw") ||
		strings.Contains(name, "common/caulk") || strings.Contains(name, "common/trig") ||
		strings.Contains(name, "common/clip") || strings.Contains(name, "common/visportal") ||
		strings.Contains(name, "common/aas") {
		return 0.0, 0.0, 1.0, 0.0
	}

	// 2. Liquids (Water, Slime, Acid, Lava, Blood pools)
	if surfType == "liquid" || surfType == "water" || surfType == "slime" || surfType == "lava" ||
		strings.Contains(name, "water") || strings.Contains(name, "liquid") ||
		strings.Contains(name, "slime") || strings.Contains(name, "acid") ||
		strings.Contains(name, "lava") || strings.Contains(name, "magma") {
		return 0.78, 95.0, 1.0, 0.0
	}

	// 3. Glass, Windows, Visors, Translucent barriers
	if surfType == "glass" || (def != nil && def.Kind == "translucent") ||
		strings.Contains(name, "glass") || strings.Contains(name, "window") ||
		strings.Contains(name, "visor") {
		return 0.92, 120.0, 24.0, 0.08
	}

	// 4. GUI Surfaces, Computer Terminals, Monitors, Base Lights, Energy / Teleport SFX
	if hasGui || strings.Contains(name, "entitygui") || strings.Contains(name, "pda_gui") ||
		strings.Contains(name, "monitor") || strings.Contains(name, "screen") ||
		strings.Contains(name, "terminal") || strings.Contains(name, "console") ||
		strings.Contains(name, "base_light") || strings.Contains(name, "light") ||
		strings.Contains(name, "lamp") || strings.Contains(name, "neon") ||
		strings.Contains(name, "glow") || strings.Contains(name, "sfx/") ||
		strings.Contains(name, "teleport") || strings.Contains(name, "portal") {
		return 0.85, 115.0, 32.0, 0.10
	}

	// 5. UAC Tech Base, Industrial Metals, Bulkheads, Doors, Trim, Pipes, Grates, Machinery
	if surfType == "metal" || surfType == "ricochet" ||
		strings.Contains(name, "base_wall") || strings.Contains(name, "base_floor") ||
		strings.Contains(name, "base_door") || strings.Contains(name, "base_trim") ||
		strings.Contains(name, "base_support") || strings.Contains(name, "mars_city") ||
		strings.Contains(name, "alphalabs") || strings.Contains(name, "enpro") ||
		strings.Contains(name, "delta") || strings.Contains(name, "cpu") ||
		strings.Contains(name, "comm") || strings.Contains(name, "recycling") ||
		strings.Contains(name, "monorail") || strings.Contains(name, "erebus") ||
		strings.Contains(name, "phobos") || strings.Contains(name, "metal") ||
		strings.Contains(name, "steel") || strings.Contains(name, "iron") ||
		strings.Contains(name, "pipe") || strings.Contains(name, "grate") ||
		strings.Contains(name, "panel") || strings.Contains(name, "vent") ||
		strings.Contains(name, "plate") || strings.Contains(name, "door") ||
		strings.Contains(name, "lift") || strings.Contains(name, "elevator") ||
		strings.Contains(name, "hangar") || strings.Contains(name, "machinery") {
		if hasSpecMap {
			return 0.72, 82.0, 22.0, 0.18
		}
		return 0.65, 72.0, 22.0, 0.20
	}

	// 6. Clean Lab Tiles, Plastic & Composite Surfaces
	if surfType == "plastic" || strings.Contains(name, "lab") ||
		strings.Contains(name, "tile") || strings.Contains(name, "medical") ||
		strings.Contains(name, "washroom") || strings.Contains(name, "plastic") {
		return 0.48, 58.0, 20.0, 0.15
	}

	// 7. Hell Biomass, Flesh, Skin, Blood, Intestines, Organic Growth
	if surfType == "flesh" || strings.Contains(name, "flesh") ||
		strings.Contains(name, "skin") || strings.Contains(name, "organic") ||
		strings.Contains(name, "biomass") || strings.Contains(name, "meat") ||
		strings.Contains(name, "gore") || strings.Contains(name, "tentacle") ||
		strings.Contains(name, "guts") || strings.Contains(name, "blood") ||
		strings.Contains(name, "corpse") || strings.Contains(name, "morgue") {
		return 0.38, 32.0, 14.0, 0.25
	}

	// 8. Wood, Crates, Cardboard
	if surfType == "wood" || surfType == "cardboard" ||
		strings.Contains(name, "wood") || strings.Contains(name, "cardboard") ||
		strings.Contains(name, "crate") || strings.Contains(name, "plank") {
		return 0.12, 16.0, 16.0, 0.24
	}

	// 9. Martian Rock, Caves, Hell Stone, Bricks, Ruins, Concrete, Dirt, Sand
	if surfType == "stone" || strings.Contains(name, "caves") ||
		strings.Contains(name, "rock") || strings.Contains(name, "dirt") ||
		strings.Contains(name, "sand") || strings.Contains(name, "stone") ||
		strings.Contains(name, "hell") || strings.Contains(name, "brick") ||
		strings.Contains(name, "concrete") || strings.Contains(name, "cement") ||
		strings.Contains(name, "ruin") || strings.Contains(name, "temple") ||
		strings.Contains(name, "cliff") || strings.Contains(name, "ground") ||
		strings.Contains(name, "ash") {
		return 0.06, 12.0, 16.0, 0.34
	}

	// 10. Default generic fallback
	if hasSpecMap {
		return 0.45, 48.0, 18.0, 0.18
	}
	return 0.18, 28.0, 16.0, 0.22
}

// HeuristicThingMaterialProperties determines specular and shininess properties for Doom 3 3D models
// (monsters, characters, weapons, ammo, items, and LWO props).
func HeuristicThingMaterialProperties(name string, def *proc.MaterialDef) (float64, float64) {
	c := strings.TrimSpace(strings.ToLower(name))
	surfType := ""
	hasSpecMap := false
	hasGui := false
	if def != nil {
		if c == "" {
			c = strings.TrimSpace(strings.ToLower(def.Name))
		}
		surfType = strings.ToLower(def.SurfaceType)
		hasSpecMap = def.SpecularMap != ""
		hasGui = def.GuiSurf != ""
	}

	// Glass, Visors, Eyes, Screens, GUI surfaces, Plasma / Energy cells
	if hasGui || surfType == "glass" || strings.Contains(c, "glass") ||
		strings.Contains(c, "visor") || strings.Contains(c, "eye") ||
		strings.Contains(c, "screen") || strings.Contains(c, "monitor") ||
		strings.Contains(c, "gui") || strings.Contains(c, "plasma") ||
		strings.Contains(c, "bfg") || strings.Contains(c, "energy") {
		return 0.90, 115.0
	}

	// Weapons, Ammo, Armor, Medkits, Canisters, Barrels, Robotic / Cybernetic units
	if strings.Contains(c, "models/weapons") || strings.HasPrefix(c, "weapon_") ||
		strings.HasPrefix(c, "ammo_") || strings.Contains(c, "armor") ||
		strings.Contains(c, "medkit") || strings.Contains(c, "canister") ||
		strings.Contains(c, "barrel") || strings.Contains(c, "sentry") ||
		strings.Contains(c, "sabaoth") || strings.Contains(c, "cyberdemon") ||
		strings.Contains(c, "tank") || strings.Contains(c, "mech") ||
		strings.Contains(c, "chrome") || strings.Contains(c, "pda") ||
		strings.Contains(c, "keycard") || strings.Contains(c, "aircan") ||
		strings.Contains(c, "backpack") {
		return 0.78, 85.0
	}

	// Industrial MapObjects & Metallic Props
	if surfType == "metal" || surfType == "ricochet" ||
		strings.Contains(c, "models/mapobjects") || strings.Contains(c, "models/items") ||
		strings.Contains(c, "metal") || strings.Contains(c, "steel") ||
		strings.Contains(c, "door") || strings.Contains(c, "lamp") ||
		strings.Contains(c, "console") || strings.Contains(c, "computer") ||
		strings.Contains(c, "machine") || strings.Contains(c, "generator") ||
		strings.Contains(c, "locker") || strings.Contains(c, "cabinet") {
		return 0.65, 72.0
	}

	// Armored / Z-Sec / Cybernetic Monsters & Marines
	if strings.Contains(c, "zsec") || strings.Contains(c, "commando") ||
		strings.Contains(c, "revenant") || strings.Contains(c, "mancubus") ||
		strings.Contains(c, "guardian") || strings.Contains(c, "vagary") ||
		strings.Contains(c, "helmet") || strings.Contains(c, "gear") ||
		strings.Contains(c, "suit") || strings.Contains(c, "marine") {
		return 0.55, 62.0
	}

	// Organic Demons, Zombies, Flesh & Skin
	if surfType == "flesh" || strings.Contains(c, "models/monsters") ||
		strings.Contains(c, "models/md5/monsters") || strings.Contains(c, "models/characters") ||
		strings.Contains(c, "models/md5/characters") || strings.Contains(c, "imp") ||
		strings.Contains(c, "pinky") || strings.Contains(c, "demon") ||
		strings.Contains(c, "hellknight") || strings.Contains(c, "archvile") ||
		strings.Contains(c, "cacodemon") || strings.Contains(c, "lostsoul") ||
		strings.Contains(c, "cherub") || strings.Contains(c, "maggot") ||
		strings.Contains(c, "trite") || strings.Contains(c, "tick") ||
		strings.Contains(c, "wraith") || strings.Contains(c, "zombie") ||
		strings.Contains(c, "fatty") || strings.Contains(c, "boney") ||
		strings.Contains(c, "morgue") || strings.Contains(c, "skin") ||
		strings.Contains(c, "flesh") || strings.Contains(c, "teeth") ||
		strings.Contains(c, "gibs") || strings.Contains(c, "blood") ||
		strings.Contains(c, "head") {
		return 0.35, 32.0
	}

	if hasSpecMap {
		return 0.45, 48.0
	}
	return 0.30, 30.0
}

// IsGUISurface checks if a material definition or material name represents a dynamic GUI surface.
func IsGUISurface(def *proc.MaterialDef, matName string) bool {
	if def != nil && def.GuiSurf != "" {
		return true
	}
	matLower := strings.ToLower(matName)
	return strings.Contains(matLower, "entitygui") || strings.Contains(matLower, "pda_gui")
}

// ResolveGUIPath determines the .gui file path for a GUI surface based on entity properties or material definition.
func ResolveGUIPath(def *proc.MaterialDef, matName string, ent *proc.MapEntity) string {
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

// ResolveGUIBackgroundForSurface returns the resolved texture/material name for a surface if it is a GUI surface.
func ResolveGUIBackgroundForSurface(arc proc.IPkArchive, materials *proc.Materials, matName string, ent *proc.MapEntity) string {
	def := materials.Resolve(matName)
	if !IsGUISurface(def, matName) {
		return ""
	}
	guiPath := ResolveGUIPath(def, matName, ent)
	if len(guiPath) > 0 && arc != nil {
		if bg := proc.ExtractGuiBackground(arc, guiPath); len(bg) > 0 {
			return bg
		}
	}
	return DefaultMonitorTexture
}

// ResolveGUIMaterial resolves and creates a config.Material for a GUI surface, or returns nil if not a GUI surface.
func ResolveGUIMaterial(arc proc.IPkArchive, materials *proc.Materials, def *proc.MaterialDef, ent *proc.MapEntity) (*config.Material, string) {
	if def == nil {
		return nil, ""
	}
	if !IsGUISurface(def, def.Name) {
		return nil, ""
	}

	bg := ResolveGUIBackgroundForSurface(arc, materials, def.Name, ent)
	if len(bg) == 0 {
		bg = DefaultMonitorTexture
	}
	bgDef := materials.Resolve(bg)
	mat := ResolveConfigMaterial(bgDef)
	mat.SpecularBoost = 0.85
	mat.Shininess = 115.0
	mat.DetailScale = 32.0
	mat.DetailFactor = 0.08
	return mat, bgDef.Name
}
