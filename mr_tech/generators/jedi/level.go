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

// Level represents a game level including metadata, textures, and a collection of sectors.
type Level struct {
	Version   string
	LevelName string
	Palette   string
	Textures  []string
	Sectors   []*Sector
	Palettes  []string
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
		case "LVT":
			if len(tokens) >= 2 {
				p.Version = tokens[1] // "1.1"
			}
		case "LEV":
			if len(tokens) >= 2 {
				p.Version = tokens[1]
			}
		case "VERSION":
			//TODO
		case "LEVELNAME":
			if len(tokens) >= 2 {
				p.LevelName = tokens[1]
			}
		case "PALETTES":
			if len(tokens) >= 2 {
				count, _ := strconv.Atoi(tokens[1])
				p.Palettes = make([]string, 0, count)
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
		case "CMAPS":
			// Salta o inizializza
		case "CMAP:":
			// Salta o salva
		case "SHADES":
			// Il numero di shade tables
		case "SHADE:":
		case "PALETTE:":
			if len(tokens) >= 2 {
				p.Palettes = append(p.Palettes, tokens[1])
			}
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
				subKey := strings.ToUpper(tokens[1])
				switch subKey {
				case "Y":
					// Formato compresso Outlaws: FLOOR Y [Alt] [Tex] [ScaleX] [ScaleY] [Light]
					if len(tokens) >= 4 {
						sector.FloorY, _ = strconv.ParseFloat(tokens[2], 64)
						sector.FloorY = -sector.FloorY
						if tokens[3] != "-" {
							sector.FloorTexture, _ = strconv.Atoi(tokens[3])
						}
					}
				case "ALTITUDE":
					sector.FloorY, _ = strconv.ParseFloat(tokens[2], 64)
				case "TEXTURE", "TEX", "TEXTURE:":
					if tokens[2] != "-" {
						sector.FloorTexture, _ = strconv.Atoi(tokens[2])
					}
				case "OFFSETS": //TODO
				case "SOUND": //TODO
				default:
					fmt.Printf("Unknown FLOOR sub-property: '%s' in line: %v\n", subKey, tokens)
				}
			}
		case "CEILING":
			if sector != nil && len(tokens) >= 3 {
				subKey := strings.ToUpper(tokens[1])
				switch subKey {
				case "Y":
					// Formato compresso Outlaws: CEILING Y [Alt] [Tex] [ScaleX] [ScaleY] [Light]
					if len(tokens) >= 4 {
						sector.CeilingY, _ = strconv.ParseFloat(tokens[2], 64)
						sector.CeilingY = -sector.CeilingY
						if tokens[3] != "-" {
							sector.CeilingTexture, _ = strconv.Atoi(tokens[3])
						}
					}
				case "ALTITUDE":
					sector.CeilingY, _ = strconv.ParseFloat(tokens[2], 64)
				case "TEXTURE", "TEX", "TEXTURE:":
					if tokens[2] != "-" {
						sector.CeilingTexture, _ = strconv.Atoi(tokens[2])
					}
				default:
					fmt.Printf("Unknown CEILING sub-property: '%s' in line: %v\n", subKey, tokens)
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
		case "WALL", "WALL:":
			if sector != nil {
				sector.AddWall(tokens)
			}
		case "TEXTURE:":
			p.doTexture(tokens)
		case "FLAGS":
			if sector != nil {
				sector.AddFlags(tokens)
			}
		case "FRICTION":
			if sector != nil && len(tokens) >= 2 {
				sector.Friction, _ = strconv.ParseFloat(tokens[1], 64)
			}
		case "GRAVITY":
			if sector != nil && len(tokens) >= 2 {
				sector.Gravity, _ = strconv.ParseFloat(tokens[1], 64)
			}
		case "ELASTICITY":
			if sector != nil && len(tokens) >= 2 {
				sector.Elasticity, _ = strconv.ParseFloat(tokens[1], 64)
			}
		case "VELOCITY":
			if sector != nil && len(tokens) >= 4 {
				sector.Velocity[0], _ = strconv.ParseFloat(tokens[1], 64) // X
				sector.Velocity[1], _ = strconv.ParseFloat(tokens[2], 64) // Y
				sector.Velocity[2], _ = strconv.ParseFloat(tokens[3], 64) // Z
			}
		case "VADJOIN":
			if sector != nil && len(tokens) >= 2 {
				sector.VAdjoin, _ = strconv.Atoi(tokens[1])
			}
		case "CMAP":
			if sector != nil && len(tokens) >= 2 {
				sector.CMap, _ = strconv.Atoi(tokens[1])
			}
		case "F_OVERLAY", "C_OVERLAY":
			//TODO: Implementare se decidi di supportare i decal multi-texture
		case "LIGHT":
			// Assorbe "LIGHT SOURCE" globale o "LIGHT" a livello di settore
			if len(tokens) >= 2 && strings.ToUpper(tokens[1]) == "SOURCE" {
				//TODO: Salvare le coordinate della luce globale se necessario
			}
		case "SLOPEDFLOOR":
			if sector != nil && len(tokens) >= 4 {
				sector.SlopedFloor[0], _ = strconv.ParseFloat(tokens[1], 64)
				sector.SlopedFloor[1], _ = strconv.ParseFloat(tokens[2], 64)
				sector.SlopedFloor[2], _ = strconv.ParseFloat(tokens[3], 64)
			}
		case "SLOPEDCEILING":
			if sector != nil && len(tokens) >= 4 {
				sector.SlopedCeiling[0], _ = strconv.ParseFloat(tokens[1], 64)
				sector.SlopedCeiling[1], _ = strconv.ParseFloat(tokens[2], 64)
				sector.SlopedCeiling[2], _ = strconv.ParseFloat(tokens[3], 64)
			}
		case "OFFSET:":
			// Assorbe l'offset se dichiarato come parametro globale del settore (es. allineamento planare)
			if sector != nil && len(tokens) >= 3 {
				// Opzionale: mappare su sector.FloorOffsetX, sector.FloorOffsetY
				_, _ = strconv.ParseFloat(tokens[1], 64)
				_, _ = strconv.ParseFloat(tokens[2], 64)
			}
		default:
			if mode == modeVertices {
				if sector != nil {
					sector.AddVertices(tokens)
				}
			} else {
				fmt.Println("unknown LEVEL keyword:", keyword)
			}
		}
	}
	return scanner.Err()
}

// doSector processes a sector definition, updating the current sector or creating a new one based on the input tokens.
// Returns an updated or newly created Sector instance, or an error if the input tokens are invalid.
func (p *Level) doSector(tokens []string, currSector *Sector) (*Sector, error) {
	id, err := GetTokenStringAt(tokens, 1)
	if err != nil {
		return nil, err
	}
	targetIdx := id
	if ord, _ := GetTokenStringAt(tokens, 3); ord == "ORD:" {
		targetIdx, _ = GetTokenStringAt(tokens, 4)
	}
	idx, err := strconv.Atoi(targetIdx)
	if err != nil {
		return nil, err
	}
	sector := NewSector(id, idx)
	if sector.Index < 0 || sector.Index >= len(p.Sectors) {
		return nil, fmt.Errorf("invalid sector id: %d", currSector.Index)
	}
	p.Sectors[sector.Index] = sector
	return sector, nil
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
	sector.Vertices = make([]geometry.XY, vCount)
}

// doWalls initializes the walls of a sector by parsing wall count from the given tokens and reallocating the walls slice.
func (p *Level) doWalls(tokens []string, sector *Sector) {
	if sector == nil {
		fmt.Printf("doWalls: nil sector\n")
		return
	}
	var err error
	sector.Walls = nil
	sector.WallIdx = 0
	wCount, err := GetTokenIntAt(tokens, 1)
	if err != nil {
		fmt.Printf("doWalls: invalid token at 1 err: %s\n", err.Error())
		return
	}
	sector.Walls = make([]*Wall, wCount)
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
