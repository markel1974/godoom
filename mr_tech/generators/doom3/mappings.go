package doom3

import (
	"strings"

	"github.com/markel1974/godoom/mr_tech/config"
)

// Doom3ThingSpec holds the default physical and rendering configuration for a Doom 3 entity type.
type Doom3ThingSpec struct {
	ModelPath string
	IdleAnim  string
	WalkAnim  string
	RunAnim   string
	Kind      config.ThingType
	Mass      float64
	Radius    float64
	Height    float64
	Speed     float64
	Health    float64
}

// ConvertAnimation maps Doom 3 animation definitions and frame intervals to Engine EnemyState values.
func ConvertAnimation(r *config.EnemyAnimations, animations []string, intervals [][2]int) {
	for idx, animName := range animations {
		frames := intervals[idx][1] - intervals[idx][0] + 1
		if frames <= 1 {
			continue
		}
		switch strings.ToLower(animName) {
		case "idle", "stand":
			r.Add(config.EnemyStateIdle, idx, frames)
		case "walk":
			r.Add(config.EnemyStateWalk, idx, frames)
		case "run":
			r.Add(config.EnemyStateRun, idx, frames)
		}
	}
}

// _doom3DictModelFilename maps Doom 3 entity classnames to their corresponding 3D model paths in PK4 archives.
var _doom3DictModelFilename = map[string]Doom3ThingSpec{
	// Monsters (MD5 animated models)
	"monster_demon_imp": {
		ModelPath: "models/md5/monsters/imp/imp.md5mesh",
		IdleAnim:  "models/md5/monsters/imp/idle1.md5anim",
		WalkAnim:  "models/md5/monsters/imp/walk1.md5anim",
		RunAnim:   "models/md5/monsters/imp/faster_walk.md5anim",
		Kind:      config.ThingEnemyDef,
		Mass:      30.0,
		Radius:    24.0,
		Height:    72.0,
		Speed:     200.0,
		Health:    130.0,
	},
	"monster_demon_pinky": {
		ModelPath: "models/md5/monsters/pinky/pinky.md5mesh",
		IdleAnim:  "models/md5/monsters/pinky/idle1.md5anim",
		WalkAnim:  "models/md5/monsters/pinky/run.md5anim",
		RunAnim:   "models/md5/monsters/pinky/run.md5anim",
		Kind:      config.ThingEnemyDef,
		Mass:      40.0,
		Radius:    36.0,
		Height:    64.0,
		Speed:     220.0,
		Health:    220.0,
	},
	"monster_zombie_fat": {
		ModelPath: "models/md5/monsters/zfat/zfat.md5mesh",
		IdleAnim:  "models/md5/monsters/zfat/idle1.md5anim",
		WalkAnim:  "models/md5/monsters/zfat/walk1.md5anim",
		RunAnim:   "models/md5/monsters/zfat/walk1.md5anim",
		Kind:      config.ThingEnemyDef,
		Mass:      35.0,
		Radius:    24.0,
		Height:    68.0,
		Speed:     120.0,
		Health:    100.0,
	},
	"monster_zombie_jumpsuit": {
		ModelPath: "models/md5/monsters/zombies/jumpsuitzombie.md5mesh",
		IdleAnim:  "models/md5/monsters/zombies/zidle1.md5anim",
		WalkAnim:  "models/md5/monsters/zombies/zwalk1.md5anim",
		RunAnim:   "models/md5/monsters/zombies/zwalk1.md5anim",
		Kind:      config.ThingEnemyDef,
		Mass:      30.0,
		Radius:    20.0,
		Height:    68.0,
		Speed:     110.0,
		Health:    90.0,
	},
	"monster_zombie_labcoat_limb": {
		ModelPath: "models/md5/monsters/zombies/labcoatzombie.md5mesh",
		IdleAnim:  "models/md5/monsters/zombies/zidle1.md5anim",
		WalkAnim:  "models/md5/monsters/zombies/zwalk1.md5anim",
		RunAnim:   "models/md5/monsters/zombies/zwalk1.md5anim",
		Kind:      config.ThingEnemyDef,
		Mass:      30.0,
		Radius:    20.0,
		Height:    68.0,
		Speed:     110.0,
		Health:    90.0,
	},
	"monster_zombie_labcoat_pipe": {
		ModelPath: "models/md5/monsters/zombies/labcoatzombie.md5mesh",
		IdleAnim:  "models/md5/monsters/zombies/zidle1.md5anim",
		WalkAnim:  "models/md5/monsters/zombies/zwalk1.md5anim",
		RunAnim:   "models/md5/monsters/zombies/zwalk1.md5anim",
		Kind:      config.ThingEnemyDef,
		Mass:      30.0,
		Radius:    20.0,
		Height:    68.0,
		Speed:     110.0,
		Health:    90.0,
	},
	"monster_zombie_suit_bloodymouth": {
		ModelPath: "models/md5/monsters/zombies/suitzombie.md5mesh",
		IdleAnim:  "models/md5/monsters/zombies/zidle1.md5anim",
		WalkAnim:  "models/md5/monsters/zombies/zwalk1.md5anim",
		RunAnim:   "models/md5/monsters/zombies/zwalk1.md5anim",
		Kind:      config.ThingEnemyDef,
		Mass:      30.0,
		Radius:    20.0,
		Height:    68.0,
		Speed:     110.0,
		Health:    90.0,
	},
	"monster_zsec_machinegun": {
		ModelPath: "models/md5/monsters/zombies/zsec_machinegun/zsecmachinegun.md5mesh",
		IdleAnim:  "models/md5/monsters/zombies/zsec_machinegun/machinegun_stand.md5anim",
		WalkAnim:  "models/md5/monsters/zombies/zsec_machinegun/machinegun_walk.md5anim",
		RunAnim:   "models/md5/monsters/zombies/zsec_machinegun/machinegun_run.md5anim",
		Kind:      config.ThingEnemyDef,
		Mass:      30.0,
		Radius:    22.0,
		Height:    70.0,
		Speed:     180.0,
		Health:    110.0,
	},
	"monster_zsec_pistol": {
		ModelPath: "models/md5/monsters/zombies/zsec_pistol/zsecpistol.md5mesh",
		IdleAnim:  "models/md5/monsters/zombies/zsec_pistol/stand.md5anim",
		WalkAnim:  "models/md5/monsters/zombies/zsec_pistol/walk2.md5anim",
		RunAnim:   "models/md5/monsters/zombies/zsec_pistol/run1.md5anim",
		Kind:      config.ThingEnemyDef,
		Mass:      30.0,
		Radius:    22.0,
		Height:    70.0,
		Speed:     180.0,
		Health:    100.0,
	},
	"monster_zsec_shotgun": {
		ModelPath: "models/md5/monsters/zombies/zsec_shotgun/zsecshotgun.md5mesh",
		IdleAnim:  "models/md5/monsters/zombies/zsec_shotgun/stand.md5anim",
		WalkAnim:  "models/md5/monsters/zombies/zsec_shotgun/walk.md5anim",
		RunAnim:   "models/md5/monsters/zombies/zsec_shotgun/run.md5anim",
		Kind:      config.ThingEnemyDef,
		Mass:      30.0,
		Radius:    22.0,
		Height:    70.0,
		Speed:     180.0,
		Health:    120.0,
	},
	"monster_demon_hellknight": {
		ModelPath: "models/md5/monsters/hellknight/hellknight.md5mesh",
		Kind:      config.ThingEnemyDef,
		Mass:      50.0,
		Radius:    40.0,
		Height:    96.0,
		Speed:     200.0,
		Health:    500.0,
	},
	"monster_demon_revenant": {
		ModelPath: "models/md5/monsters/revenant/revenant.md5mesh",
		Kind:      config.ThingEnemyDef,
		Mass:      35.0,
		Radius:    24.0,
		Height:    84.0,
		Speed:     190.0,
		Health:    270.0,
	},
	"monster_flying_cacodemon": {
		ModelPath: "models/md5/monsters/cacodemon/cacodemon.md5mesh",
		Kind:      config.ThingEnemyDef,
		Mass:      40.0,
		Radius:    32.0,
		Height:    64.0,
		Speed:     180.0,
		Health:    200.0,
	},
	"monster_flying_lostsoul": {
		ModelPath: "models/md5/monsters/lostsoul/lostsoul.md5mesh",
		Kind:      config.ThingEnemyDef,
		Mass:      20.0,
		Radius:    16.0,
		Height:    24.0,
		Speed:     240.0,
		Health:    40.0,
	},

	// Items (Armor, Medkit, PDA)
	"item_armor_security": {
		ModelPath: "models/items/armor/armor.lwo",
		Kind:      config.ThingItemDef,
		Mass:      10.0,
		Radius:    16.0,
		Height:    32.0,
	},
	"item_armor_shard": {
		ModelPath: "models/items/armor/armor_shard.lwo",
		Kind:      config.ThingItemDef,
		Mass:      5.0,
		Radius:    12.0,
		Height:    16.0,
	},
	"item_backpack": {
		ModelPath: "models/items/powerups/backpack.lwo",
		Kind:      config.ThingItemDef,
		Mass:      15.0,
		Radius:    16.0,
		Height:    24.0,
	},
	"item_medkit": {
		ModelPath: "models/items/medkit/medkit_large.lwo",
		Kind:      config.ThingItemDef,
		Mass:      10.0,
		Radius:    16.0,
		Height:    20.0,
	},
	"item_medkit_small": {
		ModelPath: "models/items/medkit/medkit_small.lwo",
		Kind:      config.ThingItemDef,
		Mass:      5.0,
		Radius:    14.0,
		Height:    16.0,
	},
	"item_pda": {
		ModelPath: "models/items/pda/pda_world.lwo",
		Kind:      config.ThingItemDef,
		Mass:      2.0,
		Radius:    10.0,
		Height:    8.0,
	},
	"item_videocd": {
		ModelPath: "models/items/videocd/videocd_world.lwo",
		Kind:      config.ThingItemDef,
		Mass:      1.0,
		Radius:    10.0,
		Height:    8.0,
	},

	// Ammo
	"ammo_belt_small": {
		ModelPath: "models/items/belt_ammo/belt.lwo",
		Kind:      config.ThingItemDef,
		Mass:      8.0,
		Radius:    14.0,
		Height:    16.0,
	},
	"ammo_clip_large": {
		ModelPath: "models/items/clip_ammo/clip_large.lwo",
		Kind:      config.ThingItemDef,
		Mass:      8.0,
		Radius:    14.0,
		Height:    16.0,
	},
	"ammo_clip_small": {
		ModelPath: "models/items/clip_ammo/clip_small.lwo",
		Kind:      config.ThingItemDef,
		Mass:      4.0,
		Radius:    12.0,
		Height:    14.0,
	},
	"ammo_shells_large": {
		ModelPath: "models/items/shells_ammo/shells_large.lwo",
		Kind:      config.ThingItemDef,
		Mass:      8.0,
		Radius:    14.0,
		Height:    16.0,
	},
	"ammo_shells_small": {
		ModelPath: "models/items/shells_ammo/shells_small.lwo",
		Kind:      config.ThingItemDef,
		Mass:      4.0,
		Radius:    12.0,
		Height:    14.0,
	},
	"ammo_rockets_large": {
		ModelPath: "models/items/rocket_ammo/rocket_large.lwo",
		Kind:      config.ThingItemDef,
		Mass:      12.0,
		Radius:    16.0,
		Height:    20.0,
	},
	"ammo_rockets_small": {
		ModelPath: "models/items/rocket_ammo/rocket_small.lwo",
		Kind:      config.ThingItemDef,
		Mass:      6.0,
		Radius:    14.0,
		Height:    16.0,
	},
	"ammo_cells_large": {
		ModelPath: "models/items/plasma_ammo/plasma_large.lwo",
		Kind:      config.ThingItemDef,
		Mass:      10.0,
		Radius:    14.0,
		Height:    16.0,
	},
	"ammo_cells_small": {
		ModelPath: "models/items/plasma_ammo/plasma_small.lwo",
		Kind:      config.ThingItemDef,
		Mass:      5.0,
		Radius:    12.0,
		Height:    14.0,
	},
	"ammo_grenades_small": {
		ModelPath: "models/items/grenade_ammo/grenade.lwo",
		Kind:      config.ThingItemDef,
		Mass:      5.0,
		Radius:    12.0,
		Height:    14.0,
	},

	// Weapons
	"weapon_chaingun": {
		ModelPath: "models/weapons/chaingun/w_chaingun.lwo",
		Kind:      config.ThingItemDef,
		Mass:      20.0,
		Radius:    18.0,
		Height:    24.0,
	},
	"weapon_flashlight": {
		ModelPath: "models/items/flashlight/flashlight2_world.lwo",
		Kind:      config.ThingItemDef,
		Mass:      2.0,
		Radius:    10.0,
		Height:    12.0,
	},
	"weapon_handgrenade": {
		ModelPath: "models/items/grenade_ammo/grenade.lwo",
		Kind:      config.ThingItemDef,
		Mass:      3.0,
		Radius:    10.0,
		Height:    12.0,
	},
	"weapon_machinegun": {
		ModelPath: "models/weapons/machinegun/w_machinegun.lwo",
		Kind:      config.ThingItemDef,
		Mass:      15.0,
		Radius:    18.0,
		Height:    24.0,
	},
	"weapon_shotgun": {
		ModelPath: "models/weapons/shotgun/w_shotgun2.lwo",
		Kind:      config.ThingItemDef,
		Mass:      15.0,
		Radius:    18.0,
		Height:    24.0,
	},
	"weapon_pistol": {
		ModelPath: "models/weapons/pistol/w_pistol.lwo",
		Kind:      config.ThingItemDef,
		Mass:      5.0,
		Radius:    12.0,
		Height:    16.0,
	},
	"weapon_rocketlauncher": {
		ModelPath: "models/weapons/rocketlauncher/w_rocketlauncher.lwo",
		Kind:      config.ThingItemDef,
		Mass:      25.0,
		Radius:    20.0,
		Height:    28.0,
	},
	"weapon_plasmagun": {
		ModelPath: "models/weapons/plasma/w_plasma.lwo",
		Kind:      config.ThingItemDef,
		Mass:      20.0,
		Radius:    20.0,
		Height:    26.0,
	},
	"weapon_bfg": {
		ModelPath: "models/weapons/bfg/w_bfg.lwo",
		Kind:      config.ThingItemDef,
		Mass:      35.0,
		Radius:    24.0,
		Height:    32.0,
	},

	// Moveables / props
	"moveable_explodingbarrel": {
		ModelPath: "models/mapobjects/fuel_barrel/exp_barrel.lwo",
		Kind:      config.ThingItemDef,
		Mass:      30.0,
		Radius:    16.0,
		Height:    48.0,
	},
	"moveable_barrel": {
		ModelPath: "models/mapobjects/fuel_barrel/exp_barrel.lwo",
		Kind:      config.ThingItemDef,
		Mass:      30.0,
		Radius:    16.0,
		Height:    48.0,
	},
}

// GetDoom3EntitySpec resolves an entity classname or model name to its Doom3ThingSpec.
func GetDoom3EntitySpec(classname string) (Doom3ThingSpec, bool) {
	if spec, ok := _doom3DictModelFilename[classname]; ok {
		return spec, true
	}

	// Dynamic fallback based on classname prefix
	parts := strings.Split(classname, "_")
	prefix := parts[0]

	switch prefix {
	case "monster":
		return Doom3ThingSpec{
			ModelPath: "models/md5/monsters/imp/imp.md5mesh",
			Kind:      config.ThingEnemyDef,
			Mass:      100.0,
			Radius:    24.0,
			Height:    72.0,
			Speed:     180.0,
			Health:    100.0,
		}, true
	case "item":
		return Doom3ThingSpec{
			ModelPath: "models/items/medkit/medkit_large.lwo",
			Kind:      config.ThingItemDef,
			Mass:      10.0,
			Radius:    16.0,
			Height:    20.0,
		}, true
	case "ammo":
		return Doom3ThingSpec{
			ModelPath: "models/items/clip_ammo/clip_large.lwo",
			Kind:      config.ThingItemDef,
			Mass:      5.0,
			Radius:    14.0,
			Height:    16.0,
		}, true
	case "weapon":
		return Doom3ThingSpec{
			ModelPath: "models/weapons/shotgun/w_shotgun2.lwo",
			Kind:      config.ThingItemDef,
			Mass:      15.0,
			Radius:    18.0,
			Height:    24.0,
		}, true
	case "moveable":
		return Doom3ThingSpec{
			ModelPath: "models/mapobjects/fuel_barrel/exp_barrel.lwo",
			Kind:      config.ThingItemDef,
			Mass:      25.0,
			Radius:    16.0,
			Height:    40.0,
		}, true
	}

	return Doom3ThingSpec{}, false
}
