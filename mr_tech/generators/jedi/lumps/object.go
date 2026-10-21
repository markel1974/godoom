package lumps

import (
	"fmt"
	"strconv"
	"strings"
)

// Object represents a 3D object in a scene with attributes for class, name, position, orientation, and other metadata.
type Object struct {
	Class            string
	Name             string
	Data             string
	X, Y, Z          float64
	Yaw, Pitch, Roll float64
	Difficulty       int
	Sector           string
	Id               string
	Flags            string
}

// NewObject creates and returns a pointer to a new instance of the Object type initialized with default values.
func NewObject() *Object {
	return &Object{}
}

// Parse processes an array of tokenized strings and assigns values to the corresponding fields of the Object instance.
func (obj *Object) Parse(tokens []string) {
	for i := 0; i < len(tokens); i += 2 {
		next := i + 1
		if next >= len(tokens) {
			break
		}
		key := strings.TrimSpace(strings.ToUpper(tokens[i]))
		val := tokens[next]

		switch key {
		case "#":
		case "CLASS:":
			obj.Class = val
		case "NAME:":
			obj.Name = val
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
			obj.Difficulty, _ = strconv.Atoi(val)
		case "SECTOR:":
			obj.Sector = val
		case "ID:":
			obj.Id = val
		case "FLAGS:":
			obj.Flags = val
		default:
			fmt.Println("Unknown CLASS object attribute: ", key)
		}
	}
}
