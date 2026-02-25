package wad

import "github.com/markel1974/godoom/engine/model"

//http://www.gamers.org/dhs/helpdocs/dmsp1666.html
//http://doomwiki.org/
//https://github.com/penberg/godoom
//https://github.com/mausimus/rtdoom/blob/master/rtdoom/Projection.cpp

//https://github.com/gamescomputersplay/wad2pic/blob/main/wad2pic.py

const (
	DefinitionWall = 0
	DefinitionVoid = 1      // In Doom un passaggio libero tra subsector
	SnapGrid       = 1000.0 // Usato per arrotondare i float a 3 decimali
	PrecisionScale = 10000.0
)

const ScaleFactor = 5.0
const Epsilon = 1e-4

//const subSectorBit = uint16(0x8000)

// Polygon represents a sequence of points in 2D space that define a closed geometric shape.
type Polygon []model.XY

type Plane struct {
	nx, ny, ndx, ndy float64
	side             int
}
