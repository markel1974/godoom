package q2

import (
	"math"
	"strings"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/generators/common"
	"github.com/markel1974/godoom/mr_tech/generators/quake/lumps"
	"github.com/markel1974/godoom/mr_tech/geometry"
)

const heavyMass = 1000

func q2KeyId(classname string) string {
	switch classname {
	case "item_key1":
		return "silver"
	case "item_key2":
		return "gold"
	case "key_blue_key":
		return "blue"
	case "key_red_key":
		return "red"
	}
	if strings.HasPrefix(classname, "key_") {
		return strings.TrimPrefix(classname, "key_")
	}
	return ""
}

// TeleportDest holds the world position and facing angle (radians) of a Quake teleport destination entity.
type TeleportDest struct {
	Pos   geometry.XYZ
	Angle float64
}

// CollectTeleportDestinations indexes all teleport destination entities by their targetname.
func CollectTeleportDestinations(entities []*lumps.Entity) map[string]TeleportDest {
	dests := make(map[string]TeleportDest)
	for _, ent := range entities {
		classname, _ := ent.GetProperty("classname")
		if classname != "info_teleport_destination" &&
			classname != "misc_teleporter_dest" &&
			classname != "target_position" {
			continue
		}
		targetName, ok := ent.GetProperty("targetname")
		if !ok || targetName == "" {
			continue
		}
		var pos geometry.XYZ
		if origin, okOrig := ent.GetProperty("origin"); okOrig {
			x, y, z, _ := lumps.ParseVector(origin)
			pos = lumps.CreateXYZ(x, y, z)
		}
		angle := 0.0
		if rawAngle, okAng := ent.GetProperty("angle"); okAng {
			if parsed, okParse := lumps.ParseFloat(rawAngle); okParse {
				angle = parsed * (math.Pi / 180.0)
			}
		}
		dests[targetName] = TeleportDest{Pos: pos, Angle: angle}
	}
	return dests
}

// BuildDoorSpec maps Quake func_door entity properties and brush bounds into a config.DoorSpec.
func BuildDoorSpec(ent *lumps.Entity, size geometry.XYZ) *config.DoorSpec {
	const (
		q1DoorStartOpen = 1
		q1DoorGoldKey   = 8
		q1DoorSilverKey = 16
	)

	var dirX, dirY, dirZ float64
	angle := 0.0
	if rawAngle, ok := ent.GetProperty("angle"); ok {
		if parsed, okParse := lumps.ParseFloat(rawAngle); okParse {
			angle = parsed
		}
	}
	switch angle {
	case -1:
		dirZ = 1.0
	case -2:
		dirZ = -1.0
	default:
		dirX, dirY, dirZ = lumps.CalcAngleDirectionZUP(angle)
	}

	lip := 8.0
	if rawLip, ok := ent.GetProperty("lip"); ok {
		if parsed, okParse := lumps.ParseFloat(rawLip); okParse {
			lip = parsed
		}
	}
	travel := math.Abs(dirX)*size.X + math.Abs(dirY)*size.Y + math.Abs(dirZ)*size.Z - lip
	if travel <= 0 {
		travel = math.Max(size.X, math.Max(size.Y, size.Z))
	}

	speed := 100.0
	if rawSpeed, ok := ent.GetProperty("speed"); ok {
		if parsed, okParse := lumps.ParseFloat(rawSpeed); okParse && parsed > 0 {
			speed = parsed
		}
	}

	waitSeconds := 3.0
	if rawWait, ok := ent.GetProperty("wait"); ok {
		if parsed, okParse := lumps.ParseFloat(rawWait); okParse {
			waitSeconds = parsed
		}
	}

	spawnFlags := ent.GetSpawnFlags()
	requiredKey := ""
	if spawnFlags&q1DoorGoldKey != 0 {
		requiredKey = "gold"
	} else if spawnFlags&q1DoorSilverKey != 0 {
		requiredKey = "silver"
	}

	targetName, _ := ent.GetProperty("targetname")
	target, _ := ent.GetProperty("target")

	return &config.DoorSpec{
		RequiredKey: requiredKey,
		OpenOffset:  geometry.XYZ{X: dirX * travel, Y: dirY * travel, Z: dirZ * travel},
		Speed:       speed,
		WaitSeconds: waitSeconds,
		StartOpen:   spawnFlags&q1DoorStartOpen != 0,
		TargetName:  targetName,
		Target:      target,
		Activation:  config.DoorActivateTouch,
	}
}

// BuildPlatSpec maps Quake func_plat entity properties and brush bounds into a downward-sliding DoorSpec.
func BuildPlatSpec(ent *lumps.Entity, size geometry.XYZ) *config.DoorSpec {
	lip := 8.0
	if rawLip, ok := ent.GetProperty("lip"); ok {
		if parsed, okParse := lumps.ParseFloat(rawLip); okParse {
			lip = parsed
		}
	}
	travel := size.Z - lip
	if rawHeight, ok := ent.GetProperty("height"); ok {
		if parsed, okParse := lumps.ParseFloat(rawHeight); okParse && parsed > 0 {
			travel = parsed
		}
	}
	if travel <= 0 {
		travel = size.Z
	}

	speed := 150.0
	if rawSpeed, ok := ent.GetProperty("speed"); ok {
		if parsed, okParse := lumps.ParseFloat(rawSpeed); okParse && parsed > 0 {
			speed = parsed
		}
	}

	waitSeconds := 3.0
	if rawWait, ok := ent.GetProperty("wait"); ok {
		if parsed, okParse := lumps.ParseFloat(rawWait); okParse {
			waitSeconds = parsed
		}
	}

	targetName, _ := ent.GetProperty("targetname")
	target, _ := ent.GetProperty("target")

	return &config.DoorSpec{
		OpenOffset:  geometry.XYZ{Z: -travel},
		Speed:       speed,
		WaitSeconds: waitSeconds,
		TargetName:  targetName,
		Target:      target,
		Activation:  config.DoorActivateTouch,
	}
}

// BuildTeleportFromPoints creates a ThingTeleportDef from a trigger brush's points and entity target.
func BuildTeleportFromPoints(id string, points []geometry.XYZ, offset geometry.XYZ, ent *lumps.Entity, dests map[string]TeleportDest) *config.Thing {
	if len(points) == 0 {
		return nil
	}
	target, ok := ent.GetProperty("target")
	if !ok || strings.TrimSpace(target) == "" {
		return nil
	}
	dest, ok := dests[target]
	if !ok {
		return nil
	}

	minPt := geometry.XYZ{X: math.MaxFloat64, Y: math.MaxFloat64, Z: math.MaxFloat64}
	maxPt := geometry.XYZ{X: -math.MaxFloat64, Y: -math.MaxFloat64, Z: -math.MaxFloat64}
	for _, p := range points {
		if p.X < minPt.X {
			minPt.X = p.X
		}
		if p.Y < minPt.Y {
			minPt.Y = p.Y
		}
		if p.Z < minPt.Z {
			minPt.Z = p.Z
		}
		if p.X > maxPt.X {
			maxPt.X = p.X
		}
		if p.Y > maxPt.Y {
			maxPt.Y = p.Y
		}
		if p.Z > maxPt.Z {
			maxPt.Z = p.Z
		}
	}

	pos := geometry.XYZ{X: (minPt.X+maxPt.X)*0.5 + offset.X, Y: (minPt.Y+maxPt.Y)*0.5 + offset.Y, Z: minPt.Z + offset.Z}
	radius := math.Max((maxPt.X-minPt.X)*0.5, (maxPt.Y-minPt.Y)*0.5)
	if radius < 16.0 {
		radius = 16.0
	}
	height := maxPt.Z - minPt.Z
	if height < 32.0 {
		height = 32.0
	}

	tpCfg := config.NewConfigThing(id, pos, 0, config.ThingTeleportDef, 1000.0, radius, height, 0)
	tpCfg.GForce = 0
	_ = common.NewItem(tpCfg)
	tpCfg.Teleport = &config.TeleportSpec{Destination: dest.Pos, TargetAngle: dest.Angle, ResetVelocity: true, Cooldown: 0.5}
	return tpCfg
}

// CreateConfigThing creates a configured thing at the given position, selecting enemy, key, weapon, or item behavior by class.
func CreateConfigThing(classname string, category string, pos geometry.XYZ, cModel *config.Model3DEntry, ent *lumps.Entity) *config.Thing {
	if strings.HasPrefix(classname, "func_door") {
		center, size := cModel.CenterWithBounds([]string{"default"})
		worldPos := geometry.XYZ{X: pos.X + center.X, Y: pos.Y + center.Y, Z: pos.Z + center.Z}
		return CreateConfigDoor(classname, worldPos, cModel, 0.0, heavyMass, 0.0, ent, size)
	}
	if strings.HasPrefix(classname, "func_plat") {
		center, size := cModel.CenterWithBounds([]string{"default"})
		worldPos := geometry.XYZ{X: pos.X + center.X, Y: pos.Y + center.Y, Z: pos.Z + center.Z}
		return CreateConfigPlat(classname, worldPos, cModel, 0.0, heavyMass, 0.0, ent, size)
	}
	if strings.HasPrefix(classname, "func_train") {
		center, _ := cModel.CenterWithBounds([]string{"default"})
		worldPos := geometry.XYZ{X: pos.X + center.X, Y: pos.Y + center.Y, Z: pos.Z + center.Z}
		return CreateConfigPlatform(classname, worldPos, cModel, 0.0, heavyMass, 0.0)
	}
	if strings.HasPrefix(classname, "func_button") {
		center, _ := cModel.CenterWithBounds([]string{"default"})
		worldPos := geometry.XYZ{X: pos.X + center.X, Y: pos.Y + center.Y, Z: pos.Z + center.Z}
		return CreateConfigButton(classname, worldPos, cModel, 0.0, 16.0, 16.0)
	}
	cModel.Center([]string{"stand", "idle", "legs_idle", "hover"})
	if keyId := q2KeyId(classname); len(keyId) > 0 {
		return CreateConfigKey(classname, pos, cModel, 0, 0.1, 0.0, keyId)
	}
	switch category {
	case "item", "ammo":
		return CreateConfigItem(classname, pos, cModel, 0, 30, 0)
	case "weapon":
		return CreateConfigWeapon(classname, pos, cModel, 0, 30, 0)
	case "enemy", "monster":
		const wakeupDistance = 400.0
		return CreateConfigEnemy(classname, pos, cModel, 0, 30, 1300, wakeupDistance)
	default:
		return CreateConfigItem(classname, pos, cModel, 0, 30, 0)
	}
}

func CreateConfigEnemy(classname string, pos geometry.XYZ, cModel *config.Model3DEntry, angle, mass, speed, wakeupDistance float64) *config.Thing {
	actions := config.NewEnemyAnimations()
	if cModel != nil {
		ConvertAnimation(actions, cModel.ActionDefinitions, cModel.ActionIntervals)
	}
	thingCfg := config.NewConfigThing(classname, pos, angle, config.ThingEnemyDef, mass, 0, 0, speed)
	thingCfg.GForce = GForce
	thingCfg.Model3DEntry = cModel
	thingCfg.JumpForce = JumpForce
	thingCfg.WakeupDistance = wakeupDistance
	_ = common.NewEnemy(thingCfg, actions)
	return thingCfg
}

func CreateConfigItem(classname string, pos geometry.XYZ, cModel *config.Model3DEntry, angle, mass, speed float64) *config.Thing {
	thingCfg := config.NewConfigThing(classname, pos, angle, config.ThingItemDef, mass, 0, 0, speed)
	thingCfg.GForce = GForce
	thingCfg.Model3DEntry = cModel
	_ = common.NewItem(thingCfg)
	return thingCfg
}

func CreateConfigWeapon(classname string, pos geometry.XYZ, cModel *config.Model3DEntry, angle, mass, speed float64) *config.Thing {
	thingCfg := config.NewConfigWeapon(classname, pos, angle, mass)
	thingCfg.GForce = GForce
	thingCfg.Model3DEntry = cModel
	_ = common.NewWeapon(thingCfg)
	return thingCfg
}

func CreateConfigKey(classname string, pos geometry.XYZ, cModel *config.Model3DEntry, angle, mass, speed float64, keyId string) *config.Thing {
	thingCfg := config.NewConfigThing(classname, pos, angle, config.ThingKeyDef, mass, 0, 0, speed)
	thingCfg.GForce = GForce
	thingCfg.Model3DEntry = cModel
	thingCfg.Mass = 0.1
	thingCfg.Key = config.NewKeySpec(keyId)
	return thingCfg
}

func CreateConfigDoor(classname string, pos geometry.XYZ, cModel *config.Model3DEntry, angle, mass, speed float64, ent *lumps.Entity, size geometry.XYZ) *config.Thing {
	thingCfg := config.NewConfigThing(classname, pos, angle, config.ThingDoorDef, mass, 0, 0, speed)
	thingCfg.GForce = 0.0
	thingCfg.Model3DEntry = cModel
	thingCfg.Door = BuildDoorSpec(ent, size)
	return thingCfg
}

func CreateConfigPlat(classname string, pos geometry.XYZ, cModel *config.Model3DEntry, angle, mass, speed float64, ent *lumps.Entity, size geometry.XYZ) *config.Thing {
	thingCfg := config.NewConfigThing(classname, pos, angle, config.ThingDoorDef, mass, 0, 0, speed)
	thingCfg.GForce = 0.0
	thingCfg.Model3DEntry = cModel
	thingCfg.Door = BuildPlatSpec(ent, size)
	return thingCfg
}

func CreateConfigPlatform(classname string, pos geometry.XYZ, cModel *config.Model3DEntry, angle, mass, speed float64) *config.Thing {
	thingCfg := config.NewConfigThing(classname, pos, angle, config.ThingPlatformDef, mass, 0, 0, speed)
	thingCfg.GForce = 0.0
	thingCfg.Model3DEntry = cModel
	return thingCfg
}

func CreateConfigButton(classname string, pos geometry.XYZ, cModel *config.Model3DEntry, angle, mass, speed float64) *config.Thing {
	thingCfg := config.NewConfigThing(classname, pos, angle, config.ThingButtonDef, mass, 0, 0, speed)
	thingCfg.GForce = 0.0
	thingCfg.Model3DEntry = cModel
	return thingCfg
}
