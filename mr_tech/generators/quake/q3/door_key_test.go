package q3

import (
	"math"
	"testing"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/generators/quake/lumps"
	"github.com/markel1974/godoom/mr_tech/geometry"
)

func TestQ3KeyAndDoorParsing(t *testing.T) {
	if got := q3KeyId("item_key1"); got != "silver" {
		t.Fatalf("q3KeyId(item_key1) = %q, want silver", got)
	}
	if got := q3KeyId("item_key2"); got != "gold" {
		t.Fatalf("q3KeyId(item_key2) = %q, want gold", got)
	}
	if got := q3KeyId("key_blue_key"); got != "blue" {
		t.Fatalf("q3KeyId(key_blue_key) = %q, want blue", got)
	}
	if got := q3KeyId("item_health"); got != "" {
		t.Fatalf("q3KeyId(item_health) = %q, want empty", got)
	}

	silverEnt := &lumps.Entity{Properties: map[string]string{
		"classname":  "func_door",
		"angle":      "0",
		"lip":        "8",
		"speed":      "200",
		"wait":       "-1",
		"spawnflags": "2064",
	}}
	silverSpec := BuildDoorSpec(silverEnt, geometry.XYZ{X: 64, Y: 16, Z: 128})
	if silverSpec.RequiredKey != "silver" {
		t.Fatalf("silver door RequiredKey = %q, want silver", silverSpec.RequiredKey)
	}
	if math.Abs(silverSpec.OpenOffset.X-56.0) > 1e-6 || math.Abs(silverSpec.OpenOffset.Y) > 1e-6 || math.Abs(silverSpec.OpenOffset.Z) > 1e-6 {
		t.Fatalf("unexpected silver OpenOffset = %+v, want {56, 0, 0}", silverSpec.OpenOffset)
	}

	goldEnt := &lumps.Entity{Properties: map[string]string{
		"classname":  "func_door",
		"angle":      "-1",
		"spawnflags": "2057",
	}}
	goldSpec := BuildDoorSpec(goldEnt, geometry.XYZ{X: 32, Y: 64, Z: 96})
	if goldSpec.RequiredKey != "gold" || !goldSpec.StartOpen {
		t.Fatalf("unexpected gold door lock/startOpen: %+v", goldSpec)
	}

	platEnt := &lumps.Entity{Properties: map[string]string{
		"classname": "func_plat",
		"height":    "120",
		"speed":     "250",
	}}
	platSpec := BuildPlatSpec(platEnt, geometry.XYZ{X: 64, Y: 64, Z: 32})
	if platSpec.OpenOffset.Z != -120 || platSpec.Speed != 250 {
		t.Fatalf("unexpected plat spec: %+v", platSpec)
	}

	// Verify factory creations
	doorCfg := CreateConfigDoor("func_door", geometry.XYZ{}, nil, 0, 1000.0, 0, silverEnt, geometry.XYZ{X: 64, Y: 16, Z: 128})
	if doorCfg.Kind != config.ThingDoorDef || doorCfg.Door == nil || doorCfg.Door.RequiredKey != "silver" {
		t.Fatalf("unexpected CreateConfigDoor: %+v", doorCfg)
	}

	platCfg := CreateConfigPlat("func_plat", geometry.XYZ{}, nil, 0, 1000.0, 0, platEnt, geometry.XYZ{X: 64, Y: 64, Z: 32})
	if platCfg.Kind != config.ThingDoorDef || platCfg.Door == nil || platCfg.Door.OpenOffset.Z != -120 {
		t.Fatalf("unexpected CreateConfigPlat: %+v", platCfg)
	}

	platformCfg := CreateConfigPlatform("func_train", geometry.XYZ{}, nil, 0, 1000.0, 0)
	if platformCfg.Kind != config.ThingPlatformDef {
		t.Fatalf("unexpected CreateConfigPlatform kind: %v", platformCfg.Kind)
	}

	btnCfg := CreateConfigButton("func_button", geometry.XYZ{}, nil, 0, 16.0, 16.0)
	if btnCfg.Kind != config.ThingButtonDef {
		t.Fatalf("unexpected CreateConfigButton kind: %v", btnCfg.Kind)
	}

	// Verify key item creation
	keyCfg := CreateConfigThing("item_key1", "item", geometry.XYZ{}, nil, nil)
	if keyCfg.Kind != config.ThingKeyDef || keyCfg.Key == nil || keyCfg.Key.KeyId != "silver" {
		t.Fatalf("unexpected keyCfg: %+v", keyCfg)
	}
}
