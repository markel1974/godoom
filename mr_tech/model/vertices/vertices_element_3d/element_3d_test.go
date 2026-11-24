package vertices_element_3d

import (
	"testing"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/model/materials"
	"github.com/markel1974/godoom/mr_tech/model/volumes"
	"github.com/markel1974/godoom/mr_tech/physics/aabb"
	"github.com/markel1974/godoom/mr_tech/physics/collisions"
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

func countOverlaps(el *Element3D, obj aabb.IObject) int {
	hits := 0
	el.QueryOverlaps(obj, func(_ aabb.IObject) bool {
		hits++
		return false
	})
	return hits
}

func TestElement3DSharesSingleTreeAcrossFramesAndRebuildsOnChange(t *testing.T) {
	const (
		numFrames = 8
		trisFrame = 12
	)
	mat := config.NewConfigMaterial("mat", []string{"t"}, config.MaterialKindSolid)
	entryCfg := config.NewModel3DEntry(numFrames, nil, nil)
	entryCfg.GroupSize = 1.0
	entryCfg.ActionDefinitions = []string{"idle", "shift"}
	entryCfg.ActionIntervals = [][2]int{{0, 3}, {4, 7}}

	for f := 0; f < numFrames; f++ {
		baseX := 0.0
		if f >= 4 {
			baseX = 100.0
		}
		tris := make([]config.Model3DEntryTriangle, trisFrame)
		for k := 0; k < trisFrame; k++ {
			tri := config.NewModel3DEntryTriangle(mat)
			z := float64(k) * 0.25
			tri.Vertices[0].Pos = geometry.XYZ{X: baseX, Y: 0, Z: z}
			tri.Vertices[1].Pos = geometry.XYZ{X: baseX + 10, Y: 0, Z: z}
			tri.Vertices[2].Pos = geometry.XYZ{X: baseX, Y: 10, Z: z}
			tris[k] = tri
		}
		entryCfg.Frames[f] = config.NewModel3DEntryFrame(tris)
	}

	cfg := config.NewConfigThing("enemy_tree_test", geometry.XYZ{}, 0, config.ThingEnemyDef, 20, 10, 20, 100)
	cfg.Element3D = entryCfg
	tex := textures.NewTexture("t", 1, 2, 2, false)
	mats := materials.NewMaterials(&stubTextures{tex: tex})

	el := NewElement3D(cfg, mats)

	rootTree := el.GetVolume(0).GetFacesTree()
	if rootTree == nil {
		t.Fatalf("expected non-nil shared tree")
	}
	expectedCap := uint(trisFrame * 2)
	if rootTree.Capacity() != expectedCap {
		t.Fatalf("expected shared tree capacity %d, got %d", expectedCap, rootTree.Capacity())
	}
	for f := 0; f < numFrames; f++ {
		if got := el.GetVolume(f).GetFacesTree(); got != rootTree {
			t.Fatalf("frame %d expected shared tree %p, got %p", f, rootTree, got)
		}
	}
	if el.GetFacesA().GetTree() != nil || el.GetFacesB().GetTree() != nil {
		t.Fatalf("expected render morph facesA/facesB not to allocate unused AABB trees")
	}
	if el.treeRebuilds != 1 {
		t.Fatalf("expected exactly 1 tree build on construction, got %d", el.treeRebuilds)
	}

	// Steady state on the same frame must not rebuild the shared tree.
	el.SetAction(0)
	el.Compile(0)
	el.Compile(0)
	if el.treeRebuilds != 1 {
		t.Fatalf("expected no redundant rebuilds on unchanged frame/action, got %d", el.treeRebuilds)
	}

	// Action 0 faces live near X in [0..10].
	probeNear := newProbeObject(1, 1, -1, 3, 3, 5)
	probeFar := newProbeObject(101, 1, -1, 103, 3, 5)
	if countOverlaps(el, probeNear) == 0 || countOverlaps(el, probeFar) != 0 {
		t.Fatalf("expected action 0 overlaps near=non-zero far=0, got near=%d far=%d", countOverlaps(el, probeNear), countOverlaps(el, probeFar))
	}
	if _, _, ok := el.IntersectRayFace(2, 2, 10, 0, 0, -1, 100); !ok {
		t.Fatalf("expected ray hit on action 0 frame")
	}
	if _, _, ok := el.IntersectRayFace(102, 2, 10, 0, 0, -1, 100); ok {
		t.Fatalf("unexpected ray hit at x=102 while action 0 is active")
	}

	// Switching action must rebuild the shared tree for the new frame's faces.
	el.SetSalt(0)
	el.SetAction(1)
	if el.treeRebuilds != 2 {
		t.Fatalf("expected treeRebuilds == 2 after SetAction(1), got %d", el.treeRebuilds)
	}
	if countOverlaps(el, probeFar) == 0 || countOverlaps(el, probeNear) != 0 {
		t.Fatalf("expected action 1 overlaps far=non-zero near=0, got far=%d near=%d", countOverlaps(el, probeFar), countOverlaps(el, probeNear))
	}
	if _, _, ok := el.IntersectRayFace(102, 2, 10, 0, 0, -1, 100); !ok {
		t.Fatalf("expected ray hit on action 1 frame")
	}

	// Advancing animation to a different frame rebuilds once per frame change.
	el.Compile(1)
	if el.treeRebuilds != 3 {
		t.Fatalf("expected treeRebuilds == 3 when Compile(1) advances frame, got %d", el.treeRebuilds)
	}
	el.Compile(1)
	if el.treeRebuilds != 3 {
		t.Fatalf("expected treeRebuilds to stay 3 when frame stays identical, got %d", el.treeRebuilds)
	}

	// Deforming the active mesh triggers one shared-tree update and preserves deformed hit height.
	el.vols.SetAllowTessellate(false)
	if !el.Deform(geometry.XYZ{X: 100, Y: 0, Z: 0}, geometry.XYZ{X: 0, Y: 0, Z: 6}, 3.0) {
		t.Fatalf("expected Deform to modify deformable solid faces")
	}
	if el.treeRebuilds != 4 {
		t.Fatalf("expected single shared-tree rebuild after Deform, got %d", el.treeRebuilds)
	}
	hitDist, _, hit := el.IntersectRayFace(100.1, 0.1, 20, 0, 0, -1, 100)
	if !hit || hitDist >= 17.5 {
		t.Fatalf("expected ray to hit raised deformed surface near Z=6 (dist < 17.5), hit=%v dist=%.3f", hit, hitDist)
	}

	// Switching frames back and forth in the pre-sized shared tree must allocate zero heap memory.
	volA := el.GetVolume(0)
	volB := el.GetVolume(4)
	allocs := testing.AllocsPerRun(50, func() {
		volA.RebuildTree()
		volB.RebuildTree()
	})
	if allocs != 0 {
		t.Fatalf("expected 0 allocations during shared-tree frame switches, got %v", allocs)
	}
	actionAllocs := testing.AllocsPerRun(50, func() {
		el.SetAction(0)
		el.SetAction(1)
	})
	if actionAllocs != 0 {
		t.Fatalf("expected 0 allocations during SetAction switches, got %v", actionAllocs)
	}
	if rootTree.Capacity() != expectedCap {
		t.Fatalf("expected shared tree capacity to remain %d without growth, got %d", expectedCap, rootTree.Capacity())
	}

	// CenterFrames keeps the single shared tree and updates active-frame spatial coordinates.
	el.CenterFrames(15, 0.2, 0.5, 490)
	if el.GetVolume(0).GetFacesTree() != rootTree {
		t.Fatalf("expected CenterFrames to preserve shared tree pointer")
	}
	var centeredHits int
	el.QueryOverlaps(newProbeObject(-6, -6, -10, 6, 6, 10), func(obj aabb.IObject) bool {
		if fc, ok := obj.(*volumes.Face); ok && fc != nil {
			centeredHits++
		}
		return false
	})
	if centeredHits == 0 {
		t.Fatalf("expected QueryOverlaps near centered origin to hit active centered faces")
	}
}

func TestVolumesSharedTreeRebuildGeometryAndDeform(t *testing.T) {
	sharedTree := aabb.NewTree(8, 0.0)
	solver := collisions.NewSolver(true, 10, 0.1, 0.5, 0, 0, 9.8)

	tex := textures.NewTexture("solid", 1, 4, 4, false)
	mat := textures.NewMaterial("solid", []*textures.Texture{tex}, int(config.MaterialKindSolid))

	v0 := volumes.NewVolumeDynamicShared(0, "f0", "thing", sharedTree, solver)
	f0 := volumes.NewFace([3]geometry.XYZ{{X: 0, Y: 0, Z: 0}, {X: 4, Y: 0, Z: 0}, {X: 0, Y: 4, Z: 0}}, mat)
	v0.AddFace(f0)
	v0.RebuildGeometry()

	v1 := volumes.NewVolumeDynamicShared(1, "f1", "thing", sharedTree, solver)
	f1 := volumes.NewFace([3]geometry.XYZ{{X: 50, Y: 0, Z: 0}, {X: 54, Y: 0, Z: 0}, {X: 50, Y: 4, Z: 0}}, mat)
	v1.AddFace(f1)
	v1.RebuildGeometry()

	if _, ok := sharedTree.GetRoot(); ok {
		t.Fatalf("expected RebuildGeometry alone not to populate shared tree")
	}

	verBefore := v0.GetFaces().GetVersion()
	v0.RebuildTree()
	if got := v0.GetFaces().GetVersion(); got != verBefore {
		t.Fatalf("expected RebuildTree not to bump geometry version: got=%d want=%d", got, verBefore)
	}

	queryAt := func(x, y, z float64) []*volumes.Face {
		var out []*volumes.Face
		sharedTree.QueryPoint3d(x, y, z, func(obj aabb.IObject) bool {
			if fc, ok := obj.(*volumes.Face); ok {
				out = append(out, fc)
			}
			return false
		})
		return out
	}

	if hits := queryAt(1, 1, 0); len(hits) != 1 || hits[0] != f0 {
		t.Fatalf("expected only v0 face f0 in shared tree, got %v", hits)
	}

	vols := volumes.NewVolumesFromSlice([]*volumes.Volume{v0, v1})

	// Ensure Volumes.Deform on shared-tree volumes updates vertex geometry without overwriting the active tree with v1.
	v1.SetAllowTessellate(false)
	v0.RebuildTree()
	if !vols.Deform(geometry.XYZ{X: 50, Y: 0, Z: 0}, geometry.XYZ{X: 0, Y: 0, Z: 2}, 3.0) {
		t.Fatalf("expected Volumes.Deform to modify v1 deformable face")
	}
	if hits := queryAt(50.5, 0.5, 1.5); len(hits) != 0 {
		t.Fatalf("expected Volumes.Deform not to clobber shared tree with non-active frame v1")
	}
	v1.RebuildTree()
	if hits := queryAt(50.5, 0.5, 1.5); len(hits) != 1 || hits[0] != f1 {
		t.Fatalf("expected v1 deformed face in shared tree after v1.RebuildTree(), got %v", hits)
	}

	// Ensure BuildCenters on shared-tree volumes recenters geometry without clobbering the shared tree with the last volume.
	v0.RebuildTree()
	vols.BuildCenters()
	if hits := queryAt(50.5, 0.5, 1.5); len(hits) != 0 {
		t.Fatalf("expected BuildCenters not to leave stale uncentered coords in shared tree")
	}
	v1.RebuildTree()
	if hits := queryAt(-1.0, -1.0, 0.0); len(hits) != 1 || hits[0] != f1 {
		t.Fatalf("expected centered v1 face in shared tree after v1.RebuildTree(), got %v", hits)
	}
}

func makeTriFrame(mat *config.Material, minZ, maxZ float64) config.Model3DEntryFrame {
	tri := config.NewModel3DEntryTriangle(mat)
	tri.Vertices[0].Pos = geometry.XYZ{X: -8, Y: -8, Z: minZ}
	tri.Vertices[1].Pos = geometry.XYZ{X: 8, Y: -8, Z: minZ}
	tri.Vertices[2].Pos = geometry.XYZ{X: 0, Y: 8, Z: maxZ}
	return config.NewModel3DEntryFrame([]config.Model3DEntryTriangle{tri})
}

func TestElement3DSolverUsesStandingPoseNotFinalDeathFrame(t *testing.T) {
	mat := config.NewConfigMaterial("mat", []string{"t"}, config.MaterialKindSolid)
	entryCfg := config.NewModel3DEntry(3, []string{"attak01", "stand01", "death01"}, []string{"death"})
	entryCfg.GroupSize = 1.0
	entryCfg.Frames[0] = makeTriFrame(mat, -22, 22)
	entryCfg.Frames[1] = makeTriFrame(mat, -24, 32)
	entryCfg.Frames[2] = makeTriFrame(mat, -26, -10) // Last frame is a flat death pose (height 16 vs stand 56)
	entryCfg.Center([]string{"stand", "idle", "legs_idle", "hover"})

	cfg := config.NewConfigThing("soldier_q2_test", geometry.XYZ{}, 0, config.ThingEnemyDef, 30, 0, 0, 100)
	cfg.Element3D = entryCfg
	tex := textures.NewTexture("t", 1, 4, 4, false)
	mats := materials.NewMaterials(&stubTextures{tex: tex})

	el := NewElement3D(cfg, mats)
	ent := el.GetSolver().GetEntity()
	if depth := ent.GetDepth(); depth < 55.9 || depth > 56.1 {
		t.Fatalf("expected solver depth from standing frame (~56.0), got %.3f", depth)
	}

	// Ground entity bottom at floorZ=0 and switch to standing action: feet must sit on Z=0, not sink.
	ent.MoveTo(0, 0, 0)
	el.SetAction(1)
	el.Compile(0)
	_, _, centerZ := ent.GetCenter()
	_, _, minZ, _, _, _ := el.GetMinMax()
	if feetZ := centerZ + minZ; feetZ < -1e-4 || feetZ > 1e-4 {
		t.Fatalf("expected standing feet at floor Z=0.0, got %.4f (centerZ=%.3f minZ=%.3f)", feetZ, centerZ, minZ)
	}

	// Model without stand/idle/hover falls back to frame 0 rather than the final death frame.
	noStandCfg := config.NewModel3DEntry(2, []string{"walk01", "death01"}, []string{"death"})
	noStandCfg.Frames[0] = makeTriFrame(mat, -25, 25)
	noStandCfg.Frames[1] = makeTriFrame(mat, -25, -10)
	cfgFallback := config.NewConfigThing("fallback_test", geometry.XYZ{}, 0, config.ThingEnemyDef, 30, 0, 0, 100)
	cfgFallback.Element3D = noStandCfg
	elFallback := NewElement3D(cfgFallback, mats)
	if depth := elFallback.GetSolver().GetEntity().GetDepth(); depth < 49.9 || depth > 50.1 {
		t.Fatalf("expected fallback solver depth from frame 0 (50.0), got %.3f", depth)
	}
}

func TestElement3DTemplateSharingLRUAndIsolation(t *testing.T) {
	const numFrames = 12
	mat := config.NewConfigMaterial("mat", []string{"t"}, config.MaterialKindSolid)
	entryCfg := config.NewModel3DEntry(numFrames, nil, nil)
	entryCfg.GroupSize = 1.0
	entryCfg.ActionDefinitions = []string{"cycle"}
	entryCfg.ActionIntervals = [][2]int{{0, numFrames - 1}}

	for f := 0; f < numFrames; f++ {
		tri := config.NewModel3DEntryTriangle(mat)
		offsetX := float64(f * 20)
		tri.Vertices[0].Pos = geometry.XYZ{X: offsetX, Y: 0, Z: 0}
		tri.Vertices[1].Pos = geometry.XYZ{X: offsetX + 10, Y: 0, Z: 0}
		tri.Vertices[2].Pos = geometry.XYZ{X: offsetX, Y: 10, Z: 2}
		entryCfg.Frames[f] = config.NewModel3DEntryFrame([]config.Model3DEntryTriangle{tri})
	}

	cfg1 := config.NewConfigThing("clone_1", geometry.XYZ{}, 0, config.ThingEnemyDef, 20, 10, 20, 100)
	cfg1.Element3D = entryCfg
	cfg2 := config.NewConfigThing("clone_2", geometry.XYZ{}, 0, config.ThingEnemyDef, 20, 10, 20, 100)
	cfg2.Element3D = entryCfg

	tex := textures.NewTexture("t", 1, 2, 2, false)
	mats := materials.NewMaterials(&stubTextures{tex: tex})

	el1 := NewElement3D(cfg1, mats)
	el2 := NewElement3D(cfg2, mats)
	if el1.tpl == nil || el1.tpl != el2.tpl {
		t.Fatalf("expected cloned Element3D instances to share compiled elementTemplate")
	}
	if el1.materializedAll || len(el1.slots) != maxCachedElementSlots {
		t.Fatalf("expected LRU mode with %d slots, got materialized=%v slots=%d", maxCachedElementSlots, el1.materializedAll, len(el1.slots))
	}
	if el1.GetVolumesLen() != numFrames || el1.GetSolver() == el2.GetSolver() || el1.GetVolume(0).GetFacesTree() == el2.GetVolume(0).GetFacesTree() {
		t.Fatalf("expected shared template with independent runtime solvers and spatial trees")
	}

	el1.SetSalt(0)
	el2.SetSalt(0)
	el1.SetAction(0)
	el2.SetAction(0)

	// Sweep through all 12 frames (> 4 LRU slots) with zero heap allocations and verify recycled slots update bounds/trees.
	for step := uint64(0); step < numFrames; step++ {
		el1.Compile(step)
		minX, _, _, maxX, _, _ := el1.GetMinMax()
		expectedMinX := float64(step * 20)
		expectedCombinedMinX := expectedMinX
		if step == numFrames-1 {
			// frameB wraps to frame 0 (X=[0..10]), widening combined [volA, volB] minX to 0.
			expectedCombinedMinX = 0
		}
		if minX < expectedCombinedMinX-0.01 || minX > expectedCombinedMinX+0.01 || maxX < expectedMinX+9.99 {
			t.Fatalf("step %d expected X bounds near [%.1f, %.1f], got [%.2f, %.2f]", step, expectedCombinedMinX, expectedMinX+10, minX, maxX)
		}
		if _, _, ok := el1.IntersectRayFace(expectedMinX+2, 2, 10, 0, 0, -1, 50); !ok {
			t.Fatalf("step %d expected ray hit at recycled frame X=%.1f", step, expectedMinX+2)
		}
	}

	allocs := testing.AllocsPerRun(24, func() {
		for tick := uint64(0); tick < numFrames; tick++ {
			el1.Compile(tick)
		}
	})
	if allocs != 0 {
		t.Fatalf("expected 0 allocations cycling through 12 frames in LRU slots, got %v", allocs)
	}

	// Deform on el1 materializes el1 cleanly while leaving el2 untouched on its active frame.
	el1.Compile(0)
	el2.Compile(0)
	if !el1.Deform(geometry.XYZ{X: 0, Y: 0, Z: 0}, geometry.XYZ{X: 0, Y: 0, Z: 5}, 3.0) {
		t.Fatalf("expected el1 Deform to apply")
	}
	if !el1.materializedAll || el2.materializedAll {
		t.Fatalf("expected only deformed el1 to materialize all volumes (el1=%v el2=%v)", el1.materializedAll, el2.materializedAll)
	}
	dist1, _, hit1 := el1.IntersectRayFace(0.1, 0.1, 20, 0, 0, -1, 50)
	dist2, _, hit2 := el2.IntersectRayFace(0.1, 0.1, 20, 0, 0, -1, 50)
	if !hit1 || !hit2 || !(dist1 < dist2-2.0) {
		t.Fatalf("expected deformed el1 surface higher than untouched el2 (hit1=%v d1=%.3f hit2=%v d2=%.3f)", hit1, dist1, hit2, dist2)
	}
}
