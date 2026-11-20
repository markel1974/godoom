package doom3

import (
	"math"
	"strconv"
	"strings"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/generators/common"
	"github.com/markel1974/godoom/mr_tech/generators/doom3/proc"
	"github.com/markel1974/godoom/mr_tech/geometry"
)

const heavyMass = 1000.0

// d3KeyId returns the normalized key identifier for a Doom 3 key classname, or an empty string if unknown.
func d3KeyId(classname string) string {
	switch classname {
	case "item_key_blue":
		return "blue"
	case "item_key_yellow":
		return "yellow"
	case "item_keycard_aco":
		return "aco"
	case "item_keycard_generic":
		return "generic"
	case "item_key1":
		return "silver"
	case "item_key2":
		return "gold"
	}
	if strings.HasPrefix(classname, "item_key_") {
		return strings.TrimPrefix(classname, "item_key_")
	}
	if strings.HasPrefix(classname, "item_keycard_") {
		return strings.TrimPrefix(classname, "item_keycard_")
	}
	if strings.HasPrefix(classname, "key_") {
		return strings.TrimPrefix(classname, "key_")
	}
	return ""
}

// buildDoorOrLift converts a supported non-dormant mover entity into a door, lift, platform, or button thing.
// It bakes the entity model and reports false when unsupported or lacking usable geometry.
func buildDoorOrLift(ent *proc.MapEntity, procModels map[string]*proc.Model, materials *proc.Materials, arc *proc.Pk4, thingsManager *Things) (*config.Thing, bool) {
	if ent.IsDormant() {
		return nil, false
	}
	classname, _ := ent.GetProperty("classname")
	if classname != "func_door" && classname != "func_door_rotating" && classname != "func_plat" && classname != "func_elevator" && classname != "func_mover" && classname != "func_train" && classname != "func_button" {
		return nil, false
	}

	var cModel *config.Model3DEntry
	modelProp, _ := ent.GetProperty("model")
	nameProp, _ := ent.GetProperty("name")

	var pModel *proc.Model
	if m, ok := procModels[modelProp]; ok {
		pModel = m
	} else if m, ok = procModels[nameProp]; ok {
		pModel = m
	}

	if pModel != nil {
		cModel = bakeProcModel(pModel, ent, materials, arc)
	} else if strings.HasSuffix(strings.ToLower(modelProp), ".lwo") && arc != nil {
		cModel = bakeLWOModel(modelProp, ent, materials, arc)
	}
	if cModel == nil || len(cModel.Frames) == 0 || len(cModel.Frames[0].Triangles) == 0 {
		return nil, false
	}

	id := nameProp
	if id == "" {
		id = classname
	}
	return CreateConfigThing(id, "", geometry.XYZ{}, cModel, ent), true
}

// bakeLWOModel loads and converts an LWO model into a Model3DEntry using entity transforms and resolved materials.
// It returns nil when the model cannot be opened, parsed, or produces no triangles.
func bakeLWOModel(modelPath string, ent *proc.MapEntity, materials *proc.Materials, arc proc.IPkArchive) *config.Model3DEntry {
	cleanPath := strings.TrimSpace(strings.ToLower(strings.ReplaceAll(modelPath, "\\", "/")))
	lf, err := arc.Open(cleanPath)
	if err != nil {
		return nil
	}
	lwd := proc.NewLWO()
	if err := lwd.Parse(lf); err != nil || len(lwd.Points) == 0 || len(lwd.Polygons) == 0 {
		return nil
	}

	tagMaterials := make([]*config.Material, len(lwd.Tags))
	for ti, tagName := range lwd.Tags {
		texNameLC := strings.ToLower(tagName)
		if materials != nil {
			if matDef := materials.Resolve(tagName); matDef != nil {
				if guiMat, _ := ResolveGUIMaterial(arc, materials, matDef, ent); guiMat != nil {
					tagMaterials[ti] = guiMat
				} else {
					tagMaterials[ti] = ResolveConfigMaterial(matDef)
				}
			}
		}
		if tagMaterials[ti] == nil {
			kind := ToMaterial(texNameLC)
			tagMaterials[ti] = config.NewConfigMaterial("lwo.part", []string{texNameLC}, kind)
			ConfigureMaterial(tagMaterials[ti], texNameLC, nil)
		}
	}

	var allTriangles []config.Model3DEntryTriangle
	for polyIdx, poly := range lwd.Polygons {
		if len(poly) < 3 {
			continue
		}
		var mat *config.Material
		if tagIdx, ok := lwd.PolyTags[polyIdx]; ok && tagIdx >= 0 && tagIdx < len(tagMaterials) {
			mat = tagMaterials[tagIdx]
		}
		if mat == nil && len(tagMaterials) > 0 {
			mat = tagMaterials[0]
		}

		getVertex := func(vi int) (geometry.XYZ, float32, float32) {
			var p geometry.XYZ
			if vi >= 0 && vi < len(lwd.Points) {
				p = ent.TransformPoint(lwd.Points[vi])
			}
			var u, v float32
			if uv, ok := lwd.VMadUVs[proc.UVKey{Poly: polyIdx, Vert: vi}]; ok {
				u, v = uv[0], uv[1]
			} else if uv, ok = lwd.VMapUVs[vi]; ok {
				u, v = uv[0], uv[1]
			}
			return p, u, v
		}

		p0, u0, v0 := getVertex(poly[0])
		for k := 1; k < len(poly)-1; k++ {
			p1, u1, v1 := getVertex(poly[k])
			p2, u2, v2 := getVertex(poly[k+1])

			tri := config.NewModel3DEntryTriangle(mat)
			tri.Vertices[0] = config.Model3DEntryVertex{Pos: p0, U: u0, V: v0}
			tri.Vertices[1] = config.Model3DEntryVertex{Pos: p2, U: u2, V: v2}
			tri.Vertices[2] = config.Model3DEntryVertex{Pos: p1, U: u1, V: v1}
			allTriangles = append(allTriangles, tri)
		}
	}

	if len(allTriangles) == 0 {
		return nil
	}
	model3d := config.NewModel3DEntry(1, []string{"default"}, []string{"death", "die"})
	model3d.Frames[0] = config.NewModel3DEntryFrame(allTriangles)
	return model3d
}

// bakeProcModel converts a proc model entity into a Model3D entry.
// It resolves materials, skips decal and shadow surfaces, and applies the entity transform.
func bakeProcModel(model *proc.Model, ent *proc.MapEntity, materials *proc.Materials, arc *proc.Pk4) *config.Model3DEntry {
	var allTriangles []config.Model3DEntryTriangle
	for _, surface := range model.Surfaces {
		matName := surface.Material
		texNameLC := strings.ToLower(matName)
		if strings.HasPrefix(texNameLC, "textures/decals/") || strings.HasSuffix(texNameLC, "_shadow") {
			continue
		}
		if materials != nil {
			if def := materials.Resolve(matName); def != nil && def.IsDecal {
				continue
			}
		}

		var material *config.Material
		if materials != nil {
			if matDef := materials.Resolve(matName); matDef != nil {
				if guiMat, _ := ResolveGUIMaterial(arc, materials, matDef, ent); guiMat != nil {
					material = guiMat
				} else {
					material = ResolveConfigMaterial(matDef)
				}
			}
		}
		if material == nil {
			material = config.NewConfigMaterial(matName, []string{texNameLC}, ToMaterial(texNameLC))
			ConfigureMaterial(material, texNameLC, nil)
		}

		for i := 0; i+2 < len(surface.Indices); i += 3 {
			i0, i1, i2 := surface.Indices[i], surface.Indices[i+1], surface.Indices[i+2]
			if i0 >= len(surface.Vertices) || i1 >= len(surface.Vertices) || i2 >= len(surface.Vertices) {
				continue
			}
			v0, v1, v2 := surface.Vertices[i0], surface.Vertices[i1], surface.Vertices[i2]
			tri := config.NewModel3DEntryTriangle(material)
			tri.Vertices[0] = config.Model3DEntryVertex{Pos: ent.TransformPoint(v0.Pos), U: float32(v0.U), V: float32(v0.V)}
			tri.Vertices[1] = config.Model3DEntryVertex{Pos: ent.TransformPoint(v1.Pos), U: float32(v1.U), V: float32(v1.V)}
			tri.Vertices[2] = config.Model3DEntryVertex{Pos: ent.TransformPoint(v2.Pos), U: float32(v2.U), V: float32(v2.V)}
			allTriangles = append(allTriangles, tri)
		}
	}
	model3d := config.NewModel3DEntry(1, []string{"default"}, []string{"death", "die"})
	model3d.Frames[0] = config.NewModel3DEntryFrame(allTriangles)
	return model3d
}

// BuildDoorSpec builds a door specification from entity properties and the provided door size.
// It derives movement direction, travel distance, speed, wait time, lock requirements, and targets.
func BuildDoorSpec(ent *proc.MapEntity, size geometry.XYZ) *config.DoorSpec {
	var dirX, dirY, dirZ float64
	hasDir := false
	if raw, ok := ent.GetProperty("movedir"); ok {
		rawTrim := strings.TrimSpace(raw)
		fields := strings.Fields(rawTrim)
		if len(fields) == 3 {
			vx, errX := strconv.ParseFloat(fields[0], 64)
			vy, errY := strconv.ParseFloat(fields[1], 64)
			vz, errZ := strconv.ParseFloat(fields[2], 64)
			if errX == nil && errY == nil && errZ == nil {
				v := ent.TransformVector(geometry.XYZ{X: vx, Y: vy, Z: vz})
				vLen := math.Sqrt(v.X*v.X + v.Y*v.Y + v.Z*v.Z)
				if vLen > 1e-6 {
					dirX = v.X / vLen
					dirY = v.Y / vLen
					dirZ = v.Z / vLen
					hasDir = true
				}
			}
		} else if v, err := strconv.ParseFloat(rawTrim, 64); err == nil {
			switch v {
			case -1:
				dirZ = 1.0
				hasDir = true
			case -2:
				dirZ = -1.0
				hasDir = true
			default:
				rad := v * math.Pi / 180.0
				dirX = math.Cos(rad)
				dirY = math.Sin(rad)
				hasDir = true
			}
		}
	}

	if !hasDir {
		if rawAngle, ok := ent.GetProperty("angle"); ok {
			if v, err := strconv.ParseFloat(strings.TrimSpace(rawAngle), 64); err == nil {
				switch v {
				case -1:
					dirZ = 1.0
					hasDir = true
				case -2:
					dirZ = -1.0
					hasDir = true
				default:
					rad := v * math.Pi / 180.0
					dirX = math.Cos(rad)
					dirY = math.Sin(rad)
					hasDir = true
				}
			}
		}
	}

	if !hasDir {
		ang := ent.GetAngle()
		rad := ang * math.Pi / 180.0
		dirX = math.Cos(rad)
		dirY = math.Sin(rad)
	}

	lip := 8.0
	if raw, ok := ent.GetProperty("lip"); ok {
		if v, err := strconv.ParseFloat(strings.TrimSpace(raw), 64); err == nil {
			lip = v
		}
	}
	travel := math.Abs(dirX)*size.X + math.Abs(dirY)*size.Y + math.Abs(dirZ)*size.Z - lip
	if travel <= 0 {
		travel = math.Max(size.X, math.Max(size.Y, size.Z))
	}

	speed := 150.0
	if raw, ok := ent.GetProperty("speed"); ok {
		if v, err := strconv.ParseFloat(strings.TrimSpace(raw), 64); err == nil && v > 0 {
			speed = v
		}
	}

	waitSeconds := 3.0
	if raw, ok := ent.GetProperty("wait"); ok {
		if v, err := strconv.ParseFloat(strings.TrimSpace(raw), 64); err == nil {
			waitSeconds = v
		}
	}

	startOpen := false
	if raw, ok := ent.GetProperty("start_open"); ok && strings.TrimSpace(raw) == "1" {
		startOpen = true
	}

	requiredKey, _ := ent.GetProperty("requires")
	if requiredKey == "" {
		if locked, ok := ent.GetProperty("locked"); ok && strings.TrimSpace(locked) == "1" {
			requiredKey, _ = ent.GetProperty("key")
		}
	}

	targetName, _ := ent.GetProperty("name")
	target, _ := ent.GetProperty("target")

	return &config.DoorSpec{
		RequiredKey: requiredKey,
		OpenOffset:  geometry.XYZ{X: dirX * travel, Y: dirY * travel, Z: dirZ * travel},
		Speed:       speed,
		WaitSeconds: waitSeconds,
		StartOpen:   startOpen,
		TargetName:  targetName,
		Target:      target,
		Activation:  config.DoorActivateTouch,
	}
}

// BuildPlatSpec creates a touch-activated platform door spec from entity properties and brush size.
// It derives travel, speed, wait time, target name, and target with sensible defaults.
func BuildPlatSpec(ent *proc.MapEntity, size geometry.XYZ) *config.DoorSpec {
	lip := 8.0
	if raw, ok := ent.GetProperty("lip"); ok {
		if v, err := strconv.ParseFloat(strings.TrimSpace(raw), 64); err == nil {
			lip = v
		}
	}
	travel := size.Z - lip
	if raw, ok := ent.GetProperty("height"); ok {
		if v, err := strconv.ParseFloat(strings.TrimSpace(raw), 64); err == nil && v > 0 {
			travel = v
		}
	}
	if travel <= 0 {
		travel = size.Z
	}
	speed := 150.0
	if raw, ok := ent.GetProperty("speed"); ok {
		if v, err := strconv.ParseFloat(strings.TrimSpace(raw), 64); err == nil && v > 0 {
			speed = v
		}
	}
	waitSeconds := 3.0
	if raw, ok := ent.GetProperty("wait"); ok {
		if v, err := strconv.ParseFloat(strings.TrimSpace(raw), 64); err == nil {
			waitSeconds = v
		}
	}
	targetName, _ := ent.GetProperty("name")
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

// CreateConfigTeleport creates a teleport config thing from a map entity and its target destination.
// It returns nil for dormant entities or unresolved positions.
func CreateConfigTeleport(ent *proc.MapEntity, procModels map[string]*proc.Model, dest *TeleportDest) *config.Thing {
	if ent.IsDormant() {
		return nil
	}
	classname, _ := ent.GetProperty("classname")
	modelProp, _ := ent.GetProperty("model")
	nameProp, _ := ent.GetProperty("name")
	var pModel *proc.Model
	if m, ok := procModels[modelProp]; ok {
		pModel = m
	} else if m, ok := procModels[nameProp]; ok {
		pModel = m
	}

	var pos geometry.XYZ
	radius, height := 32.0, 64.0

	if pModel != nil {
		minPt := geometry.XYZ{X: math.MaxFloat64, Y: math.MaxFloat64, Z: math.MaxFloat64}
		maxPt := geometry.XYZ{X: -math.MaxFloat64, Y: -math.MaxFloat64, Z: -math.MaxFloat64}
		hasVerts := false
		for _, surf := range pModel.Surfaces {
			for _, v := range surf.Vertices {
				p := ent.TransformPoint(v.Pos)
				hasVerts = true
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
		}
		if hasVerts {
			pos = geometry.XYZ{
				X: (minPt.X + maxPt.X) * 0.5,
				Y: (minPt.Y + maxPt.Y) * 0.5,
				Z: minPt.Z,
			}
			if r := math.Max((maxPt.X-minPt.X)*0.5, (maxPt.Y-minPt.Y)*0.5); r > 16.0 {
				radius = r
			}
			if h := maxPt.Z - minPt.Z; h > 32.0 {
				height = h
			}
		}
	} else if origin, err := ent.GetOrigin(); err == nil {
		pos = origin
	} else {
		return nil
	}

	id := nameProp
	if id == "" {
		id = classname
	}
	thingCfg := common.NewTeleport(id, pos, heavyMass, radius, height, &config.TeleportSpec{
		Destination:   dest.Pos,
		TargetAngle:   dest.Angle,
		ResetVelocity: true,
		Cooldown:      0.5,
	})
	thingCfg.GForce = 0
	return thingCfg.Thing
}

// CreateConfigThing creates a configured thing at pos, centering its model and selecting enemy, key, weapon, door, platform, or item behavior.
func CreateConfigThing(classname string, category string, pos geometry.XYZ, cModel *config.Model3DEntry, ent *proc.MapEntity) *config.Thing {
	checkClass := classname
	if ent != nil {
		if c, ok := ent.GetProperty("classname"); ok && c != "" {
			checkClass = c
		}
	}
	if checkClass == "func_door" || checkClass == "func_door_rotating" || strings.HasPrefix(checkClass, "func_door") {
		var size geometry.XYZ
		var worldPos geometry.XYZ
		if cModel != nil {
			center, s := cModel.CenterWithBounds([]string{"default"})
			size = s
			worldPos = geometry.XYZ{X: pos.X + center.X, Y: pos.Y + center.Y, Z: pos.Z + center.Z}
		} else {
			worldPos = pos
		}
		return CreateConfigDoor(classname, worldPos, cModel, 0.0, heavyMass, 0.0, ent, size)
	}
	if checkClass == "func_plat" || checkClass == "func_elevator" || strings.HasPrefix(checkClass, "func_plat") {
		var size geometry.XYZ
		var worldPos geometry.XYZ
		if cModel != nil {
			center, s := cModel.CenterWithBounds([]string{"default"})
			size = s
			worldPos = geometry.XYZ{X: pos.X + center.X, Y: pos.Y + center.Y, Z: pos.Z + center.Z}
		} else {
			worldPos = pos
		}
		return CreateConfigPlat(classname, worldPos, cModel, 0.0, heavyMass, 0.0, ent, size)
	}
	if checkClass == "func_mover" || checkClass == "func_train" || strings.HasPrefix(checkClass, "func_train") {
		var worldPos geometry.XYZ
		if cModel != nil {
			center, _ := cModel.CenterWithBounds([]string{"default"})
			worldPos = geometry.XYZ{X: pos.X + center.X, Y: pos.Y + center.Y, Z: pos.Z + center.Z}
		} else {
			worldPos = pos
		}
		return CreateConfigPlatform(classname, worldPos, cModel, 0.0, heavyMass, 0.0)
	}
	if checkClass == "func_button" || strings.HasPrefix(checkClass, "func_button") {
		var worldPos geometry.XYZ
		if cModel != nil {
			center, _ := cModel.CenterWithBounds([]string{"default"})
			worldPos = geometry.XYZ{X: pos.X + center.X, Y: pos.Y + center.Y, Z: pos.Z + center.Z}
		} else {
			worldPos = pos
		}
		return CreateConfigButton(classname, worldPos, cModel, 0.0, 16.0, 16.0)
	}
	if cModel != nil {
		cModel.Center([]string{"stand", "idle", "legs_idle", "hover"})
	}
	if keyId := d3KeyId(classname); len(keyId) > 0 {
		return CreateConfigKey(classname, pos, cModel, 0, 0.1, keyId)
	}
	switch category {
	case "item", "ammo":
		return CreateConfigItem(classname, pos, cModel, 0, 30, 0)
	case "weapon":
		return CreateConfigWeapon(classname, pos, cModel, 0, 30, 0)
	case "enemy", "monster":
		const wakeupDistance = 300.0
		return CreateConfigEnemy(classname, pos, cModel, 0, 30, 1150, wakeupDistance)
	default:
		return CreateConfigItem(classname, pos, cModel, 0, 30, 0)
	}
}

// CreateConfigEnemy creates an enemy thing from an optional model and basic movement and wakeup settings.
func CreateConfigEnemy(classname string, pos geometry.XYZ, cModel *config.Model3DEntry, angle, mass, speed, wakeupDistance float64) *config.Thing {
	var parts []*config.Model3DEntry
	if cModel != nil {
		parts = append(parts, cModel)
	}
	links := []config.Model3DLink{{Parent: -1, Tag: ""}}
	md3 := config.NewMD3(parts, links)
	if cModel != nil {
		md3.ActionDefinitions = cModel.ActionDefinitions
		md3.ActionIntervals = cModel.ActionIntervals
		numActions := len(cModel.ActionDefinitions)
		md3.ActionMaps = make([][]int, len(parts))
		for i := range md3.ActionMaps {
			md3.ActionMaps[i] = make([]int, numActions)
			for a := 0; a < numActions; a++ {
				md3.ActionMaps[i][a] = a
			}
		}
	}
	return CreateConfigEnemyParts(classname, pos, md3, angle, mass, speed, wakeupDistance)
}

// CreateConfigEnemyParts creates an enemy Thing with a multipart 3D model, physics settings, and animations.
func CreateConfigEnemyParts(classname string, pos geometry.XYZ, md3 *config.Model3D, angle, mass, speed, wakeupDistance float64) *config.Thing {
	actions := config.NewEnemyAnimations()
	if md3 != nil && len(md3.ActionDefinitions) > 0 {
		ConvertAnimation(actions, md3.ActionDefinitions, md3.ActionIntervals)
	}
	if len(actions.Content[config.EnemyStateIdle].Indices) == 0 {
		actions.Add(config.EnemyStateIdle, 0, 1)
		actions.Add(config.EnemyStateWalk, 0, 1)
		actions.Add(config.EnemyStateRun, 0, 1)
	}
	thingCfg := common.NewEnemy(classname, pos, angle, mass, speed, wakeupDistance, actions)
	thingCfg.MultiElement3D = md3
	thingCfg.GForce = GForce
	thingCfg.JumpForce = JumpForce
	thingCfg.Impulse = 3.0
	return thingCfg.Thing
}

// CreateConfigItem creates an item thing configuration with the given class, position, model, angle, mass, and speed.
// It applies the default gravity force, registers the item, and returns the configured thing.
func CreateConfigItem(classname string, pos geometry.XYZ, cModel *config.Model3DEntry, angle, mass, speed float64) *config.Thing {
	thingCfg := common.NewItem(classname, pos, angle, mass, speed)
	thingCfg.GForce = GForce
	thingCfg.Element3D = cModel
	return thingCfg.Thing
}

// CreateConfigWeapon creates a weapon thing configuration with the given transform, model, and mass.
func CreateConfigWeapon(classname string, pos geometry.XYZ, cModel *config.Model3DEntry, angle, mass, speed float64) *config.Thing {
	thingCfg := common.NewWeapon(classname, pos, angle, mass)
	thingCfg.GForce = GForce
	thingCfg.Element3D = cModel
	return thingCfg.Thing
}

// CreateConfigKey creates a key thing configuration with the given class, placement, model, motion values, and key ID.
func CreateConfigKey(classname string, pos geometry.XYZ, cModel *config.Model3DEntry, angle, mass float64, keyId string) *config.Thing {
	thingCfg := common.NewKey(classname, pos, angle, mass, keyId)
	thingCfg.GForce = GForce
	thingCfg.Element3D = cModel
	return thingCfg.Thing
}

// CreateConfigDoor creates a door thing configuration with model, physics, placement, and door spec data.
// It builds the door spec from the source map entity and size, then attaches it to the returned thing.
func CreateConfigDoor(classname string, pos geometry.XYZ, cModel *config.Model3DEntry, angle, mass, speed float64, ent *proc.MapEntity, size geometry.XYZ) *config.Thing {
	thingCfg := common.NewDoor(classname, pos, angle, mass, speed, BuildDoorSpec(ent, size))
	thingCfg.GForce = 0.0
	thingCfg.Element3D = cModel
	return thingCfg.Thing
}

// CreateConfigPlat creates a platform thing configuration with model, motion, and door platform settings.
func CreateConfigPlat(classname string, pos geometry.XYZ, cModel *config.Model3DEntry, angle, mass, speed float64, ent *proc.MapEntity, size geometry.XYZ) *config.Thing {
	thingCfg := common.NewDoor(classname, pos, angle, mass, speed, BuildPlatSpec(ent, size))
	thingCfg.GForce = 0.0
	thingCfg.Element3D = cModel
	return thingCfg.Thing
}

// CreateConfigPlatform creates a platform Thing at pos with the given model, angle, mass, and movement speed.
// CreateConfigPlatform sets gravity force to zero for the returned platform.
func CreateConfigPlatform(classname string, pos geometry.XYZ, cModel *config.Model3DEntry, angle, mass, speed float64) *config.Thing {
	thingCfg := common.NewPlatform(classname, pos, angle, mass, speed)
	thingCfg.GForce = 0.0
	thingCfg.Element3D = cModel
	return thingCfg.Thing
}

// CreateConfigButton creates a button thing at the given position with model, angle, mass, and speed settings.
// It initializes the button with no gravity force and attaches the supplied 3D model entry.
func CreateConfigButton(classname string, pos geometry.XYZ, cModel *config.Model3DEntry, angle, mass, speed float64) *config.Thing {
	thingCfg := common.NewButton(classname, pos, angle, mass, speed)
	thingCfg.GForce = 0.0
	thingCfg.Element3D = cModel
	return thingCfg.Thing
}
