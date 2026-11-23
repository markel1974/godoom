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
func (c *noopRenderCtx) GenTextures(n int32, t *uint32)                         { c.allocIDs(n, t) }
func (c *noopRenderCtx) BindVertexArray(_ uint32)                               {}
func (c *noopRenderCtx) BindBuffer(_, _ uint32)                                 {}
func (c *noopRenderCtx) BindTexture(_, _ uint32)                                {}
func (c *noopRenderCtx) BufferData(_ uint32, _ int, _ unsafe.Pointer, _ uint32) {}
func (c *noopRenderCtx) VertexAttribPointer(_ uint32, _ int32, _ uint32, _ bool, _ int32, _ unsafe.Pointer) {
}
func (c *noopRenderCtx) EnableVertexAttribArray(_ uint32)     {}
func (c *noopRenderCtx) TexParameteri(_, _ uint32, _ int32)   {}
func (c *noopRenderCtx) TexParameterf(_, _ uint32, _ float32) {}
func (c *noopRenderCtx) GetFloatv(_ uint32, data *float32)    { *data = 1.0 }
func (c *noopRenderCtx) TexImage3D(_ uint32, _ int32, _ int32, _, _, _, _ int32, _, _ uint32, _ unsafe.Pointer) {
}
func (c *noopRenderCtx) TexSubImage3D(_ uint32, _ int32, _, _, _, _, _, _ int32, _, _ uint32, _ unsafe.Pointer) {
}
func (c *noopRenderCtx) GenerateMipmap(_ uint32) {}

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
	glTex := NewTextures(rCtx)
	if err := glTex.Setup(en.GetTextures()); err != nil {
		t.Fatalf("textures setup failed: %v", err)
	}
	bv := NewBuilderVolume(rCtx, glTex, mCal, pl)

	// Frame 1: builds static prefix and visible things.
	bv.Compute(640, 480, vi, en)
	sBytes1, iBytes1, sGen1, fGen1 := bv.GetUploadHints()
	if sGen1 == 0 || fGen1 == 0 || sBytes1 <= 0 || iBytes1 <= 0 {
		t.Fatalf("expected valid initial build hints, got sBytes=%d iBytes=%d sGen=%d fGen=%d", sBytes1, iBytes1, sGen1, fGen1)
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
	medID := medikitThing.GetEntity().GetId()
	ammoID := ammoThing.GetEntity().GetId()
	medCacheGen1 := bv.thingCache[medID].gen
	ammoCacheGen1 := bv.thingCache[ammoID].gen

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
	if bv.thingCache[medID].gen <= medCacheGen1 {
		t.Fatalf("expected moved item cache gen to advance, got %d -> %d", medCacheGen1, bv.thingCache[medID].gen)
	}
	if bv.thingCache[ammoID].gen != ammoCacheGen1 {
		t.Fatalf("expected stationary item vertex cache gen=%d reused, got %d", ammoCacheGen1, bv.thingCache[ammoID].gen)
	}

	// Frame 4: mark sector dirty -> must invalidate static volume cache and increment staticGen.
	en.GetVolumes().GetVolumes().Get(0).SetDirty(true)
	bv.Compute(640, 480, vi, en)
	_, _, sGen4, _ := bv.GetUploadHints()
	if sGen4 <= sGen3 {
		t.Fatalf("expected staticGen increment when dirty volume changes, got %d -> %d", sGen3, sGen4)
	}

	// Frame 5: animated material frame change when tick crosses tickInterval (32) must rebuild staticGen.
	startTick := textures.GlobalTick()
	for textures.GlobalTick() <= startTick+32 {
		textures.Tick()
	}
	bv.Compute(640, 480, vi, en)
	_, _, sGen5, _ := bv.GetUploadHints()
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
