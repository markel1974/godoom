package jedi

import (
	"fmt"
	"strings"

	"github.com/markel1974/godoom/mr_tech/model/geometry"
)

// Wall represents a segment of a level boundary, containing various texture, lighting, and adjacency properties.
type Wall struct {
	Id          string
	LeftVertex  int
	RightVertex int
	Adjoin      int
	MidTexture  int
	TopTexture  int
	BotTexture  int
	SignTexture int
	Flags       int
	Light       int
	V1          int
	V2          int
	Overlay     int
	DAdjoin     int
	DMirror     int
	OffsetX     float64
	OffsetY     float64
}

// NewWall creates and returns a pointer to a new Wall instance with default field values initialized.
func NewWall() *Wall {
	return &Wall{
		LeftVertex:  -1,
		RightVertex: -1,
		Adjoin:      -1,
		MidTexture:  -1,
		TopTexture:  -1,
		BotTexture:  -1,
		SignTexture: -1,
		V1:          -1,
		V2:          -1,
		Flags:       0,
		Light:       0,
		Overlay:     -1,
		DAdjoin:     -1,
		DMirror:     -1,
	}
}

func (w *Wall) Parse(tokens []string) {
	for i := 0; i < len(tokens); i++ {
		var err error
		key := strings.ToUpper(strings.TrimSpace(tokens[i]))
		if !strings.Contains(key, ":") {
			continue
		}
		switch key {
		case "NAME:":
		case "WALL:":
			w.Id, _ = GetTokenStringAt(tokens, 1)
		case "WALK:":
		case "MIRROR:":
		case "LEFT:":
			i++
			w.LeftVertex, err = GetTokenIntAt(tokens, i)
			if err != nil {
				fmt.Printf("doWall: LEFT invalid token id at %d: %s\n", i, err.Error())
			}
		case "RIGHT:":
			i++
			w.RightVertex, err = GetTokenIntAt(tokens, i)
			if err != nil {
				fmt.Printf("doWall: RIGHT invalid token id at %d: %s\n", i, err.Error())
			}
		case "ADJOIN:":
			i++
			w.Adjoin, err = GetTokenIntAt(tokens, i)
			if err != nil {
				fmt.Printf("doWall: ADJOIN invalid token id at %d: %s\n", i, err.Error())
			}
		case "MID:":
			i++
			w.MidTexture, err = GetTokenIntAt(tokens, i)
			if err != nil {
				w.MidTexture = -1
			}
		case "TOP:":
			i++
			w.TopTexture, err = GetTokenIntAt(tokens, i)
			if err != nil {
				w.TopTexture = -1
			}
		case "BOT:":
			i++
			w.BotTexture, err = GetTokenIntAt(tokens, i)
			if err != nil {
				w.BotTexture = -1
			}
		case "SIGN:":
			i++
			w.SignTexture, err = GetTokenIntAt(tokens, i)
			if err != nil {
				w.SignTexture = -1
			}
		case "FLAGS:":
			i++
			w.Flags, err = GetTokenIntAt(tokens, i)
			if err != nil {
				fmt.Printf("doWall: FLAGS invalid token id at %d: %s\n", i, err.Error())
			}
		case "LIGHT:":
			i++
			w.Light, err = GetTokenIntAt(tokens, i)
			if err != nil {
				fmt.Printf("doWall: LIGHT invalid token id at %d: %s\n", i, err.Error())
			}
		default:
			fmt.Println("doWall: Unknown wall attribute: ", key)
		case "V1:":
			i++
			w.V1, err = GetTokenIntAt(tokens, i)
			if err != nil {
				fmt.Printf("doWall: V1 invalid token at %d: %s\n", i, err.Error())
			}
			w.LeftVertex = w.V1
		case "V2:":
			i++
			w.V2, err = GetTokenIntAt(tokens, i)
			if err != nil {
				fmt.Printf("doWall: V2 invalid token at %d: %s\n", i, err.Error())
			}
			w.RightVertex = w.V2
		case "OVERLAY:":
			i++
			w.Overlay, err = GetTokenIntAt(tokens, i)
			if err != nil {
				w.Overlay = -1
			}
		case "DADJOIN:":
			i++
			w.DAdjoin, err = GetTokenIntAt(tokens, i)
			if err != nil {
				w.DAdjoin = -1
			}
		case "DMIRROR:":
			i++
			w.DMirror, err = GetTokenIntAt(tokens, i)
			if err != nil {
				w.DMirror = -1
			}
		}
	}
}

// Sector represents a distinct area of a level, defined by its geometry, textures, light level, and properties.
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

// NewSector creates a new Sector instance with the provided identifier and initializes its fields to default values.
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

// AddFlags processes a slice of strings, extracts integer values after the first token, and stores them in the Flags field.
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

// AddWall parses token strings to construct a Wall object and adds it to the sector's Walls list.
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

// AddVertices parses vertex information from tokens and appends the resulting XY coordinates to the sector's vertices.
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

// IsSky checks if the sector is marked as a sky by evaluating the first flag in the Flags slice. Returns true or false.
func (s *Sector) IsSky() bool {
	if len(s.Flags) < 1 {
		return false
	}
	return (s.Flags[0] & 1) != 0
}

// IsAbyss checks if the sector has the "Abyss" flag set based on its Flags slice and returns true if it is set.
func (s *Sector) IsAbyss() bool {
	if len(s.Flags) < 1 {
		return false
	}
	return (s.Flags[0] & 2) != 0
}
