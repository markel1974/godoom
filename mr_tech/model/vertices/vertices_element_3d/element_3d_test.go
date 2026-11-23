package vertices_element_3d

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

func TestElement3DSharesSingleSolverAcrossFramesAndReleasesConfig(t *testing.T) {
	const numFrames = 32
	mat := config.NewConfigMaterial("mat", []string{"t"}, config.MaterialKindSolid)
	frameNames := make([]string, numFrames)
	for i := range frameNames {
		frameNames[i] = "idle0"
	}
	entryCfg := config.NewModel3DEntry(numFrames, nil, nil)
	entryCfg.ActionDefinitions = []string{"idle"}
	entryCfg.ActionIntervals = [][2]int{{0, numFrames - 1}}
	for f := 0; f < numFrames; f++ {
		tri := config.NewModel3DEntryTriangle(mat)
		tri.Vertices[0].Pos = geometry.XYZ{X: 0, Y: 0, Z: 0}
		tri.Vertices[1].Pos = geometry.XYZ{X: 10, Y: 0, Z: 0}
		tri.Vertices[2].Pos = geometry.XYZ{X: 0, Y: 10, Z: float64(f + 1)}
		entryCfg.Frames[f] = config.NewModel3DEntryFrame([]config.Model3DEntryTriangle{tri})
	}

	cfg := config.NewConfigThing("enemy_test", geometry.XYZ{}, 0, config.ThingEnemyDef, 20, 10, 20, 100)
	cfg.Element3D = entryCfg

	tex := textures.NewTexture("t", 1, 2, 2, false)
	mats := materials.NewMaterials(&stubTextures{tex: tex})

	el := NewElement3D(cfg, mats)
	if el.GetVolumesLen() != numFrames {
		t.Fatalf("expected %d frame volumes, got %d", numFrames, el.GetVolumesLen())
	}

	rootSolver := el.GetSolver()
	if rootSolver == nil {
		t.Fatalf("expected non-nil root solver")
	}
	for f := 0; f < numFrames; f++ {
		vol := el.GetVolume(f)
		if vol == nil {
			t.Fatalf("frame %d volume is nil", f)
		}
		if vol.GetSolver() != rootSolver {
			t.Fatalf("frame %d expected shared solver %p, got %p", f, rootSolver, vol.GetSolver())
		}
	}

	el.CenterFrames(15, 0.2, 0.5, 490)
	centeredSolver := el.GetSolver()
	if centeredSolver == nil {
		t.Fatalf("expected non-nil solver after CenterFrames")
	}
	for f := 0; f < numFrames; f++ {
		if got := el.GetVolume(f).GetSolver(); got != centeredSolver {
			t.Fatalf("frame %d expected updated partSolver %p after CenterFrames, got %p", f, centeredSolver, got)
		}
	}
}
