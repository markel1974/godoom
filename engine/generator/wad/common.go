package wad

import "github.com/markel1974/godoom/engine/model"

//http://www.gamers.org/dhs/helpdocs/dmsp1666.html
//http://doomwiki.org/
//https://github.com/penberg/godoom
//https://github.com/mausimus/rtdoom/blob/master/rtdoom/Projection.cpp

//https://github.com/gamescomputersplay/wad2pic/blob/main/wad2pic.py

const (
	ScaleFactor    = 10.0
	DefinitionWall = 0
	DefinitionVoid = 1 // In Doom un passaggio libero tra subsector
)

//const subSectorBit = uint16(0x8000)

// Polygon represents a sequence of points in 2D space that define a closed geometric shape.
type Polygon []model.XY
