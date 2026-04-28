package jedi

import (
	"bufio"
	"io"
	"strconv"
	"strings"
)

// Object represents a 3D object in a level, with positional and rotational data, class type, and difficulty flag.
type Object struct {
	Class            string
	Data             string
	X, Y, Z          float64
	Yaw, Pitch, Roll float64
	Diff             int // Difficulty flag
}

// Entities represents a collection of game level data, including the level name and associated objects.
type Entities struct {
	LevelName string
	Objects   []Object
}

// NewEntities creates and returns a new instance of the Entities struct.
func NewEntities() *Entities {
	return &Entities{}
}

func (e *Entities) Parse(r io.Reader) error {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		tokens := strings.Fields(line)
		if len(tokens) == 0 {
			continue
		}
		rootKey := CleanKey(tokens[0])
		switch rootKey {
		case "LEVELNAME":
			if len(tokens) > 1 {
				e.LevelName = tokens[1]
			}
		case "CLASS":
			obj := Object{}
			// Correzione: il ciclo parte da 0 per mappare correttamente le tuple [Chiave, Valore]
			for i := 0; i < len(tokens); i += 2 {
				if i+1 >= len(tokens) {
					break
				}
				key := CleanKey(tokens[i])
				val := tokens[i+1]

				switch key {
				case "CLASS":
					obj.Class = val
				case "DATA":
					obj.Data = val
				case "X":
					obj.X, _ = strconv.ParseFloat(val, 64)
				case "Y":
					obj.Y, _ = strconv.ParseFloat(val, 64)
				case "Z":
					obj.Z, _ = strconv.ParseFloat(val, 64)
				case "YAW":
					obj.Yaw, _ = strconv.ParseFloat(val, 64)
				case "PITCH":
					obj.Pitch, _ = strconv.ParseFloat(val, 64)
				case "ROLL":
					obj.Roll, _ = strconv.ParseFloat(val, 64)
				case "DIFF":
					obj.Diff, _ = strconv.Atoi(val)
				}
			}
			e.Objects = append(e.Objects, obj)
		}
	}
	return scanner.Err()
}
