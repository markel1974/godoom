package proc

import (
	"io"
	"path"
	"strconv"
	"strings"
)

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

// ResolvedEntityDef holds the fully resolved model, head, animation, and physical dependencies for a Doom 3 entity.
type ResolvedEntityDef struct {
	Classname  string
	Spawnclass string
	ModelPath  string
	HeadMesh   string
	HeadJoint  string
	IdleAnim   string
	WalkAnim   string
	RunAnim    string
	AttackAnim string
	PainAnim   string
	DeathAnim  string
	Mass       float64
	Health     float64
}

// Defs manages all entityDef and model declarations parsed from Doom 3 .def files in the PK4 archive.
type Defs struct {
	EntityDefs map[string]*EntityDef
	ModelDefs  map[string]*ModelDef
}

// NewDefs creates and returns a new empty Defs registry.
func NewDefs() *Defs {
	return &Defs{
		EntityDefs: make(map[string]*EntityDef),
		ModelDefs:  make(map[string]*ModelDef),
	}
}

// LoadFromArchive scans the "def" directory of the PK4 archive using ReadDirFilter and parses all .def files.
func (d *Defs) LoadFromArchive(arc IPkArchive) error {
	defFiles, _ := arc.ReadDirFilter("def", "\\.def$")
	for _, defName := range defFiles {
		fullPath := "def" + Pak4Separator + defName
		f, err := arc.Open(fullPath)
		if err != nil {
			continue
		}
		data, err := io.ReadAll(f)
		if err != nil {
			continue
		}
		d.parseDefData(data)
	}
	return nil
}

// parseDefData parses entityDef and model blocks from raw .def file bytes.
func (d *Defs) parseDefData(data []byte) {
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
			ed, ok := d.EntityDefs[name]
			if !ok {
				ed = &EntityDef{
					Name:       name,
					Properties: make(map[string]string),
				}
				d.EntityDefs[name] = ed
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
			md, ok := d.ModelDefs[name]
			if !ok {
				md = &ModelDef{
					Name:  name,
					Anims: make(map[string]string),
				}
				d.ModelDefs[name] = md
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

func cleanAssetPath(p string) string {
	return strings.TrimSpace(strings.ToLower(strings.ReplaceAll(p, "\\", Pak4Separator)))
}

// GetEntityProperty resolves a property key on an entityDef following its "inherit" chain.
func (d *Defs) GetEntityProperty(classname, key string) (string, bool) {
	return d.getEntityPropertyVisited(strings.ToLower(classname), strings.ToLower(key), make(map[string]bool))
}

func (d *Defs) getEntityPropertyVisited(classname, key string, visited map[string]bool) (string, bool) {
	if visited[classname] {
		return "", false
	}
	visited[classname] = true
	ed, ok := d.EntityDefs[classname]
	if !ok {
		return "", false
	}
	if val, exists := ed.Properties[key]; exists {
		return val, true
	}
	if parent, hasParent := ed.Properties["inherit"]; hasParent && len(parent) > 0 {
		return d.getEntityPropertyVisited(strings.ToLower(parent), key, visited)
	}
	return "", false
}

// ResolveModel resolves a model block name following its inherit chain, returning the mesh path and merged anim map.
func (d *Defs) ResolveModel(modelName string) (string, map[string]string, bool) {
	return d.resolveModelVisited(strings.ToLower(modelName), make(map[string]bool))
}

func (d *Defs) resolveModelVisited(modelName string, visited map[string]bool) (string, map[string]string, bool) {
	if visited[modelName] {
		return "", nil, false
	}
	visited[modelName] = true
	md, ok := d.ModelDefs[modelName]
	if !ok {
		return "", nil, false
	}

	var mesh string
	anims := make(map[string]string)

	if len(md.Inherit) > 0 {
		if pMesh, pAnims, pOk := d.resolveModelVisited(md.Inherit, visited); pOk {
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

// DiscoverModelAnimsFromDir scans the PK4 directory of a .md5mesh using ReadDirFilter and merges discovered .md5anim files.
func DiscoverModelAnimsFromDir(arc IPkArchive, meshPath string, anims map[string]string) map[string]string {
	if anims == nil {
		anims = make(map[string]string)
	}
	if arc == nil || len(meshPath) == 0 {
		return anims
	}
	meshDir := path.Dir(cleanAssetPath(meshPath))
	if len(meshDir) == 0 || meshDir == "." {
		return anims
	}
	files, err := arc.ReadDirFilter(meshDir, "\\.md5anim$")
	if err != nil {
		return anims
	}
	for _, rel := range files {
		// Only direct children of meshDir to avoid pulling sub-character anims with different skeletons
		if strings.Contains(rel, Pak4Separator) {
			continue
		}
		fullPath := meshDir + Pak4Separator + rel
		stem := strings.TrimSuffix(strings.ToLower(rel), ".md5anim")
		if _, exists := anims[stem]; !exists {
			anims[stem] = fullPath
		}
	}
	return anims
}

func pickAnim(arc IPkArchive, anims map[string]string, exactKeys []string, substrings []string) string {
	fileExists := func(p string) bool {
		if len(p) == 0 {
			return false
		}
		if arc == nil {
			return true
		}
		_, err := arc.Open(p)
		return err == nil
	}

	for _, k := range exactKeys {
		if p, ok := anims[k]; ok && fileExists(p) {
			return p
		}
	}
	for _, sub := range substrings {
		for k, p := range anims {
			if strings.Contains(k, sub) && fileExists(p) {
				return p
			}
		}
	}
	return ""
}

// ResolveEntity reconstructs all model, head, animation, and physical dependencies for a classname from .def files and PK4 directory scanning.
func (d *Defs) ResolveEntity(arc IPkArchive, classname string) (*ResolvedEntityDef, bool) {
	cn := strings.ToLower(strings.TrimSpace(classname))
	if _, ok := d.EntityDefs[cn]; !ok {
		return nil, false
	}

	res := &ResolvedEntityDef{
		Classname: cn,
	}
	if sc, ok := d.GetEntityProperty(cn, "spawnclass"); ok {
		res.Spawnclass = sc
	}
	if hStr, ok := d.GetEntityProperty(cn, "health"); ok {
		if h, err := strconv.ParseFloat(hStr, 64); err == nil {
			res.Health = h
		}
	}
	if mStr, ok := d.GetEntityProperty(cn, "mass"); ok {
		if m, err := strconv.ParseFloat(mStr, 64); err == nil {
			res.Mass = m
		}
	}

	modelProp, _ := d.GetEntityProperty(cn, "model")
	modelPropClean := cleanAssetPath(modelProp)

	var anims map[string]string
	if strings.HasSuffix(modelPropClean, ".md5mesh") || strings.HasSuffix(modelPropClean, ".lwo") {
		res.ModelPath = modelPropClean
	} else if len(modelPropClean) > 0 {
		if mesh, mAnims, ok := d.ResolveModel(modelPropClean); ok {
			res.ModelPath = mesh
			anims = mAnims
		}
	}

	// Resolve attached head ("def_head" -> entityDef/model -> mesh)
	if headDef, ok := d.GetEntityProperty(cn, "def_head"); ok && len(headDef) > 0 {
		headClean := cleanAssetPath(headDef)
		if strings.HasSuffix(headClean, ".md5mesh") {
			res.HeadMesh = headClean
		} else if headMesh, _, hOk := d.ResolveModel(headClean); hOk && len(headMesh) > 0 {
			res.HeadMesh = headMesh
		} else if headModelProp, hOk := d.GetEntityProperty(headClean, "model"); hOk {
			hmClean := cleanAssetPath(headModelProp)
			if strings.HasSuffix(hmClean, ".md5mesh") {
				res.HeadMesh = hmClean
			} else if headMesh, _, mOk := d.ResolveModel(hmClean); mOk {
				res.HeadMesh = headMesh
			}
		}
	}
	if hj, ok := d.GetEntityProperty(cn, "head_joint"); ok && len(hj) > 0 {
		res.HeadJoint = hj
	}

	// If it's an MD5 mesh, enrich animations via ReadDirFilter on the mesh directory in the PK4
	if strings.HasSuffix(res.ModelPath, ".md5mesh") {
		anims = DiscoverModelAnimsFromDir(arc, res.ModelPath, anims)
		res.IdleAnim = pickAnim(
			arc,
			anims,
			[]string{"idle", "idle1", "idle2", "stand", "alert_idle", "crawl_idle", "on4_idle", "idle_stand1", "zidle1", "zctidle", "chaingun_idle", "machinegun_stand", "shield_stand", "playdead"},
			[]string{"idle", "stand"},
		)
		res.WalkAnim = pickAnim(
			arc,
			anims,
			[]string{"walk", "walk1", "walk2", "walk3", "walk4", "walk7", "walk1_r", "zwalk1", "fastwalk", "crawl1", "travel", "travel3", "chaingun_walk", "machinegun_walk", "shield_walk", "run"},
			[]string{"walk", "crawl", "travel"},
		)
		res.RunAnim = pickAnim(
			arc,
			anims,
			[]string{"run", "run1", "run2", "run3", "run_r", "faster_walk", "fastwalk", "zctrun5", "crawl2", "chaingun_run", "machinegun_run", "shield_run", "walk", "walk1", "walk2"},
			[]string{"run", "fastwalk", "faster_walk"},
		)
		if len(res.RunAnim) == 0 {
			res.RunAnim = res.WalkAnim
		}
		if len(res.WalkAnim) == 0 {
			res.WalkAnim = res.RunAnim
		}
		res.AttackAnim = pickAnim(
			arc,
			anims,
			[]string{
				"range_attack1", "range_attack2", "range1", "attack", "attack1", "attack2", "attack3",
				"melee_attack1", "melee_attack2", "melee1", "melee_left", "melee1_walk", "on4_melee1",
				"stand_fire", "fire1", "chaingun_stand_fire", "machinegun_stand_fire", "shield_stand_fire",
				"oneshot", "attack_bfg", "range_tentacle", "leftclawattack", "slash1", "swing1", "mocapswing1", "pipeattack1",
			},
			[]string{"attack", "fire", "melee", "range", "swing", "slash"},
		)
		res.PainAnim = pickAnim(
			arc,
			anims,
			[]string{
				"pain", "pain_chest", "pain_chest1", "pain_chest2", "chest_pain", "chest_pain1", "chestpain", "chestpain1",
				"chaingun_pain_chest", "machinegun_chest_pain", "shield_chest_pain", "crawl_pain_chest", "pain_chest_on4",
				"pain1", "pain3", "pain_big1", "pain_head", "pain_head1", "pain_head3", "head_pain", "headpain", "headpain1",
				"bellypain", "leftarmpain", "walk1_pain",
			},
			[]string{"pain"},
		)
		res.DeathAnim = pickAnim(
			arc,
			anims,
			[]string{"death", "die", "playdead", "death1_pose", "death_pose"},
			[]string{"death", "die", "playdead"},
		)
	}

	return res, true
}
