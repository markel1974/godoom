package q3

import (
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/markel1974/godoom/mr_tech/generators/quake/interfaces"
)

// Shader represents a structure used to store and parse shader properties from external archives.
type Shader struct {
	additive map[string]bool
}

// NewShader creates and returns a new instance of Shader with an initialized additive map.
func NewShader() *Shader {
	return &Shader{
		additive: make(map[string]bool),
	}
}

// Retrieve returns the map of shaders with their associated additive blending flags.
func (s *Shader) Retrieve() map[string]bool {
	return s.additive
}

// Reset clears the current state of the Shader by reinitializing the additive map.
func (s *Shader) Reset() {
	s.additive = make(map[string]bool)
}

// Parse processes shader files in the archive, identifies shaders with additive blending, and updates the Shader state.
func (s *Shader) Parse(arc interfaces.IArchive) error {
	files, fErr := arc.ReadDirFilter("scripts", ".*\\.shader$")
	if fErr != nil {
		return fmt.Errorf("error reading scripts directory: %v", fErr)
	}

	reComment := regexp.MustCompile("(?m)//.*$")
	for _, fName := range files {
		rs, err := arc.Open("scripts/" + fName)
		if err != nil {
			continue
		}
		data, err := io.ReadAll(rs)
		if err != nil {
			continue
		}

		text := reComment.ReplaceAllString(string(data), "")
		text = strings.ReplaceAll(text, "{", " { ")
		text = strings.ReplaceAll(text, "}", " } ")
		text = strings.ReplaceAll(text, "\"", "")

		tokens := strings.Fields(text)
		var currentShader string
		depth := 0

		for i := 0; i < len(tokens); i++ {
			t := tokens[i]
			if t == "{" {
				depth++
				continue
			}
			if t == "}" {
				depth--
				if depth < 0 {
					depth = 0
				}
				if depth == 0 {
					currentShader = ""
				}
				continue
			}
			if depth == 0 {
				currentShader = strings.ToLower(t)
			} else if depth >= 1 {
				tl := strings.ToLower(t)
				if tl == "blendfunc" {
					if i+1 < len(tokens) {
						t2 := strings.ToLower(tokens[i+1])
						if t2 == "add" {
							s.additive[currentShader] = true
						} else if t2 == "gl_one" {
							if i+2 < len(tokens) && strings.ToLower(tokens[i+2]) == "gl_one" {
								s.additive[currentShader] = true
							}
						}
					}
				}
			}
		}
	}
	return nil
}
