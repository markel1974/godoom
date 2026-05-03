package jedi

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Object represents a 3D entity with positional, rotational, and class-related attributes.
type Object struct {
	Class            string
	Data             string
	X, Y, Z          float64
	Yaw, Pitch, Roll float64
	Diff             int // Difficulty flag
}

// NewObject initializes and returns a pointer to a new Object instance with default values.
func NewObject() *Object {
	return &Object{}
}

// Entities represents a collection of objects and a level name within a structured data model.
type Entities struct {
	LevelName string
	Waxes     []string
	Fmes      []string
	Threedos  []string
	Sounds    []string
	Objects   []*Object
}

// NewEntities initializes and returns a new instance of Entities with default values.
func NewEntities() *Entities {
	return &Entities{}
}

// Parse reads and parses input data from the provided io.Reader to populate the Entities struct with level and object data.
func (e *Entities) Parse(r io.Reader) error {
	insideComment := false
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		baseLine := scanner.Text()
		if !insideComment {
			if p0 := strings.Index(baseLine, "/*"); p0 >= 0 {
				if p1 := strings.Index(baseLine, "*/"); p1 >= 0 {
					baseLine = baseLine[:p0] + baseLine[p1+2:]
				} else {
					insideComment = true
					baseLine = baseLine[:p0]
				}
			}
		} else {
			if p := strings.Index(baseLine, "*/"); p >= 0 {
				insideComment = false
				baseLine = baseLine[p+2:]
			} else {
				continue
			}
		}
		line := strings.TrimSpace(baseLine)
		tokens := strings.Fields(line)
		if len(tokens) == 0 {
			continue
		}
		rootKey := strings.TrimSpace(strings.ToUpper(tokens[0]))
		if len(rootKey) == 0 {
			continue
		}
		switch rootKey {
		case "O":
		case "PODS":
			count, _ := GetTokenIntAt(tokens, 1)
			e.Threedos = make([]string, 0, count)
		case "FMES":
			count, _ := GetTokenIntAt(tokens, 1)
			e.Fmes = make([]string, 0, count)
		case "SPRS":
			count, _ := GetTokenIntAt(tokens, 1)
			e.Waxes = make([]string, 0, count)
		case "OBJECTS": // TODO IMPLEMENT
		case "SOUNDS": // TODO IMPLEMENT
		case "LEVELNAME": // TODO IMPLEMENT
			levelName, _ := GetTokenStringAt(tokens, 1)
			e.LevelName = levelName
		case "FLAGS:": // TODO IMPLEMENT
		case "D_YAW:": // IMPLEMENT
		case "EYE:": // TODO IMPLEMENT
		case "LOGIC:": // TODO IMPLEMENT
		case "SEQ": // TODO IMPLEMENT
		case "SEQEND": // TODO IMPLEMENT
		case "POD:":
			e.Threedos = append(e.Threedos, tokens[1])
		case "SPR:":
			e.Waxes = append(e.Waxes, tokens[1])
		case "FME:": // TODO IMPLEMENT
			e.Fmes = append(e.Fmes, tokens[1])
		case "SOUND:": // TODO IMPLEMENT
		case "TYPE:": // TODO IMPLEMENT
		case "HEIGHT:": // TODO IMPLEMENT
		case "RADIUS:": // TODO IMPLEMENT
		case "DELAY:": // TODO IMPLEMENT
		case "INTERVAL:": // TODO IMPLEMENT
		case "MAX_ALIVE:": // TODO IMPLEMENT
		case "MIN_DIST:": // TODO IMPLEMENT
		case "MAX_DIST:": // TODO IMPLEMENT
		case "NUM_TERMINATE:": // TODO IMPLEMENT
		case "PAUSE:": // TODO IMPLEMENT
		case "VUE:": // TODO IMPLEMENT
		case "VUE_APPEND:": // TODO IMPLEMENT
		case "BOSS:": // TODO IMPLEMENT
		case "MASTER:": //TODO

		case "CLASS:":
			obj := NewObject()
			for i := 0; i < len(tokens); i += 2 {
				next := i + 1
				if next >= len(tokens) {
					break
				}
				key := strings.TrimSpace(strings.ToUpper(tokens[i]))
				val := tokens[next]

				switch key {
				case "CLASS:":
					obj.Class = val
				case "DATA:":
					obj.Data = val
				case "X:":
					obj.X, _ = strconv.ParseFloat(val, 64)
				case "Y:":
					obj.Y, _ = strconv.ParseFloat(val, 64)
				case "Z:":
					obj.Z, _ = strconv.ParseFloat(val, 64)
				case "YAW:":
					obj.Yaw, _ = strconv.ParseFloat(val, 64)
				case "PITCH:", "PCH:":
					obj.Pitch, _ = strconv.ParseFloat(val, 64)
				case "ROLL:", "ROL:":
					obj.Roll, _ = strconv.ParseFloat(val, 64)
				case "DIFF:":
					obj.Diff, _ = strconv.Atoi(val)
				default:
					fmt.Println("Unknown CLASS object attribute: ", key)
				}
			}
			e.Objects = append(e.Objects, obj)
		default:
			fmt.Println("Unknown LEVEL object attribute: ", rootKey)
		}

	}
	return scanner.Err()
}
