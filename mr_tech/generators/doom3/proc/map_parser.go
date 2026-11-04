package proc

import (
	"io"
	"strings"
)

// MapParser is a parser for Doom 3 Map files, leveraging the Lexer for tokenization and block management.
type MapParser struct {
	lexer *Lexer
}

// NewMapParser initializes a new MapParser instance with the provided data as input for the lexer.
func NewMapParser(data []byte) *MapParser {
	return &MapParser{
		lexer: NewLexer(data),
	}
}

// Parse scans the input and parses all entities, returning a slice of parsed MapEntity objects or an error.
func (p *MapParser) Parse() ([]*MapEntity, error) {
	var entities []*MapEntity

	for {
		tok, err := p.lexer.NextToken()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		if tok == "{" {
			ent, err := p.parseEntity()
			if err != nil {
				return nil, err
			}
			if len(ent.Properties) > 0 {
				entities = append(entities, ent)
			}
		}
	}

	return entities, nil
}

// parseEntity parses a single map entity from the lexer and returns it, including its properties and any errors encountered.
func (p *MapParser) parseEntity() (*MapEntity, error) {
	ent := NewMapEntity()

	for {
		tok, err := p.lexer.NextToken()
		if err != nil {
			return nil, err
		}

		if tok == "}" {
			break
		}

		// Sub-blocks such as brushDef3 { ... } or patchDef2 { ... }
		if tok == "{" {
			if err := p.lexer.SkipBlock(); err != nil {
				return nil, err
			}
			continue
		}

		// Property key
		key := tok
		val, err := p.lexer.NextToken()
		if err != nil {
			return nil, err
		}

		ent.Properties[strings.ToLower(key)] = val
	}

	return ent, nil
}
