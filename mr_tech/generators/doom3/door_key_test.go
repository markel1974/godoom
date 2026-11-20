package doom3

import (
	"math"
	"testing"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/generators/doom3/proc"
	"github.com/markel1974/godoom/mr_tech/geometry"
)

func TestDoom3KeyAndDoorParsing(t *testing.T) {
	if got := d3KeyId("item_key_blue"); got != "blue" {
		t.Fatalf("d3KeyId(item_key_blue) = %q, want blue", got)
	}
	if got := d3KeyId("item_key_yellow"); got != "yellow" {
		t.Fatalf("d3KeyId(item_key_yellow) = %q, want yellow", got)
	}
	if got := d3KeyId("item_keycard_aco"); got != "aco" {
		t.Fatalf("d3KeyId(item_keycard_aco) = %q, want aco", got)
	}
	if got := d3KeyId("item_keycard_generic"); got != "generic" {
		t.Fatalf("d3KeyId(item_keycard_generic) = %q, want generic", got)
	}
	if got := d3KeyId("item_medkit"); got != "" {
		t.Fatalf("d3KeyId(item_medkit) = %q, want empty", got)
	}

	doorEnt := proc.NewMapEntity()
	doorEnt.Properties["classname"] = "func_door"
	doorEnt.Properties["name"] = "test_door_1"
	doorEnt.Properties["angle"] = "0"
	doorEnt.Properties["lip"] = "8"
	doorEnt.Properties["speed"] = "200"
	doorEnt.Properties["wait"] = "-1"
	doorEnt.Properties["requires"] = "blue"

	doorSpec := BuildDoorSpec(doorEnt, geometry.XYZ{X: 64, Y: 16, Z: 128})
	if doorSpec.RequiredKey != "blue" {
		t.Fatalf("door RequiredKey = %q, want blue", doorSpec.RequiredKey)
	}
	if math.Abs(doorSpec.OpenOffset.X-56.0) > 1e-6 || math.Abs(doorSpec.OpenOffset.Y) > 1e-6 || math.Abs(doorSpec.OpenOffset.Z) > 1e-6 {
		t.Fatalf("unexpected door OpenOffset = %+v, want {56, 0, 0}", doorSpec.OpenOffset)
	}

	platEnt := proc.NewMapEntity()
	platEnt.Properties["classname"] = "func_plat"
	platEnt.Properties["height"] = "120"
	platEnt.Properties["speed"] = "250"

	platSpec := BuildPlatSpec(platEnt, geometry.XYZ{X: 64, Y: 64, Z: 32})
	if platSpec.OpenOffset.Z != -120 || platSpec.Speed != 250 {
		t.Fatalf("unexpected plat spec: %+v", platSpec)
	}

	// Verify factory creations
	doorCfg := CreateConfigDoor("func_door", geometry.XYZ{}, nil, 0, 1000.0, 0, doorEnt, geometry.XYZ{X: 64, Y: 16, Z: 128})
	if doorCfg.Kind != config.ThingDoorDef || doorCfg.Door == nil || doorCfg.Door.RequiredKey != "blue" {
		t.Fatalf("unexpected CreateConfigDoor: %+v", doorCfg)
	}

	platCfg := CreateConfigPlat("func_plat", geometry.XYZ{}, nil, 0, 1000.0, 0, platEnt, geometry.XYZ{X: 64, Y: 64, Z: 32})
	if platCfg.Kind != config.ThingDoorDef || platCfg.Door == nil || platCfg.Door.OpenOffset.Z != -120 {
		t.Fatalf("unexpected CreateConfigPlat: %+v", platCfg)
	}

	platfCfg := CreateConfigPlatform("func_train", geometry.XYZ{}, nil, 0, 1000.0, 0)
	if platfCfg.Kind != config.ThingPlatformDef {
		t.Fatalf("unexpected CreateConfigPlatform kind: %v", platfCfg.Kind)
	}

	btnCfg := CreateConfigButton("func_button", geometry.XYZ{}, nil, 0, 16.0, 16.0)
	if btnCfg.Kind != config.ThingButtonDef {
		t.Fatalf("unexpected CreateConfigButton kind: %v", btnCfg.Kind)
	}

	// Verify key item creation
	keyCfg := CreateConfigThing("item_key_blue", "item", geometry.XYZ{}, nil, nil)
	if keyCfg.Kind != config.ThingKeyDef || keyCfg.Key == nil || keyCfg.Key.KeyId != "blue" {
		t.Fatalf("unexpected keyCfg: %+v", keyCfg)
	}
}
