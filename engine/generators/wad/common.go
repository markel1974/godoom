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

// const ScaleFactor = 5.0
const ScaleFactor = 10.0
const Epsilon = 1e-4

//const subSectorBit = uint16(0x8000)

// Polygon represents a sequence of points in 2D space that define a closed geometric shape.
type Polygon []model.XY

type Plane struct {
	nx, ny, ndx, ndy float64
	side             int
}

// SegmentData represents a line segment in 2D space, defined by its start and end points and an associated count value.
//type SegmentData struct {
//	Start PointFloat
//	End   PointFloat
//	Count int
//}

// abs returns the absolute value of the given integer x.
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// swap takes two integers a and b as input and returns them swapped in order.
func swap(a int, b int) (int, int) {
	return b, a
}

// swapF swaps the values of two float64 variables and returns them in reverse order.
func swapF(a float64, b float64) (float64, float64) {
	return b, a
}
