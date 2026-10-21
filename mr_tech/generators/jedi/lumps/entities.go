package lumps

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// Entities represents a collection of elements within a game level, such as objects, sounds, models, and textures.
type Entities struct {
	LevelName string
	Waxes     []string
	Fmes      []string
	Threedos  []string
	Sounds    []string
	Objects   []*Object
}

// NewEntities creates and returns a new instance of the Entities struct with initialized fields.
func NewEntities() *Entities {
	return &Entities{}
}

// Parse reads and processes the entity data from the provided io.Reader and populates the Entities fields accordingly.
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
		case "#":
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
		case "OBJECTS":
			count, _ := GetTokenIntAt(tokens, 1)
			e.Objects = make([]*Object, 0, count)
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
		case "FME:":
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
		case "OBT": //TODO

		case "CLASS:", "NAME:":
			obj := NewObject()
			obj.Parse(tokens)
			e.Objects = append(e.Objects, obj)
		default:
			fmt.Println("Unknown ENTITIES object attribute:", rootKey)
		}

	}
	return scanner.Err()
}
