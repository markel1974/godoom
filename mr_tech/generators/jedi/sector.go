package jedi

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/markel1974/godoom/mr_tech/model/geometry"
)

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
	SlopedFloor    [3]float64
	SlopedCeiling  [3]float64
}

// NewSector creates and returns a new Sector instance with the specified id and index, initializing default values.
func NewSector(id string, index int) *Sector {
	return &Sector{
		Id:             id,
		Index:          index,
		FloorY:         -1,
		CeilingY:       -1,
		FloorTexture:   -1,
		CeilingTexture: -1,
	}
}

// SetCeiling processes and sets the ceiling properties of a sector based on the provided tokens.
func (s *Sector) SetCeiling(tokens []string) {
	if len(tokens) < 3 {
		fmt.Printf("Invalid CEILING property: '%s' in line: %v\n", tokens[1], tokens)
	}
	subKey := strings.ToUpper(tokens[1])
	switch subKey {
	case "Y":
		// Formato compresso Outlaws: CEILING Y [Alt] [Tex] [ScaleX] [ScaleY] [Light]
		if len(tokens) >= 4 {
			s.CeilingY, _ = strconv.ParseFloat(tokens[2], 64)
			s.CeilingY = -s.CeilingY
			if tokens[3] != "-" {
				s.CeilingTexture, _ = strconv.Atoi(tokens[3])
			}
		}
	case "ALTITUDE":
		s.CeilingY, _ = strconv.ParseFloat(tokens[2], 64)
	case "TEXTURE", "TEX", "TEXTURE:":
		if tokens[2] != "-" {
			s.CeilingTexture, _ = strconv.Atoi(tokens[2])
		}
	default:
		fmt.Printf("Unknown CEILING sub-property: '%s' in line: %v\n", subKey, tokens)
	}

}

// SetFloor parses and sets the floor properties of the Sector based on the provided tokens.
func (s *Sector) SetFloor(tokens []string) {
	if len(tokens) < 3 {
		fmt.Printf("Invalid FLOOR property: '%s' in line: %v\n", tokens[1], tokens)
		return
	}
	subKey := strings.ToUpper(tokens[1])
	switch subKey {
	case "Y":
		// Formato compresso Outlaws: FLOOR Y [Alt] [Tex] [ScaleX] [ScaleY] [Light]
		if len(tokens) >= 4 {
			floorY, err := GetTokenFloatAt(tokens, 2)
			if err != nil {
				fmt.Printf("Invalid floor Y value: '%s' in line: %v\n", tokens[2], tokens)
				return
			}
			s.FloorY = -floorY
			if tokens[3] != "-" {
				floorTexture, err := GetTokenIntAt(tokens, 3)
				if err != nil {
					fmt.Printf("Invalid floor texture value: '%s' in line: %v\n", tokens[3], tokens)
					return
				}
				s.FloorTexture = floorTexture
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
	case "OFFSETS":
	case "SOUND":
	default:
		fmt.Printf("Unknown FLOOR sub-property: '%s' in line: %v\n", subKey, tokens)
	}
}

// AddFlags parses a slice of string tokens, converting them to integers and appending them to the Sector's Flags slice.
func (s *Sector) AddFlags(tokens []string) {
	// Alloca la slice in base al numero di parametri reali sulla riga
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

// AddWall adds a new Wall to the Sector by parsing the provided tokens and updating the Walls slice and WallIdx counter.
func (s *Sector) AddWall(tokens []string) {
	if s.WallIdx >= len(s.Walls) {
		fmt.Println("max wall reached!")
		return
	}
	wall := NewWall()
	wall.Parse(tokens)
	s.Walls[s.WallIdx] = wall
	s.WallIdx++
}

// AddVertices updates the Vertices slice of the Sector with coordinates parsed from the provided tokens.
func (s *Sector) AddVertices(tokens []string) {
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
