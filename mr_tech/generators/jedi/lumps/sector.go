package lumps

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/markel1974/godoom/mr_tech/geometry"
)

// Sector represents a spatial region with geometry, textures, physical properties, and relationships to walls and vertices.
type Sector struct {
	Id             string
	Index          int
	FloorY         float64
	CeilingY       float64
	FloorTexture   int
	CeilingTexture int
	FloorOffsetX   float64
	FloorOffsetY   float64
	CeilOffsetX    float64
	CeilOffsetY    float64
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

	BestCeilWall   int
	BestFloorWall  int
	CeilGrad       float64
	FloorGrad      float64
	CeilGX         float64
	FloorGX        float64
	CeilGY         float64
	FloorGY        float64
	FloorQuota     float64
	CeilQuota      float64
	HasSlopedCeil  bool
	HasSlopedFloor bool
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

		BestCeilWall:   -1,
		BestFloorWall:  -1,
		CeilGrad:       0,
		FloorGrad:      0,
		CeilGX:         0,
		FloorGX:        0,
		CeilGY:         0,
		FloorGY:        0,
		FloorQuota:     0,
		CeilQuota:      0,
		HasSlopedCeil:  false,
		HasSlopedFloor: false,
	}
}

// SetSlopedFloor sets the slope parameters of the sector's floor using the provided tokenized input.
func (s *Sector) SetSlopedFloor(tokens []string) {
	if len(tokens) < 4 {
		fmt.Printf("Invalid SLOPEDFLOOR property in line: %v\n", tokens)
		return
	}
	sector, _ := strconv.Atoi(tokens[1])
	wall, _ := strconv.Atoi(tokens[2])
	pitch, _ := strconv.Atoi(tokens[3])
	s.SlopedFloor = NewSlope(sector, pitch, wall)
}

// SetSlopedCeiling sets the slope of the ceiling using parsed float values from the provided tokens slice.
func (s *Sector) SetSlopedCeiling(tokens []string) {
	if len(tokens) < 4 {
		fmt.Printf("Invalid SLOPEDCEILING property in line: %v\n", tokens)
		return
	}
	sector, _ := strconv.Atoi(tokens[1])
	wall, _ := strconv.Atoi(tokens[2])
	pitch, _ := strconv.Atoi(tokens[3])
	s.SlopedCeiling = NewSlope(sector, pitch, wall)
}

// SetCeiling processes and sets the ceiling properties of a sector based on the provided tokens.
func (s *Sector) SetCeiling(tokens []string) {
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
		s.CeilingY = -ceilingY
		ceilingData, err := GetTokenStringAt(tokens, 3)
		if err != nil {
			fmt.Printf("Invalid CEILING texture value: '%s' in line: %v\n", tokens[3], tokens)
			return
		}
		if ceilingData == "-" {
			s.CeilingTexture = -1
		} else {
			ceilingTexture, err := strconv.Atoi(ceilingData)
			if err != nil {
				fmt.Printf("Invalid CEILINGG texture value: '%s' in line: %v\n", tokens[3], tokens)
				return
			}
			s.CeilingTexture = ceilingTexture
		}
		if len(tokens) > 4 {
			if offX, err := GetTokenFloatAt(tokens, 4); err == nil {
				s.CeilOffsetX = offX
			}
		}
		if len(tokens) > 5 {
			if offY, err := GetTokenFloatAt(tokens, 5); err == nil {
				s.CeilOffsetY = offY
			}
		}

	case "ALTITUDE":
		ceilingY, err := GetTokenFloatAt(tokens, 2)
		if err != nil {
			fmt.Printf("Invalid ALTITUDE value: '%s' in line: %v\n", tokens[2], tokens)
			return
		}
		s.CeilingY = ceilingY
	case "TEXTURE", "TEX", "TEXTURE:":
		if tokens[2] != "-" {
			s.CeilingTexture, _ = strconv.Atoi(tokens[2])
		}
		if len(tokens) > 3 {
			if offX, err := GetTokenFloatAt(tokens, 3); err == nil {
				s.CeilOffsetX = offX
			}
		}
		if len(tokens) > 4 {
			if offY, err := GetTokenFloatAt(tokens, 4); err == nil {
				s.CeilOffsetY = offY
			}
		}
	default:
		fmt.Printf("Unknown CEILING sub-property: '%s' in line: %v\n", subKey, tokens)
	}

}

// SetFloor parses and sets the floor properties of the Sector based on the provided tokens.
func (s *Sector) SetFloor(tokens []string) {
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
		s.FloorY = -floorY
		if tokens[3] != "-" {
			floorTexture, err := GetTokenIntAt(tokens, 3)
			if err != nil {
				fmt.Printf("Invalid FLOOR texture value: '%s' in line: %v\n", tokens[3], tokens)
				return
			}
			s.FloorTexture = floorTexture
		}
		if len(tokens) > 4 {
			if offX, err := GetTokenFloatAt(tokens, 4); err == nil {
				s.FloorOffsetX = offX
			}
		}
		if len(tokens) > 5 {
			if offY, err := GetTokenFloatAt(tokens, 5); err == nil {
				s.FloorOffsetY = offY
			}
		}

	case "ALTITUDE":
		floorY, err := GetTokenFloatAt(tokens, 2)
		if err != nil {
			fmt.Printf("Invalid floor Y value: '%s' in line: %v\n", tokens[2], tokens)
			return
		}
		s.FloorY = floorY
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
			s.FloorTexture = floorTextureId
		}
		if len(tokens) > 3 {
			if offX, err := GetTokenFloatAt(tokens, 3); err == nil {
				s.FloorOffsetX = offX
			}
		}
		if len(tokens) > 4 {
			if offY, err := GetTokenFloatAt(tokens, 4); err == nil {
				s.FloorOffsetY = offY
			}
		}
	case "OFFSETS":
	case "SOUND":
	default:
		fmt.Printf("Unknown FLOOR sub-property: '%s' in line: %v\n", subKey, tokens)
	}
}

// WallAdd adds a new Wall to the Sector by parsing the provided tokens and updating the Walls slice and WallIdx counter.
func (s *Sector) WallAdd(tokens []string) {
	if s.WallIdx >= len(s.Walls) {
		fmt.Println("max wall reached!")
		return
	}
	wall := NewWall()
	wall.Parse(tokens)
	s.Walls[s.WallIdx] = wall
	s.WallIdx++
}

// VertexAdd updates the Vertices slice of the Sector with coordinates parsed from the provided tokens.
func (s *Sector) VertexAdd(tokens []string) {
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

	if ord < 0 || ord > len(s.Vertices) {
		fmt.Printf("doApplyVertices: invalid vertex id %d\n", ord)
		return
	}
	s.Vertices[ord] = geometry.XY{X: ptX, Y: ptZ}
}

// VerticesInitialize initializes the Vertices slice with the size specified by the second token in the input array.
func (s *Sector) VerticesInitialize(tokens []string) {
	s.Vertices = nil
	vCount, err := GetTokenIntAt(tokens, 1)
	if err != nil {
		fmt.Printf("doVertices: invalid token at 1 err: %s\n", err.Error())
		return
	}
	s.Vertices = make([]geometry.XY, vCount)
}

// WallsInitialize initializes the Walls slice of the Sector based on the count provided in the tokens and resets WallIdx.
func (s *Sector) WallsInitialize(tokens []string) {
	var err error
	s.Walls = nil
	s.WallIdx = 0
	wCount, err := GetTokenIntAt(tokens, 1)
	if err != nil {
		fmt.Printf("doWalls: invalid token at 1 err: %s\n", err.Error())
		return
	}
	s.Walls = make([]*Wall, wCount)
}

// FlagsAdd parses a slice of string tokens, converting them to integers and appending them to the Sector's Flags slice.
func (s *Sector) FlagsAdd(tokens []string) {
	flagCount := len(tokens) - 1
	s.Flags = make([]int, 0, flagCount)
	for i := 1; i < len(tokens); i++ {
		val, err := GetTokenIntAt(tokens, i)
		if err != nil {
			fmt.Printf("doFlags: invalid token at %d: %s\n", i, err.Error())
			continue
		}
		s.Flags = append(s.Flags, val)
	}
}

// IsSky checks if the sector is marked as "sky" based on its flags. Returns true if the first flag bit is set, otherwise false.
func (s *Sector) IsSky() bool {
	if len(s.Flags) < 1 {
		return false
	}
	return (s.Flags[0] & 1) != 0
}

// IsAbyss determines if the sector has the "abyss" flag set, based on its Flags property. Returns true if set, else false.
func (s *Sector) IsAbyss() bool {
	if len(s.Flags) < 1 {
		return false
	}
	return (s.Flags[0] & 2) != 0
}

// IsCCW determines if the sector's walls are ordered counter-clockwise (CCW) based on their vertices and area calculation.
func (s *Sector) IsCCW() bool {
	n := len(s.Walls)
	if n < 3 {
		return false
	}
	var twiceArea float64
	for _, w := range s.Walls {
		if w.LeftVertex < 0 || w.LeftVertex >= len(s.Vertices) || w.RightVertex < 0 || w.RightVertex >= len(s.Vertices) {
			return false
		}
		v1 := s.Vertices[w.LeftVertex]
		v2 := s.Vertices[w.RightVertex]
		twiceArea += v1.X*v2.Y - v2.X*v1.Y
	}
	// Coordinate screen-space: Y cresce verso il basso.
	// Area positiva => contorno antiorario.
	return twiceArea > 0
}

// GetFlag retrieves the flag at the specified index in the Flags slice and applies the given mask, returning the result and a success boolean.
func (s *Sector) GetFlag(index int, mask uint32) (uint32, bool) {
	if index < 0 || index >= len(s.Flags) {
		return 0, false
	}
	return uint32(s.Flags[index]) & mask, true
}

// Compute updates the sector's sloped floor and ceiling properties by recalculating planes based on related sectors.
func (s *Sector) Compute(level *Level) (float64, float64) {
	computedFloorY := s.computeBestFloor(level)
	computedCeilY := s.CeilingY
	if s.SlopedFloor != nil {
		if pSec := level.GetSector(s.SlopedFloor.SectorIndex); pSec != nil {
			s.BestFloorWall, s.FloorGrad, s.FloorGX, s.FloorGY, s.FloorQuota, s.HasSlopedFloor = s.computePlane(pSec, s.SlopedFloor, pSec.FloorY)
		}
	}
	if s.SlopedCeiling != nil {
		if pSec := level.GetSector(s.SlopedCeiling.SectorIndex); pSec != nil {
			s.BestCeilWall, s.CeilGrad, s.CeilGX, s.CeilGY, s.CeilQuota, s.HasSlopedCeil = s.computePlane(pSec, s.SlopedCeiling, pSec.CeilingY)
		}
	}
	if s.HasSlopedFloor {
		computedFloorY = s.FloorQuota
	}
	if s.HasSlopedCeil {
		computedCeilY = s.CeilQuota
	}
	return computedFloorY, computedCeilY
}

// computeBestFloor determines the best floor height for the sector by comparing it against adjacent sectors' floor heights.
func (s *Sector) computeBestFloor(level *Level) float64 {
	bestFloorY := s.FloorY
	if flag, ok := s.GetFlag(1, 0x80000000); ok && flag != 0 {
		foundAdjacent := false
		for _, wall := range s.Walls {
			adjacent := level.GetSector(wall.Adjoin)
			if adjacent == nil {
				continue
			}
			if !foundAdjacent || adjacent.FloorY > bestFloorY {
				bestFloorY = adjacent.FloorY
				foundAdjacent = true
			}
		}
	}
	return bestFloorY
}

// ComputePlane calculates the slope plane of a sector ceiling or floor, returning optimal wall index, gradient, quota, and status.
func (s *Sector) computePlane(pSec *Sector, sl *Slope, baseZ float64) (int, float64, float64, float64, float64, bool) {
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

	bestWall := -1
	bestAbsDot := -1.0
	bestDot := 0.0

	if sl.SectorIndex == s.Index && pWall.ValidWallVertices(len(s.Vertices)) {
		bestWall = sl.WallIndex
		bestDot = 1.0
	} else {
		for wIdx, w := range s.Walls {
			if !w.ValidWallVertices(len(s.Vertices)) {
				continue
			}

			v1 := s.Vertices[w.LeftVertex]
			v2 := s.Vertices[w.RightVertex]

			dx := v2.X - v1.X
			dy := v2.Y - v1.Y
			wLen := math.Hypot(dx, dy)
			if wLen <= 0 {
				continue
			}
			//if wLen < 1e-4 {
			//	continue
			//}

			var wnX, wnY float64
			if s.IsCCW() {
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

	vStart := s.Vertices[s.Walls[bestWall].LeftVertex]
	quotaStart := baseZ - (gx*(vStart.X-pv1.X) + gy*(vStart.Y-pv1.Y))
	effGrad := grad
	if bestDot < 0 {
		effGrad = -grad
	}

	return bestWall, effGrad, gx, gy, quotaStart, true
}

// EvaluateFloorZ returns the height Z in world coordinates of the sector's floor at point (x, y) given scaleH.
func (s *Sector) EvaluateFloorZ(level *Level, pt geometry.XY, scaleH float64) float64 {
	baseFloorY, _ := s.Compute(level)
	baseZ := -baseFloorY * scaleH
	if !s.HasSlopedFloor || s.BestFloorWall < 0 || s.BestFloorWall >= len(s.Walls) {
		return baseZ
	}
	refW := s.Walls[s.BestFloorWall]
	if !refW.ValidWallVertices(len(s.Vertices)) {
		return baseZ
	}
	vRef := s.Vertices[refW.LeftVertex]
	return baseZ + (s.FloorGX*(pt.X-vRef.X)+s.FloorGY*(pt.Y-vRef.Y))*scaleH
}
