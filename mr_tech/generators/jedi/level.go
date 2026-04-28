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
	VertexIndex int
	Adjoin      int
	MidTexture  int
	TopTexture  int
	BotTexture  int
}

func NewWall() *Wall {
	return &Wall{
		Adjoin:     -1,
		MidTexture: -1,
		TopTexture: -1,
		BotTexture: -1,
	}
}

// Sector represents a sector in a level, containing geometric and texture data, as well as lighting and wall information.
type Sector struct {
	Id             int
	FloorY         float64
	CeilingY       float64
	FloorTexture   string
	CeilingTexture string
	LightLevel     float64
	Vertices       []geometry.XY
	Walls          []*Wall
}

func NewSector(id int) *Sector {
	return &Sector{Id: id}
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
		keyword := strings.ToUpper(tokens[0])

		switch keyword {
		case "TEXTURES":
			mode = modeTextures
			if len(tokens) >= 2 {
				count, _ := strconv.Atoi(tokens[1])
				p.Textures = make([]string, 0, count) // Usa p.Textures
			}

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
						sector.FloorTexture = tokens[2]
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
						sector.CeilingTexture = tokens[2]
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

		default:
			if mode == modeTextures && len(tokens) > 1 {
				p.Textures = append(p.Textures, tokens[1]) // Usa p.Textures
			} else if sector != nil {
				if mode == modeVertices && len(tokens) >= 3 {
					x, _ := strconv.ParseFloat(tokens[1], 64)
					y, _ := strconv.ParseFloat(tokens[2], 64)
					sector.Vertices = append(sector.Vertices, geometry.XY{X: x, Y: y})
				} else if mode == modeWalls && len(tokens) >= 2 {
					wall := p.createWall(tokens)
					sector.Walls = append(sector.Walls, wall)
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
		key := strings.ToUpper(tokens[i])
		switch key {
		case "LEFT:":
			if i+1 < len(tokens) {
				wall.VertexIndex, _ = strconv.Atoi(tokens[i+1])
				i++
			}
		case "ADJOIN:":
			if i+1 < len(tokens) {
				wall.Adjoin, _ = strconv.Atoi(tokens[i+1])
				i++
			}
		case "MID:":
			if i+1 < len(tokens) {
				val := tokens[i+1]
				if val != "-1" && val != "-" {
					texID, _ := strconv.Atoi(val)
					wall.MidTexture = texID
				}
				i++
			}
		case "TOP:":
			if i+1 < len(tokens) {
				val := tokens[i+1]
				if val != "-1" && val != "-" {
					texID, _ := strconv.Atoi(val)
					wall.TopTexture = texID
				}
				i++
			}
		case "BOT:":
			if i+1 < len(tokens) {
				val := tokens[i+1]
				if val != "-1" && val != "-" {
					texID, _ := strconv.Atoi(val)
					wall.BotTexture = texID
				}
				i++
			}
		}
	}

	return wall
}
