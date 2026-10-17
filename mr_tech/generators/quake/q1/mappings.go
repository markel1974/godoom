package q1

import (
	"fmt"
	"strings"

	"github.com/markel1974/godoom/mr_tech/config"
)

var _q1DictModelFilename = map[string]string{
	// Monsters
	"monster_army":        "progs/soldier.mdl", // Grunt
	"monster_enforcer":    "progs/enforcer.mdl",
	"monster_ogre":        "progs/ogre.mdl",
	"monster_demon1":      "progs/demon.mdl", // Fiend
	"monster_shambler":    "progs/shambler.mdl",
	"monster_knight":      "progs/knight.mdl",
	"monster_hell_knight": "progs/hknight.mdl", // Death Knight
	"monster_zombie":      "progs/zombie.mdl",
	"monster_dog":         "progs/dog.mdl",      // Rottweiler
	"monster_wizard":      "progs/wizard.mdl",   // Scrag
	"monster_tarbaby":     "progs/tarbaby.mdl",  // Spawn
	"monster_shalrath":    "progs/shalrath.mdl", // Vore
	"monster_fish":        "progs/fish.mdl",     // Rotfish
	"monster_boss":        "progs/boss.mdl",     // Chthon
	"monster_oldone":      "progs/oldone.mdl",   // Shub-Niggurath

	// Items / Pickups
	"item_armor1":                   "progs/armor.mdl",    // Green Armor
	"item_armor2":                   "progs/armor.mdl",    // Yellow Armor (usa skin diversa)
	"item_armorInv":                 "progs/armor.mdl",    // Red Armor (usa skin diversa)
	"item_artifact_super_damage":    "progs/quaddama.mdl", // Quad Damage
	"item_artifact_invulnerability": "progs/invulner.mdl", // Pentagram of Protection
	"item_artifact_invisibility":    "progs/invisibl.mdl", // Ring of Shadows
	"item_artifact_envirosuit":      "progs/suit.mdl",     // Biosuit

	// Ammo Drops (che usano .mdl invece di .bsp in certi casi o versioni)
	"item_spikes": "progs/w_spike.mdl", // Chiodi per la Nailgun (drop dal nemico)

	// Weapons
	"weapon_shotgun":         "progs/g_shot.mdl",
	"weapon_supershotgun":    "progs/g_shot.mdl", // Il drop usa lo stesso modello
	"weapon_nailgun":         "progs/g_nail.mdl",
	"weapon_supernailgun":    "progs/g_nail2.mdl",
	"weapon_grenadelauncher": "progs/g_rock.mdl",
	"weapon_rocketlauncher":  "progs/g_rock2.mdl",
	"weapon_lightning":       "progs/g_light.mdl",

	"item_key1": "progs/w_s_key.mdl", // Silver Key (a volte)
	"item_key2": "progs/w_g_key.mdl", // Gold Key (a volte)

	"trap_spikeshooter": "progs/spike.mdl",   // Spara chiodi normali
	"trap_shooter":      "progs/spike.mdl",   // Variante che spara chiodi
	"spike":             "progs/spike.mdl",   // Il chiodo vero e proprio
	"s_spike":           "progs/s_spike.mdl", // Super spike (usato dalla Super Nailgun)
	"laser":             "progs/laser.mdl",   // Laser (usato dagli Enforcer)
	"grenade":           "progs/grenade.mdl", // Granata (dal lanciagranate o dagli Ogre)
	"rocket":            "progs/missile.mdl", // Razzo
}

var _q1DictBModel = map[string]string{
	// Health
	"item_health":       "maps/b_bh25.bsp",  // Medkit standard (25hp)
	"item_health_large": "maps/b_bh100.bsp", // Megahealth (100hp) - *Potrebbe anche essere un b_bh10.bsp per il piccolo, controlla i source di Q1*

	// Ammo Boxes (Queste sono geometrie BSP)
	"item_spikes":        "maps/b_nail0.bsp",  // Casse di chiodi
	"item_spikes_large":  "maps/b_nail1.bsp",  // Cassa grande
	"item_shells":        "maps/b_shell0.bsp", // Scatole di cartucce
	"item_shells_large":  "maps/b_shell1.bsp", // Cassa grande
	"item_rockets":       "maps/b_rock0.bsp",  // Razzi
	"item_rockets_large": "maps/b_rock1.bsp",  // Cassa grande
	"item_cells":         "maps/b_batt0.bsp",  // Celle di energia
	"item_cells_large":   "maps/b_batt1.bsp",  // Cassa grande

	// Keys (In Quake 1 vanilla, le chiavi sono BModels)
	"item_key1": "maps/b_silkey.bsp", // Silver Key
	"item_key2": "maps/b_gokey.bsp",  // Gold Key

	// Misc
	"misc_explobox":  "maps/b_explob.bsp", // Cassa esplosiva
	"misc_explobox2": "maps/b_exbox2.bsp", // Cassa esplosiva grande
}

// GetModelFileName returns the file name of a model associated with the given classname from the predefined mapping.
func GetModelFileName(classname string) string {
	return _q1DictModelFilename[classname]
}

// GetExternalBModelFileName retrieves the file name of the external BSP model corresponding to the provided classname.
func GetExternalBModelFileName(classname string) string {
	return _q1DictBModel[classname]
}

func ConvertAnimation(r *config.EnemyAnimations, animations []string, intervals [][2]int) {
	for idx, animation := range animations {
		animation = strings.TrimSpace(strings.ToLower(animation))
		if strings.HasPrefix(animation, "stand") {
			r.Add(config.EnemyStateIdle, idx, intervals[idx][1]-intervals[idx][0])
		} else if strings.HasPrefix(animation, "hover") {
			r.Add(config.EnemyStateIdle, idx, intervals[idx][1]-intervals[idx][0])
		} else if strings.HasPrefix(animation, "walk") {
			r.Add(config.EnemyStateWalk, idx, intervals[idx][1]-intervals[idx][0])
		} else if strings.HasPrefix(animation, "run") {
			r.Add(config.EnemyStateChase, idx, intervals[idx][1]-intervals[idx][0])
		} else if strings.HasPrefix(animation, "prowl") {
			r.Add(config.EnemyStateProwl, idx, intervals[idx][1]-intervals[idx][0])
		} else if strings.HasPrefix(animation, "leap") {
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
		} else if strings.HasPrefix(animation, "death") {
			r.Add(config.EnemyStateDead, idx, intervals[idx][1]-intervals[idx][0])
		} else if strings.HasPrefix(animation, "bdeath") {
			r.Add(config.EnemyStateDead, idx, intervals[idx][1]-intervals[idx][0])
		} else if strings.HasPrefix(animation, "pull") {
			r.Add(config.EnemyStatePull, idx, intervals[idx][1]-intervals[idx][0])
		} else if strings.HasPrefix(animation, "swing") {
			r.Add(config.EnemyStateSwing, idx, intervals[idx][1]-intervals[idx][0])
		} else if strings.HasPrefix(animation, "fly") {
			r.Add(config.EnemyStateFly, idx, intervals[idx][1]-intervals[idx][0])
		} else if strings.HasPrefix(animation, "kneel") {
			r.Add(config.EnemyStateKneel, idx, intervals[idx][1]-intervals[idx][0])
		} else {
			fmt.Printf("Unknown animation: %s\n", animation)
		}
	}
	if len(r.Content[config.EnemyStateIdle].Indices) == 0 {
		fmt.Println("ERROR")
		//r.Add(config.EnemyStateIdle, 0, 0)
	}
}
