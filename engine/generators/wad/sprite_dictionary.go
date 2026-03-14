package wad

var _spriteDictionary = map[int][]string{
	// --- MOSTRI (Walk cycle frontale) ---
	3004: {"POSSA1", "POSSB1", "POSSC1", "POSSD1"}, // Zombieman
	9:    {"SPOSA1", "SPOSB1", "SPOSC1", "SPOSD1"}, // Shotgun Guy
	65:   {"CPOSA1", "CPOSB1", "CPOSC1", "CPOSD1"}, // Heavy Weapon Dude (Doom 2)
	3001: {"TROOA1", "TROOB1", "TROOC1", "TROOD1"}, // Imp
	3002: {"SARGA1", "SARGB1", "SARGC1", "SARGD1"}, // Demon
	58:   {"SARGA1", "SARGB1", "SARGC1", "SARGD1"}, // Spectre
	3003: {"BOSSA1", "BOSSB1", "BOSSC1", "BOSSD1"}, // Baron of Hell
	69:   {"BOS2A1", "BOS2B1", "BOS2C1", "BOS2D1"}, // Hell Knight (Doom 2)
	3005: {"HEADA1", "HEADB1", "HEADC1", "HEADD1"}, // Cacodemon
	3006: {"SKULA1", "SKULB1"},                     // Lost Soul
	68:   {"BSPIA1", "BSPIB1", "BSPIC1"},           // Arachnotron (Doom 2)
	71:   {"PAINA1", "PAINB1", "PAINC1"},           // Pain Elemental (Doom 2)
	66:   {"SKELA1", "SKELB1", "SKELC1", "SKELD1"}, // Revenant (Doom 2)
	67:   {"FATTA1", "FATTB1", "FATTC1"},           // Mancubus (Doom 2)
	64:   {"VILEA1", "VILEB1", "VILEC1"},           // Arch-Vile (Doom 2)
	16:   {"CYBRA1", "CYBRB1", "CYBRC1", "CYBRD1"}, // Cyberdemon
	7:    {"SPIDA1", "SPIDB1", "SPIDC1"},           // Spider Mastermind

	// --- ARMI (Pickup) ---
	2001: {"SHOTA0"}, // Shotgun
	82:   {"SGN2A0"}, // Super Shotgun (Doom 2)
	2002: {"MGUNA0"}, // Chaingun
	2003: {"LAUNA0"}, // Rocket Launcher
	2004: {"PLASA0"}, // Plasma Rifle
	2005: {"CSAWA0"}, // Chainsaw
	2006: {"BFUGA0"}, // BFG9000

	// --- MUNIZIONI ---
	2007: {"CLIPA0"}, // Ammo clip
	2048: {"AMMOA0"}, // Box of Ammo
	2008: {"SHELA0"}, // 4 Shells
	2049: {"SBOXA0"}, // Box of Shells
	2010: {"ROCKA0"}, // 1 Rocket
	2046: {"BROKA0"}, // Box of Rockets
	2047: {"CELPA0"}, // Energy Cell
	17:   {"CELPA0"}, // Energy Cell Pack (Usa stesso sprite base o varianti nel wad)
	8:    {"BPAKA0"}, // Backpack

	// --- CURE E ARMATURE ---
	2011: {"STIMA0"},                               // Stimpack
	2012: {"MEDIA0"},                               // Medikit
	2014: {"BON1A0", "BON1B0", "BON1C0", "BON1D0"}, // Health Bonus
	2015: {"BON2A0", "BON2B0", "BON2C0", "BON2D0"}, // Armor Bonus
	2018: {"ARM1A0", "ARM1B0"},                     // Green Armor
	2019: {"ARM2A0", "ARM2B0"},                     // Blue Armor
	2013: {"SOULA0", "SOULB0", "SOULC0", "SOULD0"}, // Soulsphere
	83:   {"MEGAA0", "MEGAB0", "MEGAC0", "MEGAD0"}, // Megasphere (Doom 2)

	// --- POWERUPS ---
	2022: {"PINVA0", "PINVB0", "PINVC0", "PINVD0"}, // Invulnerability
	2023: {"PSTRA0"},                               // Berserk
	2024: {"PINSA0", "PINSB0", "PINSC0", "PINSD0"}, // Partial Invisibility
	2025: {"SUITA0"},                               // Radiation Suit
	2026: {"PMAPA0", "PMAPB0", "PMAPC0", "PMAPD0"}, // Computer Map
	2045: {"PVISA0", "PVISB0"},                     // Light Amplification Visor

	// --- CHIAVI ---
	5:  {"BKEYA0", "BKEYB0"}, // Blue Keycard
	13: {"RKEYA0", "RKEYB0"}, // Red Keycard
	6:  {"YKEYA0", "YKEYB0"}, // Yellow Keycard
	40: {"BSKUA0", "BSKUB0"}, // Blue Skull Key
	38: {"RSKUA0", "RSKUB0"}, // Red Skull Key
	39: {"YSKUA0", "YSKUB0"}, // Yellow Skull Key

	// --- OSTACOLI E DECORAZIONI ANIMATE ---
	34:   {"CANDA0"},                               // Candle
	35:   {"CBRAA0"},                               // Candelabra
	44:   {"TBLUA0", "TBLUB0", "TBLUC0", "TBLUD0"}, // Tall Blue Firestick
	45:   {"TGRNA0", "TGRNB0", "TGRNC0", "TGRND0"}, // Tall Green Firestick
	46:   {"TREDA0", "TREDB0", "TREDC0", "TREDD0"}, // Tall Red Firestick
	55:   {"SBLUA0", "SBLUB0", "SBLUC0", "SBLUD0"}, // Short Blue Firestick
	56:   {"SGRNA0", "SGRNB0", "SGRNC0", "SGRND0"}, // Short Green Firestick
	57:   {"SREDA0", "SREDB0", "SREDC0", "SREDD0"}, // Short Red Firestick
	2035: {"BAR1A0", "BAR1B0"},                     // Explosive Barrel

	// --- DECORAZIONI STATICHE ---
	30:   {"COL1A0"},                               // Tall Green Pillar
	31:   {"COL2A0"},                               // Short Green Pillar
	32:   {"COL3A0"},                               // Tall Red Pillar
	33:   {"COL4A0"},                               // Short Red Pillar
	2028: {"COLUA0"},                               // Floor lamp (Yellow)
	41:   {"CEYEA0", "CEYEB0", "CEYEC0"},           // Evil Eye
	42:   {"FSKUA0", "FSKUB0", "FSKUC0"},           // Floating Skull
	43:   {"TRE1A0"},                               // Burnt Tree
	47:   {"SMITA0"},                               // Stalagmite
	48:   {"ELECA0"},                               // Tall techno pillar
	54:   {"TRE2A0"},                               // Large brown tree
	85:   {"TLMPA0", "TLMPB0", "TLMPC0", "TLMPD0"}, // Tall techno lamp
	86:   {"TLP2A0", "TLP2B0", "TLP2C0", "TLP2D0"}, // Short techno lamp

	// --- GORE, CADAVERI E IMPICCATI ---
	10: {"PLAYW0"}, // Bloody mess
	12: {"PLAYW0"}, // Bloody mess 2
	15: {"PLAYN0"}, // Dead player
	18: {"POSSL0"}, // Dead former human
	19: {"SPOSL0"}, // Dead former sergeant
	20: {"TROOM0"}, // Dead imp
	21: {"SARGN0"}, // Dead demon
	22: {"HEADL0"}, // Dead cacodemon
	24: {"POL5A0"}, // Pool of blood
	61: {"GOR3A0"}, // Hanging victim, 1-legged
	62: {"GOR5A0"}, // Hanging leg
	73: {"GOR1A0"}, // Hanging victim, guts removed
	74: {"GOR2A0"}, // Hanging victim, guts and brain removed
	79: {"POB1A0"}, // Pool of blood and flesh
	80: {"POB2A0"}, // Pool of blood

	// --- OSTACOLI INFERNALI E GORE ---
	27: {"POL4A0"},           // Skull on a pole
	28: {"POL2A0"},           // Five skulls shish kebab
	29: {"POL3A0", "POL3B0"}, // Pile of skulls and candles (Animato)
	36: {"COL5A0", "COL5B0"}, // Heart column (Animato)
	37: {"COL6A0"},           // Red skull column
	52: {"GOR4A0"},           // Hanging pair of legs

	60: {"GOR2A0"}, // Hanging victim, brains removed (Non-blocking)
	14: {""},       // Teleport Destination (invisible marker)
}
