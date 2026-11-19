package lumps

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
)

const (
	// SpawnFlagNotEasy prevents the entity from spawning on Easy difficulty (Skill 0).
	SpawnFlagNotEasy = 256
	// SpawnFlagNotMedium prevents the entity from spawning on Medium difficulty (Skill 1).
	SpawnFlagNotMedium = 512
	// SpawnFlagNotHard prevents the entity from spawning on Hard/Nightmare difficulty (Skill 2/3).
	SpawnFlagNotHard = 1024
	// SpawnFlagNotDeathmatch prevents the entity from spawning in Deathmatch mode.
	SpawnFlagNotDeathmatch = 2048
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
	return NewEntitiesFromText(text)
}

// NewEntitiesFromText parses a structured text input to extract entities with key-value properties organized in brackets.
// Returns a slice of Entity pointers or an error if parsing fails.
func NewEntitiesFromText(text string) ([]*Entity, error) {
	var entities []*Entity
	var currentEntity *Entity

	inQuotes := false
	var currentToken bytes.Buffer
	var tokens []string

	// Tokenizzazione di base: estrae { } e le stringhe tra virgolette
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

// GetProperty retrieves the value associated with the given key from the entity's properties and returns a boolean indicating success.
func (p *Entity) GetProperty(text string) (string, bool) {
	v, ok := p.Properties[text]
	return v, ok
}

func (p *Entity) GetPropertyCleaned(text string) (string, bool) {
	v, ok := p.GetProperty(text)
	if !ok {
		return "", false
	}
	return CleanString(v), ok
}

// GetSpawnFlags returns the parsed integer bitmask from the entity's "spawnflags" property.
func (p *Entity) GetSpawnFlags() int {
	val, ok := p.GetProperty("spawnflags")
	if !ok || val == "" {
		return 0
	}
	flags, err := strconv.Atoi(val)
	if err != nil {
		return 0
	}
	return flags
}

// IsLevelAvailable reports whether the entity is available to spawn at the given single-player difficulty
// (1-2: Easy, 3: Medium, 4-5: Hard) according to Quake 1 / Quake 2 spawnflags.
func (p *Entity) IsLevelAvailable(level int) bool {
	flags := p.GetSpawnFlags()
	switch level {
	case 1, 2:
		return flags&SpawnFlagNotEasy == 0
	case 3:
		return flags&SpawnFlagNotMedium == 0
	case 4, 5:
		return flags&SpawnFlagNotHard == 0
	}
	return true
}
