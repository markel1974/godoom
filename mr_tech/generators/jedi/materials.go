package jedi

import (
	"strings"

	"github.com/markel1974/godoom/mr_tech/config"
)

// ConfigureMaterial assigns physically differentiated specular and detail texture properties
// based on Jedi engine texture names, engine mode (Dark Forces vs Outlaws), and surface properties.
func ConfigureMaterial(mat *config.Material, texName string, isSky bool, isLiquid bool, eMode EngineMode) {
	if mat == nil {
		return
	}
	spec, shine, dScale, dFactor := HeuristicMaterialProperties(texName, isSky, isLiquid, eMode)
	mat.SpecularBoost = spec
	mat.Shininess = shine
	mat.DetailScale = dScale
	mat.DetailFactor = dFactor
}

// ConfigureThingMaterial assigns physical specular properties to entities, sprites, and 3DO models.
func ConfigureThingMaterial(mat *config.Material, name string, eMode EngineMode) {
	if mat == nil {
		return
	}
	spec, shine := HeuristicThingMaterialProperties(name, eMode)
	mat.SpecularBoost = spec
	mat.Shininess = shine
	mat.DetailFactor = 0.0 // Keep sprite/model textures crisp
}

// HeuristicMaterialProperties deduces realistic physical specular and detail texture properties
// based on Jedi engine texture naming conventions, specialized for Outlaws and Dark Forces.
func HeuristicMaterialProperties(texName string, isSky bool, isLiquid bool, eMode EngineMode) (float64, float64, float32, float32) {
	name := strings.ToUpper(strings.TrimSpace(texName))

	// Sky & Parallaxing surfaces
	if isSky || strings.Contains(name, "SKY") || strings.Contains(name, "STAR") ||
		strings.Contains(name, "SPACE") || strings.Contains(name, "CLOUD") ||
		strings.Contains(name, "SUN") {
		return 0.0, 0.0, 1.0, 0.0
	}

	// ==================== OUTLAWS MODE ====================
	if eMode == EngineModeOutlaws {
		// Liquids: Creeks, rivers, water troughs, muddy puddles
		if isLiquid || strings.Contains(name, "WATER") || strings.Contains(name, "CREEK") ||
			strings.Contains(name, "STREAM") || strings.Contains(name, "RIVER") ||
			strings.Contains(name, "TROUGH") || strings.Contains(name, "PUDDLE") ||
			strings.Contains(name, "WFALL") {
			return 0.78, 90.0, 1.0, 0.0
		}

		// Glass: Saloon windows, oil lanterns, mirrors, whiskey bottles
		if strings.Contains(name, "GLASS") || strings.Contains(name, "WINDOW") ||
			strings.Contains(name, "MIRROR") || strings.Contains(name, "LANTERN") ||
			strings.Contains(name, "BOTTLE") {
			return 0.90, 120.0, 24.0, 0.08
		}

		// Wrought iron, cast iron stoves, bank safes, jail bars, railroad tracks, tin roofs, steam engines
		if strings.Contains(name, "IRON") || strings.Contains(name, "STOVE") ||
			strings.Contains(name, "VAULT") || strings.Contains(name, "SAFE") ||
			strings.Contains(name, "BARS") || strings.Contains(name, "TIN") ||
			strings.Contains(name, "ROOF") || strings.Contains(name, "METAL") ||
			strings.Contains(name, "PIPE") || strings.Contains(name, "WHEEL") ||
			strings.Contains(name, "ENGINE") || strings.Contains(name, "RAIL") ||
			strings.Contains(name, "TRACK") || strings.Contains(name, "STEEL") {
			return 0.60, 65.0, 20.0, 0.25
		}

		// Polished saloon bar tops, brass rails, hotel counters, marble
		if strings.Contains(name, "BAR") || strings.Contains(name, "COUNTER") ||
			strings.Contains(name, "POLISH") || strings.Contains(name, "SHINY") ||
			strings.Contains(name, "MARB") || strings.Contains(name, "BRASS") {
			return 0.38, 48.0, 18.0, 0.18
		}

		// Saloon carpets, wallpaper, curtains, fabrics
		if strings.Contains(name, "CARPET") || strings.Contains(name, "PAPER") ||
			strings.Contains(name, "WALLPAPER") || strings.Contains(name, "RUG") ||
			strings.Contains(name, "FABRIC") || strings.Contains(name, "CURTAIN") {
			return 0.15, 20.0, 22.0, 0.20
		}

		// Weathered timber, barns, saloon siding, mine shafts, fences, crates, wooden planks
		if strings.Contains(name, "WOOD") || strings.Contains(name, "BARN") ||
			strings.Contains(name, "SALOON") || strings.Contains(name, "PLANK") ||
			strings.Contains(name, "CRATE") || strings.Contains(name, "BOX") ||
			strings.Contains(name, "FENCE") || strings.Contains(name, "CABIN") ||
			strings.Contains(name, "BOARD") || strings.Contains(name, "MINE") ||
			strings.Contains(name, "STABLE") || strings.Contains(name, "POST") ||
			strings.Contains(name, "HITCH") || strings.Contains(name, "DESK") ||
			strings.Contains(name, "LOG") || strings.Contains(name, "DOOR") {
			return 0.08, 12.0, 18.0, 0.32
		}

		// Adobe, clay, sun-baked desert earth, red rock canyons, mesas, sand, dust
		if strings.Contains(name, "ADOBE") || strings.Contains(name, "STUCCO") ||
			strings.Contains(name, "CLAY") || strings.Contains(name, "BRICK") ||
			strings.Contains(name, "CANYON") || strings.Contains(name, "CLIFF") ||
			strings.Contains(name, "MESA") || strings.Contains(name, "BUTTE") ||
			strings.Contains(name, "DESERT") || strings.Contains(name, "SAND") ||
			strings.Contains(name, "DUST") || strings.Contains(name, "TRAIL") ||
			strings.Contains(name, "DIRT") || strings.Contains(name, "ROCK") ||
			strings.Contains(name, "CAVE") || strings.Contains(name, "GROUND") {
			return 0.03, 6.0, 16.0, 0.42
		}

		// Default Outlaws fallback
		return 0.10, 16.0, 16.0, 0.30
	}

	// ==================== DARK FORCES MODE ====================
	// Liquids (Water, Sewage, Slime, Lava, Acid, Waterfalls)
	if isLiquid || strings.Contains(name, "WATER") || strings.Contains(name, "EXWAT") ||
		strings.Contains(name, "LAVA") || strings.Contains(name, "SLIME") ||
		strings.Contains(name, "SEWER") || strings.Contains(name, "ACID") ||
		strings.Contains(name, "WFALL") || strings.Contains(name, "POOL") ||
		strings.Contains(name, "RIVER") || strings.Contains(name, "STREAM") {
		return 0.80, 95.0, 1.0, 0.0
	}

	// Glass, Windows, Forcefields, Energy shields, Laser beams
	if strings.Contains(name, "GLASS") || strings.Contains(name, "WINDOW") ||
		strings.Contains(name, "SHIELD") || strings.Contains(name, "FORCE") ||
		strings.Contains(name, "ENERGY") || strings.Contains(name, "BEAM") {
		return 0.90, 120.0, 24.0, 0.08
	}

	// Computer terminals, Control panels, Screens, Monitors, Lights, Switches
	if strings.Contains(name, "COMP") || strings.Contains(name, "CTRL") ||
		strings.Contains(name, "SCREEN") || strings.Contains(name, "DISP") ||
		strings.Contains(name, "MONITOR") || strings.Contains(name, "PANEL") ||
		strings.Contains(name, "LIGHT") || strings.Contains(name, "LITE") ||
		strings.Contains(name, "LAMP") || strings.Contains(name, "GLOW") ||
		strings.Contains(name, "NEON") || strings.Contains(name, "SWITCH") {
		return 0.82, 105.0, 30.0, 0.12
	}

	// Imperial Base Metal, Steel bulkheads, Elevators, Pipes, Grates, Vents, Hangar Docks
	if strings.Contains(name, "MET") || strings.Contains(name, "MTL") ||
		strings.Contains(name, "STEEL") || strings.Contains(name, "IRON") ||
		strings.Contains(name, "DOOR") || strings.Contains(name, "ELEV") ||
		strings.Contains(name, "LIFT") || strings.Contains(name, "PIPE") ||
		strings.Contains(name, "GRATE") || strings.Contains(name, "VENT") ||
		strings.Contains(name, "HULL") || strings.Contains(name, "DOCK") ||
		strings.Contains(name, "BAY") || strings.Contains(name, "SHUTTLE") ||
		strings.Contains(name, "BARS") || strings.Contains(name, "SAFE") ||
		strings.Contains(name, "TIN") || strings.Contains(name, "RAIL") {
		return 0.68, 78.0, 22.0, 0.22
	}

	// Polished Marble & Fine Tiles
	if strings.Contains(name, "MARB") || strings.Contains(name, "TILE") ||
		strings.Contains(name, "SHINY") || strings.Contains(name, "POLISH") {
		return 0.40, 50.0, 18.0, 0.18
	}

	// Wood, Crates, Fences, Planks
	if strings.Contains(name, "WOOD") || strings.Contains(name, "PLANK") ||
		strings.Contains(name, "CRATE") || strings.Contains(name, "BOX") ||
		strings.Contains(name, "FENCE") || strings.Contains(name, "CABIN") ||
		strings.Contains(name, "BOARD") || strings.Contains(name, "DESK") ||
		strings.Contains(name, "LOG") {
		return 0.12, 16.0, 16.0, 0.26
	}

	// Stone, Brick, Stucco, Cement
	if strings.Contains(name, "BRICK") || strings.Contains(name, "STONE") ||
		strings.Contains(name, "CEMENT") || strings.Contains(name, "CONC") ||
		strings.Contains(name, "MASON") {
		return 0.06, 12.0, 16.0, 0.36
	}

	// Natural Terrain, Canyons, Desert dirt, Sand, Cliffs, Caves, Mud
	if strings.Contains(name, "CANYON") || strings.Contains(name, "CLIFF") ||
		strings.Contains(name, "ROCK") || strings.Contains(name, "DIRT") ||
		strings.Contains(name, "SAND") || strings.Contains(name, "CAVE") ||
		strings.Contains(name, "GROUND") || strings.Contains(name, "MUD") ||
		strings.Contains(name, "EARTH") {
		return 0.04, 8.0, 16.0, 0.40
	}

	// Default fallback
	return 0.15, 25.0, 16.0, 0.25
}

// HeuristicThingMaterialProperties assigns physical specular properties to entities, sprites, and 3DO models.
func HeuristicThingMaterialProperties(name string, eMode EngineMode) (float64, float64) {
	n := strings.ToUpper(strings.TrimSpace(name))

	if eMode == EngineModeOutlaws {
		// Outlaws weapons (blued steel, polished brass revolvers and rifles)
		if strings.Contains(n, "GUN") || strings.Contains(n, "RIFLE") ||
			strings.Contains(n, "SHOTGUN") || strings.Contains(n, "PISTOL") ||
			strings.Contains(n, "COLT") || strings.Contains(n, "WINCHESTER") ||
			strings.Contains(n, "GATLING") || strings.Contains(n, "SAWED") {
			return 0.78, 85.0
		}
		// Outlaws badges, gold coins, keys, ammo, dynamite, brass buckles
		if strings.Contains(n, "BADGE") || strings.Contains(n, "GOLD") ||
			strings.Contains(n, "SILVER") || strings.Contains(n, "KEY") ||
			strings.Contains(n, "AMMO") || strings.Contains(n, "BULLET") ||
			strings.Contains(n, "DYNAMITE") || strings.Contains(n, "CROWBAR") {
			return 0.82, 90.0
		}
		// 3DO vehicles & props (locomotives, stagecoaches, wagons, windmills, lanterns, iron stoves)
		if strings.Contains(n, "TRAIN") || strings.Contains(n, "STAGE") ||
			strings.Contains(n, "WAGON") || strings.Contains(n, "MILL") ||
			strings.Contains(n, "LANTERN") || strings.Contains(n, "BARREL") ||
			strings.Contains(n, "STOVE") || strings.Contains(n, "SAFE") {
			return 0.60, 65.0
		}
		// Outlaw characters (James Anderson, bandits, outlaws, townspeople)
		if strings.Contains(n, "SLIM") || strings.Contains(n, "JACK") ||
			strings.Contains(n, "FARMER") || strings.Contains(n, "SANCHEZ") ||
			strings.Contains(n, "GRAHAM") || strings.Contains(n, "HENCH") ||
			strings.Contains(n, "BANDIT") || strings.Contains(n, "SHERIFF") ||
			strings.Contains(n, "GIRL") || strings.Contains(n, "BOB") ||
			strings.Contains(n, "DOC") {
			return 0.22, 18.0
		}
		return 0.25, 20.0
	}

	// Dark Forces (Star Wars)
	// Dark Trooper metallic exoskeleton
	if strings.Contains(n, "DTROOP") || strings.Contains(n, "DARKTROOP") ||
		strings.Contains(n, "PHASE") {
		return 0.85, 95.0
	}
	// Stormtrooper plastoid armor
	if strings.Contains(n, "STORM") || strings.Contains(n, "TROOP") {
		return 0.65, 70.0
	}
	// Droids and 3DO starships (TIE Fighter, Shuttles, Probe Droids)
	if strings.Contains(n, "DROID") || strings.Contains(n, "PROBE") ||
		strings.Contains(n, "MOUSE") || strings.Contains(n, "TURRET") ||
		strings.Contains(n, "SHUTTLE") || strings.Contains(n, "TIEFIGHT") ||
		strings.Contains(n, "SPEEDER") {
		return 0.72, 80.0
	}
	// Blasters and thermal detonators
	if strings.Contains(n, "BLASTER") || strings.Contains(n, "GUN") ||
		strings.Contains(n, "RIFLE") || strings.Contains(n, "FUSION") ||
		strings.Contains(n, "MORTAR") || strings.Contains(n, "CONCUSS") ||
		strings.Contains(n, "DETONAT") {
		return 0.75, 80.0
	}
	// Shields, batteries, keycards
	if strings.Contains(n, "SHIELD") || strings.Contains(n, "BATTERY") ||
		strings.Contains(n, "KEY") || strings.Contains(n, "CARD") ||
		strings.Contains(n, "POWER") {
		return 0.80, 90.0
	}

	// Officers, Kyle Katarn, Gamorreans, Rodians
	return 0.28, 25.0
}
