package jedi

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/markel1974/godoom/mr_tech/model/geometry"
)

// modeNone represents the default mode state.
// modeTextures indicates the state used for texture lookup.
// modeVertices indicates the state used for vertex processing.
// modeWalls indicates the state used for wall processing.
const (
	modeNone     = iota
	modeTextures // Nuovo stato per la tabella di lookup
	modeVertices
	modeWalls
)

// Wall represents a segment of a level boundary, containing various texture, lighting, and adjacency properties.
type Wall struct {
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
		Flags:       0,
		Light:       0,
	}
}

// Level represents a game level including metadata, textures, and a collection of sectors.
type Level struct {
	Version   string
	LevelName string
	Palette   string
	Textures  []string
	Sectors   []*Sector
}

// NewLevel creates and initializes a new Level instance with empty textures and sectors.
func NewLevel() *Level {
	return &Level{
		Textures: make([]string, 0),
		Sectors:  make([]*Sector, 0),
	}
}

// GetTexture retrieves the texture string by its ID from the level's texture list.
// If the ID is out of bounds, it logs an error and returns an empty string.
func (p *Level) GetTexture(id int) string {
	if id >= 0 && id < len(p.Textures) {
		return p.Textures[id]
	}
	fmt.Printf("unknown texture id %d\n", id)
	return ""
}

// Parse reads and interprets level data from an io.Reader, constructing sectors, textures, and other level properties.
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
		keyword := strings.TrimSpace(strings.ToUpper(tokens[0]))

		switch keyword {
		case "LEV":
			if len(tokens) >= 2 {
				p.Version = tokens[1]
			}
		case "LEVELNAME":
			if len(tokens) >= 2 {
				p.LevelName = tokens[1]
			}
		case "PALETTE":
			if len(tokens) >= 2 {
				p.Palette = tokens[1]
			}
		case "NUMSECTORS":
			sCount, err := GetTokenIntAt(tokens, 1)
			if err != nil {
				fmt.Printf("doSector: invalid token id: %s\n", err.Error())
				return err
			}
			p.Sectors = make([]*Sector, sCount)
		case "NAME":
			//TODO IMPLEMENT
		case "SECOND":
			//TODO IMPLEMENT
		case "LAYER":
			//TODO IMPLEMENT
		case "MUSIC":
			//TODO
		case "PARALLAX":
			//TODO
		case "SECTOR":
			var err error
			sector, err = p.doSector(tokens, sector)
			if err != nil {
				return err
			}
			mode = modeNone

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
			p.doVertices(tokens, sector)
		case "WALLS":
			mode = modeWalls
			p.doWalls(tokens, sector)
		case "TEXTURES":
			mode = modeTextures
			if err := p.doTextures(tokens); err != nil {
				return err
			}
		case "WALL":
			p.doWall(tokens, sector)
		case "TEXTURE:":
			p.doTexture(tokens)
		case "FLAGS":
			p.doFlags(tokens, sector)
		default:
			if mode == modeVertices {
				p.doApplyVertices(tokens, sector)
			} else {
				fmt.Println("unknown keyword:", keyword)
			}
		}
	}

	if sector != nil {
		if sector.Id < 0 || sector.Id >= len(p.Sectors) {
			return fmt.Errorf("invalid sector id: %d", sector.Id)
		}
		p.Sectors[sector.Id] = sector
	}

	return scanner.Err()
}

// doSector processes a sector definition, updating the current sector or creating a new one based on the input tokens.
// Returns an updated or newly created Sector instance, or an error if the input tokens are invalid.
func (p *Level) doSector(tokens []string, currSector *Sector) (*Sector, error) {
	if currSector != nil {
		if currSector.Id < 0 || currSector.Id >= len(p.Sectors) {
			return nil, fmt.Errorf("invalid sector id: %d", currSector.Id)
		}
		p.Sectors[currSector.Id] = currSector
	}
	id, err := GetTokenIntAt(tokens, 1)
	if err != nil {
		return nil, err
	}
	return NewSector(id), nil
}

// doVertices initializes the vertices of a sector by parsing the vertex count and reallocating the vertices slice.
func (p *Level) doVertices(tokens []string, sector *Sector) {
	if sector == nil {
		fmt.Println("doVertices: nil sector")
		return
	}
	sector.Vertices = nil
	vCount, err := GetTokenIntAt(tokens, 1)
	if err != nil {
		fmt.Printf("doVertices: invalid token at 1 err: %s\n", err.Error())
		return
	}
	sector.Vertices = make([]geometry.XY, 0, vCount)
}

// doWalls initializes the walls of a sector by parsing wall count from the given tokens and reallocating the walls slice.
func (p *Level) doWalls(tokens []string, sector *Sector) {
	if sector == nil {
		fmt.Printf("doWalls: nil sector\n")
		return
	}
	sector.Walls = nil
	wCount, err := GetTokenIntAt(tokens, 1)
	if err != nil {
		fmt.Printf("doWalls: invalid token at 1 err: %s\n", err.Error())
		return
	}
	sector.Walls = make([]*Wall, 0, wCount)
}

// doTextures initializes the Textures slice in a Level object using the count specified in tokens. Returns an error if parsing fails.
func (p *Level) doTextures(tokens []string) error {
	p.Textures = nil
	tCount, err := GetTokenIntAt(tokens, 1)
	if err != nil {
		return err
	}
	p.Textures = make([]string, tCount)
	return nil
}

// doTexture updates the texture in the Textures slice with the given id and value extracted from the tokens list.
func (p *Level) doTexture(tokens []string) {
	val, err := GetTokenStringAt(tokens, 1)
	if err != nil {
		fmt.Printf("doTexture: invalid token at 1 err: %s\n", err.Error())
		return
	}
	id, err := GetTokenIntAt(tokens, 3)
	if err != nil {
		fmt.Printf("doTexture: invalid token at 3 err: %s\n", err.Error())
		return
	}
	if id < 0 || id >= len(p.Textures) {
		fmt.Printf("doTexture: invalid texture id %d\n", id)
	}
	p.Textures[id] = val
}

// doFlags parses and assigns flag values from the given tokens to the specified sector's Flags array.
func (p *Level) doFlags(tokens []string, sector *Sector) {
	if sector == nil {
		fmt.Println("doFlags: nil sector")
		return
	}
	var err error
	sector.Flags[0], err = GetTokenIntAt(tokens, 1)
	if err != nil {
		fmt.Printf("doFlags: invalid token id at 1: %s\n", err.Error())
		return
	}
	sector.Flags[1], err = GetTokenIntAt(tokens, 2)
	if err != nil {
		fmt.Printf("doFlags: invalid token id at 2: %s\n", err.Error())
		return
	}
	sector.Flags[2], err = GetTokenIntAt(tokens, 3)
	if err != nil {
		fmt.Printf("doFlags: invalid token id at 3: %s\n", err.Error())
		return
	}
}

// doWall parses wall properties from the provided tokens and adds a new wall to the specified sector.
func (p *Level) doWall(tokens []string, sector *Sector) {
	if sector == nil {
		fmt.Println("doWall: nil sector")
		return
	}
	wall := NewWall()
	for i := 0; i < len(tokens); i++ {
		var err error
		key := strings.ToUpper(strings.TrimSpace(tokens[i]))
		if !strings.Contains(key, ":") {
			continue
		}
		switch key {
		case "WALK:":
		case "MIRROR:":
		case "LEFT:":
			i++
			wall.LeftVertex, err = GetTokenIntAt(tokens, i)
			if err != nil {
				fmt.Printf("doWall: LEFT invalid token id at %d: %s\n", i, err.Error())
			}
		case "RIGHT:":
			i++
			wall.RightVertex, err = GetTokenIntAt(tokens, i)
			if err != nil {
				fmt.Printf("doWall: RIGHT invalid token id at %d: %s\n", i, err.Error())
			}
		case "ADJOIN:":
			i++
			wall.Adjoin, err = GetTokenIntAt(tokens, i)
			if err != nil {
				fmt.Printf("doWall: ADJOIN invalid token id at %d: %s\n", i, err.Error())
			}
		case "MID:":
			i++
			wall.MidTexture, err = GetTokenIntAt(tokens, i)
			if err != nil {
				wall.MidTexture = -1
			}
		case "TOP:":
			i++
			wall.TopTexture, err = GetTokenIntAt(tokens, i)
			if err != nil {
				wall.TopTexture = -1
			}
		case "BOT:":
			i++
			wall.BotTexture, err = GetTokenIntAt(tokens, i)
			if err != nil {
				wall.BotTexture = -1
			}
		case "SIGN:":
			i++
			wall.SignTexture, err = GetTokenIntAt(tokens, i)
			if err != nil {
				wall.SignTexture = -1
			}
		case "FLAGS:":
			i++
			wall.Flags, err = GetTokenIntAt(tokens, i)
			if err != nil {
				fmt.Printf("doWall: FLAGS invalid token id at %d: %s\n", i, err.Error())
			}
		case "LIGHT:":
			i++
			wall.Light, err = GetTokenIntAt(tokens, i)
			if err != nil {
				fmt.Printf("doWall: LIGHT invalid token id at %d: %s\n", i, err.Error())
			}
		default:
			fmt.Println("doWall: Unknown wall attribute: ", key)
		}
	}
	sector.Walls = append(sector.Walls, wall)
}

// doApplyVertices parses vertex information from tokens and appends the resulting XY coordinates to the sector's vertices.
func (p *Level) doApplyVertices(tokens []string, sector *Sector) {
	if sector == nil {
		fmt.Println("doApplyVertices: nil sector")
		return
	}
	var err error
	var ptX, ptZ float64
	found := false
	for i := 0; i < len(tokens); i++ {
		next := i + 1
		if next >= len(tokens) {
			break
		}
		key := strings.ToUpper(strings.TrimSpace(tokens[i]))
		switch key {
		case "X:":
			ptX, err = GetTokenFloatAt(tokens, next)
			if err != nil {
				fmt.Printf("doApplyVertices: invalid token id: %s\n", err.Error())
				return
			}
			found = true
		case "Z:":
			ptZ, err = GetTokenFloatAt(tokens, next)
			if err != nil {
				fmt.Printf("doApplyVertices: invalid token id: %s\n", err.Error())
				return
			}
			found = true
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
	sector.Vertices = append(sector.Vertices, geometry.XY{X: ptX, Y: ptZ})
}
