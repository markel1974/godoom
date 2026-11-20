package model

import (
	"math"
	"testing"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/model/materials"
	"github.com/markel1974/godoom/mr_tech/model/things"
	"github.com/markel1974/godoom/mr_tech/physics/rigid_body"
	"github.com/markel1974/godoom/mr_tech/textures"
)

func noopBind(config.IThingConfig) {}
func noopCollision(config.IThingConfig, float64) {
}
func noopImpact(config.IThingConfig, string, float64, float64, float64, float64, float64) {
}

func TestKeyedDoorRequiresKeyOpensAndAutoCloses(t *testing.T) {
	rigid_body.SetFps(60)
	defer rigid_body.SetFps(120)

	// Flat floor plus a wall pocket intersecting the door's open destination (verifies IgnoreStatic).
	vols := newSolidVolume([][3]geometry.XYZ{
		{{X: -300, Y: -300, Z: 0}, {X: 300, Y: -300, Z: 0}, {X: 300, Y: 300, Z: 0}},
		{{X: -300, Y: -300, Z: 0}, {X: 300, Y: 300, Z: 0}, {X: -300, Y: 300, Z: 0}},
		{{X: 80, Y: -60, Z: 100}, {X: 140, Y: -60, Z: 100}, {X: 140, Y: 60, Z: 100}},
	})

	tex := textures.NewTexture("SOLID", 1, 64, 64, false)
	cfgMat := config.NewConfigMaterial("solid", []string{"SOLID"}, config.MaterialKindSolid)
	mats := materials.NewMaterials(&stubTexHandler{tex: tex})

	doorCfg := config.NewConfigThing("door_silver", geometry.XYZ{X: 100, Y: 0, Z: 32}, 0, config.ThingDoorDef, 1000.0, 16, 16, 0)
	doorCfg.GForce = 0
	doorCfg.Element3D = makeBoxEntry(cfgMat, "door", 8, 32, 32, "", geometry.XYZ{})
	doorCfg.Door = &config.DoorSpec{
		RequiredKey: "silver",
		OpenOffset:  geometry.XYZ{X: 0, Y: 0, Z: 64},
		Speed:       128.0,
		WaitSeconds: 0.25,
		Activation:  config.DoorActivateUse,
	}
	doorCfg.OnBind = noopBind
	doorCfg.OnCollision = noopCollision
	doorCfg.OnImpact = noopImpact

	keyCfg := config.NewConfigThing("item_key1", geometry.XYZ{X: 40, Y: 0, Z: 28}, 0, config.ThingKeyDef, 0.1, 12, 24, 0)
	keyCfg.GForce = 0
	keyCfg.Element3D = makeBoxEntry(cfgMat, "key", 8, 8, 12, "", geometry.XYZ{})
	keyCfg.Key = &config.KeySpec{KeyId: "silver"}
	keyCfg.OnBind = noopBind
	keyCfg.OnCollision = noopCollision
	keyCfg.OnImpact = noopImpact

	th := NewThings(64, 8, []*config.Thing{doorCfg, keyCfg}, vols, mats)
	if err := th.Setup(); err != nil {
		t.Fatalf("Things.Setup: %v", err)
	}

	pCfg := config.NewConfigPlayer(geometry.XYZ{X: 0, Y: 0, Z: 28}, 0, 80, 100, 16, 56)
	pCfg.GForce = 0
	pCfg.Thing.OnBind = noopBind
	pCfg.Thing.OnCollision = noopCollision
	pCfg.Thing.OnImpact = noopImpact
	player := things.NewPlayer(th, pCfg, vols.GetVolumes()[0], false)
	if err := player.Initialize(); err != nil {
		t.Fatalf("player.Initialize: %v", err)
	}
	th.SetPlayer(player)

	var door *things.Door
	var key *things.Key
	for _, ent := range th.entities {
		switch v := ent.(type) {
		case *things.Door:
			door = v
		case *things.Key:
			key = v
		}
	}
	if door == nil || key == nil {
		t.Fatalf("expected both door (%v) and key (%v) to exist", door, key)
	}

	// 0. Verify initial door center and AABB align with closedPos before any movement/activation.
	dx, dy, dz := door.GetEntity().GetCenter()
	if math.Abs(dx-100.0) > 1e-6 || math.Abs(dy-0.0) > 1e-6 || math.Abs(dz-32.0) > 1e-6 {
		t.Fatalf("expected initial door center at (100, 0, 32), got (%.3f, %.3f, %.3f)", dx, dy, dz)
	}
	doorBox := door.GetAABB()
	if math.Abs(doorBox.GetMinX()-92.0) > 1e-6 || math.Abs(doorBox.GetMaxX()-108.0) > 1e-6 ||
		math.Abs(doorBox.GetMinZ()-0.0) > 1e-6 || math.Abs(doorBox.GetMaxZ()-64.0) > 1e-6 {
		t.Fatalf("expected initial door AABB [92..108, -32..32, 0..64], got [%.2f..%.2f, %.2f..%.2f, %.2f..%.2f]",
			doorBox.GetMinX(), doorBox.GetMaxX(), doorBox.GetMinY(), doorBox.GetMaxY(), doorBox.GetMinZ(), doorBox.GetMaxZ())
	}

	stepWorld := func() {
		th.StagePrepare()
		for _, ent := range th.entities {
			th.StageAdd(ent)
		}
		px, py, pz := player.GetEntity().GetCenter()
		th.Compute(px, py, pz)
	}

	// 1. Attempting to Use the locked door before grabbing the key must fail.
	player.GetEntity().MoveCenterTo(50, 0, 28)
	th.tree.UpdateObject(player)
	player.SetAngle(0)
	if player.Use() {
		t.Fatalf("expected locked door not to open before collecting silver key")
	}
	if door.IsOpen() {
		t.Fatalf("door should remain closed without silver key")
	}

	// 2. Walk player into key to collect it and remove it from Things.
	player.GetEntity().MoveCenterTo(22, 0, 28)
	th.tree.UpdateObject(player)
	for i := 0; i < 20; i++ {
		player.GetEntity().SetV(200, 0, 0)
		stepWorld()
		if player.HasKey("silver") {
			break
		}
	}
	if !player.HasKey("silver") {
		t.Fatalf("expected player to hold silver key after collision")
	}
	if _, stillPresent := th.entities[key.GetEntity().GetId()]; stillPresent {
		t.Fatalf("expected collected key to be removed from Things.entities")
	}

	// 3. Use the door now that player carries silver key; door should slide up to Z = 32 + 64 = 96.
	player.GetEntity().MoveCenterTo(50, 0, 28)
	th.tree.UpdateObject(player)
	player.GetEntity().SetV(0, 0, 0)
	player.SetAngle(0)
	if !player.Use() {
		t.Fatalf("expected Use() to unlock and open door once player has silver key")
	}
	for i := 0; i < 40; i++ {
		stepWorld()
	}
	_, _, openZ := door.GetEntity().GetCenter()
	if math.Abs(openZ-96.0) > 1.0 {
		t.Fatalf("expected door to slide to Z=96.0, got Z=%.3f", openZ)
	}

	// 4. Advance past waitSeconds + closing travel; door should auto-close back to Z=32.0.
	for i := 0; i < 60; i++ {
		stepWorld()
	}
	_, _, closedZ := door.GetEntity().GetCenter()
	if math.Abs(closedZ-32.0) > 1.0 {
		t.Fatalf("expected door to auto-close back to Z=32.0, got Z=%.3f", closedZ)
	}
}
