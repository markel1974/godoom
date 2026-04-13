package lumps

import (
	"bytes"
	"fmt"
	"io"
)

// Entity represents a collection of properties, where each property is a key-value pair formatted as strings.
type Entity struct {
	Properties map[string]string
}

// NewEntities parses entity data from a lump in a file and returns a list of entities or an error if parsing fails.
func NewEntities(rs io.ReadSeeker, lumpInfo *LumpInfo) ([]*Entity, error) {
	if err := Seek(rs, lumpInfo.Filepos); err != nil {
		return nil, err
	}
	data := make([]byte, lumpInfo.Size)
	if _, err := rs.Read(data); err != nil {
		return nil, err
	}
	// Quake text lumps sono spesso null-terminated o con garbage alla fine
	text := FromNullTerminatingString(data)
	return parseEntityText(text)
}

// parseEntityText parses a structured text input to extract entities with key-value properties organized in brackets.
// Returns a slice of Entity pointers or an error if parsing fails.
func parseEntityText(text string) ([]*Entity, error) {
	var entities []*Entity
	var currentEntity *Entity

	inQuotes := false
	var currentToken bytes.Buffer
	var tokens []string

	// 1. Tokenizzazione di base: estrae { } e le stringhe tra virgolette
	for i := 0; i < len(text); i++ {
		c := text[i]
		if c == '"' {
			inQuotes = !inQuotes
			if !inQuotes {
				tokens = append(tokens, currentToken.String())
				currentToken.Reset()
			}
		} else if inQuotes {
			currentToken.WriteByte(c)
		} else if c == '{' {
			tokens = append(tokens, "{")
		} else if c == '}' {
			tokens = append(tokens, "}")
		}
	}

	// 2. Costruzione delle entità dai token
	for i := 0; i < len(tokens); i++ {
		t := tokens[i]
		if t == "{" {
			currentEntity = &Entity{Properties: make(map[string]string)}
			entities = append(entities, currentEntity)
		} else if t == "}" {
			currentEntity = nil
		} else if currentEntity != nil {
			// Assumiamo che i token all'interno delle { } siano sempre a coppie "chiave" "valore"
			if i+1 < len(tokens) && tokens[i+1] != "{" && tokens[i+1] != "}" {
				key := tokens[i]
				value := tokens[i+1]
				currentEntity.Properties[key] = value
				i++ // Salta il token del valore per il prossimo ciclo
			}
		}
	}

	if len(entities) == 0 {
		return nil, fmt.Errorf("no entities found or failed to parse")
	}

	return entities, nil
}
