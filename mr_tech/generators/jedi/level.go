package jedi

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"

	"github.com/markel1974/godoom/mr_tech/geometry"
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
			sector, err = p.doCreateSector(tokens)
			if err != nil {
				return err
			}
			mode = modeNone

		case "AMBIENT":
			if sector != nil && len(tokens) >= 2 {
				sector.LightLevel, _ = strconv.ParseFloat(tokens[1], 64)
			}
		case "FLOOR":
			if sector != nil {
				sector.SetFloor(tokens)
			}
		case "CEILING":
			if sector != nil {
				sector.SetCeiling(tokens)
			}
		case "VERTICES":
			mode = modeVertices
			if sector != nil {
				sector.VerticesInitialize(tokens)
			}
		case "WALLS":
			mode = modeWalls
			if sector != nil {
				sector.WallsInitialize(tokens)
			}
		case "TEXTURES":
			mode = modeTextures
			if err := p.doTextures(tokens); err != nil {
				return err
			}
		case "WALL", "WALL:":
			if sector != nil {
				sector.WallAdd(tokens)
			}
		case "TEXTURE:":
			p.doTexture(tokens)
		case "FLAGS":
			if sector != nil {
				sector.FlagsAdd(tokens)
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
			if sector != nil {
				sector.SetSlopedFloor(tokens)
			}
		case "SLOPEDCEILING":
			if sector != nil {
				sector.SetSlopedCeiling(tokens)
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
					sector.VertexAdd(tokens)
				}
			} else {
				fmt.Println("unknown LEVEL keyword:", keyword)
			}
		}
	}
	return scanner.Err()
}

// doCreateSector processes a sector definition, updating the current sector or creating a new one based on the input tokens.
// Returns an updated or newly created Sector instance, or an error if the input tokens are invalid.
func (p *Level) doCreateSector(tokens []string) (*Sector, error) {
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
		return nil, fmt.Errorf("invalid sector id: %d", sector.Index)
	}
	p.Sectors[sector.Index] = sector
	return sector, nil
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

// ComputeSlopePlane calculates the slope plane of a given wall within a sector, returning its normal as an XYZ vector.
func (p *Level) ComputeSlopePlane(slope *Slope, baseZ float64) (geometry.XYZ, error) {
	if slope.SectorIndex < 0 || slope.SectorIndex >= len(p.Sectors) {
		return geometry.XYZ{X: 0, Y: 0, Z: 0}, fmt.Errorf("invalid SectorId: %d", slope.SectorIndex)
	}
	pivotSector := p.Sectors[slope.SectorIndex]

	if slope.WallIndex < 0 || slope.WallIndex >= len(pivotSector.Walls) {
		return geometry.XYZ{X: 0, Y: 0, Z: 0}, fmt.Errorf("invalid WallId: %d", slope.WallIndex)
	}
	wall := pivotSector.Walls[slope.WallIndex]

	if wall.LeftVertex < 0 || wall.LeftVertex >= len(pivotSector.Vertices) {
		return geometry.XYZ{X: 0, Y: 0, Z: 0}, fmt.Errorf("invalid LeftVertex index")
	}
	if wall.RightVertex < 0 || wall.RightVertex >= len(pivotSector.Vertices) {
		return geometry.XYZ{X: 0, Y: 0, Z: 0}, fmt.Errorf("invalid RightVertex index")
	}
	v1 := pivotSector.Vertices[wall.LeftVertex]
	v2 := pivotSector.Vertices[wall.RightVertex]

	dx := v2.X - v1.X
	dy := v2.Y - v1.Y

	length := math.Sqrt(dx*dx + dy*dy)
	if length == 0 {
		return geometry.XYZ{X: 0, Y: 0, Z: 0}, fmt.Errorf("pivot wall has zero length")
	}
	var nX, nY float64
	if pivotSector.IsCCW() {
		nX, nY = -dy/length, dx/length
	} else {
		nX, nY = dy/length, -dx/length
	}
	degrees := float64(slope.Pitch) / 100.0
	gradient := math.Tan(degrees * math.Pi / 180.0)
	slopeX := nX * gradient
	slopeY := nY * gradient
	slopeZ := baseZ - (slopeX * v1.X) - (slopeY * v1.Y)
	return geometry.XYZ{X: slopeX, Y: slopeY, Z: slopeZ}, nil
}
