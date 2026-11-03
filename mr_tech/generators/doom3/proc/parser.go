package proc

import (
	"fmt"
	"io"

	"github.com/markel1974/godoom/mr_tech/geometry"
)

// Parser parses Doom 3 precalculated map geometry (.proc files).
type Parser struct {
	lexer *Lexer
}

// NewParser initializes a new Parser for the given file data.
func NewParser(data []byte) *Parser {
	return &Parser{
		lexer: NewLexer(data),
	}
}

// Parse processes the entire .proc file and returns a populated File structure.
func (p *Parser) Parse() (*File, error) {
	proc := NewProcFile()

	for {
		tok, err := p.lexer.NextToken()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		switch tok {
		case "mapProcVersion":
			v, err := p.lexer.ReadInt()
			if err != nil {
				return nil, fmt.Errorf("failed to read mapProcVersion: %w", err)
			}
			proc.Version = v

		case "model":
			model, err := p.parseModel()
			if err != nil {
				return nil, fmt.Errorf("failed to parse model: %w", err)
			}
			proc.Models = append(proc.Models, model)

		case "interAreaPortals":
			portals, err := p.parseInterAreaPortals()
			if err != nil {
				return nil, fmt.Errorf("failed to parse interAreaPortals: %w", err)
			}
			proc.Portals = portals

		default:
			// Skip blocks like nodes { ... }, shadowModel { ... }, etc.
			peek, err := p.lexer.PeekToken()
			if err == nil && peek == "{" {
				if err := p.lexer.SkipBlock(); err != nil {
					return nil, fmt.Errorf("failed to skip block '%s': %w", tok, err)
				}
			}
		}
	}

	return proc, nil
}

func (p *Parser) parseModel() (*Model, error) {
	if err := p.lexer.Expect("{"); err != nil {
		return nil, err
	}

	name, err := p.lexer.NextToken()
	if err != nil {
		return nil, err
	}
	name = StripQuotes(name)

	numSurfaces, err := p.lexer.ReadInt()
	if err != nil {
		return nil, fmt.Errorf("failed to read numSurfaces for model %s: %w", name, err)
	}

	model := &Model{
		Name:     name,
		Surfaces: make([]*Surface, 0, numSurfaces),
	}

	for sIdx := 0; sIdx < numSurfaces; sIdx++ {
		surface, err := p.parseSurface()
		if err != nil {
			return nil, fmt.Errorf("failed to parse surface %d of model %s: %w", sIdx, name, err)
		}
		model.Surfaces = append(model.Surfaces, surface)
	}

	if err := p.lexer.Expect("}"); err != nil {
		return nil, err
	}

	return model, nil
}

func (p *Parser) parseSurface() (*Surface, error) {
	if err := p.lexer.Expect("{"); err != nil {
		return nil, err
	}

	material, err := p.lexer.NextToken()
	if err != nil {
		return nil, err
	}
	material = StripQuotes(material)

	numVerts, err := p.lexer.ReadInt()
	if err != nil {
		return nil, fmt.Errorf("failed to read numVerts: %w", err)
	}

	surface := &Surface{
		Material: material,
		Vertices: make([]Vertex, numVerts),
	}

	for v := 0; v < numVerts; v++ {
		vert, err := p.parseVertex()
		if err != nil {
			return nil, fmt.Errorf("failed to parse vertex %d: %w", v, err)
		}
		surface.Vertices[v] = vert
	}

	numIndexes, err := p.lexer.ReadInt()
	if err != nil {
		return nil, fmt.Errorf("failed to read numIndexes: %w", err)
	}

	surface.Indices = make([]int, numIndexes)
	for i := 0; i < numIndexes; i++ {
		idx, err := p.lexer.ReadInt()
		if err != nil {
			return nil, fmt.Errorf("failed to read index %d: %w", i, err)
		}
		surface.Indices[i] = idx
	}

	if err := p.lexer.Expect("}"); err != nil {
		return nil, err
	}

	return surface, nil
}

// parseVertex parses standard Doom 3 vertex format:
// ( x y z ) s t ( nx ny nz ) [ ( tx ty tz ) ( bx by bz ) ]
func (p *Parser) parseVertex() (Vertex, error) {
	var vert Vertex

	// 1. Position: ( x y z )
	if err := p.lexer.Expect("("); err != nil {
		return vert, err
	}
	x, err := p.lexer.ReadFloat()
	if err != nil {
		return vert, err
	}
	y, err := p.lexer.ReadFloat()
	if err != nil {
		return vert, err
	}
	z, err := p.lexer.ReadFloat()
	if err != nil {
		return vert, err
	}
	if err := p.lexer.Expect(")"); err != nil {
		return vert, err
	}
	vert.Pos = geometry.XYZ{X: x, Y: y, Z: z}

	// 2. Texture coordinates: s t
	s, err := p.lexer.ReadFloat()
	if err != nil {
		return vert, err
	}
	t, err := p.lexer.ReadFloat()
	if err != nil {
		return vert, err
	}
	vert.U = s
	vert.V = t

	// 3. Normal vector: ( nx ny nz )
	peek, err := p.lexer.PeekToken()
	if err == nil && peek == "(" {
		_ = p.lexer.Expect("(")
		nx, _ := p.lexer.ReadFloat()
		ny, _ := p.lexer.ReadFloat()
		nz, _ := p.lexer.ReadFloat()
		_ = p.lexer.Expect(")")
		vert.Normal = geometry.XYZ{X: nx, Y: ny, Z: nz}
	}

	// 4. Optional Tangent / Bitangent vectors: ( tx ty tz ) ( bx by bz )
	for {
		peek, err := p.lexer.PeekToken()
		if err != nil || peek != "(" {
			break
		}
		_ = p.lexer.Expect("(")
		for {
			t, err := p.lexer.NextToken()
			if err != nil || t == ")" {
				break
			}
		}
	}

	return vert, nil
}

func (p *Parser) parseInterAreaPortals() ([]*Portal, error) {
	if err := p.lexer.Expect("{"); err != nil {
		return nil, err
	}

	_, err := p.lexer.ReadInt() // numAreas
	if err != nil {
		return nil, err
	}

	numPortals, err := p.lexer.ReadInt()
	if err != nil {
		return nil, err
	}

	portals := make([]*Portal, 0, numPortals)

	for i := 0; i < numPortals; i++ {
		numPoints, err := p.lexer.ReadInt()
		if err != nil {
			return nil, err
		}
		positiveArea, err := p.lexer.ReadInt()
		if err != nil {
			return nil, err
		}
		negativeArea, err := p.lexer.ReadInt()
		if err != nil {
			return nil, err
		}

		portal := &Portal{
			Points:       make([]geometry.XYZ, numPoints),
			PositiveArea: positiveArea,
			NegativeArea: negativeArea,
		}

		for pt := 0; pt < numPoints; pt++ {
			if err := p.lexer.Expect("("); err != nil {
				return nil, err
			}
			x, _ := p.lexer.ReadFloat()
			y, _ := p.lexer.ReadFloat()
			z, _ := p.lexer.ReadFloat()
			if err := p.lexer.Expect(")"); err != nil {
				return nil, err
			}
			portal.Points[pt] = geometry.XYZ{X: x, Y: y, Z: z}
		}

		portals = append(portals, portal)
	}

	if err := p.lexer.Expect("}"); err != nil {
		return nil, err
	}

	return portals, nil
}
