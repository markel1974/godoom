package proc

import (
	"io"
	"path"
	"strings"
)

func cleanAssetPath(p string) string {
	return strings.TrimSpace(strings.ToLower(strings.ReplaceAll(p, "\\", Pak4Separator)))
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

// Load scans the "def" and "af" directories of the PK4 archive using ReadDirFilter and parses all .def and .af files.
func (d *Defs) Load() error {
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
func (d *Defs) ResolveAFBodies(ragdoll string) ([]*AFBody, bool) {
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

// ResolveEntity reconstructs all model, head, animation, and physical dependencies for a classname from .def files and PK4 directory scanning.
func (d *Defs) ResolveEntity(classname string) (*ResolvedEntity, bool) {
	cn := strings.ToLower(strings.TrimSpace(classname))
	if !d.def.HasEntity(cn) {
		return nil, false
	}

	res := NewResolvedEntity(cn)

	if err := res.Resolve(d.def, d.af); err != nil {
		return nil, false
	}

	// If it's an MD5 mesh, enrich animations via ReadDirFilter on the mesh directory in the PK4
	if strings.HasSuffix(res.ModelPath, ".md5mesh") {
		res.Anims = d.discoverModelAnims(res.ModelPath, res.Anims)
		res.IdleAnim = d.pickAnim(
			res.Anims,
			[]string{"idle", "idle1", "idle2", "stand", "alert_idle", "crawl_idle", "on4_idle", "idle_stand1", "zidle1", "zctidle", "chaingun_idle", "machinegun_stand", "shield_stand", "playdead"},
			[]string{"idle", "stand"},
		)
		res.WalkAnim = d.pickAnim(
			res.Anims,
			[]string{"walk", "walk1", "walk2", "walk3", "walk4", "walk7", "walk1_r", "zwalk1", "fastwalk", "crawl1", "travel", "travel3", "chaingun_walk", "machinegun_walk", "shield_walk", "run"},
			[]string{"walk", "crawl", "travel"},
		)
		res.RunAnim = d.pickAnim(
			res.Anims,
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
			res.Anims,
			[]string{
				"range_attack1", "range_attack2", "range1", "attack", "attack1", "attack2", "attack3",
				"melee_attack1", "melee_attack2", "melee1", "melee_left", "melee1_walk", "on4_melee1",
				"stand_fire", "fire1", "chaingun_stand_fire", "machinegun_stand_fire", "shield_stand_fire",
				"oneshot", "attack_bfg", "range_tentacle", "leftclawattack", "slash1", "swing1", "mocapswing1", "pipeattack1",
			},
			[]string{"attack", "fire", "melee", "range", "swing", "slash"},
		)
		res.PainAnim = d.pickAnim(
			res.Anims,
			[]string{
				"pain", "pain_chest", "pain_chest1", "pain_chest2", "chest_pain", "chest_pain1", "chestpain", "chestpain1",
				"chaingun_pain_chest", "machinegun_chest_pain", "shield_chest_pain", "crawl_pain_chest", "pain_chest_on4",
				"pain1", "pain3", "pain_big1", "pain_head", "pain_head1", "pain_head3", "head_pain", "headpain", "headpain1",
				"bellypain", "leftarmpain", "walk1_pain",
			},
			[]string{"pain"},
		)
		res.DeathAnim = d.pickAnim(
			res.Anims,
			[]string{"death", "die", "playdead", "death1_pose", "death_pose"},
			[]string{"death", "die", "playdead"},
		)
	}

	return res, true
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

// discoverModelAnims scans the PK4 directory of a .md5mesh using ReadDirFilter and merges discovered .md5anim files.
func (d *Defs) discoverModelAnims(meshPath string, anims map[string]string) map[string]string {
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
