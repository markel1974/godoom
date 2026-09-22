package q3

import (
	"fmt"
	"io"
	"strings"

	"github.com/markel1974/godoom/mr_tech/generators/quake/interfaces"
)

// ShaderStage represents a single rendering stage (a block within a shader).
type ShaderStage struct {
	mapData    string
	blendSrc   string
	blendDst   string
	depthWrite bool
	alphaFunc  string
	tcMods     [][]string
}

// IsAdditive returns true if this stage is configured for additive blending.
func (st *ShaderStage) IsAdditive() bool {
	return st.blendSrc == "gl_one" && st.blendDst == "gl_one"
}

// Shader represents a fully parsed Quake 3 shader with global properties and stages.
type Shader struct {
	name         string
	surfaceParms map[string]bool
	cull         string
	stages       []*ShaderStage
}

// NewShader creates and returns a new Shader instance with the provided name and cull mode.
func NewShader(name string, cull string) *Shader {
	return &Shader{
		name:         name,
		cull:         cull,
		surfaceParms: make(map[string]bool),
	}
}

// Shaders manages all parsed shaders from the Quake 3 archive.
type Shaders struct {
	container map[string]*Shader
}

// NewShaders creates and returns a new instance of Shader.
func NewShaders() *Shaders {
	return &Shaders{
		container: make(map[string]*Shader),
	}
}

// Retrieve returns the map of shaders with their associated additive blending flags.
// This is kept for backward compatibility with q3.go logic.
func (s *Shaders) Retrieve() map[string]bool {
	additive := make(map[string]bool)
	for name, sh := range s.container {
		for _, st := range sh.stages {
			if st.IsAdditive() {
				additive[name] = true
				break
			}
		}
	}
	return additive
}

// Reset clears the current state of the Shader by reinitializing the map.
func (s *Shaders) Reset() {
	s.container = make(map[string]*Shader)
}

// Parse processes shader files in the archive and builds a structured AST of Q3Shader objects.
func (s *Shaders) Parse(arc interfaces.IArchive) error {
	files, fErr := arc.ReadDirFilter("scripts", ".*\\.shader$")
	if fErr != nil {
		return fmt.Errorf("error reading scripts directory: %v", fErr)
	}

	for _, fName := range files {
		rs, err := arc.Open("scripts/" + fName)
		if err != nil {
			continue
		}
		data, err := io.ReadAll(rs)
		if err != nil {
			continue
		}
		s.parseShaderFile(string(data))
	}
	return nil
}

// parseShaderFile processes the text of a single .shader file line by line.
func (s *Shaders) parseShaderFile(data string) {
	lines := strings.Split(data, "\n")
	var currentShader *Shader
	var currentStage *ShaderStage

	depth := 0

	for _, rawLine := range lines {
		// Strip comments
		idx := strings.Index(rawLine, "//")
		if idx != -1 {
			rawLine = rawLine[:idx]
		}
		line := strings.TrimSpace(rawLine)
		if line == "" {
			continue
		}

		// Ensure braces are separated from other tokens
		line = strings.ReplaceAll(line, "{", " { ")
		line = strings.ReplaceAll(line, "}", " } ")
		line = strings.ReplaceAll(line, "\"", "")

		tokens := strings.Fields(line)
		if len(tokens) == 0 {
			continue
		}

		for i := 0; i < len(tokens); {
			t := tokens[i]
			tl := strings.ToLower(t)

			if t == "{" {
				depth++
				if depth == 2 && currentShader != nil {
					currentStage = &ShaderStage{depthWrite: true}
					currentShader.stages = append(currentShader.stages, currentStage)
				}
				i++
				continue
			}

			if t == "}" {
				depth--
				if depth < 0 {
					depth = 0
				}
				if depth == 0 {
					if currentShader != nil {
						s.container[currentShader.name] = currentShader
					}
					currentShader = nil
				}
				if depth == 1 {
					currentStage = nil
				}
				i++
				continue
			}

			if depth == 0 {
				// Shader name
				currentShader = NewShader(tl, "front")
				i++
			} else if depth == 1 && currentShader != nil {
				// Global directive
				if tl == "surfaceparm" && i+1 < len(tokens) {
					currentShader.surfaceParms[strings.ToLower(tokens[i+1])] = true
					break
				} else if tl == "cull" && i+1 < len(tokens) {
					currentShader.cull = strings.ToLower(tokens[i+1])
					break
				} else {
					break // ignore unknown global directive
				}
			} else if depth == 2 && currentStage != nil {
				// Stage directive
				if tl == "map" && i+1 < len(tokens) {
					currentStage.mapData = tokens[i+1]
					break
				} else if tl == "blendfunc" && i+1 < len(tokens) {
					arg1 := strings.ToLower(tokens[i+1])
					if arg1 == "add" {
						currentStage.blendSrc = "gl_one"
						currentStage.blendDst = "gl_one"
						currentStage.depthWrite = false
					} else if arg1 == "filter" {
						currentStage.blendSrc = "gl_dst_color"
						currentStage.blendDst = "gl_zero"
						currentStage.depthWrite = false
					} else if arg1 == "blend" {
						currentStage.blendSrc = "gl_src_alpha"
						currentStage.blendDst = "gl_one_minus_src_alpha"
						currentStage.depthWrite = false
					} else if i+2 < len(tokens) {
						currentStage.blendSrc = arg1
						currentStage.blendDst = strings.ToLower(tokens[i+2])
						currentStage.depthWrite = false
					}
					break
				} else if tl == "alphafunc" && i+1 < len(tokens) {
					currentStage.alphaFunc = strings.ToLower(tokens[i+1])
					break
				} else if tl == "depthwrite" {
					currentStage.depthWrite = true
					break
				} else if tl == "tcmod" {
					var tcMod []string
					for j := i + 1; j < len(tokens); j++ {
						tcMod = append(tcMod, strings.ToLower(tokens[j]))
					}
					currentStage.tcMods = append(currentStage.tcMods, tcMod)
					break
				} else {
					break // ignore unknown stage directive
				}
			} else {
				break // fallback
			}
		}
	}
}
