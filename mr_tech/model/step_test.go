package model

import (
	"math"
	"testing"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/model/ithing"
	"github.com/markel1974/godoom/mr_tech/model/materials"
	"github.com/markel1974/godoom/mr_tech/model/vertices"
	"github.com/markel1974/godoom/mr_tech/model/vertices/vertices_element_3d"
	"github.com/markel1974/godoom/mr_tech/model/volumes"
	"github.com/markel1974/godoom/mr_tech/physics/aabb"
	"github.com/markel1974/godoom/mr_tech/physics/collisions"
	"github.com/markel1974/godoom/mr_tech/physics/rigid_body"
	"github.com/markel1974/godoom/mr_tech/textures"
)

type stepTestObject struct {
	ent *rigid_body.Entity
	box *aabb.AABB
}

func newStepTestObject(cx, cy, cz, w, h, d float64) (*collisions.Solver, *stepTestObject) {
	solver := collisions.NewSolver(true, 80.0, 0.0, 0.2, int(rigid_body.CombineAverage), int(rigid_body.CombineAverage), 0.0)
	ent := solver.GetEntity()
	ent.Rebuild(cx-w*0.5, cy-h*0.5, cz-d*0.5, w, h, d)
	ent.SetKeepUpright(true)
	ent.LockUpright()
	obj := &stepTestObject{
		ent: ent,
		box: aabb.NewAABB(),
	}
	_ = solver.Setup(obj)
	return solver, obj
}

func (o *stepTestObject) GetAABB() *aabb.AABB {
	cx, cy, cz := o.ent.GetCenter()
	w, h, d := o.ent.GetSize()
	hx, hy, hz := w*0.5, h*0.5, d*0.5
	o.box.Rebuild(cx-hx, cy-hy, cz-hz, cx+hx, cy+hy, cz+hz)
	return o.box
}

func (o *stepTestObject) GetEntity() *rigid_body.Entity {
	return o.ent
}

func newSolidVolume(faces [][3]geometry.XYZ) *Volumes {
	mat := textures.NewMaterial("solid", nil, 0)
	vol := volumes.NewVolumeConcrete(0, "test_vol", "test")
	for _, tri := range faces {
		f := volumes.NewFace(tri, "solid", mat)
		vol.AddFace(f)
	}
	vs := NewVolumesFromSlice([]*volumes.Volume{vol})
	if err := vs.Setup(); err != nil {
		panic(err)
	}
	return vs
}

func newStepTestCage(id uint64, solver *collisions.Solver) *collisions.Cage {
	cage := collisions.NewCage(id)
	_ = cage.Setup(solver)
	return cage
}

func TestApplySafeStepUpClearHeadroom(t *testing.T) {
	// Stair riser in front at y=200 up to z=16, high ceiling at z=200.
	vols := newSolidVolume([][3]geometry.XYZ{
		{{X: -100, Y: 200, Z: 0}, {X: 100, Y: 200, Z: 0}, {X: 100, Y: 200, Z: 16}},
		{{X: -100, Y: 200, Z: 0}, {X: 100, Y: 200, Z: 16}, {X: -100, Y: 200, Z: 16}},
		{{X: -100, Y: -100, Z: 200}, {X: 100, Y: -100, Z: 200}, {X: 100, Y: 300, Z: 200}},
	})
	solver, obj := newStepTestObject(0.0, 181.0, 30.0, 40.0, 40.0, 60.0)
	cage := newStepTestCage(1, solver)
	cage.Rebuild(30.0, 1)
	cage.SetApplied()

	th := &Things{volumes: vols}
	th.applySafeStepUp(cage, 16.0)

	cx, cy, cz := obj.ent.GetCenter()
	if math.Abs(cx) > 1e-6 || math.Abs(cy-181.0) > 1e-6 || math.Abs(cz-46.0) > 1e-6 {
		t.Fatalf("expected unobstructed step-up to (0, 181, 46), got (%.3f, %.3f, %.3f)", cx, cy, cz)
	}
}

func TestApplySafeStepUpAbortsUnderCeilingOverhang(t *testing.T) {
	// Sloped ceiling overhang around z=134..142 above entity (exact Spawn overhang geometry):
	// lifting z by +27.5 collides with the ceiling bevel and must abort without kicking XY.
	vols := newSolidVolume([][3]geometry.XYZ{
		{{X: -24.5, Y: -813.9, Z: 134.2}, {X: -64.0, Y: -814.4, Z: 134.4}, {X: -52.9, Y: -812.9, Z: 142.0}},
		{{X: -24.5, Y: -813.9, Z: 134.2}, {X: -52.9, Y: -812.9, Z: 142.0}, {X: -10.0, Y: -812.9, Z: 142.0}},
	})
	solver, obj := newStepTestObject(-30.67, -816.51, 87.63, 40.0, 40.0, 60.0)
	obj.ent.SetV(10.0, -50.0, 0.0)
	cage := newStepTestCage(2, solver)
	cage.Rebuild(30.0, 1)
	cage.SetApplied()

	th := &Things{volumes: vols}
	th.applySafeStepUp(cage, 27.47)

	cx, cy, cz := obj.ent.GetCenter()
	if math.Abs(cx-(-30.67)) > 1e-9 || math.Abs(cy-(-816.51)) > 1e-9 || math.Abs(cz-87.63) > 1e-9 {
		t.Fatalf("expected blocked step-up to abort cleanly at (-30.67, -816.51, 87.63), got (%.3f, %.3f, %.3f)", cx, cy, cz)
	}
	vx, vy, vz := obj.ent.GetVelocity()
	if math.Abs(vx-10.0) > 1e-9 || math.Abs(vy-(-50.0)) > 1e-9 || math.Abs(vz) > 1e-9 {
		t.Fatalf("expected velocity preserved on aborted step-up, got (%.2f, %.2f, %.2f)", vx, vy, vz)
	}
}

func TestApplySafeStepUpZeroAllocs(t *testing.T) {
	vols := newSolidVolume([][3]geometry.XYZ{
		{{X: -100, Y: -100, Z: 200}, {X: 100, Y: -100, Z: 200}, {X: 100, Y: 300, Z: 200}},
	})
	solver, obj := newStepTestObject(0.0, 0.0, 30.0, 40.0, 40.0, 60.0)
	cage := newStepTestCage(3, solver)
	th := &Things{volumes: vols}

	allocs := testing.AllocsPerRun(100, func() {
		obj.ent.Rebuild(-20, -20, 0, 40, 40, 60)
		cage.Rebuild(30.0, 1)
		cage.SetApplied()
		th.applySafeStepUp(cage, 16.0)
	})
	if allocs != 0 {
		t.Fatalf("expected 0 heap allocs per applySafeStepUp, got %v", allocs)
	}
}

func TestStaticVolumeLocationAndInelasticCallback(t *testing.T) {
	rigid_body.SetFps(60)
	defer rigid_body.SetFps(120)

	vols := newSolidVolume([][3]geometry.XYZ{
		{{X: 100, Y: 200, Z: 0}, {X: 100, Y: -200, Z: 0}, {X: 100, Y: -200, Z: 200}},
		{{X: 100, Y: 200, Z: 0}, {X: 100, Y: -200, Z: 200}, {X: 100, Y: 200, Z: 200}},
	})
	vol := vols.GetVolumes().Get(0)

	solver, obj := newStepTestObject(75.0, 0.0, 50.0, 40.0, 40.0, 60.0)
	obj.ent.SetOnGround(false)
	obj.ent.SetV(600.0, 0.0, 0.0)
	solver.SetCanSleep(false)

	var gotOther collisions.IObject
	var gotEnergy float64
	solver.SetInelasticCollision(func(other collisions.IObject, energy float64) {
		gotOther = other
		gotEnergy = energy
	})

	coord := collisions.NewCoordinator(10, 8)
	queryContacts := func(cage *collisions.Cage, _ func(*collisions.Solver)) {
		vols.QueryCollisionCage(cage, 512)
	}
	coord.Reset()
	coord.Stage(solver)
	coord.Step(queryContacts, vols.ClampStaticCage)

	if solver.GetLocation() != vol {
		t.Fatalf("expected solver location %p (%v), got %v", vol, vol, solver.GetLocation())
	}
	if gotOther != vol {
		t.Fatalf("expected inelastic callback object %p (%v), got %v", vol, vol, gotOther)
	}
	if gotEnergy <= 0 {
		t.Fatalf("expected positive inelastic impact energy, got %v", gotEnergy)
	}
}

type stubVerticesThing struct {
	ithing.IThing
	verts ithing.IVertices
}

func (s *stubVerticesThing) GetAABB() *aabb.AABB {
	return s.verts.GetAABB()
}

func TestVertices3DEntrySetupAfterFrameSwitch(t *testing.T) {
	mats := materials.NewMaterials(nil)

	// Two frames where action 0 starts at frame 1, so NewVertices3DEntry switches
	// comp.volume away from volumes[0] during Compile(0) before Setup(thing).
	entryCfg := config.NewModel3DEntry(2, []string{"stand1", "walk1"}, nil)
	entryCfg.ActionDefinitions = []string{"walk"}
	entryCfg.ActionIntervals = [][2]int{{1, 1}}
	tri := config.NewModel3DEntryTriangle(nil)
	tri.Vertices[0].Pos = geometry.XYZ{X: -5, Y: -5, Z: -5}
	tri.Vertices[1].Pos = geometry.XYZ{X: 5, Y: -5, Z: -5}
	tri.Vertices[2].Pos = geometry.XYZ{X: 0, Y: 5, Z: 5}
	entryCfg.Frames[0].Triangles = []config.Model3DEntryTriangle{tri}
	entryCfg.Frames[1].Triangles = []config.Model3DEntryTriangle{tri}

	tCfg := config.NewConfigThing("test_md2", geometry.XYZ{}, 0, config.ThingEnemyDef, 80, 10, 20, 0)
	tCfg.Element3D = entryCfg

	v := vertices_element_3d.NewElement3D(tCfg, mats)
	th := &stubVerticesThing{verts: v}
	if err := v.Setup(th); err != nil {
		t.Fatalf("Vertices3DEntry.Setup failed: %v", err)
	}
	v.GetSolver().StagePrepare(1)
}
