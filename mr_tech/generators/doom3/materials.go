package doom3

import (
	"io"
	"strings"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/generators/doom3/proc"
)

// IPkArchive abstracts file and directory access within a package archive.
type IPkArchive interface {
	Open(p string) (io.ReadSeeker, error)
	ReadDirFilter(p string, f string) ([]string, error)
}

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

// MaterialDef holds parsed texture maps and properties for a Doom 3 material shader.
type MaterialDef struct {
	Name        string
	DiffuseMap  string
	NormalMap   string
	SpecularMap string
	Kind        config.MaterialKind
}

func NewMaterialDef(matName string) *MaterialDef {
	mat := &MaterialDef{
		Name: matName,
		Kind: ToMaterial(matName),
	}
	return mat
}

// Materials manages and resolves Doom 3 materials from .mtr files and standard naming conventions.
type Materials struct {
	defs map[string]*MaterialDef
}

// NewMaterials creates an initialized Materials manager.
func NewMaterials() *Materials {
	return &Materials{
		defs: make(map[string]*MaterialDef),
	}
}

// LoadFromArchive scans and parses all .mtr files in the materials/ folder of the archive.
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
			}
		}

		m.defs[name] = def
	}
}

// Resolve looks up a MaterialDef by material name, falling back to Doom 3 suffix conventions if not defined.
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

// CreateConfigMaterial constructs a config.Material with diffuse, normal, and specular bindings.
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
