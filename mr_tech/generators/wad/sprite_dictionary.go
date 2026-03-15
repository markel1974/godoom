package wad

// ThingDef represents the definition of a physical or graphical object with its associated properties.
type ThingDef struct {
	Sprites []string
	Radius  float64
	Height  float64
	Mass    float64
}

// _spriteDictionary is a map that associates integer keys with ThingDef structures, defining game objects and their properties.
var _spriteDictionary = map[int]ThingDef{
	// --- MOSTRI ---
	3004: {Sprites: []string{"POSSA1", "POSSB1", "POSSC1", "POSSD1"}, Radius: 20.0, Height: 56.0, Mass: 100.0},   // Zombieman
	9:    {Sprites: []string{"SPOSA1", "SPOSB1", "SPOSC1", "SPOSD1"}, Radius: 20.0, Height: 56.0, Mass: 100.0},   // Shotgun Guy
	65:   {Sprites: []string{"CPOSA1", "CPOSB1", "CPOSC1", "CPOSD1"}, Radius: 20.0, Height: 56.0, Mass: 100.0},   // Heavy Weapon Dude
	3001: {Sprites: []string{"TROOA1", "TROOB1", "TROOC1", "TROOD1"}, Radius: 20.0, Height: 56.0, Mass: 100.0},   // Imp
	3002: {Sprites: []string{"SARGA1", "SARGB1", "SARGC1", "SARGD1"}, Radius: 30.0, Height: 56.0, Mass: 400.0},   // Demon
	58:   {Sprites: []string{"SARGA1", "SARGB1", "SARGC1", "SARGD1"}, Radius: 30.0, Height: 56.0, Mass: 400.0},   // Spectre
	3003: {Sprites: []string{"BOSSA1", "BOSSB1", "BOSSC1", "BOSSD1"}, Radius: 24.0, Height: 64.0, Mass: 1000.0},  // Baron of Hell
	69:   {Sprites: []string{"BOS2A1", "BOS2B1", "BOS2C1", "BOS2D1"}, Radius: 24.0, Height: 64.0, Mass: 1000.0},  // Hell Knight
	3005: {Sprites: []string{"HEADA1", "HEADB1", "HEADC1", "HEADD1"}, Radius: 31.0, Height: 56.0, Mass: 400.0},   // Cacodemon
	3006: {Sprites: []string{"SKULA1", "SKULB1"}, Radius: 16.0, Height: 56.0, Mass: 50.0},                        // Lost Soul
	68:   {Sprites: []string{"BSPIA1", "BSPIB1", "BSPIC1"}, Radius: 64.0, Height: 64.0, Mass: 600.0},             // Arachnotron
	71:   {Sprites: []string{"PAINA1", "PAINB1", "PAINC1"}, Radius: 31.0, Height: 56.0, Mass: 400.0},             // Pain Elemental
	66:   {Sprites: []string{"SKELA1", "SKELB1", "SKELC1", "SKELD1"}, Radius: 20.0, Height: 56.0, Mass: 500.0},   // Revenant
	67:   {Sprites: []string{"FATTA1", "FATTB1", "FATTC1"}, Radius: 48.0, Height: 64.0, Mass: 1000.0},            // Mancubus
	64:   {Sprites: []string{"VILEA1", "VILEB1", "VILEC1"}, Radius: 20.0, Height: 56.0, Mass: 500.0},             // Arch-Vile
	16:   {Sprites: []string{"CYBRA1", "CYBRB1", "CYBRC1", "CYBRD1"}, Radius: 40.0, Height: 110.0, Mass: 1000.0}, // Cyberdemon
	7:    {Sprites: []string{"SPIDA1", "SPIDB1", "SPIDC1"}, Radius: 130.0, Height: 100.0, Mass: 1000.0},          // Spider Mastermind

	// --- ARMI ---
	2001: {Sprites: []string{"SHOTA0"}, Radius: 20.0, Height: 16.0, Mass: 4.0},  // Shotgun
	82:   {Sprites: []string{"SGN2A0"}, Radius: 20.0, Height: 16.0, Mass: 5.0},  // Super Shotgun
	2002: {Sprites: []string{"MGUNA0"}, Radius: 20.0, Height: 16.0, Mass: 8.0},  // Chaingun
	2003: {Sprites: []string{"LAUNA0"}, Radius: 20.0, Height: 16.0, Mass: 12.0}, // Rocket Launcher
	2004: {Sprites: []string{"PLASA0"}, Radius: 20.0, Height: 16.0, Mass: 10.0}, // Plasma Rifle
	2005: {Sprites: []string{"CSAWA0"}, Radius: 20.0, Height: 16.0, Mass: 6.0},  // Chainsaw
	2006: {Sprites: []string{"BFUGA0"}, Radius: 20.0, Height: 16.0, Mass: 25.0}, // BFG9000

	// --- MUNIZIONI ---
	2007: {Sprites: []string{"CLIPA0"}, Radius: 20.0, Height: 16.0, Mass: 0.5},  // Ammo clip
	2048: {Sprites: []string{"AMMOA0"}, Radius: 20.0, Height: 16.0, Mass: 2.0},  // Box of Ammo
	2008: {Sprites: []string{"SHELA0"}, Radius: 20.0, Height: 16.0, Mass: 0.5},  // 4 Shells
	2049: {Sprites: []string{"SBOXA0"}, Radius: 20.0, Height: 16.0, Mass: 2.0},  // Box of Shells
	2010: {Sprites: []string{"ROCKA0"}, Radius: 20.0, Height: 16.0, Mass: 1.0},  // 1 Rocket
	2046: {Sprites: []string{"BROKA0"}, Radius: 20.0, Height: 16.0, Mass: 5.0},  // Box of Rockets
	2047: {Sprites: []string{"CELPA0"}, Radius: 20.0, Height: 16.0, Mass: 1.5},  // Energy Cell
	17:   {Sprites: []string{"CELPA0"}, Radius: 20.0, Height: 16.0, Mass: 5.0},  // Energy Cell Pack
	8:    {Sprites: []string{"BPAKA0"}, Radius: 20.0, Height: 16.0, Mass: 10.0}, // Backpack

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
	2022: {Sprites: []string{"PINVA0", "PINVB0", "PINVC0", "PINVD0"}, Radius: 20.0, Height: 16.0, Mass: 1.0}, // Invulnerability
	2023: {Sprites: []string{"PSTRA0"}, Radius: 20.0, Height: 16.0, Mass: 1.0},                               // Berserk
	2024: {Sprites: []string{"PINSA0", "PINSB0", "PINSC0", "PINSD0"}, Radius: 20.0, Height: 16.0, Mass: 1.0}, // Partial Invisibility
	2025: {Sprites: []string{"SUITA0"}, Radius: 20.0, Height: 16.0, Mass: 5.0},                               // Radiation Suit
	2026: {Sprites: []string{"PMAPA0", "PMAPB0", "PMAPC0", "PMAPD0"}, Radius: 20.0, Height: 16.0, Mass: 0.5}, // Computer Map
	2045: {Sprites: []string{"PVISA0", "PVISB0"}, Radius: 20.0, Height: 16.0, Mass: 0.5},                     // Light Amplification Visor

	// --- CHIAVI ---
	5:  {Sprites: []string{"BKEYA0", "BKEYB0"}, Radius: 20.0, Height: 16.0, Mass: 0.1}, // Blue Keycard
	13: {Sprites: []string{"RKEYA0", "RKEYB0"}, Radius: 20.0, Height: 16.0, Mass: 0.1}, // Red Keycard
	6:  {Sprites: []string{"YKEYA0", "YKEYB0"}, Radius: 20.0, Height: 16.0, Mass: 0.1}, // Yellow Keycard
	40: {Sprites: []string{"BSKUA0", "BSKUB0"}, Radius: 20.0, Height: 16.0, Mass: 0.5}, // Blue Skull Key
	38: {Sprites: []string{"RSKUA0", "RSKUB0"}, Radius: 20.0, Height: 16.0, Mass: 0.5}, // Red Skull Key
	39: {Sprites: []string{"YSKUA0", "YSKUB0"}, Radius: 20.0, Height: 16.0, Mass: 0.5}, // Yellow Skull Key

	// --- OSTACOLI E DECORAZIONI ---
	2035: {Sprites: []string{"BAR1A0", "BAR1B0"}, Radius: 10.0, Height: 42.0, Mass: 100.0},                     // Explosive Barrel
	30:   {Sprites: []string{"COL1A0"}, Radius: 16.0, Height: 128.0, Mass: 1000.0},                             // Tall Green Pillar
	31:   {Sprites: []string{"COL2A0"}, Radius: 16.0, Height: 64.0, Mass: 1000.0},                              // Short Green Pillar
	32:   {Sprites: []string{"COL3A0"}, Radius: 16.0, Height: 128.0, Mass: 1000.0},                             // Tall Red Pillar
	33:   {Sprites: []string{"COL4A0"}, Radius: 16.0, Height: 64.0, Mass: 1000.0},                              // Short Red Pillar
	41:   {Sprites: []string{"CEYEA0", "CEYEB0", "CEYEC0"}, Radius: 16.0, Height: 54.0, Mass: 50.0},            // Evil Eye
	42:   {Sprites: []string{"FSKUA0", "FSKUB0", "FSKUC0"}, Radius: 16.0, Height: 54.0, Mass: 50.0},            // Floating Skull
	43:   {Sprites: []string{"TRE1A0"}, Radius: 16.0, Height: 54.0, Mass: 200.0},                               // Burnt Tree
	47:   {Sprites: []string{"SMITA0"}, Radius: 16.0, Height: 64.0, Mass: 500.0},                               // Stalagmite
	48:   {Sprites: []string{"ELECA0"}, Radius: 16.0, Height: 64.0, Mass: 1000.0},                              // Tall techno pillar
	54:   {Sprites: []string{"TRE2A0"}, Radius: 32.0, Height: 108.0, Mass: 500.0},                              // Large brown tree
	85:   {Sprites: []string{"TLMPA0", "TLMPB0", "TLMPC0", "TLMPD0"}, Radius: 16.0, Height: 68.0, Mass: 100.0}, // Tall techno lamp
	86:   {Sprites: []string{"TLP2A0", "TLP2B0", "TLP2C0", "TLP2D0"}, Radius: 16.0, Height: 54.0, Mass: 100.0}, // Short techno lamp
	2028: {Sprites: []string{"COLUA0"}, Radius: 16.0, Height: 54.0, Mass: 100.0},                               // Floor lamp (Yellow)
	34:   {Sprites: []string{"CANDA0"}, Radius: 20.0, Height: 16.0, Mass: 5.0},                                 // Candle
	35:   {Sprites: []string{"CBRAA0"}, Radius: 16.0, Height: 60.0, Mass: 50.0},                                // Candelabra
	44:   {Sprites: []string{"TBLUA0", "TBLUB0", "TBLUC0", "TBLUD0"}, Radius: 16.0, Height: 68.0, Mass: 100.0}, // Tall Blue Firestick
	45:   {Sprites: []string{"TGRNA0", "TGRNB0", "TGRNC0", "TGRND0"}, Radius: 16.0, Height: 68.0, Mass: 100.0}, // Tall Green Firestick
	46:   {Sprites: []string{"TREDA0", "TREDB0", "TREDC0", "TREDD0"}, Radius: 16.0, Height: 68.0, Mass: 100.0}, // Tall Red Firestick
	55:   {Sprites: []string{"SBLUA0", "SBLUB0", "SBLUC0", "SBLUD0"}, Radius: 16.0, Height: 16.0, Mass: 50.0},  // Short Blue Firestick
	56:   {Sprites: []string{"SGRNA0", "SGRNB0", "SGRNC0", "SGRND0"}, Radius: 16.0, Height: 16.0, Mass: 50.0},  // Short Green Firestick
	57:   {Sprites: []string{"SREDA0", "SREDB0", "SREDC0", "SREDD0"}, Radius: 16.0, Height: 16.0, Mass: 50.0},  // Short Red Firestick
	27:   {Sprites: []string{"POL4A0"}, Radius: 16.0, Height: 64.0, Mass: 100.0},                               // Skull on a pole
	28:   {Sprites: []string{"POL2A0"}, Radius: 16.0, Height: 64.0, Mass: 100.0},                               // Five skulls shish kebab
	29:   {Sprites: []string{"POL3A0", "POL3B0"}, Radius: 16.0, Height: 64.0, Mass: 100.0},                     // Pile of skulls
	36:   {Sprites: []string{"COL5A0", "COL5B0"}, Radius: 16.0, Height: 64.0, Mass: 100.0},                     // Heart column
	37:   {Sprites: []string{"COL6A0"}, Radius: 16.0, Height: 64.0, Mass: 100.0},                               // Red skull column

	// --- GORE E CADAVERI ---
	10: {Sprites: []string{"PLAYW0"}, Radius: 20.0, Height: 16.0, Mass: 80.0},  // Bloody mess
	12: {Sprites: []string{"PLAYW0"}, Radius: 20.0, Height: 16.0, Mass: 80.0},  // Bloody mess 2
	15: {Sprites: []string{"PLAYN0"}, Radius: 20.0, Height: 16.0, Mass: 80.0},  // Dead player
	18: {Sprites: []string{"POSSL0"}, Radius: 20.0, Height: 16.0, Mass: 100.0}, // Dead former human
	19: {Sprites: []string{"SPOSL0"}, Radius: 20.0, Height: 16.0, Mass: 100.0}, // Dead former sergeant
	20: {Sprites: []string{"TROOM0"}, Radius: 20.0, Height: 16.0, Mass: 100.0}, // Dead imp
	21: {Sprites: []string{"SARGN0"}, Radius: 30.0, Height: 16.0, Mass: 400.0}, // Dead demon
	22: {Sprites: []string{"HEADL0"}, Radius: 31.0, Height: 16.0, Mass: 400.0}, // Dead cacodemon
	24: {Sprites: []string{"POL5A0"}, Radius: 20.0, Height: 16.0, Mass: 5.0},   // Pool of blood
	79: {Sprites: []string{"POB1A0"}, Radius: 20.0, Height: 16.0, Mass: 5.0},   // Pool of blood and flesh
	80: {Sprites: []string{"POB2A0"}, Radius: 20.0, Height: 16.0, Mass: 5.0},   // Pool of blood
	61: {Sprites: []string{"GOR3A0"}, Radius: 16.0, Height: 68.0, Mass: 80.0},  // Hanging victim, 1-legged
	62: {Sprites: []string{"GOR5A0"}, Radius: 16.0, Height: 68.0, Mass: 20.0},  // Hanging leg
	73: {Sprites: []string{"GOR1A0"}, Radius: 16.0, Height: 84.0, Mass: 80.0},  // Hanging victim, guts removed
	74: {Sprites: []string{"GOR2A0"}, Radius: 16.0, Height: 84.0, Mass: 80.0},  // Hanging victim, guts and brain removed
	52: {Sprites: []string{"GOR4A0"}, Radius: 16.0, Height: 68.0, Mass: 80.0},  // Hanging pair of legs
	60: {Sprites: []string{"GOR2A0"}, Radius: 16.0, Height: 68.0, Mass: 80.0},  // Hanging victim (Non-blocking originale)
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
