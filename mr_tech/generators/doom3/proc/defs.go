package proc

import (
	"fmt"
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
func (d *Defs) ResolveEntity(classname string) (*ResolvedEntity, error) {
	cn := strings.ToLower(strings.TrimSpace(classname))
	if !d.def.HasEntity(cn) {
		return nil, fmt.Errorf("entity %q not found", classname)
	}
	res := NewResolvedEntity(cn)
	if err := res.Resolve(d.def, d.af); err != nil {
		return nil, err
	}
	return res, nil
}

func (d *Defs) PickAnim(anims map[string]string, exactKeys []string, substrings []string) string {
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

// DiscoverModelAnims scans the PK4 directory of a .md5mesh using ReadDirFilter and merges discovered .md5anim files.
func (d *Defs) DiscoverModelAnims(meshPath string, anims map[string]string) map[string]string {
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
