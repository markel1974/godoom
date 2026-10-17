package q3

import (
	"fmt"
	"strings"

	"github.com/markel1974/godoom/mr_tech/config"
)

var _q3DictModelFilename = map[string]string{
	// Weapons (Q3 usa .md3)
	"weapon_shotgun":         "models/weapons2/shotgun/shotgun.md3",
	"weapon_machinegun":      "models/weapons2/machinegun/machinegun.md3",
	"weapon_rocketlauncher":  "models/weapons2/rocketl/rocketl.md3",
	"weapon_grenadelauncher": "models/weapons2/grenadel/grenadel.md3",
	"weapon_lightning":       "models/weapons2/lightning/lightning.md3",
	"weapon_railgun":         "models/weapons2/railgun/railgun.md3",
	"weapon_plasmagun":       "models/weapons2/plasma/plasma.md3",
	"weapon_bfg":             "models/weapons2/bfg/bfg.md3",

	// Items (Armor, Health, Powerups)
	"item_armor_shard":  "models/powerups/armor/shard.md3",
	"item_armor_combat": "models/powerups/armor/armor_yel.md3", // Yellow Armor
	"item_armor_body":   "models/powerups/armor/armor_red.md3", // Red Armor

	"item_health_small": "models/powerups/health/small_cross.md3",
	"item_health":       "models/powerups/health/medium_cross.md3",
	"item_health_large": "models/powerups/health/large_cross.md3",
	"item_health_mega":  "models/powerups/health/large_cross.md3",

	// Powerups
	"holdable_teleporter": "models/powerups/holdable/teleporter.md3",
	"holdable_medkit":     "models/powerups/holdable/medkit.md3",
	"item_quad":           "models/powerups/instant/quad.md3",
	"item_enviro":         "models/powerups/instant/enviro.md3",
	"item_haste":          "models/powerups/instant/haste.md3",
	"item_invis":          "models/powerups/instant/invis.md3",
	"item_regen":          "models/powerups/instant/regen.md3",
	"item_flight":         "models/powerups/instant/flight.md3",

	// Ammo
	"ammo_shells":    "models/powerups/ammo/shotgunam.md3",
	"ammo_bullets":   "models/powerups/ammo/machinegunam.md3",
	"ammo_rockets":   "models/powerups/ammo/rocketam.md3",
	"ammo_grenades":  "models/powerups/ammo/grenadeam.md3",
	"ammo_lightning": "models/powerups/ammo/lightningam.md3",
	"ammo_slugs":     "models/powerups/ammo/railgunam.md3",
	"ammo_cells":     "models/powerups/ammo/plasmaam.md3",
	"ammo_bfg":       "models/powerups/ammo/bfgam.md3",
	"enemy_bot":      "models/players/sarge/",
}

var _q3DictBModel = map[string]string{
	// Q3 usa bmodels interni per l'architettura mobile (*1, *2) ma non modelli BSP esterni precompilati per le armi.
}

var _q3ShaderFallback = map[string]string{
	"textures/sfx/flame1side":  "textures/sfx/flame1",
	"textures/sfx/flame1_hell": "textures/sfx/flame1",

	"textures/skin/surface8_trans": "textures/skin/surface8",
	"textures/skin/tongue_trans":   "textures/skin/tongue",
	"textures/skin/skin6_trans":    "textures/skin/skin6",

	"textures/liquids/lavahell_750":     "textures/liquids/lavahell",
	"textures/liquids/lavahellflat_400": "textures/liquids/lavahell",
	"textures/liquids/lavahelldark":     "textures/liquids/lavahell",
	"textures/liquids/lavahell_1000":    "textures/liquids/lavahell",

	"textures/gothic_trim/pitted_rust2_trans":   "textures/gothic_trim/pitted_rust2",
	"textures/gothic_trim/column2c_trans":       "textures/gothic_trim/column2c",
	"textures/gothic_light/pentagram_light1_1k": "textures/gothic_light/pentagram_light1",
	"textures/gothic_light/ironcrosslt2_2000":   "textures/gothic_light/ironcrosslt2",
	"textures/gothic_light/ironcrosslt2_10000":  "textures/gothic_light/ironcrosslt2",
	"textures/gothic_light/pentagram_light1_5k": "textures/gothic_light/pentagram_light1",
	"textures/gothic_light/pentagram_light1_2k": "textures/gothic_light/pentagram_light1",
	"textures/gothic_door/km_arena1columna2r":   "textures/gothic_door/km_arena1columna2",

	"textures/base_light/light1_1500": "textures/base_light/light1",
	"textures/organics/dirt_trans":    "textures/organics/dirt",

	"textures/sfx/hellfogdense": "", // La nebbia non ha texture diffusa
	"textures/sfx/fog_intel":    "",
	"textures/sfx/flame1dark":   "textures/sfx/flame1",

	"textures/skies/tim_hell":  "textures/skies/inteldimclouds",
	"textures/skies/stars_red": "textures/skies/intelredclouds",
}

/*
0 = inteldimclouds.jpg ->
1 = intelredclouds.jpg ->
2 = killsky_1.jpg ->
3 = killsky_2.jpg ->
*/

func ConvertAnimation(r *config.EnemyAnimations, animations []string, intervals [][2]int) {
	for idx, animData := range animations {
		animData = strings.TrimSpace(strings.ToLower(animData))
		anim := strings.Split(animData, "_")
		var animation string
		if len(anim) > 1 {
			animation = anim[1]
		} else {
			animation = anim[0]
		}
		if strings.HasPrefix(animation, "(") {
			continue
		}
		if strings.HasPrefix(animation, "stand") || strings.HasPrefix(animation, "idle") {
			r.Add(config.EnemyStateIdle, idx, intervals[idx][1]-intervals[idx][0])
		} else if strings.HasPrefix(animation, "hover") {
			r.Add(config.EnemyStateIdle, idx, intervals[idx][1]-intervals[idx][0])
		} else if strings.HasPrefix(animation, "walk") {
			r.Add(config.EnemyStateWalk, idx, intervals[idx][1]-intervals[idx][0])
		} else if strings.HasPrefix(animation, "run") {
			r.Add(config.EnemyStateChase, idx, intervals[idx][1]-intervals[idx][0])
		} else if strings.HasPrefix(animation, "prowl") {
			r.Add(config.EnemyStateProwl, idx, intervals[idx][1]-intervals[idx][0])
		} else if strings.HasPrefix(animation, "jump") {
			r.Add(config.EnemyStateJump, idx, intervals[idx][1]-intervals[idx][0])
		} else if strings.HasPrefix(animation, "attack") {
			r.Add(config.EnemyStateMelee, idx, intervals[idx][1]-intervals[idx][0])
		} else if strings.HasPrefix(animation, "shoot") {
			r.Add(config.EnemyStateShoot, idx, intervals[idx][1]-intervals[idx][0])
		} else if strings.HasPrefix(animation, "smash") {
			r.Add(config.EnemyStateShoot, idx, intervals[idx][1]-intervals[idx][0]) // TODO IMPLEMENT
		} else if strings.HasPrefix(animation, "load") {
			r.Add(config.EnemyStateReload, idx, intervals[idx][1]-intervals[idx][0])
		} else if strings.HasPrefix(animation, "pain") {
			r.Add(config.EnemyStatePain, idx, intervals[idx][1]-intervals[idx][0])
		} else if strings.HasPrefix(animation, "dead") || strings.HasPrefix(animation, "death") {
			r.Add(config.EnemyStateDead, idx, intervals[idx][1]-intervals[idx][0])
		} else if strings.HasPrefix(animation, "pull") {
			r.Add(config.EnemyStatePull, idx, intervals[idx][1]-intervals[idx][0])
		} else if strings.HasPrefix(animation, "swing") {
			r.Add(config.EnemyStateSwing, idx, intervals[idx][1]-intervals[idx][0])
		} else if strings.HasPrefix(animation, "fly") {
			r.Add(config.EnemyStateFly, idx, intervals[idx][1]-intervals[idx][0])
		} else if strings.HasPrefix(animation, "duck") {
			r.Add(config.EnemyStateKneel, idx, intervals[idx][1]-intervals[idx][0])
		} else if strings.HasPrefix(animation, "swim") {
			r.Add(config.EnemyStateSwim, idx, intervals[idx][1]-intervals[idx][0])
		} else if strings.HasPrefix(animation, "land") {
			r.Add(config.EnemyStateLand, idx, intervals[idx][1]-intervals[idx][0])
		} else if strings.HasPrefix(animation, "turn") {
			r.Add(config.EnemyStateTurn, idx, intervals[idx][1]-intervals[idx][0])
		} else if strings.HasPrefix(animation, "back") {
			r.Add(config.EnemyStateBack, idx, intervals[idx][1]-intervals[idx][0])
		} else {
			fmt.Printf("Unknown animation: %s\n", animation)
		}
	}
}
