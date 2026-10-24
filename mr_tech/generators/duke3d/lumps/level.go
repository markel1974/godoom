package lumps

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"

	"github.com/markel1974/godoom/mr_tech/geometry"
)

type LevelHeader struct {
	Version    int32
	PosX       int32
	PosY       int32
	PosZ       int32
	Ang        int16
	CurSectNum int16
}

// Level represents a game map containing headers, sectors, walls, and sprites for structural and visual data.
type Level struct {
	Header  LevelHeader
	Sectors []Sector
	Walls   []Wall
	Sprites []Sprite
}

// NewLevel creates and returns a new instance of Level with zero-initialized fields.
func NewLevel() *Level {
	return &Level{}
}

// Parse reads and interprets map data from the provided io.Reader, initializing the Level structure with sectors, walls, and sprites.
func (l *Level) Parse(r io.Reader) error {
	if err := binary.Read(r, binary.LittleEndian, &l.Header); err != nil {
		return fmt.Errorf("reading map header: %w", err)
	}
	if l.Header.Version < 7 || l.Header.Version > 9 {
		return fmt.Errorf("unsupported build map version: %d (expected 7, 8 or 9)", l.Header.Version)
	}
	var numSectors int16
	if err := binary.Read(r, binary.LittleEndian, &numSectors); err != nil {
		return fmt.Errorf("reading numsectors: %w", err)
	}
	if numSectors < 0 {
		return fmt.Errorf("invalid sector count: %d", numSectors)
	}
	l.Sectors = make([]Sector, numSectors)
	if err := binary.Read(r, binary.LittleEndian, l.Sectors); err != nil {
		return fmt.Errorf("reading sectors: %w", err)
	}
	var numWalls int16
	if err := binary.Read(r, binary.LittleEndian, &numWalls); err != nil {
		return fmt.Errorf("reading numwalls: %w", err)
	}
	if numWalls < 0 {
		return fmt.Errorf("invalid wall count: %d", numWalls)
	}
	l.Walls = make([]Wall, numWalls)
	if err := binary.Read(r, binary.LittleEndian, l.Walls); err != nil {
		return fmt.Errorf("reading walls: %w", err)
	}

	var numSprites int16
	if err := binary.Read(r, binary.LittleEndian, &numSprites); err != nil {
		// Sprites are optional at EOF
		return nil
	}
	if numSprites > 0 {
		l.Sprites = make([]Sprite, numSprites)
		if err := binary.Read(r, binary.LittleEndian, l.Sprites); err != nil {
			return fmt.Errorf("reading sprites: %w", err)
		}
	}
	return nil
}

// GetSector returns a pointer to the Sector at the given index or nil if the index is out of bounds.
func (l *Level) GetSector(idx int) *Sector {
	if idx < 0 || idx >= len(l.Sectors) {
		return nil
	}
	return &l.Sectors[idx]
}

// GetWall returns a pointer to the wall at the specified index or nil if the index is out of range.
func (l *Level) GetWall(idx int) *Wall {
	if idx < 0 || idx >= len(l.Walls) {
		return nil
	}
	return &l.Walls[idx]
}

// ComputeFloorSlope calculates the slope gradient and direction of a floor in a specific sector.
// secIdx specifies the sector index, scaleSectorH adjusts the Z-axis scale, and scaleXY adjusts the XY-axis scale.
// flipY determines whether the Y-axis is inverted. Returns gradient, gx, gy, and ok indicating success or failure.
func (l *Level) ComputeFloorSlope(secIdx int, scaleSectorH float64, scaleXY float64, flipY bool) (gradient, gx, gy float64, ok bool) {
	sec := l.GetSector(secIdx)
	if sec == nil || !sec.IsSlopedFloor() {
		return 0, 0, 0, false
	}
	return l.computeSlope(int(sec.WallPtr), sec.FloorHeiNum, scaleSectorH, scaleXY, flipY)
}

// ComputeCeilingSlope calculates the slope of a sloped ceiling for the given sector index in a level.
// It returns the gradient, x and y direction coefficients (gx, gy), and a boolean indicating success or failure.
func (l *Level) ComputeCeilingSlope(secIdx int, scaleSectorH float64, scaleXY float64, flipY bool) (gradient, gx, gy float64, ok bool) {
	sec := l.GetSector(secIdx)
	if sec == nil || !sec.IsSlopedCeiling() {
		return 0, 0, 0, false
	}
	return l.computeSlope(int(sec.WallPtr), sec.CeilingHeiNum, scaleSectorH, scaleXY, flipY)
}

// computeSlope calculates the slope of a floor or ceiling using a reference wall and height information.
func (l *Level) computeSlope(pivotWallIdx int, heiNum int16, scaleSectorH float64, scaleXY float64, flipY bool) (gradient, gx, gy float64, ok bool) {
	w1 := l.GetWall(pivotWallIdx)
	if w1 == nil {
		return 0, 0, 0, false
	}
	w2 := l.GetWall(int(w1.Point2))
	if w2 == nil {
		return 0, 0, 0, false
	}

	v1 := geometry.XY{X: float64(w1.X) * scaleXY, Y: float64(w1.Y) * scaleXY}
	v2 := geometry.XY{X: float64(w2.X) * scaleXY, Y: float64(w2.Y) * scaleXY}
	if flipY {
		v1.Y = -v1.Y
		v2.Y = -v2.Y
	}

	dx := v2.X - v1.X
	dy := v2.Y - v1.Y
	lenW := math.Hypot(dx, dy)
	if lenW <= 0 {
		return 0, 0, 0, false
	}

	// In Build engine, sector interior is to the right of the wall when walking along wall from w1 to w2.
	// When Y is flipped (Cartesian Y up), walking along wall in clockwise order has sector interior to the right:
	// Right normal in Cartesian (Y up) for (dx, dy) is (dy / lenW, -dx / lenW).
	// Without flip (Y down), right normal for (dx, dy) is (-dy / lenW, dx / lenW).
	var nx, ny float64
	if flipY {
		nx = dy / lenW
		ny = -dx / lenW
	} else {
		nx = -dy / lenW
		ny = dx / lenW
	}

	// Height in godoom: Z_godoom = -Z_build / 256.0 * scaleSectorH.
	// dz_build / dd_build = float64(heiNum) / 256.0.
	// dz_godoom / dd_godoom = -float64(heiNum) / (256.0 * 256.0) * (scaleSectorH / scaleXY).
	grad := -float64(heiNum) / (256.0 * 256.0) * (scaleSectorH / scaleXY)

	return math.Abs(grad), grad * nx, grad * ny, true
}
