package wad

import (
	"github.com/markel1974/godoom/mr_tech/config"
)

// ThingDef represents the definition of a physical or graphical object with its associated properties.
type ThingDef struct {
	Sprite string
	Radius float64
	Height float64
	Speed  float64
	Mass   float64
	Kind   config.ThingType
}

const sSpeed = 800

// _spriteDictionary is a map that associates integer keys with ThingDef structures, defining game objects and their properties.
var _spriteDictionary = map[int]ThingDef{
	// --- MOSTRI ---
	3004: {Sprite: "POSS", Radius: 20.0, Height: 56.0, Mass: 100.0, Speed: sSpeed, Kind: config.ThingEnemyDef},    // Zombieman
	9:    {Sprite: "SPOS", Radius: 20.0, Height: 56.0, Mass: 100.0, Speed: sSpeed, Kind: config.ThingEnemyDef},    // Shotgun Guy
	65:   {Sprite: "CPOS", Radius: 20.0, Height: 56.0, Mass: 100.0, Speed: sSpeed, Kind: config.ThingEnemyDef},    // Heavy Weapon Dude
	3001: {Sprite: "TROO", Radius: 20.0, Height: 56.0, Mass: 100.0, Speed: sSpeed, Kind: config.ThingEnemyDef},    // Imp
	3002: {Sprite: "SARG", Radius: 30.0, Height: 56.0, Mass: 400.0, Speed: sSpeed, Kind: config.ThingEnemyDef},    // Demon
	58:   {Sprite: "SARG", Radius: 30.0, Height: 56.0, Mass: 400.0, Speed: sSpeed, Kind: config.ThingEnemyDef},    // Spectre
	3003: {Sprite: "BOSS", Radius: 24.0, Height: 64.0, Mass: 1000.0, Speed: sSpeed, Kind: config.ThingEnemyDef},   // Baron of Hell
	69:   {Sprite: "BOS2", Radius: 24.0, Height: 64.0, Mass: 1000.0, Speed: sSpeed, Kind: config.ThingEnemyDef},   // Hell Knight
	3005: {Sprite: "HEAD", Radius: 31.0, Height: 56.0, Mass: 400.0, Speed: sSpeed, Kind: config.ThingEnemyDef},    // Cacodemon
	3006: {Sprite: "SKUL", Radius: 16.0, Height: 56.0, Mass: 50.0, Speed: sSpeed, Kind: config.ThingEnemyDef},     // Lost Soul
	68:   {Sprite: "BSPI", Radius: 64.0, Height: 64.0, Mass: 600.0, Speed: sSpeed, Kind: config.ThingEnemyDef},    // Arachnotron
	71:   {Sprite: "PAIN", Radius: 31.0, Height: 56.0, Mass: 400.0, Speed: sSpeed, Kind: config.ThingEnemyDef},    // Pain Elemental
	66:   {Sprite: "SKEL", Radius: 20.0, Height: 56.0, Mass: 500.0, Speed: sSpeed, Kind: config.ThingEnemyDef},    // Revenant
	67:   {Sprite: "FATT", Radius: 48.0, Height: 64.0, Mass: 1000.0, Speed: sSpeed, Kind: config.ThingEnemyDef},   // Mancubus
	64:   {Sprite: "VILE", Radius: 20.0, Height: 56.0, Mass: 500.0, Speed: sSpeed, Kind: config.ThingEnemyDef},    // Arch-Vile
	16:   {Sprite: "CYBR", Radius: 40.0, Height: 110.0, Mass: 1000.0, Speed: sSpeed, Kind: config.ThingEnemyDef},  // Cyberdemon
	7:    {Sprite: "SPID", Radius: 130.0, Height: 100.0, Mass: 1000.0, Speed: sSpeed, Kind: config.ThingEnemyDef}, // Spider Mastermind

	// --- ARMI ---
	2001: {Sprite: "SHOT", Radius: 20.0, Height: 16.0, Mass: 4.0, Kind: config.ThingWeaponDef},  // Shotgun
	82:   {Sprite: "SGN2", Radius: 20.0, Height: 16.0, Mass: 5.0, Kind: config.ThingWeaponDef},  // Super Shotgun
	2002: {Sprite: "MGUN", Radius: 20.0, Height: 16.0, Mass: 8.0, Kind: config.ThingWeaponDef},  // Chaingun
	2003: {Sprite: "LAUN", Radius: 20.0, Height: 16.0, Mass: 12.0, Kind: config.ThingWeaponDef}, // Rocket Launcher
	2004: {Sprite: "PLAS", Radius: 20.0, Height: 16.0, Mass: 10.0, Kind: config.ThingWeaponDef}, // Plasma Rifle
	2005: {Sprite: "CSAW", Radius: 20.0, Height: 16.0, Mass: 6.0, Kind: config.ThingWeaponDef},  // Chainsaw
	2006: {Sprite: "BFUG", Radius: 20.0, Height: 16.0, Mass: 25.0, Kind: config.ThingWeaponDef}, // BFG9000

	// --- MUNIZIONI ---
	2007: {Sprite: "CLIP", Radius: 20.0, Height: 16.0, Mass: 0.5, Kind: config.ThingBulletDef},  // Ammo clip
	2048: {Sprite: "AMMO", Radius: 20.0, Height: 16.0, Mass: 2.0, Kind: config.ThingBulletDef},  // Box of Ammo
	2008: {Sprite: "SHEL", Radius: 20.0, Height: 16.0, Mass: 0.5, Kind: config.ThingBulletDef},  // 4 Shells
	2049: {Sprite: "SBOX", Radius: 20.0, Height: 16.0, Mass: 2.0, Kind: config.ThingBulletDef},  // Box of Shells
	2010: {Sprite: "ROCK", Radius: 20.0, Height: 16.0, Mass: 1.0, Kind: config.ThingBulletDef},  // 1 Rocket
	2046: {Sprite: "BROK", Radius: 20.0, Height: 16.0, Mass: 5.0, Kind: config.ThingBulletDef},  // Box of Rockets
	2047: {Sprite: "CELP", Radius: 20.0, Height: 16.0, Mass: 1.5, Kind: config.ThingBulletDef},  // Energy Cell
	17:   {Sprite: "CELP", Radius: 20.0, Height: 16.0, Mass: 5.0, Kind: config.ThingBulletDef},  // Energy Cell Pack
	8:    {Sprite: "BPAK", Radius: 20.0, Height: 16.0, Mass: 10.0, Kind: config.ThingBulletDef}, // Backpack

	// --- CURE E ARMATURE ---
	2011: {Sprite: "STIM", Radius: 20.0, Height: 16.0, Mass: 1.0},  // Stimpack
	2012: {Sprite: "MEDI", Radius: 20.0, Height: 16.0, Mass: 3.0},  // Medikit
	2014: {Sprite: "BON1", Radius: 20.0, Height: 16.0, Mass: 0.2},  // Health Bonus
	2015: {Sprite: "BON2", Radius: 20.0, Height: 16.0, Mass: 0.2},  // Armor Bonus
	2018: {Sprite: "ARM1", Radius: 20.0, Height: 16.0, Mass: 15.0}, // Green Armor
	2019: {Sprite: "ARM2", Radius: 20.0, Height: 16.0, Mass: 25.0}, // Blue Armor
	2013: {Sprite: "SOUL", Radius: 20.0, Height: 16.0, Mass: 1.0},  // Soulsphere
	83:   {Sprite: "MEGA", Radius: 20.0, Height: 16.0, Mass: 2.0},  // Megasphere

	// --- POWERUPS ---
	2022: {Sprite: "PINV", Radius: 20.0, Height: 16.0, Mass: 1.0, Kind: config.ThingItemDef}, // Invulnerability
	2023: {Sprite: "PSTR", Radius: 20.0, Height: 16.0, Mass: 1.0, Kind: config.ThingItemDef}, // Berserk
	2024: {Sprite: "PINS", Radius: 20.0, Height: 16.0, Mass: 1.0, Kind: config.ThingItemDef}, // Partial Invisibility
	2025: {Sprite: "SUIT", Radius: 20.0, Height: 16.0, Mass: 5.0, Kind: config.ThingItemDef}, // Radiation Suit
	2026: {Sprite: "PMAP", Radius: 20.0, Height: 16.0, Mass: 0.5, Kind: config.ThingItemDef}, // Computer Map
	2045: {Sprite: "PVIS", Radius: 20.0, Height: 16.0, Mass: 0.5, Kind: config.ThingItemDef}, // Light Amplification Visor

	// --- CHIAVI ---
	5:  {Sprite: "BKEY", Radius: 20.0, Height: 16.0, Mass: 0.1, Kind: config.ThingKeyDef}, // Blue Keycard
	13: {Sprite: "RKEY", Radius: 20.0, Height: 16.0, Mass: 0.1, Kind: config.ThingKeyDef}, // Red Keycard
	6:  {Sprite: "YKEY", Radius: 20.0, Height: 16.0, Mass: 0.1, Kind: config.ThingKeyDef}, // Yellow Keycard
	40: {Sprite: "BSKU", Radius: 20.0, Height: 16.0, Mass: 0.5, Kind: config.ThingKeyDef}, // Blue Skull Key
	38: {Sprite: "RSKU", Radius: 20.0, Height: 16.0, Mass: 0.5, Kind: config.ThingKeyDef}, // Red Skull Key
	39: {Sprite: "YSKU", Radius: 20.0, Height: 16.0, Mass: 0.5, Kind: config.ThingKeyDef}, // Yellow Skull Key

	// --- OSTACOLI E DECORAZIONI ---
	2035: {Sprite: "BAR1", Radius: 10.0, Height: 42.0, Mass: 100.0, Kind: config.ThingItemDef},   // Explosive Barrel
	30:   {Sprite: "COL1", Radius: 16.0, Height: 128.0, Mass: 1000.0, Kind: config.ThingItemDef}, // Tall Green Pillar
	31:   {Sprite: "COL2", Radius: 16.0, Height: 64.0, Mass: 1000.0, Kind: config.ThingItemDef},  // Short Green Pillar
	32:   {Sprite: "COL3", Radius: 16.0, Height: 128.0, Mass: 1000.0, Kind: config.ThingItemDef}, // Tall Red Pillar
	33:   {Sprite: "COL4", Radius: 16.0, Height: 64.0, Mass: 1000.0, Kind: config.ThingItemDef},  // Short Red Pillar
	41:   {Sprite: "CEYE", Radius: 16.0, Height: 54.0, Mass: 50.0, Kind: config.ThingItemDef},    // Evil Eye
	42:   {Sprite: "FSKU", Radius: 16.0, Height: 54.0, Mass: 50.0, Kind: config.ThingItemDef},    // Floating Skull
	43:   {Sprite: "TRE1", Radius: 16.0, Height: 54.0, Mass: 200.0, Kind: config.ThingItemDef},   // Burnt Tree
	47:   {Sprite: "SMIT", Radius: 16.0, Height: 64.0, Mass: 500.0, Kind: config.ThingItemDef},   // Stalagmite
	48:   {Sprite: "ELEC", Radius: 16.0, Height: 64.0, Mass: 1000.0, Kind: config.ThingItemDef},  // Tall techno pillar
	54:   {Sprite: "TRE2", Radius: 32.0, Height: 108.0, Mass: 500.0, Kind: config.ThingItemDef},  // Large brown tree
	85:   {Sprite: "TLMP", Radius: 16.0, Height: 68.0, Mass: 100.0, Kind: config.ThingItemDef},   // Tall techno lamp
	86:   {Sprite: "TLP2", Radius: 16.0, Height: 54.0, Mass: 100.0, Kind: config.ThingItemDef},   // Short techno lamp
	2028: {Sprite: "COLU", Radius: 16.0, Height: 54.0, Mass: 100.0, Kind: config.ThingItemDef},   // Floor lamp (Yellow)
	34:   {Sprite: "CAND", Radius: 20.0, Height: 16.0, Mass: 5.0, Kind: config.ThingItemDef},     // Candle
	35:   {Sprite: "CBRA", Radius: 16.0, Height: 60.0, Mass: 50.0, Kind: config.ThingItemDef},    // Candelabra
	44:   {Sprite: "TBLU", Radius: 16.0, Height: 68.0, Mass: 100.0, Kind: config.ThingItemDef},   // Tall Blue Firestick
	45:   {Sprite: "TGRN", Radius: 16.0, Height: 68.0, Mass: 100.0, Kind: config.ThingItemDef},   // Tall Green Firestick
	46:   {Sprite: "TRED", Radius: 16.0, Height: 68.0, Mass: 100.0, Kind: config.ThingItemDef},   // Tall Red Firestick
	55:   {Sprite: "SBLU", Radius: 16.0, Height: 16.0, Mass: 50.0, Kind: config.ThingItemDef},    // Short Blue Firestick
	56:   {Sprite: "SGRN", Radius: 16.0, Height: 16.0, Mass: 50.0, Kind: config.ThingItemDef},    // Short Green Firestick
	57:   {Sprite: "SRED", Radius: 16.0, Height: 16.0, Mass: 50.0, Kind: config.ThingItemDef},    // Short Red Firestick
	27:   {Sprite: "POL4", Radius: 16.0, Height: 64.0, Mass: 100.0, Kind: config.ThingItemDef},   // Skull on a pole
	28:   {Sprite: "POL2", Radius: 16.0, Height: 64.0, Mass: 100.0, Kind: config.ThingItemDef},   // Five skulls shish kebab
	29:   {Sprite: "POL3", Radius: 16.0, Height: 64.0, Mass: 100.0, Kind: config.ThingItemDef},   // Pile of skulls
	36:   {Sprite: "COL5", Radius: 16.0, Height: 64.0, Mass: 100.0, Kind: config.ThingItemDef},   // Heart column
	37:   {Sprite: "COL6", Radius: 16.0, Height: 64.0, Mass: 100.0, Kind: config.ThingItemDef},   // Red skull column

	// --- GORE E CADAVERI ---
	10: {Sprite: "PLAY", Radius: 20.0, Height: 16.0, Mass: 80.0, Kind: config.ThingItemDef},  // Bloody mess
	12: {Sprite: "PLAY", Radius: 20.0, Height: 16.0, Mass: 80.0, Kind: config.ThingItemDef},  // Bloody mess
	15: {Sprite: "PLAY", Radius: 20.0, Height: 16.0, Mass: 80.0, Kind: config.ThingItemDef},  // Bloody mess
	18: {Sprite: "POSS", Radius: 20.0, Height: 16.0, Mass: 100.0, Kind: config.ThingItemDef}, // Dead former human
	19: {Sprite: "SPOS", Radius: 20.0, Height: 16.0, Mass: 100.0, Kind: config.ThingItemDef}, // Dead former sergeant
	20: {Sprite: "TROO", Radius: 20.0, Height: 16.0, Mass: 100.0, Kind: config.ThingItemDef}, // Dead imp
	21: {Sprite: "SARG", Radius: 30.0, Height: 16.0, Mass: 400.0, Kind: config.ThingItemDef}, // Dead demon
	22: {Sprite: "HEAD", Radius: 31.0, Height: 16.0, Mass: 400.0, Kind: config.ThingItemDef}, // Dead cacodemon
	24: {Sprite: "POL5", Radius: 20.0, Height: 16.0, Mass: 5.0, Kind: config.ThingItemDef},   // Pool of blood
	79: {Sprite: "POB1", Radius: 20.0, Height: 16.0, Mass: 5.0, Kind: config.ThingItemDef},   // Pool of blood and flesh
	80: {Sprite: "POB2", Radius: 20.0, Height: 16.0, Mass: 5.0, Kind: config.ThingItemDef},   // Pool of blood
	61: {Sprite: "GOR3", Radius: 16.0, Height: 68.0, Mass: 80.0, Kind: config.ThingItemDef},  // Hanging victim, 1-legged
	62: {Sprite: "GOR5", Radius: 16.0, Height: 68.0, Mass: 20.0, Kind: config.ThingItemDef},  // Hanging leg
	73: {Sprite: "GOR1", Radius: 16.0, Height: 84.0, Mass: 80.0, Kind: config.ThingItemDef},  // Hanging victim, guts removed
	74: {Sprite: "GOR2", Radius: 16.0, Height: 84.0, Mass: 80.0, Kind: config.ThingItemDef},  // Hanging victim, guts and brain removed
	52: {Sprite: "GOR4", Radius: 16.0, Height: 68.0, Mass: 80.0, Kind: config.ThingItemDef},  // Hanging pair of legs
	60: {Sprite: "GOR2", Radius: 16.0, Height: 68.0, Mass: 80.0, Kind: config.ThingItemDef},  // Hanging victim (Non-blocking originale)
}

// _animationsBase defines a collection of grouped texture and flat animation sequences used for visual transitions.
var _animationsBase = [][]string{
	//flats
	{"NUKAGE1", "NUKAGE2", "NUKAGE3"},
	{"FWATER1", "FWATER2", "FWATER3", "FWATER4"},
	{"SWATER1", "SWATER2", "SWATER3", "SWATER4"},
	{"LAVA1", "LAVA2", "LAVA3", "LAVA4"},
	{"BLOOD1", "BLOOD2", "BLOOD3"},
	{"FIRELAVA", "FIRELAV2", "FIRELAV3"},
	{"FIREWALA", "FIREWALB", "FIREWALL"},
	//textures
	{"BLODGR1", "BLODGR2", "BLODGR3", "BLODGR4"},
	{"SLADRIP1", "SLADRIP2", "SLADRIP3"},
	{"BLODRIP1", "BLODRIP2", "BLODRIP3", "BLODRIP4"},
	{"FIREMAG1", "FIREMAG2", "FIREMAG3"},
	{"FIREBLU1", "FIREBLU2"},
	{"ROCKRED1", "ROCKRED2", "ROCKRED3"},
	{"GSTFONT1", "GSTFONT2", "GSTFONT3"},
}

// _doors is a map that associates specific action special IDs (int16) with corresponding door behaviors (string descriptions).
var _doors = map[int16]string{
	1:   "DR	Door Open, Wait, Close",
	2:   "W1	Door Stay Open",
	3:   "W1	Door Close",
	4:   "W1	Door",
	16:  "W1	Door Close and Open",
	26:  "DR	Door Blue Key",
	27:  "DR	Door Yellow Key",
	28:  "DR	Door Red Key",
	29:  "S1	Door",
	31:  "D1	Door Stay Open",
	32:  "D1	Door Blue Key",
	33:  "D1	Door Red Key",
	34:  "D1	Door Yellow Key",
	42:  "SR	Door Close",
	46:  "GR	Door Also Monsters",
	50:  "S1	Door Close",
	63:  "SR	Door",
	117: "GR	Door Wait Raise Fast",
	118: "GR	Door Wait Close Fast",
	150: "S1	Door Wait Raise",
	151: "S1	Door Close Wait Open",
	175: "S1	Door Close and Open",
	196: "SR	Door Close then Open",
	197: "SR	Door Wait Close",
	198: "SR	Door Raise",
	199: "SR	Door Wait Raise",
	200: "SR	Door Close Wait Open",
	201: "SR	Door Wait Raise Silent",
	202: "SR	Door Wait Raise Fast",
	203: "SR	Door Wait Close Fast",
	204: "SR	Door Raise Fast",
	205: "SR	Door Close Fast",
	206: "SR	Door Open Fast",
	207: "SR	Door Close Wait Open Fast",
	323: "UNK   Door Raise (fast 150)",
	324: "UNK   Door Close (fast 150)",
	325: "UNK   Door Raise (slow 300)",
	326: "UNK   Door Close (slow 300)",
	327: "UNK   Door Closest (fast 150)",
	328: "UNK   Door Closest (slow 300)",
	329: "UNK   Door Locked Raise",
	330: "UNK   Door Locked Closest",
}

func init() {
	for k, v := range _spriteDictionary {
		v.Mass /= 20.0
		v.Speed /= 1.0
		v.Radius /= 20.0
		v.Height /= 10.0
		_spriteDictionary[k] = v
	}
}
