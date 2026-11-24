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

// CreateConfigTeleport creates a teleport thing from brush point bounds and a target destination.
func CreateConfigTeleport(id string, points []geometry.XYZ, offset geometry.XYZ, dest *TeleportDest) *config.Thing {
	if len(points) == 0 {
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

	thingCfg := common.NewTeleport(id, pos, heavyMass, radius, height, &config.TeleportSpec{Destination: dest.Pos, TargetAngle: dest.Angle, ResetVelocity: true, Cooldown: 0.5})
	thingCfg.GForce = 0
	return thingCfg.Thing
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
	return CreateConfigThingFromCenteredModel(classname, category, pos, cModel)
}

// CreateConfigThingFromCenteredModel wraps an already-centered Model3DEntry into a Thing instance without mutating or recentering cModel.
func CreateConfigThingFromCenteredModel(classname string, category string, pos geometry.XYZ, cModel *config.Model3DEntry) *config.Thing {
	if keyId := q2KeyId(classname); len(keyId) > 0 {
		return CreateConfigKey(classname, pos, cModel, 0, 0.1, keyId)
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
	thingCfg := common.NewEnemy(classname, pos, angle, mass, speed, wakeupDistance, actions)
	thingCfg.GForce = GForce
	thingCfg.Element3D = cModel
	thingCfg.JumpForce = JumpForce
	return thingCfg.Thing
}

func CreateConfigItem(classname string, pos geometry.XYZ, cModel *config.Model3DEntry, angle, mass, speed float64) *config.Thing {
	thingCfg := common.NewItem(classname, pos, angle, mass, speed)
	thingCfg.GForce = GForce
	thingCfg.Element3D = cModel
	return thingCfg.Thing
}

func CreateConfigWeapon(classname string, pos geometry.XYZ, cModel *config.Model3DEntry, angle, mass, speed float64) *config.Thing {
	thingCfg := common.NewWeapon(classname, pos, angle, mass)
	thingCfg.GForce = GForce
	thingCfg.Element3D = cModel
	return thingCfg.Thing
}

// CreateConfigKey creates a key thing at the given position with model, angle, mass, and key ID.
func CreateConfigKey(classname string, pos geometry.XYZ, cModel *config.Model3DEntry, angle, mass float64, keyId string) *config.Thing {
	thingCfg := common.NewKey(classname, pos, angle, mass, keyId)
	thingCfg.GForce = GForce
	thingCfg.Element3D = cModel
	return thingCfg.Thing
}

func CreateConfigDoor(classname string, pos geometry.XYZ, cModel *config.Model3DEntry, angle, mass, speed float64, ent *lumps.Entity, size geometry.XYZ) *config.Thing {
	thingCfg := common.NewDoor(classname, pos, angle, mass, speed, BuildDoorSpec(ent, size))
	thingCfg.GForce = 0.0
	thingCfg.Element3D = cModel
	return thingCfg.Thing
}

func CreateConfigPlat(classname string, pos geometry.XYZ, cModel *config.Model3DEntry, angle, mass, speed float64, ent *lumps.Entity, size geometry.XYZ) *config.Thing {
	thingCfg := common.NewDoor(classname, pos, angle, mass, speed, BuildPlatSpec(ent, size))
	thingCfg.GForce = 0.0
	thingCfg.Element3D = cModel
	return thingCfg.Thing
}

func CreateConfigPlatform(classname string, pos geometry.XYZ, cModel *config.Model3DEntry, angle, mass, speed float64) *config.Thing {
	thingCfg := common.NewPlatform(classname, pos, angle, mass, speed)
	thingCfg.GForce = 0.0
	thingCfg.Element3D = cModel
	return thingCfg.Thing
}

func CreateConfigButton(classname string, pos geometry.XYZ, cModel *config.Model3DEntry, angle, mass, speed float64) *config.Thing {
	thingCfg := common.NewButton(classname, pos, angle, mass, speed)
	thingCfg.GForce = 0.0
	thingCfg.Element3D = cModel
	return thingCfg.Thing
}
