package q3

import (
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/markel1974/godoom/mr_tech/generators/quake/interfaces"
)

func ParseShaders(arc interfaces.IArchive) (map[string]bool, error) {
	additiveMats := make(map[string]bool)
	files, fErr := arc.ReadDirFilter("scripts", ".*\\.shader$")
	if fErr != nil {
		return nil, fmt.Errorf("error reading scripts directory: %v", fErr)
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
							additiveMats[currentShader] = true
						} else if t2 == "gl_one" {
							if i+2 < len(tokens) && strings.ToLower(tokens[i+2]) == "gl_one" {
								additiveMats[currentShader] = true
							}
						}
					}
				}
			}
		}
	}
	return additiveMats, nil
}
