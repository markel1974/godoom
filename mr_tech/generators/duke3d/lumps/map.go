package lumps

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"

	"github.com/markel1974/godoom/mr_tech/geometry"
)

type MapHeader struct {
	Version    int32
	PosX       int32
	PosY       int32
	PosZ       int32
	Ang        int16
	CurSectNum int16
}

type Sector struct {
	WallPtr         int16
	WallNum         int16
	CeilingZ        int32
	FloorZ          int32
	CeilingStat     int16
	FloorStat       int16
	CeilingPicNum   int16
	CeilingHeiNum   int16
	CeilingShade    int8
	CeilingPal      uint8
	CeilingXPanning uint8
	CeilingYPanning uint8
	FloorPicNum     int16
	FloorHeiNum     int16
	FloorShade      int8
	FloorPal        uint8
	FloorXPanning   uint8
	FloorYPanning   uint8
	Visibility      uint8
	Filler          uint8
	LoTag           int16
	HiTag           int16
	Extra           int16
}

func (s *Sector) IsSlopedFloor() bool {
	return s.FloorStat&2 != 0 && s.FloorHeiNum != 0
}

func (s *Sector) IsSlopedCeiling() bool {
	return s.CeilingStat&2 != 0 && s.CeilingHeiNum != 0
}

func (s *Sector) IsSkyCeiling() bool {
	return s.CeilingStat&1 != 0
}

func (s *Sector) IsSkyFloor() bool {
	return s.FloorStat&1 != 0
}

type Wall struct {
	X          int32
	Y          int32
	Point2     int16
	NextWall   int16
	NextSector int16
	CStat      int16
	PicNum     int16
	OverPicNum int16
	Shade      int8
	Pal        uint8
	XRepeat    uint8
	YRepeat    uint8
	XPanning   uint8
	YPanning   uint8
	LoTag      int16
	HiTag      int16
	Extra      int16
}

func (w *Wall) IsMasked() bool {
	return w.CStat&16 != 0
}

func (w *Wall) IsOneWay() bool {
	return w.CStat&32 != 0
}

func (w *Wall) IsTranslucent() bool {
	return w.CStat&128 != 0
}

type Sprite struct {
	X        int32
	Y        int32
	Z        int32
	CStat    int16
	PicNum   int16
	Shade    int8
	Pal      uint8
	ClipDist uint8
	Filler   uint8
	XRepeat  uint8
	YRepeat  uint8
	XOffset  int8
	YOffset  int8
	SectNum  int16
	StatNum  int16
	Ang      int16
	Owner    int16
	XVel     int16
	YVel     int16
	ZVel     int16
	LoTag    int16
	HiTag    int16
	Extra    int16
}

type Level struct {
	Header  MapHeader
	Sectors []Sector
	Walls   []Wall
	Sprites []Sprite
}

func LoadMap(r io.Reader) (*Level, error) {
	level := &Level{}

	if err := binary.Read(r, binary.LittleEndian, &level.Header); err != nil {
		return nil, fmt.Errorf("reading map header: %w", err)
	}

	if level.Header.Version < 7 || level.Header.Version > 9 {
		return nil, fmt.Errorf("unsupported build map version: %d (expected 7, 8 or 9)", level.Header.Version)
	}

	var numSectors int16
	if err := binary.Read(r, binary.LittleEndian, &numSectors); err != nil {
		return nil, fmt.Errorf("reading numsectors: %w", err)
	}
	if numSectors < 0 {
		return nil, fmt.Errorf("invalid sector count: %d", numSectors)
	}
	level.Sectors = make([]Sector, numSectors)
	if err := binary.Read(r, binary.LittleEndian, level.Sectors); err != nil {
		return nil, fmt.Errorf("reading sectors: %w", err)
	}

	var numWalls int16
	if err := binary.Read(r, binary.LittleEndian, &numWalls); err != nil {
		return nil, fmt.Errorf("reading numwalls: %w", err)
	}
	if numWalls < 0 {
		return nil, fmt.Errorf("invalid wall count: %d", numWalls)
	}
	level.Walls = make([]Wall, numWalls)
	if err := binary.Read(r, binary.LittleEndian, level.Walls); err != nil {
		return nil, fmt.Errorf("reading walls: %w", err)
	}

	var numSprites int16
	if err := binary.Read(r, binary.LittleEndian, &numSprites); err != nil {
		// Sprites are optional at EOF
		return level, nil
	}
	if numSprites > 0 {
		level.Sprites = make([]Sprite, numSprites)
		if err := binary.Read(r, binary.LittleEndian, level.Sprites); err != nil {
			return nil, fmt.Errorf("reading sprites: %w", err)
		}
	}

	return level, nil
}

func (l *Level) GetSector(idx int) *Sector {
	if idx < 0 || idx >= len(l.Sectors) {
		return nil
	}
	return &l.Sectors[idx]
}

func (l *Level) GetWall(idx int) *Wall {
	if idx < 0 || idx >= len(l.Walls) {
		return nil
	}
	return &l.Walls[idx]
}

// ComputeFloorSlope calculates the gradient vector (GX, GY) and magnitude for a sloped floor.
// In Build Engine, slope pivots around the sector's first wall (WallPtr).
func (l *Level) ComputeFloorSlope(secIdx int, scaleSectorH float64, scaleXY float64, flipY bool) (gradient, gx, gy float64, ok bool) {
	sec := l.GetSector(secIdx)
	if sec == nil || !sec.IsSlopedFloor() {
		return 0, 0, 0, false
	}
	return l.computeSlope(int(sec.WallPtr), sec.FloorHeiNum, scaleSectorH, scaleXY, flipY)
}

// ComputeCeilingSlope calculates the gradient vector (GX, GY) and magnitude for a sloped ceiling.
func (l *Level) ComputeCeilingSlope(secIdx int, scaleSectorH float64, scaleXY float64, flipY bool) (gradient, gx, gy float64, ok bool) {
	sec := l.GetSector(secIdx)
	if sec == nil || !sec.IsSlopedCeiling() {
		return 0, 0, 0, false
	}
	return l.computeSlope(int(sec.WallPtr), sec.CeilingHeiNum, scaleSectorH, scaleXY, flipY)
}

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
	if lenW < 1e-6 {
		return 0, 0, 0, false
	}

	// In Build engine, sector interior is to the right of the wall when Y goes down.
	// When Y is flipped (Cartesian Y up), sector interior is to the left of the wall: (-dy/lenW, dx/lenW).
	// Without flip: (dy/lenW, -dx/lenW).
	var nx, ny float64
	if flipY {
		nx = -dy / lenW
		ny = dx / lenW
	} else {
		nx = dy / lenW
		ny = -dx / lenW
	}

	// Height in godoom: Z_godoom = -Z_build / 16.0 * scaleSectorH.
	// dz / dd = -float64(heiNum) / (4096.0 * 16.0) * scaleSectorH / scaleXY.
	grad := -float64(heiNum) / (4096.0 * 16.0) * (scaleSectorH / scaleXY)

	return math.Abs(grad), grad * nx, grad * ny, true
}
