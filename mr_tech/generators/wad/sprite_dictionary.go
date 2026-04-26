package wad

import (
	"github.com/markel1974/godoom/mr_tech/config"
)

// ThingDef represents the definition of a physical or graphical object with its associated properties.
type ThingDef struct {
	Sprites []string
	Radius  float64
	Height  float64
	Speed   float64
	Mass    float64
	Kind    config.ThingType
}

const sSpeed = 12

// _spriteDictionary is a map that associates integer keys with ThingDef structures, defining game objects and their properties.
var _spriteDictionary = map[int]ThingDef{
	// --- MOSTRI ---
	3004: {Sprites: []string{"POSSA1", "POSSB1", "POSSC1", "POSSD1"}, Radius: 20.0, Height: 56.0, Mass: 100.0, Speed: sSpeed, Kind: config.ThingEnemyDef},   // Zombieman
	9:    {Sprites: []string{"SPOSA1", "SPOSB1", "SPOSC1", "SPOSD1"}, Radius: 20.0, Height: 56.0, Mass: 100.0, Speed: sSpeed, Kind: config.ThingEnemyDef},   // Shotgun Guy
	65:   {Sprites: []string{"CPOSA1", "CPOSB1", "CPOSC1", "CPOSD1"}, Radius: 20.0, Height: 56.0, Mass: 100.0, Speed: sSpeed, Kind: config.ThingEnemyDef},   // Heavy Weapon Dude
	3001: {Sprites: []string{"TROOA1", "TROOB1", "TROOC1", "TROOD1"}, Radius: 20.0, Height: 56.0, Mass: 100.0, Speed: sSpeed, Kind: config.ThingEnemyDef},   // Imp
	3002: {Sprites: []string{"SARGA1", "SARGB1", "SARGC1", "SARGD1"}, Radius: 30.0, Height: 56.0, Mass: 400.0, Speed: sSpeed, Kind: config.ThingEnemyDef},   // Demon
	58:   {Sprites: []string{"SARGA1", "SARGB1", "SARGC1", "SARGD1"}, Radius: 30.0, Height: 56.0, Mass: 400.0, Speed: sSpeed, Kind: config.ThingEnemyDef},   // Spectre
	3003: {Sprites: []string{"BOSSA1", "BOSSB1", "BOSSC1", "BOSSD1"}, Radius: 24.0, Height: 64.0, Mass: 1000.0, Speed: sSpeed, Kind: config.ThingEnemyDef},  // Baron of Hell
	69:   {Sprites: []string{"BOS2A1", "BOS2B1", "BOS2C1", "BOS2D1"}, Radius: 24.0, Height: 64.0, Mass: 1000.0, Speed: sSpeed, Kind: config.ThingEnemyDef},  // Hell Knight
	3005: {Sprites: []string{"HEADA1", "HEADB1", "HEADC1", "HEADD1"}, Radius: 31.0, Height: 56.0, Mass: 400.0, Speed: sSpeed, Kind: config.ThingEnemyDef},   // Cacodemon
	3006: {Sprites: []string{"SKULA1", "SKULB1"}, Radius: 16.0, Height: 56.0, Mass: 50.0, Speed: sSpeed, Kind: config.ThingEnemyDef},                        // Lost Soul
	68:   {Sprites: []string{"BSPIA1", "BSPIB1", "BSPIC1"}, Radius: 64.0, Height: 64.0, Mass: 600.0, Speed: sSpeed, Kind: config.ThingEnemyDef},             // Arachnotron
	71:   {Sprites: []string{"PAINA1", "PAINB1", "PAINC1"}, Radius: 31.0, Height: 56.0, Mass: 400.0, Speed: sSpeed, Kind: config.ThingEnemyDef},             // Pain Elemental
	66:   {Sprites: []string{"SKELA1", "SKELB1", "SKELC1", "SKELD1"}, Radius: 20.0, Height: 56.0, Mass: 500.0, Speed: sSpeed, Kind: config.ThingEnemyDef},   // Revenant
	67:   {Sprites: []string{"FATTA1", "FATTB1", "FATTC1"}, Radius: 48.0, Height: 64.0, Mass: 1000.0, Speed: sSpeed, Kind: config.ThingEnemyDef},            // Mancubus
	64:   {Sprites: []string{"VILEA1", "VILEB1", "VILEC1"}, Radius: 20.0, Height: 56.0, Mass: 500.0, Speed: sSpeed, Kind: config.ThingEnemyDef},             // Arch-Vile
	16:   {Sprites: []string{"CYBRA1", "CYBRB1", "CYBRC1", "CYBRD1"}, Radius: 40.0, Height: 110.0, Mass: 1000.0, Speed: sSpeed, Kind: config.ThingEnemyDef}, // Cyberdemon
	7:    {Sprites: []string{"SPIDA1", "SPIDB1", "SPIDC1"}, Radius: 130.0, Height: 100.0, Mass: 1000.0, Speed: sSpeed, Kind: config.ThingEnemyDef},          // Spider Mastermind

	// --- ARMI ---
	2001: {Sprites: []string{"SHOTA0"}, Radius: 20.0, Height: 16.0, Mass: 4.0, Kind: config.ThingWeaponDef},  // Shotgun
	82:   {Sprites: []string{"SGN2A0"}, Radius: 20.0, Height: 16.0, Mass: 5.0, Kind: config.ThingWeaponDef},  // Super Shotgun
	2002: {Sprites: []string{"MGUNA0"}, Radius: 20.0, Height: 16.0, Mass: 8.0, Kind: config.ThingWeaponDef},  // Chaingun
	2003: {Sprites: []string{"LAUNA0"}, Radius: 20.0, Height: 16.0, Mass: 12.0, Kind: config.ThingWeaponDef}, // Rocket Launcher
	2004: {Sprites: []string{"PLASA0"}, Radius: 20.0, Height: 16.0, Mass: 10.0, Kind: config.ThingWeaponDef}, // Plasma Rifle
	2005: {Sprites: []string{"CSAWA0"}, Radius: 20.0, Height: 16.0, Mass: 6.0, Kind: config.ThingWeaponDef},  // Chainsaw
	2006: {Sprites: []string{"BFUGA0"}, Radius: 20.0, Height: 16.0, Mass: 25.0, Kind: config.ThingWeaponDef}, // BFG9000

	// --- MUNIZIONI ---
	2007: {Sprites: []string{"CLIPA0"}, Radius: 20.0, Height: 16.0, Mass: 0.5, Kind: config.ThingBulletDef},  // Ammo clip
	2048: {Sprites: []string{"AMMOA0"}, Radius: 20.0, Height: 16.0, Mass: 2.0, Kind: config.ThingBulletDef},  // Box of Ammo
	2008: {Sprites: []string{"SHELA0"}, Radius: 20.0, Height: 16.0, Mass: 0.5, Kind: config.ThingBulletDef},  // 4 Shells
	2049: {Sprites: []string{"SBOXA0"}, Radius: 20.0, Height: 16.0, Mass: 2.0, Kind: config.ThingBulletDef},  // Box of Shells
	2010: {Sprites: []string{"ROCKA0"}, Radius: 20.0, Height: 16.0, Mass: 1.0, Kind: config.ThingBulletDef},  // 1 Rocket
	2046: {Sprites: []string{"BROKA0"}, Radius: 20.0, Height: 16.0, Mass: 5.0, Kind: config.ThingBulletDef},  // Box of Rockets
	2047: {Sprites: []string{"CELPA0"}, Radius: 20.0, Height: 16.0, Mass: 1.5, Kind: config.ThingBulletDef},  // Energy Cell
	17:   {Sprites: []string{"CELPA0"}, Radius: 20.0, Height: 16.0, Mass: 5.0, Kind: config.ThingBulletDef},  // Energy Cell Pack
	8:    {Sprites: []string{"BPAKA0"}, Radius: 20.0, Height: 16.0, Mass: 10.0, Kind: config.ThingBulletDef}, // Backpack

	// --- CURE E ARMATURE ---
	2011: {Sprites: []string{"STIMA0"}, Radius: 20.0, Height: 16.0, Mass: 1.0},                               // Stimpack
	2012: {Sprites: []string{"MEDIA0"}, Radius: 20.0, Height: 16.0, Mass: 3.0},                               // Medikit
	2014: {Sprites: []string{"BON1A0", "BON1B0", "BON1C0", "BON1D0"}, Radius: 20.0, Height: 16.0, Mass: 0.2}, // Health Bonus
	2015: {Sprites: []string{"BON2A0", "BON2B0", "BON2C0", "BON2D0"}, Radius: 20.0, Height: 16.0, Mass: 0.2}, // Armor Bonus
	2018: {Sprites: []string{"ARM1A0", "ARM1B0"}, Radius: 20.0, Height: 16.0, Mass: 15.0},                    // Green Armor
	2019: {Sprites: []string{"ARM2A0", "ARM2B0"}, Radius: 20.0, Height: 16.0, Mass: 25.0},                    // Blue Armor
	2013: {Sprites: []string{"SOULA0", "SOULB0", "SOULC0", "SOULD0"}, Radius: 20.0, Height: 16.0, Mass: 1.0}, // Soulsphere
	83:   {Sprites: []string{"MEGAA0", "MEGAB0", "MEGAC0", "MEGAD0"}, Radius: 20.0, Height: 16.0, Mass: 2.0}, // Megasphere

	// --- POWERUPS ---
	2022: {Sprites: []string{"PINVA0", "PINVB0", "PINVC0", "PINVD0"}, Radius: 20.0, Height: 16.0, Mass: 1.0, Kind: config.ThingItemDef}, // Invulnerability
	2023: {Sprites: []string{"PSTRA0"}, Radius: 20.0, Height: 16.0, Mass: 1.0, Kind: config.ThingItemDef},                               // Berserk
	2024: {Sprites: []string{"PINSA0", "PINSB0", "PINSC0", "PINSD0"}, Radius: 20.0, Height: 16.0, Mass: 1.0, Kind: config.ThingItemDef}, // Partial Invisibility
	2025: {Sprites: []string{"SUITA0"}, Radius: 20.0, Height: 16.0, Mass: 5.0, Kind: config.ThingItemDef},                               // Radiation Suit
	2026: {Sprites: []string{"PMAPA0", "PMAPB0", "PMAPC0", "PMAPD0"}, Radius: 20.0, Height: 16.0, Mass: 0.5, Kind: config.ThingItemDef}, // Computer Map
	2045: {Sprites: []string{"PVISA0", "PVISB0"}, Radius: 20.0, Height: 16.0, Mass: 0.5, Kind: config.ThingItemDef},                     // Light Amplification Visor

	// --- CHIAVI ---
	5:  {Sprites: []string{"BKEYA0", "BKEYB0"}, Radius: 20.0, Height: 16.0, Mass: 0.1, Kind: config.ThingKeyDef}, // Blue Keycard
	13: {Sprites: []string{"RKEYA0", "RKEYB0"}, Radius: 20.0, Height: 16.0, Mass: 0.1, Kind: config.ThingKeyDef}, // Red Keycard
	6:  {Sprites: []string{"YKEYA0", "YKEYB0"}, Radius: 20.0, Height: 16.0, Mass: 0.1, Kind: config.ThingKeyDef}, // Yellow Keycard
	40: {Sprites: []string{"BSKUA0", "BSKUB0"}, Radius: 20.0, Height: 16.0, Mass: 0.5, Kind: config.ThingKeyDef}, // Blue Skull Key
	38: {Sprites: []string{"RSKUA0", "RSKUB0"}, Radius: 20.0, Height: 16.0, Mass: 0.5, Kind: config.ThingKeyDef}, // Red Skull Key
	39: {Sprites: []string{"YSKUA0", "YSKUB0"}, Radius: 20.0, Height: 16.0, Mass: 0.5, Kind: config.ThingKeyDef}, // Yellow Skull Key

	// --- OSTACOLI E DECORAZIONI ---
	2035: {Sprites: []string{"BAR1A0", "BAR1B0"}, Radius: 10.0, Height: 42.0, Mass: 100.0, Kind: config.ThingItemDef},                     // Explosive Barrel
	30:   {Sprites: []string{"COL1A0"}, Radius: 16.0, Height: 128.0, Mass: 1000.0, Kind: config.ThingItemDef},                             // Tall Green Pillar
	31:   {Sprites: []string{"COL2A0"}, Radius: 16.0, Height: 64.0, Mass: 1000.0, Kind: config.ThingItemDef},                              // Short Green Pillar
	32:   {Sprites: []string{"COL3A0"}, Radius: 16.0, Height: 128.0, Mass: 1000.0, Kind: config.ThingItemDef},                             // Tall Red Pillar
	33:   {Sprites: []string{"COL4A0"}, Radius: 16.0, Height: 64.0, Mass: 1000.0, Kind: config.ThingItemDef},                              // Short Red Pillar
	41:   {Sprites: []string{"CEYEA0", "CEYEB0", "CEYEC0"}, Radius: 16.0, Height: 54.0, Mass: 50.0, Kind: config.ThingItemDef},            // Evil Eye
	42:   {Sprites: []string{"FSKUA0", "FSKUB0", "FSKUC0"}, Radius: 16.0, Height: 54.0, Mass: 50.0, Kind: config.ThingItemDef},            // Floating Skull
	43:   {Sprites: []string{"TRE1A0"}, Radius: 16.0, Height: 54.0, Mass: 200.0, Kind: config.ThingItemDef},                               // Burnt Tree
	47:   {Sprites: []string{"SMITA0"}, Radius: 16.0, Height: 64.0, Mass: 500.0, Kind: config.ThingItemDef},                               // Stalagmite
	48:   {Sprites: []string{"ELECA0"}, Radius: 16.0, Height: 64.0, Mass: 1000.0, Kind: config.ThingItemDef},                              // Tall techno pillar
	54:   {Sprites: []string{"TRE2A0"}, Radius: 32.0, Height: 108.0, Mass: 500.0, Kind: config.ThingItemDef},                              // Large brown tree
	85:   {Sprites: []string{"TLMPA0", "TLMPB0", "TLMPC0", "TLMPD0"}, Radius: 16.0, Height: 68.0, Mass: 100.0, Kind: config.ThingItemDef}, // Tall techno lamp
	86:   {Sprites: []string{"TLP2A0", "TLP2B0", "TLP2C0", "TLP2D0"}, Radius: 16.0, Height: 54.0, Mass: 100.0, Kind: config.ThingItemDef}, // Short techno lamp
	2028: {Sprites: []string{"COLUA0"}, Radius: 16.0, Height: 54.0, Mass: 100.0, Kind: config.ThingItemDef},                               // Floor lamp (Yellow)
	34:   {Sprites: []string{"CANDA0"}, Radius: 20.0, Height: 16.0, Mass: 5.0, Kind: config.ThingItemDef},                                 // Candle
	35:   {Sprites: []string{"CBRAA0"}, Radius: 16.0, Height: 60.0, Mass: 50.0, Kind: config.ThingItemDef},                                // Candelabra
	44:   {Sprites: []string{"TBLUA0", "TBLUB0", "TBLUC0", "TBLUD0"}, Radius: 16.0, Height: 68.0, Mass: 100.0, Kind: config.ThingItemDef}, // Tall Blue Firestick
	45:   {Sprites: []string{"TGRNA0", "TGRNB0", "TGRNC0", "TGRND0"}, Radius: 16.0, Height: 68.0, Mass: 100.0, Kind: config.ThingItemDef}, // Tall Green Firestick
	46:   {Sprites: []string{"TREDA0", "TREDB0", "TREDC0", "TREDD0"}, Radius: 16.0, Height: 68.0, Mass: 100.0, Kind: config.ThingItemDef}, // Tall Red Firestick
	55:   {Sprites: []string{"SBLUA0", "SBLUB0", "SBLUC0", "SBLUD0"}, Radius: 16.0, Height: 16.0, Mass: 50.0, Kind: config.ThingItemDef},  // Short Blue Firestick
	56:   {Sprites: []string{"SGRNA0", "SGRNB0", "SGRNC0", "SGRND0"}, Radius: 16.0, Height: 16.0, Mass: 50.0, Kind: config.ThingItemDef},  // Short Green Firestick
	57:   {Sprites: []string{"SREDA0", "SREDB0", "SREDC0", "SREDD0"}, Radius: 16.0, Height: 16.0, Mass: 50.0, Kind: config.ThingItemDef},  // Short Red Firestick
	27:   {Sprites: []string{"POL4A0"}, Radius: 16.0, Height: 64.0, Mass: 100.0, Kind: config.ThingItemDef},                               // Skull on a pole
	28:   {Sprites: []string{"POL2A0"}, Radius: 16.0, Height: 64.0, Mass: 100.0, Kind: config.ThingItemDef},                               // Five skulls shish kebab
	29:   {Sprites: []string{"POL3A0", "POL3B0"}, Radius: 16.0, Height: 64.0, Mass: 100.0, Kind: config.ThingItemDef},                     // Pile of skulls
	36:   {Sprites: []string{"COL5A0", "COL5B0"}, Radius: 16.0, Height: 64.0, Mass: 100.0, Kind: config.ThingItemDef},                     // Heart column
	37:   {Sprites: []string{"COL6A0"}, Radius: 16.0, Height: 64.0, Mass: 100.0, Kind: config.ThingItemDef},                               // Red skull column

	// --- GORE E CADAVERI ---
	10: {Sprites: []string{"PLAYW0"}, Radius: 20.0, Height: 16.0, Mass: 80.0, Kind: config.ThingItemDef},  // Bloody mess
	12: {Sprites: []string{"PLAYW0"}, Radius: 20.0, Height: 16.0, Mass: 80.0, Kind: config.ThingItemDef},  // Bloody mess 2
	15: {Sprites: []string{"PLAYN0"}, Radius: 20.0, Height: 16.0, Mass: 80.0, Kind: config.ThingItemDef},  // Dead player
	18: {Sprites: []string{"POSSL0"}, Radius: 20.0, Height: 16.0, Mass: 100.0, Kind: config.ThingItemDef}, // Dead former human
	19: {Sprites: []string{"SPOSL0"}, Radius: 20.0, Height: 16.0, Mass: 100.0, Kind: config.ThingItemDef}, // Dead former sergeant
	20: {Sprites: []string{"TROOM0"}, Radius: 20.0, Height: 16.0, Mass: 100.0, Kind: config.ThingItemDef}, // Dead imp
	21: {Sprites: []string{"SARGN0"}, Radius: 30.0, Height: 16.0, Mass: 400.0, Kind: config.ThingItemDef}, // Dead demon
	22: {Sprites: []string{"HEADL0"}, Radius: 31.0, Height: 16.0, Mass: 400.0, Kind: config.ThingItemDef}, // Dead cacodemon
	24: {Sprites: []string{"POL5A0"}, Radius: 20.0, Height: 16.0, Mass: 5.0, Kind: config.ThingItemDef},   // Pool of blood
	79: {Sprites: []string{"POB1A0"}, Radius: 20.0, Height: 16.0, Mass: 5.0, Kind: config.ThingItemDef},   // Pool of blood and flesh
	80: {Sprites: []string{"POB2A0"}, Radius: 20.0, Height: 16.0, Mass: 5.0, Kind: config.ThingItemDef},   // Pool of blood
	61: {Sprites: []string{"GOR3A0"}, Radius: 16.0, Height: 68.0, Mass: 80.0, Kind: config.ThingItemDef},  // Hanging victim, 1-legged
	62: {Sprites: []string{"GOR5A0"}, Radius: 16.0, Height: 68.0, Mass: 20.0, Kind: config.ThingItemDef},  // Hanging leg
	73: {Sprites: []string{"GOR1A0"}, Radius: 16.0, Height: 84.0, Mass: 80.0, Kind: config.ThingItemDef},  // Hanging victim, guts removed
	74: {Sprites: []string{"GOR2A0"}, Radius: 16.0, Height: 84.0, Mass: 80.0, Kind: config.ThingItemDef},  // Hanging victim, guts and brain removed
	52: {Sprites: []string{"GOR4A0"}, Radius: 16.0, Height: 68.0, Mass: 80.0, Kind: config.ThingItemDef},  // Hanging pair of legs
	60: {Sprites: []string{"GOR2A0"}, Radius: 16.0, Height: 68.0, Mass: 80.0, Kind: config.ThingItemDef},  // Hanging victim (Non-blocking originale)
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
