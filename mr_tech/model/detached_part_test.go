package model

import (
	"math"
	"testing"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/model/ithing"
	"github.com/markel1974/godoom/mr_tech/model/materials"
	"github.com/markel1974/godoom/mr_tech/model/things"
	"github.com/markel1974/godoom/mr_tech/model/vertices"
	"github.com/markel1974/godoom/mr_tech/physics/aabb"
	"github.com/markel1974/godoom/mr_tech/physics/rigid_body"
	"github.com/markel1974/godoom/mr_tech/textures"
)

func makeBoxEntry(mat *config.Material, name string, hx, hy, hz float64, tagName string, tagPos geometry.XYZ) *config.Model3DEntry {
	entry := config.NewModel3DEntry(1, []string{name}, nil)
	entry.ActionDefinitions = []string{"idle"}
	entry.ActionIntervals = [][2]int{{0, 0}}

	// 8 corners of an axis-aligned box [-hx..hx, -hy..hy, -hz..hz].
	v000 := geometry.XYZ{X: -hx, Y: -hy, Z: -hz}
	v100 := geometry.XYZ{X: hx, Y: -hy, Z: -hz}
	v110 := geometry.XYZ{X: hx, Y: hy, Z: -hz}
	v010 := geometry.XYZ{X: -hx, Y: hy, Z: -hz}
	v001 := geometry.XYZ{X: -hx, Y: -hy, Z: hz}
	v101 := geometry.XYZ{X: hx, Y: -hy, Z: hz}
	v111 := geometry.XYZ{X: hx, Y: hy, Z: hz}
	v011 := geometry.XYZ{X: -hx, Y: hy, Z: hz}

	addQuad := func(a, b, c, d geometry.XYZ) {
		t0 := config.NewModel3DEntryTriangle(mat)
		t0.Vertices[0].Pos, t0.Vertices[1].Pos, t0.Vertices[2].Pos = a, b, c
		t1 := config.NewModel3DEntryTriangle(mat)
		t1.Vertices[0].Pos, t1.Vertices[1].Pos, t1.Vertices[2].Pos = a, c, d
		entry.Frames[0].Triangles = append(entry.Frames[0].Triangles, t0, t1)
	}
	addQuad(v000, v010, v110, v100) // bottom
	addQuad(v001, v101, v111, v011) // top
	addQuad(v000, v100, v101, v001) // front
	addQuad(v010, v011, v111, v110) // back
	addQuad(v000, v001, v011, v010) // left
	addQuad(v100, v110, v111, v101) // right

	if tagName != "" {
		entry.Frames[0].Tags[tagName] = config.NewModel3DTag(tagPos, [3]geometry.XYZ{
			{X: 1, Y: 0, Z: 0},
			{X: 0, Y: 1, Z: 0},
			{X: 0, Y: 0, Z: 1},
		})
	}
	return entry
}

func TestDetachedLimbAutonomousSixDoFRigidBodyPhysics(t *testing.T) {
	rigid_body.SetFps(60)
	defer rigid_body.SetFps(120)

	// World: floor at z=0 and a solid back wall at x=180.
	vols := newSolidVolume([][3]geometry.XYZ{
		{{X: -400, Y: -400, Z: 0}, {X: 400, Y: -400, Z: 0}, {X: 400, Y: 400, Z: 0}},
		{{X: -400, Y: -400, Z: 0}, {X: 400, Y: 400, Z: 0}, {X: -400, Y: 400, Z: 0}},
		{{X: 180, Y: 400, Z: 0}, {X: 180, Y: -400, Z: 0}, {X: 180, Y: -400, Z: 300}},
		{{X: 180, Y: 400, Z: 0}, {X: 180, Y: -400, Z: 300}, {X: 180, Y: 400, Z: 300}},
	})

	tex := textures.NewTexture("SOLID", 1, 64, 64, false)
	cfgMat := config.NewConfigMaterial("solid", []string{"SOLID"}, config.MaterialKindSolid)
	mats := materials.NewMaterials(&stubTexHandler{tex: tex})

	lower := makeBoxEntry(cfgMat, "lower", 10, 10, 12, "tag_torso", geometry.XYZ{X: 0, Y: 0, Z: 12})
	upper := makeBoxEntry(cfgMat, "upper", 12, 12, 14, "tag_head", geometry.XYZ{X: 0, Y: 0, Z: 14})
	head := makeBoxEntry(cfgMat, "head", 8, 8, 8, "", geometry.XYZ{})

	jointsCfg := config.NewJointsConfig()
	for _, j := range jointsCfg.Joints {
		for _, tag := range j.Tags {
			if tag == "head" {
				j.CanDetach = true
				j.DetachThreshold = 0.5
			}
		}
	}

	md3 := config.NewMD3([]*config.Model3DEntry{lower, upper, head}, []config.Model3DLink{
		{Parent: -1, Tag: ""},
		{Parent: 0, Tag: "tag_torso"},
		{Parent: 1, Tag: "tag_head"},
	})
	md3.Joints = jointsCfg

	var impactDelivered bool
	enemyCfg := config.NewConfigThing("enemy_md3", geometry.XYZ{X: 0, Y: 0, Z: 34}, 0, config.ThingEnemyDef, 80, 16, 68, 0)
	enemyCfg.MultiElement3D = md3
	enemyCfg.GForce = 490.0
	enemyCfg.OnThinking = func(playerX, playerY, playerZ float64) {}
	enemyCfg.OnCollision = func(other config.IThingConfig, energy float64) {}
	enemyCfg.OnImpact = func(other config.IThingConfig, id string, force, closestDist, dirX, dirY, dirZ float64) {
		impactDelivered = true
	}
	enemyCfg.OnBind = func(other config.IThingConfig) {}

	th := NewThings(256, 10, nil, vols, mats)
	enemy := th.Create(enemyCfg, vols.GetVolumes()[0]).(*things.Enemy)
	th.addThing(enemy)

	// Run one frame so transforms settle and head center is up to date.
	th.StagePrepare()
	th.StageAdd(enemy)
	th.Compute(0, 0, 0)

	// Verify all 3 parts were injected into Things.tree from creation with 12 triangles each and owner reference.
	partLen := enemy.GetPartLen()
	if partLen != 3 {
		t.Fatalf("expected 3 segment IThings injected into Things.tree, got %d", partLen)
	}
	treeParts := make([]ithing.IThing, partLen)
	for idx := 0; idx < partLen; idx++ {
		p := enemy.GetPartAt(idx)
		treeParts[idx] = p
		sub := p.(vertices.ISubPart)
		if sub.GetOwner() == nil || !sub.GetOwner().IsEqual(enemy) {
			t.Fatalf("expected part %d owner to reference parent enemy", idx)
		}
		if sub.IsPhysicsFree() {
			t.Fatalf("expected living part %d to start kinematically attached (!IsPhysicsFree)", idx)
		}
		facesBefore, _, _, _ := p.GetUIFaces()
		nonZero := 0
		for i := 0; i < facesBefore.Len(); i++ {
			f := facesBefore.At(i)
			x0, y0, z0 := f.GetPoint0()
			x1, y1, z1 := f.GetPoint1()
			x2, y2, z2 := f.GetPoint2()
			if x0 != x1 || y0 != y1 || z0 != z1 || x1 != x2 || y1 != y2 || z1 != z2 {
				nonZero++
			}
		}
		if nonZero != 12 {
			t.Fatalf("expected 12 triangles on part %d, got %d", idx, nonZero)
		}
	}

	// Verify QueryRay in Things.tree hits the head part directly above the upper torso seam.
	headPart := treeParts[2].(*things.Detached)
	hx, hy, hz := headPart.GetEntity().GetCenter()
	var rayHit ithing.IThing
	th.QueryRay(hx-100, hy, hz+4, 1, 0, 0, 500, func(object aabb.IObject, dist float64) (float64, bool) {
		rayHit = object.(ithing.IThing)
		return dist, true
	})
	if rayHit == nil || !rayHit.IsEqual(headPart) {
		t.Fatalf("expected QueryRay at head level (z=%.2f) to hit head part directly in tree, got %v", hz+4, rayHit)
	}

	// Hit the head strongly toward +X (+500k force) directly via its emerging Impact event to sever it cleanly.
	impactDelivered = false
	headPart.Impact(enemy, "shotgun", 500000, 100, 1, 0, 0.2)
	if !impactDelivered {
		t.Fatalf("expected Impact on attached head part to forward damage to owning enemy")
	}
	if !headPart.IsPhysicsFree() || headPart.GetOwner() != nil {
		t.Fatalf("expected detached head part to become 6-DoF physics-free and release its owner reference")
	}
	if treeParts[0].(vertices.ISubPart).IsPhysicsFree() || treeParts[1].(vertices.ISubPart).IsPhysicsFree() {
		t.Fatalf("expected torso and lower parts to stay kinematically attached when head detaches")
	}

	// Simulate autonomous 6-DoF wall bounce & floor settling.
	for step := 0; step < 300; step++ {
		th.StagePrepare()
		for _, item := range th.entities {
			item.Compile(uint64(step + 1))
			th.StageAdd(item)
		}
		th.Compute(0, 0, 0)
	}

	lx, ly, lz := headPart.GetEntity().GetCenter()
	if lx >= 180.0-7.5 || lz < 7.0 || math.IsNaN(lx) || math.IsNaN(ly) || math.IsNaN(lz) {
		t.Fatalf("detached limb penetrated static world wall/floor: center=(%.3f, %.3f, %.3f)", lx, ly, lz)
	}
	if headPart.GetEntity().KeepUpright() {
		t.Fatalf("expected detached limb to tumble freely in 6-DoF (KeepUpright=false)")
	}
	if !headPart.IsSleeping() {
		t.Fatalf("expected detached limb to settle and enter sleep within 300 ticks, still awake at center=(%.3f, %.3f, %.3f)", lx, ly, lz)
	}

	// Shooting the resting limb wakes it up cleanly and moves it without forwarding damage to former owner.
	impactDelivered = false
	headPart.GetEntity().AddForceAtPoint(-200000, 0, 150000, lx, ly, lz)
	headPart.Impact(enemy, "shotgun", 250000, 100, -0.8, 0, 0.6)
	if impactDelivered {
		t.Fatalf("expected detached limb Impact to not damage former enemy owner")
	}
	if headPart.IsSleeping() {
		t.Fatalf("expected Impact/AddForceAtPoint to wake resting detached limb")
	}
}

func TestMultiBodyRagdollPGSSimulationAndJointBreak(t *testing.T) {
	rigid_body.SetFps(60)
	defer rigid_body.SetFps(120)

	// Floor at z=0 and a solid back wall at x=160.
	vols := newSolidVolume([][3]geometry.XYZ{
		{{X: -400, Y: -400, Z: 0}, {X: 400, Y: -400, Z: 0}, {X: 400, Y: 400, Z: 0}},
		{{X: -400, Y: -400, Z: 0}, {X: 400, Y: 400, Z: 0}, {X: -400, Y: 400, Z: 0}},
		{{X: 160, Y: 400, Z: 0}, {X: 160, Y: -400, Z: 0}, {X: 160, Y: -400, Z: 300}},
		{{X: 160, Y: 400, Z: 0}, {X: 160, Y: -400, Z: 300}, {X: 160, Y: 400, Z: 300}},
	})

	tex := textures.NewTexture("SOLID", 1, 64, 64, false)
	cfgMat := config.NewConfigMaterial("solid", []string{"SOLID"}, config.MaterialKindSolid)
	mats := materials.NewMaterials(&stubTexHandler{tex: tex})

	lower := makeBoxEntry(cfgMat, "lower", 10, 10, 12, "tag_torso", geometry.XYZ{X: 0, Y: 0, Z: 12})
	upper := makeBoxEntry(cfgMat, "upper", 12, 12, 14, "tag_head", geometry.XYZ{X: 0, Y: 0, Z: 14})
	head := makeBoxEntry(cfgMat, "head", 8, 8, 8, "", geometry.XYZ{})

	jointsCfg := config.NewJointsConfig()
	for _, j := range jointsCfg.Joints {
		for _, tag := range j.Tags {
			if tag == "head" {
				j.CanDetach = true
				j.DetachThreshold = 0.8
			}
		}
	}

	md3 := config.NewMD3([]*config.Model3DEntry{lower, upper, head}, []config.Model3DLink{
		{Parent: -1, Tag: ""},
		{Parent: 0, Tag: "tag_torso"},
		{Parent: 1, Tag: "tag_head"},
	})
	md3.Joints = jointsCfg

	enemyCfg := config.NewConfigThing("enemy_ragdoll", geometry.XYZ{X: 0, Y: 0, Z: 34}, 0, config.ThingEnemyDef, 80, 16, 68, 0)
	enemyCfg.MultiElement3D = md3
	enemyCfg.GForce = 490.0
	enemyCfg.OnThinking = func(playerX, playerY, playerZ float64) {}
	enemyCfg.OnCollision = func(other config.IThingConfig, energy float64) {}
	enemyCfg.OnImpact = func(other config.IThingConfig, id string, force, closestDist, dirX, dirY, dirZ float64) {}
	enemyCfg.OnBind = func(other config.IThingConfig) {}

	th := NewThings(256, 10, nil, vols, mats)
	enemy := th.Create(enemyCfg, vols.GetVolumes()[0]).(*things.Enemy)
	th.addThing(enemy)

	// Compile alive pose once, then trigger multi-body ragdoll collapse.
	th.StagePrepare()
	th.StageAdd(enemy)
	th.Compute(0, 0, 0)

	enemy.GetEntity().SetV(120, 0, 0)
	enemy.SetRagdollMode()
	if enemy.IsActive() {
		t.Fatalf("expected owner locomotion collider to deactivate when tree segments switch to ragdoll")
	}

	partLenRagdoll := enemy.GetPartLen()
	if partLenRagdoll != 3 {
		t.Fatalf("expected 3 ragdoll segment parts (lower + upper + head), got %d", partLenRagdoll)
	}
	parts := make([]*things.Detached, partLenRagdoll)
	groupID := enemy.GetEntity().GetId()
	for i := 0; i < partLenRagdoll; i++ {
		d := enemy.GetPartAt(i).(*things.Detached)
		parts[i] = d
		if !d.IsPhysicsFree() {
			t.Fatalf("expected ragdoll part %d to switch to autonomous 6-DoF physics", i)
		}
		if d.GetEntity().KeepUpright() {
			t.Fatalf("expected ragdoll part %d to unlock upright constraint (KeepUpright=false)", i)
		}
		if d.GetSolver().GetCollisionGroup() != groupID {
			t.Fatalf("ragdoll part %d missing parent collisionGroup %d, got %d", i, groupID, d.GetSolver().GetCollisionGroup())
		}
	}

	// Simulate collapse onto the floor and verify all 3 tree parts stay coupled without floor penetration or sleeping stall.
	for step := 0; step < 600; step++ {
		th.StagePrepare()
		for _, item := range th.entities {
			item.Compile(uint64(step + 1))
			th.StageAdd(item)
		}
		th.Compute(0, 0, 0)
	}

	for i, p := range parts {
		px, py, pz := p.GetEntity().GetCenter()
		if px >= 160.0 || pz < 4.0 || math.IsNaN(px) || math.IsNaN(py) || math.IsNaN(pz) {
			t.Fatalf("ragdoll part %d invalid/penetrating pos=(%.3f, %.3f, %.3f)", i, px, py, pz)
		}
		if !p.IsSleeping() {
			pvx, pvy, pvz := p.GetEntity().GetVelocity()
			pwx, pwy, pwz := p.GetEntity().GetAngularVelocity()
			ppx, ppy, ppz := p.GetSolver().GetCage().GetNetDisplacement()
			t.Fatalf("expected settled ragdoll part %d to sleep at (%.3f,%.3f,%.3f): onGround=%v v=(%.3f,%.3f,%.3f) w=(%.3f,%.3f,%.3f) net=(%.4f,%.4f,%.4f)",
				i, px, py, pz, p.GetEntity().IsOnGround(), pvx, pvy, pvz, pwx, pwy, pwz, ppx, ppy, ppz)
		}
	}

	// Verify parent+upper and upper+head remained closely linked (no ragdoll explosion).
	rx, ry, rz := parts[0].GetEntity().GetCenter()
	for i := 1; i < len(parts); i++ {
		px, py, pz := parts[i].GetEntity().GetCenter()
		if d := math.Sqrt((px-rx)*(px-rx) + (py-ry)*(py-ry) + (pz-rz)*(pz-rz)); d > 85.0 {
			t.Fatalf("ragdoll limb %d drifted too far from lower root: dist=%.2f", i, d)
		}
	}

	// Shoot the head segment hard enough to break its neck JointConstraint via Force + Impact event.
	headPart := parts[2]
	hx, hy, hz := headPart.GetEntity().GetCenter()
	headPart.GetEntity().AddForceAtPoint(400000, 0, 200000, hx, hy, hz)
	headPart.Impact(enemy, "shotgun", 450000, 100, 0.9, 0, 0.45)

	for step := 0; step < 30; step++ {
		th.StagePrepare()
		for _, item := range th.entities {
			item.Compile(uint64(360 + step))
			th.StageAdd(item)
		}
		th.Compute(0, 0, 0)
	}
	hx2, _, _ := headPart.GetEntity().GetCenter()
	if hx2-hx < 20.0 {
		t.Fatalf("expected severed ragdoll head after joint break to launch forward, deltaX=%.3f", hx2-hx)
	}
}

type stubTexHandler struct{ tex *textures.Texture }

func (m *stubTexHandler) GetNames() []string { return []string{"SOLID"} }
func (m *stubTexHandler) GetFrames(_ []string) []*textures.Texture {
	return []*textures.Texture{m.tex}
}
