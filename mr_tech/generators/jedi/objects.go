package jedi

import (
	"bufio"
	"io"
	"strconv"
	"strings"
)

// ObjAST represents a parsed structure for storing level data and associated objects in the AST (Abstract Syntax Tree).
// LevelName specifies the name of the level associated with the objects.
// Objects holds a list of LevObject entities, defining individual objects and their properties.
type ObjAST struct {
	LevelName string
	Objects   []LevObject
}

// LevObject represents an object in a level with positional, rotational, and classification data.
type LevObject struct {
	Class            string
	Data             string
	X, Y, Z          float64
	Yaw, Pitch, Roll float64
	Diff             int // Difficulty flag
}

// ParseObjects parses object data from the provided io.Reader and returns an ObjAST structure or an error if parsing fails.
func ParseObjects(r io.Reader) (*ObjAST, error) {
	ast := &ObjAST{}
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		tokens := strings.Fields(line)
		switch strings.ToUpper(tokens[0]) {
		case "LEVELNAME":
			if len(tokens) > 1 {
				ast.LevelName = tokens[1]
			}
		case "CLASS":
			// Struttura fissa: CLASS [class] DATA [data] X [x] Y [y] Z [z] PITCH [p] YAW [y] ROLL [r] DIFF [d]
			obj := LevObject{}
			for i := 1; i < len(tokens); i += 2 {
				if i+1 >= len(tokens) {
					break
				}
				key := strings.ToUpper(tokens[i])
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
			ast.Objects = append(ast.Objects, obj)
		}
	}
	return ast, scanner.Err()
}
