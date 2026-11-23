package open_gl

import (
	"testing"
	"unsafe"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/engine"
	"github.com/markel1974/godoom/mr_tech/generators/common"
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/model"
	"github.com/markel1974/godoom/mr_tech/model/ithing"
	"github.com/markel1974/godoom/mr_tech/model/things"
	"github.com/markel1974/godoom/mr_tech/model/volumes"
	"github.com/markel1974/godoom/mr_tech/physics/aabb"
	"github.com/markel1974/godoom/mr_tech/physics/rigid_body"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/gpu"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/metrics"
	"github.com/markel1974/godoom/mr_tech/textures"
)

type noopRenderCtx struct {
	api.IContext
	nextID uint32
}

func (c *noopRenderCtx) allocIDs(n int32, out *uint32) {
	if out == nil || n <= 0 {
		return
	}
	s := unsafe.Slice(out, int(n))
	for i := range s {
		c.nextID++
		s[i] = c.nextID
	}
}

func (c *noopRenderCtx) Ptr(_ interface{}) unsafe.Pointer                       { return nil }
func (c *noopRenderCtx) PtrOffset(offset int) unsafe.Pointer                    { return unsafe.Pointer(uintptr(offset)) }
func (c *noopRenderCtx) GenQueries(n int32, q *uint32)                          { c.allocIDs(n, q) }
func (c *noopRenderCtx) GenVertexArrays(n int32, a *uint32)                     { c.allocIDs(n, a) }
func (c *noopRenderCtx) GenBuffers(n int32, b *uint32)                          { c.allocIDs(n, b) }
func (c *noopRenderCtx) DeleteBuffers(_ int32, _ *uint32)                       {}
func (c *noopRenderCtx) DeleteVertexArrays(_ int32, _ *uint32)                  {}
func (c *noopRenderCtx) GenTextures(n int32, t *uint32)                         { c.allocIDs(n, t) }
func (c *noopRenderCtx) BindVertexArray(_ uint32)                               {}
func (c *noopRenderCtx) BindBuffer(_, _ uint32)                                 {}
func (c *noopRenderCtx) BindTexture(_, _ uint32)                                {}
func (c *noopRenderCtx) BufferData(_ uint32, _ int, _ unsafe.Pointer, _ uint32) {}
func (c *noopRenderCtx) BufferSubData(_ uint32, _ int, _ int, _ unsafe.Pointer) {}
func (c *noopRenderCtx) VertexAttribPointer(_ uint32, _ int32, _ uint32, _ bool, _ int32, _ unsafe.Pointer) {
}
func (c *noopRenderCtx) VertexAttribDivisor(_, _ uint32)      {}
func (c *noopRenderCtx) EnableVertexAttribArray(_ uint32)     {}
func (c *noopRenderCtx) TexParameteri(_, _ uint32, _ int32)   {}
func (c *noopRenderCtx) TexParameterf(_, _ uint32, _ float32) {}
func (c *noopRenderCtx) GetFloatv(_ uint32, data *float32)    { *data = 1.0 }
func (c *noopRenderCtx) TexImage3D(_ uint32, _ int32, _ int32, _, _, _, _ int32, _, _ uint32, _ unsafe.Pointer) {
}
func (c *noopRenderCtx) TexSubImage3D(_ uint32, _ int32, _, _, _, _, _, _ int32, _, _ uint32, _ unsafe.Pointer) {
}
func (c *noopRenderCtx) TexImage2D(_ uint32, _ int32, _ int32, _, _, _ int32, _, _ uint32, _ unsafe.Pointer) {
}
func (c *noopRenderCtx) TexSubImage2D(_ uint32, _ int32, _, _, _, _ int32, _, _ uint32, _ unsafe.Pointer) {
}
func (c *noopRenderCtx) DeleteTextures(_ int32, _ *uint32) {}
func (c *noopRenderCtx) GenerateMipmap(_ uint32)           {}

type multiTexHandler struct {
	frames []*textures.Texture
}

func (m *multiTexHandler) GetNames() []string {
	names := make([]string, len(m.frames))
	for i, f := range m.frames {
		names[i] = f.GetName()
	}
	return names
}

func (m *multiTexHandler) GetFrames(ids []string) []*textures.Texture {
	var out []*textures.Texture
	for _, id := range ids {
		for _, f := range m.frames {
			if f.GetName() == id {
				out = append(out, f)
				break
			}
		}
	}
	return out
}

func TestBuilderVolumeStaticCacheAnimInvalidationAndThingCache(t *testing.T) {
	rigid_body.SetFps(120)
	textures.SetFps(120)
	defer rigid_body.SetFps(120)

	t1 := textures.NewTexture("WALL_A", 1, 64, 64, false)
	t2 := textures.NewTexture("WALL_B", 1, 64, 64, false)
	texH := &multiTexHandler{frames: []*textures.Texture{t1, t2}}

	cal := config.NewConfigCalibration()
	cal.AspectRatio = 1.1
	scale := geometry.XYZ{X: 1, Y: 1, Z: 1}

	matStatic := config.NewConfigMaterial("static_wall", []string{"WALL_A"}, config.MaterialKindSolid)
	matAnim := config.NewConfigMaterial("anim_wall", []string{"WALL_A", "WALL_B"}, config.MaterialKindSolid)
	matItem := config.NewConfigMaterial("item_mat", []string{"WALL_A"}, config.MaterialKindSolid)

	sec := config.NewConfigSector("0", 1.0, config.LightKindAmbient, 100)
	sec.FloorY = 0
	sec.CeilY = 200
	sec.Floor = matStatic
	sec.Ceil = matStatic
	sec.Segments = []*config.Segment{
		func() *config.Segment {
			s := config.NewConfigSegment("0", config.SegmentWall, geometry.XY{X: -400, Y: -400}, geometry.XY{X: 400, Y: -400})
			s.Middle = matAnim
			return s
		}(),
		func() *config.Segment {
			s := config.NewConfigSegment("0", config.SegmentWall, geometry.XY{X: 400, Y: -400}, geometry.XY{X: 400, Y: 400})
			s.Middle = matStatic
			return s
		}(),
		func() *config.Segment {
			s := config.NewConfigSegment("0", config.SegmentWall, geometry.XY{X: 400, Y: 400}, geometry.XY{X: -400, Y: 400})
			s.Middle = matStatic
			return s
		}(),
		func() *config.Segment {
			s := config.NewConfigSegment("0", config.SegmentWall, geometry.XY{X: -400, Y: 400}, geometry.XY{X: -400, Y: -400})
			s.Middle = matStatic
			return s
		}(),
	}

	p := config.NewConfigPlayer(geometry.XYZ{X: 0, Y: 0, Z: 20}, 0.0, 40, 1800, 16.0, 60.0)
	_ = common.NewPlayer(p)
	p.GForce = 9.8 * 50

	itemCfg := config.NewConfigItem("medikit", geometry.XYZ{X: 120, Y: 0, Z: 20}, 0, 10, 0)
	itemCfg.Radius = 16
	itemCfg.Height = 16
	itemCfg.GForce = 9.8 * 50
	itemCfg.Sprite = config.NewConfigSprite(matItem)

	ammoCfg := config.NewConfigItem("ammo", geometry.XYZ{X: 140, Y: 30, Z: 20}, 0, 10, 0)
	ammoCfg.Radius = 16
	ammoCfg.Height = 16
	ammoCfg.GForce = 9.8 * 50
	ammoCfg.Sprite = config.NewConfigSprite(matItem)

	root := config.NewConfigRoot(cal, []*config.Sector{sec}, p, []*config.Thing{itemCfg, ammoCfg}, scale, texH)
	en := engine.NewEngine()
	if err := en.Setup(root); err != nil {
		t.Fatalf("engine setup failed: %v", err)
	}

	pl := en.GetPlayer()
	vi := engine.NewViewMatrix()
	vi.Update(pl)
	mCal := en.GetCalibration()

	rCtx := &noopRenderCtx{}
	glTex := gpu.NewTextures(rCtx)
	if err := glTex.Setup(en.GetTextures()); err != nil {
		t.Fatalf("textures setup failed: %v", err)
	}
	bv := NewBuilderVolume(rCtx, glTex, mCal, pl)

	// Frame 1: builds resident static meshes (zero streaming static prefix) and visible things.
	bv.Compute(640, 480, vi, en)
	sBytes1, iBytes1, sGen1, fGen1 := bv.GetUploadHints()
	if sGen1 == 0 || fGen1 == 0 || sBytes1 != 0 || iBytes1 != 0 || bv.GetResidentVolumeCount() <= 0 {
		t.Fatalf("expected resident static volumes without CPU prefix copy, got sBytes=%d iBytes=%d sGen=%d fGen=%d resVol=%d", sBytes1, iBytes1, sGen1, fGen1, bv.GetResidentVolumeCount())
	}

	var medikitThing, ammoThing ithing.IThing
	en.GetThings().QueryFrustum(vi.GetFrontFrustum(bv.frustumFront.Get()), func(obj aabb.IObject) bool {
		th := obj.(ithing.IThing)
		if th.GetId() == "medikit" {
			medikitThing = th
		} else if th.GetId() == "ammo" {
			ammoThing = th
		}
		return false
	})
	if medikitThing == nil || ammoThing == nil {
		t.Fatalf("expected medikit and ammo in front frustum")
	}
	medCacheGen1 := bv.thingsRender.Retrieve(medikitThing.GetEntity()).GetGen()
	ammoCacheGen1 := bv.thingsRender.Retrieve(ammoThing.GetEntity()).GetGen()

	// Frame 2: nothing moved and animated texture tick interval (32) not crossed -> full reuse.
	bv.Compute(640, 480, vi, en)
	_, _, sGen2, fGen2 := bv.GetUploadHints()
	if sGen2 != sGen1 || fGen2 != fGen1 {
		t.Fatalf("expected staticGen and frameGen unchanged on idle frame, got sGen %d->%d fGen %d->%d", sGen1, sGen2, fGen1, fGen2)
	}

	// Frame 3: only medikit moves; static volumes & stationary ammo cache must remain untouched.
	medikitThing.GetEntity().MoveCenterTo(130, 0, 20)
	bv.Compute(640, 480, vi, en)
	_, _, sGen3, fGen3 := bv.GetUploadHints()
	if sGen3 != sGen1 {
		t.Fatalf("expected staticGen preserved while frustum volumes unchanged, got %d != %d", sGen3, sGen1)
	}
	if fGen3 <= fGen2 {
		t.Fatalf("expected frameGen to advance when visible item moved, got %d -> %d", fGen2, fGen3)
	}
	if genNow := bv.thingsRender.Retrieve(medikitThing.GetEntity()).GetGen(); genNow <= medCacheGen1 {
		t.Fatalf("expected moved item cache gen to advance, got %d -> %d", medCacheGen1, genNow)
	}
	if ammoNow := bv.thingsRender.Retrieve(ammoThing.GetEntity()).GetGen(); ammoNow != ammoCacheGen1 {
		t.Fatalf("expected stationary item vertex cache gen=%d reused, got %d", ammoCacheGen1, ammoNow)
	}

	// Frame 4: mark sector dirty -> must invalidate static volume cache and increment staticGen.
	en.GetVolumes().GetVolumes().Get(0).SetDirty(true)
	bv.Compute(640, 480, vi, en)
	_, _, sGen4, _ := bv.GetUploadHints()
	if sGen4 <= sGen3 {
		t.Fatalf("expected staticGen increment when dirty volume changes, got %d -> %d", sGen3, sGen4)
	}

	// Frame 5: animated material frame change when tick crosses tickInterval (32) must rebuild staticGen.
	var sGen5 uint64
	for step := 0; step <= 32; step++ {
		textures.Tick()
		bv.Compute(640, 480, vi, en)
		_, _, sGen5, _ = bv.GetUploadHints()
		if sGen5 > sGen4 {
			break
		}
	}
	if sGen5 <= sGen4 {
		t.Fatalf("expected staticGen increment when animated wall texture frame switches, got %d -> %d", sGen4, sGen5)
	}
}

func TestVisibleVolumesSortFrontToBack(t *testing.T) {
	vs := NewVisibleVols(4)
	vs.Reset(4, 0, 0, 0)

	makeVolAt := func(x, y, z float64) *volumes.Volume {
		v := volumes.NewVolumeConcrete(0, "v", "v")
		f := volumes.NewFace([3]geometry.XYZ{
			{X: x - 1, Y: y - 1, Z: z - 1},
			{X: x + 1, Y: y - 1, Z: z - 1},
			{X: x, Y: y + 1, Z: z + 1},
		}, "f", nil)
		v.AddFace(f)
		v.Rebuild()
		return v
	}

	vFar := makeVolAt(300, 0, 0)
	vNear := makeVolAt(10, 0, 0)
	vMid := makeVolAt(100, 0, 0)

	vs.Add(vFar)
	vs.Add(vNear)
	vs.Add(vMid)
	vs.Sort()

	if vs.At(0) != vNear || vs.At(1) != vMid || vs.At(2) != vFar {
		t.Fatalf("unexpected sort order: got [%v, %v, %v], want [near, mid, far]", vs.At(0), vs.At(1), vs.At(2))
	}

	_ = model.NewCalibration
	_ = things.NewUI
	_ = metrics.NewFrustum
}

func TestThingsRenderSlotRecyclingAndABASafety(t *testing.T) {
	tr := NewThingsRenderWithCap(16)
	e1 := rigid_body.NewEntity(10, 0, 0.2, int(rigid_body.CombineAverage), int(rigid_body.CombineAverage), 0)
	e2 := rigid_body.NewEntity(10, 0, 0.2, int(rigid_body.CombineAverage), int(rigid_body.CombineAverage), 0)

	tr.Prepare()
	c1 := tr.Retrieve(e1)
	c1.Assign(nil, nil, 1, 0, 1, 0, 0, 0, 0, 1, 2, 3)
	tr.AddRef(c1)
	if !tr.Commit() {
		t.Fatalf("expected initial frame Commit to report change")
	}
	slot1 := e1.GetRenderMarker()
	if slot1 != 0 {
		t.Fatalf("expected first entity at slot 0, got %d", slot1)
	}

	// Step > 512 empty frames without e1 so slot 0 is pruned cleanly on a 256-frame boundary.
	for i := 0; i < 520; i++ {
		tr.Prepare()
		tr.Commit()
	}
	if tr.freeLen != 1 || tr.freeSlots[0] != slot1 {
		t.Fatalf("expected slot %d recycled into freeSlots, got freeLen=%d slots=%v", slot1, tr.freeLen, tr.freeSlots[:tr.freeLen])
	}

	// New entity e2 reuses slot 0 without allocating.
	tr.Prepare()
	c2 := tr.Retrieve(e2)
	c2.Assign(nil, nil, 2, 0, 2, 0, 0, 0, 0, 9, 9, 9)
	tr.AddRef(c2)
	tr.Commit()
	if e2.GetRenderMarker() != slot1 {
		t.Fatalf("expected e2 to reuse slot %d, got %d", slot1, e2.GetRenderMarker())
	}
	if tr.freeLen != 0 {
		t.Fatalf("expected freeSlots empty after reuse, got %d", tr.freeLen)
	}

	// Stale returning e1 still holds old renderMarker=0, must detect entID mismatch and allocate slot 1 without disturbing e2.
	tr.Prepare()
	c2Again := tr.Retrieve(e2)
	c1Returned := tr.Retrieve(e1)
	if c1Returned == c2Again {
		t.Fatalf("returning e1 collided with active e2 cache slot %d", slot1)
	}
	if e1.GetRenderMarker() != 1 || c1Returned.valid {
		t.Fatalf("expected returning e1 to get fresh uninitialized slot 1, got slot=%d valid=%v", e1.GetRenderMarker(), c1Returned.valid)
	}
	if c2Again.oX != 9 {
		t.Fatalf("e2 cached state was corrupted when e1 returned")
	}
	tr.AddRef(c2Again)
	tr.AddRef(c1Returned)
	tr.Commit()
}

func TestThingsRenderZeroAllocsHotPath(t *testing.T) {
	tr := NewThingsRenderWithCap(64)
	ents := make([]*rigid_body.Entity, 16)
	for i := range ents {
		ents[i] = rigid_body.NewEntity(10, 0, 0.2, int(rigid_body.CombineAverage), int(rigid_body.CombineAverage), 0)
	}

	// Warm up slots once (first assignment only).
	tr.Prepare()
	for _, e := range ents {
		c := tr.Retrieve(e)
		c.Assign(nil, nil, 1, 0, 1, 0, 0, 0, 0, 1, 2, 3)
		tr.AddRef(c)
	}
	tr.Commit()

	toggle := false
	allocs := testing.AllocsPerRun(200, func() {
		toggle = !toggle
		count := len(ents)
		if toggle {
			count -= 2
		}
		tr.Prepare()
		for i := 0; i < count; i++ {
			c := tr.Retrieve(ents[i])
			if i == 0 {
				c.Assign(nil, nil, 1, 0, 1, 0, 0, 0, 0, float32(count), 2, 3)
			}
			tr.AddRef(c)
		}
		tr.Commit()
	})
	if allocs != 0 {
		t.Fatalf("expected 0 heap allocations in ThingsRender hot path, got %.2f", allocs)
	}
}

func makeRigidCrateModel(mat *config.Material) *config.Model3DEntry {
	buildTri := func(a, b, c geometry.XYZ) config.Model3DEntryTriangle {
		tri := config.NewModel3DEntryTriangle(mat)
		tri.Vertices[0] = config.Model3DEntryVertex{Pos: a, U: 0, V: 0}
		tri.Vertices[1] = config.Model3DEntryVertex{Pos: b, U: 1, V: 0}
		tri.Vertices[2] = config.Model3DEntryVertex{Pos: c, U: 1, V: 1}
		return tri
	}
	v000 := geometry.XYZ{X: -12, Y: -12, Z: 0}
	v100 := geometry.XYZ{X: 12, Y: -12, Z: 0}
	v110 := geometry.XYZ{X: 12, Y: 12, Z: 0}
	v010 := geometry.XYZ{X: -12, Y: 12, Z: 0}
	v001 := geometry.XYZ{X: -12, Y: -12, Z: 24}
	v101 := geometry.XYZ{X: 12, Y: -12, Z: 24}
	v111 := geometry.XYZ{X: 12, Y: 12, Z: 24}
	v011 := geometry.XYZ{X: -12, Y: 12, Z: 24}

	tris := []config.Model3DEntryTriangle{
		// Top & bottom
		buildTri(v001, v101, v111), buildTri(v001, v111, v011),
		buildTri(v000, v010, v110), buildTri(v000, v110, v100),
		// Front & back
		buildTri(v000, v100, v101), buildTri(v000, v101, v001),
		buildTri(v110, v010, v011), buildTri(v110, v011, v111),
		// Left & right
		buildTri(v010, v000, v001), buildTri(v010, v001, v011),
		buildTri(v100, v110, v111), buildTri(v100, v111, v101),
	}
	entry := config.NewModel3DEntry(1, []string{"idle"}, nil)
	entry.Frames[0] = config.NewModel3DEntryFrame(tris)
	return entry
}

func TestBuilderVolumeRigidInstancingAndTransparentSorting(t *testing.T) {
	rigid_body.SetFps(120)
	textures.SetFps(120)
	defer rigid_body.SetFps(120)

	t1 := textures.NewTexture("WALL_A", 1, 64, 64, false)
	texH := &multiTexHandler{frames: []*textures.Texture{t1}}

	cal := config.NewConfigCalibration()
	cal.AspectRatio = 1.1
	scale := geometry.XYZ{X: 1, Y: 1, Z: 1}

	matSolid := config.NewConfigMaterial("solid_wall", []string{"WALL_A"}, config.MaterialKindSolid)
	matGlass := config.NewConfigMaterial("glass_wall", []string{"WALL_A"}, config.MaterialKindTranslucent)
	matCrate := config.NewConfigMaterial("crate_mat", []string{"WALL_A"}, config.MaterialKindSolid)

	sec := config.NewConfigSector("0", 1.0, config.LightKindAmbient, 100)
	sec.FloorY = 0
	sec.CeilY = 200
	sec.Floor = matSolid
	sec.Ceil = matSolid
	sec.Segments = []*config.Segment{
		func() *config.Segment {
			s := config.NewConfigSegment("0", config.SegmentWall, geometry.XY{X: -400, Y: -400}, geometry.XY{X: 400, Y: -400})
			s.Middle = matSolid
			return s
		}(),
		func() *config.Segment {
			s := config.NewConfigSegment("0", config.SegmentWall, geometry.XY{X: 400, Y: -400}, geometry.XY{X: 400, Y: 400})
			s.Middle = matGlass
			return s
		}(),
		func() *config.Segment {
			s := config.NewConfigSegment("0", config.SegmentWall, geometry.XY{X: 400, Y: 400}, geometry.XY{X: -400, Y: 400})
			s.Middle = matSolid
			return s
		}(),
		func() *config.Segment {
			s := config.NewConfigSegment("0", config.SegmentWall, geometry.XY{X: -400, Y: 400}, geometry.XY{X: -400, Y: -400})
			s.Middle = matSolid
			return s
		}(),
	}

	p := config.NewConfigPlayer(geometry.XYZ{X: 0, Y: 0, Z: 20}, 0.0, 40, 1800, 16.0, 60.0)
	_ = common.NewPlayer(p)
	p.GForce = 9.8 * 50

	// Measure player-only HUD/streaming vertex count first so we prove walls and rigid props add 0 streaming verts.
	baseRoot := config.NewConfigRoot(cal, []*config.Sector{sec}, p, nil, scale, texH)
	baseEn := engine.NewEngine()
	if err := baseEn.Setup(baseRoot); err != nil {
		t.Fatalf("base engine setup failed: %v", err)
	}
	basePl := baseEn.GetPlayer()
	baseVi := engine.NewViewMatrix()
	baseVi.Update(basePl)
	baseCtx := &noopRenderCtx{}
	baseTex := gpu.NewTextures(baseCtx)
	if err := baseTex.Setup(baseEn.GetTextures()); err != nil {
		t.Fatalf("base textures setup failed: %v", err)
	}
	baseBv := NewBuilderVolume(baseCtx, baseTex, baseEn.GetCalibration(), basePl)
	baseBv.Compute(640, 480, baseVi, baseEn)
	baseStreamVerts := baseBv.GetStreamingVertexCount()

	p2 := config.NewConfigPlayer(geometry.XYZ{X: 0, Y: 0, Z: 20}, 0.0, 40, 1800, 16.0, 60.0)
	_ = common.NewPlayer(p2)
	p2.GForce = 9.8 * 50

	crateA := config.NewConfigItem("crate_a", geometry.XYZ{X: 160, Y: -24, Z: 0}, 0, 10, 0)
	crateA.Radius = 16
	crateA.Height = 24
	crateA.GForce = 9.8 * 50
	crateA.Element3D = makeRigidCrateModel(matCrate)

	crateB := config.NewConfigItem("crate_b", geometry.XYZ{X: 220, Y: 24, Z: 0}, 0.5, 10, 0)
	crateB.Radius = 16
	crateB.Height = 24
	crateB.GForce = 9.8 * 50
	crateB.Element3D = makeRigidCrateModel(matCrate)

	crateC := config.NewConfigItem("crate_c", geometry.XYZ{X: 260, Y: -10, Z: 0}, 1.0, 10, 0)
	crateC.Radius = 16
	crateC.Height = 24
	crateC.GForce = 9.8 * 50
	crateC.Element3D = makeRigidCrateModel(matCrate)

	root := config.NewConfigRoot(cal, []*config.Sector{sec}, p2, []*config.Thing{crateA, crateB, crateC}, scale, texH)
	en := engine.NewEngine()
	if err := en.Setup(root); err != nil {
		t.Fatalf("engine setup failed: %v", err)
	}

	pl := en.GetPlayer()
	vi := engine.NewViewMatrix()
	vi.Update(pl)
	mCal := en.GetCalibration()

	rCtx := &noopRenderCtx{}
	glTex := gpu.NewTextures(rCtx)
	if err := glTex.Setup(en.GetTextures()); err != nil {
		t.Fatalf("textures setup failed: %v", err)
	}
	bv := NewBuilderVolume(rCtx, glTex, mCal, pl)

	// Frame 1: static world -> resident, three rigid crates -> single instanced batch (count=3), 0 added streaming verts.
	bv.Compute(640, 480, vi, en)
	if bv.GetResidentVolumeCount() == 0 {
		t.Fatalf("expected resident static volume draws > 0")
	}
	if got := bv.GetStreamingVertexCount(); got != baseStreamVerts {
		t.Fatalf("expected streaming vertex count unchanged (%d) when adding resident walls + 3 rigid props, got %d", baseStreamVerts, got)
	}
	instItems := bv.GetOpaqueInstancedItems()
	if len(instItems) != 1 || instItems[0].InstanceCount != 3 {
		t.Fatalf("expected 3 identical crates merged into 1 instanced batch with InstanceCount=3, got batches=%d (items=%+v)", len(instItems), instItems)
	}
	if !bv.GetDrawCommandsTransparent().HasCommands() {
		t.Fatalf("expected translucent wall routed into transparent draw commands")
	}

	_, _, sGen1, fGen1 := bv.GetUploadHints()

	// Find crate_a and move/rotate it: must keep sGen untouched and copy 0 vertices into streaming buffer.
	var crateThingA ithing.IThing
	en.GetThings().QueryFrustum(vi.GetFrontFrustum(bv.frustumFront.Get()), func(obj aabb.IObject) bool {
		th := obj.(ithing.IThing)
		if th.GetId() == "crate_a" {
			crateThingA = th
			return true
		}
		return false
	})
	if crateThingA == nil {
		t.Fatalf("failed to locate crate_a in frustum")
	}
	crateThingA.GetEntity().MoveCenterTo(180, -10, 0)
	crateThingA.GetEntity().SetRotationQuaternion(0, 0, 0.3826834, 0.9238795)

	bv.Compute(640, 480, vi, en)
	_, _, sGen2, fGen2 := bv.GetUploadHints()
	if sGen2 != sGen1 {
		t.Fatalf("expected staticGen unchanged when rigid prop moves/rotates, got %d -> %d", sGen1, sGen2)
	}
	if fGen2 <= fGen1 {
		t.Fatalf("expected frameGen to advance when rigid prop moves/rotates, got %d -> %d", fGen1, fGen2)
	}
	if got := bv.GetStreamingVertexCount(); got != baseStreamVerts {
		t.Fatalf("expected streaming vertex count unchanged (%d) after moving/rotating rigid prop, got %d", baseStreamVerts, got)
	}
}

func TestBuilderVolumeStaticAtlasCoalescesVolumesAndDetachesDirty(t *testing.T) {
	tA := textures.NewTexture("ATLAS_W", 1, 64, 64, false)
	texH := &multiTexHandler{frames: []*textures.Texture{tA}}

	cal := config.NewConfigCalibration()
	cal.AspectRatio = 1.1
	scale := geometry.XYZ{X: 1, Y: 1, Z: 1}
	matWall := config.NewConfigMaterial("atlas_wall", []string{"ATLAS_W"}, config.MaterialKindSolid)

	sec := config.NewConfigSector("0", 1.0, config.LightKindAmbient, 100)
	sec.FloorY = 0
	sec.CeilY = 200
	sec.Floor = matWall
	sec.Ceil = matWall
	sec.Segments = []*config.Segment{
		func() *config.Segment {
			s := config.NewConfigSegment("0", config.SegmentWall, geometry.XY{X: -400, Y: -400}, geometry.XY{X: 400, Y: -400})
			s.Middle = matWall
			return s
		}(),
		func() *config.Segment {
			s := config.NewConfigSegment("0", config.SegmentWall, geometry.XY{X: 400, Y: -400}, geometry.XY{X: 400, Y: 400})
			s.Middle = matWall
			return s
		}(),
		func() *config.Segment {
			s := config.NewConfigSegment("0", config.SegmentWall, geometry.XY{X: 400, Y: 400}, geometry.XY{X: -400, Y: 400})
			s.Middle = matWall
			return s
		}(),
		func() *config.Segment {
			s := config.NewConfigSegment("0", config.SegmentWall, geometry.XY{X: -400, Y: 400}, geometry.XY{X: -400, Y: -400})
			s.Middle = matWall
			return s
		}(),
	}

	p := config.NewConfigPlayer(geometry.XYZ{X: 0, Y: 0, Z: 20}, 0.0, 40, 1800, 16.0, 60.0)
	_ = common.NewPlayer(p)
	p.GForce = 9.8 * 50

	root := config.NewConfigRoot(cal, []*config.Sector{sec}, p, nil, scale, texH)
	en := engine.NewEngine()
	if err := en.Setup(root); err != nil {
		t.Fatalf("engine setup failed: %v", err)
	}

	pl := en.GetPlayer()
	vi := engine.NewViewMatrix()
	vi.Update(pl)
	mCal := en.GetCalibration()

	rCtx := &noopRenderCtx{}
	glTex := gpu.NewTextures(rCtx)
	if err := glTex.Setup(en.GetTextures()); err != nil {
		t.Fatalf("textures setup failed: %v", err)
	}
	bv := NewBuilderVolume(rCtx, glTex, mCal, pl)
	bv.Compute(640, 480, vi, en)

	if bv.staticAtlas == nil || bv.staticAtlas.VAO == 0 {
		t.Fatalf("expected built staticAtlas with valid VAO, got %+v", bv.staticAtlas)
	}
	if bv.staticOpaque.Len == 0 {
		t.Fatalf("expected non-zero staticOpaque items")
	}
	for i := 0; i < bv.staticOpaque.Len; i++ {
		if bv.staticOpaque.Items[i].Mesh != bv.staticAtlas {
			t.Fatalf("expected static item %d to share staticAtlas VAO, got separate mesh", i)
		}
	}

	// Dirtying volume 0 must detach only that volume cleanly into volMeshes while advancing staticGen.
	_, _, sGen1, _ := bv.GetUploadHints()
	v0 := en.GetVolumes().GetVolumes().Get(0)
	v0.SetDirty(true)
	bv.Compute(640, 480, vi, en)
	_, _, sGen2, _ := bv.GetUploadHints()
	if sGen2 <= sGen1 {
		t.Fatalf("expected staticGen advance on dirty volume detach, got %d -> %d", sGen1, sGen2)
	}
	entry0 := bv.staticEntries[v0.GetFaces()]
	if entry0 == nil || !entry0.detached {
		t.Fatalf("expected dirty volume 0 marked detached from staticAtlas")
	}
	if len(bv.volMeshes) != 1 {
		t.Fatalf("expected exactly 1 detached fallback mesh in volMeshes, got %d", len(bv.volMeshes))
	}
}
