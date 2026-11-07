package proc

import (
	"io"
	"strings"
)

func CleanMaterialName(matName string) string {
	return strings.TrimSpace(strings.ToLower(matName))
}

// MaterialDef defines the properties of a material, including its name, texture maps, and material kind.
type MaterialDef struct {
	Name        string
	DiffuseMap  string
	NormalMap   string
	SpecularMap string
	Kind        string
	SurfaceType string
	IsDecal     bool
	GuiSurf     string
}

// NewMaterialDef creates and initializes a new MaterialDef instance based on the provided material name.
func NewMaterialDef(matName string) *MaterialDef {
	nameLower := CleanMaterialName(matName)
	isDecal := strings.HasPrefix(nameLower, "textures/decals/") || strings.HasSuffix(nameLower, "_shadow")
	mat := &MaterialDef{
		Name:    matName,
		Kind:    matName,
		IsDecal: isDecal,
	}
	return mat
}

// Materials represents a collection of material definitions identified by unique string keys.
type Materials struct {
	arc  IPkArchive
	Defs map[string]*MaterialDef
}

// NewMaterials initializes and returns a pointer to a new Materials instance with an empty map for material definitions.
func NewMaterials(arc IPkArchive) *Materials {
	return &Materials{
		arc:  arc,
		Defs: make(map[string]*MaterialDef),
	}
}

// Load processes material files from the provided archive and adds their data to the Materials collection.
func (m *Materials) Load() error {
	mtrFiles, _ := m.arc.ReadDirFilter("materials", "\\.mtr$")
	for _, mtrName := range mtrFiles {
		fullPath := "materials/" + mtrName
		f, err := m.arc.Open(fullPath)
		if err != nil {
			continue
		}
		data, err := io.ReadAll(f)
		if err != nil {
			continue
		}
		m.parseMtrData(data)
	}
	return nil
}

// parseMtrPath parses a texture path from material tokens, unwrapping functions like addnormals or heightmap.
func parseMtrPath(lexer *Lexer) string {
	tok, err := lexer.NextToken()
	if err != nil {
		return ""
	}
	tokLower := strings.ToLower(StripQuotes(tok))

	if idx := strings.Index(tokLower, "("); idx != -1 {
		tokLower = tokLower[idx+1:]
		if endIdx := strings.IndexAny(tokLower, ",)"); endIdx != -1 {
			tokLower = tokLower[:endIdx]
		}
		clean := strings.Trim(tokLower, ",()")
		if strings.Contains(clean, "/") {
			return clean
		}
	}

	if tokLower == "addnormals" || tokLower == "heightmap" || tokLower == "makealpha" || tokLower == "(" {
		for {
			sub, err := lexer.NextToken()
			if err != nil || sub == "}" || sub == "{" {
				return ""
			}
			clean := strings.Trim(strings.ToLower(StripQuotes(sub)), ",()")
			if strings.Contains(clean, "/") {
				return clean
			}
		}
	}

	clean := strings.Trim(tokLower, ",()")
	if clean == "_white" || clean == "_black" || clean == "_default" || clean == "_currentrender" || strings.HasPrefix(clean, "$") {
		return ""
	}
	return clean
}

// parseMtrData parses material definitions from a byte slice and populates the Materials' map with parsed data.
func (m *Materials) parseMtrData(data []byte) {
	lexer := NewLexer(data)
	for {
		name, err := lexer.NextToken()
		if err != nil {
			break
		}
		name = strings.ToLower(StripQuotes(name))

		// Expect opening brace
		peek, err := lexer.PeekToken()
		if err != nil || peek != "{" {
			continue
		}
		_ = lexer.Expect("{")

		def := NewMaterialDef(name)
		editorImage := ""
		currentBlend := ""
		depth := 1
		for depth > 0 {
			tok, err := lexer.NextToken()
			if err != nil {
				break
			}
			if tok == "{" {
				depth++
				currentBlend = ""
				continue
			}
			if tok == "}" {
				depth--
				currentBlend = ""
				continue
			}
			tokLower := strings.ToLower(tok)
			switch tokLower {
			case "guisurf":
				guiTok, err := lexer.NextToken()
				if err == nil {
					def.GuiSurf = strings.Trim(strings.ToLower(StripQuotes(guiTok)), ",()")
				}
			case "diffusemap":
				val := parseMtrPath(lexer)
				if val != "" {
					def.DiffuseMap = val
				}
			case "bumpmap":
				val := parseMtrPath(lexer)
				if val != "" {
					def.NormalMap = val
				}
			case "specularmap":
				val := parseMtrPath(lexer)
				if val != "" {
					def.SpecularMap = val
				}
			case "qer_editorimage":
				val := parseMtrPath(lexer)
				if val != "" && editorImage == "" {
					editorImage = val
				}
			case "blend":
				blendTok, err := lexer.NextToken()
				if err == nil {
					currentBlend = strings.Trim(strings.ToLower(StripQuotes(blendTok)), ",()")
					if strings.Contains(currentBlend, "/") && def.DiffuseMap == "" {
						def.DiffuseMap = currentBlend
					}
				}
			case "map":
				val := parseMtrPath(lexer)
				if val != "" {
					if strings.Contains(currentBlend, "bumpmap") {
						if def.NormalMap == "" {
							def.NormalMap = val
						}
					} else if strings.Contains(currentBlend, "specularmap") {
						if def.SpecularMap == "" {
							def.SpecularMap = val
						}
					} else if strings.Contains(currentBlend, "diffusemap") {
						if def.DiffuseMap == "" {
							def.DiffuseMap = val
						}
					} else if def.DiffuseMap == "" {
						def.DiffuseMap = val
					}
				}
			case "if":
				peekIf, err := lexer.PeekToken()
				if err == nil && peekIf == "(" {
					_ = lexer.Expect("(")
					pDepth := 1
					for pDepth > 0 {
						pTok, pErr := lexer.NextToken()
						if pErr != nil || pTok == "}" {
							break
						}
						if pTok == "(" {
							pDepth++
						} else if pTok == ")" {
							pDepth--
						}
					}
				}
			case "translucent":
				def.Kind = "translucent" // TODO config.MaterialKindTranslucent
			case "metal", "stone", "flesh", "wood", "cardboard", "liquid", "glass", "plastic", "ricochet":
				def.SurfaceType = tokLower
			case "water", "slime", "lava", "blood", "sky":
				def.Kind = tokLower
				if def.SurfaceType == "" {
					def.SurfaceType = tokLower
				}
			case "decal_macro", "polygonoffset":
				def.IsDecal = true
			case "(", ")":
				// NOTHING TO DO
			}
		}

		if def.DiffuseMap == "" && editorImage != "" {
			editorLower := strings.ToLower(editorImage)
			if def.GuiSurf == "" && !strings.Contains(editorLower, "entitygui") {
				def.DiffuseMap = editorImage
			}
		}

		m.Defs[name] = def
	}
}

// Resolve returns a MaterialDef for the given material name, creating a new definition if it doesn't exist in the collection.
func (m *Materials) Resolve(materialName string) *MaterialDef {
	nameLower := CleanMaterialName(materialName)
	if def, ok := m.Defs[nameLower]; ok {
		return def
	}
	def := NewMaterialDef(nameLower)
	// Implicit Doom 3 convention fallback
	def.DiffuseMap = nameLower
	def.NormalMap = nameLower + "_local"
	def.SpecularMap = nameLower + "_s"
	m.Defs[nameLower] = def
	return def
}
