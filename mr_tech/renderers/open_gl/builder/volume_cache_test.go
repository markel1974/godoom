package builder

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
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/gpu/vramtex"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/metrics"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/shaders"
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
	glTex := vramtex.NewTextures(rCtx)
	if err := glTex.Setup(en.GetTextures()); err != nil {
		t.Fatalf("textures setup failed: %v", err)
	}
	bv := NewBuilder(rCtx, glTex, mCal, pl)

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
	medCacheGen1 := bv.bt.thingsRender.Retrieve(medikitThing.GetEntity()).GetGen()
	ammoCacheGen1 := bv.bt.thingsRender.Retrieve(ammoThing.GetEntity()).GetGen()

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
	if genNow := bv.bt.thingsRender.Retrieve(medikitThing.GetEntity()).GetGen(); genNow <= medCacheGen1 {
		t.Fatalf("expected moved item cache gen to advance, got %d -> %d", medCacheGen1, genNow)
	}
	if ammoNow := bv.bt.thingsRender.Retrieve(ammoThing.GetEntity()).GetGen(); ammoNow != ammoCacheGen1 {
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
		}, nil)
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
	baseTex := vramtex.NewTextures(baseCtx)
	if err := baseTex.Setup(baseEn.GetTextures()); err != nil {
		t.Fatalf("base textures setup failed: %v", err)
	}
	baseBv := NewBuilder(baseCtx, baseTex, baseEn.GetCalibration(), basePl)
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
	glTex := vramtex.NewTextures(rCtx)
	if err := glTex.Setup(en.GetTextures()); err != nil {
		t.Fatalf("textures setup failed: %v", err)
	}
	bv := NewBuilder(rCtx, glTex, mCal, pl)

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
	glTex := vramtex.NewTextures(rCtx)
	if err := glTex.Setup(en.GetTextures()); err != nil {
		t.Fatalf("textures setup failed: %v", err)
	}
	bv := NewBuilder(rCtx, glTex, mCal, pl)
	bv.Compute(640, 480, vi, en)

	if bv.bs.atlas == nil || bv.bs.atlas.VAO == 0 {
		t.Fatalf("expected built staticAtlas with valid VAO, got %+v", bv.bs.atlas)
	}
	if bv.bc.staticOpaque.Len == 0 {
		t.Fatalf("expected non-zero staticOpaque items")
	}
	for i := 0; i < bv.bc.staticOpaque.Len; i++ {
		if bv.bc.staticOpaque.Items[i].Mesh != bv.bs.atlas {
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
	entry0 := bv.bs.entries[v0.GetFaces()]
	if entry0 == nil || !entry0.detached {
		t.Fatalf("expected dirty volume 0 marked detached from staticAtlas")
	}
	if len(bv.bs.volMeshes) != 1 {
		t.Fatalf("expected exactly 1 detached fallback mesh in volMeshes, got %d", len(bv.bs.volMeshes))
	}
}

type shadowTestCtx struct {
	noopRenderCtx
	boundProgram uint32
	boundTexUnit uint32
	clearCalls   int
	drawCalls    int
	drawScissors [][4]int32
	curScissor   [4]int32
}

func (c *shadowTestCtx) resetCounts() {
	c.clearCalls = 0
	c.drawCalls = 0
	c.drawScissors = c.drawScissors[:0]
}

func (c *shadowTestCtx) ActiveTexture(t uint32) {
	if t >= api.TEXTURE0 {
		c.boundTexUnit = t - api.TEXTURE0
	}
}
func (c *shadowTestCtx) AttachShader(_, _ uint32)                                  {}
func (c *shadowTestCtx) BindBufferBase(_, _, _ uint32)                             {}
func (c *shadowTestCtx) BindFramebuffer(_, _ uint32)                               {}
func (c *shadowTestCtx) BindRenderbuffer(_, _ uint32)                              {}
func (c *shadowTestCtx) BlendEquation(_ uint32)                                    {}
func (c *shadowTestCtx) BlendFunc(_, _ uint32)                                     {}
func (c *shadowTestCtx) BlitFramebuffer(_, _, _, _, _, _, _, _ int32, _, _ uint32) {}
func (c *shadowTestCtx) CheckFramebufferStatus(_ uint32) uint32                    { return api.FRAMEBUFFER_COMPLETE }
func (c *shadowTestCtx) Clear(_ uint32)                                            { c.clearCalls++ }
func (c *shadowTestCtx) ClearColor(_, _, _, _ float32)                             {}
func (c *shadowTestCtx) ColorMask(_, _, _, _ bool)                                 {}
func (c *shadowTestCtx) CompileShader(_ uint32)                                    {}
func (c *shadowTestCtx) CreateProgram() uint32                                     { c.nextID++; return c.nextID }
func (c *shadowTestCtx) CreateShader(_ uint32) uint32                              { c.nextID++; return c.nextID }
func (c *shadowTestCtx) CullFace(_ uint32)                                         {}
func (c *shadowTestCtx) DeleteFramebuffers(_ int32, _ *uint32)                     {}
func (c *shadowTestCtx) DeleteRenderbuffers(_ int32, _ *uint32)                    {}
func (c *shadowTestCtx) DeleteShader(_ uint32)                                     {}
func (c *shadowTestCtx) DepthFunc(_ uint32)                                        {}
func (c *shadowTestCtx) DepthMask(_ bool)                                          {}
func (c *shadowTestCtx) Disable(_ uint32)                                          {}
func (c *shadowTestCtx) DrawArrays(_ uint32, _ int32, _ int32)                     {}
func (c *shadowTestCtx) DrawBuffer(_ uint32)                                       {}
func (c *shadowTestCtx) DrawBuffers(_ int32, _ *uint32)                            {}
func (c *shadowTestCtx) DrawElements(_ uint32, _ int32, _ uint32, _ unsafe.Pointer) {
	c.drawCalls++
	c.drawScissors = append(c.drawScissors, c.curScissor)
}
func (c *shadowTestCtx) DrawElementsInstanced(_ uint32, _ int32, _ uint32, _ unsafe.Pointer, _ int32) {
	c.drawCalls++
	c.drawScissors = append(c.drawScissors, c.curScissor)
}
func (c *shadowTestCtx) MultiDrawElements(_ uint32, _ *int32, _ uint32, _ *unsafe.Pointer, _ int32) {
	c.drawCalls++
	c.drawScissors = append(c.drawScissors, c.curScissor)
}
func (c *shadowTestCtx) Enable(_ uint32)                                                        {}
func (c *shadowTestCtx) FramebufferRenderbuffer(_, _, _, _ uint32)                              {}
func (c *shadowTestCtx) FramebufferTexture2D(_, _, _, _ uint32, _ int32)                        {}
func (c *shadowTestCtx) GenFramebuffers(n int32, fbos *uint32)                                  { c.allocIDs(n, fbos) }
func (c *shadowTestCtx) GenRenderbuffers(n int32, rbos *uint32)                                 { c.allocIDs(n, rbos) }
func (c *shadowTestCtx) GetProgramiv(_ uint32, _ uint32, p *int32)                              { *p = 1 }
func (c *shadowTestCtx) GetShaderiv(_ uint32, _ uint32, p *int32)                               { *p = 1 }
func (c *shadowTestCtx) GetUniformBlockIndex(_ uint32, _ *uint8) uint32                         { return 0 }
func (c *shadowTestCtx) GetUniformLocation(_ uint32, _ *uint8) int32                            { return 1 }
func (c *shadowTestCtx) LinkProgram(_ uint32)                                                   {}
func (c *shadowTestCtx) PolygonOffset(_, _ float32)                                             {}
func (c *shadowTestCtx) ReadBuffer(_ uint32)                                                    {}
func (c *shadowTestCtx) RenderbufferStorage(_, _ uint32, _, _ int32)                            {}
func (c *shadowTestCtx) RenderbufferStorageMultisample(_ uint32, _ int32, _ uint32, _, _ int32) {}
func (c *shadowTestCtx) Scissor(x, y, width, height int32) {
	c.curScissor = [4]int32{x, y, width, height}
}
func (c *shadowTestCtx) ShaderSource(_ uint32, _ int32, _ **uint8, _ *int32)   {}
func (c *shadowTestCtx) Str(_ string) *uint8                                   { var b byte; return &b }
func (c *shadowTestCtx) Strs(strs ...string) (**uint8, func())                 { var b *uint8; return &b, func() {} }
func (c *shadowTestCtx) TexParameterfv(_, _ uint32, _ *float32)                {}
func (c *shadowTestCtx) Uniform1f(_ int32, _ float32)                          {}
func (c *shadowTestCtx) Uniform1i(_ int32, _ int32)                            {}
func (c *shadowTestCtx) Uniform1iv(_ int32, _ int32, _ *int32)                 {}
func (c *shadowTestCtx) Uniform2f(_ int32, _, _ float32)                       {}
func (c *shadowTestCtx) Uniform3f(_ int32, _, _, _ float32)                    {}
func (c *shadowTestCtx) Uniform4f(_ int32, _, _, _, _ float32)                 {}
func (c *shadowTestCtx) UniformBlockBinding(_, _, _ uint32)                    {}
func (c *shadowTestCtx) UniformMatrix4fv(_ int32, _ int32, _ bool, _ *float32) {}
func (c *shadowTestCtx) UseProgram(prg uint32)                                 { c.boundProgram = prg }
func (c *shadowTestCtx) Viewport(_, _, _, _ int32)                             {}

func makeBoxSector(id string, cx, cy, half, floorZ, ceilZ float64, mat *config.Material) *config.Sector {
	sec := config.NewConfigSector(id, 1.0, config.LightKindAmbient, 100)
	sec.FloorY = floorZ
	sec.CeilY = ceilZ
	sec.Floor = mat
	sec.Ceil = mat
	sec.Segments = []*config.Segment{
		func() *config.Segment {
			s := config.NewConfigSegment(id, config.SegmentWall, geometry.XY{X: cx - half, Y: cy - half}, geometry.XY{X: cx + half, Y: cy - half})
			s.Middle = mat
			return s
		}(),
		func() *config.Segment {
			s := config.NewConfigSegment(id, config.SegmentWall, geometry.XY{X: cx + half, Y: cy - half}, geometry.XY{X: cx + half, Y: cy + half})
			s.Middle = mat
			return s
		}(),
		func() *config.Segment {
			s := config.NewConfigSegment(id, config.SegmentWall, geometry.XY{X: cx + half, Y: cy + half}, geometry.XY{X: cx - half, Y: cy + half})
			s.Middle = mat
			return s
		}(),
		func() *config.Segment {
			s := config.NewConfigSegment(id, config.SegmentWall, geometry.XY{X: cx - half, Y: cy + half}, geometry.XY{X: cx - half, Y: cy - half})
			s.Middle = mat
			return s
		}(),
	}
	return sec
}

func TestBuilderVolumeStreamingUsesChunkedStaticAtlas(t *testing.T) {
	tA := textures.NewTexture("STREAM_W", 1, 64, 64, false)
	texH := &multiTexHandler{frames: []*textures.Texture{tA}}
	cal := config.NewConfigCalibration()
	cal.AspectRatio = 1.1
	matWall := config.NewConfigMaterial("stream_wall", []string{"STREAM_W"}, config.MaterialKindSolid)

	// Place three sectors inside chunk (0,0) [0..512] and one distant sector in chunk (3,0) [1536..2048].
	secs := []*config.Sector{
		makeBoxSector("0", 120, 120, 50, 0, 160, matWall),
		makeBoxSector("1", 260, 120, 40, 0, 160, matWall),
		makeBoxSector("2", 180, 260, 40, 0, 160, matWall),
		makeBoxSector("3", 1700, 120, 40, 0, 160, matWall),
	}
	p := config.NewConfigPlayer(geometry.XYZ{X: 100, Y: 120, Z: 20}, 0.0, 40, 1800, 16.0, 60.0)
	_ = common.NewPlayer(p)
	p.GForce = 9.8 * 50

	root := config.NewConfigRoot(cal, secs, p, nil, geometry.XYZ{X: 1, Y: 1, Z: 1}, texH)
	en := engine.NewEngine()
	if err := en.Setup(root); err != nil {
		t.Fatalf("engine setup failed: %v", err)
	}

	vols := en.GetVolumes()
	vols.EnableStreaming(512.0, 1, 2, 3)
	vols.AutoSectorize(512.0)
	vols.ForceSyncStreaming(100, 120, 20)
	if !vols.HasStreamer() {
		t.Fatalf("expected HasStreamer() == true")
	}

	pl := en.GetPlayer()
	vi := engine.NewViewMatrix()
	vi.Update(pl)

	rCtx := &noopRenderCtx{}
	glTex := vramtex.NewTextures(rCtx)
	if err := glTex.Setup(en.GetTextures()); err != nil {
		t.Fatalf("textures setup failed: %v", err)
	}

	bv := NewBuilder(rCtx, glTex, en.GetCalibration(), pl)
	bv.Compute(640, 480, vi, en)

	if !bv.bs.atlasStreamMode {
		t.Fatalf("expected staticAtlasStreamMode == true when HasStreamer() active")
	}
	if len(bv.bs.streamChunks) == 0 {
		t.Fatalf("expected at least one streamed chunk static atlas built")
	}
	if len(bv.bs.volMeshes) != 0 {
		t.Fatalf("expected static streamed volumes to use chunk atlas instead of per-volume volMeshes, got volMeshes=%d", len(bv.bs.volMeshes))
	}
	if bv.bc.staticOpaque.Len < 3 {
		t.Fatalf("expected multiple visible static submeshes coalesced in chunk (0,0), got %d", bv.bc.staticOpaque.Len)
	}
	chunk0 := bv.bs.streamChunks[streamChunkKey{x: 0, y: 0}]
	if chunk0 == nil || chunk0.atlas == nil || chunk0.atlas.VAO == 0 {
		t.Fatalf("expected valid chunk (0,0) atlas")
	}
	for i := 0; i < bv.bc.staticOpaque.Len; i++ {
		if bv.bc.staticOpaque.Items[i].Mesh != chunk0.atlas {
			t.Fatalf("expected streamed static item %d to share chunk (0,0) atlas VAO", i)
		}
	}

	// Steady-state frame must reuse cached stream chunk with unchanged staticGen.
	_, _, sGen1, _ := bv.GetUploadHints()
	bv.Compute(640, 480, vi, en)
	_, _, sGen2, _ := bv.GetUploadHints()
	if sGen2 != sGen1 {
		t.Fatalf("expected unchanged staticGen on steady-state streamed frame, got %d -> %d", sGen1, sGen2)
	}

	// Activate distant sector 3 at chunk (3,0) while streaming: must build a second coalesced chunk atlas cleanly.
	vols.ForceSyncStreaming(1650, 120, 20)
	pl.GetEntity().MoveTo(1650, 120, 20)
	vi.Update(pl)
	bv.Compute(640, 480, vi, en)
	if chunk3 := bv.bs.streamChunks[streamChunkKey{x: 3, y: 0}]; chunk3 == nil || chunk3.atlas == nil || chunk3.atlas.VAO == 0 {
		t.Fatalf("expected dynamically streamed chunk (3,0) atlas after moving viewer")
	}

	// Move back to (0,0) and dirty one streamed volume: must detach only that volume without failing.
	vols.ForceSyncStreaming(100, 120, 20)
	pl.GetEntity().MoveTo(100, 120, 20)
	vi.Update(pl)
	bv.Compute(640, 480, vi, en)
	v0 := bv.bs.cachedRawVols[0]
	v0.SetDirty(true)
	bv.Compute(640, 480, vi, en)
	if len(bv.bs.volMeshes) != 1 {
		t.Fatalf("expected 1 detached fallback mesh after dirtying streamed volume, got %d", len(bv.bs.volMeshes))
	}
}

func TestPerLightShadowFrustumCullingIncludesOffCameraCastersAndCachesTiles(t *testing.T) {
	rigid_body.SetFps(120)
	textures.SetFps(120)
	defer rigid_body.SetFps(120)

	tA := textures.NewTexture("SHAD_W", 1, 8, 8, false)
	for x := range 8 {
		for y := range 8 {
			tA.SetDiffuse(x, y, 0x7f7f7fff)
		}
	}
	texH := &multiTexHandler{frames: []*textures.Texture{tA}}
	calCfg := config.NewConfigCalibration()
	calCfg.AspectRatio = 1.1
	calCfg.OrthoSize = 640

	matSolid := config.NewConfigMaterial("solid", []string{"SHAD_W"}, config.MaterialKindSolid)
	matCrate := config.NewConfigMaterial("crate", []string{"SHAD_W"}, config.MaterialKindSolid)

	// Separate spatially isolated sectors:
	// - center holds the player at (0,0)
	// - front sits in front of camera (+X)
	// - behind sits BEHIND camera (-X), inside Spotlight 0 and Room ortho box
	// - side sits far to the side (+Y = 1250), outside camera & Spotlight 0, inside Spotlight 1
	// - roof is high overhead near camera (Z = 880..920), kept in Room shadow via DEPTH_CLAMP
	secs := []*config.Sector{
		makeBoxSector("center", 0, 0, 60, 0, 160, matSolid),
		makeBoxSector("front", 220, 0, 50, 0, 160, matSolid),
		makeBoxSector("behind", -220, 0, 50, 0, 160, matSolid),
		makeBoxSector("side", 0, 1250, 50, 0, 160, matSolid),
		makeBoxSector("roof", 60, 0, 40, 880, 920, matSolid),
	}

	p := config.NewConfigPlayer(geometry.XYZ{X: 0, Y: 0, Z: 20}, 0.0, 40, 1800, 16.0, 60.0)
	_ = common.NewPlayer(p)
	p.GForce = 9.8 * 50

	crateFront := config.NewConfigItem("crate_front", geometry.XYZ{X: 210, Y: 0, Z: 0}, 0, 10, 0)
	crateFront.Radius, crateFront.Height, crateFront.GForce = 16, 24, 9.8*50
	crateFront.Element3D = makeRigidCrateModel(matCrate)

	crateBehind := config.NewConfigItem("crate_behind", geometry.XYZ{X: -210, Y: 0, Z: 0}, 0, 10, 0)
	crateBehind.Radius, crateBehind.Height, crateBehind.GForce = 16, 24, 9.8*50
	crateBehind.Element3D = makeRigidCrateModel(matCrate)

	crateSide := config.NewConfigItem("crate_side", geometry.XYZ{X: 0, Y: 1240, Z: 0}, 0, 10, 0)
	crateSide.Radius, crateSide.Height, crateSide.GForce = 16, 24, 9.8*50
	crateSide.Element3D = makeRigidCrateModel(matCrate)

	root := config.NewConfigRoot(calCfg, secs, p, []*config.Thing{crateFront, crateBehind, crateSide}, geometry.XYZ{X: 1, Y: 1, Z: 1}, texH)
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
	if err := d.Compile(&open_gl.Assets{}); err != nil {
		t.Fatalf("Depth.Compile: %v", err)
	}
	d.SetShadowsEnabled(true)

	mm := h.GetMapMetrics()
	mm.SetOrthoSize(640, 0.1, 4096)
	mm.SetCenter(0, 0, 0)
	mm.Update(vi, 1.0, 1.0)
	h.GetFlashlightMetrics().UpdateWithAim(mm.GetViewPtr(), 0, 0, 0, 0, 1)

	// Spotlight 0 (behind camera): GL pos (-80, 60, 0) pointing (-1, -0.35, 0) toward behind sector & crateBehind.
	// Spotlight 1 (far side): GL pos (0, 60, -1100) pointing (0, -0.35, -1) toward side sector & crateSide.
	var sl [metrics.ShadowLightsCount]*metrics.Light
	sl[0] = metrics.NewLight()
	sl[0].Assign(-80, 60, 0, 1, 1, 1, 1, 2.0, -1, -0.35, 0, 100, 0.9, 0.7, 1.0)
	sl[1] = metrics.NewLight()
	sl[1].Assign(0, 60, -1100, 1, 1, 1, 1, 2.0, 0, -0.35, -1, 100, 0.9, 0.7, 1.0)
	h.GetShadowLightsMetrics().Update(sl, 2)

	// Step 1: Camera compute should ONLY include crateFront, NOT crateBehind or crateSide.
	bv.Compute(640, 480, vi, en)
	camInst := bv.GetOpaqueInstancedItems()
	if len(camInst) != 1 || camInst[0].InstanceCount != 1 {
		t.Fatalf("expected camera frustum to see only crate_front (InstanceCount=1), got %+v", camInst)
	}

	// Step 2: Per-light shadow culling must find crateBehind in Spotlight 0 and crateSide in Spotlight 1!
	d.Prepare()
	bv.ComputeShadows(en, d)
	tiles := bv.GetShadowTiles()
	if !tiles[shaders.ShadowTileRoom].Active || tiles[shaders.ShadowTileRoom].Reuse {
		t.Fatalf("expected Room tile active & dirty on frame 1")
	}
	spot0 := tiles[shaders.ShadowTileLightBase]
	spot1 := tiles[shaders.ShadowTileLightBase+1]
	if !spot0.Active || !spot1.Active || spot0.RenderDC == nil || spot1.RenderDC == nil {
		t.Fatalf("expected Spotlight 0 and Spotlight 1 tiles active with compiled DrawCommandsRender")
	}
	if !spot0.RenderDC.HasSolid() || !spot1.RenderDC.HasSolid() {
		t.Fatalf("expected Spotlight 0 (off-camera behind) and Spotlight 1 (side) to retain solid casters")
	}
	if bv.bShadow.builders[shaders.ShadowTileLightBase].instItems.Len == 0 {
		t.Fatalf("expected Spotlight 0 to include off-camera crate_behind instanced caster")
	}
	if bv.bShadow.builders[shaders.ShadowTileLightBase+1].instItems.Len == 0 {
		t.Fatalf("expected Spotlight 1 to include off-camera crate_side instanced caster")
	}
	// Spotlight 1 only overlaps side sector (not front/behind/roof), so it must cull strictly fewer submeshes than Room.
	roomStaticCount := bv.bShadow.builders[shaders.ShadowTileRoom].staticItems.Len
	spot1StaticCount := bv.bShadow.builders[shaders.ShadowTileLightBase+1].staticItems.Len
	if spot1StaticCount == 0 || spot1StaticCount >= roomStaticCount {
		t.Fatalf("expected per-light culling on Spotlight 1 (%d) < Room (%d) and > 0", spot1StaticCount, roomStaticCount)
	}

	// Frame 1 (buf 1) and Frame 2 (buf 0) warm up both double-buffered atlas targets.
	rCtx.resetCounts()
	d.RenderWithTiles(bv.GetDrawCommandsOpaque(), tiles)
	if rCtx.clearCalls == 0 || rCtx.drawCalls == 0 {
		t.Fatalf("frame 1 expected clears/draws > 0")
	}
	bv.Compute(640, 480, vi, en)
	d.Prepare()
	bv.ComputeShadows(en, d)
	rCtx.resetCounts()
	d.RenderWithTiles(bv.GetDrawCommandsOpaque(), tiles)
	if rCtx.clearCalls == 0 || rCtx.drawCalls == 0 {
		t.Fatalf("frame 2 (cold second buffer) expected clears/draws > 0")
	}

	// Frame 3 & Frame 4: nothing moved -> all active tiles hit cache -> 0 clears, 0 draws, 0 allocs!
	for f := 3; f <= 4; f++ {
		bv.Compute(640, 480, vi, en)
		d.Prepare()
		bv.ComputeShadows(en, d)
		for _, slot := range []int{shaders.ShadowTileRoom, shaders.ShadowTileLightBase, shaders.ShadowTileLightBase + 1} {
			if !tiles[slot].Reuse {
				t.Fatalf("frame %d expected tile %d cache hit (Reuse=true)", f, slot)
			}
		}
		rCtx.resetCounts()
		d.RenderWithTiles(bv.GetDrawCommandsOpaque(), tiles)
		if rCtx.clearCalls != 0 || rCtx.drawCalls != 0 {
			t.Fatalf("frame %d expected 0 clears and 0 draws on steady-state cached atlas, got clears=%d draws=%d", f, rCtx.clearCalls, rCtx.drawCalls)
		}
	}

	allocs := testing.AllocsPerRun(20, func() {
		bv.ComputeShadows(en, d)
		d.RenderWithTiles(bv.GetDrawCommandsOpaque(), tiles)
	})
	if allocs > 0 {
		t.Fatalf("steady-state ComputeShadows + RenderWithTiles allocated %.2f objects/op, want 0", allocs)
	}

	// Move ONLY off-camera crate_behind -> Spotlight 0 (and Room containing it) becomes dirty,
	// while Spotlight 1 remains cached (Reuse == true)!
	en.GetThings().GetThing("crate_behind").GetEntity().MoveCenterTo(-205, 4, 0)
	bv.Compute(640, 480, vi, en)
	d.Prepare()
	bv.ComputeShadows(en, d)
	if tiles[shaders.ShadowTileLightBase].Reuse {
		t.Fatalf("expected Spotlight 0 dirty after off-camera crate_behind moved")
	}
	if !tiles[shaders.ShadowTileLightBase+1].Reuse {
		t.Fatalf("expected Spotlight 1 to stay cached when unrelated crate_behind moved")
	}
	rCtx.resetCounts()
	d.RenderWithTiles(bv.GetDrawCommandsOpaque(), tiles)
	for _, sc := range rCtx.drawScissors {
		if sc == ([4]int32{512, 0, 512, 512}) {
			t.Fatalf("Spotlight 1 tile (512,0,512,512) must not be redrawn when only crate_behind moved")
		}
	}
}

type shadowUploadTrackCtx struct {
	shadowTestCtx
	orphanCalls  int
	subDataCalls int
}

func (c *shadowUploadTrackCtx) resetUploadLogs() {
	c.orphanCalls = 0
	c.subDataCalls = 0
}

func (c *shadowUploadTrackCtx) BufferData(target uint32, size int, data unsafe.Pointer, usage uint32) {
	if target == api.ARRAY_BUFFER && data == nil {
		c.orphanCalls++
	}
	c.shadowTestCtx.BufferData(target, size, data, usage)
}

func (c *shadowUploadTrackCtx) BufferSubData(target uint32, offset int, size int, data unsafe.Pointer) {
	if target == api.ARRAY_BUFFER {
		c.subDataCalls++
	}
	c.shadowTestCtx.BufferSubData(target, offset, size, data)
}

func TestComputeShadowsSkipsRedundantInstUploadAndOrphansOnMove(t *testing.T) {
	rigid_body.SetFps(120)
	textures.SetFps(120)
	defer rigid_body.SetFps(120)

	tA := textures.NewTexture("SHAD_U", 1, 8, 8, false)
	for x := range 8 {
		for y := range 8 {
			tA.SetDiffuse(x, y, 0x7f7f7fff)
		}
	}
	texH := &multiTexHandler{frames: []*textures.Texture{tA}}
	calCfg := config.NewConfigCalibration()
	calCfg.AspectRatio = 1.1
	calCfg.OrthoSize = 640
	matSolid := config.NewConfigMaterial("solid", []string{"SHAD_U"}, config.MaterialKindSolid)
	matCrate := config.NewConfigMaterial("crate", []string{"SHAD_U"}, config.MaterialKindSolid)

	secs := []*config.Sector{makeBoxSector("room", 0, 0, 400, 0, 160, matSolid)}
	p := config.NewConfigPlayer(geometry.XYZ{X: 0, Y: 0, Z: 20}, 0.0, 40, 1800, 16.0, 60.0)
	_ = common.NewPlayer(p)
	p.GForce = 9.8 * 50

	crateModel := makeRigidCrateModel(matCrate)
	crateA := config.NewConfigItem("crate_a", geometry.XYZ{X: 160, Y: -20, Z: 0}, 0, 10, 0)
	crateA.Radius, crateA.Height, crateA.GForce = 16, 24, 9.8*50
	crateA.Element3D = crateModel
	crateB := config.NewConfigItem("crate_b", geometry.XYZ{X: 200, Y: 20, Z: 0}, 0.5, 10, 0)
	crateB.Radius, crateB.Height, crateB.GForce = 16, 24, 9.8*50
	crateB.Element3D = crateModel

	root := config.NewConfigRoot(calCfg, secs, p, []*config.Thing{crateA, crateB}, geometry.XYZ{X: 1, Y: 1, Z: 1}, texH)
	en := engine.NewEngine()
	if err := en.Setup(root); err != nil {
		t.Fatalf("engine setup failed: %v", err)
	}
	pl := en.GetPlayer()
	vi := engine.NewViewMatrix()
	vi.Update(pl)

	rCtx := &shadowUploadTrackCtx{}
	glTex := vramtex.NewTextures(rCtx)
	if err := glTex.Setup(en.GetTextures()); err != nil {
		t.Fatalf("textures setup failed: %v", err)
	}
	mCal := en.GetCalibration()
	mCal.EnableShadows = true
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
	if err := d.Compile(&open_gl.Assets{}); err != nil {
		t.Fatalf("Depth.Compile: %v", err)
	}
	d.SetShadowsEnabled(true)

	mm := h.GetMapMetrics()
	mm.SetOrthoSize(640, 0.1, 4096)
	mm.SetCenter(0, 0, 0)
	mm.Update(vi, 1.0, 1.0)
	fm := h.GetFlashlightMetrics()
	fm.UpdateWithAim(mm.GetViewPtr(), 0, 0, 0, 0, 1)

	bv.Compute(640, 480, vi, en)
	for range 2 {
		d.Prepare()
		bv.ComputeShadows(en, d)
		d.RenderWithTiles(bv.GetDrawCommandsOpaque(), bv.GetShadowTiles())
	}

	// Frame 3 transitions from (Room+Flash) to Flash-only layout (4->2 matrices);
	// Frame 4 (another Flash aim update with Room cached) must issue 0 BufferSubData calls and preserve VBO.
	fm.UpdateWithAim(mm.GetViewPtr(), 0.002, 0, 0.002, 0, 1)
	d.Prepare()
	bv.ComputeShadows(en, d)
	d.RenderWithTiles(bv.GetDrawCommandsOpaque(), bv.GetShadowTiles())
	flashOnlyVBO := bv.bShadow.instPool.ActiveVBO()

	rCtx.resetUploadLogs()
	fm.UpdateWithAim(mm.GetViewPtr(), 0.004, 0, 0.004, 0, 1)
	d.Prepare()
	bv.ComputeShadows(en, d)
	if bv.GetShadowTiles()[shaders.ShadowTileFlash].Reuse {
		t.Fatalf("expected Flash tile dirty after aim update")
	}
	if rCtx.subDataCalls != 0 {
		t.Fatalf("expected 0 BufferSubData calls when rigid shadow instances did not change, got %d", rCtx.subDataCalls)
	}
	if bv.bShadow.instPool.ActiveVBO() != flashOnlyVBO {
		t.Fatalf("expected shadowInstPool to preserve VBO %d across unchanged instance upload, got %d", flashOnlyVBO, bv.bShadow.instPool.ActiveVBO())
	}
	d.RenderWithTiles(bv.GetDrawCommandsOpaque(), bv.GetShadowTiles())

	// Move crateA: ComputeShadows must orphan, advance ring VBO, and upload once.
	en.GetThings().GetThing("crate_a").GetEntity().MoveCenterTo(165, -20, 0)
	bv.Compute(640, 480, vi, en)
	rCtx.resetUploadLogs()
	d.Prepare()
	bv.ComputeShadows(en, d)
	if rCtx.subDataCalls != 1 || rCtx.orphanCalls == 0 {
		t.Fatalf("expected 1 orphaned BufferSubData upload when crate_a moved, got orphan=%d subdata=%d", rCtx.orphanCalls, rCtx.subDataCalls)
	}
	if bv.bShadow.instPool.ActiveVBO() == flashOnlyVBO {
		t.Fatalf("expected shadowInstPool ring to advance VBO after instance mutation, stayed on %d", flashOnlyVBO)
	}
}

func TestBuilderVolumeNilGuardsAndHWOcclusion(t *testing.T) {
	rigid_body.SetFps(120)
	textures.SetFps(120)
	defer rigid_body.SetFps(120)

	t1 := textures.NewTexture("NIL_W", 1, 64, 64, false)
	texH := &multiTexHandler{frames: []*textures.Texture{t1}}
	cal := config.NewConfigCalibration()
	cal.AspectRatio = 1.1
	scale := geometry.XYZ{X: 1, Y: 1, Z: 1}
	matWall := config.NewConfigMaterial("nil_wall", []string{"NIL_W"}, config.MaterialKindSolid)

	sec := config.NewConfigSector("0", 1.0, config.LightKindAmbient, 100)
	sec.FloorY = 0
	sec.CeilY = 200
	sec.Floor = matWall
	sec.Ceil = matWall
	sec.Segments = []*config.Segment{
		func() *config.Segment {
			s := config.NewConfigSegment("0", config.SegmentWall, geometry.XY{X: -200, Y: -200}, geometry.XY{X: 200, Y: -200})
			s.Middle = matWall
			return s
		}(),
		func() *config.Segment {
			s := config.NewConfigSegment("0", config.SegmentWall, geometry.XY{X: 200, Y: -200}, geometry.XY{X: 200, Y: 200})
			s.Middle = matWall
			return s
		}(),
		func() *config.Segment {
			s := config.NewConfigSegment("0", config.SegmentWall, geometry.XY{X: 200, Y: 200}, geometry.XY{X: -200, Y: 200})
			s.Middle = matWall
			return s
		}(),
		func() *config.Segment {
			s := config.NewConfigSegment("0", config.SegmentWall, geometry.XY{X: -200, Y: 200}, geometry.XY{X: -200, Y: -200})
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
	vi := engine.NewViewMatrix()
	vi.Update(en.GetPlayer())

	rCtx := &noopRenderCtx{}
	glTex := vramtex.NewTextures(rCtx)
	if err := glTex.Setup(en.GetTextures()); err != nil {
		t.Fatalf("textures setup failed: %v", err)
	}

	bvNil := NewBuilder(rCtx, nil, nil, nil)
	if bvNil.GetHWOcclusion() == nil {
		t.Fatalf("expected NewBuilderVolume to initialize HWOcclusion even with nil dependencies")
	}
	bvNil.Compute(640, 480, nil, nil)
	bvNil.Compute(640, 480, vi, nil)
	bvNil.Compute(640, 480, nil, en)
	bvNil.Compute(640, 480, vi, en)
	bvNil.ComputeShadows(nil, nil)
	bvNil.ComputeShadows(en, nil)

	bv := NewBuilder(rCtx, glTex, en.GetCalibration(), nil)
	if bv.GetHWOcclusion() == nil {
		t.Fatalf("expected NewBuilderVolume to initialize HWOcclusion")
	}
	frustumFront := vi.GetFrontFrustum(bv.frustumFront.Get())
	bv.bs.Process(nil, frustumFront, 0, 0, 20)
	bv.bs.Process(en.GetVolumes(), nil, 0, 0, 20)
	bv.bl.Process(nil, frustumFront, 0, 0, 20, 100, 100, 100)
	bv.bl.Process(en.GetLights(), nil, 0, 0, 20, 100, 100, 100)
	bv.bt.Process(nil, frustumFront)
	bv.bt.Process(en.GetThings(), nil)
	bv.Compute(640, 480, vi, en)
	bv.ComputeShadows(en, nil)
}

func TestOffCameraResolveMeshDetachesStaticAtlasInvalidatesReuse(t *testing.T) {
	rigid_body.SetFps(120)
	textures.SetFps(120)
	defer rigid_body.SetFps(120)

	t1 := textures.NewTexture("DETACH_W", 1, 64, 64, false)
	texH := &multiTexHandler{frames: []*textures.Texture{t1}}
	cal := config.NewConfigCalibration()
	cal.AspectRatio = 1.1
	scale := geometry.XYZ{X: 1, Y: 1, Z: 1}
	matWall := config.NewConfigMaterial("detach_wall", []string{"DETACH_W"}, config.MaterialKindSolid)

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
	rCtx := &noopRenderCtx{}
	glTex := vramtex.NewTextures(rCtx)
	if err := glTex.Setup(en.GetTextures()); err != nil {
		t.Fatalf("textures setup failed: %v", err)
	}
	bv := NewBuilder(rCtx, glTex, en.GetCalibration(), pl)
	bv.Compute(640, 480, vi, en)

	if bv.bs.atlas == nil || bv.bc.staticOpaque.Len == 0 {
		t.Fatalf("expected non-empty staticAtlas draws on initial Compute")
	}
	_, _, sGen1, _ := bv.GetUploadHints()

	// Simulate an off-camera pass (e.g., ComputeShadows) detaching v0 and clearing its dirty flag
	// before the next Compute with an unchanged camera frustum.
	v0 := en.GetVolumes().GetVolumes().Get(0)
	v0.SetDirty(true)
	detachedMesh, entry, _ := bv.bs.ResolveMesh(v0, en.GetVolumes())
	if detachedMesh == nil || detachedMesh == bv.bs.atlas {
		t.Fatalf("expected ResolveMesh to return a detached fallback mesh, got %p (staticAtlas=%p)", detachedMesh, bv.bs.atlas)
	}
	if entry != nil {
		t.Fatalf("expected detached ResolveMesh call to return nil atlas entry")
	}
	if v0.IsDirty() {
		t.Fatalf("expected ResolveMesh to clear volume dirty flag")
	}

	bv.Compute(640, 480, vi, en)
	_, _, sGen2, _ := bv.GetUploadHints()
	if sGen2 <= sGen1 {
		t.Fatalf("expected staticGen to advance when detached mesh replaces cached atlas draws (%d -> %d)", sGen1, sGen2)
	}

	foundDetached := false
	for i := 0; i < bv.bc.staticOpaque.Len; i++ {
		if bv.bc.staticOpaque.Items[i].Mesh == detachedMesh {
			foundDetached = true
			break
		}
	}
	if !foundDetached {
		t.Fatalf("expected staticOpaque to emit detached mesh instead of reusing stale staticAtlas")
	}
}
