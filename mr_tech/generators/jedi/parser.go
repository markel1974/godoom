package jedi

import (
	"bufio"
	"io"
	"strconv"
	"strings"

	"github.com/markel1974/godoom/mr_tech/model/geometry"
)

// Parser is responsible for converting raw input data into a structured LevAST representation of a level.
// It reads line-by-line from an input source and constructs the abstract syntax tree (AST).
// Errors encountered during parsing are stored and returned alongside the constructed AST.
type Parser struct {
	scanner *bufio.Scanner
	ast     *LevAST
	err     error
}

// NewParser creates a new Parser instance that initializes its scanner and abstract syntax tree using the provided io.Reader.
func NewParser(r io.Reader) *Parser {
	return &Parser{
		scanner: bufio.NewScanner(r),
		ast:     &LevAST{},
	}
}

// Parse processes the input data from the parser's scanner and constructs a LevAST representing the level's structure.
func (p *Parser) Parse() (*LevAST, error) {
	currentSector := &LevSector{Id: -1}
	inSector := false

	for p.scanner.Scan() {
		line := strings.TrimSpace(p.scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		tokens := strings.Fields(line)
		keyword := strings.ToUpper(tokens[0])

		switch keyword {
		case "SECTOR":
			if currentSector != nil {
				p.ast.Sectors = append(p.ast.Sectors, *currentSector)
			}
			id, _ := strconv.Atoi(tokens[1])
			currentSector = &LevSector{Id: id}
			inSector = true

		case "AMBIENT":
			if inSector && len(tokens) >= 2 {
				currentSector.LightLevel, _ = strconv.ParseFloat(tokens[1], 64)
			}

		case "FLOOR":
			if inSector && len(tokens) >= 3 && strings.ToUpper(tokens[1]) == "ALTITUDE" {
				currentSector.FloorY, _ = strconv.ParseFloat(tokens[2], 64)
			}

		case "CEILING":
			if inSector && len(tokens) >= 3 && strings.ToUpper(tokens[1]) == "ALTITUDE" {
				currentSector.CeilingY, _ = strconv.ParseFloat(tokens[2], 64)
			}

		case "VERTEX":
			// Formato: VERTEX [index] [X] [Z] all'interno della definizione del settore
			if inSector && len(tokens) >= 4 {
				x, _ := strconv.ParseFloat(tokens[2], 64)
				y, _ := strconv.ParseFloat(tokens[3], 64)
				currentSector.Vertices = append(currentSector.Vertices, geometry.XY{X: x, Y: y})
			}

		case "WALL":
			// Formato: WALL [left_vertex] [right_vertex] ... ADJOIN [adj] ... MID [mid] TOP [top] BOT [bot]
			if inSector {
				wall := p.parseWall(tokens)
				currentSector.Walls = append(currentSector.Walls, wall)
			}
		}
	}
	if currentSector != nil {
		p.ast.Sectors = append(p.ast.Sectors, *currentSector)
	}

	if p.err == nil {
		p.err = p.scanner.Err()
	}
	return p.ast, p.err
}

// parseWall parses a wall definition from a slice of tokens and returns a LevWall struct representing the wall.
func (p *Parser) parseWall(tokens []string) LevWall {
	wall := LevWall{Adjoin: -1}

	// Il formato Jedi Engine per le pareti associa i vertici in ordine antiorario.
	// Il token[1] è tipicamente l'indice del vertice sinistro nel pool locale del settore.
	if len(tokens) > 1 {
		wall.VertexIndex, _ = strconv.Atoi(tokens[1])
	}

	for i := 2; i < len(tokens); i++ {
		val := strings.ToUpper(tokens[i])
		switch val {
		case "ADJOIN":
			if i+1 < len(tokens) {
				wall.Adjoin, _ = strconv.Atoi(tokens[i+1])
				i++
			}
		case "MID":
			if i+1 < len(tokens) {
				wall.MidTexture = tokens[i+1]
				i++
			}
		case "TOP":
			if i+1 < len(tokens) {
				wall.TopTexture = tokens[i+1]
				i++
			}
		case "BOT":
			if i+1 < len(tokens) {
				wall.BotTexture = tokens[i+1]
				i++
			}
		}
	}
	return wall
}
