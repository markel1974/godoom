package lumps

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/markel1974/godoom/mr_tech/geometry"
)

// Slope represents a sloped surface defined by a sector index, pitch, and wall index.
type Slope struct {
	SectorIndex int
	Pitch       int
	WallIndex   int

	bestCeilWall   float64
	bestFloorWall  float64
	ceilGrad       float64
	floorGrad      float64
	ceilGX         float64
	floorGX        float64
	ceilGY         float64
	floorGY        float64
	floorQuota     float64
	ceilQuota      float64
	hasSlopedCeil  bool
	hasSlopedFloor bool
}

// GetGradient calculates and returns the angle of the slope in radians using the pitch value.
func (s *Slope) GetGradient() float64 {
	degrees := float64(s.Pitch) / 100.0
	radians := degrees * (math.Pi / 180.0)
	gradient := math.Tan(radians)
	if math.IsNaN(gradient) || math.IsInf(gradient, 0) {
		return 0 //TODO return false
	}
	return gradient //TODO return true
}

// Sector represents a spatial region with geometry, textures, physical properties, and relationships to walls and vertices.
type Sector struct {
	Id             string
	Index          int
	FloorY         float64
	CeilingY       float64
	FloorTexture   int
	CeilingTexture int
	LightLevel     float64
	Vertices       []geometry.XY
	Walls          []*Wall
	WallIdx        int
	Flags          []int
	Friction       float64
	Gravity        float64
	Elasticity     float64
	Velocity       [3]float64
	VAdjoin        int
	CMap           int
	SlopedFloor    *Slope
	SlopedCeiling  *Slope
}

// NewSector creates and returns a new Sector instance with the specified id and index, initializing default values.
func NewSector(id string, index int) *Sector {
	return &Sector{
		Id:             id,
		Index:          index,
		FloorY:         0,
		CeilingY:       0,
		FloorTexture:   -1,
		CeilingTexture: -1,
	}
}

// SetSlopedFloor sets the slope parameters of the sector's floor using the provided tokenized input.
func (sec *Sector) SetSlopedFloor(tokens []string) {
	if len(tokens) < 4 {
		fmt.Printf("Invalid SLOPEDFLOOR property in line: %v\n", tokens)
		return
	}
	sector, _ := strconv.Atoi(tokens[1])
	wall, _ := strconv.Atoi(tokens[2])
	pitch, _ := strconv.Atoi(tokens[3])
	sec.SlopedFloor = &Slope{SectorIndex: sector, Pitch: pitch, WallIndex: wall}
}

// SetSlopedCeiling sets the slope of the ceiling using parsed float values from the provided tokens slice.
func (sec *Sector) SetSlopedCeiling(tokens []string) {
	if len(tokens) < 4 {
		fmt.Printf("Invalid SLOPEDCEILING property in line: %v\n", tokens)
		return
	}
	sector, _ := strconv.Atoi(tokens[1])
	wall, _ := strconv.Atoi(tokens[2])
	pitch, _ := strconv.Atoi(tokens[3])
	sec.SlopedCeiling = &Slope{SectorIndex: sector, Pitch: pitch, WallIndex: wall}
}

// SetCeiling processes and sets the ceiling properties of a sector based on the provided tokens.
func (sec *Sector) SetCeiling(tokens []string) {
	if len(tokens) < 1 {
		fmt.Printf("Invalid CEILING property: '%s' in line: %v\n", tokens[1], tokens)
	}
	subKey := strings.ToUpper(tokens[1])
	switch subKey {
	case "Y":
		//FLOOR Y [Quota Z] [ID Texture] [U-Offset] [V-Offset] [Angolo di Rotazione]
		if len(tokens) < 4 {
			fmt.Printf("Invalid CEILING Y property in line: %v\n", tokens)
			return
		}
		ceilingY, err := GetTokenFloatAt(tokens, 2)
		if err != nil {
			fmt.Printf("Invalid ceiling Y value: '%s' in line: %v\n", tokens[2], tokens)
			return
		}
		sec.CeilingY = -ceilingY
		ceilingData, err := GetTokenStringAt(tokens, 3)
		if err != nil {
			fmt.Printf("Invalid CEILING texture value: '%s' in line: %v\n", tokens[3], tokens)
			return
		}
		if ceilingData == "-" {
			sec.CeilingTexture = -1
		} else {
			ceilingTexture, err := strconv.Atoi(ceilingData)
			if err != nil {
				fmt.Printf("Invalid CEILINGG texture value: '%s' in line: %v\n", tokens[3], tokens)
				return
			}
			sec.CeilingTexture = ceilingTexture
		}

	case "ALTITUDE":
		ceilingY, err := GetTokenFloatAt(tokens, 2)
		if err != nil {
			fmt.Printf("Invalid ALTITUDE value: '%s' in line: %v\n", tokens[2], tokens)
			return
		}
		sec.CeilingY = ceilingY
	case "TEXTURE", "TEX", "TEXTURE:":
		if tokens[2] != "-" {
			sec.CeilingTexture, _ = strconv.Atoi(tokens[2])
		}
	default:
		fmt.Printf("Unknown CEILING sub-property: '%s' in line: %v\n", subKey, tokens)
	}

}

// SetFloor parses and sets the floor properties of the Sector based on the provided tokens.
func (sec *Sector) SetFloor(tokens []string) {
	if len(tokens) < 1 {
		fmt.Printf("Invalid FLOOR property: '%s' in line: %v\n", tokens[1], tokens)
		return
	}
	subKey := strings.ToUpper(tokens[1])
	switch subKey {
	case "Y":
		//FLOOR Y [Quota Z] [ID Texture] [U-Offset] [V-Offset] [Angolo di Rotazione]
		if len(tokens) < 4 {
			fmt.Printf("Invalid FLOOR Y property in line: %v\n", tokens)
			return
		}
		floorY, err := GetTokenFloatAt(tokens, 2)
		if err != nil {
			fmt.Printf("Invalid FLOOR Y value: '%s' in line: %v\n", tokens[2], tokens)
			return
		}
		sec.FloorY = -floorY
		if tokens[3] != "-" {
			floorTexture, err := GetTokenIntAt(tokens, 3)
			if err != nil {
				fmt.Printf("Invalid FLOOR texture value: '%s' in line: %v\n", tokens[3], tokens)
				return
			}
			sec.FloorTexture = floorTexture
		}

	case "ALTITUDE":
		floorY, err := GetTokenFloatAt(tokens, 2)
		if err != nil {
			fmt.Printf("Invalid floor Y value: '%s' in line: %v\n", tokens[2], tokens)
			return
		}
		sec.FloorY = floorY
	case "TEXTURE", "TEX", "TEXTURE:":
		floorTexture, err := GetTokenStringAt(tokens, 2)
		if err != nil {
			fmt.Printf("Invalid floor texture value: '%s' in line: %v\n", tokens[2], tokens)
			return
		}
		if floorTexture != "-" {
			floorTextureId, err := strconv.Atoi(floorTexture)
			if err != nil {
				fmt.Printf("Invalid floor texture value: '%s' in line: %v\n", tokens[2], tokens)
				return
			}
			sec.FloorTexture = floorTextureId
		}
	case "OFFSETS":
	case "SOUND":
	default:
		fmt.Printf("Unknown FLOOR sub-property: '%s' in line: %v\n", subKey, tokens)
	}
}

// WallAdd adds a new Wall to the Sector by parsing the provided tokens and updating the Walls slice and WallIdx counter.
func (sec *Sector) WallAdd(tokens []string) {
	if sec.WallIdx >= len(sec.Walls) {
		fmt.Println("max wall reached!")
		return
	}
	wall := NewWall()
	wall.Parse(tokens)
	sec.Walls[sec.WallIdx] = wall
	sec.WallIdx++
}

// VertexAdd updates the Vertices slice of the Sector with coordinates parsed from the provided tokens.
func (sec *Sector) VertexAdd(tokens []string) {
	var err error
	var ptX, ptZ float64
	ord := -1
	found := false
	for i := 0; i < len(tokens); i++ {
		next := i + 1
		if next >= len(tokens) {
			break
		}
		key := strings.ToUpper(strings.TrimSpace(tokens[i]))
		switch key {
		case "#":
			i++
			ord, err = GetTokenIntAt(tokens, next)
			if err != nil {
				fmt.Printf("doApplyVertices: invalid token id: %s\n", err.Error())
				return
			}
		case "X:":
			i++
			ptX, err = GetTokenFloatAt(tokens, next)
			if err != nil {
				fmt.Printf("doApplyVertices: invalid token id: %s\n", err.Error())
				return
			}
			found = true
		case "Z:":
			i++
			ptZ, err = GetTokenFloatAt(tokens, next)
			if err != nil {
				fmt.Printf("doApplyVertices: invalid token id: %s\n", err.Error())
				return
			}
			found = true
		default:
			fmt.Println("doApplyVertices: unknown token key:", key)
		}
	}
	if !found {
		ptX, err = GetTokenFloatAt(tokens, 1)
		if err != nil {
			fmt.Printf("doApplyVertices: invalid token id: %s\n", err.Error())
			return
		}
		ptZ, err = GetTokenFloatAt(tokens, 2)
		if err != nil {
			fmt.Printf("doVertices: invalid token id: %s\n", err.Error())
			return
		}
	}

	if ord < 0 || ord > len(sec.Vertices) {
		fmt.Printf("doApplyVertices: invalid vertex id %d\n", ord)
		return
	}
	sec.Vertices[ord] = geometry.XY{X: ptX, Y: ptZ}
}

// VerticesInitialize initializes the Vertices slice with the size specified by the second token in the input array.
func (sec *Sector) VerticesInitialize(tokens []string) {
	sec.Vertices = nil
	vCount, err := GetTokenIntAt(tokens, 1)
	if err != nil {
		fmt.Printf("doVertices: invalid token at 1 err: %s\n", err.Error())
		return
	}
	sec.Vertices = make([]geometry.XY, vCount)
}

// WallsInitialize initializes the Walls slice of the Sector based on the count provided in the tokens and resets WallIdx.
func (sec *Sector) WallsInitialize(tokens []string) {
	var err error
	sec.Walls = nil
	sec.WallIdx = 0
	wCount, err := GetTokenIntAt(tokens, 1)
	if err != nil {
		fmt.Printf("doWalls: invalid token at 1 err: %s\n", err.Error())
		return
	}
	sec.Walls = make([]*Wall, wCount)
}

// FlagsAdd parses a slice of string tokens, converting them to integers and appending them to the Sector's Flags slice.
func (sec *Sector) FlagsAdd(tokens []string) {
	flagCount := len(tokens) - 1
	sec.Flags = make([]int, 0, flagCount)
	for i := 1; i < len(tokens); i++ {
		val, err := GetTokenIntAt(tokens, i)
		if err != nil {
			fmt.Printf("doFlags: invalid token at %d: %s\n", i, err.Error())
			continue
		}
		sec.Flags = append(sec.Flags, val)
	}
}

// IsSky checks if the sector is marked as "sky" based on its flags. Returns true if the first flag bit is set, otherwise false.
func (sec *Sector) IsSky() bool {
	if len(sec.Flags) < 1 {
		return false
	}
	return (sec.Flags[0] & 1) != 0
}

// IsAbyss determines if the sector has the "abyss" flag set, based on its Flags property. Returns true if set, else false.
func (sec *Sector) IsAbyss() bool {
	if len(sec.Flags) < 1 {
		return false
	}
	return (sec.Flags[0] & 2) != 0
}

// IsCCW determines if the sector's walls are ordered counter-clockwise (CCW) based on their vertices and area calculation.
func (sec *Sector) IsCCW() bool {
	n := len(sec.Walls)
	if n < 3 {
		return false
	}
	var twiceArea float64
	for _, w := range sec.Walls {
		if w.LeftVertex < 0 || w.LeftVertex >= len(sec.Vertices) || w.RightVertex < 0 || w.RightVertex >= len(sec.Vertices) {
			return false
		}
		v1 := sec.Vertices[w.LeftVertex]
		v2 := sec.Vertices[w.RightVertex]
		twiceArea += v1.X*v2.Y - v2.X*v1.Y
	}
	// Coordinate screen-space: Y cresce verso il basso.
	// Area positiva => contorno antiorario.
	return twiceArea > 0
}

func (sec *Sector) GetFlag(index int, mask uint32) (uint32, bool) {
	if index < 0 && index > len(sec.Flags) {
		return 0, false
	}
	return uint32(sec.Flags[1]) & mask, true
}

// ComputePlane calculates the slope plane of a sector ceiling or floor, returning optimal wall index, gradient, quota, and status.
func (sec *Sector) ComputePlane(level *Level, isCeil bool) (int, float64, float64, float64, float64, bool) {
	sl := sec.SlopedFloor
	if isCeil {
		sl = sec.SlopedCeiling
	}

	if sl == nil || sl.SectorIndex < 0 || sl.SectorIndex >= len(level.Sectors) {
		return -1, 0, 0, 0, 0, false
	}

	pSec := level.Sectors[sl.SectorIndex]
	if sl.WallIndex < 0 || sl.WallIndex >= len(pSec.Walls) {
		return -1, 0, 0, 0, 0, false
	}

	pWall := pSec.Walls[sl.WallIndex]
	if !pWall.ValidWallVertices(len(pSec.Vertices)) {
		return -1, 0, 0, 0, 0, false
	}

	pv1 := pSec.Vertices[pWall.LeftVertex]
	pv2 := pSec.Vertices[pWall.RightVertex]

	pdx := pv2.X - pv1.X
	pdy := pv2.Y - pv1.Y
	pLen := math.Hypot(pdx, pdy)
	if pLen == 0 {
		return -1, 0, 0, 0, 0, false
	}

	var pnX, pnY float64
	if pSec.IsCCW() {
		pnX, pnY = -pdy/pLen, pdx/pLen
	} else {
		pnX, pnY = pdy/pLen, -pdx/pLen
	}

	grad := sl.GetGradient()
	gx := pnX * grad
	gy := pnY * grad

	baseZ := pSec.FloorY
	if isCeil {
		baseZ = pSec.CeilingY
	}

	bestWall := -1
	bestAbsDot := -1.0
	bestDot := 0.0

	if sl.SectorIndex == sec.Index && pWall.ValidWallVertices(len(sec.Vertices)) {
		bestWall = sl.WallIndex
		bestDot = 1.0
	} else {
		for wIdx, w := range sec.Walls {
			if !w.ValidWallVertices(len(sec.Vertices)) {
				continue
			}

			v1 := sec.Vertices[w.LeftVertex]
			v2 := sec.Vertices[w.RightVertex]

			dx := v2.X - v1.X
			dy := v2.Y - v1.Y
			wLen := math.Hypot(dx, dy)
			if wLen < 1e-4 {
				continue
			}

			var wnX, wnY float64
			if sec.IsCCW() {
				wnX, wnY = -dy/wLen, dx/wLen
			} else {
				wnX, wnY = dy/wLen, -dx/wLen
			}

			dot := wnX*pnX + wnY*pnY
			if math.Abs(dot) > bestAbsDot {
				bestAbsDot = math.Abs(dot)
				bestDot = dot
				bestWall = wIdx
			}
		}
	}

	if bestWall < 0 {
		return -1, 0, 0, 0, 0, false
	}

	vStart := sec.Vertices[sec.Walls[bestWall].LeftVertex]
	quotaStart := baseZ -
		(gx*(vStart.X-pv1.X) +
			gy*(vStart.Y-pv1.Y))

	effGrad := grad
	if bestDot < 0 {
		effGrad = -grad
	}

	return bestWall, effGrad, gx, gy, quotaStart, true
}
