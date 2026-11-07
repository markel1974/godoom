package proc

import (
	"io"
	"path"
	"strconv"
	"strings"
)

func cleanAssetPath(p string) string {
	return strings.TrimSpace(strings.ToLower(strings.ReplaceAll(p, "\\", Pak4Separator)))
}

// ResolvedEntityDef holds the fully resolved model, head, animation, and physical dependencies for a Doom 3 entity.
type ResolvedEntityDef struct {
	Classname  string
	SpawnClass string
	ModelPath  string
	HeadMesh   string
	HeadJoint  string
	Ragdoll    string
	AFBodies   []AFBody
	IdleAnim   string
	WalkAnim   string
	RunAnim    string
	AttackAnim string
	PainAnim   string
	DeathAnim  string
	Mass       float64
	Health     float64
}

// Defs manages all entityDef, model, and articulatedFigure declarations parsed from Doom 3 .def and .af files in the PK4 archive.
type Defs struct {
	arc IPkArchive
	af  *AFData
	def *DefData
}

// NewDefs creates and returns a new empty Defs registry.
func NewDefs(arc IPkArchive) *Defs {
	return &Defs{
		arc: arc,
		af:  NewAFData(),
		def: NewDefData(),
	}
}

// LoadFromArchive scans the "def" and "af" directories of the PK4 archive using ReadDirFilter and parses all .def and .af files.
func (d *Defs) LoadFromArchive() error {
	defFiles, _ := d.arc.ReadDirFilter("def", "\\.def$")
	for _, defName := range defFiles {
		fullPath := "def" + Pak4Separator + defName
		f, err := d.arc.Open(fullPath)
		if err != nil {
			continue
		}
		data, err := io.ReadAll(f)
		if err != nil {
			continue
		}
		d.def.Parse(data)
	}

	afFiles, _ := d.arc.ReadDirFilter("af", "\\.af$")
	for _, afName := range afFiles {
		fullPath := "af" + Pak4Separator + afName
		f, err := d.arc.Open(fullPath)
		if err != nil {
			continue
		}
		data, err := io.ReadAll(f)
		if err != nil {
			continue
		}
		stem := strings.TrimSuffix(strings.ToLower(path.Base(afName)), ".af")
		d.af.Parse(stem, data)
	}
	return nil
}

// ResolveAFBodies looks up the parsed AFBody slice for an articulated figure name or .af path.
func (d *Defs) ResolveAFBodies(ragdoll string) ([]AFBody, bool) {
	return d.af.ResolveBodies(ragdoll)
}

func (d *Defs) ResolveModel(modelName string) (string, map[string]string, bool) {
	return d.def.ResolveModel(modelName)
}

func (d *Defs) ModelLen() int {
	return d.def.ModelLen()
}

func (d *Defs) EntityLen() int {
	return d.def.EntityLen()
}

// discoverModelAnimsFromDir scans the PK4 directory of a .md5mesh using ReadDirFilter and merges discovered .md5anim files.
func (d *Defs) discoverModelAnimsFromDir(meshPath string, anims map[string]string) map[string]string {
	if anims == nil {
		anims = make(map[string]string)
	}
	if d.arc == nil || len(meshPath) == 0 {
		return anims
	}
	meshDir := path.Dir(cleanAssetPath(meshPath))
	if len(meshDir) == 0 || meshDir == "." {
		return anims
	}
	files, err := d.arc.ReadDirFilter(meshDir, "\\.md5anim$")
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

func (d *Defs) pickAnim(anims map[string]string, exactKeys []string, substrings []string) string {
	fileExists := func(p string) bool {
		if len(p) == 0 {
			return false
		}
		if d.arc == nil {
			return true
		}
		_, err := d.arc.Open(p)
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
func (d *Defs) ResolveEntity(classname string) (*ResolvedEntityDef, bool) {
	cn := strings.ToLower(strings.TrimSpace(classname))
	if !d.def.HasEntity(cn) {
		return nil, false
	}
	res := &ResolvedEntityDef{
		Classname: cn,
	}
	if sc, ok := d.def.GetEntityProperty(cn, "spawnclass"); ok {
		res.SpawnClass = sc
	}
	if hStr, ok := d.def.GetEntityProperty(cn, "health"); ok {
		if h, err := strconv.ParseFloat(hStr, 64); err == nil {
			res.Health = h
		}
	}
	if mStr, ok := d.def.GetEntityProperty(cn, "mass"); ok {
		if m, err := strconv.ParseFloat(mStr, 64); err == nil {
			res.Mass = m
		}
	}

	modelProp, _ := d.def.GetEntityProperty(cn, "model")
	modelPropClean := cleanAssetPath(modelProp)

	var anims map[string]string
	if strings.HasSuffix(modelPropClean, ".md5mesh") || strings.HasSuffix(modelPropClean, ".lwo") {
		res.ModelPath = modelPropClean
	} else if len(modelPropClean) > 0 {
		if mesh, mAnims, ok := d.def.ResolveModel(modelPropClean); ok {
			res.ModelPath = mesh
			anims = mAnims
		}
	}

	// Resolve attached head ("def_head" -> entityDef/model -> mesh)
	if headDef, ok := d.def.GetEntityProperty(cn, "def_head"); ok && len(headDef) > 0 {
		headClean := cleanAssetPath(headDef)
		if strings.HasSuffix(headClean, ".md5mesh") {
			res.HeadMesh = headClean
		} else if headMesh, _, hOk := d.def.ResolveModel(headClean); hOk && len(headMesh) > 0 {
			res.HeadMesh = headMesh
		} else if headModelProp, hOk := d.def.GetEntityProperty(headClean, "model"); hOk {
			hmClean := cleanAssetPath(headModelProp)
			if strings.HasSuffix(hmClean, ".md5mesh") {
				res.HeadMesh = hmClean
			} else if headMesh, _, mOk := d.def.ResolveModel(hmClean); mOk {
				res.HeadMesh = headMesh
			}
		}
	}
	if hj, ok := d.def.GetEntityProperty(cn, "head_joint"); ok && len(hj) > 0 {
		res.HeadJoint = hj
	}
	if rag, ok := d.def.GetEntityProperty(cn, "ragdoll"); ok && len(rag) > 0 {
		res.Ragdoll = rag
		if bodies, bOk := d.ResolveAFBodies(rag); bOk {
			res.AFBodies = bodies
		}
	} else if bodies, bOk := d.ResolveAFBodies(cn); bOk {
		res.Ragdoll = cn
		res.AFBodies = bodies
	}

	// If it's an MD5 mesh, enrich animations via ReadDirFilter on the mesh directory in the PK4
	if strings.HasSuffix(res.ModelPath, ".md5mesh") {
		anims = d.discoverModelAnimsFromDir(res.ModelPath, anims)
		res.IdleAnim = d.pickAnim(
			anims,
			[]string{"idle", "idle1", "idle2", "stand", "alert_idle", "crawl_idle", "on4_idle", "idle_stand1", "zidle1", "zctidle", "chaingun_idle", "machinegun_stand", "shield_stand", "playdead"},
			[]string{"idle", "stand"},
		)
		res.WalkAnim = d.pickAnim(
			anims,
			[]string{"walk", "walk1", "walk2", "walk3", "walk4", "walk7", "walk1_r", "zwalk1", "fastwalk", "crawl1", "travel", "travel3", "chaingun_walk", "machinegun_walk", "shield_walk", "run"},
			[]string{"walk", "crawl", "travel"},
		)
		res.RunAnim = d.pickAnim(
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
		res.AttackAnim = d.pickAnim(
			anims,
			[]string{
				"range_attack1", "range_attack2", "range1", "attack", "attack1", "attack2", "attack3",
				"melee_attack1", "melee_attack2", "melee1", "melee_left", "melee1_walk", "on4_melee1",
				"stand_fire", "fire1", "chaingun_stand_fire", "machinegun_stand_fire", "shield_stand_fire",
				"oneshot", "attack_bfg", "range_tentacle", "leftclawattack", "slash1", "swing1", "mocapswing1", "pipeattack1",
			},
			[]string{"attack", "fire", "melee", "range", "swing", "slash"},
		)
		res.PainAnim = d.pickAnim(
			anims,
			[]string{
				"pain", "pain_chest", "pain_chest1", "pain_chest2", "chest_pain", "chest_pain1", "chestpain", "chestpain1",
				"chaingun_pain_chest", "machinegun_chest_pain", "shield_chest_pain", "crawl_pain_chest", "pain_chest_on4",
				"pain1", "pain3", "pain_big1", "pain_head", "pain_head1", "pain_head3", "head_pain", "headpain", "headpain1",
				"bellypain", "leftarmpain", "walk1_pain",
			},
			[]string{"pain"},
		)
		res.DeathAnim = d.pickAnim(
			anims,
			[]string{"death", "die", "playdead", "death1_pose", "death_pose"},
			[]string{"death", "die", "playdead"},
		)
	}

	return res, true
}
