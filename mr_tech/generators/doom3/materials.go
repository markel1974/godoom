package doom3

import (
	"fmt"
	"io"
	"strings"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/generators/doom3/proc"
)

// IPkArchive represents an interface for accessing custom archive formats, providing file opening and directory filtering.
type IPkArchive interface {
	Open(p string) (io.ReadSeeker, error)
	ReadDirFilter(p string, f string) ([]string, error)
}

// ToMaterial determines the material kind based on the provided lowercase texture name string.
func ToMaterial(texNameLC string) config.MaterialKind {
	kind := config.MaterialKindSolid
	if strings.Contains(texNameLC, "sky") {
		kind = config.MaterialKindSky
	} else if strings.Contains(texNameLC, "water") || strings.Contains(texNameLC, "liquid") {
		kind = config.MaterialKindWater
	} else if strings.Contains(texNameLC, "slime") {
		kind = config.MaterialKindSlime
	}
	return kind
}

// MaterialDef defines the properties of a material, including its name, texture maps, and material kind.
type MaterialDef struct {
	Name        string
	DiffuseMap  string
	NormalMap   string
	SpecularMap string
	Kind        config.MaterialKind
}

// NewMaterialDef creates and initializes a new MaterialDef instance based on the provided material name.
func NewMaterialDef(matName string) *MaterialDef {
	mat := &MaterialDef{
		Name: matName,
		Kind: ToMaterial(matName),
	}
	return mat
}

// Materials represents a collection of material definitions identified by unique string keys.
type Materials struct {
	defs map[string]*MaterialDef
}

// NewMaterials initializes and returns a pointer to a new Materials instance with an empty map for material definitions.
func NewMaterials() *Materials {
	return &Materials{
		defs: make(map[string]*MaterialDef),
	}
}

// LoadFromArchive processes material files from the provided archive and adds their data to the Materials collection.
func (m *Materials) LoadFromArchive(arc IPkArchive) error {
	mtrFiles, _ := arc.ReadDirFilter("materials", "\\.mtr$")
	for _, mtrName := range mtrFiles {
		fullPath := "materials/" + mtrName
		f, err := arc.Open(fullPath)
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

// parseMtrData parses material definitions from a byte slice and populates the Materials' map with parsed data.
func (m *Materials) parseMtrData(data []byte) {
	lexer := proc.NewLexer(data)
	for {
		name, err := lexer.NextToken()
		if err != nil {
			break
		}
		name = strings.ToLower(proc.StripQuotes(name))

		// Expect opening brace
		peek, err := lexer.PeekToken()
		if err != nil || peek != "{" {
			continue
		}
		_ = lexer.Expect("{")

		def := NewMaterialDef(name)
		depth := 1
		for depth > 0 {
			tok, err := lexer.NextToken()
			if err != nil {
				break
			}
			if tok == "{" {
				depth++
				continue
			}
			if tok == "}" {
				depth--
				continue
			}
			tokLower := strings.ToLower(tok)
			switch tokLower {
			case "diffusemap":
				val, err := lexer.NextToken()
				if err == nil {
					def.DiffuseMap = strings.ToLower(proc.StripQuotes(val))
				}
			case "bumpmap":
				val, err := lexer.NextToken()
				if err == nil {
					def.NormalMap = strings.ToLower(proc.StripQuotes(val))
				}
			case "specularmap":
				val, err := lexer.NextToken()
				if err == nil {
					def.SpecularMap = strings.ToLower(proc.StripQuotes(val))
				}
			default:
				fmt.Println("Unknown material property:", tokLower)
			}
		}

		m.defs[name] = def
	}
}

// Resolve returns a MaterialDef for the given material name, creating a new definition if it doesn't exist in the collection.
func (m *Materials) Resolve(materialName string) *MaterialDef {
	nameLower := strings.ToLower(materialName)
	if def, ok := m.defs[nameLower]; ok {
		return def
	}
	def := NewMaterialDef(nameLower)
	// Implicit Doom 3 convention fallback
	def.DiffuseMap = nameLower
	def.NormalMap = nameLower + "_local"
	def.SpecularMap = nameLower + "_s"
	m.defs[nameLower] = def
	return def
}

// CreateConfigMaterial creates and returns a configured `config.Material` based on the provided material name.
func (m *Materials) CreateConfigMaterial(materialName string) *config.Material {
	def := m.Resolve(materialName)
	textures := []string{def.DiffuseMap}
	mat := config.NewConfigMaterial(def.Name, textures, def.Kind)
	if def.NormalMap != "" {
		mat.NormalMap = def.NormalMap
	}
	if def.SpecularMap != "" {
		mat.SpecularMap = def.SpecularMap
		mat.SpecularBoost = 0.4
		mat.Shininess = 32.0
	}
	return mat
}
