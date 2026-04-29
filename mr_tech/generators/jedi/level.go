package jedi

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/markel1974/godoom/mr_tech/model/geometry"
)

// modeNone represents the default mode with no specific operation.
// modeTextures indicates the mode related to texture lookup operations.
// modeVertices indicates the mode for handling vertices data.
// modeWalls indicates the mode for processing wall-related data.
const (
	modeNone     = iota
	modeTextures // Nuovo stato per la tabella di lookup
	modeVertices
	modeWalls
)

// Wall represents a wall in a sector, containing vertex and texture information for rendering and adjacency.
type Wall struct {
	//Id          int
	LeftVertex  int
	RightVertex int
	Adjoin      int
	MidTexture  int
	TopTexture  int
	BotTexture  int
	SignTexture int
	Flags       int
	Light       int
}

func NewWall() *Wall {
	return &Wall{
		//Id:          -1,
		LeftVertex:  -1,
		RightVertex: -1,
		Adjoin:      -1,
		MidTexture:  -1,
		TopTexture:  -1,
		BotTexture:  -1,
		SignTexture: -1,
		Flags:       0,
		Light:       0,
	}
}

// Level represents a game level containing textures and multiple sectors for defining its structure.
type Level struct {
	Textures []string
	Sectors  []*Sector
}

// NewLevel creates and returns a new instance of Level with initialized empty slices for Textures and Sectors.
func NewLevel() *Level {
	return &Level{
		Textures: make([]string, 0),
		Sectors:  make([]*Sector, 0),
	}
}

// GetTexture retrieves the texture name corresponding to the given texture ID from the Level's texture list.
// It returns an empty string if the ID is invalid or out of range.
func (p *Level) GetTexture(id int) string {
	if id >= 0 && id < len(p.Textures) {
		return p.Textures[id]
	}
	fmt.Printf("unknown texture id %d\n", id)
	return ""
}

// Parse reads level data from the given io.Reader, parses its content, and populates the Level structure accordingly.
func (p *Level) Parse(r io.Reader) error {
	var sector *Sector
	mode := modeNone

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		tokens := strings.Fields(line)
		keyword := CleanKey(tokens[0])

		switch keyword {
		case "SECTOR":
			if sector != nil {
				p.Sectors = append(p.Sectors, sector) // Usa p.Sectors
			}
			id, _ := strconv.Atoi(tokens[1])
			sector = NewSector(id)
			mode = modeNone

		// RIPRISTINATO: Lettura di Pavimenti, Soffitti e Illuminazione
		case "AMBIENT":
			if sector != nil && len(tokens) >= 2 {
				sector.LightLevel, _ = strconv.ParseFloat(tokens[1], 64)
			}
		case "FLOOR":
			if sector != nil && len(tokens) >= 3 {
				switch strings.ToUpper(tokens[1]) {
				case "ALTITUDE":
					sector.FloorY, _ = strconv.ParseFloat(tokens[2], 64)
				case "TEXTURE":
					if tokens[2] != "-" {
						texId, _ := strconv.Atoi(tokens[2])
						sector.FloorTexture = texId
					}
				}
			}
		case "CEILING":
			if sector != nil && len(tokens) >= 3 {
				switch strings.ToUpper(tokens[1]) {
				case "ALTITUDE":
					sector.CeilingY, _ = strconv.ParseFloat(tokens[2], 64)
				case "TEXTURE":
					if tokens[2] != "-" {
						texId, _ := strconv.Atoi(tokens[2])
						sector.CeilingTexture = texId
					}
				}
			}
		case "VERTICES":
			mode = modeVertices
			if sector != nil && len(tokens) >= 2 {
				count, _ := strconv.Atoi(tokens[1])
				sector.Vertices = make([]geometry.XY, 0, count)
			}
		case "WALLS":
			mode = modeWalls
			if sector != nil && len(tokens) >= 2 {
				count, _ := strconv.Atoi(tokens[1])
				sector.Walls = make([]*Wall, 0, count)
			}
		case "TEXTURES":
			mode = modeTextures
			if len(tokens) >= 2 {
				count, _ := strconv.Atoi(tokens[1])
				p.Textures = make([]string, 0, count) // Usa p.Textures
			}
		case "WALL":
			if sector != nil {
				wall := p.createWall(tokens)
				sector.Walls = append(sector.Walls, wall)
			}
		case "TEXTURE":
			if len(tokens) >= 3 {
				if id, err := strconv.Atoi(tokens[3]); err == nil {
					for len(p.Textures) <= id {
						p.Textures = append(p.Textures, "")
					}
					p.Textures[id] = tokens[1]
				}
			}
		case "FLAGS":
			if sector != nil && len(tokens) >= 4 {
				sector.Flags[0], _ = strconv.Atoi(tokens[1])
				sector.Flags[1], _ = strconv.Atoi(tokens[2])
				sector.Flags[2], _ = strconv.Atoi(tokens[3])
			}
		default:
			if mode == modeVertices {
				if sector != nil && len(tokens) >= 2 {
					var ptX, ptY float64
					if strings.Contains(strings.ToUpper(line), "X:") {
						for i := 0; i < len(tokens); i++ {
							next := i + 1
							if next >= len(tokens) {
								break
							}
							key := CleanKey(tokens[i])
							switch key {
							case "X":
								ptX, _ = strconv.ParseFloat(tokens[next], 64)
							case "Z":
								ptY, _ = strconv.ParseFloat(tokens[next], 64)
							}
						}
					} else if len(tokens) >= 3 {
						ptX, _ = strconv.ParseFloat(tokens[1], 64)
						ptY, _ = strconv.ParseFloat(tokens[2], 64)
					}
					sector.Vertices = append(sector.Vertices, geometry.XY{X: ptX, Y: ptY})
				}
			}
		}
	}

	if sector != nil {
		p.Sectors = append(p.Sectors, sector)
	}

	return scanner.Err()
}

// createWall constructs and initializes a Wall instance based on the provided tokens representing wall attributes.
func (p *Level) createWall(tokens []string) *Wall {
	wall := NewWall()
	for i := 0; i < len(tokens); i++ {
		key := tokens[i]
		if !strings.Contains(key, ":") {
			continue
		}
		switch key {
		case "WALK:":
			//TODO
		case "MIRROR:":
			//TODO
		case "LEFT:":
			i++
			if i < len(tokens) {
				val := tokens[i]
				wall.LeftVertex, _ = strconv.Atoi(val)
			}
		case "RIGHT:":
			i++
			if i < len(tokens) {
				val := tokens[i]
				wall.RightVertex, _ = strconv.Atoi(val)
			}
		case "ADJOIN:":
			i++
			if i < len(tokens) {
				val := tokens[i]
				wall.Adjoin, _ = strconv.Atoi(val)
			}
		case "MID:":
			i++
			if i < len(tokens) {
				val := tokens[i]
				if val != "-1" && val != "-" {
					texID, _ := strconv.Atoi(val)
					wall.MidTexture = texID
				}
			}
		case "TOP:":
			i++
			if i < len(tokens) {
				val := tokens[i]
				if val != "-1" && val != "-" {
					texID, _ := strconv.Atoi(val)
					wall.TopTexture = texID
				}
			}
		case "BOT:":
			i++
			if i < len(tokens) {
				val := tokens[i]
				if val != "-1" && val != "-" {
					texID, _ := strconv.Atoi(val)
					wall.BotTexture = texID
				}
			}
		case "SIGN:":
			i++
			if i < len(tokens) {
				val := tokens[i]
				if val != "-1" && val != "-" {
					texID, _ := strconv.Atoi(val)
					wall.SignTexture = texID
				}
			}
		case "FLAGS:":
			i++
			if i < len(tokens) {
				val := tokens[i]
				wall.Flags, _ = strconv.Atoi(val)
			}
		case "LIGHT:":
			i++
			if i < len(tokens) {
				val := tokens[i]
				wall.Light, _ = strconv.Atoi(val)
			}
		default:
			fmt.Println("Unknown wall attribute: ", key)
		}
	}

	return wall
}
