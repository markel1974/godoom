package q1

import (
	"math"
	"strings"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/generators/common"
	"github.com/markel1974/godoom/mr_tech/generators/quake/lumps"
	"github.com/markel1974/godoom/mr_tech/geometry"
)

// q1KeyId returns the Quake 1 key ID for a key item classname, or an empty string if it is not a key.
func q1KeyId(classname string) string {
	switch classname {
	case "item_key1":
		return "silver"
	case "item_key2":
		return "gold"
	}
	return ""
}

// BuildDoorSpec creates a door specification from entity properties and brush size.
// It derives movement direction, travel offset, timing, key requirements, targets, and start-open state.
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

// BuildPlatSpec creates a touch-activated platform door spec from entity properties and brush size.
// It derives downward travel from height or lip, with defaults for speed and wait time.
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

// CreateConfigThing creates a configured thing at the given position, selecting enemy, key, or item behavior by class.
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
	if keyId := q1KeyId(classname); len(keyId) > 0 {
		return CreateConfigKey(classname, pos, cModel, 0, 0.1, keyId)
	}
	switch category {
	case "item":
		return CreateConfigItem(classname, pos, cModel, 0, 30, 0)
	case "weapon":
		return CreateConfigWeapon(classname, pos, cModel, 0, 30, 0)
	case "enemy":
		const wakeupDistance = 300.0
		return CreateConfigEnemy(classname, pos, cModel, 0, 30, 1000, wakeupDistance)
	case "monster":
		const wakeupDistance = 300.0
		return CreateConfigEnemy(classname, pos, cModel, 0, 30, 1000, wakeupDistance)
	default:
		return CreateConfigItem(classname, pos, cModel, 0, 30, 0)
	}
}

// CreateConfigEnemy creates an enemy thing with model animations, physics settings, speed, and wakeup distance.
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

// CreateConfigItem creates and registers an item thing with the given class, position, model, angle, mass, and speed.
func CreateConfigItem(classname string, pos geometry.XYZ, cModel *config.Model3DEntry, angle, mass, speed float64) *config.Thing {
	thingCfg := config.NewConfigThing(classname, pos, angle, config.ThingItemDef, mass, 0, 0, speed)
	thingCfg.GForce = GForce
	thingCfg.Model3DEntry = cModel
	_ = common.NewItem(thingCfg)
	return thingCfg
}

// CreateConfigWeapon creates and returns a weapon configuration with position, model, angle, and mass.
// It applies global gravity and initializes weapon behavior for the returned configuration.
func CreateConfigWeapon(classname string, pos geometry.XYZ, cModel *config.Model3DEntry, angle, mass, speed float64) *config.Thing {
	thingCfg := config.NewConfigWeapon(classname, pos, angle, mass)
	thingCfg.GForce = GForce
	thingCfg.Model3DEntry = cModel
	_ = common.NewWeapon(thingCfg)
	return thingCfg
}

// CreateConfigKey creates a key Thing at the given position with model, movement, and key identifier settings.
func CreateConfigKey(classname string, pos geometry.XYZ, cModel *config.Model3DEntry, angle, mass float64, keyId string) *config.Thing {
	thingCfg := config.NewConfigKey(classname, pos, angle, mass, keyId)
	thingCfg.GForce = GForce
	thingCfg.Model3DEntry = cModel
	_ = common.NewKey(thingCfg)
	return thingCfg
}

// CreateConfigDoor creates a door thing configuration with model, transform, physics, and door spec data.
func CreateConfigDoor(classname string, pos geometry.XYZ, cModel *config.Model3DEntry, angle, mass, speed float64, ent *lumps.Entity, size geometry.XYZ) *config.Thing {
	thingCfg := config.NewConfigThing(classname, pos, angle, config.ThingDoorDef, mass, 0, 0, speed)
	thingCfg.GForce = 0.0
	thingCfg.Model3DEntry = cModel
	thingCfg.Door = BuildDoorSpec(ent, size)
	_ = common.NewDoor(thingCfg)
	return thingCfg
}

// CreateConfigPlat creates a platform thing configuration with model, movement, physics, and platform door data.
func CreateConfigPlat(classname string, pos geometry.XYZ, cModel *config.Model3DEntry, angle, mass, speed float64, ent *lumps.Entity, size geometry.XYZ) *config.Thing {
	thingCfg := config.NewConfigThing(classname, pos, angle, config.ThingDoorDef, mass, 0, 0, speed)
	thingCfg.GForce = 0.0
	thingCfg.Model3DEntry = cModel
	thingCfg.Door = BuildPlatSpec(ent, size)
	_ = common.NewDoor(thingCfg)
	return thingCfg
}

// CreateConfigPlatform creates a platform thing configuration with the given position, model, angle, mass, and speed.
func CreateConfigPlatform(classname string, pos geometry.XYZ, cModel *config.Model3DEntry, angle, mass, speed float64) *config.Thing {
	thingCfg := config.NewConfigThing(classname, pos, angle, config.ThingPlatformDef, mass, 0, 0, speed)
	thingCfg.GForce = 0.0
	thingCfg.Model3DEntry = cModel
	_ = common.NewPlatform(thingCfg)
	return thingCfg
}

// CreateConfigButton creates a button thing configuration with the given class, position, model, angle, mass, and speed.
func CreateConfigButton(classname string, pos geometry.XYZ, cModel *config.Model3DEntry, angle, mass, speed float64) *config.Thing {
	thingCfg := config.NewConfigThing(classname, pos, angle, config.ThingButtonDef, mass, 0, 0, speed)
	thingCfg.GForce = GForce
	thingCfg.Model3DEntry = cModel
	_ = common.NewButton(thingCfg)
	return thingCfg
}

// CreateConfigTeleport creates a teleport thing from point bounds and an entity target.
// It returns nil when points are empty, the target is missing, or no destination matches.
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

	tpCfg := config.NewConfigThing(id, pos, 0, config.ThingTeleportDef, 1000.0, radius, height, 0)
	tpCfg.GForce = 0
	tpCfg.Teleport = &config.TeleportSpec{Destination: dest.Pos, TargetAngle: dest.Angle, ResetVelocity: true, Cooldown: 0.5}
	_ = common.NewTeleport(tpCfg)
	return tpCfg
}
