package proc

import (
	"io"
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
		tok, tErr := p.lexer.NextToken()
		if tErr != nil {
			if tErr == io.EOF {
				break
			}
			return nil, tErr
		}
		if tok == "{" {
			ent := NewMapEntity()
			if err := ent.Parse(p.lexer); err != nil {
				return nil, err
			}
			if len(ent.Properties) > 0 {
				entities = append(entities, ent)
			}
		}
	}
	return entities, nil
}
