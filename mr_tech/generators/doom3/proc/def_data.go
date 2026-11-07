package proc

import "strings"

// EntityDef represents a parsed Doom 3 entityDef block from a .def file.
type EntityDef struct {
	Name       string
	Properties map[string]string
}

// ModelDef represents a parsed Doom 3 model block from a .def file.
type ModelDef struct {
	Name    string
	Inherit string
	Mesh    string
	Anims   map[string]string
}

type DefData struct {
	entityDefs map[string]*EntityDef
	modelDefs  map[string]*ModelDef
}

func NewDefData() *DefData {
	return &DefData{
		entityDefs: make(map[string]*EntityDef),
		modelDefs:  make(map[string]*ModelDef),
	}
}

// Parse parses entityDef and model blocks from raw .def file bytes.
func (d *DefData) Parse(data []byte) {
	lexer := NewLexer(data)
	for {
		tok, err := lexer.NextToken()
		if err != nil {
			break
		}
		tokLower := strings.ToLower(StripQuotes(tok))
		switch tokLower {
		case "entitydef":
			nameTok, err := lexer.NextToken()
			if err != nil {
				return
			}
			name := strings.ToLower(StripQuotes(nameTok))
			if peek, err := lexer.PeekToken(); err != nil || peek != "{" {
				continue
			}
			_ = lexer.Expect("{")
			ed, ok := d.entityDefs[name]
			if !ok {
				ed = &EntityDef{
					Name:       name,
					Properties: make(map[string]string),
				}
				d.entityDefs[name] = ed
			}
			for {
				kTok, err := lexer.NextToken()
				if err != nil || kTok == "}" {
					break
				}
				if kTok == "{" {
					_ = lexer.SkipBlock()
					continue
				}
				vTok, err := lexer.NextToken()
				if err != nil || vTok == "}" {
					break
				}
				if vTok == "{" {
					_ = lexer.SkipBlock()
					continue
				}
				key := strings.ToLower(StripQuotes(kTok))
				val := StripQuotes(vTok)
				ed.Properties[key] = val
			}

		case "model":
			nameTok, err := lexer.NextToken()
			if err != nil {
				return
			}
			name := strings.ToLower(StripQuotes(nameTok))
			if peek, err := lexer.PeekToken(); err != nil || peek != "{" {
				continue
			}
			_ = lexer.Expect("{")
			md, ok := d.modelDefs[name]
			if !ok {
				md = &ModelDef{Name: name, Anims: make(map[string]string)}
				d.modelDefs[name] = md
			}
			depth := 1
			for depth > 0 {
				mTok, err := lexer.NextToken()
				if err != nil {
					break
				}
				if mTok == "{" {
					depth++
					continue
				}
				if mTok == "}" {
					depth--
					continue
				}
				if depth != 1 {
					continue
				}
				mLower := strings.ToLower(StripQuotes(mTok))
				switch mLower {
				case "inherit":
					if pTok, err := lexer.NextToken(); err == nil && pTok != "{" && pTok != "}" {
						md.Inherit = strings.ToLower(StripQuotes(pTok))
					}
				case "mesh":
					if meshTok, err := lexer.NextToken(); err == nil && meshTok != "{" && meshTok != "}" {
						md.Mesh = cleanAssetPath(StripQuotes(meshTok))
					}
				case "anim":
					aliasTok, err1 := lexer.NextToken()
					pathTok, err2 := lexer.NextToken()
					if err1 == nil && err2 == nil && aliasTok != "}" && pathTok != "}" {
						alias := strings.ToLower(StripQuotes(aliasTok))
						animPath := cleanAssetPath(strings.TrimSuffix(StripQuotes(pathTok), ","))
						if strings.HasSuffix(animPath, ".md5anim") {
							if _, exists := md.Anims[alias]; !exists {
								md.Anims[alias] = animPath
							}
						}
					}
				}
			}

		default:
			// Skip any other top-level block (e.g. export, skin, sound) if followed by '{'
			if peek, err := lexer.PeekToken(); err == nil && peek == "{" {
				_ = lexer.Expect("{")
				depth := 1
				for depth > 0 {
					t, err := lexer.NextToken()
					if err != nil {
						break
					}
					if t == "{" {
						depth++
					} else if t == "}" {
						depth--
					}
				}
			}
		}
	}
}

func (d *DefData) HasEntity(cn string) bool {
	if _, ok := d.entityDefs[cn]; !ok {
		return false
	}
	return true
}

func (d *DefData) ModelLen() int {
	return len(d.modelDefs)
}

func (d *DefData) EntityLen() int {
	return len(d.entityDefs)
}

// GetEntityProperty resolves a property key on an entityDef following its "inherit" chain.
func (d *DefData) GetEntityProperty(classname, key string) (string, bool) {
	visited := make(map[string]bool)
	return d.doGetEntity(strings.ToLower(classname), strings.ToLower(key), visited)
}

func (d *DefData) doGetEntity(classname, key string, visited map[string]bool) (string, bool) {
	if visited[classname] {
		return "", false
	}
	visited[classname] = true
	ed, ok := d.entityDefs[classname]
	if !ok {
		return "", false
	}
	if val, exists := ed.Properties[key]; exists {
		return val, true
	}
	if parent, hasParent := ed.Properties["inherit"]; hasParent && len(parent) > 0 {
		return d.doGetEntity(strings.ToLower(parent), key, visited)
	}
	return "", false
}

// ResolveModel resolves a model block name following its inherit chain, returning the mesh path and merged anim map.
func (d *DefData) ResolveModel(modelName string) (string, map[string]string, bool) {
	visited := make(map[string]bool)
	return d.doResolveModel(strings.ToLower(modelName), visited)
}

func (d *DefData) doResolveModel(modelName string, visited map[string]bool) (string, map[string]string, bool) {
	if visited[modelName] {
		return "", nil, false
	}
	visited[modelName] = true
	md, ok := d.modelDefs[modelName]
	if !ok {
		return "", nil, false
	}

	var mesh string
	anims := make(map[string]string)

	if len(md.Inherit) > 0 {
		if pMesh, pAnims, pOk := d.doResolveModel(md.Inherit, visited); pOk {
			mesh = pMesh
			for k, v := range pAnims {
				anims[k] = v
			}
		}
	}
	if len(md.Mesh) > 0 {
		mesh = md.Mesh
	}
	for k, v := range md.Anims {
		anims[k] = v
	}
	return mesh, anims, len(mesh) > 0 || len(anims) > 0
}
