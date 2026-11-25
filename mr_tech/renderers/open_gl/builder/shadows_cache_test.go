package builder

import (
	"testing"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/engine"
	"github.com/markel1974/godoom/mr_tech/generators/common"
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/physics/rigid_body"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/gpu/vramtex"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/metrics"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/shaders"
	"github.com/markel1974/godoom/mr_tech/textures"
)

func makeAnimatedCrateModel(mat *config.Material) *config.Model3DEntry {
	f0 := makeRigidCrateModel(mat).Frames[0]
	buildTri := func(a, b, c geometry.XYZ) config.Model3DEntryTriangle {
		tri := config.NewModel3DEntryTriangle(mat)
		tri.Vertices[0] = config.Model3DEntryVertex{Pos: a, U: 0, V: 0}
		tri.Vertices[1] = config.Model3DEntryVertex{Pos: b, U: 1, V: 0}
		tri.Vertices[2] = config.Model3DEntryVertex{Pos: c, U: 1, V: 1}
		return tri
	}
	v000 := geometry.XYZ{X: -14, Y: -14, Z: 0}
	v100 := geometry.XYZ{X: 14, Y: -14, Z: 0}
	v110 := geometry.XYZ{X: 14, Y: 14, Z: 0}
	v010 := geometry.XYZ{X: -14, Y: 14, Z: 0}
	v001 := geometry.XYZ{X: -14, Y: -14, Z: 30}
	v101 := geometry.XYZ{X: 14, Y: -14, Z: 30}
	v111 := geometry.XYZ{X: 14, Y: 14, Z: 30}
	v011 := geometry.XYZ{X: -14, Y: 14, Z: 30}
	tris1 := []config.Model3DEntryTriangle{
		buildTri(v001, v101, v111), buildTri(v001, v111, v011),
		buildTri(v000, v010, v110), buildTri(v000, v110, v100),
		buildTri(v000, v100, v101), buildTri(v000, v101, v001),
		buildTri(v110, v010, v011), buildTri(v110, v011, v111),
		buildTri(v010, v000, v001), buildTri(v010, v001, v011),
		buildTri(v100, v110, v111), buildTri(v100, v111, v101),
	}
	entry := config.NewModel3DEntry(2, []string{"idle0", "idle1"}, nil)
	entry.GroupSize = 1
	entry.Frames[0] = f0
	entry.Frames[1] = config.NewModel3DEntryFrame(tris1)
	return entry
}

func TestShadowsRigidCacheABASafetyAndSweep(t *testing.T) {
	var cache ShadowRigidCache
	e1 := rigid_body.NewEntity(10, 0.2, 0.5, 0, 0, 9.8)
	e2 := rigid_body.NewEntity(10, 0.2, 0.5, 0, 0, 9.8)

	s1 := cache.Retrieve(e1, 1)
	slot1 := e1.GetShadowMarker()
	if slot1 < 0 {
		t.Fatalf("expected non-negative assigned shadowMarker slot, got %d", slot1)
	}
	s1.staticFastOK = true
	s1.hash = 0xdeadbeef

	// Evict e1 after 256+ frames without seeing e1.
	cache.Sweep(300)
	if s1.inUse || s1.staticFastOK {
		t.Fatalf("expected swept entry released and fast state cleared")
	}

	// Simulate an ABA collision: give e2 the recycled slot index manually before Retrieve.
	e2.SetShadowMarker(slot1)
	s2 := cache.Retrieve(e2, 301)
	if s2.entID != e2.GetId() || s2.staticFastOK || s2.hash != 0 {
		t.Fatalf("expected recycled slot initialized cleanly for new entity id=%d, got id=%d fast=%v hash=%x", e2.GetId(), s2.entID, s2.staticFastOK, s2.hash)
	}
}

func TestShadowsEarlyReuseInvalidatesOnMoveDeformAndAnimatedTick(t *testing.T) {
	rigid_body.SetFps(120)
	textures.SetFps(120)
	defer rigid_body.SetFps(120)

	tA := textures.NewTexture("SHAD_P2", 1, 8, 8, false)
	for x := range 8 {
		for y := range 8 {
			tA.SetDiffuse(x, y, 0x7f7f7fff)
		}
	}
	texH := &multiTexHandler{frames: []*textures.Texture{tA}}
	calCfg := config.NewConfigCalibration()
	calCfg.AspectRatio = 1.1
	calCfg.OrthoSize = 640

	matSolid := config.NewConfigMaterial("solid", []string{"SHAD_P2"}, config.MaterialKindSolid)
	matCrate := config.NewConfigMaterial("crate", []string{"SHAD_P2"}, config.MaterialKindSolid)

	secs := []*config.Sector{
		makeBoxSector("center", 0, 0, 60, 0, 160, matSolid),
		makeBoxSector("behind", -220, 0, 50, 0, 160, matSolid),
		makeBoxSector("side", 0, 1250, 50, 0, 160, matSolid),
	}

	p := config.NewConfigPlayer(geometry.XYZ{X: 0, Y: 0, Z: 20}, 0.0, 40, 1800, 16.0, 60.0)
	_ = common.NewPlayer(p)
	p.GForce = 9.8 * 50

	animThing := config.NewConfigItem("anim_behind", geometry.XYZ{X: -210, Y: 0, Z: 0}, 0, 10, 0)
	animThing.Radius, animThing.Height, animThing.GForce = 16, 24, 9.8*50
	animThing.Element3D = makeAnimatedCrateModel(matCrate)

	staticSide := config.NewConfigItem("static_side", geometry.XYZ{X: 0, Y: 1240, Z: 0}, 0, 10, 0)
	staticSide.Radius, staticSide.Height, staticSide.GForce = 16, 24, 9.8*50
	staticSide.Element3D = makeRigidCrateModel(matCrate)

	root := config.NewConfigRoot(calCfg, secs, p, []*config.Thing{animThing, staticSide}, geometry.XYZ{X: 1, Y: 1, Z: 1}, texH)
	en := engine.NewEngine()
	if err := en.Setup(root); err != nil {
		t.Fatalf("engine setup failed: %v", err)
	}

	pl := en.GetPlayer()
	vi := engine.NewViewMatrix()
	vi.Update(pl)

	rCtx := &shadowTestCtx{}
	glTex := vramtex.NewTextures(rCtx)
	if err := glTex.Setup(en.GetTextures()); err != nil {
		t.Fatalf("textures setup failed: %v", err)
	}
	mCal := en.GetCalibration()
	mCal.EnableShadows = true
	mCal.EnableShadowLights = true
	mCal.MSAASamples = 0

	bv := NewBuilder(rCtx, glTex, mCal, pl)
	h := shaders.NewHandler(bv.GetVerticesStride(), bv.GetLightsStride())
	if err := h.Setup(rCtx, mCal, pl.GetFlash(), 640, 480); err != nil {
		t.Fatalf("Handler.Setup: %v", err)
	}
	if err := h.GetMain().Init(); err != nil {
		t.Fatalf("Main.Init: %v", err)
	}
	d := h.GetDepth()
	if err := d.Compile(testShaderAssets{}); err != nil {
		t.Fatalf("Depth.Compile: %v", err)
	}
	d.SetShadowsEnabled(true)

	mm := h.GetMapMetrics()
	mm.SetOrthoSize(640, 0.1, 4096)
	mm.SetCenter(0, 0, 0)
	mm.Update(vi, 1.0, 1.0)
	fm := h.GetFlashlightMetrics()
	fm.UpdateWithAim(mm.GetViewPtr(), 0, 0, 0, 0, 1)

	var sl [metrics.ShadowLightsCount]*metrics.Light
	sl[0] = metrics.NewLight()
	sl[0].Assign(-80, 60, 0, 1, 1, 1, 1, 2.0, -1, -0.35, 0, 100, 0.9, 0.7, 1.0)
	sl[1] = metrics.NewLight()
	sl[1].Assign(0, 60, -1100, 1, 1, 1, 1, 2.0, 0, -0.35, -1, 100, 0.9, 0.7, 1.0)
	h.GetShadowLightsMetrics().Update(sl, 2)

	warmUp := func() {
		bv.Compute(640, 480, vi, en)
		for range 2 {
			d.Prepare()
			bv.ComputeShadows(en, d)
			d.RenderWithTiles(bv.GetDrawCommandsOpaque(), bv.GetShadowTiles())
		}
	}
	warmUp()
	tiles := bv.GetShadowTiles()

	// Step 1: Rotate ONLY Flash across >260 frames so Spotlights & Room hit early-reuse;
	// partial queries must not falsely sweep Room/Spotlight entries or drop reuse.
	for step := range 270 {
		aim := float32((step%8)+1) * 0.002
		fm.UpdateWithAim(mm.GetViewPtr(), aim, 0, aim, 0, 1)
		d.Prepare()
		bv.ComputeShadows(en, d)
		if !tiles[shaders.ShadowTileRoom].Reuse || !tiles[shaders.ShadowTileLightBase].Reuse || !tiles[shaders.ShadowTileLightBase+1].Reuse {
			t.Fatalf("step %d expected Room, Spot0, Spot1 Reuse=true on FlashRotateOnly, got room=%v spot0=%v spot1=%v",
				step, tiles[shaders.ShadowTileRoom].Reuse, tiles[shaders.ShadowTileLightBase].Reuse, tiles[shaders.ShadowTileLightBase+1].Reuse)
		}
		if tiles[shaders.ShadowTileFlash].Reuse {
			t.Fatalf("step %d expected Flash dirty after aim change", step)
		}
		d.RenderWithTiles(bv.GetDrawCommandsOpaque(), tiles)
	}
	sideEnt := en.GetThings().GetThing("static_side").GetEntity()
	sideSlot := sideEnt.GetShadowMarker()
	if sideSlot < 0 || !bv.bShadow.entityCache.entries[sideSlot].inUse || !bv.bShadow.entityCache.entries[sideSlot].staticFastOK {
		t.Fatalf("expected static_side cache slot & staticFastOK preserved across >260 Flash-only frames")
	}

	// Step 2: Advance GlobalTick() and compile anim_behind externally (as simulation/Main does, NOT inside Shadows query):
	// animated crate in Room/Spot0 must invalidate Room & Spot0, whereas Spot1 remains Reuse=true.
	fm.UpdateWithAim(mm.GetViewPtr(), 0, 0, 0, 0, 1)
	warmUp()
	textures.Tick()
	en.GetThings().GetThing("anim_behind").Compile(textures.GlobalTick())
	d.Prepare()
	bv.ComputeShadows(en, d)
	if tiles[shaders.ShadowTileRoom].Reuse || tiles[shaders.ShadowTileLightBase].Reuse {
		t.Fatalf("expected Room & Spot0 invalidated when tick advances on multi-frame anim_behind, got room=%v spot0=%v",
			tiles[shaders.ShadowTileRoom].Reuse, tiles[shaders.ShadowTileLightBase].Reuse)
	}
	if !tiles[shaders.ShadowTileLightBase+1].Reuse {
		t.Fatalf("expected Spot1 (tick-independent) to remain Reuse=true when GlobalTick advances")
	}
	d.RenderWithTiles(bv.GetDrawCommandsOpaque(), tiles)

	// Step 2b: Verify Shadows.Process is read-only and does NOT call Compile(tick) during tree queries
	// (sleeping off-camera anim_behind keeps its current UVs/pose when only GlobalTick advances without simulation Compile).
	warmUp()
	prevFacesA, prevFacesB, prevLerp, _ := en.GetThings().GetThing("anim_behind").GetUIFaces()
	textures.Tick()
	d.Prepare()
	bv.ComputeShadows(en, d)
	afterFacesA, afterFacesB, afterLerp, _ := en.GetThings().GetThing("anim_behind").GetUIFaces()
	if afterFacesA != prevFacesA || afterFacesB != prevFacesB || afterLerp != prevLerp {
		t.Fatalf("expected ComputeShadows never to mutate thing animation state via Compile(tick) during query")
	}
	d.RenderWithTiles(bv.GetDrawCommandsOpaque(), tiles)

	// Step 3: Re-warm both buffers, then move static_side directly via MoveCenterTo -> Spot1 invalidates!
	warmUp()
	sideEnt.MoveCenterTo(4, 1245, 0)
	d.Prepare()
	bv.ComputeShadows(en, d)
	if tiles[shaders.ShadowTileLightBase+1].Reuse {
		t.Fatalf("expected Spot1 Reuse=false immediately after static_side.MoveCenterTo")
	}
	d.RenderWithTiles(bv.GetDrawCommandsOpaque(), tiles)

	// Step 4: Re-warm, then mark behind sector volume dirty -> Room and Spot0 must invalidate, Spot1 stays cached.
	warmUp()
	rawVols := en.GetVolumes().GetVolumes()
	dirtiedBehind := false
	for viIdx := 0; viIdx < rawVols.Len(); viIdx++ {
		v := rawVols.Get(viIdx)
		if v == nil || v.GetAABB() == nil {
			continue
		}
		cx, _, _ := v.GetAABB().GetCentroid()
		if cx < -150 {
			v.SetDirty(true)
			dirtiedBehind = true
			break
		}
	}
	if !dirtiedBehind {
		t.Fatalf("expected to locate behind volume by centroid")
	}
	d.Prepare()
	bv.ComputeShadows(en, d)
	if tiles[shaders.ShadowTileRoom].Reuse || tiles[shaders.ShadowTileLightBase].Reuse {
		t.Fatalf("expected Room and Spot0 Reuse=false when behind volume marked dirty (room=%v spot0=%v)",
			tiles[shaders.ShadowTileRoom].Reuse, tiles[shaders.ShadowTileLightBase].Reuse)
	}
	if !tiles[shaders.ShadowTileLightBase+1].Reuse {
		t.Fatalf("expected Spot1 Reuse=true when unrelated behind volume dirtied")
	}
	d.RenderWithTiles(bv.GetDrawCommandsOpaque(), tiles)
}
