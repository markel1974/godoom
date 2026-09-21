package lumps

var _q2DictModelFilename = map[string]string{
	// Monsters (Quake 2 usa tris.md2 in sottocartelle)
	"monster_infantry": "models/monsters/infantry/tris.md2",
	"monster_soldier":  "models/monsters/soldier/tris.md2",
	"monster_gunner":   "models/monsters/gunner/tris.md2",

	// Items
	"item_armor_jacket": "models/items/armor/jacket/tris.md2", // Green armor
	"item_armor_combat": "models/items/armor/combat/tris.md2", // Yellow armor
	"item_armor_body":   "models/items/armor/body/tris.md2",   // Red armor

	"item_health":       "models/items/healing/medium/tris.md2",
	"item_health_large": "models/items/healing/large/tris.md2",
	"item_health_mega":  "models/items/healing/mega/tris.md2",

	// Weapons
	"weapon_shotgun":         "models/weapons/g_shotg/tris.md2",
	"weapon_supershotgun":    "models/weapons/g_shotg2/tris.md2",
	"weapon_machinegun":      "models/weapons/g_machn/tris.md2",
	"weapon_chaingun":        "models/weapons/g_chain/tris.md2",
	"weapon_grenadelauncher": "models/weapons/g_launch/tris.md2",
	"weapon_rocketlauncher":  "models/weapons/g_rocket/tris.md2",
	"weapon_hyperblaster":    "models/weapons/g_hyperb/tris.md2",
	"weapon_railgun":         "models/weapons/g_rail/tris.md2",
	"weapon_bfg":             "models/weapons/g_bfg/tris.md2",

	// Ammo
	"ammo_shells":   "models/items/ammo/shells/medium/tris.md2",
	"ammo_bullets":  "models/items/ammo/bullets/medium/tris.md2",
	"ammo_cells":    "models/items/ammo/cells/medium/tris.md2",
	"ammo_grenades": "models/items/ammo/grenades/medium/tris.md2",
	"ammo_rockets":  "models/items/ammo/rockets/medium/tris.md2",
	"ammo_slugs":    "models/items/ammo/slugs/medium/tris.md2",
}

var _q2DictBModel = map[string]string{
	// A differenza di Q1, Q2 usa pochissimi BModel esterni per gli item, la maggior parte usa i tris.md2.
	// BModel interni (*1, *2) per func_door, func_plat saranno gestiti dinamicamente nel builder.
}
