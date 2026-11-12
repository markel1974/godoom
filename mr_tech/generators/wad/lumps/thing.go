package lumps

import (
	"encoding/binary"
	"io"
	"unsafe"
)

// ThingOptionEasy represents the option for an easy difficulty level.
// ThingOptionMedium represents the option for a medium difficulty level.
// ThingOptionHard represents the option for a hard difficulty level.
// ThingOptionAmbush represents the option for ambush mode.
// ThingOptionMultiplayer represents the option for multiplayer mode.
const (
	ThingOptionEasy        uint16 = 1 << 0
	ThingOptionMedium      uint16 = 1 << 1
	ThingOptionHard        uint16 = 1 << 2
	ThingOptionAmbush      uint16 = 1 << 3
	ThingOptionMultiplayer uint16 = 1 << 4
)

type ThingType int16

const (
	// Special
	ThingTypeNone            ThingType = 0
	ThingTypePlayer1Start    ThingType = 1
	ThingTypePlayer2Start    ThingType = 2
	ThingTypePlayer3Start    ThingType = 3
	ThingTypePlayer4Start    ThingType = 4
	ThingTypeDeathmatchStart ThingType = 11
	ThingTypeTeleportLanding ThingType = 14

	// Monsters
	ThingTypeFormerHuman      ThingType = 3004
	ThingTypeFormerSergeant   ThingType = 9
	ThingTypeImp              ThingType = 3001
	ThingTypeDemon            ThingType = 3002
	ThingTypeSpectre          ThingType = 58
	ThingTypeLostSoul         ThingType = 3006
	ThingTypeCacodemon        ThingType = 3005
	ThingTypeBaronOfHell      ThingType = 3003
	ThingTypeSpiderMastermind ThingType = 7
	ThingTypeCyberdemon       ThingType = 16

	// Weapons
	ThingTypeChainsaw       ThingType = 2005
	ThingTypeShotgun        ThingType = 2001
	ThingTypeChaingun       ThingType = 2002
	ThingTypeRocketLauncher ThingType = 2003
	ThingTypePlasmaGun      ThingType = 2004
	ThingTypeBFG9000        ThingType = 2006

	// Ammunition
	ThingTypeClip          ThingType = 2007
	ThingTypeShotgunShells ThingType = 2008
	ThingTypeRocket        ThingType = 2010
	ThingTypeCellCharge    ThingType = 2047
	ThingTypeAmmoBox       ThingType = 2048
	ThingTypeShellBox      ThingType = 2049
	ThingTypeRocketBox     ThingType = 2046
	ThingTypeCellPack      ThingType = 17
	ThingTypeBackpack      ThingType = 8

	// Health and armor
	ThingTypeStimpack    ThingType = 2011
	ThingTypeMedikit     ThingType = 2012
	ThingTypeHealthBonus ThingType = 2014
	ThingTypeArmorBonus  ThingType = 2015
	ThingTypeGreenArmor  ThingType = 2018
	ThingTypeBlueArmor   ThingType = 2019
	ThingTypeSoulSphere  ThingType = 2013

	// Power-ups
	ThingTypeInvulnerability ThingType = 2022
	ThingTypeBerserk         ThingType = 2023
	ThingTypeInvisibility    ThingType = 2024
	ThingTypeRadiationSuit   ThingType = 2025
	ThingTypeComputerMap     ThingType = 2026
	ThingTypeLightAmplifier  ThingType = 2045

	// Keys
	ThingTypeBlueKeycard    ThingType = 5
	ThingTypeBlueSkullKey   ThingType = 40
	ThingTypeRedKeycard     ThingType = 13
	ThingTypeRedSkullKey    ThingType = 38
	ThingTypeYellowKeycard  ThingType = 6
	ThingTypeYellowSkullKey ThingType = 39

	// Obstacles
	ThingTypeBarrel ThingType = 2035

	// Decorations - pillars and trees
	ThingTypeTechnoPillar     ThingType = 48
	ThingTypeGreenPillar      ThingType = 30
	ThingTypeRedPillar        ThingType = 32
	ThingTypeShortGreenPillar ThingType = 31
	ThingTypeHeartPillar      ThingType = 36
	ThingTypeShortRedPillar   ThingType = 33
	ThingTypeSkullPillar      ThingType = 37
	ThingTypeStalagmite       ThingType = 47
	ThingTypeBurntTree        ThingType = 43
	ThingTypeLargeBrownTree   ThingType = 54

	// Decorations - lights and fire
	ThingTypeFloorLamp           ThingType = 2028
	ThingTypeCandle              ThingType = 34
	ThingTypeCandelabra          ThingType = 35
	ThingTypeTallBlueFirestick   ThingType = 44
	ThingTypeTallGreenFirestick  ThingType = 45
	ThingTypeTallRedFirestick    ThingType = 46
	ThingTypeShortBlueFirestick  ThingType = 55
	ThingTypeShortGreenFirestick ThingType = 56
	ThingTypeShortRedFirestick   ThingType = 57

	// Decorations - miscellaneous
	ThingTypeEvilEye       ThingType = 41
	ThingTypeFloatingSkull ThingType = 42

	// Hanging corpses
	ThingTypeHangingVictim    ThingType = 49
	ThingTypeHangingArms      ThingType = 50
	ThingTypeHangingOneLegged ThingType = 51
	ThingTypeHangingLegs      ThingType = 52
	ThingTypeHangingLeg       ThingType = 53

	// Impaled bodies
	ThingTypeImpaledHuman     ThingType = 25
	ThingTypeTwitchingImpaled ThingType = 26
	ThingTypeSkullOnPole      ThingType = 27
	ThingTypeSkullKebab       ThingType = 28
	ThingTypeSkullPile        ThingType = 29

	// Corpses and blood
	ThingTypeBloodyMess1     ThingType = 10
	ThingTypeBloodyMess2     ThingType = 12
	ThingTypePoolOfBlood     ThingType = 24
	ThingTypeDeadPlayer      ThingType = 15
	ThingTypeDeadFormerHuman ThingType = 18
	ThingTypeDeadSergeant    ThingType = 19
	ThingTypeDeadImp         ThingType = 20
	ThingTypeDeadDemon       ThingType = 21
	ThingTypeDeadCacodemon   ThingType = 22
	ThingTypeDeadLostSoul    ThingType = 23
)

// Thing represents an entity in the game with position, orientation, type, and options settings.
type Thing struct {
	X       int16
	Y       int16
	Angle   int16
	Type    int16
	Options int16
}

// NewThings reads and parses THINGS lump data from the given file and returns a slice of Thing pointers or an error.
func NewThings(f io.ReadSeekCloser, lumpInfo *LumpInfo) ([]*Thing, error) {
	var pThing Thing
	count := int(lumpInfo.Size) / int(unsafe.Sizeof(pThing))
	pThings := make([]Thing, count)
	if err := binary.Read(f, binary.LittleEndian, pThings); err != nil {
		return nil, err
	}
	things := make([]*Thing, count)
	for idx, t := range pThings {
		things[idx] = &Thing{
			X:       t.X,
			Y:       t.Y,
			Angle:   t.Angle,
			Type:    t.Type,
			Options: t.Options,
		}
	}
	return things, nil
}

// IsLevelAvailable determines if the given level is available for the Thing based on its difficulty options.
func (t *Thing) IsLevelAvailable(level int) bool {
	switch level {
	case 1, 2:
		return t.IsEasy()
	case 3:
		return t.IsMedium()
	case 4, 5:
		return t.IsHard()
	}
	return false
}

// IsEasy checks if the `Thing` is marked as easy mode by evaluating its options flag.
func (t *Thing) IsEasy() bool {
	return uint16(t.Options)&ThingOptionEasy != 0
}

// IsMedium checks if the `Thing` is configured for medium difficulty by evaluating the `ThingOptionMedium` flag.
func (t *Thing) IsMedium() bool {
	return uint16(t.Options)&ThingOptionMedium != 0
}

// IsHard checks if the `Thing` instance is marked with the "hard" difficulty option based on its Options field.
func (t *Thing) IsHard() bool {
	return uint16(t.Options)&ThingOptionHard != 0
}

// IsAmbush checks if the Thing has the ambush option set in its Options flag.
func (t *Thing) IsAmbush() bool {
	return uint16(t.Options)&ThingOptionAmbush != 0
}

// IsMultiplayerOnly checks if the Thing is configured for multiplayer-only mode based on the Options flag.
func (t *Thing) IsMultiplayerOnly() bool {
	return uint16(t.Options)&ThingOptionMultiplayer != 0
}

// GetType returns the ThingType of the current Thing instance by casting its Type field to ThingType.
func (t *Thing) GetType() ThingType {
	return ThingType(t.Type)
}
