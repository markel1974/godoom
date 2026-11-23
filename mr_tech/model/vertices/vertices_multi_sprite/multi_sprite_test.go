package vertices_multi_sprite

import (
	"testing"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/model/materials"
	"github.com/markel1974/godoom/mr_tech/textures"
)

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
