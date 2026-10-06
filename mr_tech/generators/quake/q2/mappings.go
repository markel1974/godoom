package q2

import (
	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/generators/quake/lumps"
)

var _q2DictModelFilename = map[string]string{

	"monster_berserk": "models/monsters/berserk/tris.md2",

	"monster_gladiator": "models/monsters/gladiatr/tris.md2",

	"monster_gunner": "models/monsters/gunner/tris.md2",

	"monster_infantry": "models/monsters/infantry/tris.md2",

	"monster_soldier": "models/monsters/soldier/tris.md2",

	"monster_soldier_light": "models/monsters/soldier/tris.md2",

	"monster_soldier_ss": "models/monsters/soldier/tris.md2",

	"monster_tank": "models/monsters/tank/tris.md2",

	"monster_tank_commander": "models/monsters/tank/tris.md2",

	"monster_medic": "models/monsters/medic/tris.md2",

	"monster_flipper": "models/monsters/flipper/tris.md2",

	"monster_chick": "models/monsters/bitch/tris.md2",

	"monster_parasite": "models/monsters/parasite/tris.md2",

	"monster_flyer": "models/monsters/flyer/tris.md2",

	"monster_brain": "models/monsters/brain/tris.md2",

	"monster_floater": "models/monsters/floater/tris.md2",

	"monster_hover": "models/monsters/hover/tris.md2",

	"monster_mutant": "models/monsters/mutant/tris.md2",

	"monster_supertank": "models/monsters/boss1/tris.md2",

	"monster_boss2": "models/monsters/boss2/tris.md2",

	"monster_jorg": "models/monsters/boss2/jorg/tris.md2",

	"monster_commander_body": "models/monsters/commandr/tris.md2",

	"item_health": "models/items/healing/medium/tris.md2",

	"item_health_small": "models/items/healing/large/tris.md2",

	"item_health_large": "models/items/healing/large/tris.md2",

	"item_health_mega": "models/items/mega_h/tris.md2",

	"item_armor_shard": "models/items/armor/shard/tris.md2",

	"item_armor_jacket": "models/items/armor/jacket/tris.md2",

	"item_armor_combat": "models/items/armor/combat/tris.md2",

	"item_armor_body": "models/items/armor/body/tris.md2",

	"item_power_screen": "models/items/armor/screen/tris.md2",

	"item_power_shield": "models/items/armor/shield/tris.md2",

	"item_quad": "models/items/quaddama/tris.md2",

	"item_invulnerability": "models/items/invulner/tris.md2",

	"item_silencer": "models/items/silencer/tris.md2",

	"item_breather": "models/items/breather/tris.md2",

	"item_enviro": "models/items/enviro/tris.md2",

	"item_adrenaline": "models/items/adrenal/tris.md2",

	"item_bandolier": "models/items/bandolier/tris.md2",

	"item_pack": "models/items/pack/tris.md2",

	"weapon_shotgun": "models/weapons/g_shotgun/tris.md2",

	"weapon_supershotgun": "models/weapons/g_shotgun/tris.md2",

	"weapon_machinegun": "models/weapons/g_machinegun/tris.md2",

	"weapon_chaingun": "models/weapons/g_chaingun/tris.md2",

	"weapon_grenadelauncher": "models/weapons/g_launch/tris.md2",

	"weapon_rocketlauncher": "models/weapons/g_rocket/tris.md2",

	"weapon_hyperblaster": "models/weapons/g_hyperb/tris.md2",

	"weapon_railgun": "models/weapons/g_rail/tris.md2",

	"weapon_bfg": "models/weapons/g_bfg/tris.md2",

	"ammo_shells": "models/items/ammo/shells/tris.md2",

	"ammo_bullets": "models/items/ammo/bullets/tris.md2",

	"ammo_grenades": "models/items/ammo/grenades/tris.md2",

	"ammo_cells": "models/items/ammo/cells/tris.md2",

	"ammo_rockets": "models/items/ammo/rockets/tris.md2",

	"ammo_slugs": "models/items/ammo/slugs/tris.md2",

	"misc_deadsoldier": "models/deadbods/dude/tris.md2",

	"misc_explobox": "models/objects/barrels/tris.md2",

	"misc_banner": "models/objects/banner/tris.md2",

	"misc_satellite_dish": "models/objects/satellite/tris.md2",

	"misc_viper": "models/objects/viper/tris.md2",

	"misc_viper_bomb": "models/objects/viper_bomb/tris.md2",

	"misc_bigviper": "models/objects/bigviper/tris.md2",

	"misc_strogg_ship": "models/objects/strogg_ship/tris.md2",

	"misc_blackhole": "models/objects/blackhole/tris.md2",

	"misc_eastertank": "models/monsters/boss1/tris.md2",

	"misc_easterchick": "models/monsters/bitch/tris.md2",

	"misc_easterchick2": "models/monsters/bitch/tris.md2",

	"misc_gib_arm": "models/objects/gibs/arm/tris.md2",

	"misc_gib_leg": "models/objects/gibs/leg/tris.md2",

	"misc_gib_head": "models/objects/gibs/head/tris.md2",

	"viewthing": "models/objects/viewthing/tris.md2",
}

var _q2DictBModel = map[string]string{
	// A differenza di Q1, Q2 usa pochissimi BModel esterni per gli item, la maggior parte usa i tris.md2.
	// BModel interni (*1, *2) per func_door, func_plat saranno gestiti dinamicamente nel builder.
}

func TextInfo2ToMaterialKind(info lumps.TexInfo2) config.MaterialKind {
	animKind := config.MaterialKindLoop
	if info.IsSky() {
		animKind = config.MaterialKindSky
	} else if info.IsWarp() {
		animKind = config.MaterialKindLiquid
	}
	return animKind
}
