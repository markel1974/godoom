package vertices_multi_sprite

import (
	"testing"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/model/materials"
	"github.com/markel1974/godoom/mr_tech/physics/aabb"
	"github.com/markel1974/godoom/mr_tech/textures"
)

type probeObject struct {
	box *aabb.AABB
}

func newProbeObject(minX, minY, minZ, maxX, maxY, maxZ float64) *probeObject {
	box := aabb.NewAABB()
	box.Rebuild(minX, minY, minZ, maxX, maxY, maxZ)
	return &probeObject{box: box}
}

func (p *probeObject) GetAABB() *aabb.AABB {
	return p.box
}

type stubTextures struct {
	tex *textures.Texture
}

func (s *stubTextures) GetNames() []string { return []string{"t"} }
func (s *stubTextures) GetFrames(_ []string) []*textures.Texture {
	return []*textures.Texture{s.tex}
}

func TestMultiSpriteSharesSingleSolverAndReleasesConfig(t *testing.T) {
	msCfg := config.NewMultiSprite()
	for i := 0; i < 6; i++ {
		msCfg.Add(config.NewConfigMaterial("mat", []string{"t"}, config.MaterialKindSolid))
	}
	cfg := config.NewConfigThing("sprite_test", geometry.XYZ{}, 0, config.ThingItemDef, 10, 10, 20, 0)
	cfg.MultiSprite = msCfg

	tex := textures.NewTexture("t", 1, 8, 8, false)
	mats := materials.NewMaterials(&stubTextures{tex: tex})

	ms := NewMultiSprite(cfg, mats)
	if ms.cfg != nil {
		t.Fatalf("expected MultiSprite.cfg cleared after construction")
	}
	rootSolver := ms.GetSolver()
	if rootSolver == nil {
		t.Fatalf("expected non-nil solver")
	}
	for i, vol := range ms.volumes {
		if vol == nil || vol.GetSolver() != rootSolver {
			t.Fatalf("view %d solver mismatch: got=%v want=%p", i, vol.GetSolver(), rootSolver)
		}
	}
}

type stubMultiTextures struct {
	frames map[string]*textures.Texture
}

func (s *stubMultiTextures) GetNames() []string {
	return []string{"tex8", "tex32", "tex64"}
}

func (s *stubMultiTextures) GetFrames(names []string) []*textures.Texture {
	if len(names) == 0 {
		return nil
	}
	if tex, ok := s.frames[names[0]]; ok {
		return []*textures.Texture{tex}
	}
	return nil
}

func countSpriteOverlaps(ms *MultiSprite, obj aabb.IObject) int {
	hits := 0
	ms.QueryOverlaps(obj, func(_ aabb.IObject) bool {
		hits++
		return false
	})
	return hits
}

func TestMultiSpriteSharesSingleTreeAcrossViewsAndRebuildsOnChange(t *testing.T) {
	msCfg := config.NewMultiSprite()
	msCfg.Add(config.NewConfigMaterial("mat8", []string{"tex8"}, config.MaterialKindSolid))
	msCfg.Add(config.NewConfigMaterial("mat32", []string{"tex32"}, config.MaterialKindSolid))
	msCfg.Add(config.NewConfigMaterial("mat64", []string{"tex64"}, config.MaterialKindSolid))

	cfg := config.NewConfigThing("sprite_tree_test", geometry.XYZ{}, 0, config.ThingItemDef, 10, 10, 20, 0)
	cfg.MultiSprite = msCfg

	mats := materials.NewMaterials(&stubMultiTextures{
		frames: map[string]*textures.Texture{
			"tex8":  textures.NewTexture("tex8", 1, 8, 8, false),
			"tex32": textures.NewTexture("tex32", 1, 32, 32, false),
			"tex64": textures.NewTexture("tex64", 1, 64, 64, false),
		},
	})

	ms := NewMultiSprite(cfg, mats)
	rootTree := ms.volumes[0].GetFacesTree()
	if rootTree == nil {
		t.Fatalf("expected non-nil shared tree")
	}
	startCap := rootTree.Capacity()
	for i, vol := range ms.volumes {
		if vol.GetFacesTree() != rootTree {
			t.Fatalf("view %d expected shared tree %p, got %p", i, rootTree, vol.GetFacesTree())
		}
	}
	if ms.treeRebuilds != 1 {
		t.Fatalf("expected 1 initial tree build, got %d", ms.treeRebuilds)
	}

	// Repeating the same view or compiling must not rebuild the tree.
	ms.SetAction(0)
	ms.Compile(0)
	ms.Compile(1)
	if ms.treeRebuilds != 1 {
		t.Fatalf("expected no redundant rebuilds for same view, got %d", ms.treeRebuilds)
	}

	// View 0 (8x8) occupies X in [-4, 4]; probe at X in [10, 12] should only hit larger views (32x32 / 64x64).
	outerProbe := newProbeObject(10, -1, -2, 12, 1, 2)
	if countSpriteOverlaps(ms, outerProbe) != 0 {
		t.Fatalf("unexpected overlap on 8x8 view at X=[10,12]")
	}

	ms.SetAction(1)
	if ms.treeRebuilds != 2 {
		t.Fatalf("expected treeRebuilds == 2 after switching to view 1, got %d", ms.treeRebuilds)
	}
	if countSpriteOverlaps(ms, outerProbe) == 0 {
		t.Fatalf("expected overlap on 32x32 view at X=[10,12]")
	}

	ms.SetAction(2)
	if ms.treeRebuilds != 3 {
		t.Fatalf("expected treeRebuilds == 3 after switching to view 2, got %d", ms.treeRebuilds)
	}
	if countSpriteOverlaps(ms, outerProbe) == 0 {
		t.Fatalf("expected overlap on 64x64 view at X=[10,12]")
	}

	ms.SetAction(0)
	if ms.treeRebuilds != 4 {
		t.Fatalf("expected treeRebuilds == 4 after returning to view 0, got %d", ms.treeRebuilds)
	}
	if countSpriteOverlaps(ms, outerProbe) != 0 {
		t.Fatalf("expected stale view 2 faces cleared when returning to view 0")
	}

	// Switching views in the shared tree must not allocate heap nodes or grow capacity.
	allocs := testing.AllocsPerRun(50, func() {
		ms.SetAction(1)
		ms.SetAction(0)
	})
	if allocs != 0 {
		t.Fatalf("expected 0 allocations switching views in shared tree, got %v", allocs)
	}
	if rootTree.Capacity() != startCap {
		t.Fatalf("expected shared tree capacity to remain %d, got %d", startCap, rootTree.Capacity())
	}
}
